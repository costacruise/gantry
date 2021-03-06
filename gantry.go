package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

// Gantry represents a polling listener for a given MessageSource
type Gantry struct {
	ctx context.Context
	src MessageSource

	logger Logger
}

func (g *Gantry) loop() {

	var ticker = time.NewTicker(60 * time.Second)

	// Ticker fires *after* the duration, not *every* duration, fire once to
	// avoid wait on first run
	g.HandleMessageIfExists()

	for {
		select {
		case <-g.ctx.Done():
			g.logger.Info("stopping context cancelled")
			ticker.Stop()
			return
		case <-ticker.C:
			g.HandleMessageIfExists()
		}
	}

}

// Run starts the polling loop
func (g *Gantry) Run() {
	g.loop()
}

// HandleMessageIfExists executes the payload from the message is one
// available. It returns the output of the execution. If there happens any
// error in between, it returns an empty string and the error.
func (g *Gantry) HandleMessageIfExists() error {
	g.logger.Debugf("checking for message")

	// TODO:
	//   test all cases err and msg nil, and all permutations
	//   (nil msg, nil err = empty inbox)
	//   (msg nil error = message available)
	//   (err nil msg = error)
	msg, err := g.src.ReceiveMessageWithContext(g.ctx)
	if err != nil {
		g.logger.WithFields(
			ErrorFields(err),
		).Error("receive message failed")
		return err
	}
	if msg == nil {
		g.logger.Debugf("no messages available for receipt")
		return nil
	}
	defer msg.Delete()
	messageLogger := g.logger.WithFields(Fields{
		"message": map[string]interface{}{
			"body":      msg.Body(),
			"id":        msg.ID(),
			"queued_at": msg.SentAt().Format(time.RFC3339),
		},
	})
	messageLogger.WithFields(Fields{
		"status": "message received",
	}).Infof("message id: %s", msg.ID())

	dest, err := ioutil.TempDir("", "gantry-payload")
	if err != nil {
		messageLogger.WithFields(
			ErrorFields(err),
		).Fatal("can not create temp dir")
		os.Exit(1)
	}

	err = Payloader{messageLogger}.ExtractTarGzToDir(dest, msg.Payload())
	// TODO: write test for error case

	entrypointFI, err := os.Stat(filepath.Join(dest, "entrypoint.sh"))
	if err != nil {
		err = errors.Errorf("message with id %s does contain entrypoint.sh in root directory, will be deleted", msg.ID())
		messageLogger.WithFields(ErrorFields(err)).Error("could not find entrypoint.sh")
		return err
	}
	if entrypointFI.Mode()&0111 == 0 { // check for executable bit for owner
		err = errors.Errorf("expected payload to contain executable entrypoint.sh check the filemode")
		messageLogger.WithFields(ErrorFields(err)).Error("entrypoint.sh is not executable")
		return err
	}

	// Get the current working dir, and restore it once we're done with the script
	pwd, err := os.Getwd()
	if err != nil {
		messageLogger.WithFields(ErrorFields(err)).Error("cannot get working directory")
		return errors.Wrap(err, "cannot get working directory")
	}
	defer func(old string) {
		os.Chdir(old)
	}(pwd)

	var stdErr bytes.Buffer

	// Move into temp dir and run the entrypoint.sh
	os.Chdir(dest)
	cmd := exec.CommandContext(g.ctx, "./entrypoint.sh")
	cmd.Env = msg.Body().Env.ToEnviron()
	cmd.Stderr = &stdErr

	err = cmd.Run()

	l := messageLogger.WithFields(Fields{
		"success":           err == nil,
		"status":            "completed",
		"command_env":       map[string]string(msg.Body().Env),
		"command_stderr":    stdErr.String(),
		"message_queued_at": msg.SentAt().Format(time.RFC3339),
	})

	if err != nil {
		l.WithFields(
			Fields{"error": err.Error()},
		).Error("executed entrypoint")
	} else {
		l.Info("executed entrypoint")
	}

	return err
}
