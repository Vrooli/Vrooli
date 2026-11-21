package bridge

import (
	"context"
	"encoding/json"
	"errors"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/convert"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"strings"
	"sync"
	"testing"
	"time"
)

type testBoundary struct {
	mu     sync.Mutex
	active bool
	held   bool
}

func (b *testBoundary) Within(ctx context.Context, fn func(context.Context) error) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.active {
		return ErrSelectedRead
	}
	b.held = true
	defer func() { b.held = false }()
	return fn(ctx)
}
func selectedFixture(t *testing.T) (*SelectedReader, *store.SQLiteStore, *testBoundary) {
	t.Helper()
	s, err := store.NewSQLiteStore(context.Background(), store.SQLiteConfig{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	b := &testBoundary{active: true}
	r, err := NewSelectedReader(s, b)
	if err != nil {
		t.Fatal(err)
	}
	return r, s, b
}
func selectedEvent(t *testing.T) store.Event {
	t.Helper()
	e, err := convert.EnvelopeToEvent(&domain.EventEnvelope{EventId: "event-1", EventType: "agent_manager.run.notification.v1", Source: &domain.EventSource{Scenario: "agent-manager"}})
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func allow(context.Context) error                            { return nil }
func validate(context.Context, string, string, string) error { return nil }
func TestSelectedReaderCanonicalExactAndMissing(t *testing.T) {
	r, s, b := selectedFixture(t)
	e := selectedEvent(t)
	if _, err := s.Insert(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	checks, validations := 0, 0
	recheck := func(context.Context) error {
		if !b.held {
			t.Fatal("authority boundary not held")
		}
		checks++
		return nil
	}
	raw, err := r.ReadSelected(context.Background(), "event-1", recheck, func(_ context.Context, id, typ, source string) error {
		validations++
		if !b.held || id != e.EventID || typ != e.EventType || source != e.SourceScenario {
			t.Fatal("wrong canonical identity")
		}
		return nil
	})
	if err != nil || !json.Valid(raw) || checks != 2 || validations != 1 {
		t.Fatalf("read=%s %v checks=%d validations=%d", raw, err, checks, validations)
	}
	raw, err = r.ReadSelected(context.Background(), "missing", recheck, validate)
	if err != nil || raw != nil {
		t.Fatalf("missing=%s %v", raw, err)
	}
}
func TestSelectedReaderRejectsCanonicalMismatchAndMalformed(t *testing.T) {
	for _, kind := range []string{"id", "type", "source", "protobuf", "empty", "oversize"} {
		t.Run(kind, func(t *testing.T) {
			r, s, _ := selectedFixture(t)
			e := selectedEvent(t)
			switch kind {
			case "id":
				e.EventID = "another"
			case "type":
				e.EventType = "different"
			case "source":
				e.SourceScenario = "different"
			case "protobuf":
				e.Payload = []byte("bad")
			case "empty":
				e.Payload = nil
			case "oversize":
				e.Payload = []byte(strings.Repeat("x", (1<<20)+1))
			}
			if _, err := s.Insert(context.Background(), e); err != nil {
				t.Fatal(err)
			}
			raw, err := r.ReadSelected(context.Background(), e.EventID, allow, validate)
			if err == nil || raw != nil {
				t.Fatalf("released invalid row: %d %v", len(raw), err)
			}
		})
	}
}
func TestSelectedReaderRejectsAuthorityAtEveryBoundary(t *testing.T) {
	for _, at := range []string{"initial", "identity", "final", "boundary", "cancel"} {
		t.Run(at, func(t *testing.T) {
			r, s, b := selectedFixture(t)
			if _, err := s.Insert(context.Background(), selectedEvent(t)); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			checks := 0
			check := func(context.Context) error {
				checks++
				if at == "initial" || at == "final" && checks == 2 {
					return errors.New("revoked")
				}
				return nil
			}
			valid := func(context.Context, string, string, string) error {
				if at == "identity" {
					return errors.New("wrong selection")
				}
				if at == "cancel" {
					cancel()
				}
				return nil
			}
			if at == "boundary" {
				b.active = false
			}
			raw, err := r.ReadSelected(ctx, "event-1", check, valid)
			if err == nil || raw != nil {
				t.Fatalf("released after %s: %s %v", at, raw, err)
			}
		})
	}
}
func TestSelectedReaderSerializesRevocationWithRead(t *testing.T) {
	r, s, b := selectedFixture(t)
	if _, err := s.Insert(context.Background(), selectedEvent(t)); err != nil {
		t.Fatal(err)
	}
	entered, release, invalidated := make(chan struct{}), make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := r.ReadSelected(context.Background(), "event-1", allow, func(context.Context, string, string, string) error { close(entered); <-release; return nil })
		done <- err
	}()
	select {
	case <-entered:
	case err := <-done:
		t.Fatalf("read did not reach validation: %v", err)
	case <-time.After(time.Second):
		t.Fatal("read did not enter")
	}
	attempted := make(chan struct{})
	go func() { close(attempted); b.mu.Lock(); b.active = false; b.mu.Unlock(); close(invalidated) }()
	<-attempted
	select {
	case <-invalidated:
		t.Fatal("revocation crossed read boundary")
	case <-time.After(10 * time.Millisecond):
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	<-invalidated
	if raw, err := r.ReadSelected(context.Background(), "event-1", allow, validate); err == nil || raw != nil {
		t.Fatal("read survived revocation")
	}
}
func TestSelectedReaderInvalidArgumentsAndDeadline(t *testing.T) {
	r, _, _ := selectedFixture(t)
	if _, err := NewSelectedReader(nil, nil); err == nil {
		t.Fatal("accepted nil owners")
	}
	for _, id := range []string{"", "*", "event/1", strings.Repeat("x", 129)} {
		if _, err := r.ReadSelected(context.Background(), id, allow, validate); err == nil {
			t.Fatalf("accepted %q", id)
		}
	}
	if _, err := r.ReadSelected(nil, "event-1", allow, validate); err == nil {
		t.Fatal("nil context")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.ReadSelected(ctx, "event-1", allow, validate); err == nil {
		t.Fatal("cancelled context")
	}
	if _, err := r.ReadSelected(context.Background(), "missing", func(ctx context.Context) error {
		d, ok := ctx.Deadline()
		if !ok || time.Until(d) > 8*time.Second {
			t.Fatal("unbounded owner context")
		}
		return nil
	}, validate); err != nil {
		t.Fatal(err)
	}
}
