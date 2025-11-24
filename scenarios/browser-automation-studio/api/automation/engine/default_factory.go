package engine

import "github.com/sirupsen/logrus"

// DefaultFactory constructs an engine factory using the currently supported
// engines. Today this wires Browserless with sensible fallbacks; future engines
// can be registered here without touching callers.
func DefaultFactory(log *logrus.Logger) (Factory, error) {
	engine, err := NewBrowserlessEngineWithFallback(log, true)
	if err != nil {
		return nil, err
	}
	return NewStaticFactory(engine), nil
}
