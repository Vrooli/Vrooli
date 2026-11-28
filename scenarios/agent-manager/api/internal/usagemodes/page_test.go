package usagemodes

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

// coldFacts reads a published usage decision back as Facts, the way the Hub
// shows it, so a test can run the cold-reader check on its cover.
func coldFacts(e eventbus.DomainEvent) operatordecision.Facts {
	p := e.Payload
	f := operatordecision.Facts{}
	f.Subject, _ = p["subject"].(string)
	f.Question, _ = p["question"].(string)
	f.BigPicture, _ = p["big_picture"].(string)
	f.RecommendationReason, _ = p["recommendation_reason"].(string)
	f.Explainer, _ = p["explainer_html"].(string)
	for _, o := range p["options"].([]any) {
		m := o.(map[string]any)
		f.Options = append(f.Options, operatordecision.Option{Key: m["key"].(string), Label: m["label"].(string)})
	}
	return f
}

func assertPage(t *testing.T, name string, e eventbus.DomainEvent, wants ...string) {
	t.Helper()
	f := coldFacts(e)
	if leaks := operatordecision.CoverProblems(f); len(leaks) > 0 {
		t.Errorf("%s: cover is not readable cold: %v", name, leaks)
	}
	for _, want := range append(wants, "If you do not answer:", "Your choices") {
		if !strings.Contains(f.Explainer, want) {
			t.Errorf("%s: page lacks %q", name, want)
		}
	}
}

// Every data-driven run-out decision sends source facts to the Hub template;
// other usage decisions retain their existing decision pages.
func TestUsageRunOutUsesTemplateAndOtherDecisionsKeepPages(t *testing.T) {
	ctx := context.Background()

	n, rec, _ := newNotifier(t)
	f := exhaustForecast(t0.Add(48 * time.Hour))
	f.Used = 81
	if err := n.WillExhaust(ctx, f, *f.Projected, t0); err != nil {
		t.Fatal(err)
	}
	p := rec.decisions()[0].Payload
	if p["template_name"] != "usage-mode" || p["explainer_html"] != nil {
		t.Fatalf("run-out decision must publish template data without producer HTML: %+v", p)
	}
	data := p["template_data"].(map[string]any)
	window := data["window"].(map[string]any)
	if window["reset"] != f.Window.ResetAt.UTC().Format(time.RFC3339) {
		t.Errorf("template window reset = %v", window["reset"])
	}
	if window["source"].(map[string]any)["status"] != "real" {
		t.Errorf("template window source = %+v", window["source"])
	}
	readings := window["readings"].([]any)
	if len(readings) == 0 || readings[len(readings)-1].(map[string]any)["used"] != float64(81) {
		t.Errorf("template readings do not include the real last observation: %+v", readings)
	}
	var status string
	for _, raw := range data["technical"].([]any) {
		item := raw.(map[string]any)
		if item["label"] == "Forecast status" {
			status = item["value"].(string)
		}
	}
	if status != f.Status+": "+f.Reason {
		t.Errorf("technical forecast status = %q", status)
	}
	if q := coldFacts(rec.decisions()[0]).Question; !strings.Contains(q, "Your Codex weekly allowance") || !strings.Contains(q, "46 hours before it resets") {
		t.Errorf("run-out question = %q", q)
	}

	n, rec, _ = newNotifier(t)
	if _, err := n.AskToStart(ctx, "codex", time.Hour); err != nil {
		t.Fatal(err)
	}
	assertPage(t, "timed start", rec.decisions()[0], "What changes for new work", "gpt-6-luna", "1 hour")

	n, rec, c := newNotifier(t)
	_, ask := startUntilOff(t, n, rec, c)
	assertPage(t, "until off", ask, "Applies to", "New Codex work", "When you turn it off", "o@example.com", "gpt-6-luna")

	n, rec, c = newNotifier(t)
	if _, err := n.Modes.Start(ctx, StartRequest{Preset: PresetOneTierDown, Runner: "codex", Until: "1h", StartedBy: "o@example.com"}); err != nil {
		t.Fatal(err)
	}
	c.t = t0.Add(50 * time.Minute)
	n.Flush(ctx, nil, c.t)
	assertPage(t, "ending", rec.decisions()[0], "Ends", "New work goes back to the usual models at", "gpt-6-luna")
	if big := coldFacts(rec.decisions()[0]).BigPicture; !strings.Contains(big, "If you do not answer, it ends on time") {
		t.Errorf("ending big picture = %q", big)
	}

	n, rec, _ = newNotifier(t)
	change := PolicyChange{File: "resources/codex/model-policy.json", Ops: []PolicyOp{{Op: "replace", Path: "/roles/code.economy.delivery/model", From: "gpt-6.1-terra", Value: "gpt-6-luna"}}}
	if _, err := n.ProposeRoleDefault(ctx, proposal(change)); err != nil {
		t.Fatal(err)
	}
	assertPage(t, "role default", rec.decisions()[0], "Code economy delivery", "30% cheaper per accepted outcome", "/roles/code.economy.delivery/model", "Nothing switches until you apply it")
	if tech, _ := rec.decisions()[0].Payload["technical_details"].(string); !strings.Contains(tech, `"path":"/roles/code.economy.delivery/model"`) {
		t.Errorf("the exact change belongs in the technical details: %s", tech)
	}
}
