package bridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// These are exact canonical bytes retained by the actual Events admission
// transaction in terminal-outbox-20261107/final-cross-owner-fixture/events.sqlite,
// not a protobuf reconstruction. All writes below use isolated fixture stores.
func terminalFixture(t *testing.T) store.Event {
	t.Helper()
	raw, err := os.ReadFile("testdata/terminal-canonical.pb")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != "2c65bd09bcbf9b78c79c6732a1e596b755cc090fa34667a4e8bcee25e2967bb7" {
		t.Fatal("acceptance fixture digest changed")
	}
	var env domain.EventEnvelope
	if err = proto.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	return store.Event{EventID: env.EventId, EventType: env.EventType, SourceScenario: env.Source.Scenario, CreatedAt: env.OccurredAt.AsTime(), Payload: raw}
}
func assertTerminalGolden(t *testing.T, name string, raw []byte) {
	t.Helper()
	path := "testdata/" + name
	if os.Getenv("VROOLI_BRIDGE_WRITE_TERMINAL_GOLDEN") == "1" {
		if err := os.WriteFile(path, append(raw, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	expected, err := os.ReadFile(path)
	if err != nil || strings.TrimSpace(string(expected)) != string(raw) {
		t.Fatalf("golden %s mismatch: %s (%v)", name, raw, err)
	}
}
func TestTerminalSelectedAcceptanceFixture(t *testing.T) {
	r, s, b := selectedFixture(t)
	r.boundary = &evidenceTestBoundary{b, "fixture-native-events-epoch"}
	e := terminalFixture(t)
	if _, err := s.Insert(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	raw, err := r.ReadSelectedEvidence(context.Background(), e.EventID, allow, validate)
	if err != nil {
		t.Fatal(err)
	}
	var got SelectedEvidence
	if err = json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.SourceInstance == got.Locator.SourceInstance || got.Status != "complete" || got.Category != "" || got.Locator.LifecycleVersion != "1" || got.Locator.TerminalObservationSHA256 != "617806cfea60cbd5763a8202fac7deb8565075259e6bf172a1ced23c88921b84" {
		t.Fatalf("invalid minimized terminal evidence: %s", raw)
	}
	assertTerminalGolden(t, "selected-terminal-event-evidence-v2.json", raw)
	f, g, _, _ := wakeFixture(t)
	f.now = func() time.Time { return time.Date(2026, 11, 7, 5, 16, 55, 481988288, time.UTC) }
	body, selected, err := f.Project(context.Background(), e)
	if err != nil || !selected {
		t.Fatalf("terminal not projected: %v", err)
	}
	w, err := decodeWake(body, g.policy, e.EventID)
	if err != nil || w.Events[0].Locator != got.Locator {
		t.Fatalf("terminal wake rejected: %v", err)
	}
	assertTerminalGolden(t, "terminal-wake-v2.json", []byte(body))
	if SubscriptionSpec().Enabled || SubscriptionSpec().EventPattern != RunEventType {
		t.Fatal("production subscription changed")
	}
}
func TestTerminalEvidenceRejectsMalformedFacts(t *testing.T) {
	for _, kind := range []string{"extra", "run", "source", "epoch", "version-zero", "version-leading-zero", "version-overflow", "version-number", "digest", "status", "missing", "index", "occurrence"} {
		t.Run(kind, func(t *testing.T) {
			r, s, b := selectedFixture(t)
			r.boundary = &evidenceTestBoundary{b, "fixture-native-events-epoch"}
			e := terminalFixture(t)
			var env domain.EventEnvelope
			_ = proto.Unmarshal(e.Payload, &env)
			var facts structpb.Struct
			_ = env.Data.UnmarshalTo(&facts)
			key, value := "", ""
			switch kind {
			case "extra":
				key, value = "instructions", "PRIVATE"
			case "run":
				key, value = "run_id", "00000000-0000-0000-0000-000000000000"
			case "source":
				key, value = "source_instance", "bad scope"
			case "epoch":
				key, value = "source_epoch", ""
			case "version-zero":
				key, value = "lifecycle_version", "0"
			case "version-leading-zero":
				key, value = "lifecycle_version", "01"
			case "version-overflow":
				key, value = "lifecycle_version", "9223372036854775808"
			case "version-number":
				facts.Fields["lifecycle_version"] = structpb.NewNumberValue(1)
			case "digest":
				key, value = "terminal_observation_sha256", "bad"
			case "status":
				key, value = "status", "running"
			case "missing":
				delete(facts.Fields, "source_epoch")
			case "index":
				e.SourceScenario = "other"
			case "occurrence":
				e.CreatedAt = e.CreatedAt.AddDate(0, 0, 1)
			}
			if key != "" {
				facts.Fields[key] = structpb.NewStringValue(value)
			}
			env.Data, _ = anypb.New(&facts)
			e.Payload, _ = proto.Marshal(&env)
			if _, err := s.Insert(context.Background(), e); err != nil {
				t.Fatal(err)
			}
			raw, err := r.ReadSelectedEvidence(context.Background(), e.EventID, allow, validate)
			if err == nil || raw != nil {
				t.Fatalf("released %s: %s", kind, raw)
			}
			f, _, _, _ := wakeFixture(t)
			body, selected, _ := f.Project(context.Background(), e)
			if selected || body != "" {
				t.Fatal("projected malformed terminal")
			}
		})
	}
}
func TestTerminalStatusAndLegacyLocatorVariants(t *testing.T) {
	for _, status := range []string{"complete", "failed", "cancelled", "unknown"} {
		e := terminalFixture(t)
		var env domain.EventEnvelope
		_ = proto.Unmarshal(e.Payload, &env)
		var facts structpb.Struct
		_ = env.Data.UnmarshalTo(&facts)
		facts.Fields["status"] = structpb.NewStringValue(status)
		l, category, got, err := projectRunFacts(TerminalEventType, &facts)
		if err != nil || got != status || category != "" || !validRunLocator(TerminalEventType, l) || validRunLocator(RunEventType, l) {
			t.Fatalf("bad terminal variant %s", status)
		}
		l.Schema = "agent-manager.run-event.v1"
		if validRunLocator(RunEventType, l) {
			t.Fatal("legacy accepted terminal-only fields")
		}
	}
}

func TestTerminalWakeCannotUseLegacySubscription(t *testing.T) {
	f, _, signer, _ := wakeFixture(t)
	e := terminalFixture(t)
	body, selected, err := f.Project(context.Background(), e)
	if err != nil || !selected {
		t.Fatal(err)
	}
	sub := SubscriptionSpec()
	sub.Enabled = true
	sub.ID = 7
	err = f.Deliver(context.Background(), sub, subscription.QueuedDelivery{SubscriptionID: 7, EventID: e.EventID, PayloadJSON: body})
	if err == nil || signer.calls != 0 {
		t.Fatal("terminal wake acquired legacy subscription authority")
	}
}
