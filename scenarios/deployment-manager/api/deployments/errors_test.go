package deployments

import (
	"testing"
)

func TestValidateProfileAndErrorFormatting(t *testing.T) {
	tests := []struct {
		name       string
		profileID  string
		wantValid  bool
		wantKind   ErrorKind
		wantStatus int
	}{
		{"missing", "", false, ErrorMissingID, 400},
		{"unknown", "demo", false, ErrorNotFound, 404},
		{"profile", "profile-demo", true, 0, 0},
		{"test", "test-demo", true, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidateProfile(tc.profileID)
			if result.Valid != tc.wantValid {
				t.Fatalf("Valid = %v, want %v", result.Valid, tc.wantValid)
			}
			if !tc.wantValid {
				if len(result.Errors) != 1 || result.Errors[0].Kind != tc.wantKind {
					t.Fatalf("errors = %#v", result.Errors)
				}
				if got, _ := FormatValidationError(result); got != tc.wantStatus {
					t.Fatalf("status = %d, want %d", got, tc.wantStatus)
				}
			}
		})
	}
}

func TestValidationErrorFormattingCoversRemediationKinds(t *testing.T) {
	if status, body := FormatValidationError(ValidationResult{}); status != 400 || body["error"] != "unknown validation error" {
		t.Fatalf("empty validation = %d %#v", status, body)
	}
	validation := ValidationResult{
		Errors:         []ValidationError{{Kind: ErrorSigningValidation, Message: "certificate missing"}, {Kind: ErrorValidation, Message: "target missing"}},
		RecommendedFix: "configure signing",
	}
	status, body := FormatValidationError(validation)
	if status != 424 || body["error"] != "Code signing prerequisites not met" {
		t.Fatalf("signing validation = %d %#v", status, body)
	}
	if got := validation.ErrorMessages(); len(got) != 2 || got[0] != "certificate missing" || got[1] != "target missing" {
		t.Fatalf("messages = %#v", got)
	}

	status, body = FormatValidationError(ValidationResult{
		Errors:         []ValidationError{{Kind: ErrorValidation, Message: "invalid target"}},
		RecommendedFix: "choose a target",
	})
	if status != 400 || body["error"] != "Deployment validation failed" {
		t.Fatalf("general validation = %d %#v", status, body)
	}
	for kind, want := range map[ErrorKind]int{
		ErrorMissingID:         400,
		ErrorNotFound:          404,
		ErrorValidation:        400,
		ErrorSigningValidation: 424,
		ErrorKind(99):          400,
	} {
		if got := HTTPStatusForErrorKind(kind); got != want {
			t.Errorf("HTTPStatusForErrorKind(%d) = %d, want %d", kind, got, want)
		}
	}
	first := GenerateID()
	if len(first) < len("deploy-") || first[:len("deploy-")] != "deploy-" {
		t.Fatalf("GenerateID() = %q", first)
	}
}
