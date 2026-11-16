package effortdecision

import (
	"path"
	"strings"

	"agent-manager/internal/decisionpage"
	"agent-manager/internal/goalhome"

	"github.com/vrooli/api-core/operatordecision"
)

// Page is the decision's page when the agent that raised it sent none: the
// explanation laid out for a phone, who is asking, each choice as a card with
// the recommendation, and what happens without an answer. An agent with
// something to show (a comparison, a picture, a plan) sends its own page with
// `effort decision raise --explainer page.html`, and that page is used instead.
func Page(d goalhome.Decision, f operatordecision.Facts, goalHome string) string {
	page := operatordecision.NewPage(operatordecision.Clip(d.Question, 200))
	if big := strings.TrimSpace(d.BigPicture); big != "" {
		page.Lead(big)
	}
	if who := HomeName(goalHome); who != "" {
		page.Facts(operatordecision.Row{Label: "Asked by", Value: "The agent working on " + who})
	}
	page.Options(f, nil)
	if d.DefaultAnswer != "" && !d.Deadline.IsZero() {
		return page.IfNoAnswer("\"" + d.Label(d.DefaultAnswer) + "\" applies by itself at " + d.Deadline.UTC().Format("Mon Jan 2 15:04 UTC") + ". You can still answer later and replace it.").HTML()
	}
	return page.IfNoAnswer("Nothing is decided. The agent that asked keeps waiting for your answer.").HTML()
}

// HomeName names the work a goal home holds in words, for the decision's
// subject and page: "scenarios/browser-automation-studio/docs/internal/goal" is
// "Browser automation studio", a research cohort folder is its name without the
// date. The stored path stays in the correlation, never on the cover.
func HomeName(home string) string {
	home = strings.Trim(strings.TrimSpace(home), "/")
	if home == "" {
		return ""
	}
	parts := strings.Split(home, "/")
	if len(parts) > 1 && parts[0] == "scenarios" {
		return decisionpage.Name(parts[1])
	}
	return decisionpage.Name(path.Base(home))
}
