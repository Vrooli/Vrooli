package optimization

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Start pins the target, candidate set and evaluation denominator before any
// owner-side effect. Reusing a run_id with a different request is rejected.
func (e *Engine) Start(ctx context.Context, request Request) (Record, error) {
	if e.Store == nil {
		return Record{}, fmt.Errorf("optimization store is required")
	}
	if err := validateRequest(request); err != nil {
		return Record{}, err
	}
	adapter, ok := e.Adapters[request.Target]
	if !ok || adapter == nil || adapter.Target() != request.Target {
		return Record{}, fmt.Errorf("target %q has no matching owner adapter: %w", request.Target, ErrNoAdapter)
	}
	if err := adapter.Validate(request); err != nil {
		return Record{}, fmt.Errorf("target %q rejected request (%v): %w", request.Target, err, ErrInvalidRequest)
	}
	pinned, err := cloneRequest(request)
	if err != nil {
		return Record{}, fmt.Errorf("pin optimization request: %w", err)
	}
	initial := Record{Request: pinned, Stage: StagePreflightRights, Evidence: []Evidence{}, StageAttempts: map[Stage]int{}, UpdatedAt: e.now()}
	created, err := e.Store.Create(ctx, initial)
	if err == nil {
		return created, nil
	}
	prior, loadErr := e.Store.Load(ctx, request.RunID)
	if loadErr != nil {
		return Record{}, err
	}
	if !reflect.DeepEqual(prior.Request, pinned) {
		return Record{}, fmt.Errorf("run_id %q is already pinned to a different request: %w", request.RunID, ErrInvalidRequest)
	}
	return prior, nil
}

// Advance performs no more than one idempotent owner operation and persists
// its receipt before returning. A caller may repeat it after a restart; the
// owner receives the same run/stage identity and must replay its receipt.
func (e *Engine) Advance(ctx context.Context, runID string) (Record, error) {
	// A manager normally has one worker, but callers can also advance runs
	// directly (for example, during recovery). Serialize an in-process run so
	// two callers cannot launch the same expensive owner stage concurrently.
	// Stable stage IDs remain the cross-restart idempotency boundary.
	unlock, err := e.locks.lock(ctx, runID)
	if err != nil {
		return Record{}, err
	}
	defer unlock()

	if e.Store == nil {
		return Record{}, fmt.Errorf("optimization store is required")
	}
	record, err := e.Store.Load(ctx, runID)
	if err != nil {
		return Record{}, err
	}
	if record.Stage == StageComplete || record.Stage == StageRefused {
		return record, nil
	}
	if !record.RetryAt.IsZero() && e.now().Before(record.RetryAt) {
		return record, ErrRetryNotReady
	}
	adapter := e.Adapters[record.Request.Target]
	if adapter == nil || adapter.Target() != record.Request.Target {
		return Record{}, fmt.Errorf("target %q has no matching owner adapter: %w", record.Request.Target, ErrNoAdapter)
	}
	stage := record.Stage
	if record.StageAttempts == nil {
		record.StageAttempts = map[Stage]int{}
	}
	record.StageAttempts[stage]++
	attempt := record.StageAttempts[stage]
	record.LastError = ""
	record.RetryAt = time.Time{}
	priorRevision := record.Revision
	record.Revision++
	record.UpdatedAt = e.now()
	record, err = e.Store.CompareAndSwap(ctx, priorRevision, record)
	if err != nil {
		return Record{}, err
	}

	if stage == StagePreflightDisk && !hasFreshAllow(record, StagePreflightRights, e.now()) {
		return e.refuse(ctx, record, "pre-acquisition disk admission requires current allow rights proof")
	}
	if stage == StageAcquire && (!hasFreshAllow(record, StagePreflightRights, e.now()) || !hasFreshAllow(record, StagePreflightDisk, e.now())) {
		return e.refuse(ctx, record, "acquisition requires current preflight rights and disk proofs")
	}
	if stage == StageEvaluate && (!hasReceipt(record, StageAcquire) || !hasFreshAllow(record, StagePreflightRights, e.now()) || !hasFreshAllow(record, StagePreflightDisk, e.now())) {
		return e.refuse(ctx, record, "evaluation requires acquisition and current preflight rights and disk proofs")
	}
	if stage == StageRights && (!hasReceipt(record, StageAcquire) || !hasValidEvaluation(record)) {
		return e.refuse(ctx, record, "complete fixed-suite evaluation receipt is required before rights checks")
	}
	if stage == StageDisk && !hasFreshAllow(record, StageRights, e.now()) {
		return e.refuse(ctx, record, "current allow rights proof is required before disk admission")
	}
	var evidence Evidence
	switch stage {
	case StagePreflightRights:
		evidence, err = adapter.CheckRights(ctx, runID, stage, record.Request, cloneEvidence(record.Evidence))
	case StagePreflightDisk:
		evidence, err = adapter.CheckDisk(ctx, runID, stage, record.Request, cloneEvidence(record.Evidence))
	case StageAcquire:
		evidence, err = adapter.Acquire(ctx, runID, record.Request)
	case StageEvaluate:
		evidence, err = adapter.Evaluate(ctx, runID, record.Request, cloneEvidence(record.Evidence))
	case StageRights:
		evidence, err = adapter.CheckRights(ctx, runID, stage, record.Request, cloneEvidence(record.Evidence))
	case StageDisk:
		evidence, err = adapter.CheckDisk(ctx, runID, stage, record.Request, cloneEvidence(record.Evidence))
	case StagePromote:
		if !hasFreshAllow(record, StageRights, e.now()) || !hasFreshAllow(record, StageDisk, e.now()) {
			return e.refuse(ctx, record, "rights and disk proofs must both be current and allow before promotion")
		}
		evidence, err = adapter.Promote(ctx, runID, record.Request, cloneEvidence(record.Evidence))
	case StageCleanup:
		if !hasDisposition(record, StagePromote, "promoted") {
			return e.refuse(ctx, record, "cleanup requires a recorded promotion receipt")
		}
		if !hasFreshAllow(record, StageRights, e.now()) || !hasFreshAllow(record, StageDisk, e.now()) {
			return e.refuse(ctx, record, "current rights and disk proofs are required through predecessor/loser cleanup")
		}
		evidence, err = adapter.Cleanup(ctx, runID, record.Request, cloneEvidence(record.Evidence))
	default:
		return Record{}, fmt.Errorf("unsupported optimization stage %q", record.Stage)
	}
	if err != nil {
		priorRevision := record.Revision
		record.LastError = err.Error()
		record.RetryAt = e.now().Add(e.retryDelay(attempt))
		record.Revision++
		record.UpdatedAt = e.now()
		saved, saveErr := e.Store.CompareAndSwap(ctx, priorRevision, record)
		if saveErr != nil {
			return Record{}, fmt.Errorf("persist %s retry state after %v: %w", stage, err, saveErr)
		}
		return saved, err
	}
	if evidence.Stage != stage || strings.TrimSpace(evidence.ReceiptID) == "" {
		return e.refuse(ctx, record, fmt.Sprintf("owner returned malformed %s receipt", stage))
	}
	if (isRightsStage(stage) || isDiskStage(stage)) && !wellFormedProof(evidence.Proof, e.now()) {
		return e.refuseWithEvidence(ctx, record, evidence, fmt.Sprintf("owner returned invalid %s proof", stage))
	}
	if isDiskStage(stage) && !wellFormedDiskProof(evidence.Proof, e.now()) {
		return e.refuseWithEvidence(ctx, record, evidence, fmt.Sprintf("owner returned invalid %s capacity proof", stage))
	}
	record.Evidence = append(record.Evidence, evidence)
	if stage == StageAcquire && !acquisitionCovers(record.Request, evidence) {
		return e.refuse(ctx, record, "owner acquisition receipt does not match every pinned candidate artifact")
	}
	if isRightsStage(stage) || isDiskStage(stage) {
		if !strings.EqualFold(evidence.Proof.Decision, "allow") || (evidence.Proof.RequiredBytes > 0 && evidence.Proof.AvailableBytes < evidence.Proof.RequiredBytes) {
			return e.persistRefusal(ctx, record, fmt.Sprintf("owner %s gate refused the pinned request", stage))
		}
		wantDigest := SubjectDigest(record.Request)
		if stage == StageRights || stage == StageDisk {
			wantDigest = AcquiredSubjectDigest(record.Request, record.Evidence)
		}
		if evidence.Proof.SubjectDigest != wantDigest {
			return e.persistRefusal(ctx, record, fmt.Sprintf("owner %s proof does not match the pinned request", stage))
		}
	}
	if stage == StageEvaluate {
		if !evaluationCovers(record.Request, evidence) {
			return e.refuse(ctx, record, "evaluation receipt does not prove the pinned suite and case denominator")
		}
	}
	if stage == StagePromote {
		record.WinnerID = candidateID(evidence.Details["winner_id"])
		candidate := findCandidate(record.Request.Candidates, record.WinnerID)
		if !strings.EqualFold(evidence.Disposition, "promoted") || candidate == nil || evidence.Details["artifact_digest"] != acquiredDigests(record.Evidence)[record.WinnerID] {
			return e.persistRefusal(ctx, record, "promotion receipt does not name the pinned candidate artifact")
		}
	}
	if stage == StageCleanup && !cleanupCovers(record.Request, record.WinnerID, evidence) {
		return e.persistRefusal(ctx, record, "cleanup receipt does not prove removal of every losing candidate and predecessor")
	}
	record.Stage = nextStage(stage)
	record.LastError = ""
	record.RetryAt = time.Time{}
	record.Revision++
	record.UpdatedAt = e.now()
	return e.Store.CompareAndSwap(ctx, record.Revision-1, record)
}

func (e *Engine) Run(ctx context.Context, request Request) (Record, error) {
	record, err := e.Start(ctx, request)
	if err != nil {
		return Record{}, err
	}
	casRetries := 0
	for record.Stage != StageComplete && record.Stage != StageRefused {
		record, err = e.Advance(ctx, request.RunID)
		if errors.Is(err, ErrRevision) && casRetries < 8 {
			casRetries++
			record, err = e.Store.Load(ctx, request.RunID)
			if err == nil {
				continue
			}
		}
		if err != nil {
			return record, err
		}
	}
	return record, nil
}

func (e *Engine) refuse(ctx context.Context, record Record, reason string) (Record, error) {
	record.Stage = StageRefused
	record.Failure = reason
	record.Revision++
	record.UpdatedAt = e.now()
	return e.Store.CompareAndSwap(ctx, record.Revision-1, record)
}

func (e *Engine) refuseWithEvidence(ctx context.Context, record Record, evidence Evidence, reason string) (Record, error) {
	if evidence.ReceiptID != "" {
		record.Evidence = append(record.Evidence, evidence)
	}
	return e.persistRefusal(ctx, record, reason)
}

func (e *Engine) persistRefusal(ctx context.Context, record Record, reason string) (Record, error) {
	prior := record.Revision
	record.Stage = StageRefused
	record.Failure = reason
	record.Revision++
	record.UpdatedAt = e.now()
	return e.Store.CompareAndSwap(ctx, prior, record)
}

func (e *Engine) retryDelay(attempt int) time.Duration {
	if e.RetryDelay != nil {
		return e.RetryDelay(attempt)
	}
	switch attempt {
	case 1:
		return 5 * time.Second
	case 2:
		return 15 * time.Second
	default:
		return 30 * time.Second
	}
}

// Fail makes a repeatedly failing stage terminal without repeating an owner
// operation. The manager calls it only after the bounded retry policy expires.
func (e *Engine) Fail(ctx context.Context, runID, reason string) (Record, error) {
	record, err := e.Store.Load(ctx, runID)
	if err != nil {
		return Record{}, err
	}
	if record.Stage == StageComplete || record.Stage == StageRefused {
		return record, nil
	}
	return e.refuse(ctx, record, reason)
}

func nextStage(stage Stage) Stage {
	switch stage {
	case StagePreflightRights:
		return StagePreflightDisk
	case StagePreflightDisk:
		return StageAcquire
	case StageAcquire:
		return StageEvaluate
	case StageEvaluate:
		return StageRights
	case StageRights:
		return StageDisk
	case StageDisk:
		return StagePromote
	case StagePromote:
		return StageCleanup
	case StageCleanup:
		return StageComplete
	default:
		return StageRefused
	}
}

func isRightsStage(stage Stage) bool { return stage == StagePreflightRights || stage == StageRights }
func isDiskStage(stage Stage) bool   { return stage == StagePreflightDisk || stage == StageDisk }

func hasReceipt(record Record, stage Stage) bool {
	for i := len(record.Evidence) - 1; i >= 0; i-- {
		if record.Evidence[i].Stage == stage && strings.TrimSpace(record.Evidence[i].ReceiptID) != "" {
			return true
		}
	}
	return false
}

func hasValidEvaluation(record Record) bool {
	for i := len(record.Evidence) - 1; i >= 0; i-- {
		if record.Evidence[i].Stage == StageEvaluate {
			return evaluationCovers(record.Request, record.Evidence[i])
		}
	}
	return false
}

func hasFreshAllow(record Record, stage Stage, now time.Time) bool {
	for i := len(record.Evidence) - 1; i >= 0; i-- {
		e := record.Evidence[i]
		if e.Stage == stage {
			if isDiskStage(stage) {
				return validDiskProof(e.Proof, now)
			}
			return validProof(e.Proof, now)
		}
	}
	return false
}

func wellFormedProof(proof *Proof, now time.Time) bool {
	return proof != nil && strings.TrimSpace(proof.Decision) != "" && strings.TrimSpace(proof.ReceiptID) != "" && strings.TrimSpace(proof.SubjectDigest) != "" && proof.ExpiresAt.After(now)
}

func validProof(proof *Proof, now time.Time) bool {
	return wellFormedProof(proof, now) && strings.EqualFold(proof.Decision, "allow") && (proof.RequiredBytes <= 0 || proof.AvailableBytes >= proof.RequiredBytes)
}

func wellFormedDiskProof(proof *Proof, now time.Time) bool {
	return wellFormedProof(proof, now) && proof.RequiredBytes > 0 && proof.AvailableBytes >= proof.RequiredBytes
}

func validDiskProof(proof *Proof, now time.Time) bool {
	return wellFormedDiskProof(proof, now) && strings.EqualFold(proof.Decision, "allow")
}

func hasDisposition(record Record, stage Stage, disposition string) bool {
	for i := len(record.Evidence) - 1; i >= 0; i-- {
		if record.Evidence[i].Stage == stage {
			return strings.EqualFold(record.Evidence[i].Disposition, disposition)
		}
	}
	return false
}

func acquisitionCovers(request Request, evidence Evidence) bool {
	digests, ok := stringMapFromAny(evidence.Details["candidate_artifact_digests"])
	if !ok || len(digests) != len(request.Candidates) {
		return false
	}
	seenDigests := make(map[string]string, len(digests))
	for _, candidate := range request.Candidates {
		actual := strings.TrimSpace(digests[candidate.ID])
		if actual == "" || (candidate.ArtifactDigest != "" && actual != candidate.ArtifactDigest) {
			return false
		}
		if _, duplicate := seenDigests[actual]; duplicate {
			return false
		}
		seenDigests[actual] = candidate.ID
	}
	return true
}

func cleanupCovers(request Request, winnerID string, evidence Evidence) bool {
	if !strings.EqualFold(evidence.Disposition, "cleanup_complete") || evidence.Details == nil {
		return false
	}
	receipts, ok := stringMapFromAny(evidence.Details["removal_receipts"])
	if !ok {
		return false
	}
	expected := make(map[string]bool, len(request.Candidates))
	for _, c := range request.Candidates {
		if c.ID != winnerID {
			expected[c.ID] = true
		}
	}
	if request.PredecessorID != "" && request.PredecessorID != winnerID {
		expected[request.PredecessorID] = true
	}
	if len(expected) == 0 || len(receipts) != len(expected) {
		return false
	}
	for id := range expected {
		if strings.TrimSpace(receipts[id]) == "" {
			return false
		}
	}
	return true
}

func evaluationCovers(request Request, evidence Evidence) bool {
	if evidence.Details == nil || evidence.Details["suite_id"] != request.SuiteID {
		return false
	}
	count, ok := integerFromAny(evidence.Details["case_count"])
	if !ok || count < request.MinimumCases || count < len(request.CaseIDs) || evidence.Details["case_digest"] != digestCases(request.CaseIDs) {
		return false
	}
	results, ok := mapFromAny(evidence.Details["candidate_results"])
	if !ok || len(results) != len(request.Candidates) {
		return false
	}
	for _, candidate := range request.Candidates {
		result, ok := mapFromAny(results[candidate.ID])
		if !ok {
			return false
		}
		caseCount, ok := integerFromAny(result["case_count"])
		if !ok || caseCount < count {
			return false
		}
		if result["case_digest"] != digestCases(request.CaseIDs) || !isFiniteNumber(result["score"]) {
			return false
		}
	}
	return true
}

func validateRequest(request Request) error {
	if strings.TrimSpace(request.RunID) == "" || strings.TrimSpace(request.RunID) != request.RunID || len(request.RunID) > 128 || strings.TrimSpace(request.Target) == "" || strings.TrimSpace(request.Target) != request.Target || len(request.Target) > 80 || strings.TrimSpace(request.SuiteID) == "" || strings.TrimSpace(request.SuiteID) != request.SuiteID || len(request.SuiteID) > 256 || request.MinimumCases < 1 || len(request.Candidates) < 1 || len(request.Candidates) > 16 || len(request.CaseIDs) < request.MinimumCases || len(request.CaseIDs) > 2000 || len(request.Subject) > 32 {
		return ErrInvalidRequest
	}
	if len(request.Candidates) == 1 && strings.TrimSpace(request.PredecessorID) == "" {
		return fmt.Errorf("single-candidate optimization requires a predecessor identity for cleanup: %w", ErrInvalidRequest)
	}
	seen := map[string]bool{}
	for _, candidate := range request.Candidates {
		if strings.TrimSpace(candidate.ID) == "" || strings.TrimSpace(candidate.ID) != candidate.ID || len(candidate.ID) > 256 || len(candidate.ArtifactDigest) > 256 || (candidate.ArtifactDigest != "" && strings.TrimSpace(candidate.ArtifactDigest) != candidate.ArtifactDigest) || len(candidate.PredecessorDigest) > 256 || (candidate.PredecessorDigest != "" && strings.TrimSpace(candidate.PredecessorDigest) != candidate.PredecessorDigest) || seen[candidate.ID] {
			return ErrInvalidRequest
		}
		seen[candidate.ID] = true
	}
	if request.PredecessorID != "" && (strings.TrimSpace(request.PredecessorID) != request.PredecessorID || seen[request.PredecessorID]) {
		return ErrInvalidRequest
	}
	seen = map[string]bool{}
	for _, id := range request.CaseIDs {
		if strings.TrimSpace(id) == "" || strings.TrimSpace(id) != id || len(id) > 256 || seen[id] {
			return ErrInvalidRequest
		}
		seen[id] = true
	}
	return nil
}
