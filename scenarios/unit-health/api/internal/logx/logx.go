// Package logx owns the production logger adapter used at the API boundary.
// Domain consumers keep the narrower Logger interface they actually need.
package logx

import "log"

// Standard adapts the standard logger to a consumer-owned logger seam.
type Standard struct{ Logger *log.Logger }

func (s Standard) Printf(format string, args ...any) {
	if s.Logger != nil {
		s.Logger.Printf(format, args...)
	}
}

func Default() Standard { return Standard{Logger: log.Default()} }
