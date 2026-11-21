package ai

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestNewHelperHandlersSharesOneRunnerAcrossBrowserHelpers(t *testing.T) {
	runner := NewMockAutomationRunner()
	handlers := NewHelperHandlers(logrus.New(), runner)

	require.Same(t, runner, handlers.DOM.runner)
	require.Same(t, runner, handlers.Screenshot.runner)
	require.Same(t, runner, handlers.ElementAnalysis.runner)
	analysis, ok := handlers.AIAnalysis.analyzer.(*AIElementAnalyzer)
	require.True(t, ok)
	require.Same(t, handlers.DOM, analysis.domExtractor)
}

func TestDefaultAutomationRunnerDoesNotCreateAPrivateSessionManager(t *testing.T) {
	_, err := NewDefaultAutomationRunner(logrus.New())
	require.ErrorContains(t, err, "automation engine factory is required")
}
