package engine

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// DefaultFactory constructs the local Playwright and target-owned Android
// WebView engines over one shared CDP session implementation.
func DefaultFactory(log *logrus.Logger) (Factory, error) {
	pw, err := NewPlaywrightEngineWithDefault(log)
	if err != nil {
		return nil, fmt.Errorf("playwright engine required: %w", err)
	}

	webview, err := NewAndroidWebViewEngine(pw)
	if err != nil {
		return nil, err
	}
	return NewStaticFactory(pw, webview), nil
}

// DefaultFactoryWithRecordingsRoot constructs an engine factory with an explicit recordings root.
func DefaultFactoryWithRecordingsRoot(log *logrus.Logger, recordingsRoot string) (Factory, error) {
	pw, err := NewPlaywrightEngineWithRecordingsRoot(log, recordingsRoot)
	if err != nil {
		return nil, fmt.Errorf("playwright engine required: %w", err)
	}

	webview, err := NewAndroidWebViewEngine(pw)
	if err != nil {
		return nil, err
	}
	return NewStaticFactory(pw, webview), nil
}
