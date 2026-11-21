package capacity

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// A canceled context makes this safe against a constructor that regresses the
// test-isolation guard: artifact-lock admission must refuse before SQLite opens.
// The stronger desired contract is that the missing test path is rejected first.
func TestExternalReviewBrokerImplicitTestPathRefusedBeforeInitialization(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, path := range []string{"", " \t "} {
		t.Run("path="+path, func(t *testing.T) {
			broker, err := NewBroker(ctx, path)
			if broker != nil {
				_ = broker.Close()
				t.Fatal("implicit test path unexpectedly constructed a broker")
			}
			if err == nil || !strings.Contains(err.Error(), "tests must pass an explicit") || errors.Is(err, context.Canceled) {
				t.Fatalf("implicit test path must be refused before initialization/cancellation; got %v", err)
			}
		})
	}
}
