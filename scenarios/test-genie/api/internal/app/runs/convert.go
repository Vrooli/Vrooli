package runs

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/providerreadiness"
	sharedruns "test-genie/internal/shared/runs"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
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

// withLeasePin preserves the existing RPC response shape while durable pin
// ownership moves out of the append-only run index. The lease itself remains
// authoritative; this projection is deliberately not persisted back to it.
func withLeasePin(run *runspb.RunInfo, lease sharedruns.PinLease) *runspb.RunInfo {
	return withLeasePins(run, []sharedruns.PinLease{lease})
}

func withLeasePins(run *runspb.RunInfo, leases []sharedruns.PinLease) *runspb.RunInfo {
	if run == nil {
		return nil
	}
	for _, lease := range leases {
		run.Pins = append(run.Pins, &runspb.PinInfo{
			PinnedBy: lease.Owner,
			PinnedAt: formatTime(lease.CreatedAt),
			Reason:   lease.Reason,
		})
	}
	return run
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
			info.PhasePresentation = phase.PhasePresentation
			info.FindingsSummary = phase.FindingsSummary
		}
		phases = append(phases, info)
	}
	return &runspb.RunInfo{
		RunId:           r.RunID,
		Target:          r.Scenario,
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
		Pins:                              nil,
		TreeDigest:                        r.TreeDigest,
		Preset:                            r.Preset,
		CaptureProfile:                    r.CaptureProfile,
		PlannedPhases:                     append([]string(nil), r.PlannedPhases...),
		PhaseSetDigest:                    r.PhaseSetDigest,
		DescriptorSnapshotSchemaVersion:   int32(descriptorSchemaVersion),
		DescriptorSnapshotDigest:          descriptorDigest,
		DescriptorSnapshot:                toDescriptorSnapshot(descriptors),
		ExecutionConfigurationFingerprint: executionConfigurationFingerprint(result),
		GateQuality:                       result != nil && result.GateQuality,
		EvidenceTier:                      evidenceTier(r, result),
		SourceScope:                       sourceScope(r, result),
		SourceStable:                      sourceStable(result),
		TargetRef:                         &commonv1.ValidationTarget{Kind: targetKind(r.TargetKind), Id: r.TargetID},
	}
}

func targetKind(kind string) commonv1.ValidationTargetKind {
	switch strings.TrimSpace(kind) {
	case "scenario":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO
	case "resource":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_RESOURCE
	case "asset":
		// Asset is the descriptor-era alias for a repository-owned resource.
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_RESOURCE
	case "tool":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TOOL
	case "safeguard":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD
	case "team":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TEAM
	case "package":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PACKAGE
	case "control-plane":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE
	case "docs":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_DOCS
	case "project":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT
	default:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_UNSPECIFIED
	}
}

func executionConfigurationFingerprint(result *orchestrator.SuiteExecutionResult) string {
	if result == nil {
		return ""
	}
	return result.ConfigurationFingerprint
}

func evidenceTier(record sharedruns.RunRecord, result *orchestrator.SuiteExecutionResult) string {
	if result == nil || record.TreeDigest == "" || (result.ConfigurationFingerprint == "" && !result.GateQuality) || !sourceStable(result) {
		return "degraded"
	}
	if result.GateQuality {
		return "strict"
	}
	return "shared-scoped"
}

// sourceStable keeps canonical pre-scoped terminal snapshots readable. Those
// historical strict runs had no explicit boolean, while new runs always stamp
// SourceFingerprint and can therefore represent an explicit false.
func sourceStable(result *orchestrator.SuiteExecutionResult) bool {
	return result != nil && (result.SourceStable || (result.GateQuality && result.SourceFingerprint == ""))
}

func sourceScope(record sharedruns.RunRecord, result *orchestrator.SuiteExecutionResult) string {
	if result != nil && result.SourceScope != "" {
		return result.SourceScope
	}
	if record.Scenario != "" {
		return "scenario:" + record.Scenario
	}
	return ""
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
		DocsPath:              descriptor.DocsPath,
		MaturityReference:     descriptor.MaturityReference,
		ApplicabilityDefault:  descriptor.ApplicabilityDefault,
		EvidenceKinds:         append([]string(nil), descriptor.EvidenceKinds...),
		Aliases:               append([]string(nil), descriptor.Aliases...),
		Supersedes:            append([]string(nil), descriptor.Supersedes...),
		ComparisonFingerprint: descriptor.ComparisonFingerprint,
		ComparisonMode:        descriptor.ComparisonMode,
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

// compareStablePreflightFailures compares the only durable evidence available
// when both suite attempts terminate before the first phase can run. This is
// intentionally narrower than treating empty phase sets as clean: both runs
// must carry stable, scoped source/configuration evidence and the same
// normalized lifecycle failure. A changed failure, missing provenance, or
// phase-filtered request falls through to the normal not-comparable logic.
func compareStablePreflightFailures(a, b runProjection) (*runspb.CompareRunsResponse, bool) {
	if a.result == nil || b.result == nil || len(a.result.Phases) != 0 || len(b.result.Phases) != 0 {
		return nil, false
	}
	if !sourceStable(a.result) || !sourceStable(b.result) {
		return nil, false
	}
	if a.result.SourceScope == "" || a.result.SourceScope != b.result.SourceScope ||
		a.result.SourceFingerprint == "" || a.result.SourceFingerprint != b.result.SourceFingerprint ||
		a.result.ConfigurationFingerprint == "" || a.result.ConfigurationFingerprint != b.result.ConfigurationFingerprint ||
		a.result.PhaseSetDigest == "" || a.result.PhaseSetDigest != b.result.PhaseSetDigest ||
		a.result.DescriptorSnapshotDigest == "" || a.result.DescriptorSnapshotDigest != b.result.DescriptorSnapshotDigest {
		return nil, false
	}
	failureA := stablePreflightFailureSummary(a.result.FailureReason)
	failureB := stablePreflightFailureSummary(b.result.FailureReason)
	if failureA == "" || failureA != failureB {
		return nil, false
	}

	response := &runspb.CompareRunsResponse{
		SchemaVersion: 2,
		Behavior:      "preexisting",
		Coverage:      "measured",
		Compatibility: "compatible",
		Provenance:    comparisonProvenance(a, b),
		Diagnostics: []*runspb.ComparisonDiagnostic{{
			Side: "comparison", Code: "stable_preflight_failure",
			Detail:      "both runs terminated before phases with the same stable preflight failure; no phase-level evidence was available",
			Remediation: "repair the shared scenario preflight failure before relying on phase-level coverage",
		}},
	}
	response.Verdict = legacyVerdict(response)
	return response, true
}

// stablePreflightFailureSummary removes run-specific logging context while
// retaining the lifecycle error that must remain unchanged across the two
// attempts. The final human-readable Error line is the stable contract emitted
// by vrooli; setup logs and timestamps are deliberately excluded.
func stablePreflightFailureSummary(reason string) string {
	lines := strings.Split(strings.TrimSpace(reason), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "Error:") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "Error:"))
		if before, _, ok := strings.Cut(line, " (log:"); ok {
			line = strings.TrimSpace(before)
		}
		return line
	}
	return ""
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
	projection.result = &result
	projection.schemaVersion = snapshot.SchemaVersion
	// Catalog publication is a terminal evidence guarantee, not an incidental
	// log warning. Preserve a failed publication as durable degraded state so a
	// completed run cannot imply it has an opaque evidence catalog.
	for _, warning := range result.Warnings {
		if strings.HasPrefix(strings.TrimSpace(warning), "artifact catalog unavailable:") {
			projection.degraded = append(projection.degraded, strings.TrimSpace(warning))
		}
	}
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
	response := &runspb.CompareRunsResponse{
		SchemaVersion: 2, Behavior: "unknown", Coverage: "measured", Compatibility: "compatible",
		Provenance: comparisonProvenance(a, b),
	}
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
		reasons, _ := phaseComparisonReasons(
			a, b, recordA, okA, recordB, okB, descriptorA, hasDescriptorA, descriptorB, hasDescriptorB,
		)
		diff.Reasons = reasons
		classifyPhaseComparison(diff, a, b, recordA, okA, recordB, okB, descriptorA, hasDescriptorA, descriptorB, hasDescriptorB)
		// A phase that was deliberately inapplicable on both sides is visible in
		// the phase detail but contributes no behavioral measurement. It must not
		// turn an otherwise comparable baseline into an unusable comparison.
		if !bestEffortProviderOutageIsNeutral(a, b, recordA, okA, recordB, okB, descriptorA, descriptorB) && !symmetricNeutralAbsence(a, b, recordA, okA, recordB, okB, descriptorA, descriptorB) && !providerReadinessGapNeutral(a, b, recordA, okA, recordB, okB, descriptorA, descriptorB) && !additivePhaseAdditionNeutral(diff, descriptorA, descriptorB) {
			response.Behavior = aggregateBehavior(response.Behavior, diff.Behavior)
			response.Coverage = aggregateDimension(response.Coverage, diff.Coverage, "measured")
			response.Compatibility = aggregateDimension(response.Compatibility, diff.Compatibility, "compatible")
			response.Provenance = aggregateDimension(response.Provenance, diff.Provenance, "verified")
		}
		response.Diagnostics = append(response.Diagnostics, diff.Diagnostics...)
		out = append(out, diff)
	}
	// An empty phase set is not evidence of a clean comparison. The only
	// supported zero-phase comparable case is handled before this function by
	// compareStablePreflightFailures; all other empty requests remain
	// explicitly unmeasured.
	if len(out) == 0 {
		response.Behavior = "unknown"
		response.Coverage = "unmeasured"
	}
	// A run containing only neutral best-effort coverage gaps has no measured
	// behavioral delta, but it is still safe to say no regression was observed.
	// Keep the phase diagnostics visible without turning absence of evidence into
	// a global not-comparable verdict.
	if response.Behavior == "unknown" && len(out) > 0 {
		response.Behavior = "clean"
	}
	response.Phases = out
	response.Verdict = legacyVerdict(response)
	return response
}

// classifyPhaseComparison keeps behavior, whether it was measured, validator
// compatibility, and source quality independent. A consumer may gate any of
// these dimensions, but no dimension is allowed to masquerade as another.
func classifyPhaseComparison(diff *runspb.PhaseDiff, a, b runProjection, recordA sharedruns.PhaseRecord, okA bool, recordB sharedruns.PhaseRecord, okB bool, descriptorA *sharedruns.PhaseDescriptorSnapshot, hasDescriptorA bool, descriptorB *sharedruns.PhaseDescriptorSnapshot, hasDescriptorB bool) {
	diff.Behavior, diff.Coverage, diff.Compatibility, diff.Provenance = "unknown", "measured", "compatible", comparisonProvenance(a, b)
	legacyIndex := legacyIndexComparable(a, b)
	add := func(side, code, detail string) {
		diff.Diagnostics = append(diff.Diagnostics, &runspb.ComparisonDiagnostic{Side: side, Code: code, Detail: detail})
	}
	if (a.descriptorErr != nil || b.descriptorErr != nil) && !legacyIndex {
		diff.Compatibility = "legacy"
		add("comparison", "legacy_descriptor_metadata", "one or both runs lack a compatible descriptor snapshot")
	}
	if !legacyIndex && (!hasDescriptorA || !hasDescriptorB) {
		if hasDescriptorB {
			diff.Compatibility = "new"
			add("comparison", "new_phase", "phase exists only in the current catalog")
		} else {
			diff.Compatibility = "retired"
			add("comparison", "retired_phase", "phase exists only in the baseline catalog")
		}
	} else if !legacyIndex && (descriptorA.ComparisonFingerprint == "" || descriptorB.ComparisonFingerprint == "") {
		diff.Compatibility = "legacy"
		add("comparison", "missing_comparison_fingerprint", "captured descriptor predates semantic comparison fingerprints")
	} else if !legacyIndex && descriptorA.ComparisonFingerprint != descriptorB.ComparisonFingerprint {
		diff.Compatibility = descriptorB.ComparisonMode
		if diff.Compatibility == "" {
			diff.Compatibility = "changed-unreviewed"
		}
		add("comparison", "validator_contract_changed", "same phase key has a different validation semantic fingerprint")
	}
	if descriptorInapplicable(descriptorA) || descriptorInapplicable(descriptorB) {
		diff.Coverage = "unmeasured"
		add("comparison", "inapplicable", "phase did not apply to one or both captured targets")
	}
	inapplicable := descriptorInapplicable(descriptorA) || descriptorInapplicable(descriptorB)
	if !okA && !inapplicable {
		if hasDescriptorB && !hasDescriptorA {
			diff.Coverage = "current-only"
		} else {
			diff.Coverage = "baseline-missing"
		}
		add("baseline", "phase_record_missing", "baseline has no phase result")
	}
	if !okB && !inapplicable {
		diff.Coverage = "current-missing"
		add("current", "phase_record_missing", "current run has no phase result")
	}
	if okA {
		appendCoverageDiagnostic(diff, "baseline", recordA)
		appendReadinessDiagnostic(diff, "baseline", a.result, diff.Phase)
	}
	if okB {
		appendCoverageDiagnostic(diff, "current", recordB)
		appendReadinessDiagnostic(diff, "current", b.result, diff.Phase)
	}
	if providerReadinessGapNeutral(a, b, recordA, okA, recordB, okB, descriptorA, descriptorB) {
		diff.Coverage, diff.Behavior, diff.Verdict = "measured", "preexisting", verdictPreexisting
		add("comparison", "provider_gap_preexisting", "the same stable provider-readiness gap was present, or an advisory provider was unavailable, on both comparable runs")
		return
	}
	if bestEffortProviderOutageResolved(a, b, recordA, okA, recordB, okB, descriptorA, descriptorB) {
		diff.Coverage, diff.Behavior, diff.Verdict = "current-only", "unknown", verdictClean
		add("comparison", "provider_recovered", "baseline best-effort provider outage is resolved; current result is additional evidence, not a baseline comparison")
		return
	}
	if diff.Coverage == "measured" && (isUnmeasuredStatus(recordA.Status) || isUnmeasuredStatus(recordB.Status)) {
		diff.Coverage = "unmeasured"
	}
	if additivePhasePassedWithoutBaseline(diff, recordB, okB, hasDescriptorA, hasDescriptorB, descriptorB) {
		return
	}
	if (diff.Coverage != "measured" && diff.Coverage != "current-only") || (diff.Compatibility != "compatible" && diff.Compatibility != "new") {
		diff.Verdict = verdictNotComparable
		return
	}
	if (okA && !isMeasuredTerminalStatus(recordA.Status)) || (okB && !isMeasuredTerminalStatus(recordB.Status)) {
		diff.Behavior, diff.Verdict = "unknown", verdictNotComparable
		add("comparison", "unknown_phase_status", "one or both phase results use an unrecognized terminal status")
		return
	}
	switch {
	case isFailed(recordB.Status) && okA && !isFailed(recordA.Status) && recordA.Status != "":
		diff.Behavior, diff.Verdict, diff.Regressions = "regression", verdictRegression, []string{diff.Phase}
	case isFailed(recordB.Status) && !okA:
		diff.Behavior, diff.Verdict, diff.NewFailures = "new-failure", verdictNewFailure, []string{diff.Phase}
	case isFailed(recordB.Status) && isFailed(recordA.Status):
		diff.Behavior, diff.Verdict, diff.PreexistingFailures = "preexisting", verdictPreexisting, []string{diff.Phase}
	case !isFailed(recordB.Status) && isFailed(recordA.Status):
		diff.Behavior, diff.Verdict, diff.ClearedFailures = "cleared", verdictClean, []string{diff.Phase}
	default:
		diff.Behavior, diff.Verdict = "clean", verdictClean
	}
}

// additivePhasePassedWithoutBaseline is the explicit escape hatch for a phase
// introduced after an immutable baseline was captured. It is deliberately
// narrow: only a current-only additive phase with a measured pass is neutral.
// A failed additive phase continues through the normal new-failure path, and
// an absent/unknown result remains incomparable.
func additivePhasePassedWithoutBaseline(diff *runspb.PhaseDiff, recordB sharedruns.PhaseRecord, okB, hasDescriptorA, hasDescriptorB bool, descriptorB *sharedruns.PhaseDescriptorSnapshot) bool {
	if hasDescriptorA || !hasDescriptorB || descriptorB == nil || descriptorB.ComparisonMode != "additive" || !okB || recordB.Status != "passed" {
		return false
	}
	diff.Coverage = "current-only"
	diff.Compatibility = "additive"
	diff.Behavior = "clean"
	diff.Verdict = verdictClean
	diff.Diagnostics = append(diff.Diagnostics, &runspb.ComparisonDiagnostic{
		Side: "comparison", Code: "additive_phase",
		Detail: "phase was introduced after the baseline and passed as an explicitly additive validator; current-only evidence is retained without claiming before coverage",
	})
	return true
}

func additivePhaseAdditionNeutral(diff *runspb.PhaseDiff, descriptorA, descriptorB *sharedruns.PhaseDescriptorSnapshot) bool {
	return descriptorA == nil && descriptorB != nil && descriptorB.ComparisonMode == "additive" && diff.Behavior == "clean" && diff.Coverage == "current-only" && diff.Compatibility == "additive"
}

func appendReadinessDiagnostic(diff *runspb.PhaseDiff, side string, result *orchestrator.SuiteExecutionResult, phase string) {
	if result == nil {
		return
	}
	for _, outcome := range result.ProviderReadiness {
		if outcome.Phase != phase || outcome.Ready {
			continue
		}
		lifecycle := ""
		if outcome.Started {
			lifecycle = "started"
		}
		if outcome.Restarted {
			lifecycle = "restarted"
		}
		observations := []string{outcome.ErrorString()}
		diff.Diagnostics = append(diff.Diagnostics, &runspb.ComparisonDiagnostic{
			Side: side, Code: "provider_readiness", Detail: outcome.Message,
			Classification: string(outcome.Status), LifecycleAction: lifecycle,
			Remediation: providerreadiness.Remediation(outcome), Observations: observations,
		})
	}
}

func isUnmeasuredStatus(status string) bool {
	return status == "provider_unavailable" || status == "skipped" || status == "not_executable" || status == "not_run" || status == "missing"
}

// isMeasuredTerminalStatus is deliberately narrow. New phase result states
// must be classified explicitly before they can be trusted as behavioral
// evidence; treating an unfamiliar state as a pass would hide regressions.
func isMeasuredTerminalStatus(status string) bool {
	return status == "passed" || status == "failed"
}

func appendCoverageDiagnostic(diff *runspb.PhaseDiff, side string, record sharedruns.PhaseRecord) {
	if !isUnmeasuredStatus(record.Status) {
		return
	}
	code, detail := "phase_not_measured", side+" phase did not execute"
	if record.Status == "provider_unavailable" {
		code, detail = "provider_unavailable", side+" provider was unavailable"
	}
	if record.Status == "missing" && record.ArtifactBacked {
		code, detail = "missing_artifact", side+" phase artifact is missing"
	}
	diff.Diagnostics = append(diff.Diagnostics, &runspb.ComparisonDiagnostic{Side: side, Code: code, Detail: detail, Remediation: "restore the provider or evidence, then rerun this phase"})
}

func comparisonProvenance(a, b runProjection) string {
	if legacyIndexComparable(a, b) {
		return "legacy-index"
	}
	tier := "strict"
	for _, projection := range []runProjection{a, b} {
		if projection.result == nil {
			return "legacy"
		}
		if !sourceStable(projection.result) {
			return "volatile"
		}
		record := projection.record
		if record.TreeDigest == "" || (projection.result.ConfigurationFingerprint == "" && !projection.result.GateQuality) {
			return "legacy"
		}
		if !projection.result.GateQuality {
			tier = "shared-scoped"
		}
	}
	return tier
}

// legacyIndexComparable permits a historical run that predates canonical
// terminal snapshots to remain useful as an anchor when its compact index
// still carries a stable tree digest, matching phase-set digest, and phase
// records on both sides. The result remains explicitly marked legacy-index;
// it never participates in source-keyed reuse or cache eligibility.
func legacyIndexComparable(a, b runProjection) bool {
	if a.descriptorErr == nil && b.descriptorErr == nil {
		return false
	}
	if a.record.TreeDigest == "" || b.record.TreeDigest == "" {
		return false
	}
	if a.record.PhaseSetDigest == "" || a.record.PhaseSetDigest != b.record.PhaseSetDigest {
		return false
	}
	return len(a.record.Phases) > 0 && len(b.record.Phases) > 0
}

func aggregateBehavior(current, next string) string {
	rank := map[string]int{"unknown": 0, "clean": 1, "cleared": 2, "preexisting": 3, "new-failure": 4, "regression": 5}
	if rank[next] > rank[current] {
		return next
	}
	return current
}

func aggregateDimension(current, next, healthy string) string {
	if current == healthy {
		return next
	}
	if next == healthy {
		return current
	}
	if current == next {
		return current
	}
	return "mixed"
}

func legacyVerdict(response *runspb.CompareRunsResponse) string {
	switch response.Behavior {
	case "regression":
		return verdictRegression
	case "new-failure":
		return verdictNewFailure
	}
	if response.Coverage != "measured" || response.Compatibility != "compatible" || (response.Provenance != "strict" && response.Provenance != "shared-scoped" && response.Provenance != "legacy-index") {
		return verdictNotComparable
	}
	if response.Behavior == "preexisting" {
		return verdictPreexisting
	}
	return verdictClean
}

func symmetricNeutralAbsence(a, b runProjection, recordA sharedruns.PhaseRecord, okA bool, recordB sharedruns.PhaseRecord, okB bool, descriptorA, descriptorB *sharedruns.PhaseDescriptorSnapshot) bool {
	if descriptorInapplicable(descriptorA) && descriptorInapplicable(descriptorB) {
		return true
	}
	// Catalog evolution can add or retire a conditional phase that is explicitly
	// inapplicable for the only run that knows about it. Because an inapplicable
	// phase has no execution record, this is also a matched absence of behavioral
	// evidence rather than an asymmetric exercised surface. Keep its diagnostic
	// visible, but do not poison the aggregate comparison.
	if descriptorA == nil && descriptorInapplicable(descriptorB) && !okB {
		return true
	}
	if descriptorB == nil && descriptorInapplicable(descriptorA) && !okA {
		return true
	}
	return okA && okB && recordA.Status == "provider_unavailable" && recordB.Status == "provider_unavailable" && providerUnavailableIsBestEffort(descriptorA) && providerUnavailableIsBestEffort(descriptorB)
}

// providerReadinessGapNeutral preserves explicit provider-readiness evidence
// without letting an external provider outage invalidate otherwise measured
// phase comparisons. It is deliberately limited to absent phase records,
// stable shared source/configuration provenance, and either an advisory phase
// whose provider was skipped best-effort or identical unavailable outcomes.
func providerReadinessGapNeutral(a, b runProjection, recordA sharedruns.PhaseRecord, okA bool, recordB sharedruns.PhaseRecord, okB bool, descriptorA, descriptorB *sharedruns.PhaseDescriptorSnapshot) bool {
	if a.result == nil || b.result == nil || !sourceStable(a.result) || !sourceStable(b.result) {
		return false
	}
	// The source fingerprint is expected to differ when comparing a baseline
	// against a changed implementation. It identifies the content being
	// measured, not the comparability of an external provider-readiness gap.
	// Keep the target scope stable so unrelated runs cannot inherit this
	// neutralization. The full-run configuration fingerprint is intentionally not
	// compared: it changes when an unrelated phase descriptor or source-level
	// configuration changes, while the provider gap itself remains phase-local
	// and is proven below by the captured descriptor and readiness identity.
	if a.result.SourceScope == "" || a.result.SourceScope != b.result.SourceScope {
		return false
	}
	phase := ""
	if descriptorB != nil {
		phase = descriptorB.Phase
	} else if descriptorA != nil {
		phase = descriptorA.Phase
	}
	if phase == "" {
		return false
	}
	readinessA := readinessForPhase(a.result, phase)
	readinessB := readinessForPhase(b.result, phase)
	if len(readinessA) == 0 && len(readinessB) == 0 {
		return false
	}
	if advisoryPhase(descriptorA) && advisoryPhase(descriptorB) && hasSkippedBestEffort(readinessB) && (!okB || !isMeasuredTerminalStatus(recordB.Status)) {
		return true
	}
	if (okA && isMeasuredTerminalStatus(recordA.Status)) || (okB && isMeasuredTerminalStatus(recordB.Status)) {
		return false
	}
	if len(readinessA) != len(readinessB) {
		return false
	}
	for i := range readinessA {
		if !equivalentProviderReadinessOutcome(readinessA[i], readinessB[i]) {
			return false
		}
	}
	return true
}

// equivalentProviderReadinessOutcome compares the stable identity of a
// provider gap while ignoring lifecycle-wrapper noise. Lifecycle commands
// include timestamps, run ids, pids, and log paths in their error text; those
// values change on every attempt even when the underlying provider failure is
// identical. The final Error line is the stable lifecycle contract and keeps
// a changed failure distinguishable from the same failure repeated.
func equivalentProviderReadinessOutcome(a, b providerreadiness.Outcome) bool {
	return a.Phase == b.Phase &&
		a.ProviderScenario == b.ProviderScenario &&
		a.Status == b.Status &&
		a.Ready == b.Ready &&
		a.BestEffort == b.BestEffort &&
		a.Started == b.Started &&
		a.Restarted == b.Restarted &&
		stableProviderReadinessError(a.ErrorString()) == stableProviderReadinessError(b.ErrorString())
}

func stableProviderReadinessError(raw string) string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "Error:") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "Error:"))
		if before, _, ok := strings.Cut(line, " (log:"); ok {
			line = strings.TrimSpace(before)
		}
		return line
	}
	return strings.TrimSpace(raw)
}

func readinessForPhase(result *orchestrator.SuiteExecutionResult, phase string) []providerreadiness.Outcome {
	if result == nil {
		return nil
	}
	var outcomes []providerreadiness.Outcome
	for _, outcome := range result.ProviderReadiness {
		if outcome.Phase == phase {
			outcomes = append(outcomes, outcome)
		}
	}
	return outcomes
}

func hasSkippedBestEffort(outcomes []providerreadiness.Outcome) bool {
	for _, outcome := range outcomes {
		if outcome.SkipsWithoutFailure() {
			return true
		}
	}
	return false
}

func advisoryPhase(descriptor *sharedruns.PhaseDescriptorSnapshot) bool {
	return descriptor != nil && (descriptor.Policy.ResultGating == "advisory" || descriptor.Policy.Unavailable == "advisory")
}

func providerUnavailableIsBestEffort(descriptor *sharedruns.PhaseDescriptorSnapshot) bool {
	return descriptor != nil && descriptor.Policy.Unavailable == "skip_without_failing"
}

func bestEffortProviderOutageResolved(a, b runProjection, recordA sharedruns.PhaseRecord, okA bool, recordB sharedruns.PhaseRecord, okB bool, descriptorA, descriptorB *sharedruns.PhaseDescriptorSnapshot) bool {
	if !okA || !okB || !isMeasuredTerminalStatus(recordB.Status) || !nonBlockingProviderPhase(descriptorA) || !nonBlockingProviderPhase(descriptorB) {
		return false
	}
	if recordA.Status == "provider_unavailable" {
		return true
	}
	if recordA.Status != "skipped" {
		return false
	}
	if a.result == nil || b.result == nil || !sourceStable(a.result) || !sourceStable(b.result) || a.result.SourceScope == "" || a.result.SourceScope != b.result.SourceScope {
		return false
	}
	phase := ""
	if descriptorB != nil {
		phase = descriptorB.Phase
	} else if descriptorA != nil {
		phase = descriptorA.Phase
	}
	return phase != "" && hasSkippedBestEffort(readinessForPhase(a.result, phase))
}

func bestEffortProviderOutageIsNeutral(a, b runProjection, recordA sharedruns.PhaseRecord, okA bool, recordB sharedruns.PhaseRecord, okB bool, descriptorA, descriptorB *sharedruns.PhaseDescriptorSnapshot) bool {
	return bestEffortProviderOutageResolved(a, b, recordA, okA, recordB, okB, descriptorA, descriptorB)
}

func nonBlockingProviderPhase(descriptor *sharedruns.PhaseDescriptorSnapshot) bool {
	return descriptor != nil && (providerUnavailableIsBestEffort(descriptor) || advisoryPhase(descriptor))
}

// descriptorInapplicable reports whether a captured phase descriptor recorded
// the phase as not applicable for its run.
func descriptorInapplicable(d *sharedruns.PhaseDescriptorSnapshot) bool {
	return d != nil && d.Applicability.Status == "not_applicable"
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
	legacyIndex := legacyIndexComparable(a, b)
	add := func(code runspb.PhaseComparisonReasonCode, detail string, blocks bool) {
		reasons = append(reasons, &runspb.PhaseComparisonReason{Code: code, Detail: detail})
		forceNotComparable = forceNotComparable || blocks
	}
	if (a.descriptorErr != nil || b.descriptorErr != nil) && !legacyIndex {
		code := runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_LEGACY_METADATA_UNAVAILABLE
		if descriptorSnapshotIncompatible(a.descriptorErr) || descriptorSnapshotIncompatible(b.descriptorErr) {
			code = runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_INCOMPATIBLE_SCHEMA
		}
		add(code, "one or both runs lack a compatible captured descriptor snapshot", true)
	}
	if !legacyIndex && !hasDescriptorA && hasDescriptorB {
		add(runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_NEW_PHASE, "phase exists only in the current run catalog", false)
	}
	if !legacyIndex && hasDescriptorA && !hasDescriptorB {
		add(runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_RETIRED_PHASE, "phase exists only in the baseline run catalog", true)
	}
	if descriptorInapplicable(descriptorA) {
		add(runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_INAPPLICABLE, "phase was not applicable in the baseline run", true)
	}
	if descriptorInapplicable(descriptorB) {
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
