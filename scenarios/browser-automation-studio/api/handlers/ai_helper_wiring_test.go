package handlers

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	autosession "github.com/vrooli/browser-automation-studio/automation/session"
	aihandlers "github.com/vrooli/browser-automation-studio/handlers/ai"
)

func TestAIHelperWiringRequiresAndUsesTheCanonicalSessionBroker(t *testing.T) {
	log := logrus.New()
	broker, err := autosession.NewManager(autosession.WithLogger(log))
	require.NoError(t, err)

	withoutBroker, missingRunner := newAIHelperHandlers(log, nil)
	require.Nil(t, missingRunner)
	require.NotNil(t, withoutBroker.DOM)
	require.NotNil(t, withoutBroker.Screenshot)
	require.NotNil(t, withoutBroker.ElementAnalysis)

	withBroker, runner := newAIHelperHandlers(log, broker)
	require.NotNil(t, runner)
	require.IsType(t, &aihandlers.DefaultAutomationRunner{}, runner)
	require.NotNil(t, withBroker.DOM)
	require.NotNil(t, withBroker.Screenshot)
	require.NotNil(t, withBroker.ElementAnalysis)
	require.NotNil(t, withBroker.AIAnalysis)
}
