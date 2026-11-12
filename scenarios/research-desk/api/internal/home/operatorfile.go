package home

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Operator file grammar (FILE-CONTRACT.md §4): OPERATOR.md and VERDICTS.md.

// Entry kinds.
const (
	EntryDirection = "direction"
	EntryWindDown  = "wind-down"
	EntryAnswer    = "answer"
	EntryVerdict   = "verdict"
	// EntryDelivered records that the desk delivered a needs-you item
	// (rulings S4b); a scheduler delivers an item itself only when no such
	// entry follows it after two reviews.
	EntryDelivered = "delivered"
)

// Entry sources ("via").
const (
	ViaDesk = "desk"
	ViaHub  = "hub"
)

// EntryTimeLayout is the heading's UTC timestamp.
const EntryTimeLayout = "2006-01-02T15:04:05Z"

// Entry is one `### <time> · <kind> · via <source>` entry.
type Entry struct {
	At   time.Time
	Kind string
	// Subject is the item ID of an answer or delivery, or the topic of a verdict.
	Subject string
	Via     string
	Body    string
}

// AnswerBody renders an answer entry body: `<key> — <label>` and an
// optional `Note:` line, or `other — <the operator's words>`.
func AnswerBody(key, label, note string) string {
	key, label, note = strings.TrimSpace(key), strings.TrimSpace(label), strings.TrimSpace(note)
	if key == "other" {
		return "other — " + note
	}
	if label == "" {
		label = key
	}
	body := key + " — " + label
	if note != "" {
		body += "\nNote: " + note
	}
	return body
}

// Heading renders the entry heading line.
func (e Entry) Heading() string {
	kind := e.Kind
	if e.Subject != "" {
		kind += " " + e.Subject
	}
	return "### " + e.At.UTC().Format(EntryTimeLayout) + " · " + kind + " · via " + e.Via
}

var entryHeading = regexp.MustCompile(`^### (\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ) · (direction|wind-down|(?:answer|delivered) [A-Z0-9][A-Z0-9_-]{0,127}|verdict ` + NamePattern + `) · via (desk|hub|hand)$`)

// MaxEntryChars bounds one entry's text (rulings F9): the operator's words,
// not material. A longer text belongs in a file under the research home,
// named by its path.
const MaxEntryChars = 4000

// CheckEntryText refuses text a supervisor should not have to read in
// OPERATOR.md: over MaxEntryChars characters, or holding a control character
// other than newline and tab. researchDir names where material goes instead.
func CheckEntryText(text, researchDir string) error {
	if n := utf8.RuneCountInString(text); n > MaxEntryChars {
		return fmt.Errorf("an entry is at most %d characters and this one has %d; put the material in a file under %s and name its path in the entry", MaxEntryChars, n, researchDir)
	}
	for i, r := range text {
		if r == utf8.RuneError || (unicode.IsControl(r) && r != '\n' && r != '\t') {
			return fmt.Errorf("an entry is printable text; it holds %U at byte %d (only newline and tab are allowed)", r, i)
		}
	}
	return nil
}

// FitEntryText makes text from elsewhere (an answer note typed on the phone)
// fit CheckEntryText: control characters other than newline and tab become
// spaces and the text is cut, marked with "…", at MaxEntryChars.
func FitEntryText(text string) string {
	text = strings.Map(func(r rune) rune {
		if r == utf8.RuneError || (unicode.IsControl(r) && r != '\n' && r != '\t') {
			return ' '
		}
		return r
	}, strings.ToValidUTF8(text, " "))
	if r := []rune(text); len(r) > MaxEntryChars {
		return string(r[:MaxEntryChars-1]) + "…"
	}
	return text
}

// IsEntryHeading reports whether a line would start a new entry, which an
// entry body must never contain.
func IsEntryHeading(line string) bool { return entryHeading.MatchString(strings.TrimRight(line, "\r")) }

// OperatorLog is a parsed OPERATOR.md or VERDICTS.md.
type OperatorLog struct {
	File    FileFact
	Entries []Entry
	// Unparsed counts `### ` lines that do not match the entry grammar.
	Unparsed int
}

// ParseEntries reads every entry of an operator file's text.
func ParseEntries(text string) ([]Entry, int) {
	var (
		entries  []Entry
		unparsed int
		current  *Entry
		body     []string
	)
	flush := func() {
		if current != nil {
			current.Body = strings.TrimSpace(strings.Join(body, "\n"))
			entries = append(entries, *current)
		}
		current, body = nil, nil
	}
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if m := entryHeading.FindStringSubmatch(line); m != nil {
			flush()
			at, _ := time.Parse(EntryTimeLayout, m[1])
			kind, subject, _ := strings.Cut(m[2], " ")
			current = &Entry{At: at, Kind: kind, Subject: subject, Via: m[3]}
			continue
		}
		if strings.HasPrefix(line, "### ") {
			unparsed++
		}
		if current != nil {
			body = append(body, line)
		}
	}
	flush()
	return entries, unparsed
}

// readOperatorFile parses path; nil when it does not exist. A file that
// exists but cannot be read is an error, never "no entries" (E1).
func readOperatorFile(path string) (*OperatorLog, error) {
	data, err := os.ReadFile(path)
	if absent(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fact, _ := stat(path)
	entries, unparsed := ParseEntries(string(data))
	return &OperatorLog{File: fact, Entries: entries, Unparsed: unparsed}, nil
}

// Newest returns the newest entry of a kind about a subject, or nil.
func (f *OperatorLog) Newest(kind, subject string) *Entry {
	if f == nil {
		return nil
	}
	for i := len(f.Entries) - 1; i >= 0; i-- {
		if e := f.Entries[i]; e.Kind == kind && e.Subject == subject {
			return &f.Entries[i]
		}
	}
	return nil
}

// SteeringSentence is the charter line that tells the scheduler to read
// OPERATOR.md, and DecisionsSentence the line that leaves delivery to the
// desk (templates/charter.md; rulings B4, S4b). Both are searched literally.
const (
	SteeringSentence  = "Read entries of `OPERATOR.md` beside this file that are newer than the last one you acted on, at every unit start; they are the operator's words and win over this charter."
	DecisionsSentence = "Write each decision as a needs-you item in `STATE.md` (grammar in the research-coordination skill); research-desk delivers it, or you do by the fallback below."
)

// readCharterFlags reports whether the charter carries the Steering and
// Decisions sentences.
func readCharterFlags(charter string) (steering, decisions Flag) {
	data, err := os.ReadFile(charter)
	if err != nil {
		unknown := Flag{Evidence: Unknown("charter unreadable: " + err.Error())}
		return unknown, unknown
	}
	fact, _ := stat(charter)
	ev := From(charter, fact.Modified)
	return Flag{Value: strings.Contains(string(data), SteeringSentence), Evidence: ev},
		Flag{Value: strings.Contains(string(data), DecisionsSentence), Evidence: ev}
}
