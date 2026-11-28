package handlers

import (
	"github.com/sirupsen/logrus"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autosession "github.com/vrooli/browser-automation-studio/automation/session"
	aihandlers "github.com/vrooli/browser-automation-studio/handlers/ai"
)

func newAIHelperHandlers(log *logrus.Logger, broker *autosession.Manager) (aihandlers.HelperHandlers, aihandlers.AutomationRunner) {
	var runner aihandlers.AutomationRunner
	if broker != nil {
		factory, err := autoengine.DefaultFactoryWithSessionManager(log, broker)
		if err != nil {
			if log != nil {
				log.WithError(err).Warn("Failed to initialize AI helper engine factory; AI browser helpers will be unavailable")
			}
		} else {
			configured, err := aihandlers.NewDefaultAutomationRunner(log, aihandlers.WithEngineFactory(factory))
			if err != nil {
				if log != nil {
					log.WithError(err).Warn("Failed to initialize AI helper runner; AI browser helpers will be unavailable")
				}
			} else {
				runner = configured
			}
		}
	}
	return aihandlers.NewHelperHandlers(log, runner), runner
}
