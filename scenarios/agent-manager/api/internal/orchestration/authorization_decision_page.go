// This file freezes the text of an authorization decision when the request is
// raised: who is asking, what a grant would allow and never allow, and for how
// long, so the published ask and every re-send carry the same plain words.

package orchestration

import (
	"fmt"
	"html"
	"strings"
	"time"

	"agent-manager/internal/decisionpage"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/operatordecision"
)

// Authorization decision text. Everything the operator reads is frozen on the
// pending request when it is raised (freezeAuthorizationDecision), so a retry
// or a restart publishes the exact same event even after this wording changes.

// authorizationCallerName names the verified caller for a stranger: a known
// service by name, an agent session as such.
func authorizationCallerName(subject string) string {
	subject = strings.TrimSpace(subject)
	switch {
	case subject == TreasuryServiceSubject:
		return "Treasury"
	case subject == "":
		return "A caller"
	}
	if _, err := uuid.Parse(subject); err == nil {
		return "An agent session"
	}
	return operatordecision.Clip(decisionpage.Name(subject), 80)
}

// grantLifetime is "30 days" or "1 hour".
func grantLifetime(d time.Duration) string {
	if days := int(d.Hours()) / 24; days >= 1 {
		return operatordecision.Count(days, "day", "days")
	}
	return operatordecision.Count(max(int(d.Hours()), 1), "hour", "hours")
}

func concurrencyWords(p *domain.PendingAuthorizationRequest) string {
	if p.MaxConcurrentRuns > 0 {
		return fmt.Sprintf("at most %d at a time", p.MaxConcurrentRuns)
	}
	return "with no limit on how many at once"
}

// freezeAuthorizationDecision writes the cover, the technical details and the
// page onto the pending request.
func freezeAuthorizationDecision(p *domain.PendingAuthorizationRequest) {
	who, life := authorizationCallerName(p.Subject), grantLifetime(p.GrantTTL)
	subjectWho := who
	if strings.HasPrefix(who, "A ") || strings.HasPrefix(who, "An ") {
		subjectWho = strings.ToLower(who[:1]) + who[1:]
	}
	lead := who + " asked Agent Manager to start an agent run and was stopped, because it has no permission to."
	allows := []string{"Start agent runs without asking you again", "For " + life + ", " + concurrencyWords(p)}
	never := []string{"Start anything after the permission ends", "Act as you, or change any other permission"}
	effects := map[string]string{
		"approve": "Records your permission now. " + who + " goes ahead on its next try.",
		"reject":  "Nothing is granted. " + who + " stays refused and may ask again tomorrow.",
	}
	var big strings.Builder
	switch {
	case isResearchBudgetIngestRequest(p):
		p.DecisionSubject = "Let Treasury record one run's research spending?"
		p.DecisionQuestion = fmt.Sprintf("Treasury wants to record how much one agent run spent against the research budget. Let it file this one report within the next %s?", life)
		big.WriteString("Treasury keeps the research budget. To count what an agent run spent, it needs your one-time permission for that exact run. ")
		big.WriteString("Approving lets it record budget evidence for that run only; it cannot spend money, read other runs or do anything else. ")
		big.WriteString("Approve only if you recognize this research run and want its spending counted. ")
		big.WriteString("Refusing records nothing, and that run's spending stays uncounted. If you do not answer, nothing is recorded.")
		lead = "Treasury needs your permission to count what one agent run spent against the research budget."
		allows = []string{"Record budget evidence for this one agent run", "Within " + life + ", after which the permission ends"}
		never = []string{"Spend money", "Read or change any other run", "Act outside this one report"}
		effects["approve"] = "Treasury records this run's spending once. The permission then ends."
		effects["reject"] = "Nothing is recorded; this run's spending stays out of the research budget."
	case isBrowserEvidenceRecordRequest(p):
		p.DecisionSubject = "Let Browser Automation Studio report one run's browser session evidence?"
		p.DecisionQuestion = fmt.Sprintf("Browser Automation Studio wants to report browser session evidence for one agent run. Allow this exact run for the next %s?", life)
		big.WriteString("Browser Automation Studio asked Agent Manager to append sanitized browser session and action evidence to one active agent run. ")
		big.WriteString("Approving allows only the named run, only the browser-evidence.record operation, and only until the one-time permission expires or you revoke it. ")
		big.WriteString("It does not open a browser, prove that a session was live, authorize another run, or grant billing authority. BAS must separately prove that each session is active in its broker before any evidence can be trusted. ")
		big.WriteString("Refusing records no browser evidence. If you do not answer, nothing is granted.")
		lead = "Browser Automation Studio needs one-time permission to report evidence for this exact run."
		allows = []string{"Append sanitized browser session/action evidence to this one run", "Within " + life + ", after which the permission ends"}
		never = []string{"Open or control a browser", "Report evidence for another run", "Assert that a browser session is live without BAS broker proof", "Authorize usage or billing"}
		effects["approve"] = "BAS may report evidence for this run after its broker validates the active session."
		effects["reject"] = "No browser evidence permission is granted."
	case p.Kind == domain.PendingAuthorizationKindOneTime:
		p.DecisionSubject = fmt.Sprintf("Let %s start agent runs for the next %s?", subjectWho, life)
		p.DecisionQuestion = fmt.Sprintf("%s tried to start an agent run without permission. Let it start runs for the next %s, %s?", who, life, concurrencyWords(p))
		fmt.Fprintf(&big, "%s asked Agent Manager to start an agent run, and it has no permission to. Each agent run uses AI quota. ", who)
		fmt.Fprintf(&big, "Approving lets it start runs for %s, %s; the permission is recorded as yours, ends by itself and can be revoked any time. ", life, concurrencyWords(p))
		big.WriteString("Approving does not start anything itself. Refusing gives no permission and leaves any it already has unchanged. If you do not answer, nothing is granted.")
		if p.SessionID != "" {
			allows[0] = "Start agent runs from the one session that asked"
		}
	default:
		p.DecisionSubject = fmt.Sprintf("Let %s start agent runs on its own for %s?", subjectWho, life)
		p.DecisionQuestion = fmt.Sprintf("%s wants to start agent runs on its own, with no person present. Let it, for the next %s, %s?", who, life, concurrencyWords(p))
		fmt.Fprintf(&big, "%s asked Agent Manager to start an agent run with nobody present, and it has no standing permission to. Each agent run uses AI quota. ", who)
		fmt.Fprintf(&big, "Approving lets it start runs by itself for %s, %s; the permission is recorded as yours, ends by itself and can be revoked any time. ", life, concurrencyWords(p))
		big.WriteString("Approving does not start anything itself. Refusing gives no permission and leaves any it already has unchanged. If you do not answer, nothing is granted.")
	}
	p.DecisionBigPicture = big.String()

	concurrency := "no concurrency limit"
	if p.MaxConcurrentRuns > 0 {
		concurrency = fmt.Sprintf("at most %d concurrent runs", p.MaxConcurrentRuns)
	}
	p.DecisionTechnicalDetails = fmt.Sprintf(
		"Subject: %s\nOperation: %s\nTarget: %s\nGrant kind: %s\nGrant lifetime: %s\nConcurrency: %s\nApproval is explicit, owner-attributed, and revocable; no answer issues no grant.",
		p.Subject, p.Operation, p.Target, p.Kind, humanizeGrantTTL(p.GrantTTL), concurrency)
	if p.SessionID != "" {
		p.DecisionTechnicalDetails += "\nGrant scope: session-scoped."
	}

	f := authorizationFacts(p)
	p.DecisionExplainer = operatordecision.NewPage(p.DecisionSubject).
		Lead(lead).
		Facts(
			operatordecision.Row{Label: "Who is asking", Value: who},
			operatordecision.Row{Label: "Wants to", Value: allows[0]},
			operatordecision.Row{Label: "For how long", Value: life},
			operatordecision.Row{Label: "Approved by", Value: "You, by name, in Agent Manager's records"},
			operatordecision.Row{Label: "Can be undone", Value: "Yes. Revoke it any time from Agent Manager."},
		).
		Raw(`<div class="hub-pair"><section><h3>Approving allows</h3>`+htmlList(allows)+`</section><section><h3>It never allows</h3>`+htmlList(never)+`</section></div>`).
		Options(f, effects).
		IfNoAnswer("Nothing is granted. The request closes after a day.").
		HTML()
}

// authorizationFacts is the operator decision built from the frozen request.
func authorizationFacts(p *domain.PendingAuthorizationRequest) operatordecision.Facts {
	allow, reason := "Allow it for "+grantLifetime(p.GrantTTL), "Allow only a caller you expect to start agent work by itself; the permission lets it use AI quota without asking again."
	if isResearchBudgetIngestRequest(p) {
		allow, reason = "Allow this one report", "Allow it only if you recognize this research run and want its spending counted."
	} else if isBrowserEvidenceRecordRequest(p) {
		allow, reason = "Allow this one run's browser evidence", "Allow only if you recognize the active browser-capable run; BAS must still prove each session from its broker."
	}
	return operatordecision.Facts{
		SubjectKey: p.SubjectKey, Subject: p.DecisionSubject, Question: p.DecisionQuestion,
		BigPicture: p.DecisionBigPicture, TechnicalDetails: p.DecisionTechnicalDetails, Explainer: p.DecisionExplainer,
		Options:     []operatordecision.Option{{Key: "approve", Label: allow}, {Key: "reject", Label: "Refuse"}},
		Recommended: "reject", RecommendationReason: reason,
		// Issuing a grant is reversible: it is bounded, revocable, and expires. No
		// silent default is set, so nothing is granted without an explicit answer.
		Reversible: true,
		Urgency:    "normal",
		Correlation: map[string]string{
			"authorization_request_id": p.ID,
			"subject":                  p.Subject,
			"operation":                p.Operation,
			"target":                   p.Target,
			"kind":                     p.Kind,
		},
	}
}

func htmlList(items []string) string {
	var b strings.Builder
	b.WriteString("<ul>")
	for _, item := range items {
		b.WriteString("<li>" + html.EscapeString(item) + "</li>")
	}
	b.WriteString("</ul>")
	return b.String()
}
