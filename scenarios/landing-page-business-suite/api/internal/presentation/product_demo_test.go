package presentation

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestCanonicalPlaybackURLMatchesSharedCorpus(t *testing.T) {
	data, err := os.ReadFile("testdata/playback-url-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Cases []struct {
			Name         string `json:"name"`
			Provider     string `json:"provider"`
			ExternalURL  string `json:"external_url"`
			Accepted     bool   `json:"accepted"`
			CanonicalURL string `json:"canonical_url"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if len(corpus.Cases) < 20 {
		t.Fatalf("shared playback corpus is unexpectedly small: %d cases", len(corpus.Cases))
	}
	for _, testCase := range corpus.Cases {
		t.Run(testCase.Name, func(t *testing.T) {
			got, err := CanonicalPlaybackURL(testCase.Provider, testCase.ExternalURL)
			if !testCase.Accepted {
				if err == nil {
					t.Fatalf("unsafe or unsupported URL accepted as %q", got)
				}
				return
			}
			if err != nil || got != testCase.CanonicalURL {
				t.Fatalf("CanonicalPlaybackURL() = %q, %v; want %q", got, err, testCase.CanonicalURL)
			}
		})
	}
}

func TestProductDemoRecordedAndFixtureModesAreDisjoint(t *testing.T) {
	recorded := map[string]any{
		"heading": "Recorded demo", "description": "A recording", "renderer_ref": "video",
		"poster_ref": "poster", "alt_text": "Recorded demo poster",
		"playback": map[string]any{
			"provider": "youtube", "external_url": "https://youtu.be/dQw4w9WgXcQ", "layout": "split",
			"play_label": "Play demo", "caption": "A configured recording", "unavailable_label": "Video unavailable",
		},
	}
	issues := &ValidationError{}
	validateProductDemo("recorded", recorded, "content", issues)
	if len(issues.Issues) != 0 {
		t.Fatalf("valid recorded demo rejected: %+v", issues.Issues)
	}

	fixture := map[string]any{"heading": "Fixture demo", "description": "An illustration", "renderer_ref": "workflow", "fixture_ref": "workflow", "alt_text": "Workflow illustration"}
	issues = &ValidationError{}
	validateProductDemo("static", fixture, "content", issues)
	if len(issues.Issues) != 0 {
		t.Fatalf("valid fixture demo rejected: %+v", issues.Issues)
	}

	for _, field := range []string{"playback", "poster_ref", "media_ref"} {
		bad := cloneStringAnyMap(fixture)
		bad[field] = map[string]any{"provider": "youtube"}
		if field != "playback" {
			bad[field] = "asset"
		}
		issues = &ValidationError{}
		validateProductDemo("interactive", bad, "content", issues)
		if !hasValidationIssue(issues, field) {
			t.Errorf("fixture mode accepted ignored %s: %+v", field, issues.Issues)
		}
	}

	for _, field := range []string{"fixture_ref", "media_ref", "playback", "poster_ref"} {
		bad := cloneStringAnyMap(recorded)
		if field == "playback" {
			bad[field] = nil
		} else if field == "poster_ref" {
			bad[field] = ""
		} else {
			bad[field] = "asset"
		}
		issues = &ValidationError{}
		validateProductDemo("recorded", bad, "content", issues)
		if !hasValidationIssue(issues, field) {
			t.Errorf("recorded mode accepted forbidden/missing %s: %+v", field, issues.Issues)
		}
	}

	bad := cloneStringAnyMap(recorded)
	bad["renderer_ref"] = "workflow"
	issues = &ValidationError{}
	validateProductDemo("recorded", bad, "content", issues)
	if !hasValidationIssue(issues, "renderer_ref") {
		t.Fatal("recorded mode accepted a non-video renderer")
	}
}

func TestProductDemoPlaybackRequiresAllTypedLabels(t *testing.T) {
	value := map[string]any{
		"provider": "youtube", "external_url": "https://youtu.be/dQw4w9WgXcQ", "layout": "stacked",
		"play_label": "", "caption": "Caption", "unavailable_label": "Unavailable",
	}
	issues := &ValidationError{}
	validateProductDemoPlayback(value, "content.playback", issues)
	if !hasValidationIssue(issues, "play_label") {
		t.Fatalf("missing playback label was accepted: %+v", issues.Issues)
	}
}

func cloneStringAnyMap(source map[string]any) map[string]any {
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func hasValidationIssue(issues *ValidationError, field string) bool {
	for _, issue := range issues.Issues {
		if strings.Contains(issue.Path, field) {
			return true
		}
	}
	return false
}
