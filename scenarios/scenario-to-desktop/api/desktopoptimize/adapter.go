// Package desktopoptimize owns the DP1 desktop-startup target adapter used by
// the shared local-model-performance optimizer. It deliberately delegates
// measurement, artifact promotion, predecessor removal, and rollback to the
// scenario-to-desktop owners that already own those effects.
package desktopoptimize

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	RequestSchema = "desktop-startup.optimize-request.v1"
	ReceiptSchema = "desktop-startup.optimize-receipt.v1"
	TargetID      = "desktop-startup"
)

var requiredPhases = []string{"process_to_splash_first_paint", "splash_to_app_ready", "runtime_spawn_to_ready"}

type Identity struct {
	Scenario        string `json:"scenario"`
	ArtifactDigest  string `json:"artifact_digest"`
	HostFingerprint string `json:"host_fingerprint"`
	Display         string `json:"display"`
	DeploymentMode  string `json:"deployment_mode"`
	ProfilerMode    string `json:"profiler_mode"`
}

type GateProof struct {
	Decision      string    `json:"decision"`
	ReceiptID     string    `json:"receipt_id"`
	SubjectDigest string    `json:"subject_digest"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type DiskProof struct {
	Decision       string    `json:"decision"`
	LeaseID        string    `json:"lease_id"`
	SubjectDigest  string    `json:"subject_digest"`
	RequiredBytes  int64     `json:"required_bytes"`
	AvailableBytes int64     `json:"available_bytes"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type Candidate struct {
	ID                string            `json:"id"`
	PredecessorDigest string            `json:"predecessor_digest"`
	ArtifactDigest    string            `json:"artifact_digest"`
	StartupProfile    map[string]string `json:"startup_profile"`
}

// AcquisitionProof identifies the target-owned immutable artifact selected by
// Program Runtime's acquire stage. The engine adapter must populate it from
// that stage's verified receipt, after applying any optional expected digest.
type AcquisitionProof struct {
	ReceiptID         string `json:"receipt_id"`
	CandidateID       string `json:"candidate_id"`
	ArtifactDigest    string `json:"artifact_digest"`
	PredecessorDigest string `json:"predecessor_digest"`
	SubjectDigest     string `json:"subject_digest"`
}

// Request carries immutable identities and owner-issued admission evidence.
// The adapter does not interpret a profile as authorization to alter a host.
type Request struct {
	Schema  string   `json:"schema"`
	RunID   string   `json:"run_id"`
	Target  string   `json:"target"`
	Subject Identity `json:"subject"`
	// ProofBindingDigest lets an upstream optimizer bind LM5/capacity proofs to
	// its full immutable request. When absent, the legacy DP1 identity digest is
	// used. The upstream adapter must compute and validate this value itself.
	ProofBindingDigest          string           `json:"proof_binding_digest,omitempty"`
	PreflightProofBindingDigest string           `json:"preflight_proof_binding_digest,omitempty"`
	BaselineRef                 string           `json:"baseline_ref"`
	Candidate                   Candidate        `json:"candidate"`
	Acquisition                 AcquisitionProof `json:"acquisition"`
	PreflightRights             GateProof        `json:"preflight_rights"`
	PreflightDisk               DiskProof        `json:"preflight_disk"`
	Rights                      GateProof        `json:"rights"`
	Disk                        DiskProof        `json:"disk"`
	MinimumPairs                int              `json:"minimum_pairs"`
	MaxRegressionFraction       float64          `json:"max_regression_fraction"`
}

type Sample struct {
	RunID       string `json:"run_id"`
	EvidenceRef string `json:"evidence_ref"`
	DurationMS  int64  `json:"duration_ms"`
}

type Pair struct {
	Class     string `json:"class"`
	Phase     string `json:"phase"`
	PairID    string `json:"pair_id"`
	Control   Sample `json:"control"`
	Candidate Sample `json:"candidate"`
}

type Matrix struct {
	Subject         Identity `json:"subject"`
	CandidateDigest string   `json:"candidate_digest"`
	Pairs           []Pair   `json:"pairs"`
}

type MeasurementPlan struct {
	Target        string   `json:"target"`
	Classes       []string `json:"classes"`
	Phases        []string `json:"phases"`
	PairsPerClass int      `json:"pairs_per_class"`
	Pairing       string   `json:"pairing"`
}

// MinimumPlan is the DP1 denominator. It remains explicit even when a host
// cannot run it; unsupported cells must be surfaced by the evaluator as an
// error/unavailable owner result, never silently omitted.
func MinimumPlan(req Request) MeasurementPlan {
	count := req.MinimumPairs
	if count < 5 {
		count = 5
	}
	return MeasurementPlan{Target: TargetID, Classes: []string{"cold", "warm"}, Phases: append([]string(nil), requiredPhases...), PairsPerClass: count, Pairing: "control_then_candidate_same_host_same_artifact_cohort"}
}

type EvaluationOwner interface {
	MeasurePaired(context.Context, string, Request, MeasurementPlan) (Matrix, error)
}
type PromotionOwner interface {
	Promote(context.Context, string, Request) (string, error)
	RemovePredecessor(context.Context, string, Request, string) (string, error)
	Rollback(context.Context, string, Request, string) (string, error)
}

// Journal supports idempotent resume: Find returns the newest durable receipt
// for a run ID, allowing a restarted engine to recover terminal dispositions.
type Journal interface {
	Append(Receipt) error
	Find(runID string) (Receipt, bool, error)
}

type Dependencies struct {
	Evaluator EvaluationOwner
	Promotion PromotionOwner
	Journal   Journal
	Now       func() time.Time
}

type PairSummary struct {
	Class             string   `json:"class"`
	Phase             string   `json:"phase"`
	Count             int      `json:"count"`
	ControlP50MS      int64    `json:"control_p50_ms"`
	CandidateP50MS    int64    `json:"candidate_p50_ms"`
	ControlP95MS      int64    `json:"control_p95_ms"`
	CandidateP95MS    int64    `json:"candidate_p95_ms"`
	ControlEvidence   []string `json:"control_evidence"`
	CandidateEvidence []string `json:"candidate_evidence"`
}

type Receipt struct {
	Schema                    string        `json:"schema"`
	RunID                     string        `json:"run_id"`
	Disposition               string        `json:"disposition"`
	Stage                     string        `json:"stage,omitempty"`
	Reason                    string        `json:"reason,omitempty"`
	StartedAt                 time.Time     `json:"started_at"`
	FinishedAt                time.Time     `json:"finished_at"`
	RequestDigest             string        `json:"request_digest"`
	Subject                   Identity      `json:"subject"`
	Candidate                 Candidate     `json:"candidate"`
	Matrix                    []PairSummary `json:"paired_matrix,omitempty"`
	PromotionReceipt          string        `json:"promotion_receipt,omitempty"`
	PredecessorRemovalReceipt string        `json:"predecessor_removal_receipt,omitempty"`
	RollbackReceipt           string        `json:"rollback_receipt,omitempty"`
}

// Measure performs only DP1 measurement. The stable owner operation ID lets an
// evaluator replay an interrupted call without launching a second matrix.
func Measure(ctx context.Context, deps Dependencies, req Request) Receipt {
	r, done := prepare(ctx, deps, req)
	if done {
		return r
	}
	if r.Stage == "measurement-complete" || r.Stage == "promotion-intent" || r.Stage == "promotion-complete" || r.Stage == "predecessor-removal-intent" || r.Stage == "complete" {
		return r
	}
	if err := ValidateMeasurementRequestAt(req, stageNow(deps)); err != nil {
		return recordTerminal(deps, r, "refused", err.Error())
	}
	if deps.Evaluator == nil {
		return recordTerminal(deps, r, "refused", "measurement owner is required")
	}
	if r.Stage != "measurement-intent" {
		r.Disposition, r.Stage = "running", "measurement-intent"
		if err := deps.Journal.Append(r); err != nil {
			return recordTerminal(deps, r, "failed", "could not persist measurement intent: "+err.Error())
		}
	}
	matrix, err := deps.Evaluator.MeasurePaired(ctx, operationID(req.RunID, "measure"), req, MinimumPlan(req))
	if err != nil {
		r.Disposition, r.Reason = "running", "measurement owner retry required: "+err.Error()
		_ = deps.Journal.Append(r)
		return r
	}
	summaries, err := ValidateMatrix(req, matrix)
	if err == nil {
		err = acceptMatrix(summaries, req.MaxRegressionFraction)
	}
	if err != nil {
		return recordTerminal(deps, r, "refused", err.Error())
	}
	r.Matrix = summaries
	r.Disposition, r.Stage, r.Reason = "running", "measurement-complete", ""
	if err := deps.Journal.Append(r); err != nil {
		r.Disposition, r.Stage, r.Reason = "running", "measurement-intent", "measurement receipt persistence retry required: "+err.Error()
		return r
	}
	return r
}

// Promote performs only candidate promotion and requires a persisted passing
// matrix. It never removes the predecessor.
func Promote(ctx context.Context, deps Dependencies, req Request) Receipt {
	r := loadForStage(deps, req)
	if terminal(r) || r.Disposition == "refused" || r.Disposition == "failed" {
		return r
	}
	if r.Stage == "promotion-complete" || r.Stage == "predecessor-removal-intent" || r.Stage == "complete" {
		return r
	}
	if r.Stage != "measurement-complete" && r.Stage != "promotion-intent" {
		return recordTerminal(deps, r, "refused", "a complete accepted DP1 measurement receipt is required before promotion")
	}
	if err := ValidateRequest(req, stageNow(deps)); err != nil {
		return recordTerminal(deps, r, "refused", err.Error())
	}
	if deps.Promotion == nil {
		return recordTerminal(deps, r, "refused", "promotion owner is required")
	}
	if r.Stage != "promotion-intent" {
		r.Disposition, r.Stage = "running", "promotion-intent"
		if err := deps.Journal.Append(r); err != nil {
			return recordTerminal(deps, r, "failed", "could not persist promotion intent: "+err.Error())
		}
	}
	id, err := deps.Promotion.Promote(ctx, operationID(req.RunID, "promote"), req)
	if err != nil || strings.TrimSpace(id) == "" {
		reason := "candidate promotion owner returned no receipt identity"
		if err != nil {
			reason = "candidate promotion owner retry required: " + err.Error()
		}
		r.Disposition, r.Reason = "running", reason
		_ = deps.Journal.Append(r)
		return r
	}
	r.PromotionReceipt = id
	r.Disposition, r.Stage, r.Reason = "running", "promotion-complete", ""
	if err := deps.Journal.Append(r); err != nil {
		r.Disposition, r.Stage, r.Reason = "running", "promotion-intent", "promotion receipt persistence retry required: "+err.Error()
		return r
	}
	return r
}

// Cleanup removes the predecessor after a persisted promotion and rolls back
// through the owner if removal fails. It is independently restartable.
func Cleanup(ctx context.Context, deps Dependencies, req Request) Receipt {
	r := loadForStage(deps, req)
	if terminal(r) {
		return r
	}
	if r.Stage == "complete" || r.Disposition == "refused" || r.Disposition == "failed" {
		return r
	}
	if r.Stage != "promotion-complete" && r.Stage != "predecessor-removal-intent" {
		return recordTerminal(deps, r, "refused", "a persisted promotion receipt is required before predecessor removal")
	}
	if strings.TrimSpace(r.PromotionReceipt) == "" {
		return recordTerminal(deps, r, "refused", "promotion receipt identity is required before predecessor removal")
	}
	if err := ValidateRequest(req, stageNow(deps)); err != nil {
		return recordTerminal(deps, r, "refused", err.Error())
	}
	if deps.Promotion == nil {
		return recordTerminal(deps, r, "refused", "predecessor removal and rollback owner is required")
	}
	if r.Stage != "predecessor-removal-intent" {
		r.Disposition, r.Stage = "running", "predecessor-removal-intent"
		if err := deps.Journal.Append(r); err != nil {
			r.Disposition, r.Stage, r.Reason = "running", "promotion-complete", "predecessor-removal intent persistence retry required: "+err.Error()
			return r
		}
	}
	id, err := deps.Promotion.RemovePredecessor(ctx, operationID(req.RunID, "remove-predecessor"), req, r.PromotionReceipt)
	if err != nil || strings.TrimSpace(id) == "" {
		reason := "predecessor removal owner returned no receipt identity"
		if err != nil {
			reason = "predecessor removal failed: " + err.Error()
		}
		return rollbackFailure(ctx, deps, req, r, reason)
	}
	r.PredecessorRemovalReceipt = id
	r.Disposition, r.Stage, r.Reason, r.FinishedAt = "promoted", "complete", "", stageNow(deps)
	if err := deps.Journal.Append(r); err != nil {
		r.Disposition, r.Stage, r.FinishedAt, r.Reason = "running", "predecessor-removal-intent", time.Time{}, "predecessor-removal receipt persistence retry required: "+err.Error()
		return r
	}
	return r
}

// Execute preserves the original all-in-one API as a wrapper over the staged
// operations. Program Runtime may instead call each stage independently.
func Execute(ctx context.Context, deps Dependencies, req Request) Receipt {
	prior := loadForStage(deps, req)
	if !terminal(prior) && prior.Stage == "" {
		if err := ValidateRequest(req, stageNow(deps)); err != nil {
			return recordTerminal(deps, prior, "refused", err.Error())
		}
	}
	r := Measure(ctx, deps, req)
	if r.Disposition != "running" {
		return r
	}
	if r.Stage == "measurement-complete" || r.Stage == "promotion-intent" {
		r = Promote(ctx, deps, req)
	}
	if r.Disposition != "running" {
		return r
	}
	if r.Stage == "promotion-complete" || r.Stage == "predecessor-removal-intent" {
		return Cleanup(ctx, deps, req)
	}
	return r
}

func prepare(ctx context.Context, deps Dependencies, req Request) (Receipt, bool) {
	r := loadForStage(deps, req)
	if r.Disposition == "failed" || r.Disposition == "refused" || r.Disposition == "recovery-required" {
		return r, true
	}
	if r.Stage != "" {
		return r, false
	}
	if err := ValidateMeasurementRequestAt(req, stageNow(deps)); err != nil {
		return recordTerminal(deps, r, "refused", err.Error()), true
	}
	if deps.Journal == nil {
		return recordTerminal(deps, r, "refused", "durable receipt journal owner is required"), true
	}
	return r, false
}

func loadForStage(deps Dependencies, req Request) Receipt {
	now := stageNow(deps)
	r := Receipt{Schema: ReceiptSchema, RunID: strings.TrimSpace(req.RunID), StartedAt: now, Subject: req.Subject, Candidate: req.Candidate, RequestDigest: requestDigest(req)}
	if deps.Journal == nil {
		r.Disposition, r.Reason, r.FinishedAt = "refused", "durable receipt journal owner is required", now
		return r
	}
	prior, found, err := deps.Journal.Find(req.RunID)
	if err != nil {
		r.Disposition, r.Reason, r.FinishedAt = "failed", "recover prior run receipt: "+err.Error(), now
		return r
	}
	if !found {
		return r
	}
	if prior.RequestDigest != r.RequestDigest {
		r.Disposition, r.Reason, r.FinishedAt = "refused", "run_id is already bound to a different immutable request", now
		return r
	}
	return prior
}

func recordTerminal(deps Dependencies, r Receipt, disposition, reason string) Receipt {
	r.Disposition, r.Reason, r.FinishedAt = disposition, reason, stageNow(deps)
	if deps.Journal != nil {
		if err := deps.Journal.Append(r); err != nil {
			r.Disposition = "failed"
			r.Reason = joinReason(reason, "receipt journal append failed: "+err.Error())
		}
	}
	return r
}

func rollbackFailure(ctx context.Context, deps Dependencies, req Request, r Receipt, reason string) Receipt {
	rollbackID, rollbackErr := deps.Promotion.Rollback(ctx, operationID(req.RunID, "rollback"), req, r.PromotionReceipt)
	r.RollbackReceipt = rollbackID
	if rollbackErr != nil {
		reason += "; rollback failed: " + rollbackErr.Error()
	}
	return recordTerminal(deps, r, "failed", reason)
}

func stageNow(deps Dependencies) time.Time {
	if deps.Now != nil {
		return deps.Now().UTC()
	}
	return time.Now().UTC()
}

func operationID(runID, stage string) string { return strings.TrimSpace(runID) + "/" + stage }

func terminal(r Receipt) bool {
	return r.Disposition == "promoted" || r.Disposition == "refused" || r.Disposition == "failed"
}

func ValidateRequest(r Request, now time.Time) error {
	if err := validateRequestShape(r); err != nil {
		return err
	}
	proofDigest := proofBindingDigest(r)
	if !validProof(r.Rights.Decision, r.Rights.ReceiptID, r.Rights.SubjectDigest, proofDigest, r.Rights.ExpiresAt, now) {
		return errors.New("rights gate is absent, stale, or not bound to the selected desktop artifact")
	}
	if r.Disk.Decision != "admit" || r.Disk.LeaseID == "" || r.Disk.RequiredBytes <= 0 || r.Disk.AvailableBytes < r.Disk.RequiredBytes || r.Disk.SubjectDigest != proofDigest || !r.Disk.ExpiresAt.After(now) {
		return errors.New("disk gate is absent, insufficient, stale, or not bound to the selected desktop artifact")
	}
	return nil
}

// ValidateMeasurementRequestAt checks immutable target/acquisition data and
// the preflight proofs which admit a measurement before Program Runtime's
// evaluation stage. Mutating stages must use ValidateRequest instead.
func ValidateMeasurementRequest(r Request) error {
	return ValidateMeasurementRequestAt(r, time.Now())
}

func ValidateMeasurementRequestAt(r Request, now time.Time) error {
	if err := validateRequestShape(r); err != nil {
		return err
	}
	rights, disk := r.PreflightRights, r.PreflightDisk
	if rights.ReceiptID == "" && rights.Decision == "" {
		rights = r.Rights
	}
	if disk.LeaseID == "" && disk.Decision == "" {
		disk = r.Disk
	}
	proofDigest := preflightProofBindingDigest(r)
	if !validProof(rights.Decision, rights.ReceiptID, rights.SubjectDigest, proofDigest, rights.ExpiresAt, now) {
		return errors.New("preflight rights gate is absent, stale, or not bound to the requested desktop cohort")
	}
	if disk.Decision != "admit" || disk.LeaseID == "" || disk.RequiredBytes <= 0 || disk.AvailableBytes < disk.RequiredBytes || disk.SubjectDigest != proofDigest || !disk.ExpiresAt.After(now) {
		return errors.New("preflight disk gate is absent, insufficient, stale, or not bound to the requested desktop cohort")
	}
	return nil
}

func validateRequestShape(r Request) error {
	if r.Schema != RequestSchema {
		return errors.New("unsupported request schema")
	}
	if strings.TrimSpace(r.RunID) == "" || r.Target != TargetID {
		return errors.New("run_id and desktop-startup target are required")
	}
	if r.Subject.Scenario == "" || r.Subject.ArtifactDigest == "" || r.Subject.HostFingerprint == "" || r.Subject.Display == "" || r.Subject.DeploymentMode == "" || r.Subject.ProfilerMode == "" {
		return errors.New("complete baseline cohort identity is required")
	}
	if r.BaselineRef == "" || r.Candidate.ID == "" || r.Candidate.ArtifactDigest == "" || r.Candidate.PredecessorDigest == "" {
		return errors.New("baseline and candidate identities are required")
	}
	if r.Candidate.PredecessorDigest != r.Subject.ArtifactDigest {
		return errors.New("candidate predecessor does not match the selected baseline artifact")
	}
	if r.Acquisition.ReceiptID == "" || r.Acquisition.CandidateID != r.Candidate.ID || r.Acquisition.ArtifactDigest != r.Candidate.ArtifactDigest || r.Acquisition.PredecessorDigest != r.Candidate.PredecessorDigest || r.Acquisition.SubjectDigest != proofBindingDigest(r) {
		return errors.New("acquisition receipt does not match the exact candidate artifact and proof subject")
	}
	if len(r.Candidate.StartupProfile) == 0 {
		return errors.New("candidate startup profile is empty")
	}
	if r.MinimumPairs < 5 {
		return errors.New("DP1 requires at least five paired samples in each cold and warm cohort")
	}
	if r.MaxRegressionFraction < 0 || r.MaxRegressionFraction > .25 {
		return errors.New("max_regression_fraction must be between 0 and 0.25")
	}
	return nil
}

func validProof(decision, id, digest, expectedDigest string, expiresAt, now time.Time) bool {
	return decision == "allow" && id != "" && digest == expectedDigest && expiresAt.After(now)
}

func proofBindingDigest(r Request) string {
	if strings.TrimSpace(r.ProofBindingDigest) != "" {
		return r.ProofBindingDigest
	}
	return identityDigest(r.Subject)
}

func preflightProofBindingDigest(r Request) string {
	if strings.TrimSpace(r.PreflightProofBindingDigest) != "" {
		return r.PreflightProofBindingDigest
	}
	return proofBindingDigest(r)
}

func ValidateMatrix(req Request, m Matrix) ([]PairSummary, error) {
	if m.Subject != req.Subject || m.CandidateDigest != req.Candidate.ArtifactDigest {
		return nil, errors.New("measurement matrix identity differs from requested cohort or candidate")
	}
	counts := map[string]map[string][]Pair{"cold": {}, "warm": {}}
	seen := map[string]bool{}
	runPairs := map[string]Pair{}
	for _, p := range m.Pairs {
		if p.Class != "cold" && p.Class != "warm" {
			return nil, fmt.Errorf("unsupported measurement class %q", p.Class)
		}
		if !contains(requiredPhases, p.Phase) {
			return nil, fmt.Errorf("unexpected DP1 phase %q", p.Phase)
		}
		if p.Control.RunID == "" || p.Candidate.RunID == "" || p.Control.EvidenceRef == "" || p.Candidate.EvidenceRef == "" {
			return nil, errors.New("every paired sample requires run and evidence identities")
		}
		if p.Control.DurationMS < 0 || p.Candidate.DurationMS < 0 {
			return nil, errors.New("negative startup duration is invalid")
		}
		if p.PairID == "" {
			return nil, errors.New("every control/candidate sample pair requires a shared pair_id")
		}
		pairKey := p.Class + "\x00" + p.PairID
		if prior, ok := runPairs[pairKey]; ok && (prior.Control.RunID != p.Control.RunID || prior.Candidate.RunID != p.Candidate.RunID) {
			return nil, errors.New("all phase rows for one pair_id must come from the same control and candidate launches")
		}
		runPairs[pairKey] = p
		key := strings.Join([]string{p.Class, p.Phase, p.PairID}, "\x00")
		if seen[key] {
			return nil, errors.New("duplicate paired sample identity")
		}
		seen[key] = true
		counts[p.Class][p.Phase] = append(counts[p.Class][p.Phase], p)
	}
	var out []PairSummary
	for _, class := range []string{"cold", "warm"} {
		for _, phase := range requiredPhases {
			pairs := counts[class][phase]
			if len(pairs) < req.MinimumPairs {
				return nil, fmt.Errorf("DP1 matrix incomplete: %s/%s has %d pairs; requires %d", class, phase, len(pairs), req.MinimumPairs)
			}
			control, candidate := make([]int64, 0, len(pairs)), make([]int64, 0, len(pairs))
			ce, ne := []string{}, []string{}
			for _, p := range pairs {
				control = append(control, p.Control.DurationMS)
				candidate = append(candidate, p.Candidate.DurationMS)
				ce = append(ce, p.Control.EvidenceRef)
				ne = append(ne, p.Candidate.EvidenceRef)
			}
			sort.Slice(control, func(i, j int) bool { return control[i] < control[j] })
			sort.Slice(candidate, func(i, j int) bool { return candidate[i] < candidate[j] })
			out = append(out, PairSummary{Class: class, Phase: phase, Count: len(pairs), ControlP50MS: nearest(control, 50), CandidateP50MS: nearest(candidate, 50), ControlP95MS: nearest(control, 95), CandidateP95MS: nearest(candidate, 95), ControlEvidence: ce, CandidateEvidence: ne})
		}
	}
	return out, nil
}

func acceptMatrix(rows []PairSummary, tolerance float64) error {
	improved := false
	for _, r := range rows {
		if float64(r.CandidateP95MS) > float64(r.ControlP95MS)*(1+tolerance) {
			return fmt.Errorf("candidate regresses DP1 phase %s/%s beyond allowed tolerance", r.Class, r.Phase)
		}
		if r.Phase == "process_to_splash_first_paint" && r.CandidateP50MS < r.ControlP50MS {
			improved = true
		}
	}
	if !improved {
		return errors.New("candidate did not improve median first-paint startup in either cohort")
	}
	return nil
}

func nearest(v []int64, p int) int64 {
	i := (len(v)*p+99)/100 - 1
	if i < 0 {
		i = 0
	}
	return v[i]
}

func contains(v []string, s string) bool {
	for _, x := range v {
		if x == s {
			return true
		}
	}
	return false
}

func identityDigest(i Identity) string {
	raw := fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s\x00%s", i.Scenario, i.ArtifactDigest, i.HostFingerprint, i.Display, i.DeploymentMode, i.ProfilerMode)
	sum := sha256.Sum256([]byte(raw))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func requestDigest(r Request) string {
	// Gate receipts are stage evidence, not immutable target identity. They are
	// first checked for measurement and then rechecked after evaluation before
	// mutation, while every target/acquisition identity remains pinned.
	r.Rights = GateProof{}
	r.Disk = DiskProof{}
	r.PreflightRights = GateProof{}
	r.PreflightDisk = DiskProof{}
	r.ProofBindingDigest = ""
	r.PreflightProofBindingDigest = ""
	raw, _ := json.Marshal(r)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func joinReason(a, b string) string {
	if a == "" {
		return b
	}
	return a + "; " + b
}
