package intelligence

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrProviderSurvivesErrorWrapping(t *testing.T) {
	err := fmt.Errorf("provider request failed: %w", ErrProvider)
	if !errors.Is(err, ErrProvider) {
		t.Fatalf("wrapped provider error must retain its domain identity: %v", err)
	}
}

func TestGatewayErrorsSurviveErrorWrapping(t *testing.T) {
	for _, domainErr := range []error{
		ErrNoAPIKeyConfigured,
		ErrModelNotAllowed,
		ErrAIGatewayUnavailable,
		ErrStreamingNotSupported,
	} {
		wrapped := fmt.Errorf("gateway failure: %w", domainErr)
		if !errors.Is(wrapped, domainErr) {
			t.Fatalf("wrapped gateway error must retain domain identity: %v", domainErr)
		}
	}
}
