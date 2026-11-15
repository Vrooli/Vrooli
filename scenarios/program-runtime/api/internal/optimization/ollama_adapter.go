package optimization

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	cliv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1/cliv1connect"
	"program-runtime/internal/bindings"
)

const (
	ollamaRoleTarget = "ollama-role"
	ollamaRoleCount  = 3
	ollamaMinCases   = 20
)

// OllamaRoleAdapter uses the root control plane's fixed typed Connect owner
// operation. Only installed resource-confirmed artifact digests enter the
// acquisition receipt. Evaluation, admission, promotion, and cleanup remain
// unavailable until their domain owners expose receipts.
type OllamaRoleAdapter struct {
	client     connect.HTTPClient
	resolveURL func(context.Context, string) (string, error)
}

// NewOllamaRoleAdapter returns the production owner adapter. The root project
// Connect endpoint is resolved through the same discovery contract used by
// Program Runtime's governed binding registry.
func NewOllamaRoleAdapter() *OllamaRoleAdapter {
	return NewOllamaRoleAdapterWithClient(http.DefaultClient, bindings.ResolveTargetURLDefault)
}

// NewOllamaRoleAdapterWithClient is the typed test seam for the fixed root
// control-plane Connect dependency.
func NewOllamaRoleAdapterWithClient(client connect.HTTPClient, resolveURL func(context.Context, string) (string, error)) *OllamaRoleAdapter {
	return &OllamaRoleAdapter{client: client, resolveURL: resolveURL}
}

func (a *OllamaRoleAdapter) Target() string { return ollamaRoleTarget }

func (a *OllamaRoleAdapter) Validate(request Request) error {
	if request.Target != ollamaRoleTarget {
		return fmt.Errorf("unsupported target %q", request.Target)
	}
	role, _ := request.Subject["role"].(string)
	if strings.TrimSpace(role) == "" {
		return errors.New("subject.role is required for Ollama optimization")
	}
	if len(request.Candidates) != ollamaRoleCount {
		return fmt.Errorf("Ollama role optimization requires exactly %d candidates", ollamaRoleCount)
	}
	seen := make(map[string]struct{}, len(request.Candidates))
	seenDigests := make(map[string]string, len(request.Candidates))
	for _, candidate := range request.Candidates {
		if strings.TrimSpace(candidate.ID) == "" || strings.TrimSpace(candidate.ArtifactDigest) == "" {
			return errors.New("each Ollama candidate must pin its expected registry/policy artifact digest")
		}
		if _, duplicate := seen[candidate.ID]; duplicate {
			return fmt.Errorf("Ollama candidate %q is duplicated", candidate.ID)
		}
		seen[candidate.ID] = struct{}{}
		if prior, duplicate := seenDigests[candidate.ArtifactDigest]; duplicate {
			return fmt.Errorf("Ollama candidates %q and %q pin the same artifact digest", prior, candidate.ID)
		}
		seenDigests[candidate.ArtifactDigest] = candidate.ID
	}
	if request.MinimumCases < ollamaMinCases || len(request.CaseIDs) < ollamaMinCases {
		return fmt.Errorf("Ollama role optimization requires at least %d fixed cases", ollamaMinCases)
	}
	return nil
}

func (a *OllamaRoleAdapter) Acquire(ctx context.Context, stageID string, request Request) (Evidence, error) {
	if err := a.Validate(request); err != nil {
		return Evidence{}, err
	}
	receipt, _, err := a.inspectRoleCandidates(ctx, stageID+":acquire", request)
	if err != nil {
		return Evidence{}, err
	}
	byModel := make(map[string]ollamaRoleOwnerCandidate, len(receipt.Candidates))
	for _, candidate := range receipt.Candidates {
		if _, duplicate := byModel[candidate.Model]; duplicate {
			return Evidence{}, fmt.Errorf("Ollama owner returned duplicate candidate %q", candidate.Model)
		}
		byModel[candidate.Model] = candidate
	}
	digests := make(map[string]string, len(request.Candidates))
	for _, candidate := range request.Candidates {
		owned, exists := byModel[candidate.ID]
		if !exists || !owned.Installed || !owned.PolicyEligible || owned.Rights != "allowed" || strings.TrimSpace(owned.Digest) == "" {
			return Evidence{}, fmt.Errorf("Ollama owner has no installed, policy-eligible digest for candidate %q", candidate.ID)
		}
		if candidate.ArtifactDigest != owned.Digest {
			return Evidence{}, fmt.Errorf("Ollama owner digest for candidate %q differs from the pinned expectation", candidate.ID)
		}
		digests[candidate.ID] = owned.Digest
	}
	return Evidence{Stage: StageAcquire, ReceiptID: receipt.ReceiptID, Details: map[string]any{
		"candidate_artifact_digests": digests,
		"owner_receipt":              receipt,
	}}, nil
}

func (a *OllamaRoleAdapter) CheckRights(ctx context.Context, runID string, stage Stage, request Request, prior []Evidence) (Evidence, error) {
	if stage != StagePreflightRights && stage != StageRights {
		return Evidence{}, fmt.Errorf("unsupported Ollama rights stage %q", stage)
	}
	if err := a.Validate(request); err != nil {
		return Evidence{}, err
	}
	expectedDigests := make(map[string]string, len(request.Candidates))
	if stage == StagePreflightRights {
		for _, candidate := range request.Candidates {
			expectedDigests[candidate.ID] = candidate.ArtifactDigest
		}
	} else {
		expectedDigests = acquiredDigests(prior)
		if len(expectedDigests) != len(request.Candidates) {
			return Evidence{}, errors.New("post-evaluation rights check requires the complete owner acquisition receipt")
		}
	}
	preflightPolicySHA := ""
	if stage == StageRights {
		for _, priorEvidence := range prior {
			if priorEvidence.Stage != StagePreflightRights || priorEvidence.Details == nil {
				continue
			}
			preflightPolicySHA, _ = priorEvidence.Details["policy_sha256"].(string)
			break
		}
		if preflightPolicySHA == "" {
			return Evidence{}, errors.New("post-evaluation rights check requires the preflight resource policy receipt")
		}
	}
	receipt, role, err := a.inspectRoleCandidates(ctx, runID+":"+string(stage), request)
	if err != nil {
		return Evidence{}, err
	}
	byModel := make(map[string]ollamaRoleOwnerCandidate, len(receipt.Candidates))
	for _, candidate := range receipt.Candidates {
		if _, duplicate := byModel[candidate.Model]; duplicate {
			return Evidence{}, fmt.Errorf("Ollama owner returned duplicate rights candidate %q", candidate.Model)
		}
		byModel[candidate.Model] = candidate
	}
	if len(byModel) != len(request.Candidates) {
		return Evidence{}, errors.New("Ollama owner rights receipt does not cover every requested candidate")
	}
	decision := "allow"
	reasons := make([]string, 0)
	rightsEvidence := make(map[string]ollamaRoleRightsEvidence, len(request.Candidates))
	checkedAt := time.Now().UTC()
	for _, pinned := range request.Candidates {
		owner, ok := byModel[pinned.ID]
		if !ok {
			return Evidence{}, fmt.Errorf("Ollama owner rights receipt omitted candidate %q", pinned.ID)
		}
		facts := owner.RightsEvidence
		rightsEvidence[pinned.ID] = facts
		if err := validateOllamaRightsEvidence(facts, receipt.PolicySHA256, checkedAt); err != nil {
			decision = "deny"
			reasons = append(reasons, fmt.Sprintf("%s: rights evidence incomplete: %v", pinned.ID, err))
			continue
		}
		if facts.Status != "allowed" || owner.Rights != "allowed" {
			decision = "deny"
			reason := strings.TrimSpace(owner.Reason)
			if reason == "" {
				reason = "current Ollama rights decision is " + facts.Status
			}
			reasons = append(reasons, pinned.ID+": "+reason)
		}
		if stage == StageRights {
			if !owner.Installed || owner.Digest != expectedDigests[pinned.ID] {
				decision = "deny"
				reasons = append(reasons, pinned.ID+": current runtime digest differs from the acquired candidate")
			}
			if !owner.PolicyEligible {
				decision = "deny"
				reasons = append(reasons, pinned.ID+": candidate is no longer policy-eligible for its target role")
			}
		}
	}
	if stage == StageRights && receipt.PolicySHA256 != preflightPolicySHA {
		decision = "deny"
		reasons = append(reasons, "authoritative Ollama policy changed after preflight")
	}
	if decision != "allow" && len(reasons) == 0 {
		reasons = append(reasons, "one or more Ollama candidates do not have current allowed rights evidence")
	}
	proofDigest := SubjectDigest(request)
	if stage == StageRights {
		proofDigest = AcquiredSubjectDigest(request, prior)
	}
	ownerReceiptID := receipt.ReceiptID + ":" + string(stage)
	proofExpiry := checkedAt.Add(2 * time.Minute)
	for _, facts := range rightsEvidence {
		evaluatedAt, parseErr := time.Parse(time.RFC3339Nano, facts.EvaluatedAt)
		if parseErr == nil && !evaluatedAt.IsZero() && evaluatedAt.Before(proofExpiry) {
			proofExpiry = evaluatedAt.Add(2 * time.Minute)
		}
	}
	details := map[string]any{
		"role": role, "policy_sha256": receipt.PolicySHA256,
		"owner_receipt": receipt, "candidate_rights_evidence": rightsEvidence,
	}
	if len(reasons) > 0 {
		details["refusals"] = reasons
	}
	return Evidence{
		Stage: stage, ReceiptID: ownerReceiptID, Proof: &Proof{
			Decision: decision, ReceiptID: ownerReceiptID, SubjectDigest: proofDigest, ExpiresAt: proofExpiry,
		}, Details: details,
	}, nil
}

func (a *OllamaRoleAdapter) inspectRoleCandidates(ctx context.Context, requestID string, request Request) (ollamaRoleOwnerReceipt, string, error) {
	role, _ := request.Subject["role"].(string)
	if a == nil || a.client == nil || a.resolveURL == nil {
		return ollamaRoleOwnerReceipt{}, role, fmt.Errorf("Ollama resource inspection owner is unavailable: %w", ErrOwnerUnavailable)
	}
	baseURL, err := a.resolveURL(ctx, "vrooli")
	if err != nil {
		return ollamaRoleOwnerReceipt{}, role, fmt.Errorf("resolve root control-plane API: %w", err)
	}
	client := cliv1connect.NewOllamaOptimizationServiceClient(a.client, baseURL)
	models := make([]string, len(request.Candidates))
	for i, candidate := range request.Candidates {
		models[i] = candidate.ID
	}
	response, err := client.InspectRoleCandidates(ctx, connect.NewRequest(&cliv1.InspectOllamaRoleCandidatesRequest{
		RequestId: requestID, Target: request.Target, Role: role,
		CandidateModels: models, FixedSuiteId: request.SuiteID, FixedCaseIds: append([]string(nil), request.CaseIDs...),
	}))
	if err != nil {
		return ollamaRoleOwnerReceipt{}, role, fmt.Errorf("inspect Ollama role candidates through the control plane: %w", err)
	}
	var receipt ollamaRoleOwnerReceipt
	decoder := json.NewDecoder(bytes.NewBufferString(response.Msg.GetReceiptJson()))
	if err := decoder.Decode(&receipt); err != nil {
		return ollamaRoleOwnerReceipt{}, role, fmt.Errorf("decode Ollama role owner receipt: %w", err)
	}
	if receipt.SchemaVersion != "ollama.optimize.receipt.v1" || receipt.RequestID != requestID || receipt.Target != request.Target || receipt.Role != role || strings.TrimSpace(receipt.ReceiptID) == "" {
		return ollamaRoleOwnerReceipt{}, role, errors.New("Ollama role owner receipt does not match the pinned request")
	}
	return receipt, role, nil
}

func validateOllamaRightsEvidence(facts ollamaRoleRightsEvidence, policySHA string, now time.Time) error {
	if facts.Status != "allowed" && facts.Status != "denied" && facts.Status != "unknown" {
		return errors.New("status must be allowed, denied, or unknown")
	}
	if !isSHA256Hex(facts.DecisionSHA256) || !isSHA256Hex(policySHA) || facts.PolicySHA256 != policySHA {
		return errors.New("decision or policy digest is missing or inconsistent")
	}
	evaluatedAt, err := time.Parse(time.RFC3339Nano, facts.EvaluatedAt)
	if err != nil || evaluatedAt.UTC().Format(time.RFC3339Nano) != facts.EvaluatedAt || evaluatedAt.After(now.Add(5*time.Second)) || evaluatedAt.Before(now.Add(-2*time.Minute)) {
		return errors.New("evaluation timestamp is invalid or stale")
	}
	if facts.ContextSHA256 != "" && !isSHA256Hex(facts.ContextSHA256) {
		return errors.New("context digest is invalid")
	}
	if facts.ArtifactSHA256 != "" && !isSHA256Hex(facts.ArtifactSHA256) {
		return errors.New("artifact rights digest is invalid")
	}
	if facts.Status == "allowed" && (facts.ContextID == "" || facts.ContextVersion == "" || facts.ContextSHA256 == "" || facts.ArtifactID == "" || facts.ArtifactSHA256 == "") {
		return errors.New("allowed decision lacks trusted context or source artifact identity")
	}
	return nil
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return true
}

func (*OllamaRoleAdapter) Evaluate(context.Context, string, Request, []Evidence) (Evidence, error) {
	return Evidence{}, fmt.Errorf("LM6 model evaluation owner is unavailable; candidate scores will not be synthesized: %w", ErrOwnerUnavailable)
}

func (*OllamaRoleAdapter) CheckDisk(context.Context, string, Stage, Request, []Evidence) (Evidence, error) {
	return Evidence{}, fmt.Errorf("LM6 optimizer disk admission owner is unavailable; no disk proof will be synthesized: %w", ErrOwnerUnavailable)
}

func (*OllamaRoleAdapter) Promote(context.Context, string, Request, []Evidence) (Evidence, error) {
	return Evidence{}, fmt.Errorf("LM6 restart-safe role promotion owner is unavailable: %w", ErrOwnerUnavailable)
}

func (*OllamaRoleAdapter) Cleanup(context.Context, string, Request, []Evidence) (Evidence, error) {
	return Evidence{}, fmt.Errorf("LM6 cross-consumer cleanup owner is unavailable: %w", ErrOwnerUnavailable)
}

type ollamaRoleOwnerReceipt struct {
	SchemaVersion string                     `json:"schema_version"`
	ReceiptID     string                     `json:"receipt_id"`
	RequestID     string                     `json:"request_id"`
	Target        string                     `json:"target"`
	Role          string                     `json:"role"`
	Status        string                     `json:"status"`
	PolicySHA256  string                     `json:"policy_sha256,omitempty"`
	Candidates    []ollamaRoleOwnerCandidate `json:"candidates"`
	Refusals      []string                   `json:"refusals,omitempty"`
}

type ollamaRoleOwnerCandidate struct {
	Model          string                   `json:"model"`
	Installed      bool                     `json:"installed"`
	Digest         string                   `json:"digest,omitempty"`
	PolicyEligible bool                     `json:"policy_eligible"`
	Rights         string                   `json:"rights"`
	RightsEvidence ollamaRoleRightsEvidence `json:"rights_evidence"`
	Reason         string                   `json:"reason,omitempty"`
}

type ollamaRoleRightsEvidence struct {
	Status         string `json:"status"`
	DecisionSHA256 string `json:"decision_sha256"`
	EvaluatedAt    string `json:"evaluated_at"`
	ContextID      string `json:"context_id,omitempty"`
	ContextVersion string `json:"context_version,omitempty"`
	ContextSHA256  string `json:"context_sha256,omitempty"`
	ArtifactID     string `json:"artifact_id,omitempty"`
	ArtifactSHA256 string `json:"artifact_sha256,omitempty"`
	PolicySHA256   string `json:"policy_sha256,omitempty"`
}

var _ Adapter = (*OllamaRoleAdapter)(nil)
