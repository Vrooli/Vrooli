package capacitysync

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
)

// fakePS returns a fixed loaded-model list (or an error to simulate /api/ps down).
type fakePS struct {
	running []ensure.RunningModel
	err     error
}

func (f fakePS) ListRunning(context.Context) ([]ensure.RunningModel, error) {
	return f.running, f.err
}

// recExec records the vrooli capacity calls and returns canned list output.
type recExec struct {
	listJSON string
	calls    []string
}

func (r *recExec) exec(_ context.Context, name string, args ...string) ([]byte, error) {
	joined := name + " " + strings.Join(args, " ")
	r.calls = append(r.calls, joined)
	if len(args) >= 2 && args[0] == "capacity" && args[1] == "list" {
		return []byte(r.listJSON), nil
	}
	return []byte("{}"), nil
}

func newHandlers(ps fakePS, listJSON string) (*Handlers, *recExec) {
	rec := &recExec{listJSON: listJSON}
	h := &Handlers{
		Stdout:    &strings.Builder{},
		Stderr:    &strings.Builder{},
		GetEnv:    func(string) string { return "" },
		NewClient: func() psClient { return ps },
		Exec:      rec.exec,
	}
	return h, rec
}

func callContains(calls []string, sub string) bool {
	for _, c := range calls {
		if strings.Contains(c, sub) {
			return true
		}
	}
	return false
}

// A model loaded with no existing claim → the poller creates the declared
// preferred/floor reservation rather than making the reservation disappear
// between model loads.
func TestSyncClaimsWhenModelLoadsWithNoClaim(t *testing.T) {
	ps := fakePS{running: []ensure.RunningModel{{Name: "qwen3:4b", SizeVRAM: 4 << 30}}}
	h, rec := newHandlers(ps, `{"claims":[]}`)
	h.syncOnce(context.Background())
	if !callContains(rec.calls, "capacity claim") {
		t.Fatalf("expected a claim call, got %v", rec.calls)
	}
	if !callContains(rec.calls, "--preferred "+itoa(11<<30)) || !callContains(rec.calls, "--floor "+itoa(3<<30)) {
		t.Errorf("claim must carry the declared model ladder, got %v", rec.calls)
	}
	if !callContains(rec.calls, `"steps":[{"label":"qwen3.5:9b"`) {
		t.Errorf("claim must include the policy-derived degrade profile, got %v", rec.calls)
	}
}

// Everything unloaded with an active claim → the poller releases it (no churn).
func TestSyncReleasesWhenAllUnloaded(t *testing.T) {
	ps := fakePS{running: nil}
	h, rec := newHandlers(ps, `{"claims":[{"claim_id":"clm-o","owner_id":"ollama","amount_bytes":4294967296,"generation":3}]}`)
	h.syncOnce(context.Background())
	if !callContains(rec.calls, "capacity release --claim-id clm-o") {
		t.Fatalf("expected a release of the ollama claim, got %v", rec.calls)
	}
	if callContains(rec.calls, "capacity claim") {
		t.Errorf("must not claim when nothing is loaded, got %v", rec.calls)
	}
}

// Footprint steady → the poller only heartbeats (keeps the claim alive).
func TestSyncHeartbeatsWhenSteady(t *testing.T) {
	ps := fakePS{running: []ensure.RunningModel{{Name: "qwen3:4b", SizeVRAM: 4 << 30}}}
	h, rec := newHandlers(ps, `{"claims":[{"claim_id":"clm-o","owner_id":"ollama","amount_bytes":4294967296,"generation":3}]}`)
	h.syncOnce(context.Background())
	if !callContains(rec.calls, "capacity heartbeat --claim-id clm-o --generation 3") {
		t.Fatalf("expected a heartbeat, got %v", rec.calls)
	}
	if callContains(rec.calls, "capacity release") || callContains(rec.calls, "capacity claim") {
		t.Errorf("steady footprint must not churn the claim, got %v", rec.calls)
	}
}

// /api/ps down → fail-open: the poller touches nothing.
func TestSyncFailOpenOnPollError(t *testing.T) {
	ps := fakePS{err: context.DeadlineExceeded}
	h, rec := newHandlers(ps, `{"claims":[{"claim_id":"clm-o","owner_id":"ollama","amount_bytes":4294967296,"generation":3}]}`)
	h.syncOnce(context.Background())
	for _, c := range rec.calls {
		if strings.Contains(c, "capacity claim") || strings.Contains(c, "capacity release") || strings.Contains(c, "capacity heartbeat") {
			t.Fatalf("poll error must leave the ledger unchanged, got call %q", c)
		}
	}
}

// A materially different footprint → resize (release + reclaim).
func TestSyncResizesOnFootprintChange(t *testing.T) {
	ps := fakePS{running: []ensure.RunningModel{{Name: "qwen3:30b", SizeVRAM: 12 << 30}}}
	h, rec := newHandlers(ps, `{"claims":[{"claim_id":"clm-o","owner_id":"ollama","amount_bytes":4294967296,"generation":3}]}`)
	h.syncOnce(context.Background())
	if !callContains(rec.calls, "capacity release --claim-id clm-o") || !callContains(rec.calls, "capacity claim") {
		t.Fatalf("a materially-changed footprint must release+reclaim, got %v", rec.calls)
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
