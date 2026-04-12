package main

import (
	"io"
	"log/slog"

	"github.com/vrooli/vrooli/internal/logx"
)

func createCommandLogger(globals globalOptions, stderr io.Writer) (*slog.Logger, func()) {
	logger, _, restore := logx.Install(logx.Options{
		Component:      "vrooli",
		Writer:         stderr,
		Verbose:        globals.verbose,
		SetDefault:     true,
		RedirectStdlib: true,
	})
	return logger, restore
}

func debugLog(logger *slog.Logger, msg string, args ...any) {
	if logger == nil {
		return
	}
	logger.Debug(msg, args...)
}
