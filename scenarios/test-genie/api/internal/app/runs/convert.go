package runs

import (
	"encoding/json"
	"errors"
	"time"

	"test-genie/internal/orchestrator"
	sharedruns "test-genie/internal/shared/runs"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// Per-phase verdict values (also the surface-diff vocabulary in Plan A).
const (
	verdictClean         = "clean"
	verdictRegression    = "regression"
	verdictNewFailure    = "new-failure"
	verdictPreexisting   = "preexisting"
	verdictNotComparable = "not-comparable"
)

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func toRunInfo(r sharedruns.RunRecord) *runspb.RunInfo {
	return toTerminalRunInfo(r, nil, nil)
}

// toTerminalRunInfo is the single terminal RunInfo projector used by GetRun
// and WaitRun. The compact snapshot record owns identity/status/duration while
// the heavy result enriches each phase with its provider-computed standing.
func toTerminalRunInfo(r sharedruns.RunRecord, result *orchestrator.SuiteExecutionResult, descriptors *sharedruns.DescriptorSnapshot) *runspb.RunInfo {
	descriptorSchemaVersion := r.DescriptorSnapshotSchemaVersion
	descriptorDigest := r.DescriptorSnapshotDigest
	if descriptors != nil {
		if descriptorSchemaVersion == 0 {
			descriptorSchemaVersion = descriptors.SchemaVersion
		}
		if descriptorDigest == "" {
			descriptorDigest = descriptors.Digest
		}
	}
	resultPhases := make(map[string]orchestrator.PhaseExecutionResult)
	if result != nil {
		for _, phase := range result.Phases {
			resultPhases[phase.Name] = phase
		}
	}
	phases := make([]*runspb.PhaseInfo, 0, len(r.Phases))
	for _, p := range r.Phases {
		info := &runspb.PhaseInfo{
			Name:            p.Name,
			Status:          p.Status,
			DurationSeconds: float64(p.DurationSeconds),
			Comparable:      !p.NonComparable,
			Advisory:        p.Advisory,
			ArtifactBacked:  p.ArtifactBacked,
			NonComparable:   p.NonComparable,
		}
		if phase, ok := resultPhases[p.Name]; ok {
			info.MaturityStanding = phase.MaturityStanding
			info.FindingsSummary = phase.FindingsSummary
		}
		phases = append(phases, info)
	}
	pins := make([]*runspb.PinInfo, 0, len(r.Pins))
	for _, p := range r.Pins {
		pins = append(pins, &runspb.PinInfo{
			PinnedBy: p.PinnedBy,
			PinnedAt: formatTime(p.PinnedAt),
			Reason:   p.Reason,
		})
	}
	return &runspb.RunInfo{
		RunId:           r.RunID,
		Scenario:        r.Scenario,
		StartedAt:       formatTime(r.StartedAt),
		CompletedAt:     formatTime(r.CompletedAt),
		Status:          r.Status,
		Phases:          phases,
		GitSha:          r.GitSha,
		GitBranch:       r.GitBranch,
		GitDirty:        r.GitDirty,
		GitDirtySummary: r.GitDirtySummary,
		Diagnostics: &runspb.DiagnosticsInfo{
			Video:   r.Diagnostics.Video,
			Console: r.Diagnostics.Console,
			Network: r.Diagnostics.Network,
			Har:     r.Diagnostics.HAR,
			Trace:   r.Diagnostics.Trace,
			Dom:     r.Diagnostics.DOM,
		},
		Pins:                            pins,
		TreeDigest:                      r.TreeDigest,
		Preset:                          r.Preset,
		CaptureProfile:                  r.CaptureProfile,
		PlannedPhases:                   append([]string(nil), r.PlannedPhases...),
		PhaseSetDigest:                  r.PhaseSetDigest,
		DescriptorSnapshotSchemaVersion: int32(descriptorSchemaVersion),
		DescriptorSnapshotDigest:        descriptorDigest,
		DescriptorSnapshot:              toDescriptorSnapshot(descriptors),
	}
}

func toDescriptorSnapshot(snapshot *sharedruns.DescriptorSnapshot) *runspb.RunDescriptorSnapshot {
	if snapshot == nil {
		return nil
	}
	out := &runspb.RunDescriptorSnapshot{
		SchemaVersion: int32(snapshot.SchemaVersion),
		Digest:        snapshot.Digest,
		Phases:        make([]*runspb.RunPhaseDescriptor, 0, len(snapshot.Phases)),
	}
	for i := range snapshot.Phases {
		out.Phases = append(out.Phases, toPhaseDescriptor(&snapshot.Phases[i]))
	}
	return out
}

func toPhaseDescriptor(descriptor *sharedruns.PhaseDescriptorSnapshot) *runspb.RunPhaseDescriptor {
	if descriptor == nil {
		return nil
	}
	reasonCodes := make([]string, 0, len(descriptor.Applicability.Reasons))
	reasons := make([]string, 0, len(descriptor.Applicability.Reasons))
	for _, reason := range descriptor.Applicability.Reasons {
		reasonCodes = append(reasonCodes, reason.Code)
		reasons = append(reasons, reason.Message)
	}
	return &runspb.RunPhaseDescriptor{
		Phase:         descriptor.Phase,
		DisplayName:   descriptor.DisplayName,
		Description:   descriptor.Description,
		Provider:      descriptor.Provider,
		OrderHint:     int32(descriptor.OrderHint),
		PhaseClass:    descriptor.PhaseClass,
		RuntimeClass:  descriptor.RuntimeClass,
		Dimensions:    append([]string(nil), descriptor.Dimensions...),
		FindingSource: descriptor.FindingSource,
		Policy: &runspb.PhaseDescriptorPolicy{
			Selection: descriptor.Policy.Selection, ProviderReadiness: descriptor.Policy.ProviderReadiness,
			ProviderLifecycle: descriptor.Policy.ProviderLifecycle, Freshness: descriptor.Policy.Freshness,
			ResultGating: descriptor.Policy.ResultGating, Unavailable: descriptor.Policy.Unavailable,
		},
		DocsPath:             descriptor.DocsPath,
		MaturityReference:    descriptor.MaturityReference,
		ApplicabilityDefault: descriptor.ApplicabilityDefault,
		EvidenceKinds:        append([]string(nil), descriptor.EvidenceKinds...),
		Aliases:              append([]string(nil), descriptor.Aliases...),
		Supersedes:           append([]string(nil), descriptor.Supersedes...),
		Applicability: &runspb.PhaseApplicabilityDecision{
			Status: descriptor.Applicability.Status, ReasonCodes: reasonCodes,
			Reasons: reasons, Planned: descriptor.Applicability.Planned,
		},
	}
}

type runProjection struct {
	record        sharedruns.RunRecord
	result        *orchestrator.SuiteExecutionResult
	schemaVersion int
	descriptors   *sharedruns.DescriptorSnapshot
	descriptorErr error
	degraded      []string
}

// loadRunProjection reads terminal truth from the canonical snapshot. Legacy
// and invalid snapshots deliberately fall back only to the compact index and
// carry an explicit degraded reason so absent heavy fields cannot look known.
func loadRunProjection(idx *sharedruns.Index, runID string) (runProjection, error) {
	rec, err := idx.Find(runID)
	if err != nil {
		return runProjection{}, err
	}
	projection := runProjection{record: rec}
	descriptors, descriptorErr := sharedruns.ReadDescriptorSnapshot(idx.ScenarioDir(), runID)
	projection.descriptorErr = descriptorErr
	if descriptorErr == nil {
		projection.descriptors = &descriptors
		if rec.DescriptorSnapshotSchemaVersion != 0 && rec.DescriptorSnapshotSchemaVersion != descriptors.SchemaVersion {
			projection.degraded = append(projection.degraded, "descriptor snapshot schema does not match the run index")
		}
		if rec.DescriptorSnapshotDigest != "" && rec.DescriptorSnapshotDigest != descriptors.Digest {
			projection.degraded = append(projection.degraded, "descriptor snapshot digest does not match the run index")
		}
	}
	if !isTerminalStatus(rec.Status) {
		return projection, nil
	}
	if descriptorErr != nil {
		if errors.Is(descriptorErr, sharedruns.ErrDescriptorSnapshotNotFound) {
			projection.degraded = append(projection.degraded, "legacy run predates descriptor snapshots")
		} else {
			projection.degraded = append(projection.degraded, "descriptor snapshot unavailable: "+descriptorErr.Error())
		}
	}
	snapshot, err := idx.ReadTerminalSnapshot(runID)
	if err != nil {
		projection.degraded = append(projection.degraded, terminalSnapshotDegradedReason(err))
		return projection, nil
	}
	var result orchestrator.SuiteExecutionResult
	if err := json.Unmarshal(snapshot.Result, &result); err != nil {
		projection.degraded = append(projection.degraded, "terminal snapshot result is corrupt: "+err.Error())
		return projection, nil
	}
	projection.record = snapshot.Run
	// Pins are mutable retention metadata and intentionally remain index-owned.
	projection.record.Pins = append([]sharedruns.PinRecord(nil), rec.Pins...)
	projection.result = &result
	projection.schemaVersion = snapshot.SchemaVersion
	return projection, nil
}

func terminalSnapshotDegradedReason(err error) string {
	if errors.Is(err, sharedruns.ErrSnapshotNotFound) {
		return "legacy run predates canonical terminal snapshots"
	}
	return "terminal snapshot unavailable: " + err.Error()
}

func phaseRecordMap(r sharedruns.RunRecord) map[string]sharedruns.PhaseRecord {
	m := make(map[string]sharedruns.PhaseRecord, len(r.Phases))
	for _, p := range r.Phases {
		if p.NonComparable {
			continue
		}
		m[p.Name] = p
	}
	return m
}

func phaseComparable(p sharedruns.PhaseRecord) bool {
	return !p.NonComparable
}

// isFailed reports whether a phase status counts as a failure for diffing.
// Empty (absent) is handled by the caller.
func isFailed(status string) bool {
	return status == "failed"
}

// comparePhases classifies each phase between baseline run A and current run B.
// Comparison is phase-level: a phase passing in A and failing in B is a
// regression; failing in B but absent in A is a new failure; failing in both is
// preexisting; failing in A and passing in B is cleared.
func comparePhases(a, b runProjection, phaseFilter string) *runspb.CompareRunsResponse {
	recordsA := phaseRecordMap(a.record)
	recordsB := phaseRecordMap(b.record)
	descriptorsA := descriptorSnapshotMap(a.descriptors)
	descriptorsB := descriptorSnapshotMap(b.descriptors)

	// Stable historical ordering: current run's captured descriptors first, then
	// baseline-only descriptors, with phase records as a legacy fallback.
	seen := make(map[string]bool)
	var order []string
	appendDescriptorOrder := func(snapshot *sharedruns.DescriptorSnapshot) {
		if snapshot == nil {
			return
		}
		for _, descriptor := range snapshot.Phases {
			if !seen[descriptor.Phase] {
				order = append(order, descriptor.Phase)
				seen[descriptor.Phase] = true
			}
		}
	}
	appendRecordOrder := func(record sharedruns.RunRecord) {
		for _, phase := range record.Phases {
			if phaseComparable(phase) && !seen[phase.Name] {
				order = append(order, phase.Name)
				seen[phase.Name] = true
			}
		}
	}
	appendDescriptorOrder(b.descriptors)
	appendDescriptorOrder(a.descriptors)
	appendRecordOrder(b.record)
	appendRecordOrder(a.record)

	out := make([]*runspb.PhaseDiff, 0, len(order))
	worst := verdictClean
	for _, name := range order {
		if phaseFilter != "" && name != phaseFilter {
			continue
		}
		recordA, okA := recordsA[name]
		recordB, okB := recordsB[name]
		descriptorA, hasDescriptorA := descriptorsA[name]
		descriptorB, hasDescriptorB := descriptorsB[name]
		sa, sb := recordA.Status, recordB.Status

		diff := &runspb.PhaseDiff{
			Phase: name, StatusA: sa, StatusB: sb,
			DescriptorA: toPhaseDescriptor(descriptorA),
			DescriptorB: toPhaseDescriptor(descriptorB),
		}
		reasons, forceNotComparable := phaseComparisonReasons(
			a, b, recordA, okA, recordB, okB, descriptorA, hasDescriptorA, descriptorB, hasDescriptorB,
		)
		diff.Reasons = reasons

		switch {
		case forceNotComparable:
			diff.Verdict = verdictNotComparable
		case !okB:
			// Phase only present in baseline; nothing to judge in current run.
			diff.Verdict = verdictNotComparable
		case isFailed(sb) && okA && !isFailed(sa) && sa != "":
			diff.Verdict = verdictRegression
			diff.Regressions = []string{name}
		case isFailed(sb) && !okA:
			diff.Verdict = verdictNewFailure
			diff.NewFailures = []string{name}
		case isFailed(sb) && isFailed(sa):
			diff.Verdict = verdictPreexisting
			diff.PreexistingFailures = []string{name}
		case !isFailed(sb) && isFailed(sa):
			diff.Verdict = verdictClean
			diff.ClearedFailures = []string{name}
		default:
			diff.Verdict = verdictClean
		}

		worst = worsen(worst, diff.Verdict)
		out = append(out, diff)
	}

	return &runspb.CompareRunsResponse{Phases: out, Verdict: worst}
}

func descriptorSnapshotMap(snapshot *sharedruns.DescriptorSnapshot) map[string]*sharedruns.PhaseDescriptorSnapshot {
	out := map[string]*sharedruns.PhaseDescriptorSnapshot{}
	if snapshot == nil {
		return out
	}
	for i := range snapshot.Phases {
		descriptor := &snapshot.Phases[i]
		out[descriptor.Phase] = descriptor
	}
	return out
}

func phaseComparisonReasons(
	a, b runProjection,
	recordA sharedruns.PhaseRecord,
	okA bool,
	recordB sharedruns.PhaseRecord,
	okB bool,
	descriptorA *sharedruns.PhaseDescriptorSnapshot,
	hasDescriptorA bool,
	descriptorB *sharedruns.PhaseDescriptorSnapshot,
	hasDescriptorB bool,
) ([]*runspb.PhaseComparisonReason, bool) {
	var reasons []*runspb.PhaseComparisonReason
	forceNotComparable := false
	add := func(code runspb.PhaseComparisonReasonCode, detail string, blocks bool) {
		reasons = append(reasons, &runspb.PhaseComparisonReason{Code: code, Detail: detail})
		forceNotComparable = forceNotComparable || blocks
	}
	if a.descriptorErr != nil || b.descriptorErr != nil {
		code := runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_LEGACY_METADATA_UNAVAILABLE
		if descriptorSnapshotIncompatible(a.descriptorErr) || descriptorSnapshotIncompatible(b.descriptorErr) {
			code = runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_INCOMPATIBLE_SCHEMA
		}
		add(code, "one or both runs lack a compatible captured descriptor snapshot", true)
	}
	if !hasDescriptorA && hasDescriptorB {
		add(runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_NEW_PHASE, "phase exists only in the current run catalog", false)
	}
	if hasDescriptorA && !hasDescriptorB {
		add(runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_RETIRED_PHASE, "phase exists only in the baseline run catalog", true)
	}
	if descriptorA != nil && descriptorA.Applicability.Status == "not_applicable" {
		add(runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_INAPPLICABLE, "phase was not applicable in the baseline run", true)
	}
	if descriptorB != nil && descriptorB.Applicability.Status == "not_applicable" {
		add(runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_INAPPLICABLE, "phase is not applicable in the current run", true)
	}
	if okA {
		appendExecutionComparisonReason(&reasons, &forceNotComparable, recordA, "baseline")
	}
	if okB {
		appendExecutionComparisonReason(&reasons, &forceNotComparable, recordB, "current")
	}
	return reasons, forceNotComparable
}

func appendExecutionComparisonReason(reasons *[]*runspb.PhaseComparisonReason, forceNotComparable *bool, record sharedruns.PhaseRecord, side string) {
	add := func(code runspb.PhaseComparisonReasonCode, detail string) {
		*reasons = append(*reasons, &runspb.PhaseComparisonReason{Code: code, Detail: detail})
		*forceNotComparable = true
	}
	switch record.Status {
	case "provider_unavailable":
		add(runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_PROVIDER_UNAVAILABLE, side+" provider was unavailable")
	case "skipped", "not_executable", "not_run":
		add(runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_SKIPPED, side+" phase did not execute")
	case "missing":
		if record.ArtifactBacked {
			add(runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_MISSING_ARTIFACT, side+" phase artifact is missing")
		} else {
			add(runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_SKIPPED, side+" phase result is missing")
		}
	}
}

func descriptorSnapshotIncompatible(err error) bool {
	return errors.Is(err, sharedruns.ErrUnsupportedDescriptorSnapshotVersion) || errors.Is(err, sharedruns.ErrInvalidDescriptorSnapshot)
}

// worsen returns the more severe of two verdicts for the overall summary.
// Severity (high→low): regression > not-comparable > new-failure > preexisting > clean.
func worsen(current, next string) string {
	return maxRank(current, next, map[string]int{
		verdictClean:         0,
		verdictPreexisting:   1,
		verdictNewFailure:    2,
		verdictNotComparable: 3,
		verdictRegression:    4,
	})
}

func maxRank(a, b string, rank map[string]int) string {
	if rank[b] > rank[a] {
		return b
	}
	return a
}
