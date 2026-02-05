package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func TestCampaignAutoOptionsValidation(t *testing.T) {
	if (campaignAutoOptions{}).enabled() {
		t.Fatal("expected empty options to be disabled")
	}

	opts := campaignAutoOptions{location: "/tmp", tag: "demo"}
	if !opts.enabled() {
		t.Fatal("expected options with location+tag to be enabled")
	}
	if err := opts.validate(); err != nil {
		t.Fatalf("expected valid options, got error: %v", err)
	}

	opts = campaignAutoOptions{location: "/tmp"}
	if err := opts.validate(); err == nil {
		t.Fatal("expected validation error when tag missing")
	}
}

func TestAPIPath(t *testing.T) {
	makeApp := func(base string) *App {
		return &App{
			core: &cliapp.ScenarioApp{
				HTTPClient: cliutil.NewHTTPClient(cliutil.HTTPClientOptions{
					BaseOptions: cliutil.APIBaseOptions{Override: base},
				}),
			},
		}
	}

	app := makeApp("http://localhost:8080")
	if got := app.apiPath("campaigns"); got != "/api/v1/campaigns" {
		t.Fatalf("expected /api/v1/campaigns, got %q", got)
	}

	app = makeApp("http://localhost:8080/api/v1")
	if got := app.apiPath("/campaigns"); got != "/campaigns" {
		t.Fatalf("expected /campaigns, got %q", got)
	}
}

func TestParseJSONInput(t *testing.T) {
	if value, err := parseJSONInput(""); err != nil || value != nil {
		t.Fatalf("expected nil result for empty input, got %v (err=%v)", value, err)
	}

	if _, err := parseJSONInput("{invalid json"); err == nil {
		t.Fatal("expected error for invalid JSON input")
	}

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "metadata.json")
	if err := os.WriteFile(filePath, []byte(`{"team":"core","count":2}`), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	value, err := parseJSONInput("@" + filePath)
	if err != nil {
		t.Fatalf("expected valid JSON from file, got error: %v", err)
	}
	if value["team"] != "core" {
		t.Fatalf("expected team=core, got %v", value["team"])
	}
}

func TestBuildQueryAndJoinPatterns(t *testing.T) {
	values := buildQuery(map[string]string{
		"limit":   "10",
		"empty":   "",
		"trimmed": "  ",
	})
	if got := values.Get("limit"); got != "10" {
		t.Fatalf("expected limit=10, got %q", got)
	}
	if values.Get("empty") != "" || values.Get("trimmed") != "" {
		t.Fatal("expected empty query params to be skipped")
	}

	patterns := joinPatterns([]string{" *.go ", "", "  *.ts"})
	if strings.Contains(patterns, "  ") || patterns != "*.go,*.ts" {
		t.Fatalf("unexpected joined patterns: %q", patterns)
	}
}

func TestNormalizePathListAndEnsureFilePath(t *testing.T) {
	paths := normalizePathList([]string{" ./one ", "one", "", "two", "two"})
	expected := []string{"./one", "one", "two"}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("expected %v, got %v", expected, paths)
	}

	if _, err := ensureFilePath(" "); err == nil {
		t.Fatal("expected error for empty file path")
	}

	cleaned, err := ensureFilePath("foo/../bar")
	if err != nil {
		t.Fatalf("unexpected error cleaning path: %v", err)
	}
	if cleaned != "bar" {
		t.Fatalf("expected cleaned path to be bar, got %q", cleaned)
	}
}
