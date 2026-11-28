package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/convert"
)

type terminalAdmissionBoundaryFunc func(context.Context, *http.Request, func(context.Context, TerminalAdmissionScope) error) error

func (f terminalAdmissionBoundaryFunc) WithTerminalAdmission(ctx context.Context, r *http.Request, fn func(context.Context, TerminalAdmissionScope) error) error {
	return f(ctx, r, fn)
}

type terminalHandlerTransport func(*http.Request) (*http.Response, error)

func (f terminalHandlerTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func terminalHandlerBoundary(source string) TerminalAdmissionBoundary {
	return terminalAdmissionBoundaryFunc(func(ctx context.Context, _ *http.Request, fn func(context.Context, TerminalAdmissionScope) error) error {
		return fn(ctx, TerminalAdmissionScope{source, "epoch-1"})
	})
}
func terminalHandlerRequest(t *testing.T, seed, id string) *http.Request {
	t.Helper()
	body, err := json.Marshal(eventbus.TerminalAdmissionRequest{Schema: eventbus.TerminalAdmissionSchema, EventID: id, IntentJSON: seed})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, eventbus.TerminalAdmissionPath, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestTerminalAdmissionRouteAbsentByDefault(t *testing.T) {
	s := &Server{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/internal/terminal-admissions", strings.NewReader(`{}`))
	s.routes().ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("default route status=%d", w.Code)
	}
}

func TestTerminalAdmissionHandlerActualClientLostAckAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.sqlite")
	s := terminalAdmissionFixture(t, path)
	terminalAdmissionSubscription(t, s, "fixture")
	seed, id := terminalAdmissionIntent(t, nil)
	calls := 0
	var first []byte
	client, err := eventbus.NewTerminalAdmissionClient("http://127.0.0.1:1", terminalHandlerTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		s.terminalAdmissionBoundary = terminalHandlerBoundary("fixture-source")
		w := httptest.NewRecorder()
		s.routes().ServeHTTP(w, r)
		if calls == 1 {
			first = append([]byte(nil), w.Body.Bytes()...)
			if w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
			if err := s.store.Close(); err != nil {
				t.Fatal(err)
			}
			s = terminalAdmissionFixture(t, path)
			return nil, errors.New("lost response after commit")
		}
		if !bytes.Equal(first, w.Body.Bytes()) {
			t.Fatal("replay changed acknowledgment")
		}
		return w.Result(), nil
	}), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.AcceptTerminalEventIntent(context.Background(), seed, id); err == nil {
		t.Fatal("lost ack was success")
	}
	terminalAdmissionCounts(t, s, 1, 1, 1)
	ack, err := client.AcceptTerminalEventIntent(context.Background(), seed, id)
	if err != nil || eventbus.ValidateDomainEventAcknowledgment(seed, id, ack) != nil {
		t.Fatal("invalid replay", err)
	}
	terminalAdmissionCounts(t, s, 1, 1, 1)
}

func TestTerminalAdmissionHandlerBoundaryRejectsWithoutWrites(t *testing.T) {
	for _, mode := range []string{"denied", "no-callback", "source-mismatch", "epoch-mismatch", "nil-context", "cancelled-detached", "late-callback"} {
		t.Run(mode, func(t *testing.T) {
			s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
			seed, id := terminalAdmissionIntent(t, nil)
			r := terminalHandlerRequest(t, seed, id)
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()
			r = r.WithContext(ctx)
			var late func(context.Context, TerminalAdmissionScope) error
			s.terminalAdmissionBoundary = terminalAdmissionBoundaryFunc(func(_ context.Context, _ *http.Request, fn func(context.Context, TerminalAdmissionScope) error) error {
				scope := TerminalAdmissionScope{"fixture-source", "epoch-1"}
				switch mode {
				case "denied":
					return errors.New("revoked")
				case "no-callback":
					return nil
				case "source-mismatch":
					scope.SourceInstance = "other"
				case "epoch-mismatch":
					scope.SourceEpoch = "epoch-2"
				case "nil-context":
					return fn(nil, scope)
				case "cancelled-detached":
					cancel()
				case "late-callback":
					late = fn
					return nil
				}
				return fn(context.Background(), scope)
			})
			w := httptest.NewRecorder()
			s.routes().ServeHTTP(w, r)
			if w.Code != http.StatusForbidden {
				t.Fatalf("status=%d", w.Code)
			}
			if late != nil && late(context.Background(), TerminalAdmissionScope{"fixture-source", "epoch-1"}) == nil {
				t.Fatal("late callback accepted")
			}
			terminalAdmissionCounts(t, s, 0, 0, 0)
		})
	}
}
func TestTerminalAdmissionHandlerBoundaryFailureAfterCommitIsNotAcknowledged(t *testing.T) {
	for _, mode := range []string{"repeated-callback", "error-after-callback", "cancel-after-callback"} {
		t.Run(mode, func(t *testing.T) {
			s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
			seed, id := terminalAdmissionIntent(t, nil)
			s.terminalAdmissionBoundary = terminalAdmissionBoundaryFunc(func(ctx context.Context, _ *http.Request, fn func(context.Context, TerminalAdmissionScope) error) error {
				scope := TerminalAdmissionScope{"fixture-source", "epoch-1"}
				inner, cancel := context.WithCancel(ctx)
				defer cancel()
				if err := fn(inner, scope); err != nil {
					return err
				}
				if mode == "cancel-after-callback" {
					cancel()
					return nil
				}
				if mode == "error-after-callback" {
					return errors.New("authority boundary did not complete")
				}
				_ = fn(ctx, scope)
				return nil
			})
			w := httptest.NewRecorder()
			s.routes().ServeHTTP(w, terminalHandlerRequest(t, seed, id))
			if w.Code != 403 {
				t.Fatal("failed boundary acknowledged", w.Code)
			}
			terminalAdmissionCounts(t, s, 1, 0, 1)
			s.terminalAdmissionBoundary = terminalHandlerBoundary("fixture-source")
			w = httptest.NewRecorder()
			s.routes().ServeHTTP(w, terminalHandlerRequest(t, seed, id))
			if w.Code != 200 {
				t.Fatal("authorized retry could not recover committed acceptance")
			}
			terminalAdmissionCounts(t, s, 1, 0, 1)
		})
	}
}

func TestTerminalAdmissionHandlerRevocationSerializesWithAcceptance(t *testing.T) {
	s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
	seed, id := terminalAdmissionIntent(t, nil)
	var mu sync.Mutex
	revoked := false
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	s.terminalAdmissionBoundary = terminalAdmissionBoundaryFunc(func(ctx context.Context, _ *http.Request, fn func(context.Context, TerminalAdmissionScope) error) error {
		mu.Lock()
		defer mu.Unlock()
		if revoked {
			return errors.New("revoked")
		}
		close(entered)
		<-release
		return fn(ctx, TerminalAdmissionScope{"fixture-source", "epoch-1"})
	})
	w := httptest.NewRecorder()
	go func() { defer close(done); s.routes().ServeHTTP(w, terminalHandlerRequest(t, seed, id)) }()
	<-entered
	revokedDone := make(chan struct{})
	go func() { mu.Lock(); revoked = true; mu.Unlock(); close(revokedDone) }()
	close(release)
	<-done
	<-revokedDone
	if w.Code != 200 {
		t.Fatal("authorized in-flight admission failed", w.Code)
	}
	w = httptest.NewRecorder()
	s.routes().ServeHTTP(w, terminalHandlerRequest(t, seed, id))
	if w.Code != 403 {
		t.Fatal("revoked replay acknowledged")
	}
	terminalAdmissionCounts(t, s, 1, 0, 1)
}
func TestTerminalAdmissionHandlerRejectsMalformedAndGenericDuplicate(t *testing.T) {
	for _, mode := range []string{"unknown-field", "wrong-schema", "bad-method", "query", "encoding", "oversized", "generic-duplicate"} {
		t.Run(mode, func(t *testing.T) {
			s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
			s.terminalAdmissionBoundary = terminalHandlerBoundary("fixture-source")
			seed, id := terminalAdmissionIntent(t, nil)
			r := terminalHandlerRequest(t, seed, id)
			switch mode {
			case "unknown-field":
				r.Body = io.NopCloser(strings.NewReader(`{"extra":true}`))
			case "wrong-schema":
				body, _ := io.ReadAll(r.Body)
				r.Body = io.NopCloser(strings.NewReader(strings.Replace(string(body), eventbus.TerminalAdmissionSchema, "wrong", 1)))
			case "bad-method":
				r.Method = "PUT"
			case "query":
				r.URL.RawQuery = "source_epoch=epoch-1"
			case "encoding":
				r.Header.Set("Content-Encoding", "gzip")
			case "oversized":
				r.Body = io.NopCloser(strings.NewReader(strings.Repeat("x", eventbus.TerminalAdmissionMaxRequestBytes+1)))
			case "generic-duplicate":
				env, canonical, _ := eventbus.RetainedDomainEventEnvelope(seed, id)
				event, err := convert.EnvelopeToEvent(env)
				event.Payload = canonical
				if err != nil {
					t.Fatal(err)
				}
				if _, err := s.admitEvent(context.Background(), event); err != nil {
					t.Fatal(err)
				}
			}
			w := httptest.NewRecorder()
			s.handleTerminalAdmission(w, r)
			if w.Code < 400 {
				t.Fatal("invalid request acknowledged", mode, w.Code)
			}
			want := 0
			if mode == "generic-duplicate" {
				want = 1
				if w.Code != 409 {
					t.Fatal(w.Code)
				}
			}
			terminalAdmissionCounts(t, s, want, 0, 0)
		})
	}
}

// TestTerminalAdmissionTransportOwnerFixture is subprocess test machinery, not
// a service. Every invocation reopens the real SQLite owner and serves exactly
// one HTTP request in memory. Files contain synthetic test facts only. There is
// no listener, discovery, credential, live storage, signing, or queue delivery.
func TestTerminalAdmissionTransportOwnerFixture(t *testing.T) {
	dir := os.Getenv("VROOLI_TERMINAL_ADMISSION_FIXTURE_DIR")
	if dir == "" {
		t.Skip("explicit isolated cross-owner fixture directory required")
	}
	if !filepath.IsAbs(dir) {
		t.Fatal("fixture directory must be absolute")
	}
	encoded, err := os.ReadFile(filepath.Join(dir, "request.http"))
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) > eventbus.TerminalAdmissionMaxRequestBytes+8192 {
		t.Fatal("fixture request oversized")
	}
	r, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(encoded)))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	s := terminalAdmissionFixture(t, filepath.Join(dir, "events.sqlite"))
	s.terminalAdmissionBoundary = terminalHandlerBoundary("test-installation")
	switch os.Getenv("VROOLI_TERMINAL_ADMISSION_FIXTURE_MODE") {
	case "":
	case "denied":
		s.terminalAdmissionBoundary = terminalAdmissionBoundaryFunc(func(context.Context, *http.Request, func(context.Context, TerminalAdmissionScope) error) error {
			return errors.New("fixture authority revoked")
		})
	case "source-mismatch":
		s.terminalAdmissionBoundary = terminalHandlerBoundary("other-installation")
	default:
		t.Fatal("unknown synthetic fixture mode")
	}
	var count int
	if err := s.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM subscriptions").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		terminalAdmissionSubscription(t, s, "transport-fixture")
	}
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, r)
	response := w.Result()
	defer response.Body.Close()
	var output bytes.Buffer
	if err := response.Write(&output); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "response.http"), output.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for key, table := range map[string]string{"events": "events", "admissions": "terminal_event_admissions", "deliveries": "subscription_deliveries"} {
		var count int
		if err := s.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		counts[key] = count
	}
	body, _ := json.Marshal(counts)
	if err := os.WriteFile(filepath.Join(dir, "counts.json"), body, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalAdmissionHandlerDetachedBoundaryPreservesPinnedStorage(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(map[bool]string{false: "current-pin", true: "retired-pin"}[stale], func(t *testing.T) {
			dir := t.TempDir()
			primaryPath := filepath.Join(dir, "primary.sqlite")
			testPath := filepath.Join(dir, "test.sqlite")
			primary := terminalAdmissionFixture(t, primaryPath)
			isolated := terminalAdmissionFixture(t, testPath)
			routed, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: primaryPath, MaxOpenConns: 1, MaxIdleConns: 1})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = routed.Close() })
			if err := routed.InstallTestPool(context.Background(), testPath, "terminal-fixture", time.Minute); err != nil {
				t.Fatal(err)
			}
			pinned, err := routed.PinTestPool(context.Background(), "terminal-fixture")
			if err != nil {
				t.Fatal(err)
			}
			if stale {
				if err := routed.ClearTestPool("terminal-fixture"); err != nil {
					t.Fatal(err)
				}
			}
			primary.db = routed
			primary.terminalAdmissionBoundary = terminalAdmissionBoundaryFunc(func(_ context.Context, _ *http.Request, fn func(context.Context, TerminalAdmissionScope) error) error {
				return fn(context.Background(), TerminalAdmissionScope{"fixture-source", "epoch-1"})
			})
			seed, id := terminalAdmissionIntent(t, nil)
			request := terminalHandlerRequest(t, seed, id).WithContext(pinned)
			w := httptest.NewRecorder()
			primary.routes().ServeHTTP(w, request)
			if stale {
				if w.Code < 400 {
					t.Fatal("retired pin was acknowledged")
				}
				terminalAdmissionCounts(t, isolated, 0, 0, 0)
			} else {
				if w.Code != 200 {
					t.Fatal("pinned admission failed", w.Code, w.Body.String())
				}
				terminalAdmissionCounts(t, isolated, 1, 0, 1)
			}
			terminalAdmissionCounts(t, primary, 0, 0, 0)
		})
	}
}
