package clianatomy

import (
	"os"
	"path/filepath"
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
func Register(){ cliapp.LoadFromManifest(manifest, "search", nil) }`,
		"domains/validate/register.go": `package validate
func Register(){ cliapp.LoadFromManifest(manifest, "validate", nil) }`,
		"manifest.json": `{"groups":[{"name":"search","commands":[{"name":"query"}]},{"name":"validate","commands":[{"name":"scenario"}]}]}`,
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
	negative := Search(root, "sample-cli", "underwater penguin invoice")
	if len(negative.Results) != 0 || len(negative.Degradations) == 0 {
		t.Fatalf("negative query should degrade without a strong hit: %+v", negative)
	}
}
