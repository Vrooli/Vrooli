// Package optimize defines the resource-owned target adapter contract used by
// LM6. It deliberately owns only Ollama facts and refuses lifecycle effects
// whose safety cannot yet be proved by this resource.
package optimize

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

const (
	RoleChatDefault = "chat.default"
	MinFixedCases   = 20
	CandidateCount  = 3
)

// RoleRequest is the stable callable request shape for the first LM6 target.
// EvaluationReceiptID is an optional opaque legacy reference on this
// inspection-only request. Inspect never verifies it or treats it as evaluation
// evidence; a later evaluation-verification operation must require an owner
// receipt after Evaluate completes.
type RoleRequest struct {
	RequestID           string   `json:"request_id"`
	Target              string   `json:"target"`
	Role                string   `json:"role"`
	CandidateModels     []string `json:"candidate_models"`
	FixedSuiteID        string   `json:"fixed_suite_id"`
	FixedCaseIDs        []string `json:"fixed_case_ids"`
	EvaluationReceiptID string   `json:"evaluation_receipt_id"`
	WinnerModel         string   `json:"winner_model,omitempty"`
}

// Validate rejects ambiguous or under-sized requests before any owner is
// called. It does not claim that a referenced evaluation receipt is valid.
func (r RoleRequest) Validate() error {
	if strings.TrimSpace(r.RequestID) == "" {
		return errors.New("request_id is required")
	}
	if r.Target != "ollama-role" || strings.TrimSpace(r.Role) == "" {
		return fmt.Errorf("unsupported Ollama optimization target/role %q/%q", r.Target, r.Role)
	}
	if len(r.CandidateModels) != CandidateCount {
		return fmt.Errorf("exactly %d candidate_models are required", CandidateCount)
	}
	seenModels := make(map[string]struct{}, len(r.CandidateModels))
	for _, model := range r.CandidateModels {
		model = strings.TrimSpace(model)
		if model == "" {
			return errors.New("candidate model names must not be empty")
		}
		if _, ok := seenModels[model]; ok {
			return fmt.Errorf("candidate model %q is duplicated", model)
		}
		seenModels[model] = struct{}{}
	}
	if strings.TrimSpace(r.FixedSuiteID) == "" {
		return errors.New("fixed_suite_id is required")
	}
	if len(r.FixedCaseIDs) < MinFixedCases {
		return fmt.Errorf("at least %d fixed_case_ids are required", MinFixedCases)
	}
	seenCases := make(map[string]struct{}, len(r.FixedCaseIDs))
	for _, id := range r.FixedCaseIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			return errors.New("fixed case IDs must not be empty")
		}
		if _, ok := seenCases[id]; ok {
			return fmt.Errorf("fixed case %q is duplicated", id)
		}
		seenCases[id] = struct{}{}
	}
	if r.WinnerModel != "" {
		if _, ok := seenModels[strings.TrimSpace(r.WinnerModel)]; !ok {
			return errors.New("winner_model must be one of the requested candidates")
		}
	}
	return nil
}

// CandidateCheck records only facts observed by the Ollama resource owner.
type CandidateCheck struct {
	Model               string         `json:"model"`
	Installed           bool           `json:"installed"`
	Digest              string         `json:"digest,omitempty"`
	SizeBytes           int64          `json:"size_bytes,omitempty"`
	Capabilities        []string       `json:"capabilities,omitempty"`
	MissingCapabilities []string       `json:"missing_capabilities,omitempty"`
	PolicyEligible      bool           `json:"policy_eligible"`
	Rights              string         `json:"rights"`
	RightsEvidence      RightsEvidence `json:"rights_evidence"`
	Reason              string         `json:"reason,omitempty"`
}

// RightsEvidence records a current source-rights evaluation, independent of
// whether the corresponding Ollama runtime artifact is installed.
type RightsEvidence struct {
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

// Receipt is the durable response contract for the role adapter. Status is
// refused until evaluation, rights, disk admission, promotion, and cleanup
// are all owner-verified. Refusal reasons name the missing owner boundary.
type Receipt struct {
	SchemaVersion       string           `json:"schema_version"`
	ReceiptID           string           `json:"receipt_id"`
	RequestID           string           `json:"request_id"`
	Target              string           `json:"target"`
	Role                string           `json:"role"`
	Status              string           `json:"status"`
	CreatedAt           string           `json:"created_at"`
	PolicySHA256        string           `json:"policy_sha256,omitempty"`
	Candidates          []CandidateCheck `json:"candidates,omitempty"`
	EvaluationReceiptID string           `json:"evaluation_receipt_id,omitempty"`
	EvaluationVerified  bool             `json:"evaluation_verified"`
	DiskAdmission       string           `json:"disk_admission"`
	Promotion           string           `json:"promotion"`
	Cleanup             string           `json:"cleanup"`
	Refusals            []string         `json:"refusals,omitempty"`
}

// Inspect uses the Ollama model inventory and policy/rights owners to attest
// candidate facts without acquiring, loading, or evaluating models. The
// lifecycle remains explicitly refused until evaluation, shared capacity,
// promotion, and consumer-reference owners are connected.
func Inspect(ctx context.Context, request RoleRequest, inventory []ensure.ModelInfo, p policy.Policy, getenv func(string) string) (Receipt, error) {
	return inspectAt(ctx, request, inventory, p, getenv, time.Now().UTC())
}

func inspectAt(ctx context.Context, request RoleRequest, inventory []ensure.ModelInfo, p policy.Policy, getenv func(string) string, now time.Time) (Receipt, error) {
	if err := request.Validate(); err != nil {
		return NewRefusal(request, "invalid request: "+err.Error()), err
	}
	receipt := NewRefusal(request,
		"evaluation receipt is not verified by an owner API",
		"shared disk admission receipt is not attached",
		"role promotion has no restart-safe policy and consumer transaction",
		"candidate cleanup has no complete cross-consumer reference proof",
	)
	receipt.PolicySHA256 = p.LoadedSHA256()
	rolePolicy, roleExists := p.Roles[request.Role]
	if !roleExists {
		receipt.Refusals = append(receipt.Refusals, fmt.Sprintf("role %q is not present in the authoritative Ollama model policy", request.Role))
		return receipt, nil
	}
	byName := make(map[string]ensure.ModelInfo, len(inventory))
	for _, item := range inventory {
		byName[item.Name] = item
	}
	receipt.Candidates = make([]CandidateCheck, 0, len(request.CandidateModels))
	trustedContext, contextErr := policy.LoadDefaultRightsContext(getenv)
	if contextErr != nil {
		trustedContext = sharedpolicy.RightsContext{}
	}
	for _, name := range request.CandidateModels {
		modelPolicy, modelKnown := p.Models[name]
		decision, decisionErr := p.EvaluateModelRights(name, sharedpolicy.RightsUseLocalInference, trustedContext, now)
		if decisionErr != nil {
			decision = sharedpolicy.EvaluateRights(nil, sharedpolicy.RightsUseLocalInference, trustedContext, now)
		}
		status := string(decision.Status)
		check := CandidateCheck{Model: name, Rights: "refused", RightsEvidence: RightsEvidence{
			Status: status, DecisionSHA256: decision.DecisionSHA256, EvaluatedAt: decision.EvaluatedAt,
			ContextID: decision.ContextID, ContextVersion: decision.ContextVersion, ContextSHA256: decision.ContextSHA256,
			ArtifactID: decision.ArtifactID, ArtifactSHA256: decision.ArtifactSHA256, PolicySHA256: p.LoadedSHA256(),
		}}
		if !modelKnown {
			check.Reason = "candidate has no source artifact-rights entry in authoritative Ollama policy"
			receipt.Candidates = append(receipt.Candidates, check)
			continue
		}
		if modelPolicy.Rights == nil {
			check.Reason = "source artifact-rights facts are unknown"
			receipt.Candidates = append(receipt.Candidates, check)
			continue
		}
		if contextErr != nil {
			check.Reason = "trusted rights context unavailable: " + contextErr.Error()
			receipt.Candidates = append(receipt.Candidates, check)
			continue
		}
		if decisionErr != nil {
			check.Reason = decisionErr.Error()
			receipt.Candidates = append(receipt.Candidates, check)
			continue
		}
		if status == string(sharedpolicy.RightsAllowed) {
			check.Rights = "allowed"
		} else {
			check.Rights = "refused"
		}
		if status != string(sharedpolicy.RightsAllowed) {
			check.Reason = decision.Reason
		}
		info, exists := byName[name]
		if !exists {
			if check.Reason == "" {
				check.Reason = "candidate is not installed; acquisition must pass resource-ollama ensure and shared disk admission"
			}
			receipt.Candidates = append(receipt.Candidates, check)
			continue
		}
		check.Installed = true
		check.Digest = info.Digest
		check.SizeBytes = info.Size
		check.Capabilities = append([]string(nil), modelPolicy.Capabilities...)
		check.MissingCapabilities = missingCapabilities(rolePolicy.RequiredCapabilities, modelPolicy.Capabilities)
		check.PolicyEligible = modelPolicy.DefaultEligible && modelPolicy.Rights != nil && len(check.MissingCapabilities) == 0
		if !check.PolicyEligible {
			check.Reason = "candidate lacks an eligible model policy entry, rights facts, or capabilities required by the target role"
			receipt.Candidates = append(receipt.Candidates, check)
			continue
		}
		if strings.TrimSpace(info.Digest) == "" {
			check.Reason = "Ollama inventory omitted the candidate content digest"
			receipt.Candidates = append(receipt.Candidates, check)
			continue
		}
		receipt.Candidates = append(receipt.Candidates, check)
	}
	if ctx.Err() != nil {
		return receipt, ctx.Err()
	}
	return receipt, nil
}

func missingCapabilities(required, available []string) []string {
	set := make(map[string]struct{}, len(available))
	for _, capability := range available {
		set[strings.TrimSpace(capability)] = struct{}{}
	}
	missing := make([]string, 0)
	for _, capability := range required {
		capability = strings.TrimSpace(capability)
		if capability == "" {
			continue
		}
		if _, ok := set[capability]; !ok {
			missing = append(missing, capability)
		}
	}
	return missing
}

// NewRefusal creates a stable, explicit refusal receipt. It is useful even
// when an owner dependency is unavailable, so callers can persist why the
// unattended operation did not proceed.
func NewRefusal(request RoleRequest, reasons ...string) Receipt {
	return Receipt{
		SchemaVersion:       "ollama.optimize.receipt.v1",
		ReceiptID:           request.RequestID + ":resource-ollama",
		RequestID:           request.RequestID,
		Target:              request.Target,
		Role:                request.Role,
		Status:              "refused",
		CreatedAt:           time.Now().UTC().Format(time.RFC3339Nano),
		EvaluationReceiptID: request.EvaluationReceiptID,
		DiskAdmission:       "not-admitted",
		Promotion:           "refused",
		Cleanup:             "refused",
		Refusals:            append([]string(nil), reasons...),
	}
}
