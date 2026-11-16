package operatordecision

import (
	"strings"
	"testing"
	"time"
)

func TestPageIsAPhoneFirstSelfContainedExplainer(t *testing.T) {
	f := Facts{
		Question:             "Delete old build caches?",
		Options:              []Option{{Key: "delete", Label: "Delete them"}, {Key: "keep", Label: "Keep them"}},
		Recommended:          "delete",
		RecommendationReason: "They rebuild on their own.",
	}
	got := NewPage("Delete 4.2 GB of <old> caches?").
		Lead("The disk is 93% full.\n\nThese caches rebuild.").
		Facts(Row{"Disk", "93% full"}).
		Table([]string{"Folder", "Size"}, [][]string{{"go-build", "3.1 GB"}}).
		Options(f, map[string]string{"delete": "Frees 4.2 GB now."}).
		IfNoAnswer("Nothing is deleted.").
		HTML()
	for _, want := range []string{
		`<meta name="viewport" content="width=device-width, initial-scale=1">`,
		"<h1>Delete 4.2 GB of &lt;old&gt; caches?</h1>",
		"<p>The disk is 93% full.</p><p>These caches rebuild.</p>",
		`<td data-label="Size">3.1 GB</td>`,
		`<section class="is-recommended"><span class="hub-chip is-primary">Recommended</span><h3>Delete them</h3>`,
		`data-od-answer="keep"`,
		"They rebuild on their own.",
		"If you do not answer:",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	for _, banned := range []string{"<form", "<input", "http://", "https://", "src=\"//"} {
		if strings.Contains(got, banned) {
			t.Errorf("page contains %q, which the Hub frame blocks", banned)
		}
	}
}

func TestFactsCarryTheExplainerAndBoundItsSize(t *testing.T) {
	f := Facts{Question: "Q?", Options: []Option{{Key: "a", Label: "A"}, {Key: "b", Label: "B"}}, Explainer: "<!doctype html><p>page</p>"}
	if err := f.Validate(); err != nil {
		t.Fatalf("valid facts refused: %v", err)
	}
	if got := f.Payload()["explainer_html"]; got != f.Explainer {
		t.Fatalf("payload explainer_html = %v", got)
	}
	f.Explainer = strings.Repeat("x", MaxExplainerBytes+1)
	if err := f.Validate(); err == nil {
		t.Fatal("an oversize explainer was accepted; the event would pass the 1 MiB intake limit")
	}
	f.Explainer = ""
	if _, ok := f.Payload()["explainer_html"]; ok {
		t.Fatal("a page-less decision must emit the bytes it always did")
	}
}

func TestFactsCarryBoundedTemplateDataWithoutHTML(t *testing.T) {
	f := Facts{
		Question:     "Use this model while quota is at risk?",
		Options:      []Option{{Key: "start", Label: "Start"}, {Key: "not_now", Label: "Not now"}},
		TemplateName: "usage-mode",
		TemplateData: map[string]any{"used_percent": 72.5, "points": []any{map[string]any{"at": "2026-11-16T12:00:00Z", "used_percent": 72.5}}},
	}
	if err := f.Validate(); err != nil {
		t.Fatalf("valid template facts refused: %v", err)
	}
	payload := f.Payload()
	if payload["template_name"] != "usage-mode" {
		t.Fatalf("template_name = %v", payload["template_name"])
	}
	if got := payload["template_data"].(map[string]any)["used_percent"]; got != 72.5 {
		t.Fatalf("template_data used_percent = %v", got)
	}
	if _, ok := payload["explainer_html"]; ok {
		t.Fatal("data-driven page must not publish producer HTML")
	}

	f.TemplateData = nil
	if err := f.Validate(); err == nil {
		t.Fatal("template data without a template name was accepted")
	}
	f.TemplateName = ""
	f.TemplateData = map[string]any{"oversize": strings.Repeat("x", MaxTemplateDataBytes)}
	if err := f.Validate(); err == nil {
		t.Fatal("oversize template data was accepted")
	}
}

func TestCoverPageReShowsTheWholeCoverAndSaysWhatHappensNext(t *testing.T) {
	f := Facts{
		Question:             "Delete <old> build caches?",
		BigPicture:           "The disk is 93% full.",
		Options:              []Option{{Key: "delete", Label: "Delete them"}, {Key: "keep", Label: "Keep them"}},
		Recommended:          "delete",
		RecommendationReason: "They rebuild on their own.",
		Packet:               "/home/me/.vrooli/research/work/CACHE-1/BRIEF.md",
	}
	deadline := time.Date(2026, 11, 22, 9, 30, 0, 0, time.UTC)
	got := CoverPage(f, "BRIEF.md", deadline)
	for _, want := range []string{
		`<meta name="generator" content="operatordecision.CoverPage">`,
		"<h1>Delete &lt;old&gt; build caches?</h1>",
		"<p>The disk is 93% full.</p>",
		`<span class="hub-chip is-primary">Recommended</span><h3>Delete them</h3>`,
		"They rebuild on their own.",
		"Tap “Choose this”, then send.",
		"What happens when you approve",
		`data-hub-open="BRIEF.md"`,
		"stays open until Sun Nov 22, 09:30 UTC, then closes unanswered",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("cover page lacks %q", want)
		}
	}
	if strings.Contains(got, f.Packet) {
		t.Error("the page shows the packet path; paths are for the executing agent, not the operator")
	}

	f.Packet, f.DefaultAnswer, f.Reversible = "", "keep", true
	got = CoverPage(f, "", deadline)
	if strings.Contains(got, "What happens when you approve") {
		t.Error("a decision with no packet must not promise prepared work")
	}
	if !strings.Contains(got, "At Sun Nov 22, 09:30 UTC “Keep them” is chosen for you.") {
		t.Error("the page does not state the default that applies with no answer")
	}
}

func TestPacketTravelsInThePayloadAndMustBeAnAbsolutePath(t *testing.T) {
	f := Facts{Question: "Q?", Options: []Option{{Key: "a", Label: "A"}, {Key: "b", Label: "B"}}, Packet: "/srv/work/X/BRIEF.md"}
	if err := f.Validate(); err != nil {
		t.Fatalf("valid packet refused: %v", err)
	}
	if got := f.Payload()["packet"]; got != f.Packet {
		t.Fatalf("payload packet = %v", got)
	}
	for _, bad := range []string{"work/X/BRIEF.md", "/srv/a\nb", "/" + strings.Repeat("x", MaxPacketPath)} {
		f.Packet = bad
		if err := f.Validate(); err == nil {
			t.Errorf("packet %q was accepted", bad)
		}
	}
	f.Packet = ""
	if _, ok := f.Payload()["packet"]; ok {
		t.Fatal("a decision with no packet must emit the bytes it always did")
	}
}
