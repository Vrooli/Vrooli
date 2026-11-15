package optimization

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	desktopStartupTarget = "desktop-startup"
	desktopMinCases      = 30
	desktopMinPairs      = 5
)

type desktopIdentity struct {
	Scenario        string `json:"scenario"`
	ArtifactDigest  string `json:"artifact_digest"`
	HostFingerprint string `json:"host_fingerprint"`
	Display         string `json:"display"`
	DeploymentMode  string `json:"deployment_mode"`
	ProfilerMode    string `json:"profiler_mode"`
}

type desktopCandidate struct {
	ID                string            `json:"id"`
	PredecessorDigest string            `json:"predecessor_digest"`
	ArtifactDigest    string            `json:"artifact_digest"`
	StartupProfile    map[string]string `json:"startup_profile"`
}

type desktopAcquisitionProof struct {
	ReceiptID         string `json:"receipt_id"`
	CandidateID       string `json:"candidate_id"`
	ArtifactDigest    string `json:"artifact_digest"`
	PredecessorDigest string `json:"predecessor_digest"`
	SubjectDigest     string `json:"subject_digest"`
}

type desktopGateProof struct {
	Decision      string    `json:"decision"`
	ReceiptID     string    `json:"receipt_id"`
	SubjectDigest string    `json:"subject_digest"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type desktopDiskProof struct {
	Decision       string    `json:"decision"`
	LeaseID        string    `json:"lease_id"`
	SubjectDigest  string    `json:"subject_digest"`
	RequiredBytes  int64     `json:"required_bytes"`
	AvailableBytes int64     `json:"available_bytes"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type desktopRequest struct {
	Schema                      string                  `json:"schema"`
	RunID                       string                  `json:"run_id"`
	Target                      string                  `json:"target"`
	Subject                     desktopIdentity         `json:"subject"`
	ProofBindingDigest          string                  `json:"proof_binding_digest"`
	PreflightProofBindingDigest string                  `json:"preflight_proof_binding_digest"`
	BaselineRef                 string                  `json:"baseline_ref"`
	Candidate                   desktopCandidate        `json:"candidate"`
	Acquisition                 desktopAcquisitionProof `json:"acquisition"`
	PreflightRights             desktopGateProof        `json:"preflight_rights"`
	PreflightDisk               desktopDiskProof        `json:"preflight_disk"`
	Rights                      desktopGateProof        `json:"rights"`
	Disk                        desktopDiskProof        `json:"disk"`
	MinimumPairs                int                     `json:"minimum_pairs"`
	MaxRegressionFraction       float64                 `json:"max_regression_fraction"`
}

// DesktopStartupAdapter pins the Program Runtime target contract. It remains
// fail-closed until scenario-to-desktop exposes an artifact acquisition owner.
// In particular, no DP1 stage call is reachable without that acquisition.
type DesktopStartupAdapter struct{}

func NewDesktopStartupAdapter() *DesktopStartupAdapter { return &DesktopStartupAdapter{} }

func (*DesktopStartupAdapter) Target() string { return desktopStartupTarget }

func (a *DesktopStartupAdapter) Validate(request Request) error {
	if request.Target != desktopStartupTarget {
		return fmt.Errorf("unsupported target %q", request.Target)
	}
	if len(request.Candidates) != 1 {
		return errors.New("desktop-startup requires exactly one candidate and its predecessor")
	}
	if request.MinimumCases < desktopMinCases || len(request.CaseIDs) < desktopMinCases {
		return fmt.Errorf("desktop-startup requires at least %d fixed cases", desktopMinCases)
	}
	candidate := request.Candidates[0]
	if strings.TrimSpace(request.PredecessorID) == "" || candidate.ArtifactDigest == "" || candidate.PredecessorDigest == "" {
		return errors.New("desktop-startup candidate, artifact digest, and predecessor identity must be pinned")
	}
	if candidate.PredecessorDigest != stringSubject(request.Subject, "artifact_digest") {
		return errors.New("desktop-startup candidate predecessor digest must match subject.artifact_digest")
	}
	for _, key := range []string{"scenario", "artifact_digest", "host_fingerprint", "display", "deployment_mode", "profiler_mode"} {
		if strings.TrimSpace(stringSubject(request.Subject, key)) == "" {
			return fmt.Errorf("desktop-startup subject.%s is required", key)
		}
	}
	if strings.TrimSpace(request.SuiteID) == "" {
		return errors.New("desktop-startup suite_id is required")
	}
	if len(stringMap(request.Subject["startup_profile"])) == 0 {
		return errors.New("desktop-startup subject.startup_profile must contain target-owned settings")
	}
	return nil
}

func (a *DesktopStartupAdapter) Acquire(context.Context, string, Request) (Evidence, error) {
	return Evidence{}, fmt.Errorf("desktop-startup artifact acquisition owner is unavailable; no artifact will be synthesized: %w", ErrOwnerUnavailable)
}

func (*DesktopStartupAdapter) Evaluate(context.Context, string, Request, []Evidence) (Evidence, error) {
	return Evidence{}, fmt.Errorf("desktop-startup artifact acquisition owner is unavailable; DP1 measurement will not be called: %w", ErrOwnerUnavailable)
}

func (a *DesktopStartupAdapter) CheckRights(context.Context, string, Stage, Request, []Evidence) (Evidence, error) {
	return Evidence{}, fmt.Errorf("desktop-startup rights proof owner is unavailable; no proof will be synthesized: %w", ErrOwnerUnavailable)
}

func (a *DesktopStartupAdapter) CheckDisk(context.Context, string, Stage, Request, []Evidence) (Evidence, error) {
	return Evidence{}, fmt.Errorf("desktop-startup optimizer disk admission owner is unavailable; no proof will be synthesized: %w", ErrOwnerUnavailable)
}

func (*DesktopStartupAdapter) Promote(context.Context, string, Request, []Evidence) (Evidence, error) {
	return Evidence{}, fmt.Errorf("desktop-startup acquisition and promotion owners are unavailable: %w", ErrOwnerUnavailable)
}

func (*DesktopStartupAdapter) Cleanup(context.Context, string, Request, []Evidence) (Evidence, error) {
	return Evidence{}, fmt.Errorf("desktop-startup acquisition and predecessor cleanup owners are unavailable: %w", ErrOwnerUnavailable)
}

func (a *DesktopStartupAdapter) mapRequest(request Request, evidence []Evidence, requireFreshProofs bool) (desktopRequest, error) {
	if err := a.Validate(request); err != nil {
		return desktopRequest{}, err
	}
	acquired, ok := findEvidence(evidence, StageAcquire)
	if !ok || strings.TrimSpace(acquired.ReceiptID) == "" {
		return desktopRequest{}, errors.New("desktop-startup requires the acquired candidate owner receipt")
	}
	actualDigests, ok := stringMapFromAny(acquired.Details["candidate_artifact_digests"])
	if !ok {
		return desktopRequest{}, errors.New("desktop-startup acquire receipt has no owner-confirmed artifact digests")
	}
	candidate := request.Candidates[0]
	actual := strings.TrimSpace(actualDigests[candidate.ID])
	if actual == "" || actual != candidate.ArtifactDigest {
		return desktopRequest{}, errors.New("desktop-startup acquired digest differs from the pinned candidate digest")
	}
	preflightRights, ok := findEvidence(evidence, StagePreflightRights)
	if !ok || preflightRights.Proof == nil {
		return desktopRequest{}, errors.New("desktop-startup requires current preflight rights proof")
	}
	preflightDisk, ok := findEvidence(evidence, StagePreflightDisk)
	if !ok || preflightDisk.Proof == nil {
		return desktopRequest{}, errors.New("desktop-startup requires current preflight disk proof")
	}
	var rights, disk Evidence
	if requireFreshProofs {
		var ok bool
		rights, ok = findEvidence(evidence, StageRights)
		if !ok || rights.Proof == nil {
			return desktopRequest{}, errors.New("desktop-startup requires refreshed rights proof")
		}
		disk, ok = findEvidence(evidence, StageDisk)
		if !ok || disk.Proof == nil {
			return desktopRequest{}, errors.New("desktop-startup requires refreshed disk proof")
		}
	}
	preDigest, fullDigest := SubjectDigest(request), AcquiredSubjectDigest(request, evidence)
	if err := validateDesktopRightsProof(preflightRights.Proof, preDigest, time.Now()); err != nil {
		return desktopRequest{}, fmt.Errorf("desktop-startup preflight rights proof: %w", err)
	}
	if err := validateDesktopDiskProof(preflightDisk.Proof, preDigest, time.Now()); err != nil {
		return desktopRequest{}, fmt.Errorf("desktop-startup preflight disk proof: %w", err)
	}
	if requireFreshProofs {
		if err := validateDesktopRightsProof(rights.Proof, fullDigest, time.Now()); err != nil {
			return desktopRequest{}, fmt.Errorf("desktop-startup refreshed rights proof: %w", err)
		}
		if err := validateDesktopDiskProof(disk.Proof, fullDigest, time.Now()); err != nil {
			return desktopRequest{}, fmt.Errorf("desktop-startup refreshed disk proof: %w", err)
		}
	}
	identity := desktopIdentity{
		Scenario: stringSubject(request.Subject, "scenario"), ArtifactDigest: stringSubject(request.Subject, "artifact_digest"),
		HostFingerprint: stringSubject(request.Subject, "host_fingerprint"), Display: stringSubject(request.Subject, "display"),
		DeploymentMode: stringSubject(request.Subject, "deployment_mode"), ProfilerMode: stringSubject(request.Subject, "profiler_mode"),
	}
	return desktopRequest{
		Schema: "desktop-startup.optimize-request.v1", RunID: request.RunID, Target: request.Target, Subject: identity,
		ProofBindingDigest: fullDigest, PreflightProofBindingDigest: preDigest,
		BaselineRef: request.SuiteID, Candidate: desktopCandidate{ID: candidate.ID, ArtifactDigest: actual, PredecessorDigest: candidate.PredecessorDigest, StartupProfile: stringMap(request.Subject["startup_profile"])},
		Acquisition:     desktopAcquisitionProof{ReceiptID: acquired.ReceiptID, CandidateID: candidate.ID, ArtifactDigest: actual, PredecessorDigest: candidate.PredecessorDigest, SubjectDigest: fullDigest},
		PreflightRights: gateProof(preflightRights.Proof), PreflightDisk: diskProof(preflightDisk.Proof),
		Rights: gateProof(proofOrNil(rights)), Disk: diskProof(proofOrNil(disk)),
		MinimumPairs: desktopMinPairs, MaxRegressionFraction: floatSubject(request.Subject, "max_regression_fraction", 0.05),
	}, nil
}

func gateProof(proof *Proof) desktopGateProof {
	if proof == nil {
		return desktopGateProof{}
	}
	return desktopGateProof{Decision: proof.Decision, ReceiptID: proof.ReceiptID, SubjectDigest: proof.SubjectDigest, ExpiresAt: proof.ExpiresAt}
}

func diskProof(proof *Proof) desktopDiskProof {
	if proof == nil {
		return desktopDiskProof{}
	}
	decision := proof.Decision
	if decision == "allow" {
		decision = "admit"
	}
	return desktopDiskProof{Decision: decision, LeaseID: proof.ReceiptID, SubjectDigest: proof.SubjectDigest, RequiredBytes: proof.RequiredBytes, AvailableBytes: proof.AvailableBytes, ExpiresAt: proof.ExpiresAt}
}

func proofOrNil(evidence Evidence) *Proof { return evidence.Proof }

func validateDesktopRightsProof(proof *Proof, expectedDigest string, now time.Time) error {
	if proof == nil || proof.Decision != "allow" || strings.TrimSpace(proof.ReceiptID) == "" || proof.SubjectDigest != expectedDigest || !proof.ExpiresAt.After(now) {
		return errors.New("proof is absent, denied, expired, or bound to another subject")
	}
	return nil
}

func validateDesktopDiskProof(proof *Proof, expectedDigest string, now time.Time) error {
	if proof == nil || proof.Decision != "allow" || strings.TrimSpace(proof.ReceiptID) == "" || proof.SubjectDigest != expectedDigest || proof.RequiredBytes <= 0 || proof.AvailableBytes < proof.RequiredBytes || !proof.ExpiresAt.After(now) {
		return errors.New("lease is absent, denied, expired, insufficient, or bound to another subject")
	}
	return nil
}

func findEvidence(evidence []Evidence, stage Stage) (Evidence, bool) {
	for index := len(evidence) - 1; index >= 0; index-- {
		if evidence[index].Stage == stage {
			return evidence[index], true
		}
	}
	return Evidence{}, false
}

func stringSubject(subject map[string]any, key string) string {
	value, _ := subject[key].(string)
	return strings.TrimSpace(value)
}

func stringMap(value any) map[string]string {
	result := map[string]string{}
	switch items := value.(type) {
	case map[string]string:
		for key, item := range items {
			result[key] = item
		}
	case map[string]any:
		for key, raw := range items {
			if item, ok := raw.(string); ok {
				result[key] = item
			}
		}
	}
	return result
}

func floatSubject(subject map[string]any, key string, fallback float64) float64 {
	switch value := subject[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	default:
		return fallback
	}
}

var _ Adapter = (*DesktopStartupAdapter)(nil)
