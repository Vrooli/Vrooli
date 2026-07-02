// Package matrix is the ONE traceability join: OT × requirement ×
// validation × evidence. Every surface that renders traceability (report
// CLI, assessment findings, UI matrix, fleet rollups) consumes this
// package's output — no surface re-derives the join.
package matrix

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"business-health/internal/evidence"
	"business-health/internal/extraction"

	intent "intent-go"
)

// Row is one OT × requirement pairing. An empty RequirementID marks an
// orphaned operational target; an empty OTID marks a requirement whose
// prd_ref resolves to nothing.
type Row struct {
	OTID       string
	OTTitle    string
	OTChecked  bool
	OTPriority string

	RequirementID     string
	RequirementTitle  string
	RequirementStatus string
	Criticality       string
	Module            string

	Validations []ValidationCell
	Evidence    EvidenceCell

	Unproven       bool
	UnprovenReason string
}

// ValidationCell is one declared validation entry.
type ValidationCell struct {
	Type      string
	Phase     string
	Status    string
	Ref       string
	RefExists bool
}

// EvidenceCell is the read-only evidence rollup for one requirement.
type EvidenceCell struct {
	// OTStatus/OTCompletionRate come from the snapshot's OT rollup (the
	// per-requirement grain test-genie publishes).
	OTStatus         string
	OTCompletionRate float64
	SnapshotAt       time.Time
	SnapshotStale    bool
	Manual           *evidence.Attestation
	ManualExpired    bool
}

// DriftEntry is one evidence drift/staleness observation.
type DriftEntry struct {
	// Kind: "stale_snapshot" | "expired_manual" | "status_unearned" |
	// "unproven_claim".
	Kind      string
	SubjectID string
	Detail    string
}

// Result is the joined traceability model for one scenario.
type Result struct {
	Scenario string
	Rows     []Row
	Drift    []DriftEntry
	// DegradedReason is non-empty when evidence artifacts were missing or
	// unreadable (declared state still renders).
	DegradedReason string
	Registry       RegistrySummary
	AsOf           time.Time
}

// RegistrySummary is the registry shape at a glance.
type RegistrySummary struct {
	ModuleCount            int
	RequirementCount       int
	OperationalTargetCount int
	StatusCounts           map[string]int
	StarterTemplate        bool
}

// Inputs carries everything the join reads. All fields are plain data —
// the join itself is pure.
type Inputs struct {
	Contract     extraction.Contract
	Snapshot     evidence.SyncSnapshot
	HasSnapshot  bool
	Staleness    evidence.Staleness
	Attestations map[string]evidence.Attestation
	Now          time.Time
}

// completeStatuses are requirement statuses that CLAIM the work is done —
// the statuses that need evidence to be honest.
var completeStatuses = map[string]struct{}{"complete": {}}

// Join computes the traceability matrix and drift entries.
func Join(in Inputs) Result {
	c := in.Contract
	res := Result{Scenario: c.Scenario, AsOf: in.Now}
	res.Registry = summarizeRegistry(c)

	if !in.HasSnapshot {
		res.DegradedReason = in.Staleness.Detail
	}

	otByID := make(map[string]intent.OperationalTarget, len(c.PRDDoc.Targets))
	for _, ot := range c.PRDDoc.Targets {
		otByID[ot.ID] = ot
	}
	snapOTByID := make(map[string]evidence.SyncTarget, len(in.Snapshot.OperationalTargets))
	for _, t := range in.Snapshot.OperationalTargets {
		snapOTByID[intent.CanonicalOTID(t.ID)] = t
	}

	// Requirement rows (one per requirement; OT side joined via prd_ref).
	coveredOTs := make(map[string]struct{})
	for _, r := range c.Registry.Requirements() {
		if r.ID == "" {
			continue
		}
		row := Row{
			RequirementID:     r.ID,
			RequirementTitle:  r.Title,
			RequirementStatus: r.Status,
			Criticality:       r.Criticality,
			Module:            r.Module,
		}
		otID := intent.CanonicalOTID(r.PRDRef)
		if ot, ok := otByID[otID]; ok {
			row.OTID = ot.ID
			row.OTTitle = ot.Title
			row.OTChecked = ot.Checked
			row.OTPriority = ot.Tier
			coveredOTs[ot.ID] = struct{}{}
		}
		for _, v := range r.Validations {
			cell := ValidationCell{Type: v.Type, Phase: v.Phase, Status: v.Status, Ref: v.Ref}
			cell.RefExists = refExists(c, v)
			row.Validations = append(row.Validations, cell)
		}
		row.Evidence = evidenceCellFor(row, snapOTByID, in)
		row.Unproven, row.UnprovenReason = unprovenVerdict(row, in)
		res.Rows = append(res.Rows, row)
	}

	// Orphaned operational targets (no requirement points at them).
	for _, ot := range c.PRDDoc.Targets {
		if _, ok := coveredOTs[ot.ID]; ok {
			continue
		}
		row := Row{OTID: ot.ID, OTTitle: ot.Title, OTChecked: ot.Checked, OTPriority: ot.Tier}
		row.Unproven = ot.Checked
		if ot.Checked {
			row.UnprovenReason = "operational target is checked off but no requirement points at it"
		}
		res.Rows = append(res.Rows, row)
	}

	sort.SliceStable(res.Rows, func(i, j int) bool {
		if res.Rows[i].OTID != res.Rows[j].OTID {
			return res.Rows[i].OTID < res.Rows[j].OTID
		}
		return res.Rows[i].RequirementID < res.Rows[j].RequirementID
	})

	res.Drift = deriveDrift(res, in)
	return res
}

func summarizeRegistry(c extraction.Contract) RegistrySummary {
	sum := RegistrySummary{
		ModuleCount:            len(c.Registry.Modules),
		OperationalTargetCount: len(c.PRDDoc.Targets),
		StatusCounts:           map[string]int{},
	}
	for _, r := range c.Registry.Requirements() {
		sum.RequirementCount++
		if r.Status != "" {
			sum.StatusCounts[r.Status]++
		}
		if r.HasTag("template-starter") {
			sum.StarterTemplate = true
		}
	}
	return sum
}

// refExists resolves a validation ref through the contract's requirement
// claims (whose refs were normalized and path-checked by intent-go during
// extraction) — the join never touches the filesystem.
func refExists(c extraction.Contract, v intent.RegistryValidation) bool {
	if strings.TrimSpace(v.Ref) == "" {
		return false
	}
	ref := intent.NormalizeRef(v.Ref, v.Type)
	if ref.Kind != intent.RefCode {
		return true // doc/manual refs are not path-checked
	}
	for _, claim := range c.Requirements {
		for _, r := range claim.Refs {
			if r.Raw == v.Ref {
				return refResolves(c.ScenarioDir, r)
			}
		}
	}
	return refResolves(c.ScenarioDir, ref)
}

func refResolves(scenarioDir string, ref intent.Ref) bool {
	// Reuse intent-go's existence semantics via CheckRefExists on a
	// single-claim probe: zero findings means the ref resolves.
	probe := []intent.CapabilityClaim{{ID: "probe", Altitude: intent.Requirement, Refs: []intent.Ref{ref}}}
	return len(intent.CheckRefExists(scenarioDir, probe)) == 0
}

func evidenceCellFor(row Row, snapOTs map[string]evidence.SyncTarget, in Inputs) EvidenceCell {
	cell := EvidenceCell{SnapshotStale: in.Staleness.Stale}
	if in.HasSnapshot {
		cell.SnapshotAt = in.Snapshot.GeneratedAt
		if row.OTID != "" {
			if t, ok := snapOTs[row.OTID]; ok {
				cell.OTStatus = t.Status
				cell.OTCompletionRate = t.CompletionRate
			}
		}
	}
	if a, ok := in.Attestations[row.RequirementID]; ok {
		att := a
		cell.Manual = &att
		cell.ManualExpired = a.Expired(in.Now)
	}
	return cell
}

// unprovenVerdict derives the honesty flag for one requirement row.
func unprovenVerdict(row Row, in Inputs) (bool, string) {
	_, claimsComplete := completeStatuses[row.RequirementStatus]
	if !claimsComplete {
		return false, ""
	}
	// A complete claim with no snapshot ever written is pure assertion.
	if !in.HasSnapshot {
		return true, fmt.Sprintf("requirement %s is declared complete but no requirements-sync snapshot exists to earn it", row.RequirementID)
	}
	// A complete claim whose only validations are manual needs an unexpired
	// attestation.
	onlyManual := len(row.Validations) > 0
	for _, v := range row.Validations {
		if v.Type != "manual" {
			onlyManual = false
			break
		}
	}
	if onlyManual {
		if row.Evidence.Manual == nil {
			return true, fmt.Sprintf("requirement %s is declared complete but its manual validation was never attested (manual-log)", row.RequirementID)
		}
		if row.Evidence.ManualExpired {
			return true, fmt.Sprintf("requirement %s is declared complete but its only attestation expired %s", row.RequirementID, row.Evidence.Manual.ExpiresAt.UTC().Format(time.RFC3339))
		}
	}
	return false, ""
}

func deriveDrift(res Result, in Inputs) []DriftEntry {
	var out []DriftEntry
	if in.Staleness.Stale {
		out = append(out, DriftEntry{Kind: "stale_snapshot", Detail: in.Staleness.Detail})
	}
	for _, row := range res.Rows {
		if row.Evidence.Manual != nil && row.Evidence.ManualExpired {
			out = append(out, DriftEntry{
				Kind:      "expired_manual",
				SubjectID: row.RequirementID,
				Detail:    fmt.Sprintf("latest attestation by %s expired %s", row.Evidence.Manual.AttestedBy, row.Evidence.Manual.ExpiresAt.UTC().Format(time.RFC3339)),
			})
		}
		if row.Unproven {
			subject := row.RequirementID
			if subject == "" {
				subject = row.OTID
			}
			kind := "unproven_claim"
			if row.RequirementID != "" && !in.HasSnapshot {
				kind = "status_unearned"
			}
			out = append(out, DriftEntry{Kind: kind, SubjectID: subject, Detail: row.UnprovenReason})
		}
		// Checked OT without a complete linked requirement: the classic
		// premature checkbox.
		if row.OTChecked && row.RequirementID != "" {
			if _, done := completeStatuses[row.RequirementStatus]; !done {
				out = append(out, DriftEntry{
					Kind:      "unproven_claim",
					SubjectID: row.OTID,
					Detail:    fmt.Sprintf("operational target %s is checked off but linked requirement %s is %q", row.OTID, row.RequirementID, displayStatus(row.RequirementStatus)),
				})
			}
		}
	}
	return out
}

func displayStatus(s string) string {
	if s == "" {
		return "unset"
	}
	return s
}

// ModuleName renders a module path as its folder name (UI/CLI nicety).
func ModuleName(modulePath string) string {
	return path.Base(path.Dir(modulePath))
}
