package runs

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/providerreadiness"
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
	response := &runspb.CompareRunsResponse{SchemaVersion: 2, Behavior: "unknown", Coverage: "measured", Compatibility: "compatible", Provenance: "verified"}
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
		if !symmetricNeutralAbsence(recordA, okA, recordB, okB, descriptorA, descriptorB) {
			response.Behavior = aggregateBehavior(response.Behavior, diff.Behavior)
			response.Coverage = aggregateDimension(response.Coverage, diff.Coverage, "measured")
			response.Compatibility = aggregateDimension(response.Compatibility, diff.Compatibility, "compatible")
			response.Provenance = aggregateDimension(response.Provenance, diff.Provenance, "verified")
		}
		response.Diagnostics = append(response.Diagnostics, diff.Diagnostics...)
		out = append(out, diff)
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
	add := func(side, code, detail string) {
		diff.Diagnostics = append(diff.Diagnostics, &runspb.ComparisonDiagnostic{Side: side, Code: code, Detail: detail})
	}
	if a.descriptorErr != nil || b.descriptorErr != nil {
		diff.Compatibility = "legacy"
		add("comparison", "legacy_descriptor_metadata", "one or both runs lack a compatible descriptor snapshot")
	}
	if !hasDescriptorA || !hasDescriptorB {
		if hasDescriptorB {
			diff.Compatibility = "new"
			add("comparison", "new_phase", "phase exists only in the current catalog")
		} else {
			diff.Compatibility = "retired"
			add("comparison", "retired_phase", "phase exists only in the baseline catalog")
		}
	} else if descriptorA.ComparisonFingerprint == "" || descriptorB.ComparisonFingerprint == "" {
		diff.Compatibility = "legacy"
		add("comparison", "missing_comparison_fingerprint", "captured descriptor predates semantic comparison fingerprints")
	} else if descriptorA.ComparisonFingerprint != descriptorB.ComparisonFingerprint {
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
	if diff.Coverage == "measured" && (isUnmeasuredStatus(recordA.Status) || isUnmeasuredStatus(recordB.Status)) {
		diff.Coverage = "unmeasured"
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
	if response.Coverage != "measured" || response.Compatibility != "compatible" || (response.Provenance != "strict" && response.Provenance != "shared-scoped") {
		return verdictNotComparable
	}
	if response.Behavior == "preexisting" {
		return verdictPreexisting
	}
	return verdictClean
}

func symmetricNeutralAbsence(recordA sharedruns.PhaseRecord, okA bool, recordB sharedruns.PhaseRecord, okB bool, descriptorA, descriptorB *sharedruns.PhaseDescriptorSnapshot) bool {
	if descriptorInapplicable(descriptorA) && descriptorInapplicable(descriptorB) {
		return true
	}
	return okA && okB && recordA.Status == "provider_unavailable" && recordB.Status == "provider_unavailable" && providerUnavailableIsBestEffort(descriptorA) && providerUnavailableIsBestEffort(descriptorB)
}

func providerUnavailableIsBestEffort(descriptor *sharedruns.PhaseDescriptorSnapshot) bool {
	return descriptor != nil && descriptor.Policy.Unavailable == "skip_without_failing"
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
