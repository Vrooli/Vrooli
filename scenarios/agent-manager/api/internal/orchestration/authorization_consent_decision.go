// This file turns a pending interactive consent request into the operator's
// decision: one Notification Hub ask with a page that says exactly what
// approving allows (operations, targets and how far each reaches, for how long,
// who asked and why) and recommends the narrowest reach.
//
// The cover and page lead with the requester's own stated reason (quoted, and
// labelled as unverified), the session that asked, and what the permission lets
// it do in everyday words; the recommendation weighs that purpose against how
// far the permission reaches.
//
// The Hub only presents it. Approval needs the operator's own verified sign-in
// on Agent Manager's consent page (ApproveAuthorization), so the ask's approve
// option is that page's link and the Hub refuses an "approve" answer for it. A
// Hub answer never grants interactive authority; when the operator approves on
// the page, the ask is withdrawn.

package orchestration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/decisionpage"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/operatordecision"
)

// ConsentDecisionEventType is the decision a pending interactive consent
// request raises. Notification Hub recognises it (with the consent page as its
// context_url) and takes "approve" only on that page.
const ConsentDecisionEventType = "agent_manager.authorization_consent.decision_requested.v1"

// consentSubjectKey names the one open decision for a consent request.
func consentSubjectKey(id uuid.UUID) string { return "consent:" + id.String() }

// targetReach says in words how far one target reaches under the canonical
// evaluator (identity.OperationGrant.Allows): an exact target, every target
// under a "domain:prefix/*", or the whole domain for "domain:*".
type targetReach struct {
	Kind, Words string
	Wildcard    bool
}

func reachOf(target string) targetReach {
	domainName, rest, _ := strings.Cut(target, ":")
	where := decisionpage.Name(domainName)
	switch {
	case rest == "*":
		return targetReach{Kind: "item", Words: "Everything in " + where + ", including things made later", Wildcard: true}
	case strings.HasSuffix(rest, "/*"):
		kind := strings.TrimSuffix(rest, "/*")
		return targetReach{Kind: kind, Words: "Every " + kind + " in " + where + ", including ones made later", Wildcard: true}
	}
	kind, name, found := strings.Cut(rest, "/")
	if !found {
		return targetReach{Kind: "item", Words: "Only " + decisionpage.Name(rest) + " in " + where}
	}
	return targetReach{Kind: kind, Words: "Only the " + kind + " " + decisionpage.Name(name)}
}

// consentScope summarises a request's reach for the question: "1 commission",
// "6 commissions", "every commission".
func consentScope(g *domain.AuthorizationGrant) (summary string, wildcard bool, targets int) {
	kinds := map[string]bool{}
	kind := ""
	for _, rule := range g.PolicyRules() {
		for _, target := range rule.Targets {
			reach := reachOf(target)
			targets++
			kind = reach.Kind
			kinds[reach.Kind] = true
			if reach.Wildcard {
				wildcard = true
			}
		}
	}
	switch {
	case wildcard && len(kinds) == 1:
		return "every " + kind, true, targets
	case wildcard:
		return "whole areas", true, targets
	case len(kinds) == 1:
		return operatordecision.Count(targets, kind, kind+"s"), false, targets
	}
	return operatordecision.Count(targets, "target", "targets"), false, targets
}

// harnessNames spells the coding agents the way their makers do.
var harnessNames = map[string]string{"claude-code": "Claude Code", "codex": "Codex", "opencode": "OpenCode"}

// consentAgentName names the requesting session for a stranger.
func consentAgentName(harnessKind string) string {
	if name := harnessNames[strings.ToLower(strings.TrimSpace(harnessKind))]; name != "" {
		return "a " + name + " agent"
	}
	if name := decisionpage.Name(harnessKind); name != "" {
		return "a " + name + " agent"
	}
	return "an agent session"
}

// policyWords is what a reviewed policy lets an agent do, in everyday words,
// for the operator who has never read the policy catalogue. Action completes
// "Allow an agent to …". WideOnly marks a permission that exists only as a
// whole-area wildcard, so nothing narrower can be requested and the lifetime
// is its real limit. The rows follow docs/concepts/AUTHORIZATION.md.
type policyWords struct {
	Action   string
	Does     []string
	Never    []string
	WideOnly bool
}

var policyCatalogue = map[string]policyWords{
	"run-creation": {Action: "start new agent runs", WideOnly: true,
		Does:  []string{"Start a new Claude or Codex run for any task or profile", "Each run it starts uses model quota and computer time until it finishes"},
		Never: []string{"Stop, change or delete runs that already exist", "Approve or reject reviews"}},
	"agent-investigation": {Action: "investigate agent runs", WideOnly: true,
		Does:  []string{"Start an investigation into why a run behaved as it did, and read the result"},
		Never: []string{"Stop, change or delete any run"}},
	"agent-management": {Action: "stop, continue or wake agent runs",
		Does:  []string{"Stop a named run, continue or wake a parked one, cancel or update a named task", "Investigate runs"},
		Never: []string{"Delete runs for good", "Stop every run, unless the permission names all of them"}},
	"run-deletion": {Action: "permanently delete agent runs",
		Does:  []string{"Delete the named run for good. This cannot be undone"},
		Never: []string{"Decide a run's review"}},
	"effort-metadata": {Action: "edit an effort's details",
		Does:  []string{"Change an effort's goal home or display name"},
		Never: []string{"Change who may act on the effort, or withdraw it"}},
	"delivery-management": {Action: "start, pause, resume or stop an effort",
		Does:  []string{"Start, pause, resume or stop the work of an effort that already exists"},
		Never: []string{"Create new efforts or change what they are for"}},
	"research-management": {Action: "launch or control research commissions",
		Does:  []string{"Launch or control the named research commissions"},
		Never: []string{"Answer decisions, give verdicts or issue permissions"}},
	"research-wind-down": {Action: "wind down named research commissions",
		Does:  []string{"Add the fixed wind-down instruction to each named commission so its manager closes its current unit and writes a closeout"},
		Never: []string{"Launch, pause, resume or stop commissions, or give freeform directions"}},
	"research-closeout": {Action: "end named research commissions after closeout",
		Does:  []string{"Stop only the named commissions whose manager goal is paused after closeout"},
		Never: []string{"Launch, pause, resume, discard or otherwise control commissions"}},
	"swarm-bookkeeping": {Action: "update Swarm Manager's records",
		Does: []string{"Record, change or remove items on Swarm Manager's backlog"}},
	"team-preparation": {Action: "prepare draft teams",
		Does:  []string{"Edit draft team revisions that are switched off and have not started"},
		Never: []string{"Start or activate a team, or grant it anything"}},
	"team-management": {Action: "change teams and their members",
		Does: []string{"Change the named team and who is in it"}},
	"prompt-agent-creation": {Action: "create one catalogue agent",
		Does:  []string{"Create the one named catalogue entry"},
		Never: []string{"Change or delete it afterwards, or touch its members or files"}},
	"deployment-operations": {Action: "run deployments",
		Does: []string{"Deploy, change or tear down deployments"}},
	"nutrition-management": {Action: "manage nutrition plans",
		Does: []string{"Change nutrition plans and their data, including removing them"}},
	"treasury-spending": {Action: "spend from Treasury",
		Does:  []string{"Make the one authorized payment, if Treasury's own limits also allow it"},
		Never: []string{"Change Treasury's rules or kill switch, or pay more than its signed limits"}},
	"treasury-wallet-reading": {Action: "read the Treasury wallet",
		Does: []string{"See the derived wallet"}},
}

// consentActionWords says what the request does, for the question and the
// subject: the catalogue's action when known, else the policy's name.
func consentActionWords(g *domain.AuthorizationGrant) (action string, known bool) {
	rules := g.PolicyRules()
	var actions []string
	known = true
	for _, rule := range rules {
		words, ok := policyCatalogue[rule.Policy]
		if !ok {
			known = false
			actions = append(actions, "use "+strings.ToLower(decisionpage.Name(rule.Policy)))
			continue
		}
		actions = append(actions, words.Action)
	}
	return strings.Join(actions, " and "), known
}

// consentWideOnly is true when every wildcard the request holds is one its
// policy can only be asked for as a wildcard.
func consentWideOnly(g *domain.AuthorizationGrant) bool {
	wide := false
	for _, rule := range g.PolicyRules() {
		for _, target := range rule.Targets {
			if !reachOf(target).Wildcard {
				continue
			}
			wide = true
			if !policyCatalogue[rule.Policy].WideOnly {
				return false
			}
		}
	}
	return wide
}

// consentPurpose is the requester's own stated reason, verbatim: its big
// picture without the packet line. It is the requester's claim, never proof.
func consentPurpose(g *domain.AuthorizationGrant) string {
	if g.RequesterContext == nil {
		return ""
	}
	_, words := operatordecision.PacketLine(g.RequesterContext.BigPicture)
	return strings.TrimSpace(words)
}

// consentSession names the asking session: its run label when Agent Manager
// has one, else the harness and a short session id.
func consentSession(g *domain.AuthorizationGrant, label string) string {
	who := consentAgentName(g.HarnessKind)
	switch {
	case strings.TrimSpace(label) != "":
		return fmt.Sprintf("%s working on “%s”", who, operatordecision.Clip(strings.TrimSpace(label), 80))
	case g.HarnessSession != "":
		id := g.HarnessSession
		if len(id) > 8 {
			id = id[len(id)-8:]
		}
		return fmt.Sprintf("%s (session ending %s)", who, id)
	}
	return who
}

// consentRecommendation weighs the stated purpose and the lifetime against how
// far the permission reaches. A permission that exists only as a wildcard is
// judged on its reason and its lifetime, since no narrower request exists; any
// other wildcard is declined with the narrower request spelled out.
func consentRecommendation(g *domain.AuthorizationGrant, wildcard bool, purpose, life string) (string, string) {
	said := ""
	if purpose != "" {
		said = fmt.Sprintf(" It says it is for “%s” (its own words, not checked).", operatordecision.Clip(purpose, 120))
	}
	switch {
	case !wildcard:
		return "approve", fmt.Sprintf("It reaches only what it names and ends by itself after %s.%s", life, said)
	case consentWideOnly(g):
		long := g.DurationSeconds > int64(24*time.Hour/time.Second)
		switch {
		case purpose == "":
			return "decline", "It gives no reason, and this permission only comes in a wide form. Decline, and let it ask again saying what it is for."
		case long:
			return "decline", fmt.Sprintf("This permission cannot be narrowed, so its limit is time, and %s is more than a task like this needs.%s Let it ask again for 24 hours.", life, said)
		}
		return "approve", fmt.Sprintf("This permission cannot be narrowed to chosen items, so its limit is time: it ends by itself after %s.%s", life, said)
	}
	return "decline", fmt.Sprintf("It reaches everything of that kind, including things made later, which is wider than a task usually needs.%s Decline, and let it ask again naming only the ones it needs.", said)
}

func consentOperations(g *domain.AuthorizationGrant) []string {
	var operations []string
	seen := map[string]bool{}
	for _, rule := range g.PolicyRules() {
		policy, _ := authn.InteractivePolicy(rule.Policy)
		for _, operation := range policy.Operations {
			if !seen[operation] {
				seen[operation] = true
				operations = append(operations, strings.ToLower(decisionpage.Name(operation)))
			}
		}
	}
	return operations
}

func consentPolicyWords(g *domain.AuthorizationGrant) string {
	var names []string
	for _, rule := range g.PolicyRules() {
		names = append(names, strings.ToLower(decisionpage.Name(rule.Policy)))
	}
	return strings.Join(names, " and ")
}

// consentFacts is the operator decision for a pending consent request whose
// approval page is consentURL. sessionLabel is the asking run's label when
// Agent Manager has one, else "".
func consentFacts(g *domain.AuthorizationGrant, consentURL, sessionLabel string) operatordecision.Facts {
	who := consentAgentName(g.HarnessKind)
	session := consentSession(g, sessionLabel)
	life := decisionpage.Duration(time.Duration(g.DurationSeconds) * time.Second)
	deadline := g.RequestExpiresAt.UTC().Format("Mon Jan 2, 15:04 UTC")
	scope, wildcard, _ := consentScope(g)
	action, known := consentActionWords(g)
	purpose := consentPurpose(g)

	// A known permission says what it does; the area is added only when the
	// request names specific items, since a wide-only permission has no choice.
	where := ""
	if !known || !consentWideOnly(g) {
		where = " on " + scope
	}
	questionWho := who
	if strings.TrimSpace(sessionLabel) != "" {
		questionWho = session
	}
	question := fmt.Sprintf("Allow %s to %s%s for %s?", questionWho, action, where, life)
	// The push shows the question and the subject only. The requester's claim
	// stays out of them (it is unverified text from the agent); the cover's big
	// picture and the page quote it, attributed.
	subject := "Permission request: " + action
	big := fmt.Sprintf("%s is waiting for your permission to %s%s. ", upperFirst(session), action, where)
	if purpose != "" {
		big += fmt.Sprintf("It says why, in its own words (not checked): “%s” ", purpose)
	} else {
		big += "It gave no reason. "
	}
	if does := consentDoes(g); len(does) > 0 {
		big += fmt.Sprintf("Approving lets it: %s. Nothing else, for %s from when you approve, and you can revoke it any time. ", strings.Join(does, "; "), life)
	}
	big += "Only you can approve it, with your sign-in on Agent Manager's page, which the approve button opens. " +
		fmt.Sprintf("If you do not answer by %s, the request lapses and nothing is allowed.", deadline)

	recommended, reason := consentRecommendation(g, wildcard, purpose, life)
	f := operatordecision.Facts{
		Subject:  operatordecision.Clip(subject, 120),
		Question: operatordecision.Clip(question, operatordecision.MaxQuestion),
		Options: []operatordecision.Option{
			{Key: "approve", Label: "Allow it — confirm with your sign-in on the next screen"},
			{Key: "decline", Label: "Don't allow it — it stays blocked and may ask again"},
		},
		Recommended: recommended, RecommendationReason: reason,
		BigPicture:       operatordecision.Clip(big, operatordecision.MaxBigPictureCodePoints),
		TechnicalDetails: consentTechnicalDetails(g, consentURL),
		ContextURL:       consentURL,
		Deadline:         g.RequestExpiresAt.UTC(),
		Urgency:          "normal",
		SubjectKey:       consentSubjectKey(g.ID),
		AssertedAt:       g.CreatedAt.UTC(),
		Correlation:      map[string]string{"authorization_id": g.ID.String(), "run_id": g.RunID.String(), "policy": g.Policy},
	}
	f.Packet = consentPacket(g)
	f.Explainer = consentPage(g, f, session, life, deadline, purpose, wildcard)
	return f
}

// consentDoes lists, in words, what the requested permissions let the agent do.
// A policy the catalogue does not know contributes its operations instead.
func consentDoes(g *domain.AuthorizationGrant) []string {
	var out []string
	seen := map[string]bool{}
	for _, rule := range g.PolicyRules() {
		lines := policyCatalogue[rule.Policy].Does
		if len(lines) == 0 {
			if ops := consentOperations(&domain.AuthorizationGrant{Policy: rule.Policy, Targets: rule.Targets}); len(ops) > 0 {
				lines = []string{upperFirst(strings.Join(ops, " and "))}
			}
		}
		for _, line := range lines {
			if !seen[line] {
				seen[line] = true
				out = append(out, line)
			}
		}
	}
	return out
}

// consentNever lists what the requested permissions do not cover.
func consentNever(g *domain.AuthorizationGrant, life string) []string {
	var out []string
	seen := map[string]bool{}
	for _, rule := range g.PolicyRules() {
		for _, line := range policyCatalogue[rule.Policy].Never {
			if !seen[line] {
				seen[line] = true
				out = append(out, line)
			}
		}
	}
	return append(out, "Act as you, or change any other permission", "Last past "+life+"; you can revoke it sooner")
}

// consentPacket is the absolute path of the packet the requester named with a
// `Packet:` line in its own words or details, or "" when it named none.
func consentPacket(g *domain.AuthorizationGrant) string {
	if g.RequesterContext == nil {
		return ""
	}
	for _, text := range []string{g.RequesterContext.BigPicture, g.RequesterContext.TechnicalDetails} {
		if path, _ := operatordecision.PacketLine(text); path != "" {
			if rest, ok := strings.CutPrefix(path, "~/"); ok {
				if home, err := os.UserHomeDir(); err == nil {
					path = filepath.Join(home, rest)
				}
			}
			if filepath.IsAbs(path) && operatordecision.PacketPathProblem(path) == "" {
				return path
			}
		}
	}
	return ""
}

// consentBrief is the part of the requester's packet the operator needs:
// what each target builds, what runs on approval, what happens otherwise.
func consentBrief(g *domain.AuthorizationGrant) []operatordecision.BriefSection {
	path := consentPacket(g)
	if path == "" {
		return nil
	}
	sections, err := operatordecision.ReadBrief(path)
	if err != nil {
		return nil
	}
	var shown []operatordecision.BriefSection
	for _, s := range sections {
		h := strings.ToLower(s.Heading)
		if strings.HasPrefix(h, "what ") || strings.HasPrefix(h, "on approval") || strings.HasPrefix(h, "on each answer") {
			shown = append(shown, s)
		}
	}
	return shown
}

func consentTechnicalDetails(g *domain.AuthorizationGrant, consentURL string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Authorization request: %s\nApproval page: %s\n", g.ID, consentURL)
	for _, rule := range g.PolicyRules() {
		policy, _ := authn.InteractivePolicy(rule.Policy)
		fmt.Fprintf(&b, "Policy %s: operations %s; targets %s\n", rule.Policy, strings.Join(policy.Operations, ", "), strings.Join(rule.Targets, ", "))
	}
	fmt.Fprintf(&b, "Lifetime after approval: %s\nReview deadline: %s\n", humanizeDurationSeconds(g.DurationSeconds), g.RequestExpiresAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "Requesting session: %s %s; run %s; process %d\n", g.HarnessKind, g.HarnessSession, g.RunID, g.Binding.PID)
	if g.RequesterContext != nil && g.RequesterContext.TechnicalDetails != "" {
		b.WriteString("Requester's details: " + g.RequesterContext.TechnicalDetails)
	}
	return operatordecision.Clip(b.String(), operatordecision.MaxTechnicalDetailsCodePoints)
}

func consentPage(g *domain.AuthorizationGrant, f operatordecision.Facts, session, life, deadline, purpose string, wildcard bool) string {
	var rows [][]string
	for _, rule := range g.PolicyRules() {
		for _, target := range rule.Targets {
			reach := reachOf(target)
			words := reach.Words
			if reach.Wildcard && policyCatalogue[rule.Policy].WideOnly {
				words += ". This permission only comes in this wide form; what limits it is the " + life + " lifetime"
			}
			rows = append(rows, []string{decisionpage.Name(rule.Policy), words})
		}
	}
	allows := append(consentDoes(g), "For "+life+" from when you approve")
	if len(consentDoes(g)) == 0 {
		allows = []string{upperFirst(strings.Join(consentOperations(g), " and ")) + ", only in the places listed below", "For " + life + " from when you approve"}
	}
	page := operatordecision.NewPage(f.Question).Raw(decisionpage.Style).
		Lead(upperFirst(session) + " is blocked until you answer. Approving opens Agent Manager's page, where you confirm with your sign-in.")
	if purpose != "" {
		page.Section("What it says it is doing", "").Raw(decisionpage.Quote(purpose) + `<p class="dp-note">Its own words, sent with the request. Agent Manager has not checked them.</p>`)
	} else {
		page.Section("What it says it is doing", "It gave no reason with the request.")
	}
	page.Facts(
		operatordecision.Row{Label: "Who is asking", Value: upperFirst(session)},
		operatordecision.Row{Label: "For how long", Value: life + ", starting when you approve"},
		operatordecision.Row{Label: "Decide by", Value: deadline},
		operatordecision.Row{Label: "Can be undone", Value: "Yes. Revoke it any time on Agent Manager's page."},
	).
		Section("What approving lets it do", "").
		Raw(`<div class="hub-pair"><section><h3>Approving allows</h3>`+htmlList(allows)+`</section><section><h3>It never allows</h3>`+htmlList(consentNever(g, life))+`</section></div>`).
		Section("Where it reaches", "").
		Table([]string{"Permission", "Reaches"}, rows)
	// The requester's packet says what approving starts; show its own words,
	// so what the operator approves is what the packet will run.
	for _, s := range consentBrief(g) {
		page.Section(s.Heading, "").Markdown(s.Body)
	}
	declineEffect := "Nothing is allowed. The agent stays blocked and may ask again for less."
	switch {
	case wildcard && consentWideOnly(g):
		declineEffect = "Nothing is allowed. The agent stays blocked and may ask again, saying what it is for."
	case wildcard:
		declineEffect = "Nothing is allowed. The agent stays blocked and may ask again naming only the items it needs."
	}
	return page.Options(f, map[string]string{
		"approve": "Agent Manager's page opens. Confirm there with your sign-in, and the agent goes ahead.",
		"decline": declineEffect,
	}).
		IfNoAnswer("Nothing is allowed. The request lapses at " + deadline + ".").
		Raw(decisionpage.Details("The exact permission", f.TechnicalDetails)).
		HTML()
}

// withdrawConsentDecision closes the operator's decision for a consent request
// that was decided elsewhere (approved on the consent page, or replaced by a
// newer request). It changes no authority; a failure is logged and the Hub's
// deadline remains the backstop.
func (o *Orchestrator) withdrawConsentDecision(ctx context.Context, g *domain.AuthorizationGrant, reason string) {
	if o == nil || o.approvalPublisher == nil || g == nil || g.NotificationSnapshot == nil || g.NotificationSnapshot.EventType != ConsentDecisionEventType {
		return
	}
	event := operatordecision.BuildWithdrawalEventTyped(runApprovalSource, ConsentDecisionEventType, o.now().UTC(), operatordecision.Withdrawal{SubjectKey: consentSubjectKey(g.ID), Reason: reason})
	if err := o.approvalPublisher.PublishDomainEvent(ctx, event); err != nil {
		approvalLog().Warn("consent decision withdrawal failed", "authorization_id", g.ID.String(), obs.KeyError, err.Error())
	}
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
