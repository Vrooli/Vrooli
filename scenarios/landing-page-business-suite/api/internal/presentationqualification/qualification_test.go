package presentationqualification

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"landing-page-business-suite-api/internal/presentation"
	"landing-page-business-suite-api/internal/presentationseed"
)

type fakeValidationReader struct {
	receipt *validationv1.ValidationReceipt
	err     error
}

func (f fakeValidationReader) ReadValidation(context.Context, string) (*validationv1.ValidationReceipt, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.receipt, nil
}

type fakeReleaseReader struct {
	release *releasesv1.ReleaseView
	err     error
}

func (f fakeReleaseReader) ReadRelease(context.Context, string) (*releasesv1.ReleaseView, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.release, nil
}

type fakeAssetVerifier struct{ calls int }

func (f *fakeAssetVerifier) VerifyPublication(context.Context, presentation.Document) error {
	f.calls++
	return nil
}

func TestTypedReferencesRejectUntypedAndPathValues(t *testing.T) {
	validationID, err := ParseValidationRef("test-genie:validation:receipt-1")
	require.NoError(t, err)
	require.Equal(t, "receipt-1", validationID)
	releaseID, err := ParseReleaseRef("deployment-manager:release:release-1")
	require.NoError(t, err)
	require.Equal(t, "release-1", releaseID)
	for _, value := range []string{
		"receipt-1", "test-genie:validation:", "test-genie:validation:../receipt", "test-genie:validation:receipt/1",
		"deployment-manager:release:release/1", "deployment-manager:release:release:1",
	} {
		if strings.HasPrefix(value, testGenieValidationPrefix) {
			_, err = ParseValidationRef(value)
		} else {
			_, err = ParseReleaseRef(value)
		}
		require.Error(t, err, value)
	}
}

func TestAvailableCapabilityRequiresAuthoritativeMatchingEvidence(t *testing.T) {
	document := seedDocument(t)
	app := &document.Apps[0]
	app.Publication = presentation.PublicationPublished
	capability := &app.Capabilities[0]
	owner := "aquila-owner"
	validationID := "receipt-aquila-sessions"
	releaseID := "release-aquila-sessions"
	validationRef := validationRef(validationID)
	releaseRef := releaseRef(releaseID)
	capability.Status = presentation.CapabilityAvailable
	capability.EvidenceRefs = append(capability.EvidenceRefs, validationRef)
	capability.OwnerQualification = &presentation.OwnerQualification{Owner: owner, EvidenceRef: validationRef, ReleaseRef: releaseRef, Qualified: true}

	receipt := &validationv1.ValidationReceipt{
		ReceiptId: validationID, IntentId: "intent-aquila-sessions",
		State:            validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED,
		AdmittedIdentity: &validationv1.SourceIdentity{Identity: "source-1", Commit: "commit-aquila"},
		ObservedIdentity: &validationv1.SourceIdentity{Identity: "source-1", Commit: "commit-aquila"},
		Evidence:         []*validationv1.EvidenceReference{{EvidenceId: "behavior-aquila-sessions", Kind: "capability-behavior", Owner: "test-genie", SubjectId: "web-console"}},
		TerminalAt:       timestamppb.Now(),
	}
	release := &releasesv1.ReleaseView{
		ReleaseId: releaseID, ProfileId: "aquila-profile", GitCommitHash: "commit-aquila",
		Candidate: &releasesv1.Candidate{
			CandidateId: "candidate-aquila-sessions", SourceRevision: "commit-aquila",
			CapabilityDeclaration: &releasesv1.CapabilityDeclaration{SupportOwner: owner, ReleaseAuthority: "aquila-release-authority"},
		},
	}
	binding := CapabilityBinding{
		AppKey: app.Key, CapabilityID: capability.ID, Owner: owner, ReleaseProfileID: "aquila-profile",
		ScenarioID: "web-console", ValidationIntentID: receipt.IntentId, ValidationRef: validationRef, ReleaseRef: releaseRef,
		RequiredEvidence: []EvidenceSelector{{EvidenceID: "behavior-aquila-sessions", Kind: "capability-behavior", Owner: "test-genie", SubjectID: "web-console"}},
	}
	verifier, err := NewVerifier(fakeValidationReader{receipt: receipt}, fakeReleaseReader{release: release}, NewStaticBindings(map[string]CapabilityBinding{bindingKey(app.Key, capability.ID): binding}), &fakeAssetVerifier{})
	require.NoError(t, err)
	require.NoError(t, verifier.VerifyPublication(context.Background(), document))

	for name, mutate := range map[string]func(*validationv1.ValidationReceipt){
		"wrong intent":     func(value *validationv1.ValidationReceipt) { value.IntentId = "other-intent" },
		"wrong receipt id": func(value *validationv1.ValidationReceipt) { value.ReceiptId = "other-receipt" },
		"wrong evidence":   func(value *validationv1.ValidationReceipt) { value.Evidence[0].EvidenceId = "other-evidence" },
		"changed source":   func(value *validationv1.ValidationReceipt) { value.ObservedIdentity.Commit = "other-commit" },
		"not terminal pass": func(value *validationv1.ValidationReceipt) {
			value.State = validationv1.ReceiptState_RECEIPT_STATE_FAILED
		},
	} {
		t.Run(name, func(t *testing.T) {
			copy := proto.Clone(receipt).(*validationv1.ValidationReceipt)
			mutate(copy)
			bad := *verifier
			bad.Validations = fakeValidationReader{receipt: copy}
			require.ErrorIs(t, bad.VerifyPublication(context.Background(), document), ErrUnqualified)
		})
	}
}

func TestAvailableCapabilityFailsClosedWithoutServerBindingOrReaders(t *testing.T) {
	document := seedDocument(t)
	privateApp := &document.Apps[1]
	privateApp.Capabilities = []presentation.Capability{{
		ID: "private-capability", Label: "Private capability", Benefits: []string{"Private benefit"}, Status: presentation.CapabilityAvailable,
		EvidenceRefs: []string{"private-owner-ref"}, OwnerQualification: &presentation.OwnerQualification{Owner: "private-owner", EvidenceRef: "private-owner-ref", ReleaseRef: "private-release", Qualified: true},
	}}
	verifier, err := NewVerifier(fakeValidationReader{err: errors.New("private capability must not trigger Test Genie read")}, fakeReleaseReader{err: errors.New("private capability must not trigger Deployment Manager read")}, nil, nil)
	require.NoError(t, err)
	err = verifier.VerifyPublication(context.Background(), document)
	require.NoError(t, err)
}

func TestPublicAssetsRequireAssetVerifier(t *testing.T) {
	document := seedDocument(t)
	document.Apps[0].Publication = presentation.PublicationPublished
	verifier, err := NewVerifier(nil, nil, nil, nil)
	require.NoError(t, err)
	err = verifier.VerifyPublication(context.Background(), document)
	require.ErrorIs(t, err, ErrUnqualified)
	require.Contains(t, err.Error(), "asset verifier")
}

func TestAvailableCapabilityRequiresReaders(t *testing.T) {
	document := seedDocument(t)
	document.Apps[0].Publication = presentation.PublicationPublished
	capability := &document.Apps[0].Capabilities[0]
	capability.Status = presentation.CapabilityAvailable
	capability.EvidenceRefs = append(capability.EvidenceRefs, validationRef("receipt-1"))
	capability.OwnerQualification = &presentation.OwnerQualification{Owner: "owner", EvidenceRef: validationRef("receipt-1"), ReleaseRef: releaseRef("release-1"), Qualified: true}
	verifier, err := NewVerifier(nil, nil, nil, &fakeAssetVerifier{})
	require.NoError(t, err)
	err = verifier.VerifyPublication(context.Background(), document)
	require.ErrorIs(t, err, ErrUnqualified)
}

func TestNonAvailableStatusesDoNotRequireQualificationReaders(t *testing.T) {
	document := seedDocument(t)
	verifier, err := NewVerifier(nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, verifier.VerifyPublication(context.Background(), document))
}

func TestStaticBindingsAreCopiedAndUnknownCapabilityFailsClosed(t *testing.T) {
	original := map[string]CapabilityBinding{"web-console\x00sessions": {AppKey: "web-console", CapabilityID: "sessions", RequiredEvidence: []EvidenceSelector{{EvidenceID: "e", Kind: "k", Owner: "o", SubjectID: "s"}}}}
	bindings := NewStaticBindings(original)
	original["web-console\x00sessions"] = CapabilityBinding{}
	binding, err := bindings.ResolveBinding(context.Background(), "web-console", "sessions")
	require.NoError(t, err)
	require.Equal(t, "web-console", binding.AppKey)
	_, err = bindings.ResolveBinding(context.Background(), "missing", "capability")
	require.Error(t, err)
}

func TestGeneratedReaderConstructorsRequireClients(t *testing.T) {
	_, err := NewTestGenieValidationReader(nil)
	require.Error(t, err)
	_, err = NewDeploymentManagerReleaseReader(nil)
	require.Error(t, err)
}

func seedDocument(t *testing.T) presentation.Document {
	t.Helper()
	document, err := presentationseed.Recommended()
	require.NoError(t, err)
	return document
}

var _ ValidationReader = (*TestGenieValidationReader)(nil)
var _ ReleaseReader = (*DeploymentManagerReleaseReader)(nil)
