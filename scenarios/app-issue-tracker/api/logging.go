package main

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

func logInfo(msg string, attrs ...any) {
	getLogger().Info(msg, attrs...)
}

func logWarn(msg string, attrs ...any) {
	getLogger().Warn(msg, attrs...)
}

func logError(msg string, attrs ...any) {
	getLogger().Error(msg, attrs...)
}

func logErrorErr(msg string, err error, attrs ...any) {
	attrs = append(attrs, slog.Any("error", err))
	getLogger().Error(msg, attrs...)
}

func logDebug(msg string, attrs ...any) {
	getLogger().Debug(msg, attrs...)
}

func withLogger(logger *slog.Logger) func() {
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
