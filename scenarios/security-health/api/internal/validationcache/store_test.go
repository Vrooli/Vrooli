package validationcache

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"security-health/internal/validation"

	_ "modernc.org/sqlite"
)

func testStore(t *testing.T) (*Store, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/cache.db?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	return New(db), db
}

func TestStoreRoundTripAndFingerprintInvalidation(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	key := validation.EvidenceKey{Scenario: "demo", Scanner: "gitleaks", Fingerprint: "sha256:old"}
	record := validation.EvidenceRecord{
		Key: key,
		Findings: []validation.Finding{{
			RuleID: "gitleaks.generic", Severity: validation.SeverityError,
			Title: "Potential secret detected", Description: "matched value is redacted",
			Remediation: "rotate it", FilePath: "api/main.go:10", Scanner: "gitleaks",
			Class: validation.FindingSecret,
		}},
		ExpiresAt: now.Add(time.Hour),
	}
	if err := store.Store(ctx, record, now); err != nil {
		t.Fatal(err)
	}
	got, ok, err := store.Load(ctx, key, now)
	if err != nil || !ok {
		t.Fatalf("load ok=%v err=%v", ok, err)
	}
	if len(got.Findings) != 1 || got.Findings[0].RuleID != record.Findings[0].RuleID {
		t.Fatalf("record = %+v", got)
	}
	changed := key
	changed.Fingerprint = "sha256:new"
	if _, ok, err := store.Load(ctx, changed, now); err != nil || ok {
		t.Fatalf("changed fingerprint ok=%v err=%v", ok, err)
	}
	if err := store.Store(ctx, validation.EvidenceRecord{Key: changed, ExpiresAt: now.Add(time.Hour)}, now); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.Load(ctx, key, now); err != nil || ok {
		t.Fatalf("old fingerprint survived replacement: ok=%v err=%v", ok, err)
	}
}

// [REQ:REQ-P0-022]
func TestStorePersistsOnlyNormalizedFindingPayload(t *testing.T) {
	t.Log("[REQ:REQ-P0-022]")
	store, db := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	const rawSecretCanary = "AKIA-RAW-SECRET-CANARY"
	record := validation.EvidenceRecord{
		Key: validation.EvidenceKey{Scenario: "demo", Scanner: "gitleaks", Fingerprint: "sha256:redacted"},
		Findings: []validation.Finding{{
			RuleID: "gitleaks.generic", Severity: validation.SeverityError,
			Title: "Potential secret detected", Description: "matched value is redacted",
			Remediation: "rotate it", FilePath: "secret.env:1", Scanner: "gitleaks",
			Class: validation.FindingSecret,
		}},
		ExpiresAt: now.Add(time.Hour),
	}
	if err := store.Store(ctx, record, now); err != nil {
		t.Fatal(err)
	}
	var payload string
	if err := db.QueryRow(`SELECT findings_json FROM validation_evidence_cache`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(payload, rawSecretCanary) || strings.Contains(payload, `"secret":`) || strings.Contains(payload, `"match":`) {
		t.Fatalf("scanner-native secret material persisted: %s", payload)
	}
	if !strings.Contains(payload, `"rule_id":"gitleaks.generic"`) {
		t.Fatalf("normalized finding missing: %s", payload)
	}
}

func TestStoreCorruptionAndExpiryFailClosed(t *testing.T) {
	store, db := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	key := validation.EvidenceKey{Scenario: "demo", Scanner: "gosec", Fingerprint: "sha256:corrupt"}
	_, err := db.Exec(`
		INSERT INTO validation_evidence_cache
		(scenario, scanner, fingerprint, payload_version, findings_json, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		key.Scenario, key.Scanner, key.Fingerprint, payloadVersion, `{bad`,
		now.Format(time.RFC3339Nano), now.Add(time.Hour).Format(time.RFC3339Nano))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.Load(ctx, key, now); err == nil || ok {
		t.Fatalf("corrupt load ok=%v err=%v", ok, err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM validation_evidence_cache`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("corrupt row count = %d, want 0", count)
	}

	expired := validation.EvidenceRecord{Key: key, ExpiresAt: now.Add(time.Minute)}
	if err := store.Store(ctx, expired, now); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.Load(ctx, key, now.Add(2*time.Minute)); err != nil || ok {
		t.Fatalf("expired load ok=%v err=%v", ok, err)
	}
}

func TestDeleteExpiredIsBounded(t *testing.T) {
	store, db := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	for _, scanner := range []string{"one", "two", "three"} {
		key := validation.EvidenceKey{Scenario: "demo", Scanner: scanner, Fingerprint: "sha256:" + scanner}
		if err := store.Store(ctx, validation.EvidenceRecord{Key: key, ExpiresAt: now.Add(time.Minute)}, now); err != nil {
			t.Fatal(err)
		}
	}
	deleted, err := store.DeleteExpired(ctx, now.Add(2*time.Minute), 2)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 2 {
		t.Fatalf("deleted = %d, want 2", deleted)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM validation_evidence_cache`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("remaining = %d, want 1", count)
	}
}
