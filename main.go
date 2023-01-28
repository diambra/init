package main

import (
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func assert(logger log.Logger, err error) {
	if err == nil {
		return
	}
	level.Error(logger).Log("msg", err.Error())
	os.Exit(1)
}

func main() {
	var (
		logger = log.NewLogfmtLogger(os.Stderr)

		s = os.Getenv("SOURCES")
	)
	if s == "" {
		level.Info(logger).Log("msg", "SOURCES not set, exiting")
		os.Exit(0)
	}

	init, err := NewInitializerFromString(logger, s)
	if err != nil {
		level.Error(logger).Log("msg", err.Error())
		os.Exit(1)
	}
	if err := init.Init(); err != nil {
		level.Error(logger).Log("msg", err.Error())
		os.Exit(1)
	}
}
