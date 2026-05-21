package manifest_test

import (
	"strings"
	"testing"

	"architecture-cartographer/internal/manifest"
)

const yamlSample = `
manifest_version: v1
scenario: demo
domains:
  - name: graph
    paths:
      - api/internal/graph/**
    glossary:
      - GraphSnapshot
  - name: manifest
    paths:
      - api/internal/manifest/**
    allowed_dependencies:
      - graph
shared_substrate:
  - api/internal/clock/**
signal_weights:
  path-token: 1.7
thresholds:
  - tier: auto_place
    min_value: 0.85
transitional:
  - id: tmp
    kind: allow_cycle
    locator: x -> y
    rationale: WIP
    expires_when: after:2026-09-01
`

const jsonSample = `{
  "manifest_version": "v1",
  "scenario": "demo",
  "domains": [
    {"name": "graph", "paths": ["api/internal/graph/**"]}
  ],
  "thresholds": [{"tier": "auto_place", "min_value": 0.85}]
}`

func TestParse_YAML_RoundTrip(t *testing.T) {
	m, ct, diags, err := manifest.Parse([]byte(yamlSample), manifest.ContentTypeUnspecified)
	if err != nil {
		t.Fatalf("Parse: %v (diags=%v)", err, diags)
	}
	if ct != manifest.ContentTypeYAML {
		t.Fatalf("expected YAML detection, got %q", ct)
	}
	if m.Scenario != "demo" {
		t.Fatalf("Scenario: %q", m.Scenario)
	}
	if len(m.Domains) != 2 {
		t.Fatalf("Domains: %+v", m.Domains)
	}
	if m.Domains[0].Glossary[0] != "GraphSnapshot" {
		t.Fatalf("Glossary not parsed: %+v", m.Domains[0].Glossary)
	}
	if m.SignalWeights.Weights["path-token"] != 1.7 {
		t.Fatalf("SignalWeights overlay missing: %+v", m.SignalWeights)
	}
	if m.ContentHash == "" {
		t.Fatalf("ContentHash should be set")
	}
}

func TestParse_JSON_DetectedFromBrace(t *testing.T) {
	m, ct, _, err := manifest.Parse([]byte(jsonSample), manifest.ContentTypeUnspecified)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if ct != manifest.ContentTypeJSON {
		t.Fatalf("expected JSON detection, got %q", ct)
	}
	if m.Thresholds[0].Tier != "auto_place" {
		t.Fatalf("Thresholds: %+v", m.Thresholds)
	}
}

func TestParse_HintWinsOverDetection(t *testing.T) {
	if _, _, _, err := manifest.Parse([]byte(jsonSample), manifest.ContentTypeJSON); err != nil {
		t.Fatalf("explicit JSON hint should parse: %v", err)
	}
}

func TestParse_EmptyInputReturnsError(t *testing.T) {
	_, _, diags, err := manifest.Parse([]byte(""), manifest.ContentTypeUnspecified)
	if err == nil {
		t.Fatalf("expected error for empty source")
	}
	if len(diags) != 1 || diags[0].Code != "MANIFEST_EMPTY_SOURCE" {
		t.Fatalf("expected EMPTY_SOURCE diag, got %+v", diags)
	}
}

func TestParse_InvalidYAMLDiagnosed(t *testing.T) {
	bad := []byte("scenario: demo\ndomains: [ not closed")
	_, _, diags, err := manifest.Parse(bad, manifest.ContentTypeYAML)
	if err == nil {
		t.Fatalf("expected error for invalid YAML")
	}
	if len(diags) == 0 || !strings.Contains(diags[0].Message, "parse error") {
		t.Fatalf("expected parse-error diag, got %+v", diags)
	}
}

func TestParse_UnknownVersionDiagnosed(t *testing.T) {
	src := []byte(`{"manifest_version":"v99","scenario":"demo"}`)
	_, _, diags, err := manifest.Parse(src, manifest.ContentTypeJSON)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	found := false
	for _, d := range diags {
		if d.Code == "MANIFEST_UNKNOWN_VERSION" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected UNKNOWN_VERSION diag, got %+v", diags)
	}
}

func TestParse_ContentHashStableForSameInput(t *testing.T) {
	m1, _, _, _ := manifest.Parse([]byte(yamlSample), manifest.ContentTypeYAML)
	m2, _, _, _ := manifest.Parse([]byte(yamlSample), manifest.ContentTypeYAML)
	if m1.ContentHash != m2.ContentHash {
		t.Fatalf("ContentHash unstable: %q vs %q", m1.ContentHash, m2.ContentHash)
	}
}

func TestParse_ContentHashDiffersForDifferentInput(t *testing.T) {
	m1, _, _, _ := manifest.Parse([]byte(yamlSample), manifest.ContentTypeYAML)
	m2, _, _, _ := manifest.Parse([]byte(yamlSample+"\n# trailing comment\n"), manifest.ContentTypeYAML)
	if m1.ContentHash == m2.ContentHash {
		t.Fatalf("ContentHash collided on different inputs")
	}
}
