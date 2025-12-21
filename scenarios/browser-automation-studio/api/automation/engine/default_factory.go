package engine

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// DefaultFactory constructs an engine factory using Playwright as the
// sole automation engine.
func DefaultFactory(log *logrus.Logger) (Factory, error) {
	pw, err := NewPlaywrightEngineWithDefault(log)
	if err != nil {
		return nil, fmt.Errorf("playwright engine required: %w", err)
	}

	return NewStaticFactory(pw), nil
}

// DefaultFactoryWithRecordingsRoot constructs an engine factory with an explicit recordings root.
func DefaultFactoryWithRecordingsRoot(log *logrus.Logger, recordingsRoot string) (Factory, error) {
	pw, err := NewPlaywrightEngineWithRecordingsRoot(log, recordingsRoot)
	if err != nil {
		return nil, fmt.Errorf("playwright engine required: %w", err)
	}

	return NewStaticFactory(pw), nil
}
