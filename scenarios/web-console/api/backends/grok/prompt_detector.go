// Package grok holds Grok CLI harness behavior for Web Console.
package grok

import "web-console/internal/backend"

// promptDetector is the Grok CLI screen detector. No Grok screen has been
// captured as a fixture yet, so it reports nothing and the session's activity
// falls back to the output clock (working while output arrives, otherwise
// unknown). Add behavior only together with a captured fixture.
type promptDetector struct{}

// DefaultPromptDetector returns the Grok CLI screen detector.
func DefaultPromptDetector() backend.PromptDetector { return promptDetector{} }

func (promptDetector) Analyze(backend.ScreenView) backend.PromptAnalysis {
	return backend.PromptAnalysis{}
}
