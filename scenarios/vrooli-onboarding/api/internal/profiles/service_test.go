package profiles

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func writeProfile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsUnsafeAndUnboundedProfiles(t *testing.T) {
	profile := Profile{SchemaVersion: SupportedSchemaVersion, ID: "unsafe", Version: "1.0.0", TitleKey: "unsafe.title", DescriptionKey: "unsafe.description", Owner: "test", CompatibleCatalogMajor: 1, Provenance: Provenance{Source: "test", Revision: "r1"}, Questions: []Question{{ID: "q", Type: "number"}}}
	if err := Validate(profile); err == nil {
		t.Fatal("Validate accepted unsupported question type")
	}
	profile.Questions = []Question{{ID: "q", Type: "text", PromptKey: "q", VisibleWhen: []byte(`{"includeProfile":"other"}`)}}
	if err := Validate(profile); err == nil {
		t.Fatal("Validate accepted an unsupported condition operator")
	}
}

func TestLoadDirectoryRejectsMultipleDefaultProfiles(t *testing.T) {
	dir := t.TempDir()
	profile := `{"schemaVersion":"1.0.0","version":"1.0.0","default":true,"titleKey":"title","descriptionKey":"description","owner":"test","compatibleCatalogMajor":1,"questions":[],"rules":[],"manualSelection":{"available":true},"provenance":{"source":"test","reviewRevision":"r1"}}`
	writeProfile(t, dir, "one.json", strings.Replace(profile, `"titleKey":"title"`, `"id":"one","titleKey":"title"`, 1))
	writeProfile(t, dir, "two.json", strings.Replace(profile, `"titleKey":"title"`, `"id":"two","titleKey":"title"`, 1))
	if _, err := LoadDirectory(dir); err == nil || !strings.Contains(err.Error(), "at most one") {
		t.Fatalf("LoadDirectory error = %v, want multiple-default rejection", err)
	}
}

func TestValidateEnforcesTypedDefaultsAndKnownConditionAnswers(t *testing.T) {
	profile := Profile{
		SchemaVersion: SupportedSchemaVersion, ID: "typed", Version: "1.0.0",
		TitleKey: "typed.title", DescriptionKey: "typed.description", Owner: "test", CompatibleCatalogMajor: 1,
		Provenance: Provenance{Source: "test", Revision: "r1"},
		Questions: []Question{
			{ID: "purpose", Type: "single-select", PromptKey: "purpose", Options: []Option{{ID: "use", LabelKey: "use"}}, Default: []byte(`"use"`)},
			{ID: "platforms", Type: "multi-select", PromptKey: "platforms", Options: []Option{{ID: "linux", LabelKey: "linux"}}, VisibleWhen: []byte(`{"contains":{"answer":"missing","value":"use"}}`)},
		},
	}
	if err := Validate(profile); err == nil || !strings.Contains(err.Error(), "unknown answer") {
		t.Fatalf("Validate error = %v, want unknown answer", err)
	}
	profile.Questions[1].VisibleWhen = []byte(`{"contains":{"answer":"purpose","value":"use"}}`)
	profile.Questions[0].Default = []byte(`"missing"`)
	if err := Validate(profile); err == nil || !strings.Contains(err.Error(), "supported option") {
		t.Fatalf("Validate error = %v, want invalid default", err)
	}
}

func TestValidateRejectsVisibilityCyclesAndUnreachableRequiredQuestions(t *testing.T) {
	base := Profile{
		SchemaVersion: SupportedSchemaVersion, ID: "graph", Version: "1.0.0",
		TitleKey: "graph.title", DescriptionKey: "graph.description", Owner: "test", CompatibleCatalogMajor: 1,
		Provenance: Provenance{Source: "test", Revision: "r1"},
	}
	base.Questions = []Question{
		{ID: "one", Type: "boolean", PromptKey: "one", VisibleWhen: []byte(`{"eq":{"answer":"two","value":true}}`)},
		{ID: "two", Type: "boolean", PromptKey: "two", VisibleWhen: []byte(`{"eq":{"answer":"one","value":true}}`)},
	}
	if err := Validate(base); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("cycle validation error = %v, want cycle", err)
	}
	base.Questions = []Question{{ID: "choice", Type: "single-select", PromptKey: "choice", Options: []Option{{ID: "a", LabelKey: "a"}}}, {ID: "never", Type: "text", PromptKey: "never", Required: true, VisibleWhen: []byte(`{"eq":{"answer":"choice","value":"missing"}}`)}}
	if err := Validate(base); err == nil || !strings.Contains(err.Error(), "unreachable") {
		t.Fatalf("unreachable validation error = %v, want unreachable", err)
	}
}

func TestEvaluateAppliesDefaultsAndProducesStableExplanationDigest(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "defaults.json", `{
      "schemaVersion":"1.0.0", "id":"defaults", "version":"1.0.0", "titleKey":"defaults.title", "descriptionKey":"defaults.description", "owner":"test", "compatibleCatalogMajor":1,
      "questions":[{"id":"purpose","type":"single-select","promptKey":"purpose","required":true,"default":"use","options":[{"id":"use","labelKey":"use"}]}],
      "rules":[{"id":"local","when":{"eq":{"answer":"purpose","value":"use"}},"recommend":[{"capabilityRef":"development.local","scenarioRefs":["desktop"],"reasonKey":"reason.local"}]}],
      "manualSelection":{"available":true}, "provenance":{"source":"test","reviewRevision":"r1"}
    }`)
	service := Service{ProfileDir: func(context.Context) (string, error) { return dir, nil }, Catalog: func(context.Context) ([]Scenario, error) {
		return []Scenario{{Name: "desktop", Resources: []string{"runtime"}}}, nil
	}}
	first, err := service.Evaluate(context.Background(), "defaults", map[string]any{}, map[string]any{"target": "local"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Evaluate(context.Background(), "defaults", map[string]any{}, map[string]any{"target": "local"})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Valid || first.Digest == "" || first.Digest != second.Digest || len(first.Explanations) != 1 || first.Recommendations[0].RuleID != "local" {
		t.Fatalf("evaluation = %+v, want defaulted stable explained result", first)
	}
	if len(first.Issues) != 0 || len(first.Scenarios) != 1 || first.Scenarios[0] != "desktop" {
		t.Fatalf("defaulted evaluation = %+v", first)
	}
	answerOrderChanged, err := service.Evaluate(context.Background(), "defaults", map[string]any{"unused": true}, map[string]any{"target": "local"})
	if err != nil {
		t.Fatal(err)
	}
	if answerOrderChanged.Digest == first.Digest {
		t.Fatal("digest ignored an answer change")
	}
}

func TestEvaluateCanonicalizesEquivalentAnswerAndOutputOrdering(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "canonical.json", `{
      "schemaVersion":"1.0.0", "id":"canonical", "version":"1.0.0", "titleKey":"canonical.title", "descriptionKey":"canonical.description", "owner":"test", "compatibleCatalogMajor":1,
      "questions":[{"id":"purposes","type":"multi-select","promptKey":"purposes","required":true,"options":[{"id":"use","labelKey":"use"},{"id":"develop","labelKey":"develop"}]}],
      "rules":[
			{"id":"z-rule","when":{"contains":{"answer":"purposes","value":"use"}},"recommend":[{"capabilityRef":"capability.z","scenarioRefs":["zeta", "alpha"],"reasonKey":"reason.z"}]},
			{"id":"a-rule","when":{"contains":{"answer":"purposes","value":"develop"}},"recommend":[{"capabilityRef":"capability.a","scenarioRefs":["alpha"],"reasonKey":"reason.a"}]}
      ],
      "manualSelection":{"available":true}, "provenance":{"source":"test","reviewRevision":"r1"}
    }`)
	service := Service{
		ProfileDir: func(context.Context) (string, error) { return dir, nil },
		Catalog: func(context.Context) ([]Scenario, error) {
			return []Scenario{{Name: "zeta"}, {Name: "alpha"}}, nil
		},
	}
	first, err := service.Evaluate(context.Background(), "canonical", map[string]any{"purposes": []any{"use", "develop"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Evaluate(context.Background(), "canonical", map[string]any{"purposes": []any{"develop", "use"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("equivalent multi-select ordering changed digest: %q != %q", first.Digest, second.Digest)
	}
	if len(first.Recommendations) != 2 || first.Recommendations[0].CapabilityRef != "capability.a" || first.Recommendations[1].CapabilityRef != "capability.z" {
		t.Fatalf("recommendations were not canonically ordered: %+v", first.Recommendations)
	}
	if len(first.Explanations) != 2 || first.Explanations[0].CapabilityRef != "capability.a" || first.Explanations[1].CapabilityRef != "capability.z" {
		t.Fatalf("explanations were not canonically ordered: %+v", first.Explanations)
	}
}

func TestEvaluateMergesSharedCapabilityReasonsAndKeepsCanonicalSource(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "shared.json", `{
      "schemaVersion":"1.0.0", "id":"shared", "version":"1.0.0", "titleKey":"shared.title", "descriptionKey":"shared.description", "owner":"test", "compatibleCatalogMajor":1,
      "questions":[{"id":"purposes","type":"multi-select","promptKey":"purposes","required":true,"options":[{"id":"use","labelKey":"use"},{"id":"develop","labelKey":"develop"}]}],
      "rules":[
        {"id":"z-purpose","when":{"contains":{"answer":"purposes","value":"use"}},"recommend":[{"capabilityRef":"capability.shared","scenarioRefs":["shared"],"reasonKey":"reason.use"}]},
        {"id":"a-purpose","when":{"contains":{"answer":"purposes","value":"develop"}},"recommend":[{"capabilityRef":"capability.shared","scenarioRefs":["shared"],"reasonKey":"reason.develop"}]}
      ],
      "manualSelection":{"available":true}, "provenance":{"source":"test","reviewRevision":"r1"}
    }`)
	service := Service{
		ProfileDir: func(context.Context) (string, error) { return dir, nil },
		Catalog:    func(context.Context) ([]Scenario, error) { return []Scenario{{Name: "shared"}}, nil },
	}
	result, err := service.Evaluate(context.Background(), "shared", map[string]any{"purposes": []any{"use", "develop"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid || len(result.Recommendations) != 1 || result.Recommendations[0].RuleID != "a-purpose" || result.Recommendations[0].ReasonKey != "reason.develop" {
		t.Fatalf("shared recommendation = %+v, want one canonical recommendation", result.Recommendations)
	}
	if len(result.Explanations) != 2 {
		t.Fatalf("shared explanations = %+v, want both purpose reasons", result.Explanations)
	}
}

func TestEvaluatePresetAnswersYieldToExplicitAnswersAndAreDigestBound(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "preset.json", `{
      "schemaVersion":"1.0.0", "id":"preset", "version":"1.0.0", "titleKey":"preset.title", "descriptionKey":"preset.description", "owner":"test", "compatibleCatalogMajor":1,
      "questions":[{"id":"purpose","type":"single-select","promptKey":"purpose","required":true,"default":"generic","options":[{"id":"generic","labelKey":"generic"},{"id":"preset","labelKey":"preset"},{"id":"explicit","labelKey":"explicit"}]}],
      "rules":[
        {"id":"preset-rule","when":{"eq":{"answer":"purpose","value":"preset"}},"recommend":[{"capabilityRef":"capability.preset","reasonKey":"reason.preset"}]},
        {"id":"explicit-rule","when":{"eq":{"answer":"purpose","value":"explicit"}},"recommend":[{"capabilityRef":"capability.explicit","reasonKey":"reason.explicit"}]}
      ],
      "manualSelection":{"available":true}, "provenance":{"source":"test","reviewRevision":"r1"}
    }`)
	service := Service{
		ProfileDir: func(context.Context) (string, error) { return dir, nil },
		Catalog:    func(context.Context) ([]Scenario, error) { return nil, nil },
	}
	preset := &Preset{ID: "installation", Version: "2.0.0", Source: "customer", Answers: map[string]any{"purpose": "preset"}}
	fromPreset, err := service.EvaluateWithPreset(context.Background(), "preset", nil, nil, nil, preset)
	if err != nil {
		t.Fatal(err)
	}
	if !fromPreset.Valid || fromPreset.Preset == nil || fromPreset.Preset.Source != "customer" || len(fromPreset.Recommendations) != 1 || fromPreset.Recommendations[0].CapabilityRef != "capability.preset" {
		t.Fatalf("preset evaluation = %+v", fromPreset)
	}
	fromExplicit, err := service.EvaluateWithPreset(context.Background(), "preset", map[string]any{"purpose": "explicit"}, nil, nil, preset)
	if err != nil {
		t.Fatal(err)
	}
	if !fromExplicit.Valid || len(fromExplicit.Recommendations) != 1 || fromExplicit.Recommendations[0].CapabilityRef != "capability.explicit" || fromExplicit.Digest == fromPreset.Digest {
		t.Fatalf("explicit evaluation = %+v, want explicit answer to override preset", fromExplicit)
	}
}

func TestEvaluateReportsTypedOutstandingAndManualConflicts(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "conflicts.json", `{
      "schemaVersion":"1.0.0", "id":"conflicts", "version":"1.0.0", "titleKey":"conflicts.title", "descriptionKey":"conflicts.description", "owner":"test", "compatibleCatalogMajor":1,
      "questions":[{"id":"purpose","type":"single-select","promptKey":"purpose","required":true,"options":[{"id":"use","labelKey":"use"}]}],
      "rules":[{"id":"recommend","when":{"eq":{"answer":"purpose","value":"use"}},"recommend":[{"capabilityRef":"capability.one","scenarioRefs":["missing"],"reasonKey":"reason.one"},{"capabilityRef":"capability.one","scenarioRefs":["available"],"reasonKey":"reason.one"}]}],
      "manualSelection":{"available":true}, "provenance":{"source":"test","reviewRevision":"r1"}
    }`)
	service := Service{
		ProfileDir: func(context.Context) (string, error) { return dir, nil },
		Catalog:    func(context.Context) ([]Scenario, error) { return []Scenario{{Name: "available"}}, nil },
	}
	base, err := service.Evaluate(context.Background(), "conflicts", map[string]any{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if base.Valid || len(base.Outstanding) != 1 || base.Outstanding[0].Code != "required" {
		t.Fatalf("missing-answer evaluation = %+v, want typed required outstanding", base)
	}
	selected, err := service.Evaluate(context.Background(), "conflicts", map[string]any{"purpose": "use"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Valid || len(selected.Outstanding) != 1 || selected.Outstanding[0].Code != "unsupported" {
		t.Fatalf("unsupported evaluation = %+v, want typed unsupported outstanding", selected)
	}
	decisions := map[string]bool{selected.Recommendations[0].Key: false, selected.Recommendations[1].Key: true}
	conflicted, err := service.EvaluateWithManualDecisions(context.Background(), "conflicts", map[string]any{"purpose": "use"}, nil, decisions)
	if err != nil {
		t.Fatal(err)
	}
	if conflicted.Valid || len(conflicted.Conflicts) != 1 || conflicted.Conflicts[0].Code != "conflicting_manual_decisions" {
		t.Fatalf("conflicted evaluation = %+v, want explicit manual conflict", conflicted)
	}
}

func TestEvaluateProfilesComposesProfileOrderIndependently(t *testing.T) {
	makeProfile := func(id, capability string) Profile {
		return Profile{
			SchemaVersion: SupportedSchemaVersion, ID: id, Version: "1.0.0", TitleKey: id + ".title", DescriptionKey: id + ".description", Owner: "test", CompatibleCatalogMajor: 1,
			Rules:           []Rule{{ID: "recommend", Recommend: []Recommendation{{CapabilityRef: capability, ReasonKey: "reason." + id}}}},
			ManualSelection: Manual{Available: true}, Provenance: Provenance{Source: "test", Revision: id + "-r1"},
		}
	}
	first, err := EvaluateProfiles(EvaluationInput{
		Profiles:        []Profile{makeProfile("beta", "capability.beta"), makeProfile("alpha", "capability.alpha")},
		CatalogRevision: "catalog-r1",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EvaluateProfiles(EvaluationInput{
		Profiles:        []Profile{makeProfile("alpha", "capability.alpha"), makeProfile("beta", "capability.beta")},
		CatalogRevision: "catalog-r1",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest || first.CatalogRevision != "catalog-r1" || len(first.Profiles) != 2 || first.Profiles[0].ID != "alpha" {
		t.Fatalf("composed evaluations differ: first=%+v second=%+v", first, second)
	}
	if len(first.Recommendations) != 2 || first.Recommendations[0].CapabilityRef != "capability.alpha" || first.Recommendations[1].CapabilityRef != "capability.beta" {
		t.Fatalf("composed recommendations = %+v, want stable merged order", first.Recommendations)
	}
}

func TestEvaluateProfilesDeduplicatesIdenticalQuestionsAndRejectsConflictingDefinitions(t *testing.T) {
	question := Question{ID: "purpose", Type: "single-select", PromptKey: "purpose", Required: true, Options: []Option{{ID: "local", LabelKey: "local"}}}
	makeProfile := func(id string, profileQuestion Question) Profile {
		return Profile{
			SchemaVersion: SupportedSchemaVersion, ID: id, Version: "1.0.0", TitleKey: id + ".title", DescriptionKey: id + ".description", Owner: "test", CompatibleCatalogMajor: 1,
			Questions: []Question{profileQuestion}, ManualSelection: Manual{Available: true}, Provenance: Provenance{Source: "test", Revision: id + "-r1"},
		}
	}
	result, err := EvaluateProfiles(EvaluationInput{Profiles: []Profile{makeProfile("beta", question), makeProfile("alpha", question)}, Answers: map[string]any{"purpose": "local"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Questions) != 1 || result.Questions[0].ID != "purpose" || !result.Valid {
		t.Fatalf("composed questions = %+v, valid=%v; want one shared question and valid result", result.Questions, result.Valid)
	}
	conflicting := question
	conflicting.PromptKey = "different-purpose"
	if _, err := EvaluateProfiles(EvaluationInput{Profiles: []Profile{makeProfile("alpha", question), makeProfile("beta", conflicting)}}, nil); err == nil || !strings.Contains(err.Error(), `question "purpose" differs`) {
		t.Fatalf("conflicting composition error = %v, want question conflict", err)
	}
}

func TestEvaluateProfilesStaysBoundedAtMaximumProfileInput(t *testing.T) {
	profile := Profile{
		SchemaVersion:          SupportedSchemaVersion,
		ID:                     "maximum",
		Version:                "1.0.0",
		TitleKey:               "maximum.title",
		DescriptionKey:         "maximum.description",
		Owner:                  "test",
		CompatibleCatalogMajor: 1,
		ManualSelection:        Manual{Available: true},
		Provenance:             Provenance{Source: "test", Revision: "maximum-r1"},
		Questions:              make([]Question, MaxQuestions),
		Rules:                  make([]Rule, MaxRules),
	}
	answers := make(map[string]any, MaxQuestions)
	for index := range profile.Questions {
		id := fmt.Sprintf("question-%03d", index)
		profile.Questions[index] = Question{ID: id, Type: "text", PromptKey: id}
		answers[id] = "answer"
	}
	for index := range profile.Rules {
		profile.Rules[index] = Rule{ID: fmt.Sprintf("rule-%03d", index)}
		for recommendationIndex := 0; recommendationIndex < MaxRecommendations/MaxRules; recommendationIndex++ {
			capability := fmt.Sprintf("capability-%04d", index*(MaxRecommendations/MaxRules)+recommendationIndex)
			profile.Rules[index].Recommend = append(profile.Rules[index].Recommend, Recommendation{
				CapabilityRef: capability,
				ReasonKey:     "reason.maximum",
			})
		}
	}
	if err := Validate(profile); err != nil {
		t.Fatalf("maximum supported profile rejected: %v", err)
	}

	started := time.Now()
	result, err := EvaluateProfiles(EvaluationInput{
		Profiles:        []Profile{profile},
		Answers:         answers,
		TargetContext:   map[string]any{"catalogRevision": "catalog-max"},
		CatalogRevision: "catalog-max",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("maximum supported profile evaluation took %s; want <= 2s", elapsed)
	}
	if !result.Valid || len(result.Questions) != MaxQuestions || len(result.Recommendations) != MaxRecommendations || result.Digest == "" {
		t.Fatalf("maximum profile evaluation = valid=%t questions=%d recommendations=%d digest=%q", result.Valid, len(result.Questions), len(result.Recommendations), result.Digest)
	}
}

func TestServiceEvaluateProfilesLoadsMultipleProfiles(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "one.json", `{"schemaVersion":"1.0.0","id":"one","version":"1.0.0","titleKey":"one.title","descriptionKey":"one.description","owner":"test","compatibleCatalogMajor":1,"questions":[],"rules":[{"id":"one-rule","recommend":[{"capabilityRef":"capability.one","reasonKey":"one.reason"}]}],"manualSelection":{"available":true},"provenance":{"source":"test","reviewRevision":"one-r1"}}`)
	writeProfile(t, dir, "two.json", `{"schemaVersion":"1.0.0","id":"two","version":"1.0.0","titleKey":"two.title","descriptionKey":"two.description","owner":"test","compatibleCatalogMajor":1,"questions":[],"rules":[{"id":"two-rule","recommend":[{"capabilityRef":"capability.two","reasonKey":"two.reason"}]}],"manualSelection":{"available":true},"provenance":{"source":"test","reviewRevision":"two-r1"}}`)
	service := Service{ProfileDir: func(context.Context) (string, error) { return dir, nil }, Catalog: func(context.Context) ([]Scenario, error) { return nil, nil }}
	result, err := service.EvaluateProfiles(context.Background(), []string{"two", "one"}, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Profiles) != 2 || result.Profiles[0].ID != "one" || result.Profiles[1].ID != "two" || len(result.Recommendations) != 2 {
		t.Fatalf("service composition = %+v", result)
	}
}

func TestEvaluateAppliesManualRecommendationDecisionsWithoutLosingExplanation(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "manual.json", `{
      "schemaVersion":"1.0.0", "id":"manual", "version":"1.0.0", "titleKey":"manual.title", "descriptionKey":"manual.description", "owner":"test", "compatibleCatalogMajor":1,
      "questions":[],
      "rules":[{"id":"default","recommend":[{"capabilityRef":"release.desktop","scenarioRefs":["desktop"],"reasonKey":"reason.desktop"},{"capabilityRef":"release.mail","scenarioRefs":["mail"],"reasonKey":"reason.mail"}]}],
      "manualSelection":{"available":true}, "provenance":{"source":"test","reviewRevision":"r1"}
    }`)
	service := Service{ProfileDir: func(context.Context) (string, error) { return dir, nil }, Catalog: func(context.Context) ([]Scenario, error) {
		return []Scenario{{Name: "desktop"}, {Name: "mail"}}, nil
	}}
	base, err := service.Evaluate(context.Background(), "manual", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(base.Recommendations) != 2 || !base.Recommendations[0].Selected || base.Recommendations[0].Key == "" {
		t.Fatalf("base recommendations = %+v, want stable selected keys", base.Recommendations)
	}
	removedKey := base.Recommendations[0].Key
	result, err := service.EvaluateWithManualDecisions(context.Background(), "manual", nil, nil, map[string]bool{removedKey: false})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid || len(result.Scenarios) != 1 || result.Scenarios[0] != "mail" {
		t.Fatalf("manual evaluation = %+v, want only mail selected", result)
	}
	if len(result.Recommendations) != 2 || result.Recommendations[0].Selected {
		t.Fatalf("manual recommendations = %+v, want removed recommendation retained with selected=false", result.Recommendations)
	}
	if len(result.Explanations) != 2 || result.Explanations[0].ReasonKey == "" || result.Explanations[0].Selected {
		t.Fatalf("manual explanations = %+v, want provenance for removed recommendation", result.Explanations)
	}
}

func TestLoadDirectoryRejectsUnknownFieldsAndOversizedProfiles(t *testing.T) {
	base := `{"schemaVersion":"1.0.0","id":"safe","version":"1.0.0","titleKey":"safe.title","descriptionKey":"safe.description","owner":"test","compatibleCatalogMajor":1,"questions":[],"rules":[],"manualSelection":{"available":true},"provenance":{"source":"test","reviewRevision":"r1"}}`
	t.Run("unknown executable-looking field", func(t *testing.T) {
		dir := t.TempDir()
		writeProfile(t, dir, "unsafe.json", strings.TrimSuffix(base, "}")+`,"script":"echo bad"}`)
		if _, err := LoadDirectory(dir); err == nil || !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("LoadDirectory error = %v, want unknown field", err)
		}
	})
	t.Run("size bound", func(t *testing.T) {
		dir := t.TempDir()
		writeProfile(t, dir, "large.json", fmt.Sprintf(`{"schemaVersion":"1.0.0","id":"large","version":"1.0.0","titleKey":"large.title","descriptionKey":"%s","owner":"test","compatibleCatalogMajor":1,"questions":[],"rules":[],"manualSelection":{"available":true},"provenance":{"source":"test","reviewRevision":"r1"}}`, strings.Repeat("x", MaxProfileBytes)))
		if _, err := LoadDirectory(dir); err == nil || !strings.Contains(err.Error(), "byte limit") {
			t.Fatalf("LoadDirectory error = %v, want byte limit", err)
		}
	})
}

func TestLoadDirectoryRejectsOversizedCatalogs(t *testing.T) {
	dir := t.TempDir()
	base := `{"schemaVersion":"1.0.0","id":"profile-%d","version":"1.0.0","titleKey":"profile.title","descriptionKey":"profile.description","owner":"test","compatibleCatalogMajor":1,"questions":[],"rules":[],"manualSelection":{"available":true},"provenance":{"source":"test","reviewRevision":"r1"}}`
	for index := 0; index < MaxProfiles+1; index++ {
		writeProfile(t, dir, fmt.Sprintf("profile-%d.json", index), fmt.Sprintf(base, index))
	}
	if _, err := LoadDirectory(dir); err == nil || !strings.Contains(err.Error(), "catalog limit") {
		t.Fatalf("LoadDirectory error = %v, want profile catalog limit", err)
	}
}

func TestEvaluateProfileUsesConditionalQuestionsAndDeduplicatesClosure(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "release.json", `{
      "schemaVersion":"1.0.0", "id":"release", "version":"1.0.0", "titleKey":"release.title", "descriptionKey":"release.description", "owner":"test", "compatibleCatalogMajor":1,
      "questions":[
        {"id":"purposes","type":"multi-select","promptKey":"purposes","required":true,"options":[{"id":"publish","labelKey":"publish"}],"minSelections":1},
        {"id":"platform","type":"single-select","promptKey":"platform","required":true,"visibleWhen":{"contains":{"answer":"purposes","value":"publish"}},"options":[{"id":"linux","labelKey":"linux"}]}
      ],
      "rules":[
        {"id":"one","when":{"contains":{"answer":"purposes","value":"publish"}},"recommend":[{"capabilityRef":"release.desktop","scenarioRefs":["desktop","shared"],"reasonKey":"one"}]},
        {"id":"two","when":{"contains":{"answer":"purposes","value":"publish"}},"recommend":[{"capabilityRef":"release.mail","scenarioRefs":["shared"],"reasonKey":"two"}]},
        {"id":"duplicate","when":{"contains":{"answer":"purposes","value":"publish"}},"recommend":[{"capabilityRef":"release.desktop","scenarioRefs":["shared","desktop"],"reasonKey":"one"}]}
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
	if len(result.Recommendations) != 2 || len(result.Explanations) != 3 {
		t.Fatalf("recommendation deduplication lost composition evidence: %+v", result)
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
      "schemaVersion":"1.0.0", "id":"bad", "version":"1.0.0", "titleKey":"bad.title", "descriptionKey":"bad.description", "owner":"test", "compatibleCatalogMajor":1,
      "questions":[{"id":"purposes","type":"multi-select","promptKey":"purposes","required":true,"options":[{"id":"use","labelKey":"use"}],"minSelections":1}],
      "rules":[{"id":"bad-ref","recommend":[{"capabilityRef":"unknown","scenarioRefs":["missing"],"reasonKey":"bad.reason"}]}],
      "provenance":{"source":"test","reviewRevision":"r1"}
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

func TestEvaluateProfileResolvesContextAndSettingConditions(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "context.json", `{
      "schemaVersion":"1.0.0", "id":"context", "version":"1.0.0", "titleKey":"context.title", "descriptionKey":"context.description", "owner":"test", "compatibleCatalogMajor":1,
      "questions":[{"id":"platform","type":"text","promptKey":"platform","required":true,"visibleWhen":{"eq":{"context":"targetRole","value":"publisher"}}}],
      "rules":[{"id":"release","when":{"eq":{"setting":"environment","value":"release"}},"recommend":[{"capabilityRef":"release.desktop","scenarioRefs":["desktop"],"reasonKey":"reason.release"}]}],
      "manualSelection":{"available":true}, "provenance":{"source":"test","reviewRevision":"r1"}
    }`)
	service := Service{
		ProfileDir: func(context.Context) (string, error) { return dir, nil },
		Catalog:    func(context.Context) ([]Scenario, error) { return []Scenario{{Name: "desktop"}}, nil },
	}
	result, err := service.Evaluate(context.Background(), "context", map[string]any{"platform": "macos"}, map[string]any{"targetRole": "publisher", "environment": "release"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid || len(result.Questions) != 1 || len(result.Recommendations) != 1 || result.Recommendations[0].CapabilityRef != "release.desktop" {
		t.Fatalf("contextual evaluation = %+v, want visible question and recommendation", result)
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
	if !byID["general-purpose"].Default {
		t.Fatal("general-purpose profile is not marked as the catalog default")
	}

	service := Service{
		ProfileDir: func(context.Context) (string, error) { return dir, nil },
		Catalog: func(context.Context) ([]Scenario, error) {
			return []Scenario{
				{Name: "vrooli-onboarding"}, {Name: "landing-page-business-suite"}, {Name: "scenario-authenticator"},
				{Name: "scenario-to-desktop"}, {Name: "scenario-to-cloud"}, {Name: "scenario-to-android"}, {Name: "scenario-to-ios"}, {Name: "vrooli-bridge"}, {Name: "landing-page-business-suite"}, {Name: "scenario-authenticator"},
			}, nil
		},
	}
	result, err := service.Evaluate(context.Background(), "general-purpose", map[string]any{
		"purposes":           []any{"publish-apps"},
		"desktopPlatforms":   []any{"linux"},
		"mobilePlatforms":    []any{"android", "ios"},
		"hosting":            "managed-vps",
		"distributionMethod": "direct-download",
		"accountOwnership":   "operator-owned",
		"payments":           "stripe",
		"mail":               "provider",
		"downloadStorage":    "s3",
		"remoteTargets":      "connected-host",
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

func TestRepositoryProfilesExposeConditionalReleaseContextWithoutSecrets(t *testing.T) {
	dir := repositoryProfilesDir(t)
	profiles, err := LoadDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	var general Profile
	for _, profile := range profiles {
		if profile.ID == "general-purpose" {
			general = profile
			break
		}
	}
	if general.ID == "" {
		t.Fatal("general-purpose profile is missing")
	}
	questionIDs := make(map[string]bool, len(general.Questions))
	for _, question := range general.Questions {
		questionIDs[question.ID] = true
	}
	for _, id := range []string{"distributionMethod", "accountOwnership", "payments", "mail", "downloadStorage", "remoteTargets"} {
		if !questionIDs[id] {
			t.Fatalf("general-purpose profile omitted release context question %q", id)
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "general-purpose.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"secret", "password", "token", "credentialValue", "consent"} {
		if strings.Contains(strings.ToLower(string(data)), strings.ToLower(forbidden)) {
			t.Fatalf("general-purpose profile contains forbidden sensitive or consent field %q", forbidden)
		}
	}

	service := Service{
		ProfileDir: func(context.Context) (string, error) { return dir, nil },
		Catalog: func(context.Context) ([]Scenario, error) {
			return []Scenario{
				{Name: "scenario-to-desktop"}, {Name: "scenario-to-cloud"}, {Name: "scenario-to-android"}, {Name: "scenario-to-ios"}, {Name: "vrooli-bridge"}, {Name: "landing-page-business-suite"}, {Name: "scenario-authenticator"},
			}, nil
		},
	}
	local, err := service.Evaluate(context.Background(), "general-purpose", map[string]any{"purposes": []any{"use-local-apps"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !local.Valid || len(local.Questions) != 1 {
		t.Fatalf("local-purpose evaluation = %+v, want only the purpose question and no release requirements", local)
	}
	guarded, err := service.Evaluate(context.Background(), "general-purpose", map[string]any{
		"purposes": []any{"use-local-apps"}, "hosting": "managed-vps", "mobilePlatforms": []any{"android"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !guarded.Valid || len(guarded.Recommendations) != 0 {
		t.Fatalf("hidden publishing answers changed local evaluation = %+v", guarded)
	}
	publish, err := service.Evaluate(context.Background(), "general-purpose", map[string]any{
		"purposes": []any{"publish-apps"}, "desktopPlatforms": []any{"linux"}, "mobilePlatforms": []any{"android"}, "hosting": "managed-vps",
		"distributionMethod": "direct-download", "accountOwnership": "operator-owned", "payments": "stripe", "mail": "provider", "downloadStorage": "s3", "remoteTargets": "connected-host",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !publish.Valid || !contains(publish.Scenarios, "scenario-to-android") || !contains(publish.Scenarios, "scenario-to-cloud") || !contains(publish.Scenarios, "vrooli-bridge") {
		t.Fatalf("publishing evaluation = %+v, want conditional release closure", publish)
	}
}

func TestRepositoryProfilesConformToCanonicalJSONSchema(t *testing.T) {
	profilesDir := repositoryProfilesDir(t)
	repositoryRoot := filepath.Clean(filepath.Join(profilesDir, "..", "..", ".."))
	schemaBytes, err := os.ReadFile(filepath.Join(repositoryRoot, ".vrooli", "schemas", "onboarding-profile.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("onboarding-profile.schema.json", bytes.NewReader(schemaBytes)); err != nil {
		t.Fatalf("register canonical onboarding profile schema: %v", err)
	}
	schema, err := compiler.Compile("onboarding-profile.schema.json")
	if err != nil {
		t.Fatalf("compile canonical onboarding profile schema: %v", err)
	}

	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(profilesDir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var document any
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatalf("decode %s: %v", entry.Name(), err)
		}
		if err := schema.Validate(document); err != nil {
			t.Errorf("profile %s does not conform to canonical schema: %v", entry.Name(), err)
		}
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
