package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
)

var (
	// common concerns
	queueURL   string
	outputType string
	debug      bool

	// for consume
	visibilityTimeout int64

	// for publish
	sourceDir string
	environ   env
)

func init() {
	flag.StringVar(&queueURL, "sqs-queue-url", "", "The full SQS queue URL to use to receive messages")
	flag.StringVar(&outputType, "o", "", "set -o json to print output as JSON")
	flag.StringVar(&sourceDir, "dir", "", "The directory to pack into the tarball and publish to sqs")
	flag.BoolVar(&debug, "debug", false, "Enable debug output")
	flag.Int64Var(&visibilityTimeout, "sqs-visibility-timeout-sec", 300, "The number of seconds messages received by this working should be invisible to other workers (before deletion)")
	flag.Var(&environ, "e", "The environment variables which will be injected to the executable payload")
	flag.Parse()
}

func publish(logger Logger) {
	p := Payloader{}
	payload, err := p.DirToTarGz(sourceDir)
	if err != nil {
		logger.WithFields(ErrorFields(err)).Fatal("can not pack directory info tar archive")
	}

	err = NewAWSSQS(queueURL, logger, visibilityTimeout).PublishPayload(environ, payload)
	if err != nil {
		logger.WithFields(ErrorFields(err)).Fatal("can not publish payload")
	}
}

func consume(logger Logger) {

	var ctx, cancel = context.WithCancel(context.Background())

	var g = Gantry{
		logger: logger,
		src:    NewAWSSQS(queueURL, logger, visibilityTimeout),
		ctx:    ctx,
	}
	go g.Run()

	var sigs = make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	var sig = <-sigs
	cancel()
	logger.Infof("exiting %s", sig)

}

func main() {

	logrus.SetLevel(logrus.DebugLevel)

	if strings.ToLower(outputType) == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	logger := NewLogrusLogger(
		logrus.StandardLogger().WithField("component", "gantry"),
	)

	if len(queueURL) == 0 {
		flag.PrintDefaults()
		logger.Fatal("please specify queue url via -sqs-queue-url")
	}
	if _, err := url.Parse(queueURL); err != nil {
		logger.WithFields(ErrorFields(err)).Fatalf("can't parse url, try again please")
	}

	switch flag.Arg(0) {
	case "publish":
		publish(logger.WithFields(Fields{"action": "publish"}))
	case "consume":
		consume(logger.WithFields(Fields{"action": "consume"}))
	default:
		fmt.Fprintf(os.Stderr, "command must be set to one of either %q or %q\n", "publish", "consume")
		os.Exit(2)
	}

}
