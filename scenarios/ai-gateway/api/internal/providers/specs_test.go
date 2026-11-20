package providers

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDefaultProviderSpecsPreservesSeed(t *testing.T) {
	want := []ProviderSpec{
		{ID: ProviderOllama, CommandName: "resource-ollama", Locality: "local", Family: FamilyOllama, Text: true},
		{ID: ProviderOpenRouter, CommandName: "resource-openrouter", Locality: "remote", Family: FamilyOpenAICompatible, Text: true, SpendCapable: true},
		{ID: ProviderOrcaRouter, CommandName: "resource-orcarouter", Locality: "remote", Family: FamilyOpenAICompatible, Text: true},
		{ID: ProviderAbliteration, CommandName: "resource-abliteration", Locality: "remote", Family: FamilyOpenAICompatible, Text: true},
		{ID: ProviderPiiDetector, CommandName: "resource-pii-detector", Locality: "local", Family: FamilyPII},
		{ID: ProviderOllaya, CommandName: "resource-ollaya", Locality: "local", Family: FamilyDecision},
	}
	got := DefaultProviderSpecs()
	if len(got) != len(want) {
		t.Fatalf("DefaultProviderSpecs() returned %d specs, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("spec %d = %+v, want %+v", index, got[index], want[index])
		}
	}
}

func TestDefaultProviderSpecsReturnsCopy(t *testing.T) {
	first := DefaultProviderSpecs()
	first[0].ID = "mutated"
	first[0].Family = "mutated"
	second := DefaultProviderSpecs()
	if second[0].ID != ProviderOllama || second[0].Family != FamilyOllama {
		t.Fatalf("registry was mutated through a caller copy: %+v", second[0])
	}
}

func TestLoadProviderSpecsValid(t *testing.T) {
	specs, err := LoadProviderSpecs([]byte(`[
		{"id":"one","command_name":"resource-one","locality":"local","family":"ollama","text":true},
		{"id":"two","command_name":"resource-two","locality":"remote","family":"openai-compatible","spend_capable":true}
	]`))
	if err != nil {
		t.Fatalf("LoadProviderSpecs() error = %v", err)
	}
	if len(specs) != 2 || specs[0].ID != "one" || !specs[1].SpendCapable {
		t.Fatalf("LoadProviderSpecs() = %+v", specs)
	}
}

func TestLoadProviderSpecsRejectsInvalidSeeds(t *testing.T) {
	tests := []struct {
		name    string
		seed    string
		wantErr string
	}{
		{
			name:    "malformed JSON",
			seed:    `[`,
			wantErr: "decode provider specs",
		},
		{
			name:    "missing id",
			seed:    `[{"command_name":"resource-one","locality":"local","family":"ollama"}]`,
			wantErr: "id is required",
		},
		{
			name:    "padded id",
			seed:    `[{"id":" one","command_name":"resource-one","locality":"local","family":"ollama"}]`,
			wantErr: "surrounding whitespace",
		},
		{
			name:    "duplicate id",
			seed:    `[{"id":"one","command_name":"resource-one","locality":"local","family":"ollama"},{"id":"one","command_name":"resource-two","locality":"local","family":"ollama"}]`,
			wantErr: "duplicate id",
		},
		{
			name:    "missing command",
			seed:    `[{"id":"one","locality":"local","family":"ollama"}]`,
			wantErr: "command_name is required",
		},
		{
			name:    "padded command",
			seed:    `[{"id":"one","command_name":" resource-one","locality":"local","family":"ollama"}]`,
			wantErr: "surrounding whitespace",
		},
		{
			name:    "bad locality",
			seed:    `[{"id":"one","command_name":"resource-one","locality":"nearby","family":"ollama"}]`,
			wantErr: "must be local or remote",
		},
		{
			name:    "missing family",
			seed:    `[{"id":"one","command_name":"resource-one","locality":"local"}]`,
			wantErr: "family is required",
		},
		{
			name:    "unknown family",
			seed:    `[{"id":"one","command_name":"resource-one","locality":"local","family":"carrier-pigeon"}]`,
			wantErr: "not executable by the adapter",
		},
		{
			name:    "empty registry",
			seed:    `[]`,
			wantErr: "declares no providers",
		},
		{
			name:    "trailing content",
			seed:    `[] []`,
			wantErr: "trailing JSON content",
		},
		{
			name:    "unknown field",
			seed:    `[{"id":"one","command_name":"resource-one","locality":"local","family":"ollama","typo":true}]`,
			wantErr: "unknown field",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadProviderSpecs([]byte(test.seed))
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("LoadProviderSpecs() error = %v, want containing %q", err, test.wantErr)
			}
		})
	}
}

func TestSpecForAndDerivedSets(t *testing.T) {
	spec, ok := SpecFor("  openrouter  ")
	if !ok || spec.CommandName != "resource-openrouter" {
		t.Fatalf("SpecFor(openrouter) = %+v, %v", spec, ok)
	}
	if _, ok := SpecFor("not-a-provider"); ok {
		t.Fatal("SpecFor accepted an unregistered provider")
	}

	text := TextProviderIDs()
	for _, id := range []string{ProviderOllama, ProviderOpenRouter, ProviderOrcaRouter, ProviderAbliteration} {
		if _, ok := text[id]; !ok {
			t.Errorf("TextProviderIDs() omitted %q", id)
		}
	}
	for _, id := range []string{ProviderPiiDetector, ProviderOllaya, ProviderMetered} {
		if _, ok := text[id]; ok {
			t.Errorf("TextProviderIDs() included non-text provider %q", id)
		}
	}

	commands := AllowedCommands()
	for _, spec := range DefaultProviderSpecs() {
		if _, ok := commands[spec.CommandName]; !ok {
			t.Errorf("AllowedCommands() omitted %q", spec.CommandName)
		}
	}
}

func TestSupportedFamiliesRejectsMeteredSeedRow(t *testing.T) {
	families := SupportedFamilies()
	for _, family := range []string{FamilyOllama, FamilyOpenAICompatible, FamilyPII, FamilyDecision} {
		if _, ok := families[family]; !ok {
			t.Errorf("SupportedFamilies() omitted %q", family)
		}
	}
	if _, ok := families[FamilyMetered]; ok {
		t.Error("SupportedFamilies() must not include the metered door family")
	}

	seed, err := json.Marshal([]ProviderSpec{{
		ID: "metered", CommandName: "resource-metered", Locality: "remote", Family: FamilyMetered,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadProviderSpecs(seed); err == nil || !strings.Contains(err.Error(), "not executable by the adapter") {
		t.Fatalf("LoadProviderSpecs() error = %v, want unsupported family rejection", err)
	}
}
