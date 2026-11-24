package facts

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestScanAmbientSeamsPreservesLayerAndArgumentContext(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"api/provider.go": `package demo
import ("log"; "net/http"; "os"; "time")
func NewProvider(*http.Client, *log.Logger) {}
func Module() { NewProvider(http.DefaultClient, log.Default()); _ = os.Getenv("MODE"); _ = time.Now() }
`,
		"api/provider_test.go": `package demo
import "time"
func TestProviderClock() { _ = time.Now() }
`,
	}
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	response, err := scanAmbientSeams(root, "sample", "http client environment clock test logger provider", 12)
	if err != nil {
		t.Fatal(err)
	}
	var environment, client, testClock, logger bool
	for _, hit := range response.Results {
		if hit.Kind == "environment" && hit.SourcePath == "api/provider.go" && hit.SourceSHA256 != "" && hit.ContextSymbol == "Module" && hit.SourceLayer == "NON_TEST_SOURCE_CANDIDATE" {
			environment = slices.Contains(hit.MatchedTerms, "environment") && slices.Contains(hit.MatchedFields, "kind")
		}
		if hit.Kind == "http_client" && hit.ArgumentCallee == "NewProvider" && hit.ArgumentContextStatus == "STATIC_ARGUMENT_RELATIONSHIP" {
			client = slices.Contains(hit.MatchedTerms, "client") && slices.Contains(hit.MatchedFields, "argument_callee")
		}
		if hit.Kind == "clock" && hit.SourceLayer == "TEST_SOURCE_CANDIDATE" && hit.ArgumentContextStatus == "UNKNOWN_STATIC_CONTEXT_ONLY" {
			testClock = true
		}
		if hit.Kind == "logger" && hit.ArgumentCallee == "NewProvider" && hit.ArgumentContextStatus == "STATIC_ARGUMENT_RELATIONSHIP" && slices.Contains(hit.MatchedTerms, "logger") {
			logger = true
		}
	}
	if !environment || !client || !testClock || !logger {
		t.Fatalf("static seam evidence missing or overstated: env=%t client=%t testclock=%t logger=%t response=%+v", environment, client, testClock, logger, response)
	}
}

func TestSearchAmbientSeamsNegativeQueryAbstains(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "api.go")
	if err := os.WriteFile(path, []byte("package sample\nfunc Run() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	response, err := scanAmbientSeams(root, "sample", "underwater penguin", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 0 || len(response.Degradations) == 0 || !strings.Contains(response.Degradations[len(response.Degradations)-1].Message, "No recognized ambient seam") {
		t.Fatalf("negative query should abstain: %+v", response)
	}
	if response.MatchStatus != "UNKNOWN" {
		t.Fatalf("negative query match status = %q, want UNKNOWN", response.MatchStatus)
	}
}

func TestScanAmbientSeamsReturnsExactTermToSourceEvidence(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "api.go")
	if err := os.WriteFile(path, []byte("package sample\nimport \"os\"\nfunc Run() { _ = os.Getenv(\"MODE\") }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	response, err := scanAmbientSeams(root, "sample", "environment", 5)
	if err != nil {
		t.Fatal(err)
	}
	if response.MatchStatus != "MATCHED" || len(response.Results) != 1 {
		t.Fatalf("exact query response = %+v", response)
	}
	hit := response.Results[0]
	if len(hit.MatchEvidence) != 1 {
		t.Fatalf("match evidence = %+v, want one exact field link", hit.MatchEvidence)
	}
	evidence := hit.MatchEvidence[0]
	if evidence.Term != "environment" || evidence.Field != "kind" || evidence.Value != "environment" || evidence.SourcePath != "api.go" || evidence.Line != 3 || evidence.SourceSHA256 != hit.SourceSHA256 {
		t.Fatalf("match evidence = %+v, want term/field/value and exact source revision", evidence)
	}

	substring, err := scanAmbientSeams(root, "sample", "env", 5)
	if err != nil {
		t.Fatal(err)
	}
	if substring.MatchStatus != "UNKNOWN" || len(substring.Results) != 0 {
		t.Fatalf("substring query should not match environment: %+v", substring)
	}
}

func TestScanAmbientSeamsMarksEqualScoreCandidatesUnknown(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "clock.go")
	body := "package sample\nimport \"time\"\nfunc First() { _ = time.Now() }\nfunc Second() { _ = time.Now() }\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	response, err := scanAmbientSeams(root, "sample", "clock", 5)
	if err != nil {
		t.Fatal(err)
	}
	if response.MatchStatus != "UNKNOWN" || len(response.Results) != 2 {
		t.Fatalf("tied response = %+v, want two candidates with UNKNOWN status", response)
	}
	for _, hit := range response.Results {
		if hit.MatchStatus != "UNKNOWN" {
			t.Errorf("candidate %q status = %q, want UNKNOWN", hit.ID, hit.MatchStatus)
		}
	}
	if !slices.ContainsFunc(response.Degradations, func(item AmbientSeamDegradation) bool { return item.Code == "ambiguous_match_tie" }) {
		t.Fatalf("tie degradation missing: %+v", response.Degradations)
	}
}

func TestScanAmbientSeamsUnsupportedSyntaxIsUnknown(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "clock.go")
	if err := os.WriteFile(path, []byte("package sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	response, err := scanAmbientSeams(root, "sample", `clock OR "environment"`, 5)
	if err != nil {
		t.Fatal(err)
	}
	if response.MatchStatus != "UNKNOWN" || len(response.Results) != 0 {
		t.Fatalf("unsupported syntax response = %+v, want no match and UNKNOWN", response)
	}
	if len(response.Degradations) != 1 || response.Degradations[0].Code != "query_unsupported_syntax" {
		t.Fatalf("unsupported syntax degradation = %+v", response.Degradations)
	}
}
