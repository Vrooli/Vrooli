package readiness

import (
	"context"
	"testing"
	"time"
)

func TestPromotionClientRequiresConfiguredInputs(t *testing.T) {
	client := NewPromotionClient()
	if _, err := client.PublishReadinessFact(context.Background(), "", "demo", "abc", true, time.Time{}); err == nil {
		t.Fatal("expected missing node refusal")
	}
}
