package credentialuse

import (
	"errors"
	"testing"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	credentialusev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/credential_use"
)

func protectedPolicy() *credentialusev1.BrowserSessionPolicy {
	return &credentialusev1.BrowserSessionPolicy{
		Origin: "https://login.example.test", DocumentId: "document-1",
		ExposureClass:     credentialusev1.BrowserExposureClass_BROWSER_EXPOSURE_CLASS_PROTECTED,
		AllowedOperations: []string{"navigate", "click"},
	}
}

func TestValidateActionRejectsOriginRaceAndUnsafeActions(t *testing.T) {
	policy := protectedPolicy()
	if err := ValidateAction(policy, &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://attacker.example.test"}}}, policy.GetOrigin(), policy.GetDocumentId()); !errors.Is(err, ErrOriginChanged) {
		t.Fatalf("cross-origin navigation error = %v", err)
	}
	for _, actionType := range []basactions.ActionType{basactions.ActionType_ACTION_TYPE_EVALUATE, basactions.ActionType_ACTION_TYPE_EXTRACT, basactions.ActionType_ACTION_TYPE_COOKIE_STORAGE} {
		if err := ValidateAction(policy, &basactions.ActionDefinition{Type: actionType}, policy.GetOrigin(), policy.GetDocumentId()); !errors.Is(err, ErrActionDenied) {
			t.Fatalf("unsafe action %s error = %v", actionType, err)
		}
	}
}

func TestValidateActionBindsDocumentAndReportsBroaderExposure(t *testing.T) {
	policy := protectedPolicy()
	policy.AllowedOperations = []string{"click"}
	if err := ValidateAction(policy, &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK}, policy.GetOrigin(), "new-document"); !errors.Is(err, ErrDocumentChanged) {
		t.Fatalf("document race error = %v", err)
	}
	policy.ExposureClass = credentialusev1.BrowserExposureClass_BROWSER_EXPOSURE_CLASS_UNRESTRICTED_SESSION
	if err := ValidateAction(policy, &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK}, policy.GetOrigin(), policy.GetDocumentId()); !errors.Is(err, ErrUnsupportedExposure) {
		t.Fatalf("unrestricted exposure error = %v", err)
	}
}
