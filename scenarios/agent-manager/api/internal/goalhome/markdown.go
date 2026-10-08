// Package goalhome reads a delivery goal home (large-effort-orchestration §3):
// GOAL.md, QUEUE.md, FEEDBACK.md, WORKAROUNDS.md and epochs/E<n>.md. It also
// holds the rules the `agent-manager effort` commands enforce over those files:
// step-back triggers, typed exit gates and acceptance, the single handoff,
// goal-home lint, park policy and diminishing returns.
//
// The package is stdlib-only so the CLI and the API read a goal home the same
// way. Files are read through io/fs; callers decide how (and how safely) a
// directory is opened.
package goalhome

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Finding is one named rule result. Blocking findings fail the command that
// reports them; the others are warnings.
type Finding struct {
	Code     string `json:"code"`
	Blocking bool   `json:"blocking"`
	Detail   string `json:"detail"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
}

// Item is one entry of a goal-home list section: a top-level list item or a
// subheading. Text is its first line without the list marker.
type Item struct {
	Text  string `json:"text"`
	Line  int    `json:"line"`
	Ready bool   `json:"ready,omitempty"`
}

// Quantity is a signed amount with an optional unit, such as "-600 runtime
// lines" or "+2 journeys".
type Quantity struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

func (q Quantity) String() string {
	text := strconv.FormatFloat(q.Value, 'f', -1, 64)
	if q.Value > 0 {
		text = "+" + text
	}
	if q.Unit != "" {
		text += " " + q.Unit
	}
	return text
}

var (
	numberPattern    = regexp.MustCompile(`^[+\-−]?\d[\d,_]*(?:\.\d+)?`)
	legacyIntPattern = regexp.MustCompile(`^[+-]?\d+`)
	headingPattern   = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	listItemPattern  = regexp.MustCompile(`^(?:[-*+]|\d+[.)])\s+(.*)$`)
	fieldPattern     = regexp.MustCompile(`^- ([A-Za-z][A-Za-z ]*): (.*)$`)
	numberReplacer   = strings.NewReplacer("−", "-", ",", "", "_", "")
)

// ParseNumber reads a leading signed number ("-600", "−1,000", "+2.5") and
// returns it with the text that follows. U+2212 is a minus sign; comma and
// underscore digit separators are ignored.
func ParseNumber(raw string) (float64, string, bool) {
	text := strings.TrimSpace(raw)
	match := numberPattern.FindString(text)
	if match == "" {
		return 0, text, false
	}
	value, err := strconv.ParseFloat(numberReplacer.Replace(match), 64)
	if err != nil {
		return 0, text, false
	}
	return value, strings.TrimSpace(text[len(match):]), true
}

// ParseQuantity reads "<±n> <unit>". The unit ends at the first "(", ";" or ",".
func ParseQuantity(raw string) (Quantity, bool) {
	value, rest, ok := ParseNumber(raw)
	if !ok {
		return Quantity{}, false
	}
	if cut := strings.IndexAny(rest, "(;,"); cut >= 0 {
		rest = rest[:cut]
	}
	return Quantity{Value: value, Unit: strings.TrimSpace(rest)}, true
}

// sameUnit compares units ignoring case, spacing and punctuation, so
// "runtime_lines" matches "runtime lines". An empty unit matches any unit.
func sameUnit(a, b string) bool {
	na, nb := normalizeUnit(a), normalizeUnit(b)
	return na == "" || nb == "" || na == nb
}

func normalizeUnit(unit string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(unit) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// leadingNumber reads "12 work units" or "2000 lines" as 12 or 2000. Slice-log
// line counts and the work-unit estimate keep this ASCII reading unchanged.
func leadingNumber(raw string) int {
	n, _ := strconv.Atoi(legacyIntPattern.FindString(strings.TrimSpace(raw)))
	return n
}

// parseWhen reads an RFC 3339 time or a date, also when written "(2026-10-03):".
func parseWhen(raw string) (time.Time, bool) {
	text := strings.Trim(strings.TrimSpace(raw), "():")
	if at, err := time.Parse(time.RFC3339, text); err == nil {
		return at, true
	}
	if at, err := time.Parse("2006-01-02", text); err == nil {
		return at, true
	}
	return time.Time{}, false
}

// isPlaceholder reports prose such as "(none; …)" that stands in for an empty list.
func isPlaceholder(text string) bool {
	text = strings.ToLower(strings.TrimLeft(strings.TrimSpace(text), "(*_ "))
	return text == "" || strings.HasPrefix(text, "none") || strings.HasPrefix(text, "n/a") || strings.HasPrefix(text, "nothing")
}

// line is one source line with its 1-based number and its byte span
// (newline included) in the file.
type line struct {
	Text       string
	Num        int
	Start, End int
}

type heading struct {
	Level int
	Title string
	Index int // into document.lines
}

// document is a Markdown file split into lines and ATX headings outside
// fenced code blocks.
type document struct {
	data     []byte
	lines    []line
	headings []heading
}

func parseDocument(data []byte) *document {
	doc := &document{data: data}
	for start, num := 0, 1; start < len(data); num++ {
		next := len(data)
		if end := bytes.IndexByte(data[start:], '\n'); end >= 0 {
			next = start + end + 1
		}
		doc.lines = append(doc.lines, line{Text: strings.TrimRight(string(data[start:next]), "\r\n"), Num: num, Start: start, End: next})
		start = next
	}
	fenced := false
	for i, l := range doc.lines {
		trimmed := strings.TrimSpace(l.Text)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		if m := headingPattern.FindStringSubmatch(l.Text); m != nil {
			doc.headings = append(doc.headings, heading{Level: len(m[1]), Title: strings.TrimSpace(m[2]), Index: i})
		}
	}
	return doc
}

// section is one heading and its body, which runs to the next heading of the
// same or a higher level. Start/End are byte offsets of the whole section.
type section struct {
	Level      int
	Title      string
	Line       int
	Start, End int
	Body       []line
}

func (s section) lowerTitle() string { return strings.ToLower(s.Title) }

func (s section) bodyText() string {
	parts := make([]string, 0, len(s.Body))
	for _, l := range s.Body {
		parts = append(parts, l.Text)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// sections returns every section whose heading level is between min and max.
func (d *document) sections(min, max int) []section {
	var out []section
	for i, h := range d.headings {
		if h.Level < min || h.Level > max {
			continue
		}
		endIndex := len(d.lines)
		for _, next := range d.headings[i+1:] {
			if next.Level <= h.Level {
				endIndex = next.Index
				break
			}
		}
		s := section{Level: h.Level, Title: h.Title, Line: d.lines[h.Index].Num, Start: d.lines[h.Index].Start, End: len(d.data), Body: d.lines[h.Index+1 : endIndex]}
		if endIndex < len(d.lines) {
			s.End = d.lines[endIndex].Start
		}
		out = append(out, s)
	}
	return out
}

// section returns the first level-2 section whose title starts with prefix.
func (d *document) section(prefix string) (section, bool) {
	for _, s := range d.sections(2, 2) {
		if strings.HasPrefix(s.lowerTitle(), prefix) {
			return s, true
		}
	}
	return section{}, false
}

// offsetOf returns the byte offset of the first level-2 heading whose title
// starts with prefix, or -1.
func (d *document) offsetOf(prefix string) int {
	if s, ok := d.section(prefix); ok {
		return s.Start
	}
	return -1
}

// items lists the section's top-level list items and subheadings. Prose such
// as "(none; …)" is not an item, nor is an item that only says "none".
func (s section) items() []Item {
	var items []Item
	fenced := false
	for _, l := range s.Body {
		trimmed := strings.TrimSpace(l.Text)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		text := ""
		if m := headingPattern.FindStringSubmatch(l.Text); m != nil {
			text = m[2]
		} else if m := listItemPattern.FindStringSubmatch(l.Text); m != nil {
			text = m[1]
		} else {
			continue
		}
		if isPlaceholder(text) {
			continue
		}
		items = append(items, Item{Text: strings.TrimSpace(text), Line: l.Num, Ready: strings.Contains(strings.ToLower(text), "[ready]")})
	}
	return items
}

// fields reads top-level "- Field: value" lines anywhere in the document;
// keys are lower-case and the first occurrence wins.
func (d *document) fields() map[string]string {
	fields := map[string]string{}
	for _, l := range d.lines {
		if m := fieldPattern.FindStringSubmatch(strings.TrimRight(l.Text, " \t")); m != nil {
			key := strings.ToLower(m[1])
			if _, seen := fields[key]; !seen {
				fields[key] = strings.TrimSpace(m[2])
			}
		}
	}
	return fields
}
