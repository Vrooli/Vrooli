package clianatomy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchReportsScenarioAppCompositionAndNegativeDegradation(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "scenarios", "sample-cli", "cli")
	for name, body := range map[string]string{
		"app.go": `package main
import "github.com/vrooli/cli-core/cliapp"
func NewApp(){ cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{CommandGroups: domains.CommandGroups, SubcommandGroups: subcommandGroups}) }`,
		"domains/domains.go": `package domains
func SubcommandGroups(){ for _, reg := range []func(){search.Register, validate.Register} { reg() } }`,
		"domains/search/register.go": `package search
func Register(){ bindings := map[string]any{"SearchService.Search": nil}; group, _ := cliapp.LoadFromManifest(manifest, "search", bindings); group.DefaultSubcommand = "query" }`,
		"domains/validate/register.go": `package validate
func Register(){ cliapp.LoadFromManifest(manifest, "validate", nil) }`,
		"manifest.json": `{"groups":[{"name":"search","commands":[{"name":"query","binding":{"kind":"connect-rpc","service":"SearchService","method":"Search"}}]},{"name":"validate","commands":[{"name":"scenario","binding":{"kind":"connect-rpc","service":"ValidateService","method":"Scenario"}}]},{"name":"orphan","commands":[{"name":"run","binding":{"kind":"local","handler":"run"}}]}]}`,
	} {
		p := filepath.Join(base, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got := Search(root, "sample-cli", "ScenarioApp command groups subcommand composition")
	if len(got.Results) != 1 || got.Results[0].Attestation != "PARTIAL" || len(got.Results[0].Evidence) < 5 {
		t.Fatalf("composition evidence missing: %+v", got)
	}
	if !strings.Contains(got.Results[0].Text, "all declared handler keys match") || !strings.Contains(got.Results[0].Text, "default search → query (manifest command exists)") || !strings.Contains(got.Results[0].Text, "NOT REGISTERED") {
		t.Fatalf("group/leaf compatibility matrix missing: %+v", got.Results[0])
	}

	registerPath := filepath.Join(base, "domains/search/register.go")
	register, err := os.ReadFile(registerPath)
	if err != nil {
		t.Fatal(err)
	}
	// An implementation key that no manifest command consumes remains visible
	// as UNUSED rather than being mistaken for a working command.
	extraBinding := []byte(strings.Replace(string(register), `"SearchService.Search": nil`, `"SearchService.Search": nil, "SearchService.Unused": nil`, 1))
	if err := os.WriteFile(registerPath, extraBinding, 0644); err != nil {
		t.Fatal(err)
	}
	extra := Search(root, "sample-cli", "ScenarioApp command group leaf handler manifest")
	if len(extra.Results) != 1 || !strings.Contains(extra.Results[0].Text, "UNUSED handlers SearchService.Unused") || extra.Results[0].Attestation != "PARTIAL" {
		t.Fatalf("unused handler must remain partial: %+v", extra)
	}

	// A declared command with no matching concrete handler key must be reported
	// as MISSING. The source analyzer never upgrades this to runtime evidence.
	register = []byte(strings.Replace(string(register), `"SearchService.Search": nil`, `"SearchService.Other": nil`, 1))
	if err := os.WriteFile(registerPath, register, 0644); err != nil {
		t.Fatal(err)
	}
	missing := Search(root, "sample-cli", "ScenarioApp command group leaf handler manifest")
	if len(missing.Results) != 1 || !strings.Contains(missing.Results[0].Text, "MISSING handlers") || missing.Results[0].Attestation != "PARTIAL" {
		t.Fatalf("missing handler must remain partial: %+v", missing)
	}

	// If a registered constructor cannot be opened, the analyzer must not call
	// manifest-only groups absent: membership is UNKNOWN.
	domainsPath := filepath.Join(base, "domains/domains.go")
	domains, err := os.ReadFile(domainsPath)
	if err != nil {
		t.Fatal(err)
	}
	domains = []byte(strings.Replace(string(domains), "validate.Register", "absent.Register", 1))
	if err := os.WriteFile(domainsPath, domains, 0644); err != nil {
		t.Fatal(err)
	}
	unknown := Search(root, "sample-cli", "ScenarioApp group registration manifest composition")
	if len(unknown.Results) != 1 || !strings.Contains(unknown.Results[0].Text, "registration membership UNKNOWN") || unknown.Results[0].Attestation != "PARTIAL" {
		t.Fatalf("unresolved constructor membership must remain UNKNOWN: %+v", unknown)
	}

	invalidScope := Search(root, "../outside", "CLI manifest group")
	if len(invalidScope.Results) != 0 || len(invalidScope.Degradations) == 0 || invalidScope.Degradations[0].Code != "invalid_scope_or_query" {
		t.Fatalf("invalid scenario scope must degrade explicitly: %+v", invalidScope)
	}
	negative := Search(root, "sample-cli", "underwater penguin invoice")
	if len(negative.Results) != 0 || len(negative.Degradations) == 0 {
		t.Fatalf("negative query should degrade without a strong hit: %+v", negative)
	}
}
