package review

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// EvidenceProjector turns failing rows in catalog_gate_evidence into review
// findings. It reads another domain's table but writes nothing: catalog
// coverage remains the owner of evidence, and Review owns the projected queue.
type EvidenceProjector struct {
	db    *sql.DB
	clock func() time.Time
}

// NewEvidenceProjector reads the scenario's primary pool. catalog_gate_evidence
// is created by internal/catalogcoverage; a database that predates that schema
// projects no findings rather than failing the Review surface.
func NewEvidenceProjector(db *sql.DB) *EvidenceProjector {
	return &EvidenceProjector{db: db, clock: time.Now}
}

// Project returns one finding per failing (asset, gate, version). Repeated
// evidence rows for the same identity are collapsed to the newest recorded
// observation so retention history cannot duplicate a queue item.
func (p *EvidenceProjector) Project(ctx context.Context) ([]Finding, error) {
	if p == nil || p.db == nil {
		return nil, fmt.Errorf("evidence projector is not configured")
	}
	columns, err := tableColumns(ctx, p.db, "catalog_gate_evidence")
	if err != nil {
		return nil, fmt.Errorf("inspect catalog_gate_evidence: %w", err)
	}
	if len(columns) == 0 {
		return nil, nil
	}
	required := []string{"asset_id", "target", "gate", "version", "result", "recorded_at"}
	for _, column := range required {
		if !columns[column] {
			return nil, fmt.Errorf("catalog_gate_evidence is missing required column %q", column)
		}
	}
	measurement := "''"
	if columns["measurement_json"] {
		measurement = "measurement_json"
	}
	query := `SELECT asset_id, target, gate, version, result, ` + measurement + `, recorded_at
FROM catalog_gate_evidence
WHERE LOWER(TRIM(result)) <> 'pass'
ORDER BY asset_id, gate, version, recorded_at DESC`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("read gate evidence: %w", err)
	}
	defer rows.Close()

	byID := make(map[string]Finding)
	for rows.Next() {
		var (
			assetID     string
			target      string
			gate        string
			version     string
			result      string
			measurement string
			recordedAt  string
		)
		if err := rows.Scan(&assetID, &target, &gate, &version, &result, &measurement, &recordedAt); err != nil {
			return nil, fmt.Errorf("scan gate evidence: %w", err)
		}
		subjectKind := "asset"
		id := findingID(subjectKind, assetID, gate, version)
		candidate := Finding{
			ID:           id,
			SubjectKind:  subjectKind,
			SubjectID:    assetID,
			Gate:         gate,
			Severity:     severityForGate(gate),
			Status:       StatusOpen,
			Message:      findingMessage(gate, assetID, version, result),
			EvidenceJSON: evidenceJSON(target, version, measurement, recordedAt),
			DocsRef:      docsRefForGate(gate),
			CreatedAt:    normalizeRecordedAt(recordedAt, p.now()),
		}
		// The query is newest-first per identity, so the first row wins. The
		// explicit comparison keeps that true if the ORDER BY is ever changed.
		existing, seen := byID[id]
		if !seen || candidate.CreatedAt > existing.CreatedAt {
			byID[id] = candidate
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gate evidence: %w", err)
	}

	findings := make([]Finding, 0, len(byID))
	for _, finding := range byID {
		findings = append(findings, finding)
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].ID < findings[j].ID })
	return findings, nil
}

func (p *EvidenceProjector) now() time.Time {
	if p.clock != nil {
		return p.clock().UTC()
	}
	return time.Now().UTC()
}

// findingID is the projection identity. The same failing gate/version keeps
// one stable id across refreshes, which is what lets status and owner survive.
func findingID(subjectKind, subjectID, gate, version string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{subjectKind, subjectID, gate, version}, "|")))
	return hex.EncodeToString(sum[:])[:16]
}

// severityForGate maps the consequence of a gate family. Release provenance and
// accessibility failures block release; composition and contract failures are
// high; everything else is medium by default.
func severityForGate(gate string) string {
	lower := strings.ToLower(gate)
	switch {
	case strings.Contains(lower, "release-provenance"), strings.Contains(lower, "accessibility"):
		return SeverityBlocker
	case strings.Contains(lower, "composition"), strings.Contains(lower, "contract"):
		return SeverityHigh
	default:
		return SeverityMedium
	}
}

func findingMessage(gate, assetID, version, result string) string {
	versionLabel := strings.TrimSpace(version)
	if versionLabel == "" {
		versionLabel = "unversioned"
	}
	resultLabel := strings.TrimSpace(result)
	if resultLabel == "" {
		resultLabel = "unknown"
	}
	return fmt.Sprintf("%s gate failed for %s at %s (result %s)", gate, assetID, versionLabel, resultLabel)
}

func evidenceJSON(target, version, measurement, recordedAt string) string {
	payload := map[string]any{
		"target":      target,
		"version":     version,
		"recorded_at": recordedAt,
	}
	if trimmed := strings.TrimSpace(measurement); trimmed != "" {
		var decoded any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
			payload["measurement"] = decoded
		} else {
			// Measurement producers are expected to emit JSON, but an older or
			// malformed value must stay visible rather than disappear.
			payload["measurement"] = measurement
		}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func normalizeRecordedAt(value string, fallback time.Time) string {
	if parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value)); err == nil {
		return parsed.UTC().Format(timestampLayout)
	}
	return fallback.Format(timestampLayout)
}

// docsByGate is intentionally small: it points the queue at the document that
// explains the gate family, not at a generated per-gate page that would drift.
var docsByGate = map[string]string{
	"accessibility":            "docs/concepts/STORY-CONTRACT.md",
	"release-provenance":       "docs/concepts/DATA.md",
	"composition":              "docs/concepts/UI-SPEC-RECONCILIATION.md",
	"composition-contract":     "docs/concepts/UI-SPEC-RECONCILIATION.md",
	"contract":                 "docs/concepts/DOMAINS.md",
	"usage-contract":           "docs/concepts/DOMAINS.md",
	"version-mirror-integrity": "docs/concepts/DATA.md",
}

func docsRefForGate(gate string) string {
	if ref, ok := docsByGate[strings.ToLower(strings.TrimSpace(gate))]; ok {
		return ref
	}
	return "docs/concepts/one-asset-one-verdict.md"
}

// tableColumns returns the column set for a table, or an empty map when the
// table does not exist. Names are SQL identifiers from package constants, never
// caller input.
func tableColumns(ctx context.Context, db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var (
			cid       int
			name      string
			columnTyp string
			notNull   int
			defaultV  any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &columnTyp, &notNull, &defaultV, &pk); err != nil {
			return nil, err
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return columns, nil
}
