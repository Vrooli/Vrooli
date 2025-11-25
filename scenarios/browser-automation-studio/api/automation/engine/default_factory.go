package engine

import "github.com/sirupsen/logrus"

// DefaultFactory constructs an engine factory using the currently supported
// engines. Today this wires Browserless with sensible fallbacks; future engines
// can be registered here without touching callers.
func DefaultFactory(log *logrus.Logger) (Factory, error) {
	engines := []AutomationEngine{}

	browserlessEngine, err := NewBrowserlessEngineWithFallback(log, true)
	if err != nil {
		return nil, err
	}
	engines = append(engines, browserlessEngine)

	if pw, err := NewPlaywrightEngineWithDefault(log); err == nil {
		engines = append(engines, pw)
	} else if log != nil {
		log.WithError(err).Debug("Playwright engine not registered; driver URL missing")
	}

	return NewStaticFactory(engines...), nil
}
