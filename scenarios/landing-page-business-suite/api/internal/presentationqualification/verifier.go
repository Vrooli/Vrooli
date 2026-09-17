package presentationqualification

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	releasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases/releasesv1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
	"google.golang.org/protobuf/proto"
	"landing-page-business-suite-api/internal/presentation"
)

var ErrUnqualified = errors.New("presentation capability is not publication-qualified")

// AssetPublicationVerifier is intentionally structural. The existing
// presentationassets.Verifier remains the owner of released asset bytes;
// capability qualification does not duplicate that authority. The second
// document is the active published revision (nil when none), which lets the
// asset owner inherit qualification for assets unchanged since that revision.
type AssetPublicationVerifier interface {
	VerifyPublication(context.Context, presentation.Document, *presentation.Document) error
}

// ValidationReader is the read-only Test Genie authority for terminal
// validation receipts. Implementations must read by the exact receipt ID.
type ValidationReader interface {
	ReadValidation(context.Context, string) (*validationv1.ValidationReceipt, error)
}

// ReleaseReader is the read-only Deployment Manager authority for release
// identity. It is deliberately not a delivery/published-status interface.
type ReleaseReader interface {
	ReadRelease(context.Context, string) (*releasesv1.ReleaseView, error)
}

// CapabilityBinding is server-owned declarative membership data. The
// presentation document cannot establish that a receipt belongs to one
// capability because neither owner schema carries that relationship.
// ValidationIntentID and RequiredEvidence must point at existing owner test
// definitions/evidence selectors; they are not filesystem paths or claims
// supplied by the document.
type CapabilityBinding struct {
	AppKey             string
	CapabilityID       string
	Owner              string
	ReleaseProfileID   string
	ScenarioID         string
	ValidationIntentID string
	ValidationRef      string
	ReleaseRef         string
	RequiredEvidence   []EvidenceSelector
}

// EvidenceSelector names one required Test Genie evidence reference. All
// fields that are present must match exactly; no wildcard selectors are
// accepted. At least one selector is required for every available capability.
type EvidenceSelector struct {
	EvidenceID string
	Kind       string
	Owner      string
	SubjectID  string
	Digest     string
}

// BindingResolver supplies immutable-in-process, server-owned mappings. It
// has no persistence or mutation path and must fail closed for an unknown
// app/capability pair.
type BindingResolver interface {
	ResolveBinding(context.Context, string, string) (CapabilityBinding, error)
}

// StaticBindings is a convenient adapter for a server-owned declarative map.
// NewStaticBindings copies the map and selector slices; it is not a receipt
// registry and does not import, write, or trust owner evidence.
type StaticBindings struct{ values map[string]CapabilityBinding }

func NewStaticBindings(values map[string]CapabilityBinding) *StaticBindings {
	copyValues := make(map[string]CapabilityBinding, len(values))
	for key, binding := range values {
		binding.RequiredEvidence = append([]EvidenceSelector(nil), binding.RequiredEvidence...)
		copyValues[key] = binding
	}
	return &StaticBindings{values: copyValues}
}

func (m *StaticBindings) ResolveBinding(_ context.Context, appKey, capabilityID string) (CapabilityBinding, error) {
	if m == nil {
		return CapabilityBinding{}, fmt.Errorf("%w: capability binding map is unavailable", ErrUnqualified)
	}
	binding, ok := m.values[bindingKey(appKey, capabilityID)]
	if !ok {
		return CapabilityBinding{}, fmt.Errorf("%w: no server-owned mapping for %s/%s", ErrUnqualified, appKey, capabilityID)
	}
	binding.RequiredEvidence = append([]EvidenceSelector(nil), binding.RequiredEvidence...)
	return binding, nil
}

// TestGenieValidationReader adapts the generated Test Genie ValidationService
// client. The Connect procedure is
// /vrooli.test_genie.v1.validation.ValidationService/GetValidation.
type TestGenieValidationReader struct {
	Client validationconnect.ValidationServiceClient
}

func NewTestGenieValidationReader(client validationconnect.ValidationServiceClient) (*TestGenieValidationReader, error) {
	if client == nil {
		return nil, errors.New("test-genie validation client is required")
	}
	return &TestGenieValidationReader{Client: client}, nil
}

func (r *TestGenieValidationReader) ReadValidation(ctx context.Context, id string) (*validationv1.ValidationReceipt, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("test-genie validation reader is unavailable")
	}
	response, err := r.Client.GetValidation(ctx, connect.NewRequest(&validationv1.GetValidationRequest{ReceiptId: id}))
	if err != nil {
		return nil, fmt.Errorf("read Test Genie validation receipt %q: %w", id, err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetReceipt() == nil {
		return nil, fmt.Errorf("read Test Genie validation receipt %q: empty receipt", id)
	}
	receipt := response.Msg.GetReceipt()
	if receipt.GetReceiptId() != id {
		return nil, fmt.Errorf("read Test Genie validation receipt %q: owner returned receipt %q", id, receipt.GetReceiptId())
	}
	return proto.Clone(receipt).(*validationv1.ValidationReceipt), nil
}

// DeploymentManagerReleaseReader adapts the generated Deployment Manager
// ReleasesService client. The Connect procedure is
// /vrooli.deployment_manager.v1.releases.ReleasesService/Get.
type DeploymentManagerReleaseReader struct {
	Client releasesconnect.ReleasesServiceClient
}

func NewDeploymentManagerReleaseReader(client releasesconnect.ReleasesServiceClient) (*DeploymentManagerReleaseReader, error) {
	if client == nil {
		return nil, errors.New("deployment-manager releases client is required")
	}
	return &DeploymentManagerReleaseReader{Client: client}, nil
}

func (r *DeploymentManagerReleaseReader) ReadRelease(ctx context.Context, id string) (*releasesv1.ReleaseView, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("deployment-manager release reader is unavailable")
	}
	response, err := r.Client.Get(ctx, connect.NewRequest(&releasesv1.GetReleaseRequest{ReleaseId: id}))
	if err != nil {
		return nil, fmt.Errorf("read Deployment Manager release %q: %w", id, err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetRelease() == nil {
		return nil, fmt.Errorf("read Deployment Manager release %q: empty release", id)
	}
	return proto.Clone(response.Msg.GetRelease()).(*releasesv1.ReleaseView), nil
}

type Verifier struct {
	Validations ValidationReader
	Releases    ReleaseReader
	Bindings    BindingResolver
	Assets      AssetPublicationVerifier
}

func NewVerifier(validations ValidationReader, releases ReleaseReader, bindings BindingResolver, assets AssetPublicationVerifier) (*Verifier, error) {
	return &Verifier{Validations: validations, Releases: releases, Bindings: bindings, Assets: assets}, nil
}

// VerifyPublication validates the document, then independently qualifies every
// available capability from owner readbacks. OwnerQualification.Qualified is
// ignored as authority. Delivery availability, deployment status, and asset
// publication are separate concerns.
func (v *Verifier) VerifyPublication(ctx context.Context, document presentation.Document, active *presentation.Document) error {
	if v == nil {
		return fmt.Errorf("%w: verifier is unavailable", ErrUnqualified)
	}
	if err := presentation.Validate(document); err != nil {
		return fmt.Errorf("%w: document validation: %v", ErrUnqualified, err)
	}
	views, err := presentation.PublicViews(document)
	if err != nil {
		return fmt.Errorf("%w: enumerate public views: %v", ErrUnqualified, err)
	}
	visibleCapabilities := publicCapabilityIDs(views)
	for _, app := range document.Apps {
		if !publicApp(app) {
			continue
		}
		for _, capability := range app.Capabilities {
			if capability.Status != presentation.CapabilityAvailable || !visibleCapabilities[capability.ID] {
				continue
			}
			if err := v.verifyCapability(ctx, app.Key, capability); err != nil {
				return fmt.Errorf("%w: app %q capability %q: %v", ErrUnqualified, app.Key, capability.ID, err)
			}
		}
	}
	if publicViewsHaveAssets(views) && v.Assets == nil {
		return fmt.Errorf("%w: public views contain assets but no asset verifier is configured", ErrUnqualified)
	}
	if v.Assets != nil && publicViewsHaveAssets(views) {
		if err := v.Assets.VerifyPublication(ctx, document, active); err != nil {
			return fmt.Errorf("%w: asset publication: %v", ErrUnqualified, err)
		}
	}
	return nil
}

func (v *Verifier) verifyCapability(ctx context.Context, appKey string, capability presentation.Capability) error {
	qualification := capability.OwnerQualification
	if qualification == nil {
		return fmt.Errorf("missing owner qualification receipt")
	}
	validationID, err := ParseValidationRef(qualification.EvidenceRef)
	if err != nil {
		return err
	}
	releaseID, err := ParseReleaseRef(qualification.ReleaseRef)
	if err != nil {
		return err
	}
	if !contains(capability.EvidenceRefs, qualification.EvidenceRef) {
		return errors.New("owner evidence reference is not declared by the capability")
	}
	if v.Validations == nil || v.Releases == nil || v.Bindings == nil {
		return errors.New("authoritative validation, release, and binding readers are required")
	}
	binding, err := v.Bindings.ResolveBinding(ctx, appKey, capability.ID)
	if err != nil {
		return err
	}
	if err := binding.validate(appKey, capability.ID, qualification.Owner, validationID, releaseID); err != nil {
		return err
	}
	receipt, err := v.Validations.ReadValidation(ctx, validationID)
	if err != nil {
		return err
	}
	if err := verifyValidationReceipt(receipt, validationID, binding); err != nil {
		return err
	}
	release, err := v.Releases.ReadRelease(ctx, releaseID)
	if err != nil {
		return err
	}
	if err := verifyReleaseView(release, binding, qualification.Owner, releaseID, receipt); err != nil {
		return err
	}
	return nil
}

func (b CapabilityBinding) validate(appKey, capabilityID, owner, validationID, releaseID string) error {
	if b.AppKey != appKey || b.CapabilityID != capabilityID {
		return errors.New("server-owned capability binding does not match capability")
	}
	for name, value := range map[string]string{
		"owner": b.Owner, "release_profile_id": b.ReleaseProfileID, "scenario_id": b.ScenarioID,
		"validation_intent_id": b.ValidationIntentID, "validation_ref": b.ValidationRef, "release_ref": b.ReleaseRef,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("server-owned capability binding %s is required", name)
		}
	}
	if b.Owner != owner || b.ValidationRef != validationRef(validationID) || b.ReleaseRef != releaseRef(releaseID) {
		return errors.New("configured owner qualification does not match server-owned capability binding")
	}
	if _, err := ParseValidationRef(b.ValidationRef); err != nil {
		return fmt.Errorf("server-owned capability binding validation_ref: %w", err)
	}
	if _, err := ParseReleaseRef(b.ReleaseRef); err != nil {
		return fmt.Errorf("server-owned capability binding release_ref: %w", err)
	}
	if len(b.RequiredEvidence) == 0 {
		return errors.New("server-owned capability binding requires behavioral evidence selectors")
	}
	for i, selector := range b.RequiredEvidence {
		if err := selector.validate(); err != nil {
			return fmt.Errorf("behavioral evidence selector %d: %w", i, err)
		}
		if selector.SubjectID != b.ScenarioID {
			return errors.New("behavioral evidence selector does not match the server-owned scenario")
		}
	}
	return nil
}

func verifyValidationReceipt(receipt *validationv1.ValidationReceipt, validationID string, binding CapabilityBinding) error {
	if receipt == nil {
		return errors.New("Test Genie returned no validation receipt")
	}
	if receipt.GetReceiptId() != validationID {
		return errors.New("Test Genie receipt ID does not match the requested typed reference")
	}
	if receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || receipt.GetTerminalAt() == nil {
		return fmt.Errorf("Test Genie validation receipt is not a terminal pass")
	}
	if receipt.GetIntentId() != binding.ValidationIntentID {
		return errors.New("Test Genie validation intent does not match server-owned capability binding")
	}
	admitted, observed := receipt.GetAdmittedIdentity(), receipt.GetObservedIdentity()
	if admitted == nil || observed == nil || admitted.GetIdentity() == "" || observed.GetIdentity() == "" || !proto.Equal(admitted, observed) {
		return errors.New("Test Genie admitted and observed source identities are missing or differ")
	}
	if admitted.GetCommit() == "" || observed.GetCommit() == "" || admitted.GetCommit() != observed.GetCommit() {
		return errors.New("Test Genie admitted and observed source commits are missing or differ")
	}
	for _, selector := range binding.RequiredEvidence {
		matched := false
		for _, evidence := range receipt.GetEvidence() {
			if selector.matches(evidence) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("Test Genie receipt lacks required behavioral evidence %q", selector.Kind)
		}
	}
	return nil
}

func verifyReleaseView(release *releasesv1.ReleaseView, binding CapabilityBinding, owner, releaseID string, receipt *validationv1.ValidationReceipt) error {
	if release == nil || release.GetReleaseId() != releaseID || release.GetProfileId() != binding.ReleaseProfileID {
		return errors.New("Deployment Manager release identity does not match server-owned capability binding")
	}
	if release.GetGitCommitHash() == "" || release.GetGitCommitHash() != receipt.GetAdmittedIdentity().GetCommit() || release.GetGitCommitHash() != receipt.GetObservedIdentity().GetCommit() {
		return errors.New("Deployment Manager release source revision does not match Test Genie source identity")
	}
	candidate := release.GetCandidate()
	if candidate == nil || candidate.GetCandidateId() == "" || candidate.GetSourceRevision() != release.GetGitCommitHash() {
		return errors.New("Deployment Manager release lacks an exact candidate/source identity")
	}
	declaration := candidate.GetCapabilityDeclaration()
	if declaration == nil || declaration.GetSupportOwner() != owner || declaration.GetReleaseAuthority() == "" {
		return errors.New("Deployment Manager release lacks owner capability declaration")
	}
	return nil
}

func (s EvidenceSelector) validate() error {
	if strings.TrimSpace(s.EvidenceID) == "" || strings.TrimSpace(s.Kind) == "" || strings.TrimSpace(s.Owner) == "" || strings.TrimSpace(s.SubjectID) == "" {
		return errors.New("evidence_id, kind, owner, and subject_id are required")
	}
	for name, value := range map[string]string{"evidence_id": s.EvidenceID, "kind": s.Kind, "owner": s.Owner, "subject_id": s.SubjectID, "digest": s.Digest} {
		if strings.ContainsAny(value, "/\\") || strings.Contains(value, "..") || strings.ContainsAny(value, " \t\r\n") {
			return fmt.Errorf("%s must be an opaque selector value", name)
		}
	}
	return nil
}

func (s EvidenceSelector) matches(evidence *validationv1.EvidenceReference) bool {
	return evidence != nil && evidence.GetKind() == s.Kind && evidence.GetOwner() == s.Owner && evidence.GetSubjectId() == s.SubjectID && (s.EvidenceID == "" || evidence.GetEvidenceId() == s.EvidenceID) && (s.Digest == "" || evidence.GetDigest() == s.Digest)
}

func bindingKey(appKey, capabilityID string) string { return appKey + "\x00" + capabilityID }
func validationRef(id string) string                { return testGenieValidationPrefix + id }
func releaseRef(id string) string                   { return deploymentManagerReleasePrefix + id }
func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func publicApp(app presentation.App) bool {
	return app.Enabled && app.Visibility == presentation.VisibilityPublic && app.Publication == presentation.PublicationPublished
}

func publicCapabilityIDs(views []presentation.ResolveResult) map[string]bool {
	ids := make(map[string]bool)
	for _, view := range views {
		for _, capability := range view.Capabilities {
			ids[capability.ID] = true
		}
	}
	return ids
}

func publicViewsHaveAssets(views []presentation.ResolveResult) bool {
	for _, view := range views {
		if len(view.Assets) > 0 {
			return true
		}
	}
	return false
}
