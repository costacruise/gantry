package main

import (
	"bytes"
	"context"
	"io"
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
			g.logger.Debug("tick, checking for new messages")
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
func (g *Gantry) HandleMessageIfExists() (string, error) {

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
		return "", err
	}
	if msg == nil {
		g.logger.Debugf("no messages available for receipt")
		return "", nil
	}
	defer msg.Delete()
	g.logger.Infof("message id: %s", msg.ID())
	g.logger.Debugf("message payload: %s", msg.Payload())

	dest, err := ioutil.TempDir("", "gantry-payload")
	if err != nil {
		g.logger.Fatalf("can not create temp dir %s", err)
	}

	err = Payloader{g.logger}.Base64EncTarGzToDir(dest, msg.Payload())

	entrypointFI, err := os.Stat(filepath.Join(dest, "entrypoint.sh"))
	if err != nil {
		return "", errors.Errorf("message with id %s does contain entrypoint.sh in root directory, will be deleted", msg.ID())
	}
	if entrypointFI.Mode()&0111 == 0 { // check for executable bit for owner
		return "", errors.Errorf("expected payload to contain executable entrypoint.sh check the filemode")
	}

	// Get the current working dir, and restore it once we're done with the script
	pwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "cannot get working directory")
	}
	defer func(old string) {
		os.Chdir(old)
	}(pwd)

	var out bytes.Buffer

	// LogWriter implements io.Writer but writes to a structured logger
	lw := LogWriter{g.logger, 0}

	// MultiWriter as we'd like to capture output in a []byte for
	// testing purposes
	multi := io.MultiWriter(&lw, &out)

	// Move into temp dir and run the entrypoint.sh
	os.Chdir(dest)
	cmd := exec.CommandContext(g.ctx, "./entrypoint.sh")
	cmd.Stdout = multi
	cmd.Stderr = multi

	if lw.len == 0 {
		// this can mean that if it's a script, not a binary it may be missing a
		// shebang line. Check /tmp for the unpacked messages
		g.logger.Warn("entrypoint.sh produced no output on stdout/err")
	}

	err = cmd.Run()

	return out.String(), nil

}
