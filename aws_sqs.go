package main

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
)

// NewAWSSQS returns a messageQueue to publish and receive messages over amazon SQS
// service
func NewAWSSQS(queueURL string, logger Logger, visibilityTimeout int64) MessageQueue {
	return awsSQS{
		client:            sqs.New(session.Must(session.NewSession())),
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

func (as awsSQS) PublishPayload(b []byte) error {
	// Add MessageAttributes with debug info like digest maybe
	smi := sqs.SendMessageInput{
		MessageBody: aws.String(string(b)),
		QueueUrl:    &as.queueURL,
	}

	smo, err := as.client.SendMessage(&smi)
	if err != nil {
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
		MaxNumberOfMessages: aws.Int64(1),
		QueueUrl:            aws.String(as.queueURL),
		VisibilityTimeout:   aws.Int64(as.visibilityTimeout),
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

	// TODO: Check receivedMsg attributes and look for a CDU defined "mime type" (application/gzip?)

	msg = awsSQSMessage{
		id:            *receivedMsg.MessageId,
		receiptHandle: *receivedMsg.ReceiptHandle,
		payload:       []byte(*receivedMsg.Body),
		deleteFn: func() error {
			if _, err := as.client.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      &as.queueURL,
				ReceiptHandle: receivedMsg.ReceiptHandle,
			}); err != nil {
				return errors.Errorf("aws-sqs-message: could not delete message with id %s", *receivedMsg.MessageId)
			}
			as.logger.Infof("aws-sqs-message: deleted message with id %s", *receivedMsg.MessageId)
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
	payload       []byte
	deleteFn      func() error
}

func (asm awsSQSMessage) ID() string      { return asm.id }
func (asm awsSQSMessage) Payload() []byte { return asm.payload }
func (asm awsSQSMessage) Delete() error   { return asm.deleteFn() }
