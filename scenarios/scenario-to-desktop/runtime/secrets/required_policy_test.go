package secrets

import (
	"reflect"
	"testing"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/testutil"
)

func TestRequiredSecretPolicyMatchesAcrossEnforcementPaths(t *testing.T) {
	required, optional := true, false
	definitions := []manifest.Secret{
		{ID: "implicit", Target: manifest.SecretTarget{Type: "env", Name: "IMPLICIT"}},
		{ID: "required", Required: &required, Target: manifest.SecretTarget{Type: "env", Name: "REQUIRED"}},
		{ID: "unicode_required", Required: &required, Target: manifest.SecretTarget{Type: "env", Name: "UNICODE_REQUIRED"}},
		{ID: "optional", Required: &optional, Target: manifest.SecretTarget{Type: "env", Name: "OPTIONAL"}},
		{ID: "present", Required: &required, Target: manifest.SecretTarget{Type: "env", Name: "PRESENT"}},
	}
	manifestDoc := &manifest.Manifest{Secrets: definitions}
	values := map[string]string{
		"required":         " 	 ",
		"unicode_required": "\u00a0",
		"optional":         " ",
		"present":          " credential ",
	}
	wantMissing := []string{"implicit", "required", "unicode_required"}
	wantValidation := "secret implicit: required secret is missing; secret required: required secret is missing; secret unicode_required: required secret is missing"

	validator := NewValidator(manifestDoc)
	if got := validator.Validate(values).Error(); got != wantValidation {
		t.Fatalf("Validate() = %q, want %q", got, wantValidation)
	}
	if got := validator.MissingRequired(values); !reflect.DeepEqual(got, wantMissing) {
		t.Fatalf("Validator.MissingRequired() = %v, want %v", got, wantMissing)
	}

	singleResults := map[string]bool{
		"implicit":         validator.ValidateSecret("implicit", "  ") != nil,
		"required":         validator.ValidateSecret("required", " 	") != nil,
		"unicode_required": validator.ValidateSecret("unicode_required", "\u00a0") != nil,
		"optional":         validator.ValidateSecret("optional", " ") == nil,
		"present":          validator.ValidateSecret("present", " credential ") == nil,
	}
	wantSingleResults := map[string]bool{"implicit": true, "required": true, "unicode_required": true, "optional": true, "present": true}
	if !reflect.DeepEqual(singleResults, wantSingleResults) {
		t.Fatalf("ValidateSecret outcomes = %v, want %v", singleResults, wantSingleResults)
	}

	store := NewManager(manifestDoc)
	store.Set(values)
	if got := store.MissingRequiredFrom(values); !reflect.DeepEqual(got, wantMissing) {
		t.Fatalf("Manager.MissingRequiredFrom() = %v, want %v", got, wantMissing)
	}
	if got := store.MissingRequired(); !reflect.DeepEqual(got, wantMissing) {
		t.Fatalf("Manager.MissingRequired() = %v, want %v", got, wantMissing)
	}
	if err := store.Validate(values); err == nil || err.Error() != wantValidation {
		t.Fatalf("Manager.Validate() = %v, want %q", err, wantValidation)
	}

	injector := NewInjector(store, testutil.NewMockFileSystem(), t.TempDir())
	env := map[string]string{"KEEP": "before"}
	if err := injector.Apply(env, manifest.Service{ID: "required-service", Secrets: []string{"implicit"}}); err == nil {
		t.Fatal("Apply() accepted an absent secret whose Required field is omitted")
	}
	if want := map[string]string{"KEEP": "before"}; !reflect.DeepEqual(env, want) {
		t.Fatalf("rejected Apply() changed environment to %v, want %v", env, want)
	}
	if err := injector.Apply(env, manifest.Service{ID: "unicode-required-service", Secrets: []string{"unicode_required"}}); err == nil {
		t.Fatal("Apply() accepted a required secret containing only Unicode whitespace")
	}
	if want := map[string]string{"KEEP": "before"}; !reflect.DeepEqual(env, want) {
		t.Fatalf("Unicode-whitespace rejection changed environment to %v, want %v", env, want)
	}

	env = map[string]string{}
	if err := injector.Apply(env, manifest.Service{ID: "ready-service", Secrets: []string{"optional", "present"}}); err != nil {
		t.Fatalf("Apply() rejected an empty optional secret or present required secret: %v", err)
	}
	if want := map[string]string{"PRESENT": "credential"}; !reflect.DeepEqual(env, want) {
		t.Fatalf("Apply() environment = %v, want %v", env, want)
	}
}
