package home

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Needs-you grammar (FILE-CONTRACT.md §5). One parser for STATE.md
// "## Needs you" and ATTENTION.md "## Actionable now". The grammar's cases live
// beside the agents' file check, skills/research-coordination/scripts/
// needs_you_cases.json, and both parsers run them (needsyou_cases_test.go).

// Option is one answer an item offers.
type Option struct {
	Key   string
	Label string
}

// NeedsYouItem is one `### <ID> — <title>` item.
type NeedsYouItem struct {
	ID    string
	Title string
	// Body is the item's text below the heading, verbatim.
	Body string
	// Recommended is set when a `Recommended: <key> — <label>` line names an option.
	Recommended *Option
	// RecommendedProblem says why a `Recommended:` line could not be parsed or
	// did not name one of the item's offered choices.
	RecommendedProblem string
	// Problems are every way the item breaks the grammar, RecommendedProblem
	// among them, worded as needs_you.py words them.
	Problems []string
	Options  []Option
	// optionsLine is true once an `Options:` line is read: an empty one
	// offers nothing rather than the defaults.
	optionsLine bool
	// decision is true once a `Recommended:` line is read, well formed or not.
	decision bool
	// Packet is the brief a worker executes on the answer: work/<ID>/BRIEF.md
	// of the level that raised the item. Every decision carries one.
	Packet string
	// Explainer is the optional `Explainer:` line: a self-contained HTML page
	// that shows the decision (a comparison, a plan, a schedule) and goes with
	// the Hub decision as its explainer. An absolute path, or one relative to
	// the folder of the file holding the item; see ExplainerPath.
	Explainer string
	Secret    string
	// Reason is the optional `Reason:` line: one plain sentence on why the
	// recommendation is right.
	Reason string
	// Details is the optional `Details:` line: IDs, paths and evidence for
	// the agent that acts on the answer, kept out of the operator's text.
	Details string
	// Delivered is the scheduler's `Delivered: <route> <UTC>` line: it
	// delivered the item itself, so the courier never sends it.
	Delivered string
	// Commission is an ATTENTION.md item's `Commission: <id>` line; empty
	// means a global item.
	Commission string
	// Source is the file the item was read from.
	Source string
}

// Legacy is section text that is not an item (older numbered lists).
type Legacy struct {
	Source string
	Text   string
	// CommissionID owns the STATE.md the text is in; empty for ATTENTION.md.
	CommissionID string
}

// Links are the markdown links in the text, resolved as an item's are.
func (l Legacy) Links() []Link { return NeedsYouItem{Body: l.Text, Source: l.Source}.Links() }

// Section names that hold needs-you items.
const (
	StateNeedsSection     = "Needs you"
	AttentionNeedsSection = "Actionable now"
	AttentionFile         = "ATTENTION.md"
)

// DefaultOptions apply to an item without an `Options:` line.
var DefaultOptions = []Option{{Key: "accept", Label: "Accept the recommendation"}, {Key: "hold", Label: "Hold this for now"}}

var (
	itemHeading = regexp.MustCompile(`^### ([A-Z0-9][A-Z0-9_-]{0,127}) — (.+?)\s*$`)
	optionText  = regexp.MustCompile(`^([a-z0-9-]{1,32}) — (.+)$`)
)

// emptySection is the only line of a section with no items.
const emptySection = "No actionable items are recorded."

// ParseNeedsYou reads the items and legacy text of one `## <section>` of text.
func ParseNeedsYou(text, section, source string) ([]NeedsYouItem, []Legacy) {
	var (
		items   []NeedsYouItem
		legacy  []string
		current *NeedsYouItem
		body    []string
		in      bool
	)
	flush := func() {
		if current == nil {
			return
		}
		if current.Recommended != nil {
			if _, ok := current.Choice(current.Recommended.Key); !ok {
				current.RecommendedProblem = `Recommended key "` + current.Recommended.Key + `" is not offered; fix the Recommended key or add it to Options`
				current.Recommended = nil
			}
		}
		if current.RecommendedProblem != "" {
			current.Problems = append(current.Problems, current.RecommendedProblem)
		}
		current.Problems = append(current.Problems, current.packetProblems()...)
		current.Body = strings.TrimSpace(strings.Join(body, "\n"))
		items = append(items, *current)
		current, body = nil, nil
	}
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			in = strings.TrimSpace(strings.TrimPrefix(line, "## ")) == section
			continue
		}
		if !in {
			continue
		}
		if strings.HasPrefix(line, "### ") {
			flush()
			m := itemHeading.FindStringSubmatch(line)
			if m == nil {
				// needs_you.py: "malformed item heading"; the desk shows it as
				// text that is not an item.
				legacy = append(legacy, line)
				continue
			}
			current = &NeedsYouItem{ID: m[1], Title: m[2], Source: source}
			continue
		}
		if current == nil {
			if t := strings.TrimSpace(line); t != "" && t != emptySection {
				legacy = append(legacy, line)
			}
			continue
		}
		body = append(body, line)
		current.field(line)
	}
	flush()
	var out []Legacy
	if text := strings.TrimSpace(strings.Join(legacy, "\n")); text != "" {
		out = append(out, Legacy{Source: source, Text: text})
	}
	return items, out
}

// field reads one optional `Name:` line of an item body. An older item's
// `- **Delivered…**` bullet counts as a `Delivered:` line.
func (it *NeedsYouItem) field(line string) {
	if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "- **Delivered"); ok {
		it.Delivered = strings.TrimSpace(strings.Trim(strings.TrimSpace(rest), "*:"))
		if it.Delivered == "" {
			it.Delivered = "an older Delivered line"
		}
		return
	}
	name, value, ok := strings.Cut(strings.TrimSpace(line), ":")
	if !ok {
		return
	}
	value = strings.TrimSpace(value)
	switch name {
	case "Recommended":
		it.decision = true
		if m := optionText.FindStringSubmatch(value); m != nil {
			it.Recommended = &Option{Key: m[1], Label: m[2]}
			it.RecommendedProblem = ""
		} else {
			it.RecommendedProblem = "malformed Recommended line; use Recommended: <key> — <label>"
		}
	case "Options":
		if it.optionsLine {
			it.Problems = append(it.Problems, "duplicate Options: line; keep one options list")
			return
		}
		it.optionsLine = true
		for _, part := range strings.Split(value, "; ") {
			m := optionText.FindStringSubmatch(strings.TrimSpace(part))
			switch {
			case value == "":
			case m == nil:
				it.Problems = append(it.Problems, "malformed option "+strconv.Quote(part)+"; use <key> — <label>")
			default:
				if _, dup := it.Choice(m[1]); dup {
					it.Problems = append(it.Problems, "duplicate option key "+strconv.Quote(m[1]))
				}
				it.Options = append(it.Options, Option{Key: m[1], Label: m[2]})
			}
		}
	case "Packet":
		it.Packet = value
	case "Explainer":
		it.Explainer = value
		if ext := strings.ToLower(filepath.Ext(value)); ext != ".html" && ext != ".htm" {
			it.Problems = append(it.Problems, "Explainer: must name an .html page, a path inside the research home")
		}
	case "Reason":
		it.Reason = value
	case "Details":
		it.Details = value
	case "Secret":
		it.Secret = value
	case "Delivered":
		it.Delivered = value
	case "Commission":
		it.Commission = value
	}
}

// BriefFile is the name of a decision's packet: the brief a worker gets.
const BriefFile = "BRIEF.md"

// packetProblems say why a decision's `Packet:` and `Explainer:` lines do not
// name its own folder, work/<ID>/, holding its brief and its page; none when
// they do. The raiser understands the decision best, so it writes the brief a
// worker executes on the answer cold (LEVELS.md, "Decisions travel up"); a
// workstream's brief makes the executor re-derive the decision.
func (it *NeedsYouItem) packetProblems() []string {
	if !it.decision {
		return nil
	}
	var out []string
	folder := filepath.Dir(it.Packet)
	switch {
	case it.Packet == "":
		out = append(out, "a decision needs a Packet: line naming its brief, work/"+it.ID+"/"+BriefFile+", so the answer can be executed without its raiser")
	case filepath.Base(it.Packet) != BriefFile || filepath.Base(folder) != it.ID:
		out = append(out, "Packet: must name the decision's brief, work/"+it.ID+"/"+BriefFile+", in a folder named for the decision, never a workstream's brief; not "+it.Packet)
	}
	switch {
	case it.Explainer == "":
		out = append(out, "a decision needs an Explainer: line naming its page beside its brief, work/"+it.ID+"/page.html")
	case it.Packet != "" && filepath.Dir(it.Explainer) != folder:
		out = append(out, "Explainer: must sit beside the brief, in "+folder+"/; not "+it.Explainer)
	}
	return out
}

// IsDecision is true for an item with a `Recommended:` line, well formed or not.
func (it *NeedsYouItem) IsDecision() bool { return it.decision }

// FreshDays is how old a brief's `Checked:` line may be when its decision goes
// out: the premise is checked live before the operator is asked.
const FreshDays = 7

var (
	checkedLine = regexp.MustCompile(`(?m)^Checked: (\d{4}-\d{2}-\d{2})\S*((?:\s+\S+){3,})`)
	// AnswerBullet is a `## On each answer` bullet: the answer's key, then what it settles.
	AnswerBullet  = regexp.MustCompile("^[-*]\\s+(?:\\*\\*)?`?([a-z0-9-]{1,32})`?(?:\\s*\\([^)]*\\))?[.:]?(?:\\*\\*)?[.:]?\\s*(?:— )?(.*)$")
	answersHeader = regexp.MustCompile(`(?m)^## On each answer[^\n]*\n`)
)

// BriefProblems say what a fresh executor would have to re-derive from a
// decision's brief: the premise checked live (a dated `Checked:` line saying
// what and how, at most FreshDays old at now; a zero now skips the age) and
// what each answer settles, `other` (the operator's own words) included.
// needs_you.py's brief_problems says the same; the briefs cases bind both.
func BriefProblems(text string, keys []string, now time.Time) []string {
	var out []string
	switch m := checkedLine.FindStringSubmatch(text); {
	case m == nil:
		out = append(out, "no `Checked: <YYYY-MM-DD> <the premise, verified live, and how>` line")
	case !now.IsZero():
		if day, err := time.Parse("2006-01-02", m[1]); err == nil && int(now.UTC().Truncate(24*time.Hour).Sub(day).Hours()/24) > FreshDays {
			out = append(out, "Checked: "+m[1]+" is over "+strconv.Itoa(FreshDays)+" days old; check the premise live again")
		}
	}
	said := map[string]bool{}
	if loc := answersHeader.FindStringIndex(text); loc != nil {
		body := "\n" + text[loc[1]:]
		if end := strings.Index(body, "\n## "); end >= 0 {
			body = body[:end]
		}
		for _, line := range strings.Split(body, "\n") {
			if b := AnswerBullet.FindStringSubmatch(strings.TrimSpace(line)); b != nil && strings.TrimSpace(b[2]) != "" {
				said[b[1]] = true
			}
		}
	}
	var missing []string
	for _, k := range append(append([]string{}, keys...), "other") {
		if !said[k] {
			missing = append(missing, "`"+k+"`")
		}
	}
	if len(missing) > 0 {
		out = append(out, "`## On each answer` says nothing for "+strings.Join(missing, ", ")+"; give each answer a bullet saying what it settles (`other` is the operator's own words)")
	}
	return out
}

// PacketPath is the item's packet as an absolute path, resolved as
// ExplainerPath is; empty when the item names none.
func (it NeedsYouItem) PacketPath() string {
	switch {
	case it.Packet == "":
		return ""
	case filepath.IsAbs(it.Packet) || it.Source == "":
		return it.Packet
	}
	return filepath.Join(filepath.Dir(it.Source), it.Packet)
}

// fieldNames are the `Name:` lines field reads; they are data for the desk,
// not prose for the operator.
var fieldNames = map[string]bool{
	"Recommended": true, "Options": true, "Packet": true, "Explainer": true, "Secret": true,
	"Reason": true, "Details": true, "Delivered": true, "Commission": true,
}

// Prose is the item's body without its field lines: the explanation written
// for the operator.
func (it *NeedsYouItem) Prose() string {
	var keep []string
	for _, line := range strings.Split(it.Body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "- **Delivered") {
			continue
		}
		if name, _, ok := strings.Cut(t, ":"); ok && fieldNames[name] {
			continue
		}
		keep = append(keep, line)
	}
	return strings.TrimSpace(strings.Join(keep, "\n"))
}

// ExplainerPath is where the item's explainer page is: the `Explainer:` line as
// an absolute path, a relative one resolved against the folder of the file the
// item was read from. Empty when the item names none.
func (it NeedsYouItem) ExplainerPath() string {
	switch {
	case it.Explainer == "":
		return ""
	case filepath.IsAbs(it.Explainer) || it.Source == "":
		return it.Explainer
	}
	return filepath.Join(filepath.Dir(it.Source), it.Explainer)
}

// Choices returns the item's options, or the defaults without an Options: line.
func (it *NeedsYouItem) Choices() []Option {
	if len(it.Options) > 0 || it.optionsLine {
		return it.Options
	}
	return DefaultOptions
}

// Choice returns the option with key, if the item offers it.
func (it *NeedsYouItem) Choice(key string) (Option, bool) {
	for _, o := range it.Choices() {
		if o.Key == key {
			return o, true
		}
	}
	return Option{}, false
}

// readNeedsYou parses one file's section. A missing file has no items; a
// file that exists but cannot be read is an error, never "no items".
func readNeedsYou(path, section string) ([]NeedsYouItem, []Legacy, error) {
	data, err := os.ReadFile(path)
	if absent(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	items, legacy := ParseNeedsYou(string(data), section, path)
	return items, legacy, nil
}

// Link is one markdown link in an item's text.
type Link struct {
	Label string
	// Target is resolved against the folder of the file holding the item
	// when it is a relative path; a URL or an absolute path stays as written.
	Target string
}

var (
	markdownLink = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	labelledLine = regexp.MustCompile(`^[-*]\s+\*\*([^*]+?):?\*\*:?\s*(.*)$`)
)

// Instruction is what the item itself says to do: the text of its first
// labelled line whose label names the operator's part ("Needs you",
// "Still needed from you", "Decision", "What"), else its first line.
func (i NeedsYouItem) Instruction() string {
	first := ""
	for _, line := range strings.Split(i.Body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := labelledLine.FindStringSubmatch(line)
		if m == nil {
			if first == "" {
				first = strings.TrimLeft(line, "-* ")
			}
			continue
		}
		label := strings.ToLower(m[1])
		for _, word := range []string{"need", "decision", "what", "action"} {
			if strings.Contains(label, word) {
				return m[2]
			}
		}
		if first == "" {
			first = m[2]
		}
	}
	return first
}

// Links lists the item's markdown links in order.
func (i NeedsYouItem) Links() []Link {
	var out []Link
	for _, m := range markdownLink.FindAllStringSubmatch(i.Body, -1) {
		target := m[2]
		if !strings.Contains(target, "://") && !strings.HasPrefix(target, "/") && !strings.HasPrefix(target, "#") && !strings.HasPrefix(target, "mailto:") && i.Source != "" {
			target = filepath.Join(filepath.Dir(i.Source), target)
		}
		out = append(out, Link{Label: m[1], Target: target})
	}
	return out
}
