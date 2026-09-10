package profiles

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeProfile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsUnsafeAndUnboundedProfiles(t *testing.T) {
	profile := Profile{SchemaVersion: SupportedSchemaVersion, ID: "unsafe", Version: "1", Owner: "test", Questions: []Question{{ID: "q", Type: "number"}}}
	if err := Validate(profile); err == nil {
		t.Fatal("Validate accepted unsupported question type")
	}
	profile.Questions = []Question{{ID: "q", Type: "text", VisibleWhen: []byte(`{"includeProfile":"other"}`)}}
	if err := Validate(profile); err == nil {
		t.Fatal("Validate accepted an unsupported condition operator")
	}
}

func TestEvaluateProfileUsesConditionalQuestionsAndDeduplicatesClosure(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "release.json", `{
      "schemaVersion":"1.0.0", "id":"release", "version":"1.0.0", "owner":"test",
      "questions":[
        {"id":"purposes","type":"multi-select","required":true,"options":[{"id":"publish","labelKey":"publish"}],"minSelections":1},
        {"id":"platform","type":"single-select","required":true,"visibleWhen":{"contains":{"answer":"purposes","value":"publish"}},"options":[{"id":"linux","labelKey":"linux"}]}
      ],
      "rules":[
        {"id":"one","when":{"contains":{"answer":"purposes","value":"publish"}},"recommend":[{"capabilityRef":"release.desktop","scenarioRefs":["desktop","shared"],"reasonKey":"one"}]},
        {"id":"two","when":{"contains":{"answer":"purposes","value":"publish"}},"recommend":[{"capabilityRef":"release.mail","scenarioRefs":["shared"],"reasonKey":"two"}]}
      ],
      "manualSelection":{"available":true}, "provenance":{"source":"test","reviewRevision":"r1"}
    }`)
	service := Service{ProfileDir: func(_ context.Context) (string, error) { return dir, nil }, Catalog: func(_ context.Context) ([]Scenario, error) {
		return []Scenario{{Name: "desktop", Resources: []string{"runtime", "shared-resource"}}, {Name: "shared", Resources: []string{"shared-resource"}}}, nil
	}}
	result, err := service.Evaluate(context.Background(), "release", map[string]any{"purposes": []any{"publish"}, "platform": "linux"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid || len(result.Issues) != 0 {
		t.Fatalf("evaluation = %+v", result)
	}
	if len(result.Questions) != 2 || len(result.Scenarios) != 2 || len(result.Resources) != 2 {
		t.Fatalf("evaluation did not compose deterministically: %+v", result)
	}
	if result.Scenarios[0] != "desktop" || result.Scenarios[1] != "shared" {
		t.Fatalf("scenario order = %v", result.Scenarios)
	}
	if result.Resources[0] != "runtime" || result.Resources[1] != "shared-resource" {
		t.Fatalf("resource order = %v", result.Resources)
	}
}

func TestEvaluateProfileReportsMissingRequiredAnswersAndUnknownScenario(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "bad.json", `{
      "schemaVersion":"1.0.0", "id":"bad", "version":"1.0.0", "owner":"test",
      "questions":[{"id":"purposes","type":"multi-select","required":true,"options":[{"id":"use","labelKey":"use"}],"minSelections":1}],
      "rules":[{"id":"bad-ref","recommend":[{"capabilityRef":"unknown","scenarioRefs":["missing"]}]}]
    }`)
	service := Service{ProfileDir: func(_ context.Context) (string, error) { return dir, nil }, Catalog: func(_ context.Context) ([]Scenario, error) { return nil, nil }}
	result, err := service.Evaluate(context.Background(), "bad", map[string]any{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Valid || len(result.Issues) != 2 {
		t.Fatalf("evaluation = %+v", result)
	}
	if result.Issues[0].Code != "required" {
		t.Fatalf("first issue = %+v", result.Issues[0])
	}
}

func TestRepositoryProfilesCoverGeneralPublishingAndCustomerPaths(t *testing.T) {
	dir := repositoryProfilesDir(t)
	profiles, err := LoadDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]Profile, len(profiles))
	for _, profile := range profiles {
		byID[profile.ID] = profile
	}
	for _, id := range []string{"local-use", "general-purpose", "develop-and-publish", "customer-preinstalled"} {
		if _, ok := byID[id]; !ok {
			t.Fatalf("repository profile %q is missing", id)
		}
	}

	service := Service{
		ProfileDir: func(context.Context) (string, error) { return dir, nil },
		Catalog: func(context.Context) ([]Scenario, error) {
			return []Scenario{
				{Name: "vrooli-onboarding"}, {Name: "landing-page-business-suite"}, {Name: "scenario-authenticator"},
				{Name: "scenario-to-desktop"}, {Name: "scenario-to-cloud"}, {Name: "scenario-to-android"}, {Name: "scenario-to-ios"},
			}, nil
		},
	}
	result, err := service.Evaluate(context.Background(), "general-purpose", map[string]any{
		"purposes":         []any{"publish-apps"},
		"desktopPlatforms": []any{"linux"},
		"mobilePlatforms":  []any{"android", "ios"},
		"hosting":          "managed-vps",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid || len(result.Issues) != 0 {
		t.Fatalf("general-purpose evaluation = %+v", result)
	}
	for _, name := range []string{"scenario-to-desktop", "scenario-to-cloud", "scenario-to-android", "scenario-to-ios"} {
		if !contains(result.Scenarios, name) {
			t.Errorf("general-purpose result omitted %q: %v", name, result.Scenarios)
		}
	}

	customer, err := service.Evaluate(context.Background(), "customer-preinstalled", map[string]any{
		"optionalCapabilities": []any{"development", "managed-hosting"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !customer.Valid || customer.Profile.ProvenanceSource != "customer-preset" || customer.Profile.ProvenanceRevision == "" {
		t.Fatalf("customer preset evaluation = %+v", customer)
	}
}

func repositoryProfilesDir(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for current := filepath.Clean(cwd); ; current = filepath.Dir(current) {
		candidate := filepath.Join(current, "profiles")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			if _, profileErr := os.Stat(filepath.Join(candidate, "local-use.json")); profileErr != nil {
				continue
			}
			return candidate
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	t.Fatalf("could not locate onboarding profiles from %s", cwd)
	return ""
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
