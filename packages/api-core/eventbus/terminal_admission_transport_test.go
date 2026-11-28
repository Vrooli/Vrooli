package eventbus

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type terminalRoundTripFunc func(*http.Request) (*http.Response, error)

func (f terminalRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func terminalTransportSeed(t *testing.T) (string, string, DomainEventAcknowledgment) {
	t.Helper()
	event := DomainEvent{Source: "agent-manager", EventType: "agent_manager.run.terminal.v1", Occurred: time.Date(2026, 11, 7, 10, 0, 0, 123, time.UTC), Payload: map[string]any{"source_instance": "test-installation", "source_epoch": "epoch-1", "run_id": "11111111-1111-4111-8111-111111111111", "lifecycle_version": "1", "status": "complete", "terminal_observation_sha256": strings.Repeat("b", 64)}}
	seed, _ := json.Marshal(event)
	id := digestBytes(seed)
	_, canonical, err := RetainedDomainEventEnvelope(string(seed), id)
	if err != nil {
		t.Fatal(err)
	}
	return string(seed), id, DomainEventAcknowledgment{Schema: DomainEventAcknowledgmentSchema, EventID: id, CanonicalSHA256: digestBytes(canonical), StoreID: 17}
}
func terminalHTTPResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestTerminalAdmissionClientPreservesSeedAndHasNoAmbientState(t *testing.T) {
	seed, id, ack := terminalTransportSeed(t)
	encoded, _ := json.Marshal(ack)
	calls := 0
	t.Setenv("HTTPS_PROXY", "http://forbidden.invalid")
	t.Setenv("VROOLI_AGENT_IDENTITY_TOKEN", "must-not-be-read")
	client, err := NewTerminalAdmissionClient("https://events.invalid", terminalRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		body, _ := io.ReadAll(r.Body)
		request, err := DecodeTerminalAdmissionRequest(body)
		if err != nil || request.IntentJSON != seed || request.EventID != id || request.Schema != "vrooli-events.terminal-admission.v1" {
			t.Fatal("retained request changed", err)
		}
		if r.URL.String() != "https://events.invalid/api/v1/internal/terminal-admissions" || r.Method != "POST" || len(r.Header) != 2 || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Fatal("ambient or redirected request", r.URL, r.Header)
		}
		if _, ok := r.Context().Deadline(); !ok {
			t.Fatal("unbounded request")
		}
		return terminalHTTPResponse(200, string(encoded)), nil
	}), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.AcceptTerminalEventIntent(context.Background(), seed, id)
	if err != nil || got != ack || calls != 1 {
		t.Fatalf("ack=%+v calls=%d err=%v", got, calls, err)
	}
	if client.client.Jar != nil {
		t.Fatal("cookie jar enabled")
	}
}
func TestTerminalAdmissionClientRejectsUnacknowledgedResponses(t *testing.T) {
	seed, id, ack := terminalTransportSeed(t)
	encoded, _ := json.Marshal(ack)
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"empty200", 200, ""}, {"generic200", 200, `{"ok":true}`}, {"accepted202", 202, string(encoded)}, {"no-content204", 204, ""}, {"generic409", 409, string(encoded)}, {"redirect", 307, string(encoded)},
		{"wrong-id", 200, strings.Replace(string(encoded), id, strings.Repeat("c", 64), 1)}, {"wrong-digest", 200, strings.Replace(string(encoded), ack.CanonicalSHA256, strings.Repeat("d", 64), 1)},
		{"missing-store", 200, strings.Replace(string(encoded), `"store_id":17`, `"store_id":0`, 1)}, {"duplicate", 200, strings.Replace(string(encoded), `"store_id":17`, `"store_id":2,"store_id":17`, 1)},
		{"unknown", 200, strings.TrimSuffix(string(encoded), "}") + `,"extra":true}`}, {"suffix", 200, string(encoded) + "{}"}, {"oversized", 200, strings.Repeat("x", 1025)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client, err := NewTerminalAdmissionClient("http://127.0.0.1:1", terminalRoundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				resp := terminalHTTPResponse(tc.status, tc.body)
				resp.Header.Set("Location", "https://elsewhere.invalid")
				return resp, nil
			}), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.AcceptTerminalEventIntent(context.Background(), seed, id)
			if err == nil || got != (DomainEventAcknowledgment{}) || calls != 1 {
				t.Fatalf("ack=%+v calls=%d err=%v", got, calls, err)
			}
		})
	}
}
func TestTerminalAdmissionClientTimeoutCancellationAndInvalidSeed(t *testing.T) {
	seed, id, _ := terminalTransportSeed(t)
	calls := 0
	client, err := NewTerminalAdmissionClient("http://127.0.0.1:1", terminalRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		<-r.Context().Done()
		return nil, r.Context().Err()
	}), 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.AcceptTerminalEventIntent(context.Background(), seed, id); err == nil || calls != 1 {
		t.Fatal("timeout was success")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.AcceptTerminalEventIntent(ctx, seed, id); !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal("cancelled call transmitted")
	}
	if _, err := client.AcceptTerminalEventIntent(context.Background(), seed+" ", id); err == nil || calls != 1 {
		t.Fatal("invalid seed transmitted")
	}
}
func TestTerminalAdmissionClientRequiresExplicitBoundedTransport(t *testing.T) {
	unused := terminalRoundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("constructor performed I/O"); return nil, nil })
	for _, origin := range []string{"http://events.invalid", "http://192.0.2.1", "http://[2001:db8::1]", "http://localhost.attacker.invalid", "", "file:///tmp/events", "http://user:pass@events.invalid", "http://events.invalid/", "http://events.invalid?", "http://events.invalid?a=b", "http://events.invalid#x", "http://events.invalid/api"} {
		if _, err := NewTerminalAdmissionClient(origin, unused, time.Second); err == nil {
			t.Fatal("accepted origin", origin)
		}
	}
	for _, timeout := range []time.Duration{0, -1, 31 * time.Second} {
		if _, err := NewTerminalAdmissionClient("http://127.0.0.1:1", unused, timeout); err == nil {
			t.Fatal("accepted timeout", timeout)
		}
	}
	var typedNil terminalRoundTripFunc
	for _, transport := range []http.RoundTripper{nil, typedNil, http.DefaultTransport, &http.Transport{Proxy: http.ProxyURL(&url.URL{Scheme: "http", Host: "proxy.invalid"})}, &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}} {
		if _, err := NewTerminalAdmissionClient("https://events.invalid", transport, time.Second); err == nil {
			t.Fatalf("accepted missing/proxy/insecure transport %T", transport)
		}
	}
	for _, origin := range []string{"http://localhost", "http://127.0.0.1:1", "http://[::1]:1", "https://events.invalid"} {
		if _, err := NewTerminalAdmissionClient(origin, unused, time.Second); err != nil {
			t.Fatalf("valid origin %s: %v", origin, err)
		}
	}
	explicit := &http.Transport{}
	client, err := NewTerminalAdmissionClient("http://127.0.0.1:1", explicit, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if client.client.Transport == explicit {
		t.Fatal("mutable transport not frozen")
	}
}
func TestTerminalAdmissionProtocolRejectsAmbiguousRequest(t *testing.T) {
	seed, id, _ := terminalTransportSeed(t)
	body, _ := json.Marshal(TerminalAdmissionRequest{TerminalAdmissionSchema, id, seed})
	good := string(body)
	for _, bad := range []string{"", good + " ", good + "{}", strings.Replace(good, `"schema":`, `"extra":true,"schema":`, 1), strings.Replace(good, `"event_id":`, `"event_id":"wrong","event_id":`, 1), strings.Repeat("x", TerminalAdmissionMaxRequestBytes+1)} {
		if _, err := DecodeTerminalAdmissionRequest([]byte(bad)); err == nil {
			t.Fatal("ambiguous request accepted")
		}
	}
}
