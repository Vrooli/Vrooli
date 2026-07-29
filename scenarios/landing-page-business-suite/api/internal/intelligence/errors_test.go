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
