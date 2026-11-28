// Package logx defines the API's injectable logging boundary.
package logx

import "log"

// Logger is the small structured-message surface used by API handlers.
type Logger interface {
	Printf(format string, args ...any)
	Infof(format string, args ...any)
}

type standard struct{ logger *log.Logger }

func (s standard) Printf(format string, args ...any) { s.logger.Printf(format, args...) }
func (s standard) Infof(format string, args ...any)  { s.logger.Printf(format, args...) }

// NewStandard adapts the process logger at the composition root.
func NewStandard(logger *log.Logger) Logger {
	if logger == nil {
		logger = log.Default()
	}
	return standard{logger: logger}
}
