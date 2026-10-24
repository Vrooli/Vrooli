package main

import (
	"io"
	"log/slog"
	"os"

	"github.com/vrooli/vrooli/internal/logx"
)

func createCommandLogger(globals globalOptions, stderr io.Writer) (*slog.Logger, func()) {
	logger, _, restore := logx.InstallAndReport(logx.Options{
		Component:      "vrooli",
		Writer:         stderr,
		Format:         globals.LogFormat(),
		Verbose:        globals.Verbose,
		Quiet:          resolveQuiet(globals, stderr),
		SetDefault:     true,
		RedirectStdlib: true,
	})
	return logger, restore
}

// resolveQuiet decides whether the structured-log level should be raised
// to warn. Explicit --verbose always wins. Explicit --quiet always wins.
// When neither is set, VROOLI_LOG_LEVEL is not explicitly set, and stderr
// is an interactive terminal, we treat the session as quiet so the info
// lifecycle stream stops leaking to humans — the full log file still
// captures info records via the lifecycle MultiWriter, and CI (non-TTY)
// keeps the full info stream on stderr.
func resolveQuiet(globals globalOptions, stderr io.Writer) bool {
	if globals.Verbose {
		return false
	}
	if globals.Quiet {
		return true
	}
	if os.Getenv(logx.LogLevelEnvVar) != "" {
		return false
	}
	return isTerminal(stderr)
}

// isTerminal reports whether w is an interactive terminal. Works without
// external dependencies by inspecting the underlying *os.File's mode.
// Anything that is not an *os.File (buffers, pipes, nil) is treated as
// non-interactive.
func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok || file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func debugLog(logger *slog.Logger, msg string, args ...any) {
	if logger == nil {
		return
	}
	logger.Debug(msg, args...)
}
