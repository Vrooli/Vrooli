package measures

import (
	"testing"

	"github.com/vrooli/api-core/schedule"
)

func TestDeclarationsCoverStatefulDomains(t *testing.T) {
	if len(declarations()) != 10 {
		t.Fatalf("declarations=%d", len(declarations()))
	}
}

func TestHandlerWithCollectorsRejectsMissingCollector(t *testing.T) {
	if _, err := HandlerWithCollectors(schedule.System(), map[string]func() int{}); err == nil {
		t.Fatal("missing collector must fail registration")
	}
}
