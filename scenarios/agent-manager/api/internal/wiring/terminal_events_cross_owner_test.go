package wiring

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/vrooli/api-core/eventbus"
)

// terminalOwnerProcessTransport is fixture-only. It executes a compiled Events
// owner test over bounded HTTP bytes on local files, never a listener or socket.
// Each attempt opens and closes the actual Events SQLite store and handler.
type terminalOwnerProcessTransport struct {
	binary          string
	dir             string
	mode            string
	loseResponse    bool
	requests        [][]byte
	acknowledgments []eventbus.DomainEventAcknowledgment
}

func (f *terminalOwnerProcessTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var encoded bytes.Buffer
	if err := r.Write(&encoded); err != nil {
		return nil, err
	}
	f.requests = append(f.requests, append([]byte(nil), encoded.Bytes()...))
	if err := os.WriteFile(filepath.Join(f.dir, "request.http"), encoded.Bytes(), 0600); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(r.Context(), f.binary, "-test.run=^TestTerminalAdmissionTransportOwnerFixture$")
	cmd.Env = []string{"VROOLI_TERMINAL_ADMISSION_FIXTURE_DIR=" + f.dir, "VROOLI_TERMINAL_ADMISSION_FIXTURE_MODE=" + f.mode}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("isolated Events owner process: %w: %s", err, output)
	}
	body, err := os.ReadFile(filepath.Join(f.dir, "response.http"))
	if err != nil {
		return nil, err
	}
	response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(body)), r)
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusOK {
		raw, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			return nil, err
		}
		var ack eventbus.DomainEventAcknowledgment
		if err = json.Unmarshal(raw, &ack); err != nil {
			return nil, err
		}
		f.acknowledgments = append(f.acknowledgments, ack)
		response.Body = io.NopCloser(bytes.NewReader(raw))
		if f.loseResponse {
			f.loseResponse = false
			response.Body.Close()
			return nil, errors.New("synthetic response loss after actual Events commit")
		}
	}
	return response, nil
}
func terminalOwnerCounts(t *testing.T, dir string, want int) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(dir, "counts.json"))
	if err != nil {
		t.Fatal(err)
	}
	var counts map[string]int
	if err = json.Unmarshal(body, &counts); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"events", "admissions", "deliveries"} {
		if counts[key] != want {
			t.Fatalf("%s=%d want %d", key, counts[key], want)
		}
	}
}
func TestTerminalCompositionActualEventsOwner(t *testing.T) {
	binary := os.Getenv("VROOLI_TERMINAL_ADMISSION_FIXTURE_BINARY")
	if binary == "" {
		t.Skip("explicit compiled Events owner fixture required")
	}
	if !filepath.IsAbs(binary) {
		t.Fatal("fixture binary must be absolute")
	}
	dir := t.TempDir()
	if parent := os.Getenv("VROOLI_TERMINAL_COMPOSITION_EVIDENCE_DIR"); parent != "" {
		if !filepath.IsAbs(parent) {
			t.Fatal("evidence parent must be absolute")
		}
		var err error
		dir, err = os.MkdirTemp(parent, "cross-owner-")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Retained isolated owner evidence: %s", dir)
	}
	path := filepath.Join(dir, "native.sqlite")
	db := terminalDB(t, path)
	transport := &terminalOwnerProcessTransport{binary: binary, dir: dir}
	client, err := eventbus.NewTerminalAdmissionClient("http://127.0.0.1:1", transport, 14*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	cfg := terminalConfig(client.AcceptTerminalEventIntent)
	repos, worker := terminalCompose(t, db, cfg)
	run := terminalRun(t, repos, domain.RunStatusUnknown)
	var seed, id string
	if err = db.QueryRowContext(t.Context(), "SELECT event_intent_json,event_intent_id FROM run_terminal_receipts WHERE run_id=?", run.ID).Scan(&seed, &id); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"denied", "source-mismatch"} {
		transport.mode = mode
		if ack, e := client.AcceptTerminalEventIntent(t.Context(), seed, id); e == nil || ack != (eventbus.DomainEventAcknowledgment{}) {
			t.Fatalf("%s accepted: %+v %v", mode, ack, e)
		}
		terminalOwnerCounts(t, dir, 0)
	}
	transport.mode = ""
	transport.loseResponse = true
	if err = worker.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	waitTerminal(t, worker, func(s TerminalEventsStatus) bool { return s.Attempts == 1 })
	worker.Stop()
	terminalOwnerCounts(t, dir, 1)
	if worker.Status().Acknowledged != 0 || terminalCount(t, db, "run_terminal_event_acknowledgments") != 0 {
		t.Fatal("uncertain transport retained native acknowledgment")
	}
	originalAck := transport.acknowledgments[0]
	if err = eventbus.ValidateDomainEventAcknowledgment(seed, id, originalAck); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	// Fresh owner graph and reopened native DB resume the immutable intent.
	db = terminalDB(t, path)
	_, worker = terminalCompose(t, db, cfg)
	if err = worker.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	waitTerminal(t, worker, func(s TerminalEventsStatus) bool { return s.Attempts == 1 })
	worker.Stop()
	terminalOwnerCounts(t, dir, 1)
	if len(transport.acknowledgments) != 2 || transport.acknowledgments[1] != originalAck || terminalCount(t, db, "run_terminal_event_acknowledgments") != 1 {
		t.Fatal("replay changed acceptance or failed native retention")
	}
	if !bytes.Equal(transport.requests[2], transport.requests[3]) {
		t.Fatal("retry changed exact HTTP request")
	}
	var nativeAck string
	if err = db.QueryRowContext(t.Context(), "SELECT acknowledgment_json FROM run_terminal_event_acknowledgments WHERE event_intent_id=?", id).Scan(&nativeAck); err != nil {
		t.Fatal(err)
	}
	var decoded eventbus.DomainEventAcknowledgment
	if json.Unmarshal([]byte(nativeAck), &decoded) != nil || decoded != originalAck {
		t.Fatal("native acknowledgment differs from actual Events owner")
	}
	// Third restart must not contact Events, even if current receiver authority is
	// now denied. Already-retained ack is durable evidence, not permission to act.
	transport.mode = "denied"
	_, worker = terminalCompose(t, db, cfg)
	if err = worker.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	waitTerminal(t, worker, func(s TerminalEventsStatus) bool { return s.Attempts == 1 })
	worker.Stop()
	if len(transport.requests) != 4 {
		t.Fatal("acknowledged receipt redispatched")
	}
	var retainedSeed string
	if err = db.QueryRowContext(t.Context(), "SELECT event_intent_json FROM run_terminal_receipts WHERE run_id=?", run.ID).Scan(&retainedSeed); err != nil {
		t.Fatal(err)
	}
	if retainedSeed != seed || !strings.Contains(seed, `"status":"unknown"`) {
		t.Fatal("original unknown observation changed")
	}
	report, _ := json.MarshalIndent(map[string]any{"schema": "terminal-owner-composition-fixture.v1", "ack": originalAck, "request_attempts": len(transport.requests), "authority_denials": 2, "response_loss_after_commit": true, "exact_retry": true, "native_restart_replay": true, "network_requests": 0, "services_started": 0}, "", "  ")
	if err = os.WriteFile(filepath.Join(dir, "qualification.json"), append(report, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
