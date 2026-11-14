package bridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"os"
	"strings"
	"testing"
	"time"
)

type evidenceTestBoundary struct {
	*testBoundary
	source string
}

func (b *evidenceTestBoundary) WithinEvidence(ctx context.Context, fn func(context.Context, string) error) error {
	return b.Within(ctx, func(inner context.Context) error { return fn(inner, b.source) })
}
func TestSelectedEvidenceExactDigestAndMinimization(t *testing.T) {
	r, s, b := selectedFixture(t)
	r.boundary = &evidenceTestBoundary{b, "fixture-native-events-epoch"}
	_, _, _, e := wakeFixture(t)
	// Unknown wire bytes prove the digest is not of a JSON reconstruction.
	e.Payload = protowire.AppendTag(e.Payload, 100, protowire.VarintType)
	e.Payload = protowire.AppendVarint(e.Payload, 7)
	if _, err := s.Insert(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	checks := 0
	raw, err := r.ReadSelectedEvidence(context.Background(), e.EventID, func(ctx context.Context) error {
		checks++
		if !b.held {
			t.Fatal("boundary not held")
		}
		if d, ok := ctx.Deadline(); !ok || time.Until(d) > 8*time.Second {
			t.Fatal("unbounded")
		}
		return nil
	}, validate)
	if err != nil {
		t.Fatal(err)
	}
	var got SelectedEvidence
	if err = json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(e.Payload)
	if got.SourceContentDigest != hex.EncodeToString(sum[:]) || got.SourceInstance != "fixture-native-events-epoch" || got.Category != "run-completed" || got.Locator.RunID != "11111111-1111-4111-8111-111111111111" || checks != 2 {
		t.Fatalf("bad evidence: %s", raw)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(raw, &fields); err != nil || len(fields) != 8 {
		t.Fatalf("unexpected fields: %s", raw)
	}
	for _, v := range []string{"PRIVATE", "private.test", "instructions", "workReferences", "body", "title"} {
		if strings.Contains(string(raw), v) {
			t.Fatalf("payload leak: %s", raw)
		}
	}
	golden, err := os.ReadFile("testdata/selected-event-evidence-v2.json")
	if err != nil {
		t.Fatalf("golden missing; actual=%s", raw)
	}
	if string(raw) != strings.TrimSpace(string(golden)) {
		t.Fatalf("golden mismatch; actual=%s", raw)
	}
	missing, err := r.ReadSelectedEvidence(context.Background(), "missing", allow, validate)
	if err != nil || missing != nil {
		t.Fatal("invented missing evidence")
	}
}
func TestSelectedEvidenceFailsClosed(t *testing.T) {
	for _, kind := range []string{"no-capability", "revoked", "initial", "final", "identity", "cancel", "source", "unsupported", "occurrence", "index", "payload"} {
		t.Run(kind, func(t *testing.T) {
			r, s, b := selectedFixture(t)
			eb := &evidenceTestBoundary{b, "fixture-native-events-epoch"}
			r.boundary = eb
			_, _, _, e := wakeFixture(t)
			switch kind {
			case "no-capability":
				r.boundary = b
			case "revoked":
				b.active = false
			case "source":
				eb.source = ""
			case "unsupported":
				e.EventType = "other"
			case "occurrence":
				e.CreatedAt = e.CreatedAt.Add(time.Second)
			case "index":
				e.SourceScenario = "other"
			case "payload":
				e.Payload = []byte("malformed")
			}
			if _, err := s.Insert(context.Background(), e); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			checks := 0
			raw, err := r.ReadSelectedEvidence(ctx, e.EventID, func(context.Context) error {
				checks++
				if kind == "initial" || kind == "final" && checks == 2 {
					return errors.New("revoked")
				}
				return nil
			}, func(context.Context, string, string, string) error {
				if kind == "identity" {
					return errors.New("denied")
				}
				if kind == "cancel" {
					cancel()
				}
				return nil
			})
			if err == nil || raw != nil {
				t.Fatalf("released %s: %s", kind, raw)
			}
		})
	}
}

func TestSelectedEvidenceCanonicalPayloadValidation(t *testing.T) {
	for _, kind := range []string{"missing-occurrence", "unsupported-type", "bad-category", "bad-run", "missing-data", "failed"} {
		t.Run(kind, func(t *testing.T) {
			r, s, b := selectedFixture(t)
			r.boundary = &evidenceTestBoundary{b, "fixture-native-events-epoch"}
			_, _, _, e := wakeFixture(t)
			var env domain.EventEnvelope
			if err := proto.Unmarshal(e.Payload, &env); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "missing-occurrence":
				env.OccurredAt = nil
			case "unsupported-type":
				env.EventType = "other"
				e.EventType = "other"
			case "missing-data":
				env.Data = nil
			default:
				category, run := "run-completed", "11111111-1111-4111-8111-111111111111"
				if kind == "bad-category" {
					category = "approved"
				}
				if kind == "bad-run" {
					run = "00000000-0000-0000-0000-000000000000"
				}
				if kind == "failed" {
					category = "run-failed"
				}
				facts, _ := structpb.NewStruct(map[string]any{"category": category, "correlation": map[string]any{"run_id": run}})
				env.Data, _ = anypb.New(facts)
			}
			e.Payload, _ = proto.Marshal(&env)
			if _, err := s.Insert(context.Background(), e); err != nil {
				t.Fatal(err)
			}
			raw, err := r.ReadSelectedEvidence(context.Background(), e.EventID, allow, validate)
			if kind == "failed" {
				if err != nil || !strings.Contains(string(raw), `"category":"run-failed"`) {
					t.Fatalf("failed event rejected: %s %v", raw, err)
				}
				return
			}
			if err == nil || raw != nil {
				t.Fatalf("released invalid %s: %s", kind, raw)
			}
		})
	}
}

// Deliberately broken future composition: swallowing errors must not release bytes.
type swallowingEvidenceBoundary struct {
	*testBoundary
	repeat bool
}

func (b *swallowingEvidenceBoundary) WithinEvidence(ctx context.Context, fn func(context.Context, string) error) error {
	return b.Within(ctx, func(inner context.Context) error {
		_ = fn(inner, "fixture-native-events-epoch")
		if b.repeat {
			_ = fn(inner, "fixture-native-events-epoch")
		}
		return nil
	})
}
func TestSelectedEvidenceRejectsSwallowedBoundaryErrors(t *testing.T) {
	for _, kind := range []string{"final-denial", "multiple-callbacks", "missing-final-denial", "missing-multiple-callbacks"} {
		t.Run(kind, func(t *testing.T) {
			r, s, b := selectedFixture(t)
			repeat := strings.Contains(kind, "multiple")
			r.boundary = &swallowingEvidenceBoundary{b, repeat}
			_, _, _, e := wakeFixture(t)
			if _, err := s.Insert(context.Background(), e); err != nil {
				t.Fatal(err)
			}
			id := e.EventID
			if strings.HasPrefix(kind, "missing") {
				id = "missing"
			}
			checks := 0
			raw, err := r.ReadSelectedEvidence(context.Background(), id, func(context.Context) error {
				checks++
				if !repeat && checks == 2 {
					return errors.New("final release denied")
				}
				return nil
			}, validate)
			if err == nil || raw != nil {
				t.Fatalf("released after swallowed %s: %s", kind, raw)
			}
		})
	}
}
