package events

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	eventsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/events"
)

func TestEventRowsAndLimitValidation(t *testing.T) {
	if got := eventRows(nil); len(got) != 1 {
		t.Fatalf("empty event rows = %#v", got)
	}
	if got := eventRows([]*eventsv1.Event{{Type: "session.created", SessionId: "abcdefghi", Timestamp: "2026-01-01T00:00:00Z"}}); len(got) != 1 {
		t.Fatalf("event rows = %#v", got)
	}
	h := &handlers{}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "limit"}}}, Flags: map[string]string{"limit": "bad"}})
	if err := h.run(ctx); err == nil {
		t.Fatal("invalid limit unexpectedly succeeded")
	}
}
