package logx

import (
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
)

const logLevelEnvVar = "VROOLI_LOG_LEVEL"

// Options configures the logger shape used across project-level Go commands.
type Options struct {
	Name   string
	Writer io.Writer
	JSON   bool
}

// LevelFromEnv returns the slog level controlled by VROOLI_LOG_LEVEL.
func LevelFromEnv() slog.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(logLevelEnvVar))) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// New returns a logger configured for project-level commands.
func New(opts Options) *slog.Logger {
	writer := opts.Writer
	if writer == nil {
		writer = os.Stderr
	}

	handlerOptions := &slog.HandlerOptions{Level: LevelFromEnv()}

	var handler slog.Handler
	if opts.JSON {
		handler = slog.NewJSONHandler(writer, handlerOptions)
	} else {
		handler = slog.NewTextHandler(writer, handlerOptions)
	}

	logger := slog.New(handler)
	if strings.TrimSpace(opts.Name) != "" {
		logger = logger.With("component", opts.Name)
	}
	return logger
}

// RedirectStandardLibrary routes log.Printf-style calls through slog so older
// code can share one logger during the migration.
func RedirectStandardLibrary(logger *slog.Logger) {
	if logger == nil {
		return
	}

	stdLogger := slog.NewLogLogger(logger.Handler(), LevelFromEnv())
	log.SetFlags(0)
	log.SetOutput(stdLogger.Writer())
}
