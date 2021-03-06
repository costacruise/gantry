package main

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
)

type messageBody struct {
	Env env `json:"env"`
}

// NewAWSSQS returns a messageQueue to publish and receive messages over amazon SQS
// service
func NewAWSSQS(queueURL string, logger Logger, visibilityTimeout int64) MessageQueue {
	config := aws.NewConfig()
	if debug {
		config = config.WithLogLevel(aws.LogDebug | aws.LogDebugWithRequestErrors | aws.LogDebugWithHTTPBody)
	}
	return awsSQS{
		client: sqs.New(
			session.Must(session.NewSession()),
			config,
		),
		logger:            logger.WithFields(Fields{"component": "aws-sqs-src"}),
		queueURL:          queueURL,
		visibilityTimeout: visibilityTimeout,
	}
}

type awsSQS struct {
	// Common to publish and consume
	client   *sqs.SQS
	queueURL string

	// consumer vars
	visibilityTimeout int64

	logger Logger
}

func (as awsSQS) PublishPayload(env map[string]string, b []byte) error {
	body := messageBody{Env: env}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return errors.Wrap(err, "error while marshaling message body")
	}

	// Add MessageAttributes with debug info like digest maybe
	smi := sqs.SendMessageInput{
		MessageBody: aws.String(string(bodyBytes)),
		QueueUrl:    &as.queueURL,
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"data": &sqs.MessageAttributeValue{
				BinaryValue: b,
				DataType:    aws.String("Binary"),
			},
		},
	}

	smo, err := as.client.SendMessage(&smi)
	if err != nil {
		as.logger.WithFields(Fields{
			"payload_length": len(b),
			"message_body":   body,
		}.logError(err)).Errorf("Error sending sns message")
		return errors.Wrap(err, "could not send payload to SQS")
	}

	as.logger.Infof("published payload with message id %s", *smo.MessageId)

	return nil
}

func (as awsSQS) ReceiveMessageWithContext(ctx context.Context) (Message, error) {
	var (
		receivedMsg *sqs.Message
		msg         Message
	)

	as.logger.Debugf("checking for single message on sqs queue %s", as.queueURL)

	rmi := sqs.ReceiveMessageInput{
		MaxNumberOfMessages:   aws.Int64(1),
		QueueUrl:              aws.String(as.queueURL),
		VisibilityTimeout:     aws.Int64(as.visibilityTimeout),
		AttributeNames:        []*string{aws.String("SentTimestamp")},
		MessageAttributeNames: []*string{aws.String("data")},
	}

	resp, err := as.client.ReceiveMessageWithContext(ctx, &rmi)
	if err != nil {
		return nil, errors.Wrap(err, "receive message with context")
	}

	if len(resp.Messages) > 0 {
		receivedMsg = resp.Messages[0]
		as.logger.Debugf("got message, this message will be invisible to other clients for %d sec", as.visibilityTimeout)
	} else {
		as.logger.Debugf("no messages on queue")
		return nil, nil
	}

	data := []byte{}

	dataAttr, ok := receivedMsg.MessageAttributes["data"]
	if ok {
		data = dataAttr.BinaryValue
	} else {
		as.logger.Warnf("received no data attribute")
	}

	sentAt := time.Time{}
	sentAtAttr, ok := receivedMsg.Attributes["SentTimestamp"]
	if ok {
		timestamp, err := strconv.ParseInt(*sentAtAttr, 10, 64)
		if err == nil {
			sentAt = time.Unix(0, timestamp*int64(time.Millisecond))
		} else {
			as.logger.WithFields(Fields{
				"sent_timestamp": sentAtAttr,
			}.logError(err)).Warn("malformed sent timestamp")
		}
	} else {
		as.logger.Warn("received no SentTimestamp attribute")
	}

	var body messageBody
	err = json.Unmarshal([]byte(*receivedMsg.Body), &body)
	if err != nil {
		as.logger.WithFields(ErrorFields(err)).Warn("error while unmarshaling SQS message body")
	}

	msg = awsSQSMessage{
		id:            *receivedMsg.MessageId,
		receiptHandle: *receivedMsg.ReceiptHandle,
		sentAt:        sentAt,
		body:          body,
		payload:       data,
		deleteFn: func() error {
			if _, err := as.client.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      &as.queueURL,
				ReceiptHandle: receivedMsg.ReceiptHandle,
			}); err != nil {
				return errors.Errorf("aws-sqs-message: could not delete message with id %s", *receivedMsg.MessageId)
			}
			as.logger.WithFields(Fields{
				"message_id": *receivedMsg.MessageId,
			}).Infof("aws-sqs-message: deleted message")
			return nil
		},
	}

	if receivedMsg.Body == nil {
		as.logger.Debugf("got message with empty payload, message will be deleted")
		return nil, errors.Errorf("message body was empty %s", *receivedMsg.MessageId)
	}

	as.logger.Infof("message body checksum was md5: %s", *receivedMsg.MD5OfBody)

	return msg, nil
}

// An awsSQSMessage represents a SQS message. See
// http://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-queue-message-identifiers.html
// for details.
type awsSQSMessage struct {
	id            string
	receiptHandle string
	sentAt        time.Time
	body          messageBody
	payload       []byte
	deleteFn      func() error
}

func (asm awsSQSMessage) ID() string        { return asm.id }
func (asm awsSQSMessage) SentAt() time.Time { return asm.sentAt }
func (asm awsSQSMessage) Body() messageBody { return asm.body }
func (asm awsSQSMessage) Payload() []byte   { return asm.payload }
func (asm awsSQSMessage) Delete() error     { return asm.deleteFn() }
