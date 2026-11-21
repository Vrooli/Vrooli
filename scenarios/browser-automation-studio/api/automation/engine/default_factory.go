package engine

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/session"
)

// DefaultFactoryWithSessionManager builds execution engines over the API's
// canonical session broker. Requiring the broker prevents engine construction
// from silently creating a second session lifecycle owner.
func DefaultFactoryWithSessionManager(log *logrus.Logger, broker *session.Manager) (Factory, error) {
	pw, err := NewPlaywrightEngineWithSessionManager(broker, log)
	if err != nil {
		return nil, fmt.Errorf("playwright engine required: %w", err)
	}
	webview, err := NewAndroidWebViewEngine(pw)
	if err != nil {
		return nil, err
	}
	return NewStaticFactory(pw, webview), nil
}
