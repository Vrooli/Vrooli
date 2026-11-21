package database

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/runmodel"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/eventbus"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func evidenceSelection(t *testing.T, r *Repositories, run *domain.Run) TerminalEvidenceSelector {
	t.Helper()
	receipt := readReceipt(t, r, run)
	ack, err := r.Runs.(*runRepository).TerminalEventAcknowledgment(t.Context(), run.ID, run.LifecycleVersion)
	if err != nil {
		t.Fatal(err)
	}
	return TerminalEvidenceSelector{Schema: "agent-manager.run-terminal.v1", OwnerID: "agent-manager", SourceInstance: receiptTestScope.SourceInstance, SourceEpoch: receiptTestScope.SourceEpoch, RunID: run.ID.String(), LifecycleVersion: strconv.FormatInt(run.LifecycleVersion, 10), TerminalObservationSHA256: receipt.ObservationSHA256, TerminalEventID: ack.EventID, CanonicalEventSHA256: ack.CanonicalSHA256}
}
func allowTerminalEvidence(context.Context, TerminalEvidenceSelector) error { return nil }
func TestTerminalEvidenceExactMinimizedSelection(t *testing.T) {
	for _, status := range []domain.RunStatus{domain.RunStatusComplete, domain.RunStatusFailed, domain.RunStatusCancelled, domain.RunStatusUnknown} {
		t.Run(string(status), func(t *testing.T) {
			db, done := setupTestDB(t)
			defer done()
			r := receiptRepos(t, db)
			run := receiptRun(t, r)
			finishReceiptRun(t, r, run, status)
			if _, err := terminalOutbox(t, r, codecAck).Dispatch(t.Context(), 1); err != nil {
				t.Fatal(err)
			}
			s := evidenceSelection(t, r, run)
			calls := 0
			reader, err := NewTerminalEvidenceReader(r, func(_ context.Context, got TerminalEvidenceSelector) error {
				calls++
				if got != s {
					t.Fatal("authority selection differs")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			before := time.Now().UTC()
			got, err := reader.Select(t.Context(), s)
			if err != nil || calls != 2 || got.Schema != "agent-manager.selected-terminal-run-evidence.v1" || got.Status != string(status) || got.Lineage != nil || got.ObservedAt.Before(before) || got.Reference.Version != s.TerminalObservationSHA256 {
				t.Fatalf("selection %#v %v calls=%d", got, err, calls)
			}
			raw, _ := json.Marshal(got)
			for _, forbidden := range []string{"task_id", "owner_epoch", "transcript", "execution_mode", "captured_at", "label", "event_intent_json"} {
				if strings.Contains(string(raw), forbidden) {
					t.Fatalf("leaked %s", forbidden)
				}
			}
			if status != domain.RunStatusUnknown {
				if err := r.Runs.Delete(t.Context(), run.ID, run.Status, run.LifecycleVersion); err != nil {
					t.Fatal(err)
				}
			}
			again, err := reader.Select(t.Context(), s)
			if err != nil || again.Reference != got.Reference || again.ObservedAt.Before(got.ObservedAt) {
				t.Fatalf("immutable selection %#v %v", again, err)
			}
		})
	}
}
func TestTerminalEvidenceDisabled(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	if _, err := NewTerminalEvidenceReader(receiptRepos(t, db), nil); err == nil {
		t.Fatal("no authority accepted")
	}
	if _, err := NewTerminalEvidenceReader(NewRepositories(db, logrus.New()), allowTerminalEvidence); err == nil {
		t.Fatal("production reader enabled")
	}
}
func TestTerminalEvidenceScopeJoinAndRevocation(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	r := receiptRepos(t, db)
	run := receiptRun(t, r)
	finishReceiptRun(t, r, run, domain.RunStatusComplete)
	if _, err := terminalOutbox(t, r, codecAck).Dispatch(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	original := evidenceSelection(t, r, run)
	for _, mode := range []string{"source", "epoch", "owner", "schema", "run", "lifecycle", "noncanonical_lifecycle", "observation", "event", "canonical_digest", "denied_before", "revoked_release", "cancelled_release"} {
		t.Run(mode, func(t *testing.T) {
			s := original
			calls := 0
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			switch mode {
			case "source":
				s.SourceInstance = "other"
			case "epoch":
				s.SourceEpoch = "other"
			case "owner":
				s.OwnerID = "other"
			case "schema":
				s.Schema = "other"
			case "run":
				s.RunID = uuid.NewString()
			case "lifecycle":
				s.LifecycleVersion = "999"
			case "noncanonical_lifecycle":
				s.LifecycleVersion = "01"
			case "observation":
				s.TerminalObservationSHA256 = strings.Repeat("0", 64)
			case "event":
				s.TerminalEventID = strings.Repeat("0", 64)
			case "canonical_digest":
				s.CanonicalEventSHA256 = strings.Repeat("0", 64)
			}
			reader, err := NewTerminalEvidenceReader(r, func(context.Context, TerminalEvidenceSelector) error {
				calls++
				if mode == "denied_before" || mode == "revoked_release" && calls == 2 {
					return errors.New("authority denied")
				}
				if mode == "cancelled_release" && calls == 2 {
					cancel()
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			got, err := reader.Select(ctx, s)
			if err == nil || got != (SelectedTerminalEvidence{}) {
				t.Fatalf("released invalid evidence %#v %v", got, err)
			}
			if mode == "denied_before" && calls != 1 {
				t.Fatal("continued after denied authority")
			}
		})
	}
}
func TestTerminalEvidenceMissingReceiptOrAck(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	r := receiptRepos(t, db)
	run := receiptRun(t, r)
	finishReceiptRun(t, r, run, domain.RunStatusComplete)
	receipt := readReceipt(t, r, run)
	ack, _ := codecAck(t.Context(), receipt.EventIntentJSON, receipt.EventIntentID)
	s := TerminalEvidenceSelector{Schema: "agent-manager.run-terminal.v1", OwnerID: "agent-manager", SourceInstance: receiptTestScope.SourceInstance, SourceEpoch: receiptTestScope.SourceEpoch, RunID: run.ID.String(), LifecycleVersion: strconv.FormatInt(run.LifecycleVersion, 10), TerminalObservationSHA256: receipt.ObservationSHA256, TerminalEventID: receipt.EventIntentID, CanonicalEventSHA256: ack.CanonicalSHA256}
	reader, _ := NewTerminalEvidenceReader(r, allowTerminalEvidence)
	if got, err := reader.Select(t.Context(), s); err == nil || got != (SelectedTerminalEvidence{}) {
		t.Fatal("unacknowledged receipt qualified")
	}
	s.RunID = uuid.NewString()
	if got, err := reader.Select(t.Context(), s); err == nil || got != (SelectedTerminalEvidence{}) {
		t.Fatal("missing receipt qualified")
	}
}
func TestTerminalEvidenceCrossOwnerFixture(t *testing.T) {
	dir := os.Getenv("VROOLI_TERMINAL_OUTBOX_FIXTURE_DIR")
	if dir == "" {
		t.Skip("explicit cross-owner fixture path not requested")
	}
	db := outboxOpen(t, filepath.Join(dir, "agent-manager.db"))
	defer db.Close()
	r := receiptRepos(t, db)
	raw, err := os.ReadFile(filepath.Join(dir, "intent.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture terminalOwnerFixture
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	s := evidenceSelection(t, r, &domain.Run{ID: fixture.RunID, LifecycleVersion: fixture.LifecycleVersion})
	reader, err := NewTerminalEvidenceReader(r, func(_ context.Context, got TerminalEvidenceSelector) error {
		if got != s {
			return errors.New("fixture selector not authorized")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := reader.Select(t.Context(), s)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "selected-terminal-evidence.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestTerminalEvidenceCorruptSourceBinding(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	r := receiptRepos(t, db)
	run := receiptRun(t, r)
	finishReceiptRun(t, r, run, domain.RunStatusComplete)
	if _, err := terminalOutbox(t, r, codecAck).Dispatch(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	s := evidenceSelection(t, r, run)
	reader, _ := NewTerminalEvidenceReader(r, allowTerminalEvidence)
	mustReceiptExec(t, db, `DROP TRIGGER run_terminal_receipt_source_no_update; UPDATE run_terminal_receipt_source SET source_epoch='changed'`)
	if got, err := reader.Select(t.Context(), s); err == nil || got != (SelectedTerminalEvidence{}) {
		t.Fatal("corrupt source binding accepted")
	}
}

func TestTerminalEvidenceCorruptAckAndMismatchedIntent(t *testing.T) {
	for _, mode := range []string{"ack", "intent"} {
		t.Run(mode, func(t *testing.T) {
			db, done := setupTestDB(t)
			defer done()
			r := receiptRepos(t, db)
			run := receiptRun(t, r)
			finishReceiptRun(t, r, run, domain.RunStatusComplete)
			if _, err := terminalOutbox(t, r, codecAck).Dispatch(t.Context(), 1); err != nil {
				t.Fatal(err)
			}
			s := evidenceSelection(t, r, run)
			if mode == "ack" {
				mustReceiptExec(t, db, `DROP TRIGGER run_terminal_event_acknowledgments_no_update; UPDATE run_terminal_event_acknowledgments SET acknowledgment_json='{}'`)
			} else {
				// Tamper both integrity hashes and the ack coherently. Individually valid
				// bytes still cannot join when the event describes a different status.
				receipt := readReceipt(t, r, run)
				var intent eventbus.DomainEvent
				if err := json.Unmarshal([]byte(receipt.EventIntentJSON), &intent); err != nil {
					t.Fatal(err)
				}
				intent.Payload["status"] = "failed"
				encoded, err := json.Marshal(intent)
				if err != nil {
					t.Fatal(err)
				}
				id := runmodel.SHA256(encoded)
				ack, err := codecAck(t.Context(), string(encoded), id)
				if err != nil {
					t.Fatal(err)
				}
				ackJSON, _ := json.Marshal(ack)
				mustReceiptExec(t, db, `DROP TRIGGER run_terminal_receipts_no_update; DROP TRIGGER run_terminal_event_acknowledgments_no_update`)
				tx, err := db.BeginTxx(t.Context(), nil)
				if err != nil {
					t.Fatal(err)
				}
				defer tx.Rollback()
				if _, err := tx.Exec(`PRAGMA defer_foreign_keys=ON`); err != nil {
					t.Fatal(err)
				}
				if _, err := tx.Exec(`UPDATE run_terminal_receipts SET event_intent_json=?,event_intent_id=?`, string(encoded), id); err != nil {
					t.Fatal(err)
				}
				if _, err := tx.Exec(`UPDATE run_terminal_event_acknowledgments SET event_intent_id=?,acknowledgment_json=?`, id, string(ackJSON)); err != nil {
					t.Fatal(err)
				}
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
				s.TerminalEventID = id
				s.CanonicalEventSHA256 = ack.CanonicalSHA256
			}
			reader, _ := NewTerminalEvidenceReader(r, allowTerminalEvidence)
			if got, err := reader.Select(t.Context(), s); err == nil || got != (SelectedTerminalEvidence{}) {
				t.Fatalf("corrupt %s qualified %#v %v", mode, got, err)
			}
		})
	}
}
