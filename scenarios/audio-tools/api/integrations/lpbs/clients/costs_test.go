package clients

import "testing"

func TestCostConstants(t *testing.T) {
	if CostTranscribePerSecond <= 0 {
		t.Fatalf("CostTranscribePerSecond must be positive")
	}
	if CostSynthesizePerKChars <= 0 {
		t.Fatalf("CostSynthesizePerKChars must be positive")
	}
	if CostSummarizePerKTokens <= 0 {
		t.Fatalf("CostSummarizePerKTokens must be positive")
	}
	if AppBundleKey == "" {
		t.Fatalf("AppBundleKey must be set")
	}
}
