package coverage

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"data-backup-manager/internal/attention"
	"data-backup-manager/internal/coverage"

	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/api-core/retention"
)

// Answer keys. standing backs up every current item and records the operator's
// consent to cover anything new that holds secrets; protect_all and
// reacquire_all apply one disposition to every current item; as_recommended
// applies each item's recommendation; groups backs up the groups switched on in
// the form question and leaves the rest; later records that nothing was decided; "other"
// (the Hub's free text) takes per-item JSON for exceptions.
const (
	answerStanding      = "standing"
	answerProtectAll    = "protect_all"
	answerReacquireAll  = "reacquire_all"
	answerAsRecommended = "as_recommended"
	answerLater         = "later"
	// answerGroups backs up the groups the operator switched on in the Hub's
	// "groups" form question (Response("groups").Selected holds the group keys).
	answerGroups = "groups"
	// groupsQuestionID is the form question's id, and the id Groups ties its switches to.
	groupsQuestionID = "groups"
)

// groupSpec describes one kind of owner the way the operator thinks of it.
type groupSpec struct {
	kind, key, title, short, holds string
}

var groupSpecs = []groupSpec{
	{"tool", "tool", "Your AI tools' home folders", "AI tools", "Sign-ins, chat history, memory and settings. The history exists nowhere else; a stale sign-in after a restore is fixed by signing in again."},
	{"resource", "resource", "AI tools and local services", "AI tools and local services", "Chat history, memory and sign-ins of the AI coding tools, plus service secrets such as Vault's encrypted store. The history and Vault's secrets exist nowhere else."},
	{"scenario", "scenario", "Vrooli apps' keys and records", "Vrooli apps", "Signing keys, saved logins and private records the apps created. Losing them breaks those apps' sign-ins or loses their history."},
	{"safeguard", "safeguard", "Host settings from setup", "host settings", "Kernel, AppArmor, sudo and systemd files that Vrooli setup wrote. Setup can write them again; a copy records exactly how they were set."},
	{"", "other", "Other items", "other items", "Data an owner declared as holding secrets."},
}

func groupSpecFor(kind string) groupSpec {
	for _, g := range groupSpecs {
		if g.kind == kind {
			return g
		}
	}
	return groupSpecs[len(groupSpecs)-1]
}

func groupTitle(kind string) string { return groupSpecFor(kind).title }
func groupKey(kind string) string   { return groupSpecFor(kind).key }

// sensitiveGroup is one group of candidates with its totals.
type sensitiveGroup struct {
	spec  groupSpec
	items []coverage.Suggestion
	bytes int64
}

func groupSensitive(items []coverage.Suggestion) []sensitiveGroup {
	by := map[string]*sensitiveGroup{}
	for _, spec := range groupSpecs {
		by[spec.key] = &sensitiveGroup{spec: spec}
	}
	for _, it := range items {
		g := by[groupKey(it.OwnerKind)]
		g.items = append(g.items, it)
		g.bytes += it.ApproxBytes
	}
	var out []sensitiveGroup
	for _, spec := range groupSpecs {
		g := by[spec.key]
		if len(g.items) == 0 {
			continue
		}
		sort.SliceStable(g.items, func(i, j int) bool { return g.items[i].ApproxBytes > g.items[j].ApproxBytes })
		out = append(out, *g)
	}
	return out
}

// ownerNames are the owners as the operator knows them.
var ownerNames = map[string]string{
	"claude-code": "Claude Code", "codex": "Codex", "grok": "Grok", "antigravity": "Antigravity",
	"workspace-sandbox": "Workspace sandbox", "swarm-manager": "Swarm Manager", "vault": "Vault",
	"kopia": "Kopia", "browser-automation-studio": "Browser Automation Studio", "notification-hub": "Notification Hub",
	"speaker-verification": "Speaker Verification", "web-console": "Web Console", "ollama": "Ollama",
	"adguard-home": "AdGuard Home", "twilio": "Twilio", "test-genie": "Test Genie",
}

// aiOwners are the AI coding tools whose history exists nowhere else.
var aiOwners = map[string]bool{"claude-code": true, "codex": true, "grok": true, "antigravity": true}

func ownerName(owner string) string {
	if name, ok := ownerNames[owner]; ok {
		return name
	}
	return strings.ReplaceAll(strings.ReplaceAll(owner, "_", " "), "-", " ")
}

// plainItems say what an item is to the operator, not what its manifest says.
var plainItems = map[string]string{
	"claude-code/home":                           "Claude Code history, memory, settings and sign-in",
	"codex/home":                                 "Codex sessions, memories and sign-in",
	"grok/home":                                  "Grok sessions, memory and sign-in",
	"antigravity/home":                           "Antigravity conversations, memory and sign-in",
	"workspace-sandbox/archives":                 "Saved patches and file copies from sandboxed agent work",
	"swarm-manager/events-db":                    "Swarm Manager's event history and approvals",
	"browser-automation-studio/session_profiles": "Saved browser logins",
	"notification-hub/decision-attachments":      "Files attached to your decisions",
	"speaker-verification/profiles":              "Saved voice profiles",
	"vault/data":                                 "Vault's encrypted secrets store",
	"vault/config":                               "Vault's settings",
	"kopia/config":                               "Backup tool settings and its repository key",
	"kopia/state":                                "Backup tool state",
	"ollama/credentials":                         "Ollama's sign-in key",
	"web-console/routed-agent-identity":          "The key that identifies your terminal agents",
	"web-console/routed-control-plane-keys":      "Web Console's signing keys",
	"notification-hub/vapid-private-key":         "The key that signs your phone notifications",
	"test-genie/seed_processes":                  "Test Genie's saved test processes",
	"test-genie/seed_sessions":                   "Test Genie's saved test sessions",
}

// plainItem is one candidate in the operator's words.
func plainItem(it coverage.Suggestion) string {
	text := plainItems[it.Owner+"/"+it.Name]
	if text == "" {
		if groupKey(it.OwnerKind) == "safeguard" {
			text = "Setup's " + strings.ReplaceAll(strings.ReplaceAll(it.Name, "_", " "), "-", " ") + " (" + ownerName(it.Owner) + ")"
		} else {
			text = ownerName(it.Owner) + ": " + strings.ReplaceAll(strings.ReplaceAll(it.Name, "_", " "), "-", " ")
		}
	}
	if len(it.Sidecars) > 0 {
		text += ", with its live " + strings.Join(it.Sidecars, " and ") + " files"
	}
	return text
}

// recommendedDisposition is reacquire only where the owner declares how to get
// the data back and it is not in the minimal recovery set; otherwise protect.
func recommendedDisposition(s sensitiveSnapshot) string {
	if s.ReacquisitionCommand != "" && s.ReacquisitionEvidence != "" && !s.Critical {
		return "reacquire"
	}
	return "protect"
}

// slice is one owner's share of the total size.
type slice struct {
	label string
	bytes int64
	items int
}

// ownerSlices ranks owners by size and folds the tail into "N others".
func ownerSlices(items []coverage.Suggestion, keep int) []slice {
	by := map[string]*slice{}
	var order []string
	for _, it := range items {
		s, ok := by[it.Owner]
		if !ok {
			s = &slice{label: ownerName(it.Owner)}
			by[it.Owner] = s
			order = append(order, it.Owner)
		}
		s.bytes += it.ApproxBytes
		s.items++
	}
	ranked := make([]slice, 0, len(order))
	for _, owner := range order {
		ranked = append(ranked, *by[owner])
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].bytes > ranked[j].bytes })
	if len(ranked) <= keep {
		return ranked
	}
	rest := slice{}
	for _, s := range ranked[keep:] {
		rest.bytes += s.bytes
		rest.items += s.items
	}
	rest.label = fmt.Sprintf("%d other places", rest.items)
	return append(ranked[:keep:keep], rest)
}

func sumBytes(items []coverage.Suggestion) int64 {
	var total int64
	for _, it := range items {
		total += it.ApproxBytes
	}
	return total
}

func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// aiHistoryOwners names the AI coding tools among the candidates, largest first.
func aiHistoryOwners(items []coverage.Suggestion) []string {
	var names []string
	seen := map[string]bool{}
	ranked := append([]coverage.Suggestion(nil), items...)
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].ApproxBytes > ranked[j].ApproxBytes })
	for _, it := range ranked {
		if aiOwners[it.Owner] && !seen[it.Owner] {
			seen[it.Owner] = true
			names = append(names, ownerName(it.Owner))
		}
	}
	return names
}

func ordinal(n int) string {
	words := []string{"", "first", "second", "third", "fourth", "fifth", "sixth"}
	if n >= 1 && n < len(words) {
		return words[n]
	}
	return fmt.Sprintf("%dth", n)
}

// sensitiveMenu offers only answers that can succeed for the current set, and
// puts the standing rule first because it is the answer that ends the question.
// effects say, per answer, what choosing it does in numbers the operator can
// check, so the page adds information instead of repeating the labels.
func sensitiveMenu(groups []sensitiveGroup, snapshots []sensitiveSnapshot) (options []operatordecision.Option, recommended string, effects map[string]string) {
	n := len(snapshots)
	var total int64
	for _, g := range groups {
		total += g.bytes
	}
	protect, reacquire := 0, 0
	reacquireAll := true
	for _, s := range snapshots {
		if s.ReacquisitionCommand == "" || s.ReacquisitionEvidence == "" {
			reacquireAll = false
		}
		if recommendedDisposition(s) == "reacquire" {
			reacquire++
		} else {
			protect++
		}
	}
	mixed := reacquire > 0
	effects = map[string]string{
		answerStanding:   fmt.Sprintf("All %d are added today (%s). Anything new that holds secrets is added automatically from now on, and you get a weekly note listing what was added. Your answer is recorded with the time, and you can turn the rule off or keep one item out later.", n, retention.FormatBytes(total)),
		answerProtectAll: fmt.Sprintf("All %d are added today (%s). Nothing is decided about new ones, so the next place that holds secrets asks you again.", n, retention.FormatBytes(total)),
		answerLater:      fmt.Sprintf("Nothing is added. The question closes after 7 days, and these %d places stay without a backup.", n),
	}
	recommended = answerStanding
	options = append(options,
		operatordecision.Option{Key: answerStanding, Label: fmt.Sprintf("Back up all %d now, and anything new that holds secrets from now on. Tell me what is added.", n)},
		operatordecision.Option{Key: answerProtectAll, Label: fmt.Sprintf("Back up all %d now. Ask me again about new ones.", n)},
	)
	if len(groups) > 1 {
		options = append(options, operatordecision.Option{Key: answerGroups, Label: "Back up only the groups I switch on below. Host settings start off."})
		effects[answerGroups] = "Adds only the groups switched on in the list above, and leaves out the rest. Those stay without a backup, and no standing rule is recorded. The groups start with the AI and app groups on and host settings off; change them before you send."
	}
	if mixed {
		// Some owners can give their data back, so backing up everything is no longer
		// the plain recommendation; the standing rule stays on the menu.
		recommended = answerAsRecommended
		options = append(options, operatordecision.Option{Key: answerAsRecommended, Label: "Follow each item's recommendation: back up what exists nowhere else, get the rest back from its owner"})
		effects[answerAsRecommended] = fmt.Sprintf("Backs up the %d that exist nowhere else and records the other %d as ones to get back from their owners. No standing rule is recorded.", protect, reacquire)
	}
	if reacquireAll {
		options = append(options, operatordecision.Option{Key: answerReacquireAll, Label: fmt.Sprintf("Back up none of the %d: get each back from its owner if it is lost", n)})
		effects[answerReacquireAll] = "Nothing is added. Each item is recorded as one to get back from its owner after a loss; that still needs the owner's own authorization."
	}
	options = append(options, operatordecision.Option{Key: answerLater, Label: "Not now: add nothing, and they stay without a backup"})
	return options, recommended, effects
}

// startsOn is whether a group starts switched on: everything except the host
// settings, which setup can write again.
func startsOn(g sensitiveGroup) bool { return g.spec.key != "safeguard" }

// groupsQuestion is the Hub form question the page's group switches fill: a
// multi-choice whose choice keys are the group keys.
func groupsQuestion(groups []sensitiveGroup) operatordecision.Question {
	q := operatordecision.Question{
		ID: groupsQuestionID, Type: operatordecision.QuestionMultiChoice,
		Prompt:               "Which groups should be backed up? Used only with \"Back up only the groups I switch on\".",
		RecommendationReason: "Host settings are tiny and setup can write them again; the AI and app groups exist nowhere else.",
	}
	for _, g := range groups {
		q.Choices = append(q.Choices, operatordecision.Choice{Key: g.spec.key, Label: g.spec.title, Detail: fmt.Sprintf("%d items · %s", len(g.items), retention.FormatBytes(g.bytes))})
		if startsOn(g) {
			q.Recommended = append(q.Recommended, g.spec.key)
		}
	}
	return q
}

func reasonFor(recommended string, total int64) string {
	if recommended == answerAsRecommended {
		return "Some of these can be re-created by their owners, so this backs up only what exists nowhere else. Choose the standing rule instead to stop the repeat questions."
	}
	return fmt.Sprintf("Your backups are encrypted, and the %s cost far less than losing this history. A standing rule also ends the repeat questions.", retention.FormatBytes(total))
}

// sensitiveReviewFacts writes the review from the current candidates: a cover
// a stranger can answer from the push, and a page that leads with what would be
// lost, shows where the size is, lets the operator choose by group and offers
// the standing rule that ends the question. Locators and owner commands stay
// out of every field.
func sensitiveReviewFacts(items []coverage.Suggestion, snapshots []sensitiveSnapshot, askID, fingerprint, generation, attentionDigest string, deadline time.Time, earlier []time.Time) operatordecision.Facts {
	n := len(snapshots)
	total := sumBytes(items)
	groups := groupSensitive(items)
	options, recommended, effects := sensitiveMenu(groups, snapshots)
	ai := aiHistoryOwners(items)

	subject := fmt.Sprintf("%d places that hold secrets have no backup. Back up from now on?", n)
	lost := fmt.Sprintf("%d places that hold sign-ins, keys or private history", n)
	if len(ai) > 0 {
		subject = "AI coding history has no backup. Back up secrets from now on?"
		lost = fmt.Sprintf("Your %s history and %d other places that hold secrets", joinNames(ai), n-len(ai))
	}
	askedTimes := len(earlier) + 1
	recurrence := ""
	if askedTimes > 1 {
		recurrence = fmt.Sprintf("This is the %s time Data Backup Manager has asked. Choosing the standing rule is what stops the question from coming back.\n\n", ordinal(askedTimes))
	}
	facts := operatordecision.Facts{
		Subject:              subject,
		Question:             fmt.Sprintf("%s have no backup (%s). Back them all up, and new ones from now on?", lost, retention.FormatBytes(total)),
		Options:              options,
		Recommended:          recommended,
		RecommendationReason: reasonFor(recommended, total),
		Reversible:           false,
		Urgency:              "high",
		BigPicture: fmt.Sprintf("Data Backup Manager found %d places on this machine that hold secrets, such as sign-ins, keys and private chat history, and have no backup. If the disk fails, they are gone.\n\n"+
			recurrence+
			"Because they hold secrets, it never adds them without you. The standing rule is that consent given once: from then on anything new that holds secrets is added automatically, and you get a weekly note listing what was added. You can turn the rule off, or keep one item out of it, at any time.\n\n"+
			"The risk is small: restoring an old sign-in can stop working until you sign in again. If you do not answer within 7 days, nothing is added.", n),
		TechnicalDetails: sensitiveTechnicalDetails(snapshots, attentionDigest),
		Deadline:         deadline,
		Correlation:      map[string]string{"ask_id": askID, "candidate_fingerprint": fingerprint, "generation": generation},
		SubjectKey:       "dbm-sensitive-coverage",
	}
	if len(groups) > 1 {
		facts.Questions = []operatordecision.Question{groupsQuestion(groups)}
	}
	facts.Explainer = sensitivePage(facts, groups, effects, total, earlier, deadline)
	return facts
}

func sensitiveTechnicalDetails(snapshots []sensitiveSnapshot, attentionDigest string) string {
	list := make([]string, 0, len(snapshots))
	for _, row := range snapshots {
		line := fmt.Sprintf("- %s · %s / %s · recommended %s", row.ID, row.Owner, row.Name, recommendedDisposition(row))
		if row.ReacquisitionCommand != "" {
			line += " · owner reacquisition evidence is declared"
		}
		if row.ReacquisitionNeedsOperator {
			line += " · requires separate owner/operator authorization"
		}
		list = append(list, line)
	}
	return operatordecision.Clip(fmt.Sprintf("Current sensitive candidates (%d):\n%s\n\n"+
		"Answers: standing = back up all and record the standing rule; groups = back up the groups selected in the groups form question (keys: tool, resource, scenario, safeguard, other); protect_all, as_recommended, reacquire_all, later.\n\n"+
		"For a per-item decision, choose Other and provide a JSON array with exactly one item per suggestion, for example [{\"suggestion_id\":\"stable-id\",\"decision\":\"protect\"},{\"suggestion_id\":\"another-id\",\"decision\":\"reacquire\"},{\"suggestion_id\":\"third-id\",\"decision\":\"exclude\",\"reason\":\"Owner confirmed this is ephemeral\"}]. "+
		"To also record the standing rule with exceptions, send {\"standing\":true,\"decisions\":[...]} with protect and exclude only; each excluded item stays out of the rule. "+
		"Each decision must be protect, reacquire, or exclude; every exclusion needs a reason. Reacquire is accepted only with owner-declared command and evidence metadata; any separate owner/operator authorization remains required. "+
		"The callback re-reads and fingerprints all candidates before any registration. The result is recorded and a follow-up notice is published. No locators, owner commands or credential values are included here.\n\n"+
		"The standing rule: `data-backup-manager coverage sensitive-policy` shows it; `data-backup-manager coverage revoke-sensitive-policy` turns it off; `data-backup-manager coverage exclude-sensitive --owner O --name N --reason R` keeps one item out.\n\n"+
		"Attention context (%s), Actionable now: %s.", len(snapshots), strings.Join(list, "\n"), attention.File(), attentionDigest), operatordecision.MaxTechnicalDetailsCodePoints)
}

// sensitivePage is the decision's page. It leads with what would be lost, shows
// where the size is, groups the items in the operator's words, and spends the
// option cards on what each answer does in numbers, not on repeating the menu.
func sensitivePage(facts operatordecision.Facts, groups []sensitiveGroup, effects map[string]string, total int64, earlier []time.Time, deadline time.Time) string {
	var all []coverage.Suggestion
	for _, g := range groups {
		all = append(all, g.items...)
	}
	n := len(all)
	page := operatordecision.NewPage(facts.Subject)
	ai := aiHistoryOwners(all)
	lead := fmt.Sprintf("If this computer's disk fails, these %d places are gone: they hold sign-ins, keys and private history, and none has a backup. Data Backup Manager never adds a place that holds secrets without you.", n)
	if len(ai) > 0 {
		lead = fmt.Sprintf("Your %s history, memory and sign-ins have no backup. If this computer's disk fails they are gone, along with %d other places that hold secrets. Data Backup Manager never adds a place that holds secrets without you.", joinNames(ai), n-len(ai))
	}
	page.Lead(lead)
	if len(earlier) > 0 {
		marks := make([]string, 0, len(earlier)+1)
		for _, at := range earlier {
			marks = append(marks, at.UTC().Format("2 Jan"))
		}
		marks = append(marks, "Today")
		page.Recurrence(operatordecision.Recurrence{
			Count: len(earlier) + 1, Since: earlier[0].UTC().Format("2 Jan"), Marks: marks,
			Stop: "Choosing the standing rule ends the question: anything new that holds secrets is then added for you, and you get a weekly note.",
		})
	}

	slices := ownerSlices(all, 4)
	shares := make([]operatordecision.Share, 0, len(slices))
	for _, s := range slices {
		shares = append(shares, operatordecision.Share{Label: s.label, Value: float64(s.bytes), Display: retention.FormatBytes(s.bytes), Note: sliceNote(s, all, groups)})
	}
	page.Proportion("Where the backup size is", retention.FormatBytes(total), shares...)

	cards := make([]operatordecision.Group, 0, len(groups))
	for _, g := range groups {
		items := make([]string, 0, len(g.items))
		for _, it := range g.items {
			items = append(items, plainItem(it)+" · "+retention.FormatBytes(it.ApproxBytes))
		}
		cards = append(cards, operatordecision.Group{
			Key: g.spec.key, Title: g.spec.title, Summary: g.spec.holds,
			Meta:  fmt.Sprintf("%d items · %s", len(g.items), retention.FormatBytes(g.bytes)),
			Items: items, Selected: startsOn(g),
		})
	}
	page.Section(fmt.Sprintf("The %d places, in %d groups", n, len(groups)), "Switch a group off to leave it out, then choose \"Back up only the groups I switch on\" below. Open a group to see what is in it.").Groups(groupsQuestionID, cards...)

	page.Options(facts, effects)
	page.Section("Staying in control of the rule", "The rule is recorded with your name and the time of your answer. You can look at it, turn it off, or keep one item out of it from a terminal on this computer.").
		Command(operatordecision.Command{
			Line:  "data-backup-manager coverage revoke-sensitive-policy",
			Where: "To turn the rule off later, run this in a terminal on this computer",
			Steps: []string{"Stops the rule from adding anything new.", "Leaves what it already added backed up."},
			Shows: "Run data-backup-manager coverage sensitive-policy first: it lists who consented, when, what the rule added and what it keeps out.",
			Undo:  "Run data-backup-manager coverage review-sensitive and choose the standing rule again. Nothing already backed up is lost either way.",
		})
	page.Callout("warning", "What a restore cannot fix:", "A restored sign-in may be out of date, so you would sign in again once. The data itself comes back intact.")
	page.IfNoAnswer("Nothing is added. After 7 days the question closes and these places stay without a backup.")
	return page.HTML()
}

// sliceNote says what a share of the bar is: the one item it holds, or the
// groups the long tail is made of.
func sliceNote(s slice, all []coverage.Suggestion, groups []sensitiveGroup) string {
	if s.items == 1 {
		for _, it := range all {
			if ownerName(it.Owner) == s.label {
				return strings.TrimPrefix(strings.TrimPrefix(plainItem(it), s.label+" "), s.label+": ")
			}
		}
	}
	if strings.HasSuffix(s.label, "other places") {
		kinds := map[string]bool{}
		var names []string
		shown := map[string]bool{}
		for _, sl := range ownerSlices(all, 4)[:4] {
			shown[sl.label] = true
		}
		for _, it := range all {
			if !shown[ownerName(it.Owner)] && !kinds[groupKey(it.OwnerKind)] {
				kinds[groupKey(it.OwnerKind)] = true
				names = append(names, groupSpecFor(it.OwnerKind).short)
			}
		}
		return "Mostly " + joinNames(names)
	}
	return ""
}
