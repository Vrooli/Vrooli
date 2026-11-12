package home

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Needs-you grammar (FILE-CONTRACT.md §5). One parser for STATE.md
// "## Needs you" and ATTENTION.md "## Actionable now".

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
	// RecommendedProblem says why a `Recommended:` line could not be read.
	RecommendedProblem string
	Options            []Option
	Packet             string
	Secret             string
	// Delivered is the scheduler's `Delivered: <route> <UTC>` line: it
	// delivered the item itself, so the courier never sends it.
	Delivered string
	// Commission is an ATTENTION.md item's `Commission: <id>` line; empty
	// means a global item (rulings S4d).
	Commission string
	// Source is the file the item was read from.
	Source string
}

// Legacy is section text that is not an item (older numbered lists).
type Legacy struct {
	Source string
	Text   string
}

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
// `- **Delivered…**` bullet counts as a `Delivered:` line (F6).
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
		if m := optionText.FindStringSubmatch(value); m != nil {
			it.Recommended = &Option{Key: m[1], Label: m[2]}
			it.RecommendedProblem = ""
		} else {
			it.RecommendedProblem = "the Recommended: line does not start with `<key> — <label>`"
		}
	case "Options":
		it.Options = nil
		for _, part := range strings.Split(value, "; ") {
			if m := optionText.FindStringSubmatch(strings.TrimSpace(part)); m != nil {
				it.Options = append(it.Options, Option{Key: m[1], Label: m[2]})
			}
		}
	case "Packet":
		it.Packet = value
	case "Secret":
		it.Secret = value
	case "Delivered":
		it.Delivered = value
	case "Commission":
		it.Commission = value
	}
}

// Choices returns the item's options, or the defaults without an Options: line.
func (it *NeedsYouItem) Choices() []Option {
	if len(it.Options) > 0 {
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
// file that exists but cannot be read is an error, never "no items" (E1).
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
// "Still needed from you", "Decision", "What"), else its first line
// (rulings P6).
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
