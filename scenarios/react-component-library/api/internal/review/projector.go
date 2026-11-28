package review

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// EvidenceProjector turns current, failing evidence in catalog_gate_evidence
// into review findings. It reads another domain's table but writes nothing:
// catalog coverage remains the owner of evidence, and Review owns the projected
// queue.
//
// Two rules keep Review about the present:
//
//  1. Only the newest observation for an (asset, gate, version) identity is
//     considered. A later pass therefore clears an older failure instead of
//     leaving a stale finding behind forever.
//  2. Only an actual failure is a finding. Passing, unmeasured, skipped and
//     other non-failure observations are evidence gaps or successes; Coverage
//     owns them. Treating "unmeasured" as a medium finding is what previously
//     filled the queue with items that were never failures.
type EvidenceProjector struct {
	db    queryPort
	clock func() time.Time
}

// NewEvidenceProjector reads catalog evidence through the supplied SQL query
// port. Production wires the routed database so request contexts select the
// correct pool. A database without catalog_gate_evidence projects no findings.
func NewEvidenceProjector(db queryPort) *EvidenceProjector {
	return &EvidenceProjector{db: db, clock: time.Now}
}

// Project returns one finding per current, failing (asset, gate, version).
// Evidence for a superseded version is retained by Coverage but is not a review
// item; otherwise the queue grows without bound every time an asset is
// released. Non-catalog observations (the corpus and supplemental probes) and
// catalog definitions with no published component are not asset findings
// either: nobody can act on them from the library.
func (p *EvidenceProjector) Project(ctx context.Context) ([]Finding, error) {
	if p == nil || p.db == nil {
		return nil, fmt.Errorf("evidence projector is not configured")
	}
	columns, err := tableColumns(ctx, p.db, "catalog_gate_evidence")
	if err != nil {
		return nil, fmt.Errorf("inspect catalog_gate_evidence: %w", err)
	}
	if len(columns) == 0 {
		// Nil means "this database has no evidence owner", not "the evidence
		// owner says there are zero findings". The service must not prune a
		// durable queue because a focused database never created the table.
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
	// Collapse history in SQL rather than reading every retained observation.
	// A pass on the newest row suppresses the identity even if it failed before.
	query := `WITH ranked AS (
  SELECT asset_id, target, gate, version, result, ` + measurement + ` AS measurement, recorded_at,
         ROW_NUMBER() OVER (
           PARTITION BY asset_id, gate, version
           ORDER BY recorded_at DESC, id DESC
         ) AS rn
  FROM catalog_gate_evidence
)
SELECT asset_id, target, gate, version, result, measurement, recorded_at
FROM ranked
WHERE rn = 1
ORDER BY asset_id, gate, version`

	live, err := p.liveVersions(ctx)
	if err != nil {
		return nil, err
	}

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
		if !isFailureResult(result) {
			continue
		}
		if isNonCatalogObservation(assetID) || !live.isLive(assetID, version) {
			continue
		}
		subjectKind := "asset"
		id := findingID(subjectKind, assetID, gate, version)
		byID[id] = Finding{
			ID:           id,
			SubjectKind:  subjectKind,
			SubjectID:    assetID,
			Gate:         gate,
			Severity:     severityForGate(gate),
			Status:       StatusOpen,
			Message:      findingMessage(gate, assetID, version, result),
			EvidenceJSON: evidenceJSON(target, version, result, measurement, recordedAt),
			DocsRef:      docsRefForGate(gate),
			CreatedAt:    normalizeRecordedAt(recordedAt, p.now()),
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

// liveVersionIndex is the set of versions a person can act on today: each
// component's latest release and its open draft. It is keyed twice because
// evidence uses the catalog id ("adapters.validation") while components use a
// library id ("react-component-library:ValidationAdapter"). The catalogId
// header is authoritative; normalized slug matching is the fallback for
// components whose header has not been indexed yet.
type liveVersionIndex struct {
	enabled     bool
	byCatalogID map[string]map[string]bool
	byName      map[string]map[string]bool
}

func (live liveVersionIndex) isLive(assetID, version string) bool {
	if !live.enabled {
		return true
	}
	versions, known := live.versionsFor(assetID)
	if !known {
		// A catalog asset with no published component is not review work. It
		// may still be a Coverage gap, but nobody can act on it in the library.
		return false
	}
	return versions[strings.TrimSpace(version)]
}

func (live liveVersionIndex) versionsFor(assetID string) (map[string]bool, bool) {
	if versions, ok := live.byCatalogID[strings.TrimSpace(assetID)]; ok {
		return versions, true
	}
	versions, ok := live.byName[normalizeAssetName(assetID)]
	return versions, ok
}

// liveVersions builds the index above. When the component table is absent
// (focused tests and read-only consumers) scoping is disabled rather than
// hiding evidence.
func (p *EvidenceProjector) liveVersions(ctx context.Context) (liveVersionIndex, error) {
	live := liveVersionIndex{
		byCatalogID: map[string]map[string]bool{},
		byName:      map[string]map[string]bool{},
	}
	columns, err := tableColumns(ctx, p.db, "components")
	if err != nil {
		return live, fmt.Errorf("inspect components: %w", err)
	}
	if len(columns) == 0 {
		return live, nil
	}
	if !columns["slug"] || !columns["latest_version"] || !columns["draft_version"] {
		return live, nil
	}
	live.enabled = true

	rows, err := p.db.QueryContext(ctx, `SELECT slug, latest_version, draft_version FROM components`)
	if err != nil {
		return live, fmt.Errorf("read component versions: %w", err)
	}
	for rows.Next() {
		var slug, latest, draft string
		if err := rows.Scan(&slug, &latest, &draft); err != nil {
			rows.Close()
			return live, fmt.Errorf("scan component version: %w", err)
		}
		if versions := currentVersions(latest, draft); len(versions) > 0 {
			live.byName[normalizeAssetName(slug)] = versions
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return live, fmt.Errorf("iterate component versions: %w", err)
	}
	rows.Close()

	headerColumns, err := tableColumns(ctx, p.db, "component_headers")
	if err != nil {
		return live, fmt.Errorf("inspect component_headers: %w", err)
	}
	if !headerColumns["component_id"] || !headerColumns["field"] || !headerColumns["value"] {
		return live, nil
	}
	headerRows, err := p.db.QueryContext(ctx, `
SELECT h.value, c.latest_version, c.draft_version
FROM component_headers h
JOIN components c ON c.id = h.component_id
WHERE h.field = 'catalogId'`)
	if err != nil {
		return live, fmt.Errorf("read component catalog ids: %w", err)
	}
	defer headerRows.Close()
	for headerRows.Next() {
		var catalogID, latest, draft string
		if err := headerRows.Scan(&catalogID, &latest, &draft); err != nil {
			return live, fmt.Errorf("scan component catalog id: %w", err)
		}
		if versions := currentVersions(latest, draft); len(versions) > 0 {
			live.byCatalogID[strings.TrimSpace(catalogID)] = versions
		}
	}
	if err := headerRows.Err(); err != nil {
		return live, fmt.Errorf("iterate component catalog ids: %w", err)
	}
	return live, nil
}

func currentVersions(latest, draft string) map[string]bool {
	versions := make(map[string]bool, 2)
	for _, version := range []string{strings.TrimSpace(latest), strings.TrimSpace(draft)} {
		if version != "" {
			versions[version] = true
		}
	}
	return versions
}

// normalizeAssetName bridges the catalog's dotted ids ("controls.button",
// "react-component-library:Button") and the component slug ("Button") without
// pretending they are the same namespace. Only alphanumerics survive, so
// "use-press", "usePress" and "UsePress" compare equal.
func normalizeAssetName(id string) string {
	name := strings.TrimSpace(id)
	if index := strings.LastIndex(name, ":"); index >= 0 {
		name = name[index+1:]
	}
	if index := strings.LastIndex(name, "."); index >= 0 {
		name = name[index+1:]
	}
	var builder strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// isFailureResult distinguishes a failed check from the absence of a result.
// Unmeasured and skipped checks are coverage gaps, not review findings.
func isFailureResult(result string) bool {
	switch strings.ToLower(strings.TrimSpace(result)) {
	case "", "pass", "passed", "skipped", "skip", "unmeasured", "not-run", "not_run", "pending":
		return false
	default:
		return true
	}
}

// isNonCatalogObservation reports observations that belong to the corpus or
// workbench probes rather than to a catalog asset. Coverage keeps them; Review
// does not ask a person to triage the corpus as if it were a component.
func isNonCatalogObservation(assetID string) bool {
	return strings.HasPrefix(assetID, "__corpus__") ||
		strings.HasPrefix(assetID, "workbench.") ||
		strings.HasPrefix(assetID, "supplemental.")
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
	// "reduced-motion" reads as a check, not an internal identifier. Keep the
	// gate name lowercase so searches for the identifier still match.
	gateLabel := strings.ReplaceAll(strings.TrimSpace(gate), "-", " ")
	return fmt.Sprintf("%s check failed for %s at %s (result %s)", gateLabel, assetID, versionLabel, resultLabel)
}

func evidenceJSON(target, version, result, measurement, recordedAt string) string {
	payload := map[string]any{
		"target":      target,
		"version":     version,
		"result":      result,
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
func tableColumns(ctx context.Context, db queryPort, table string) (map[string]bool, error) {
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
