package routing

import "context"

// ClassifyResult and Classifier are retained solely for in-process consumers
// that still construct Deps in tests. The production wiring does not provide a
// Classifier, and no Ollama classifier implementation remains in this package.
// They can be removed once downstream fixtures stop setting the retired seam.
type ClassifyResult struct {
	ProviderIDs []string
	Types       []string
	Confidence  float64
	Rationale   string
	WebShaped   bool
}

type Classifier interface {
	Classify(context.Context, string, []ProviderProfile) (ClassifyResult, error)
	Available(context.Context) bool
}
