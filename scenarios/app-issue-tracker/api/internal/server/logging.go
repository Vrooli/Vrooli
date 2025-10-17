package server

import (
	"log/slog"
	"os"
	"sync"
)

var (
	loggerOnce sync.Once
	loggerMu   sync.RWMutex
	baseLogger *slog.Logger
)

func getLogger() *slog.Logger {
	loggerOnce.Do(func() {
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})
		baseLogger = slog.New(handler)
	})
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return baseLogger
}

func LogInfo(msg string, attrs ...any) {
	getLogger().Info(msg, attrs...)
}

func LogWarn(msg string, attrs ...any) {
	getLogger().Warn(msg, attrs...)
}

func LogError(msg string, attrs ...any) {
	getLogger().Error(msg, attrs...)
}

func LogErrorErr(msg string, err error, attrs ...any) {
	attrs = append(attrs, slog.Any("error", err))
	getLogger().Error(msg, attrs...)
}

func LogDebug(msg string, attrs ...any) {
	getLogger().Debug(msg, attrs...)
}

func WithLogger(logger *slog.Logger) func() {
	loggerOnce.Do(func() {
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})
		baseLogger = slog.New(handler)
	})

	loggerMu.Lock()
	old := baseLogger
	baseLogger = logger
	loggerMu.Unlock()

	return func() {
		loggerMu.Lock()
		baseLogger = old
		loggerMu.Unlock()
	}
}
