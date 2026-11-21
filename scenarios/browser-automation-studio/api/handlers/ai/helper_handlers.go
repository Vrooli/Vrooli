package ai

import "github.com/sirupsen/logrus"

// HelperHandlers groups the AI browser helpers that share one automation
// runner and therefore one lifecycle owner.
type HelperHandlers struct {
	DOM             *DOMHandler
	Screenshot      *ScreenshotHandler
	ElementAnalysis *ElementAnalysisHandler
	AIAnalysis      *AIAnalysisHandler
}

// NewHelperHandlers wires every AI browser helper to the supplied runner. The
// caller owns runner construction and must provide the scenario's canonical
// browser/session lifecycle owner.
func NewHelperHandlers(log *logrus.Logger, runner AutomationRunner) HelperHandlers {
	dom := NewDOMHandler(log, WithDOMRunner(runner))
	return HelperHandlers{
		DOM:             dom,
		Screenshot:      NewScreenshotHandler(log, WithScreenshotRunner(runner)),
		ElementAnalysis: NewElementAnalysisHandler(log, WithElementRunner(runner)),
		AIAnalysis:      NewAIAnalysisHandler(log, dom),
	}
}
