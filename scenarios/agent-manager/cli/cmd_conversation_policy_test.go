package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/conversationpolicy"
)

// Named sample values shared across the policy CLI tests.
const (
	cliOllama      = "ollama"
	cliSecretModel = "secret-model"
)

// policyApp sets up an App whose policy file is isolated to a temp path via
// the env override, returning the resolved path for assertions.
func policyApp(t *testing.T) (*App, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "conversation-policy.json")
	t.Setenv(conversationpolicy.FileEnvVar, path)
	return &App{}, path
}

// TestConversationPolicyAddListRemoveRoundTrip drives the local file-backed
// policy CLI: add two rules, list them, remove one, and confirm the on-disk
// document matches at each step.
func TestConversationPolicyAddListRemoveRoundTrip(t *testing.T) {
	app, path := policyApp(t)

	if err := app.conversationPolicy([]string{"add", "--action", "exclude", "--enforce-at", "import", "--harness", cliOllama, "--name", "no-local"}); err != nil {
		t.Fatalf("add import rule: %v", err)
	}
	if err := app.conversationPolicy([]string{"add", "--action", "exclude", "--enforce-at", "search", "--model", cliSecretModel}); err != nil {
		t.Fatalf("add search rule: %v", err)
	}

	doc := loadPolicyDoc(t, path)
	if len(doc.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(doc.Rules))
	}
	if doc.Rules[0].Harness != cliOllama || doc.Rules[0].EnforceAt != conversationpolicy.EnforceImport {
		t.Fatalf("unexpected first rule: %+v", doc.Rules[0])
	}

	listing := captureStdout(t, func() error { return app.conversationPolicy([]string{"list"}) })
	for _, want := range []string{"no-local", cliSecretModel} {
		if !strings.Contains(listing, want) {
			t.Fatalf("list output missing %q: %s", want, listing)
		}
	}

	if err := app.conversationPolicy([]string{"remove", "0"}); err != nil {
		t.Fatalf("remove rule 0: %v", err)
	}
	doc = loadPolicyDoc(t, path)
	if len(doc.Rules) != 1 || doc.Rules[0].Model != cliSecretModel {
		t.Fatalf("remove did not leave the search rule: %+v", doc.Rules)
	}
}

func loadPolicyDoc(t *testing.T, path string) conversationpolicy.Document {
	t.Helper()
	doc, err := conversationpolicy.LoadDocument(path)
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	return doc
}

// TestConversationPolicyRejectsInvalidRule proves a malformed add is refused
// and never written (no silent partial policy).
func TestConversationPolicyRejectsInvalidRule(t *testing.T) {
	app, path := policyApp(t)
	if err := app.conversationPolicy([]string{"add", "--action", "drop", "--enforce-at", "import"}); err == nil {
		t.Fatalf("expected invalid action to be rejected")
	}
	if err := app.conversationPolicy([]string{"add", "--action", "exclude"}); err == nil {
		t.Fatalf("expected missing enforce-at to be rejected")
	}
	if doc := loadPolicyDoc(t, path); len(doc.Rules) != 0 {
		t.Fatalf("invalid rules must not be persisted, got %+v", doc.Rules)
	}
}

// TestConversationPolicyWarnsOnImportUnobservableField pins A-P1-7: adding an
// import rule that matches on model (unobservable at the import seam) still
// succeeds but surfaces a warning so the operator is not misled.
func TestConversationPolicyWarnsOnImportUnobservableField(t *testing.T) {
	app, path := policyApp(t)
	out := captureStdout(t, func() error {
		return app.conversationPolicy([]string{"add", "--action", "exclude", "--enforce-at", "import", "--model", cliSecretModel})
	})
	if !strings.Contains(out, "warning") || !strings.Contains(out, "model") {
		t.Fatalf("expected an import-unobservable warning naming model, got: %s", out)
	}
	// The rule is still persisted (warn, do not reject).
	if doc := loadPolicyDoc(t, path); len(doc.Rules) != 1 {
		t.Fatalf("import+model rule should still be saved, got %+v", doc.Rules)
	}
}

// TestConversationPolicyTestEvaluation drives `policy test`, confirming it
// prints the matching rule and the resulting import/search outcome, and that
// --json exposes the stable imported/searchable/import/search keys.
func TestConversationPolicyTestEvaluation(t *testing.T) {
	app, path := policyApp(t)
	seedPolicy(t, path,
		conversationpolicy.Rule{Name: "no-ollama", Action: conversationpolicy.ActionExclude, EnforceAt: conversationpolicy.EnforceImport, Harness: cliOllama},
		conversationpolicy.Rule{Name: "hide-secret", Action: conversationpolicy.ActionExclude, EnforceAt: conversationpolicy.EnforceSearch, Model: cliSecretModel},
	)

	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"import exclusion", []string{"test", "--harness", cliOllama}, []string{"EXCLUDED at import", "no-ollama"}},
		{"search hide", []string{"test", "--harness", "codex", "--model", cliSecretModel}, []string{"HIDDEN from search", "hide-secret"}},
		{"kept", []string{"test", "--harness", "claude-code", "--model", "sonnet"}, []string{"IMPORTED and SEARCHABLE"}},
	}
	for _, tc := range cases {
		out := captureStdout(t, func() error { return app.conversationPolicy(tc.args) })
		for _, want := range tc.want {
			if !strings.Contains(out, want) {
				t.Fatalf("%s: output missing %q: %s", tc.name, want, out)
			}
		}
	}

	// --json must expose the keys other tooling consumes.
	raw := captureStdout(t, func() error { return app.conversationPolicy([]string{"test", "--harness", cliOllama, "--json"}) })
	var payload struct {
		Imported   bool `json:"imported"`
		Searchable bool `json:"searchable"`
		Import     struct {
			Matched bool `json:"matched"`
			Hidden  bool `json:"hidden"`
		} `json:"import"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("parse --json output %q: %v", raw, err)
	}
	if payload.Imported || payload.Searchable || !payload.Import.Hidden || !payload.Import.Matched {
		t.Fatalf("json shape did not reflect an import exclusion: %+v (%s)", payload, raw)
	}
}

func seedPolicy(t *testing.T, path string, rules ...conversationpolicy.Rule) {
	t.Helper()
	if err := conversationpolicy.SaveDocument(path, conversationpolicy.Document{Rules: rules}); err != nil {
		t.Fatalf("seed policy: %v", err)
	}
}
