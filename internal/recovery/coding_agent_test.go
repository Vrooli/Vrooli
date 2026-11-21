package recovery

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/vrooli/cli-core/agentcatalog/harness"
)

func TestCodingAgentDiagnosticTruncationPreservesValidUnicode(t *testing.T) {
	value := truncateCodingAgentString(strings.Repeat("x", 511) + "é more detail")
	if !utf8.ValidString(value) || !strings.HasSuffix(value, "...") || len(value) > 517 {
		t.Fatalf("bounded diagnostic split a Unicode rune: bytes=%d valid=%v", len(value), utf8.ValidString(value))
	}
	if got := truncateCodingAgentString(string([]byte{0xff})); !utf8.ValidString(got) {
		t.Fatal("invalid diagnostic bytes survived normalization")
	}
	if got := truncateCodingAgentText(errors.New(strings.Repeat("x", 511) + "é detail")); !utf8.ValidString(got) || got != value {
		t.Fatal("error diagnostics bypassed the shared Unicode-safe boundary")
	}
}

func TestCodingAgentRejectsInputsThatWouldPoisonItsLedger(t *testing.T) {
	for _, req := range []CodingAgentRequest{{Scenario: "../other-owner", Requester: "fixture"}, {Scenario: "web-console", Requester: strings.Repeat("x", 4097)}, {Scenario: "web-console", Requester: string([]byte{0xff})}} {
		b, err := NewCodingAgentBroker(filepath.Join(t.TempDir(), "receipts.json"), func(context.Context, string, CodingAgentRequest) error {
			t.Error("invalid admission launched a runner")
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := b.Submit(req); err == nil {
			t.Fatal("invalid input was admitted")
		}
		if records, err := b.Snapshot(); err != nil || len(records) != 0 {
			t.Fatalf("rejected admission poisoned the ledger: %+v %v", records, err)
		}
	}
}

func TestCodingAgentBrokerDeduplicatesPersistsAndFallsBack(t *testing.T) {
	path := t.TempDir() + "/coding-agent-records.json"
	base := time.Date(2026, 9, 10, 14, 0, 0, 0, time.UTC)
	var calls atomic.Int32
	done := make(chan struct{}, 1)
	runner := func(_ context.Context, name string, _ CodingAgentRequest) error {
		calls.Add(1)
		if name == "opencode" {
			return ErrCodingAgentUnavailable
		}
		done <- struct{}{}
		return nil
	}
	b, err := NewCodingAgentBroker(path, runner)
	if err != nil {
		t.Fatal(err)
	}
	b.now = func() time.Time { return base }
	b.runners = []string{"opencode", "codex"}
	first, err := b.Submit(CodingAgentRequest{Scenario: "agent-manager", Requester: "test", Reason: "unhealthy"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := b.Submit(CodingAgentRequest{Scenario: "agent-manager", Requester: "test", Reason: "same incident"})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("duplicate admission IDs differ: %s != %s", first.ID, second.ID)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("fallback runner did not complete")
	}
	waitCodingAgentOutcome(t, b, "succeeded")
	if got := calls.Load(); got != 2 {
		t.Fatalf("runner calls = %d, want opencode failure plus codex fallback", got)
	}

	reloaded, err := NewCodingAgentBroker(path, runner)
	if err != nil {
		t.Fatal(err)
	}
	records, err := reloaded.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Outcome != "succeeded" || records[0].Runner != "codex" {
		t.Fatalf("persisted records = %+v, want one successful codex record", records)
	}

	reloaded.now = func() time.Time { return base.Add(CodingAgentRecoveryWindow + time.Second) }
	third, err := reloaded.Submit(CodingAgentRequest{Scenario: "agent-manager", Requester: "test", Reason: "next window"})
	if err != nil {
		t.Fatal(err)
	}
	if third.ID == first.ID {
		t.Fatal("new rolling window reused the old admission")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("next-window runner did not complete")
	}
	waitCodingAgentOutcome(t, reloaded, "succeeded")
}

func waitCodingAgentOutcome(t *testing.T, b *CodingAgentBroker, outcome string) []CodingAgentRecord {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		records, err := b.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if len(records) > 0 && records[len(records)-1].Outcome == outcome {
			return records
		}
		select {
		case <-deadline.C:
			t.Fatalf("outcome did not become %s: %+v", outcome, records)
		case <-tick.C:
		}
	}
}

func TestCodingAgentBrokerConcurrentOwnersShareAdmissionAndPreserveUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.json")
	var calls atomic.Int32
	release := make(chan struct{})
	runner := func(context.Context, string, CodingAgentRequest) error {
		calls.Add(1)
		<-release
		return nil
	}
	// Construct every broker before admission, reproducing overlapping old
	// and replacement API processes that initially saw the same empty ledger.
	brokers := make([]*CodingAgentBroker, 12)
	for i := range brokers {
		b, err := NewCodingAgentBroker(path, runner)
		if err != nil {
			t.Fatal(err)
		}
		brokers[i] = b
	}
	var wg sync.WaitGroup
	ids := make(chan string, len(brokers))
	for _, b := range brokers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := b.Submit(CodingAgentRequest{Scenario: "web-console", Requester: "test"})
			if err != nil {
				t.Error(err)
				return
			}
			ids <- r.ID
		}()
	}
	wg.Wait()
	close(ids)
	var admitted string
	for id := range ids {
		if admitted != "" && admitted != id {
			t.Errorf("duplicate IDs: %s and %s", admitted, id)
		}
		admitted = id
	}
	// A distinct scenario must retain its receipt when the first worker saves
	// a later outcome from its original broker instance.
	if _, err := brokers[1].Submit(CodingAgentRequest{Scenario: "agent-manager", Requester: "test"}); err != nil {
		t.Fatal(err)
	}
	close(release)
	waitCodingAgentOutcome(t, brokers[0], "succeeded")
	deadline := time.After(3 * time.Second)
	for {
		records, err := brokers[0].Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if len(records) == 2 && records[0].Outcome == "succeeded" && records[1].Outcome == "succeeded" {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("updates lost: %+v", records)
		case <-time.After(time.Millisecond):
		}
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("runner calls=%d, want one per scenario", got)
	}
}

func TestCodingAgentBrokerAdmissionAcrossProcesses(t *testing.T) {
	const childKey = "VROOLI_RECOVERY_TEST_LEDGER"
	if path := os.Getenv(childKey); path != "" {
		b, err := NewCodingAgentBroker(path, func(context.Context, string, CodingAgentRequest) error { select {} })
		if err != nil {
			t.Fatal(err)
		}
		if _, err := b.Submit(CodingAgentRequest{Scenario: "web-console", Requester: "child"}); err != nil {
			t.Fatal(err)
		}
		return
	}
	path := filepath.Join(t.TempDir(), "receipts.json")
	var children []*exec.Cmd
	for range 4 {
		cmd := exec.Command(os.Args[0], "-test.run=^TestCodingAgentBrokerAdmissionAcrossProcesses$")
		cmd.Env = append(os.Environ(), childKey+"="+path)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		children = append(children, cmd)
	}
	for _, cmd := range children {
		if err := cmd.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	b, err := NewCodingAgentBroker(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	records, err := b.Snapshot()
	if err != nil || len(records) != 1 {
		t.Fatalf("cross-process receipts=%+v error=%v", records, err)
	}
}

func TestCodingAgentBrokerUnconfirmedWorkNeverFallsBackOrOverlaps(t *testing.T) {
	for _, outcome := range []string{"timeout", "ambiguous error"} {
		t.Run(outcome, func(t *testing.T) {
			var calls atomic.Int32
			b, err := NewCodingAgentBroker("", func(ctx context.Context, _ string, _ CodingAgentRequest) error {
				calls.Add(1)
				if outcome == "timeout" {
					<-ctx.Done()
					return nil
				}
				return errors.New("runner exited without confirming descendant termination")
			})
			if err != nil {
				t.Fatal(err)
			}
			b.timeout = time.Millisecond
			first, err := b.Submit(CodingAgentRequest{Scenario: "agent-manager", Requester: "test"})
			if err != nil {
				t.Fatal(err)
			}
			waitCodingAgentOutcome(t, b, "unconfirmed")
			current, err := b.Snapshot()
			if err != nil || len(current) != 1 || !current[0].FinishedAt.IsZero() || current[0].ObservationEndedAt == nil {
				t.Fatalf("observer completion became execution completion: %+v %v", current, err)
			}
			b.now = func() time.Time { return first.CreatedAt.Add(2 * CodingAgentRecoveryWindow) }
			next, err := b.Submit(CodingAgentRequest{Scenario: "agent-manager", Requester: "test"})
			if err != nil || next.ID != first.ID || calls.Load() != 1 {
				t.Fatalf("overlapping work: next=%+v error=%v calls=%d", next, err, calls.Load())
			}
		})
	}
}

func TestCodingAgentBrokerFailsClosedOnUnreadableLedger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.json")
	var calls atomic.Int32
	b, err := NewCodingAgentBroker(path, func(context.Context, string, CodingAgentRequest) error { calls.Add(1); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("corrupted"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Submit(CodingAgentRequest{Scenario: "web-console", Requester: "test"}); err == nil {
		t.Fatal("corrupt ledger admitted work")
	}
	if _, err := b.Snapshot(); err == nil {
		t.Fatal("snapshot hid the read failure")
	}
	if calls.Load() != 0 {
		t.Fatal("runner called without durable admission")
	}
}

func TestCodingAgentBrokerCancelledAdmissionStartsNothing(t *testing.T) {
	var calls atomic.Int32
	b, err := NewCodingAgentBroker("", func(context.Context, string, CodingAgentRequest) error { calls.Add(1); return nil })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := b.SubmitContext(ctx, CodingAgentRequest{Scenario: "web-console", Requester: "test"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("admission error=%v", err)
	}
	records, err := b.Snapshot()
	if err != nil || len(records) != 0 || calls.Load() != 0 {
		t.Fatalf("cancelled admission records=%+v error=%v calls=%d", records, err, calls.Load())
	}
}

func TestCodingAgentBrokerRestartDoesNotAssumeOldWorkStopped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.json")
	if err := os.WriteFile(path, []byte(`[{"id":"old-running","scenario":"web-console","requester":"old-autoheal","outcome":"running","created_at":"2026-11-01T00:00:00Z"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	b, err := NewCodingAgentBroker(path, func(context.Context, string, CodingAgentRequest) error { calls.Add(1); return nil })
	if err != nil {
		t.Fatal(err)
	}
	b.now = func() time.Time { return time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC) }
	receipt, err := b.Submit(CodingAgentRequest{Scenario: "web-console", Requester: "replacement-autoheal"})
	if err != nil || receipt.ID != "old-running" || calls.Load() != 0 {
		t.Fatalf("lost ownership admitted duplicate: receipt=%+v error=%v calls=%d", receipt, err, calls.Load())
	}
}

// Which harnesses may run an unattended investigation is a decision, so a new
// harness has to be placed on one side of it. Antigravity is left out: its live
// enforcement canary and live detection check are still pending.
func TestDefaultRunnersAccountForEveryDeclaredHarness(t *testing.T) {
	excluded := map[string]bool{"agy": true}
	included := map[string]bool{}
	for _, runner := range DefaultCodingAgentRunners {
		included[runner] = true
	}
	all := harness.Builtin().All()
	for _, descriptor := range all {
		if included[descriptor.ID] == excluded[descriptor.ID] {
			t.Errorf("harness %q must be in exactly one of DefaultCodingAgentRunners and the exclusions", descriptor.ID)
		}
	}
	if len(included)+len(excluded) != len(all) {
		t.Errorf("%d default and %d excluded runners, but %d declared harnesses", len(included), len(excluded), len(all))
	}
}
