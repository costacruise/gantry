package main

import (
	"context"
	"time"
)

type Gantry struct {
	ctx context.Context
	src MessageSource

	logger Logger
}

func (g *Gantry) loop() {

	var ticker = time.NewTicker(60 * time.Second)

	// Ticker fires *after* the duration, not *every* duration, fire once to
	// avoid wait on first run
	g.getMessage()

	for {
		select {
		case <-g.ctx.Done():
			g.logger.Info("stopping context cancelled")
			ticker.Stop()
			return
		case <-ticker.C:
			g.logger.Debug("tick, checking for new messages")
			g.getMessage()
		}
	}

}

func (g *Gantry) Run() {
	g.loop()
}

func (g *Gantry) getMessage() {

	g.logger.Debugf("checking for message")

	// TODO:
	//   test all cases err and msg nil, and all permutations
	//   (nil msg, nil err = empty inbox)
	//   (msg nil error = message available)
	//   (err nil msg = error)
	msg, err := g.src.ReceiveMessageWithContext(g.ctx)
	if err != nil {
		g.logger.Fatalf("receive with context failed: %s", err)
		return
	}
	if msg == nil {
		g.logger.Debugf("no messages available for receipt")
		return
	}
	defer msg.Delete()
	g.logger.Infof("got message: %s", msg.Id())

	// = Payloader{g.logger}
	// p.,,,,
	// dir, err := g.unpackToFilesystem(msg)
	// err = g.runEntrypointIn(dir)

}
