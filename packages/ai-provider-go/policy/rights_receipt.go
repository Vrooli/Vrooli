package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const RightsDecisionReceiptSchemaVersion = "rights-decision/v1"

// RightsDecisionReceipt is the canonical, persistable record for one
// model-rights evaluation. It contains no prompt or request body. Its
// self-hash provides stable content identity, not a publisher signature or a
// grant: current facts and trusted context must still be re-read at every
// execution boundary.
type RightsDecisionReceipt struct {
	SchemaVersion       string                   `json:"schema_version"`
	Role                string                   `json:"role"`
	Provider            string                   `json:"provider"`
	Model               string                   `json:"model"`
	Use                 RightsUse                `json:"use"`
	RightsPolicySHA256  string                   `json:"rights_policy_sha256"`
	PolicySchemaVersion string                   `json:"policy_schema_version"`
	ArtifactRights      *ArtifactRights          `json:"artifact_rights,omitempty"`
	RuntimeIdentity     *RuntimeArtifactIdentity `json:"runtime_identity,omitempty"`
	Context             RightsContext            `json:"trusted_context"`
	Decision            RightsDecision           `json:"decision"`
	Eligibility         RightsStatus             `json:"eligibility"`
	ReceiptSHA256       string                   `json:"receipt_sha256"`
}

// RightsDecisionReceiptExpectation is assembled from the current owner policy,
// runtime observation, and trusted context. It must not be populated from a
// request body or from the receipt being checked.
type RightsDecisionReceiptExpectation struct {
	Role                string
	Provider            string
	Model               string
	Use                 RightsUse
	PolicySchemaVersion string
	ArtifactRights      *ArtifactRights
	RuntimeIdentity     *RuntimeArtifactIdentity
	Context             RightsContext
}

type rightsPolicyProjection struct {
	SchemaVersion       string                   `json:"schema_version"`
	Role                string                   `json:"role"`
	Provider            string                   `json:"provider"`
	Model               string                   `json:"model"`
	Use                 RightsUse                `json:"use"`
	PolicySchemaVersion string                   `json:"policy_schema_version"`
	ArtifactRights      *ArtifactRights          `json:"artifact_rights,omitempty"`
	RuntimeIdentity     *RuntimeArtifactIdentity `json:"runtime_identity,omitempty"`
}

type rightsDecisionReceiptContent struct {
	SchemaVersion       string                   `json:"schema_version"`
	Role                string                   `json:"role"`
	Provider            string                   `json:"provider"`
	Model               string                   `json:"model"`
	Use                 RightsUse                `json:"use"`
	RightsPolicySHA256  string                   `json:"rights_policy_sha256"`
	PolicySchemaVersion string                   `json:"policy_schema_version"`
	ArtifactRights      *ArtifactRights          `json:"artifact_rights,omitempty"`
	RuntimeIdentity     *RuntimeArtifactIdentity `json:"runtime_identity,omitempty"`
	Context             RightsContext            `json:"trusted_context"`
	Decision            RightsDecision           `json:"decision"`
	Eligibility         RightsStatus             `json:"eligibility"`
}

// NewRightsDecisionReceipt computes the rights decision from the supplied
// owner facts and trusted context, derives fail-closed eligibility, and binds
// the result to the rights-policy projection. evaluatedAt is normally the
// timestamp carried from planning to execution.
func NewRightsDecisionReceipt(expected RightsDecisionReceiptExpectation, evaluatedAt time.Time) RightsDecisionReceipt {
	artifact := cloneArtifactRights(expected.ArtifactRights)
	runtimeIdentity := cloneRuntimeIdentity(expected.RuntimeIdentity)
	r := RightsDecisionReceipt{
		SchemaVersion: RightsDecisionReceiptSchemaVersion,
		Role:          strings.TrimSpace(expected.Role), Provider: strings.TrimSpace(expected.Provider),
		Model: strings.TrimSpace(expected.Model), Use: expected.Use,
		PolicySchemaVersion: strings.TrimSpace(expected.PolicySchemaVersion),
		ArtifactRights:      artifact, RuntimeIdentity: runtimeIdentity,
		Context: expected.Context,
	}
	r.RightsPolicySHA256 = rightsPolicyProjectionSHA256(r.rightsPolicyProjection())
	r.Decision = EvaluateRights(r.ArtifactRights, r.Use, r.Context, evaluatedAt)
	r.Eligibility = rightsReceiptEligibility(r)
	r.ReceiptSHA256 = r.canonicalSHA256()
	return r
}

func (r RightsDecisionReceipt) rightsPolicyProjection() rightsPolicyProjection {
	return rightsPolicyProjection{
		SchemaVersion: RightsDecisionReceiptSchemaVersion,
		Role:          r.Role, Provider: r.Provider, Model: r.Model, Use: r.Use,
		PolicySchemaVersion: r.PolicySchemaVersion,
		ArtifactRights:      r.ArtifactRights, RuntimeIdentity: r.RuntimeIdentity,
	}
}

func rightsPolicyProjectionSHA256(projection rightsPolicyProjection) string {
	return canonicalDigest(projection)
}

func (r RightsDecisionReceipt) receiptContent() rightsDecisionReceiptContent {
	return rightsDecisionReceiptContent{
		SchemaVersion: r.SchemaVersion, Role: r.Role, Provider: r.Provider,
		Model: r.Model, Use: r.Use, RightsPolicySHA256: r.RightsPolicySHA256,
		PolicySchemaVersion: r.PolicySchemaVersion,
		ArtifactRights:      r.ArtifactRights, RuntimeIdentity: r.RuntimeIdentity,
		Context: r.Context, Decision: r.Decision, Eligibility: r.Eligibility,
	}
}

func (r RightsDecisionReceipt) canonicalSHA256() string {
	return canonicalDigest(r.receiptContent())
}

// MarshalCanonical returns the stable persisted representation after checking
// the receipt's own decision and content hashes.
func (r RightsDecisionReceipt) MarshalCanonical() ([]byte, error) {
	if err := VerifyRightsDecisionReceiptSelf(r); err != nil {
		return nil, err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// ParseRightsDecisionReceipt parses and validates one canonical persisted
// receipt. Call VerifyRightsDecisionReceipt with fresh owner inputs before
// treating it as current, and RequireAllowedRightsDecisionReceipt before use.
func ParseRightsDecisionReceipt(data []byte) (RightsDecisionReceipt, error) {
	var receipt RightsDecisionReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return RightsDecisionReceipt{}, err
	}
	if err := VerifyRightsDecisionReceiptSelf(receipt); err != nil {
		return RightsDecisionReceipt{}, err
	}
	return receipt, nil
}

// VerifyRightsDecisionReceiptSelf validates the versioned record, its
// self-hash, embedded rights decision, and derived eligibility. This verifies
// internal consistency only; it does not establish the authenticity of any
// included source or context.
func VerifyRightsDecisionReceiptSelf(receipt RightsDecisionReceipt) error {
	if receipt.SchemaVersion != RightsDecisionReceiptSchemaVersion || strings.TrimSpace(receipt.Role) == "" ||
		strings.TrimSpace(receipt.Provider) == "" || strings.TrimSpace(receipt.Model) == "" ||
		strings.TrimSpace(receipt.PolicySchemaVersion) == "" || !validRightsUse(receipt.Use) {
		return errors.New("rights decision receipt identity or schema is incomplete")
	}
	if !isSHA256(receipt.RightsPolicySHA256) || !isSHA256(receipt.ReceiptSHA256) {
		return errors.New("rights decision receipt digest is invalid")
	}
	if err := VerifyRightsDecision(receipt.Decision); err != nil {
		return fmt.Errorf("rights decision receipt decision is invalid: %w", err)
	}
	evaluatedAt, err := time.Parse(time.RFC3339Nano, receipt.Decision.EvaluatedAt)
	if err != nil {
		return errors.New("rights decision receipt evaluation time is invalid")
	}
	decision := EvaluateRights(receipt.ArtifactRights, receipt.Use, receipt.Context, evaluatedAt)
	if decision.DecisionSHA256 != receipt.Decision.DecisionSHA256 {
		return errors.New("rights decision receipt does not match its embedded facts and context")
	}
	if want := rightsPolicyProjectionSHA256(receipt.rightsPolicyProjection()); want != receipt.RightsPolicySHA256 {
		return errors.New("rights decision receipt policy projection is inconsistent")
	}
	if want := rightsReceiptEligibility(receipt); want != receipt.Eligibility {
		return errors.New("rights decision receipt eligibility is inconsistent")
	}
	if want := receipt.canonicalSHA256(); want != receipt.ReceiptSHA256 {
		return errors.New("rights decision receipt content hash does not match")
	}
	return nil
}

// VerifyRightsDecisionReceipt compares a persisted receipt with current
// owner-controlled facts. It succeeds for a consistent unknown/refusal record;
// use RequireAllowedRightsDecisionReceipt separately before execution.
func VerifyRightsDecisionReceipt(receipt RightsDecisionReceipt, current RightsDecisionReceiptExpectation) error {
	if err := VerifyRightsDecisionReceiptSelf(receipt); err != nil {
		return err
	}
	if receipt.Role != strings.TrimSpace(current.Role) || receipt.Provider != strings.TrimSpace(current.Provider) ||
		receipt.Model != strings.TrimSpace(current.Model) || receipt.Use != current.Use ||
		receipt.PolicySchemaVersion != strings.TrimSpace(current.PolicySchemaVersion) ||
		!reflect.DeepEqual(receipt.ArtifactRights, current.ArtifactRights) ||
		!reflect.DeepEqual(receipt.RuntimeIdentity, current.RuntimeIdentity) ||
		!reflect.DeepEqual(receipt.Context, current.Context) {
		return errors.New("rights decision receipt differs from current owner facts or trusted context")
	}
	evaluatedAt, err := time.Parse(time.RFC3339Nano, receipt.Decision.EvaluatedAt)
	if err != nil {
		return errors.New("rights decision receipt evaluation time is invalid")
	}
	expected := NewRightsDecisionReceipt(current, evaluatedAt)
	if receipt.RightsPolicySHA256 != expected.RightsPolicySHA256 || receipt.Decision.DecisionSHA256 != expected.Decision.DecisionSHA256 || receipt.ReceiptSHA256 != expected.ReceiptSHA256 {
		return errors.New("rights decision receipt is stale for the current owner facts or trusted context")
	}
	return nil
}

// RequireAllowedRightsDecisionReceipt applies the execution gate after
// receipt consistency/currentness has been verified.
func RequireAllowedRightsDecisionReceipt(receipt RightsDecisionReceipt) error {
	if err := VerifyRightsDecisionReceiptSelf(receipt); err != nil {
		return err
	}
	if receipt.Eligibility != RightsAllowed || receipt.Decision.Status != RightsAllowed {
		return fmt.Errorf("rights decision receipt is not eligible (%s: %s)", receipt.Decision.Status, receipt.Decision.Reason)
	}
	return nil
}

func rightsReceiptEligibility(receipt RightsDecisionReceipt) RightsStatus {
	if receipt.Decision.Status == RightsDenied {
		return RightsDenied
	}
	if receipt.Decision.Status != RightsAllowed || !isSHA256(receipt.RightsPolicySHA256) || receipt.ArtifactRights == nil ||
		!RuntimeArtifactIdentityMatches(receipt.RuntimeIdentity, receipt.ArtifactRights, receipt.Provider, receipt.Model, receipt.RuntimeIdentityDigest()) {
		return RightsUnknown
	}
	return RightsAllowed
}

func (r RightsDecisionReceipt) RuntimeIdentityDigest() string {
	if r.RuntimeIdentity == nil {
		return ""
	}
	return r.RuntimeIdentity.Digest
}

func cloneArtifactRights(value *ArtifactRights) *ArtifactRights {
	if value == nil {
		return nil
	}
	copy := *value
	copy.Conditions = append([]RightsCondition(nil), value.Conditions...)
	copy.Obligations = append([]string(nil), value.Obligations...)
	copy.Lineage = append([]ArtifactComponent(nil), value.Lineage...)
	return &copy
}

func cloneRuntimeIdentity(value *RuntimeArtifactIdentity) *RuntimeArtifactIdentity {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
