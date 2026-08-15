package capacitysync

import (
	"context"
	"strings"
	"testing"
)

func TestSyncOnceHeartbeatsActiveClaim(t *testing.T) {
	var calls []string
	h := &Handlers{
		Exec: func(_ context.Context, name string, args ...string) ([]byte, error) {
			calls = append(calls, name+" "+strings.Join(args, " "))
			return []byte(`{"claims":[{"claim_id":"clm-r","owner_id":"reranker","generation":4}]}`), nil
		},
	}
	h.syncOnce(context.Background())
	if len(calls) != 2 || !strings.Contains(calls[1], "capacity heartbeat --claim-id clm-r --generation 4") {
		t.Fatalf("calls = %v, want active claim heartbeat", calls)
	}
}

func TestSyncOnceReadmitsMissingClaim(t *testing.T) {
	var calls []string
	h := &Handlers{
		Exec: func(_ context.Context, name string, args ...string) ([]byte, error) {
			calls = append(calls, name+" "+strings.Join(args, " "))
			return []byte(`{"claims":[]}`), nil
		},
	}
	h.syncOnce(context.Background())
	if len(calls) != 2 || !strings.Contains(calls[1], "resource start reranker --json") {
		t.Fatalf("calls = %v, want resource re-admission", calls)
	}
}
