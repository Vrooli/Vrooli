// Package logx centralizes project-level logger construction so the CLI and API
// share one logging contract for level parsing, formatting, default logger
// installation, and stdlib log redirection.
package logx

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
)

const LogLevelEnvVar = "VROOLI_LOG_LEVEL"

// Options configures the logger shape used across project-level Go commands.
type Options struct {
	Component      string
	Writer         io.Writer
	JSON           bool
	Verbose        bool
	SetDefault     bool
	RedirectStdlib bool
	ReplaceAttr    func(groups []string, a slog.Attr) slog.Attr
}

// Diagnostics captures decisions made while configuring a logger.
type Diagnostics struct {
	Level    slog.Level
	Warnings []string
}

// ParseLevel parses a user-provided log level string into a slog level.
func ParseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error", "err":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unsupported log level %q", strings.TrimSpace(value))
	}
}

// LevelFromEnv returns the slog level controlled by VROOLI_LOG_LEVEL.
func LevelFromEnv() (slog.Level, error) {
	return ParseLevel(os.Getenv(LogLevelEnvVar))
}

// New returns a logger configured for project-level commands together with the
// effective level and any configuration warnings.
func New(opts Options) (*slog.Logger, Diagnostics) {
	writer := opts.Writer
	if writer == nil {
		writer = os.Stderr
	}

	level, warnings := resolveLevel(opts.Verbose)
	handlerOptions := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: opts.ReplaceAttr,
	}

	var handler slog.Handler
	if opts.JSON {
		handler = slog.NewJSONHandler(writer, handlerOptions)
	} else {
		handler = slog.NewTextHandler(writer, handlerOptions)
	}

	logger := slog.New(handler)
	if strings.TrimSpace(opts.Component) != "" {
		logger = logger.With("component", opts.Component)
	}

	return logger, Diagnostics{
		Level:    level,
		Warnings: warnings,
	}
}

// Install creates a logger, optionally installs it as the slog default, and
// optionally redirects the stdlib logger. The returned restore function undoes
// any stdlib redirection and is a no-op when RedirectStdlib is false.
func Install(opts Options) (*slog.Logger, Diagnostics, func()) {
	logger, diagnostics := New(opts)
	if opts.SetDefault {
		slog.SetDefault(logger)
	}

	restoreStdlib := func() {}
	if opts.RedirectStdlib {
		restoreStdlib = RedirectStandardLibrary(logger, diagnostics.Level)
	}

	for _, warning := range diagnostics.Warnings {
		logger.Warn(warning, "env_var", LogLevelEnvVar, "value", strings.TrimSpace(os.Getenv(LogLevelEnvVar)))
	}

	return logger, diagnostics, restoreStdlib
}

func resolveLevel(verbose bool) (slog.Level, []string) {
	level, err := LevelFromEnv()
	warnings := []string{}
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Invalid %s value; using info level", LogLevelEnvVar))
		level = slog.LevelInfo
	}
	if verbose && level > slog.LevelDebug {
		level = slog.LevelDebug
	}
	return level, warnings
}

// RedirectStandardLibrary routes log.Printf-style calls through slog so older
// code can share one logger during the migration. The returned restore
// function reinstates the prior stdlib logger configuration.
func RedirectStandardLibrary(logger *slog.Logger, level slog.Leveler) func() {
	if logger == nil {
		return func() {}
	}

	originalWriter := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()

	if level == nil {
		level = slog.LevelInfo
	}
	resolvedLevel := slog.LevelInfo
	if level != nil {
		resolvedLevel = level.Level()
	}
	stdLogger := slog.NewLogLogger(logger.Handler(), resolvedLevel)
	log.SetFlags(0)
	log.SetPrefix("")
	log.SetOutput(stdLogger.Writer())

	return func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	}
}
