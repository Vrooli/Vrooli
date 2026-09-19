package validationcache

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"security-health/internal/validation"
)

const (
	payloadVersion          = 1
	maxEvidencePayloadBytes = 8 << 20
)

// SQLExecutor is satisfied by *sql.DB and api-core's routed database.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store is the SQLite implementation of validation.EvidenceStore.
type Store struct {
	db SQLExecutor
}

// New constructs a validation evidence store over the scenario-owned DB.
func New(db SQLExecutor) *Store { return &Store{db: db} }

// Load returns only an exact, unexpired, supported-version match. Corrupt rows
// are removed and reported so the coordinator can record a cache error and run
// the scanner normally.
func (s *Store) Load(ctx context.Context, key validation.EvidenceKey, now time.Time) (validation.EvidenceRecord, bool, error) {
	if err := key.Validate(); err != nil {
		return validation.EvidenceRecord{}, false, err
	}
	var fingerprint, findingsJSON, expiresAt string
	var version int
	err := s.db.QueryRowContext(ctx, `
		SELECT fingerprint, payload_version, findings_json, expires_at
		FROM validation_evidence_cache
		WHERE scenario = ? AND scanner = ?`, key.Scenario, key.Scanner,
	).Scan(&fingerprint, &version, &findingsJSON, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return validation.EvidenceRecord{}, false, nil
	}
	if err != nil {
		return validation.EvidenceRecord{}, false, fmt.Errorf("load validation evidence: %w", err)
	}
	if fingerprint != key.Fingerprint || version != payloadVersion {
		return validation.EvidenceRecord{}, false, nil
	}
	expires, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		s.deleteExact(ctx, key)
		return validation.EvidenceRecord{}, false, fmt.Errorf("invalid validation evidence expiry: %w", err)
	}
	if !expires.After(now) {
		s.deleteExact(ctx, key)
		return validation.EvidenceRecord{}, false, nil
	}
	if len(findingsJSON) > maxEvidencePayloadBytes {
		s.deleteExact(ctx, key)
		return validation.EvidenceRecord{}, false, errors.New("validation evidence payload exceeds limit")
	}
	var payload cachedPayload
	if err := json.Unmarshal([]byte(findingsJSON), &payload); err != nil || payload.Version != payloadVersion {
		s.deleteExact(ctx, key)
		if err == nil {
			err = fmt.Errorf("unsupported payload version %d", payload.Version)
		}
		return validation.EvidenceRecord{}, false, fmt.Errorf("decode validation evidence: %w", err)
	}
	return validation.EvidenceRecord{
		Key:       key,
		Findings:  fromCachedFindings(payload.Findings),
		ExpiresAt: expires,
	}, true, nil
}

// Store atomically replaces the current scanner row for the scenario.
func (s *Store) Store(ctx context.Context, record validation.EvidenceRecord, now time.Time) error {
	if err := record.Key.Validate(); err != nil {
		return err
	}
	if !record.ExpiresAt.After(now) {
		return errors.New("validation evidence expiry must be in the future")
	}
	payloadBytes, err := json.Marshal(cachedPayload{
		Version:  payloadVersion,
		Findings: toCachedFindings(record.Findings),
	})
	if err != nil {
		return fmt.Errorf("encode validation evidence: %w", err)
	}
	if len(payloadBytes) > maxEvidencePayloadBytes {
		return errors.New("validation evidence payload exceeds limit")
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO validation_evidence_cache (
			scenario, scanner, fingerprint, payload_version, findings_json, created_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(scenario, scanner) DO UPDATE SET
			fingerprint = excluded.fingerprint,
			payload_version = excluded.payload_version,
			findings_json = excluded.findings_json,
			created_at = excluded.created_at,
			expires_at = excluded.expires_at`,
		record.Key.Scenario,
		record.Key.Scanner,
		record.Key.Fingerprint,
		payloadVersion,
		string(payloadBytes),
		now.UTC().Format(time.RFC3339Nano),
		record.ExpiresAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("store validation evidence: %w", err)
	}
	return nil
}

// DeleteExpired removes at most limit stale rows. The bound keeps incidental
// cleanup from becoming request-path work proportional to fleet size.
func (s *Store) DeleteExpired(ctx context.Context, now time.Time, limit int) (int64, error) {
	if limit <= 0 {
		return 0, nil
	}
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM validation_evidence_cache
		WHERE rowid IN (
			SELECT rowid FROM validation_evidence_cache
			WHERE expires_at <= ? ORDER BY expires_at LIMIT ?
		)`, now.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return 0, fmt.Errorf("delete expired validation evidence: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count deleted validation evidence: %w", err)
	}
	return deleted, nil
}

func (s *Store) deleteExact(ctx context.Context, key validation.EvidenceKey) {
	_, _ = s.db.ExecContext(ctx, `
		DELETE FROM validation_evidence_cache
		WHERE scenario = ? AND scanner = ? AND fingerprint = ?`,
		key.Scenario, key.Scanner, key.Fingerprint,
	)
}

// cachedFinding is an explicit persistence allowlist. Adding a scanner-native
// field to validation.Finding cannot silently make it durable.
type cachedFinding struct {
	RuleID         string                   `json:"rule_id"`
	Severity       validation.Severity      `json:"severity"`
	Title          string                   `json:"title"`
	Description    string                   `json:"description"`
	Remediation    string                   `json:"remediation"`
	FilePath       string                   `json:"file_path"`
	Scanner        string                   `json:"scanner"`
	Class          validation.FindingClass  `json:"class"`
	Confidence     string                   `json:"confidence"`
	EvidenceState  validation.EvidenceState `json:"evidence_state"`
	Owner          string                   `json:"owner"`
	FixClass       validation.FixClass      `json:"fix_class"`
	FixPreviewable bool                     `json:"fix_previewable"`
	PolicyImpact   string                   `json:"policy_impact"`
}

type cachedPayload struct {
	Version  int             `json:"version"`
	Findings []cachedFinding `json:"findings"`
}

func toCachedFindings(findings []validation.Finding) []cachedFinding {
	out := make([]cachedFinding, 0, len(findings))
	for _, finding := range findings {
		out = append(out, cachedFinding{
			RuleID: finding.RuleID, Severity: finding.Severity, Title: finding.Title,
			Description: finding.Description, Remediation: finding.Remediation,
			FilePath: finding.FilePath, Scanner: finding.Scanner, Class: finding.Class,
			Confidence: finding.Confidence, EvidenceState: finding.EvidenceState,
			Owner: finding.Owner, FixClass: finding.FixClass,
			FixPreviewable: finding.FixPreviewable, PolicyImpact: finding.PolicyImpact,
		})
	}
	return out
}

func fromCachedFindings(findings []cachedFinding) []validation.Finding {
	out := make([]validation.Finding, 0, len(findings))
	for _, finding := range findings {
		out = append(out, validation.Finding{
			RuleID: finding.RuleID, Severity: finding.Severity, Title: finding.Title,
			Description: finding.Description, Remediation: finding.Remediation,
			FilePath: finding.FilePath, Scanner: finding.Scanner, Class: finding.Class,
			Confidence: finding.Confidence, EvidenceState: finding.EvidenceState,
			Owner: finding.Owner, FixClass: finding.FixClass,
			FixPreviewable: finding.FixPreviewable, PolicyImpact: finding.PolicyImpact,
		})
	}
	return out
}

var _ validation.EvidenceStore = (*Store)(nil)
