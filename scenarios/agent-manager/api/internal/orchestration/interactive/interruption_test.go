package interactive

import "testing"

func TestClassifyScreenInterruption(t *testing.T) {
	tests := []struct {
		name, screen, kind string
		want bool
	}{
		{"capacity", "Selected model is at capacity. Please try a different model.", "model_capacity", true},
		{"rate limit", "429: rate limit exceeded; retry later", "rate_limit", true},
		{"provider unavailable", "Provider temporarily unavailable", "provider_unavailable", true},
		{"ordinary prose", "The agent discussed model capacity in the report", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ClassifyScreenInterruption(tt.screen)
			if ok != tt.want || (ok && got.Kind != tt.kind) {
				t.Fatalf("classification = (%+v, %v), want kind=%q detected=%v", got, ok, tt.kind, tt.want)
			}
		})
	}
}
