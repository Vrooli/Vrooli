package commandref

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	"cli-health/internal/aisearch"
)

type fakeDiscovery struct {
	scenarios      []string
	records        map[string][]aisearch.CommandRecord
	external       []aisearch.ExternalCLI
	extRecs        map[string][]aisearch.CommandRecord
	refreshRecords map[string][]aisearch.CommandRecord
	refreshCalls   []string
}

func (f fakeDiscovery) ListScenarios(context.Context) ([]string, error) {
	return append([]string(nil), f.scenarios...), nil
}

func (f fakeDiscovery) Discover(_ context.Context, scenario string) ([]aisearch.CommandRecord, error) {
	return append([]aisearch.CommandRecord(nil), f.records[scenario]...), nil
}

func (f fakeDiscovery) ListExternalCLIs() []aisearch.ExternalCLI {
	return append([]aisearch.ExternalCLI(nil), f.external...)
}

func (f fakeDiscovery) DiscoverExternal(_ context.Context, cli aisearch.ExternalCLI) ([]aisearch.CommandRecord, error) {
	return append([]aisearch.CommandRecord(nil), f.extRecs[cli.Name]...), nil
}

func (f *fakeDiscovery) RefreshOwner(_ context.Context, owner string) ([]aisearch.CommandRecord, bool, error) {
	f.refreshCalls = append(f.refreshCalls, owner)
	records, ok := f.refreshRecords[owner]
	if !ok {
		return nil, false, nil
	}
	return append([]aisearch.CommandRecord(nil), records...), true, nil
}

func TestValidateManifestBackedCommandValidatesArguments(t *testing.T) {
	svc := Service{Discovery: fakeDiscovery{
		scenarios: []string{"demo"},
		records: map[string][]aisearch.CommandRecord{
			"demo": {{
				Origin:   "demo",
				Group:    "items",
				Name:     "get",
				FullPath: "demo items get",
				Source:   aisearch.SourceManifest,
				Args: &cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "id", Required: true}},
					Flags:       []cliapp.Flag{{Name: "verbose", Bool: true}},
				},
			}},
		},
	}}

	got := svc.Validate(context.Background(), Request{CommandText: "demo items get abc --verbose"})
	if got.Verdict != VerdictValid {
		t.Fatalf("verdict = %s, want %s (%+v)", got.Verdict, VerdictValid, got.Issues)
	}
	if got.Level != LevelArgumentShapeValidated {
		t.Fatalf("level = %s, want %s", got.Level, LevelArgumentShapeValidated)
	}

	bad := svc.Validate(context.Background(), Request{CommandText: "demo items get abc --bogus"})
	if bad.Verdict != VerdictInvalid {
		t.Fatalf("bad verdict = %s, want %s", bad.Verdict, VerdictInvalid)
	}
	if len(bad.Issues) == 0 || bad.Issues[0].Code != "invalid_arguments" {
		t.Fatalf("bad issues = %+v, want invalid_arguments", bad.Issues)
	}
}

func TestValidateTokenizerAcceptsQuotingEscapesAndInlineFlagValues(t *testing.T) {
	svc := Service{Discovery: fakeDiscovery{
		scenarios: []string{"demo"},
		records: map[string][]aisearch.CommandRecord{
			"demo": {{
				Origin:   "demo",
				Group:    "items",
				Name:     "get",
				FullPath: "demo items get",
				Source:   aisearch.SourceManifest,
				Args: &cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "id", Required: true},
						{Name: "extra", Repeated: true},
					},
					Flags: []cliapp.Flag{{Name: "title"}},
				},
			}},
		},
	}}

	got := svc.Validate(context.Background(), Request{CommandText: `demo items get "abc 123" escaped\ value --title=hello`})
	if got.Verdict != VerdictValid {
		t.Fatalf("verdict = %s, want %s (%+v)", got.Verdict, VerdictValid, got.Issues)
	}
	if got.Level != LevelArgumentShapeValidated {
		t.Fatalf("level = %s, want %s", got.Level, LevelArgumentShapeValidated)
	}
}

func TestValidateHelpDerivedCommandReturnsPartial(t *testing.T) {
	svc := Service{Discovery: fakeDiscovery{
		external: []aisearch.ExternalCLI{{Name: "vrooli", Binary: "vrooli"}},
		extRecs: map[string][]aisearch.CommandRecord{
			"vrooli": {{Origin: "vrooli", FullPath: "vrooli scenario test", Source: aisearch.SourceHelp}},
		},
	}}

	got := svc.Validate(context.Background(), Request{CommandText: "vrooli scenario test cli-health"})
	if got.Verdict != VerdictPartial {
		t.Fatalf("verdict = %s, want %s", got.Verdict, VerdictPartial)
	}
	if got.Level != LevelCommandExists {
		t.Fatalf("level = %s, want %s", got.Level, LevelCommandExists)
	}
}

func TestValidateQualifierSkipsCurrentExistence(t *testing.T) {
	got := (Service{}).Validate(context.Background(), Request{
		CommandText: "missing command",
		Qualifiers:  []string{"future"},
	})
	if got.Verdict != VerdictSkipped {
		t.Fatalf("verdict = %s, want %s", got.Verdict, VerdictSkipped)
	}
	if got.Level != LevelSkippedByQualifier {
		t.Fatalf("level = %s, want %s", got.Level, LevelSkippedByQualifier)
	}
}

func TestValidateRejectsUnsupportedShellSyntax(t *testing.T) {
	cases := []string{
		"demo list | jq .",
		"demo list > out.txt",
		"demo list && demo other",
		"demo list $(pwd)",
		"demo list `pwd`",
		"demo list\nother command",
	}
	for _, commandText := range cases {
		got := (Service{}).Validate(context.Background(), Request{CommandText: commandText})
		if got.Verdict != VerdictUnsupported {
			t.Fatalf("%q verdict = %s, want %s", commandText, got.Verdict, VerdictUnsupported)
		}
		if got.Level != LevelUnsupportedSyntax {
			t.Fatalf("%q level = %s, want %s", commandText, got.Level, LevelUnsupportedSyntax)
		}
	}
}

func TestValidateRedirectionGuidanceExplainsPlaceholders(t *testing.T) {
	got := (Service{}).Validate(context.Background(), Request{CommandText: "cli-health configure token <token>"})
	if got.Verdict != VerdictUnsupported {
		t.Fatalf("verdict = %s, want %s", got.Verdict, VerdictUnsupported)
	}
	if len(got.Guidance) == 0 {
		t.Fatalf("guidance is empty")
	}
	guidance := strings.Join(got.Guidance, "\n")
	if !strings.Contains(guidance, "shell redirection operators") {
		t.Fatalf("guidance = %q, want redirection explanation", guidance)
	}
	if !strings.Contains(guidance, "wrap it in double quotes") {
		t.Fatalf("guidance = %q, want quoted-placeholder recommendation", guidance)
	}
	if strings.Contains(guidance, "TOKEN_VALUE") {
		t.Fatalf("guidance = %q, must no longer recommend TOKEN_VALUE placeholders", guidance)
	}
}

func TestValidateInvalidCommandReturnsSuggestions(t *testing.T) {
	svc := Service{Discovery: fakeDiscovery{
		scenarios: []string{"demo"},
		records: map[string][]aisearch.CommandRecord{
			"demo": {
				{Origin: "demo", FullPath: "demo docs health", Source: aisearch.SourceManifest},
				{Origin: "demo", FullPath: "demo docs list", Source: aisearch.SourceManifest},
			},
		},
	}}

	got := svc.Validate(context.Background(), Request{CommandText: "demo docs healt"})
	if got.Verdict != VerdictInvalid {
		t.Fatalf("verdict = %s, want %s", got.Verdict, VerdictInvalid)
	}
	if len(got.Suggestions) == 0 || got.Suggestions[0].Command != "demo docs health" {
		t.Fatalf("suggestions = %+v, want demo docs health first", got.Suggestions)
	}
}

func TestValidateDoesNotRefreshByDefault(t *testing.T) {
	discovery := &fakeDiscovery{
		scenarios: []string{"demo"},
		records: map[string][]aisearch.CommandRecord{
			"demo": {{Origin: "demo", FullPath: "demo docs health", Source: aisearch.SourceManifest}},
		},
		refreshRecords: map[string][]aisearch.CommandRecord{
			"demo": {{Origin: "demo", FullPath: "demo docs heal", Source: aisearch.SourceManifest}},
		},
	}
	svc := Service{Discovery: discovery}

	got := svc.Validate(context.Background(), Request{CommandText: "demo docs heal"})
	if got.Verdict != VerdictInvalid {
		t.Fatalf("verdict = %s, want %s", got.Verdict, VerdictInvalid)
	}
	if len(discovery.refreshCalls) != 0 {
		t.Fatalf("refresh calls = %+v, want none", discovery.refreshCalls)
	}
}

func TestValidateRefreshesOnceOnMiss(t *testing.T) {
	discovery := &fakeDiscovery{
		scenarios: []string{"demo"},
		records: map[string][]aisearch.CommandRecord{
			"demo": {{Origin: "demo", FullPath: "demo docs health", Source: aisearch.SourceManifest}},
		},
		refreshRecords: map[string][]aisearch.CommandRecord{
			"demo": {{
				Origin:   "demo",
				FullPath: "demo docs heal",
				Source:   aisearch.SourceManifest,
				Args:     &cliapp.ArgSchema{},
			}},
		},
	}
	svc := Service{Discovery: discovery}

	got := svc.Validate(context.Background(), Request{
		CommandText:   "demo docs heal",
		RefreshPolicy: "on_miss",
	})
	if got.Verdict != VerdictValid {
		t.Fatalf("verdict = %s, want %s (%+v)", got.Verdict, VerdictValid, got.Issues)
	}
	if got.CanonicalCommand != "demo docs heal" {
		t.Fatalf("canonical = %q, want demo docs heal", got.CanonicalCommand)
	}
	if len(discovery.refreshCalls) != 1 || discovery.refreshCalls[0] != "demo" {
		t.Fatalf("refresh calls = %+v, want [demo]", discovery.refreshCalls)
	}
}
