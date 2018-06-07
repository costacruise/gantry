package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
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
		// TODO: think about whether fatalf is a clever part of
		//       __LOG__ interface
		g.logger.Fatalf("receive with context failed: %s", err)
		return err
	}
	if msg == nil {
		g.logger.Debugf("no messages available for receipt")
		return nil
	}
	defer msg.Delete()
	g.logger.WithFields(Fields{
		"status":            "message received",
		"message_queued_at": msg.SentAt().Format(time.RFC3339),
	}).Infof("message id: %s", msg.ID())

	dest, err := ioutil.TempDir("", "gantry-payload")
	if err != nil {
		g.logger.Fatalf("can not create temp dir %s", err)
		os.Exit(1)
	}

	err = Payloader{g.logger}.ExtractTarGzToDir(dest, msg.Payload())
	// TODO: write test for error case

	entrypointFI, err := os.Stat(filepath.Join(dest, "entrypoint.sh"))
	if err != nil {
		return errors.Errorf("message with id %s does contain entrypoint.sh in root directory, will be deleted", msg.ID())
	}
	if entrypointFI.Mode()&0111 == 0 { // check for executable bit for owner
		return errors.Errorf("expected payload to contain executable entrypoint.sh check the filemode")
	}

	// Get the current working dir, and restore it once we're done with the script
	pwd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "cannot get working directory")
	}
	defer func(old string) {
		os.Chdir(old)
	}(pwd)

	var out bytes.Buffer

	stdOutLogger := log.New(&out, "stdout>> ", 0)
	stdErrLogger := log.New(&out, "stderr>> ", 0)

	// Move into temp dir and run the entrypoint.sh
	os.Chdir(dest)
	cmd := exec.CommandContext(g.ctx, "./entrypoint.sh")
	cmd.Env = msg.Body().Env.ToEnviron()
	cmd.Stdout = &LogWriter{writeFn: stdOutLogger.Print}
	cmd.Stderr = &LogWriter{writeFn: stdErrLogger.Print}

	err = cmd.Run()

	l := g.logger.WithFields(Fields{
		"success":           err == nil,
		"status":            "completed",
		"command_env":       map[string]string(msg.Body().Env),
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
