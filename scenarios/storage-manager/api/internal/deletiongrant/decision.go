package deletiongrant

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/api-core/retention"

	repocontract "github.com/vrooli/repo-contract-go"
)

// pageRows caps the per-item table; the rest are summed in one line.
const pageRows = 80

// request is what one deletion request amounts to, in the operator's terms.
type request struct {
	in    RequestInput
	items []PathInfo
	bytes int64
	files int
	partial, anyGone,
	allGone bool
	count map[Disposition]int
}

func summarize(items []PathInfo, in RequestInput) request {
	r := request{in: in, items: items, count: map[Disposition]int{}, allGone: true}
	for _, it := range items {
		r.bytes += it.Bytes
		r.files += it.Files
		r.partial = r.partial || it.Partial
		r.anyGone = r.anyGone || !it.Exists
		r.allGone = r.allGone && !it.Exists
		r.count[it.Disposition]++
	}
	return r
}

// buildFacts writes the decision from what the request would delete: the
// cover a stranger can answer from the push, and a page with every item, its
// size, age, owner and whether it comes back.
func buildFacts(askID string, items []PathInfo, in RequestInput) operatordecision.Facts {
	r := summarize(items, in)
	n := len(items)
	approve := fmt.Sprintf("Delete it: the agent may remove exactly this %s within %s", r.kind(items[0]), humanDuration(in.TTL))
	keep := "Keep it: nothing is deleted and the agent is told no"
	if n > 1 {
		approve = fmt.Sprintf("Delete them: the agent may remove exactly these %d items within %s", n, humanDuration(in.TTL))
		keep = "Keep them: nothing is deleted and the agent is told no"
	}
	facts := operatordecision.Facts{
		// One open decision per path set: a repeat request updates it, and the
		// reconcile pass closes it when it lapses or its paths are gone.
		SubjectKey: askID,
		Subject:    operatordecision.Clip(fmt.Sprintf("Delete %s (%s)?", r.shortWhat(), r.size()), 120),
		Question: operatordecision.Clip(fmt.Sprintf("May an agent delete %s (%s) %s? %s",
			r.what(), r.size(), r.where(), r.consequence()), operatordecision.MaxQuestion),
		Options: []operatordecision.Option{{Key: "approve", Label: approve}, {Key: "deny", Label: keep}},
		// Deletion is irreversible: never auto-apply at a deadline.
		Reversible:       false,
		Urgency:          "normal",
		BigPicture:       operatordecision.Clip(r.bigPicture(), operatordecision.MaxBigPictureCodePoints),
		TechnicalDetails: r.technicalDetails(askID, operatordecision.MaxTechnicalDetailsCodePoints),
		Correlation:      correlation(askID, in),
	}
	facts.Recommended, facts.RecommendationReason = r.recommendation()
	facts.Explainer = r.page(facts)
	return facts
}

func correlation(askID string, in RequestInput) map[string]string {
	c := map[string]string{
		correlationAskID: askID,
		extraRequestedBy: in.RequestedBy,
		// Optional park companion: the agent, if it parks, parks on the
		// operator-decision producer with this key (the decision id).
		operatordecision.CorrelationAgentAwaitKey: askID,
	}
	// The trusted requester run id (from agent-provenance) lets agent-manager
	// deliver the answer back to the agent, parked or running; the stable
	// web-console session id is the fallback handle once its token ages out.
	if strings.TrimSpace(in.RaisedByRunID) != "" {
		c[operatordecision.CorrelationRaisedByRunID] = in.RaisedByRunID
	}
	if strings.TrimSpace(in.WebConsoleSessionID) != "" {
		c[operatordecision.CorrelationWebConsoleSessionID] = in.WebConsoleSessionID
	}
	return c
}

// recommendation: deny when protected data is in the set, approve when every
// item's owner says it rebuilds, and no preference otherwise, because only the
// operator knows whether undeclared data is still wanted.
func (r request) recommendation() (string, string) {
	switch {
	case r.count[Protected] > 0:
		return "deny", "Protected data does not come back once deleted; approve only if you know this copy is no longer needed."
	case r.count[Regenerable] == len(r.items):
		return "approve", "Their owners declare all of it rebuildable, so deleting it loses nothing that cannot come back."
	}
	return "", ""
}

func (r request) kind(it PathInfo) string {
	switch {
	case !it.Exists:
		return "item"
	case it.Dir:
		return "folder"
	}
	return "file"
}

// what names the set: one item by name, or a count and the name they share.
func (r request) what() string {
	if len(r.items) == 1 {
		return fmt.Sprintf("the %s “%s”", r.kind(r.items[0]), filepath.Base(r.items[0].Path))
	}
	noun := r.plural()
	if prefix := sharedPrefix(r.items); prefix != "" {
		return fmt.Sprintf("%d %s named “%s…”", len(r.items), noun, prefix)
	}
	return fmt.Sprintf("%d %s", len(r.items), noun)
}

func (r request) shortWhat() string {
	if len(r.items) == 1 {
		return operatordecision.Clip(filepath.Base(r.items[0].Path), 60)
	}
	return fmt.Sprintf("%d %s", len(r.items), r.plural())
}

func (r request) plural() string {
	dirs, files := 0, 0
	for _, it := range r.items {
		switch r.kind(it) {
		case "folder":
			dirs++
		case "file":
			files++
		}
	}
	switch len(r.items) {
	case dirs:
		return "folders"
	case files:
		return "files"
	}
	return "items"
}

// sharedPrefix is the start of the name every item shares, when it is long
// enough to recognise them by.
func sharedPrefix(items []PathInfo) string {
	prefix := filepath.Base(items[0].Path)
	for _, it := range items[1:] {
		name := filepath.Base(it.Path)
		i := 0
		for i < len(prefix) && i < len(name) && prefix[i] == name[i] {
			i++
		}
		prefix = prefix[:i]
	}
	for len(prefix) > 0 && !utf8.ValidString(prefix) {
		prefix = prefix[:len(prefix)-1]
	}
	if utf8.RuneCountInString(prefix) < 4 {
		return ""
	}
	return prefix
}

func (r request) size() string {
	if r.allGone {
		return "already gone"
	}
	if r.partial {
		return "at least " + retention.FormatBytes(r.bytes)
	}
	return retention.FormatBytes(r.bytes)
}

// where names the place in everyday words; the exact folder is on the page.
func (r request) where() string {
	parent := r.parent()
	home := r.in.home
	if home != "" {
		if runtimeHome, err := repocontract.VrooliUserRoot(home); err == nil && within(parent, runtimeHome) {
			return "in Vrooli's runtime folder"
		}
		if within(parent, home) {
			return "in your home folder"
		}
	}
	return "outside your home folder"
}

// parent is the deepest folder that holds every item.
func (r request) parent() string {
	parent := filepath.Dir(r.items[0].Path)
	for _, it := range r.items[1:] {
		for parent != string(filepath.Separator) && parent != "." && !within(it.Path, parent) {
			parent = filepath.Dir(parent)
		}
	}
	return parent
}

func within(path, root string) bool {
	return path == root || strings.HasPrefix(path, strings.TrimSuffix(root, string(filepath.Separator))+string(filepath.Separator))
}

// show is a path as the operator reads it: the home folder written as ~.
func (r request) show(path string) string {
	if r.in.home != "" && within(path, r.in.home) {
		return "~" + strings.TrimPrefix(path, r.in.home)
	}
	return path
}

// consequence is the one sentence that decides most answers: what does not
// come back.
func (r request) consequence() string {
	n := len(r.items)
	some := func(k int, one, all, part string) string {
		switch {
		case n == 1:
			return one
		case k == n:
			return fmt.Sprintf(all, n)
		}
		return fmt.Sprintf(part, k, n)
	}
	switch {
	case r.allGone:
		return some(n, "It is already gone.", "All %d are already gone.", "")
	case r.count[Protected] > 0:
		return some(r.count[Protected], "It is protected data that does not come back.",
			"All %d are protected data that does not come back.", "%d of the %d are protected data that does not come back.")
	case r.count[Undeclared] > 0:
		return some(r.count[Undeclared], "No owner declares it, so nothing would rebuild it.",
			"No owner declares any of the %d, so nothing would rebuild them.", "No owner declares %d of the %d, so nothing would rebuild them.")
	case r.count[OwnerConfirms] > 0:
		return some(r.count[OwnerConfirms], "Only its owner can say whether it is safe to delete.",
			"Only their owners can say whether the %d are safe to delete.", "Only their owners can say whether %d of the %d are safe to delete.")
	}
	return some(n, "Its owner says it rebuilds itself.", "Their owners say all %d rebuild themselves.", "")
}

func (r request) bigPicture() string {
	var b strings.Builder
	fmt.Fprintf(&b, "An agent asked to delete %s (%s) %s.", r.what(), r.size(), r.where())
	if reason := strings.TrimSpace(r.in.Reason); reason != "" {
		fmt.Fprintf(&b, " Its reason: “%s”", operatordecision.Clip(reason, 600))
	}
	fmt.Fprintf(&b, "\n\n%s Deleting cannot be undone.", r.consequence())
	fmt.Fprintf(&b, "\n\nIf you approve, the agent may delete exactly what is listed, nothing beside or above it, for the next %s. If you say no, or nobody answers within %s, nothing is deleted.",
		humanDuration(r.in.TTL), humanDuration(AskWindow))
	return b.String()
}

// comesBack is the answer to "if I delete it, does it come back?".
func comesBack(it PathInfo) string {
	switch it.Disposition {
	case Regenerable:
		return "Yes, it rebuilds itself"
	case Protected:
		return "No, protected data"
	case OwnerConfirms:
		return "Only its owner knows"
	}
	return "No, nothing rebuilds it"
}

func ownerName(it PathInfo) string {
	if it.Owner == "" {
		return "Nobody"
	}
	return it.Owner
}

func (r request) itemSize(it PathInfo) string {
	switch {
	case !it.Exists:
		return "already gone"
	case it.Partial:
		return "at least " + retention.FormatBytes(it.Bytes)
	}
	return retention.FormatBytes(it.Bytes)
}

// ago says how long since something changed, the way a person would.
func ago(now, t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	days := int(now.Sub(t).Hours() / 24)
	switch {
	case days < 1:
		return "today"
	case days < 2:
		return "yesterday"
	case days < 14:
		return fmt.Sprintf("%d days ago", days)
	case days < 60:
		return fmt.Sprintf("%d weeks ago", days/7)
	case days < 730:
		return fmt.Sprintf("%d months ago", days/30)
	}
	return fmt.Sprintf("%d years ago", days/365)
}

func humanDuration(d time.Duration) string {
	hours := int(d.Round(time.Hour).Hours())
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	case hours%24 == 0 && hours >= 24:
		if hours == 24 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", hours/24)
	case hours == 1:
		return "1 hour"
	}
	return fmt.Sprintf("%d hours", hours)
}

// page shows everything the request would delete: the total, what does not
// come back, a summary by owner, then each item, largest first.
func (r request) page(facts operatordecision.Facts) string {
	sorted := append([]PathInfo(nil), r.items...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Bytes > sorted[j].Bytes })

	newest, oldest := time.Time{}, time.Time{}
	for _, it := range sorted {
		if it.Changed.IsZero() {
			continue
		}
		if newest.IsZero() || it.Changed.After(newest) {
			newest = it.Changed
		}
		if oldest.IsZero() || it.Changed.Before(oldest) {
			oldest = it.Changed
		}
	}
	changed := ago(r.in.now, newest)
	if len(r.items) > 1 && ago(r.in.now, oldest) != changed {
		changed = fmt.Sprintf("between %s and %s", ago(r.in.now, newest), ago(r.in.now, oldest))
	}
	contents := fmt.Sprintf("%d %s", len(r.items), r.plural())
	if len(r.items) == 1 {
		contents = "1 " + r.kind(r.items[0])
	}
	if r.files > 0 && r.plural() != "files" {
		contents += fmt.Sprintf(", %d files inside", r.files)
	}

	page := operatordecision.NewPage(fmt.Sprintf("Delete %s (%s)?", r.what(), r.size()))
	lead := fmt.Sprintf("An agent asked to delete %s %s.", r.what(), r.where())
	if reason := strings.TrimSpace(r.in.Reason); reason != "" {
		lead += fmt.Sprintf(" Its reason: “%s”", reason)
	}
	page.Lead(lead).Facts(
		operatordecision.Row{Label: "Frees", Value: r.size()},
		operatordecision.Row{Label: "Holds", Value: contents},
		operatordecision.Row{Label: "Where", Value: r.show(r.parent())},
		operatordecision.Row{Label: "Last changed", Value: changed},
		operatordecision.Row{Label: "If you approve", Value: fmt.Sprintf("the agent may delete exactly these, for %s", humanDuration(r.in.TTL))},
	)
	tone := "warning"
	switch {
	case r.count[Protected] > 0:
		tone = "danger"
	case r.count[Regenerable] == len(r.items):
		tone = "success"
	}
	page.Callout(tone, "Does it come back?", r.consequence()+" Deleting cannot be undone.")
	if r.partial {
		page.Callout("", "Sizes are a floor.", "Measuring stopped before the end of a very large folder, so the real size is larger than shown.")
	}

	type group struct {
		owner, back string
		items       int
		bytes       int64
	}
	groups := map[string]*group{}
	var order []string
	for _, it := range sorted {
		key := ownerName(it) + "\x00" + comesBack(it)
		g := groups[key]
		if g == nil {
			g = &group{owner: ownerName(it), back: comesBack(it)}
			groups[key] = g
			order = append(order, key)
		}
		g.items++
		g.bytes += it.Bytes
	}
	if len(order) > 1 {
		rows := make([][]string, 0, len(order))
		for _, key := range order {
			g := groups[key]
			rows = append(rows, []string{g.owner, g.back, fmt.Sprint(g.items), retention.FormatBytes(g.bytes)})
		}
		page.Section("By owner", "").Table([]string{"Owner", "Comes back?", "Items", "Size"}, rows)
	}

	oneFolder := r.parent() == filepath.Dir(sorted[0].Path)
	for _, it := range sorted[1:] {
		oneFolder = oneFolder && filepath.Dir(it.Path) == r.parent()
	}
	headers := []string{"Name", "Size", "Last changed", "Owner", "Comes back?"}
	if !oneFolder {
		headers = []string{"Name", "Folder", "Size", "Last changed", "Owner", "Comes back?"}
	}
	rows := make([][]string, 0, min(len(sorted), pageRows))
	for _, it := range sorted[:min(len(sorted), pageRows)] {
		row := []string{filepath.Base(it.Path)}
		if !oneFolder {
			row = append(row, r.show(filepath.Dir(it.Path)))
		}
		rows = append(rows, append(row, r.itemSize(it), ago(r.in.now, it.Changed), ownerName(it), comesBack(it)))
	}
	note := "Largest first."
	if len(sorted) > pageRows {
		var rest int64
		for _, it := range sorted[pageRows:] {
			rest += it.Bytes
		}
		note = fmt.Sprintf("The %d largest of %d. The other %d hold %s together; every path is in the technical details.", pageRows, len(sorted), len(sorted)-pageRows, retention.FormatBytes(rest))
	}
	page.Section("What would be deleted", note).Table(headers, rows)

	n := len(r.items)
	page.Options(facts, map[string]string{
		"approve": fmt.Sprintf("Storage Manager opens deletion of exactly %s for %s. The agent is told and deletes %s; nothing beside or above is opened.", operatordecision.Plural(n, "this item", fmt.Sprintf("these %d items", n)), humanDuration(r.in.TTL), operatordecision.Plural(n, "it", "them")),
		"deny":    "Nothing is deleted. The agent is told no and has to find another way to free space.",
	})
	page.IfNoAnswer(fmt.Sprintf("Nothing is deleted. The request closes by itself after %s, or as soon as %s gone.", humanDuration(AskWindow), operatordecision.Plural(n, "the item is", "all the items are")))
	return page.HTML()
}

// technicalDetails is the exact path set for the next agent, within limit
// bytes (so within the code-point limit too): one line per containing directory with its count, then every path
// with its size and declaration until the budget runs out. The full set is
// stored on the ask; the grant opens only that set.
func (r request) technicalDetails(askID string, limit int) string {
	byDir := map[string]int{}
	for _, it := range r.items {
		byDir[filepath.Dir(it.Path)]++
	}
	dirs := make([]string, 0, len(byDir))
	for dir := range byDir {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	var out strings.Builder
	fmt.Fprintf(&out, "Decision: %s\nRequested by: %s\nGrant validity if approved: %s\nMeasured: %s", askID, r.in.RequestedBy, r.in.TTL, r.in.now.UTC().Format(time.RFC3339))
	if r.partial {
		out.WriteString(" (walk budget ran out: sizes marked >= are floors)")
	}
	fmt.Fprintf(&out, "\n\n%d path(s) in %d directory(ies):\n", len(r.items), len(dirs))
	for _, dir := range dirs {
		fmt.Fprintf(&out, "  %s/ — %d\n", dir, byDir[dir])
	}
	out.WriteString("\nPaths (size, newest change, declaration):\n")
	const reserve = 120 // room for the "not shown" line
	used := out.Len()   // bytes: they bound the code-point limit too, as Clip does
	for i, it := range r.items {
		size := retention.FormatBytes(it.Bytes)
		if it.Partial {
			size = ">=" + size
		}
		if !it.Exists {
			size = "gone"
		}
		decl := string(it.Disposition)
		if it.Source != "" {
			decl += ": " + it.Source
		}
		if it.Disposition == Protected {
			decl = fmt.Sprintf("PROTECTED: inside %s, which %s protects", it.Root, it.Source)
		}
		line := fmt.Sprintf("  - %s · %s · %s · %s\n", it.Path, size, changedStamp(it.Changed), decl)
		n := len(line)
		if used+n > limit-reserve {
			fmt.Fprintf(&out, "  … and %d more not shown (%d protected among all paths).\n", len(r.items)-i, r.count[Protected])
			break
		}
		out.WriteString(line)
		used += n
	}
	return operatordecision.Clip(out.String(), limit)
}

func changedStamp(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.UTC().Format("2006-01-02")
}
