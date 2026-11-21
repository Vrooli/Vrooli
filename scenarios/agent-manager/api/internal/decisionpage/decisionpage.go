// Package decisionpage holds what Agent Manager's decision producers share
// when they build a decision's page (operatordecision.Page): plain names for
// internal slugs, durations, a readable diff, a timeline, quotes and details.
// Clipping, plurals, the size fit and the cover check are api-core's
// operatordecision helpers; this package only adds the blocks they lack.
package decisionpage

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/vrooli/api-core/operatordecision"
)

// Name turns a slug such as "device-studio-delivery-2026-10-20" into words a
// stranger can read ("Device studio delivery"): separators become spaces and a
// trailing date is dropped.
func Name(slug string) string {
	s := strings.TrimSpace(slug)
	s = trailingStamp.ReplaceAllString(s, "")
	s = strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(s)
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return strings.TrimSpace(slug)
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

var trailingStamp = regexp.MustCompile(`(?i)([-_](\d{8}(t\d+z?)?|\d{4}-\d{2}-\d{2}))+$`)

// Duration says a span the way a person would: "26 minutes", "16 hours",
// "about 3 days".
func Duration(d time.Duration) string {
	switch {
	case d < time.Hour:
		return operatordecision.Count(max(int(d.Minutes()), 1), "minute", "minutes")
	case d < 48*time.Hour:
		return operatordecision.Count(int(d.Hours()), "hour", "hours")
	}
	return fmt.Sprintf("about %d days", int(d.Hours()+12)/24)
}

// Step is one entry of a Timeline.
type Step struct {
	When, Title, Text string
	Done, Now         bool
}

// Timeline renders ordered steps with the Hub kit's timeline block.
func Timeline(steps ...Step) string {
	var b strings.Builder
	b.WriteString(`<ol class="hub-timeline">`)
	for _, s := range steps {
		class := ""
		switch {
		case s.Now:
			class = ` class="is-now"`
		case s.Done:
			class = ` class="is-done"`
		}
		b.WriteString("<li" + class + ">")
		if s.When != "" {
			b.WriteString(`<span class="hub-when">` + html.EscapeString(s.When) + "</span>")
		}
		b.WriteString("<h3>" + html.EscapeString(s.Title) + "</h3>")
		if s.Text != "" {
			b.WriteString("<p>" + html.EscapeString(s.Text) + "</p>")
		}
		b.WriteString("</li>")
	}
	b.WriteString("</ol>")
	return b.String()
}

// Quote renders someone else's words (an agent's summary, a task brief) as an
// escaped block quote that keeps its line breaks.
func Quote(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return `<blockquote class="dp-quote">` + strings.ReplaceAll(html.EscapeString(text), "\n", "<br>") + "</blockquote>"
}

// Details renders a collapsed section holding escaped preformatted text.
func Details(summary, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return "<details class=\"dp-more\"><summary>" + html.EscapeString(summary) + "</summary><pre>" + html.EscapeString(text) + "</pre></details>"
}

// Diff renders a unified diff as one collapsible block per file, added lines
// green and removed lines red, at most max bytes of diff text. Files that do
// not fit are named in a closing note, so the reader knows the view is cut.
func Diff(unified string, max int) string {
	files := splitDiff(unified)
	if len(files) == 0 {
		return ""
	}
	var b strings.Builder
	used, shown := 0, 0
	for _, f := range files {
		if used+len(f.body) > max && shown > 0 {
			break
		}
		body := f.body
		if len(body) > max {
			body = body[:max]
		}
		used += len(body)
		shown++
		open := ""
		if len(files) <= 3 {
			open = " open"
		}
		fmt.Fprintf(&b, `<details class="dp-diff"%s><summary><code>%s</code> <span class="dp-add">+%d</span> <span class="dp-del">−%d</span></summary><pre>`,
			open, html.EscapeString(f.name), f.added, f.removed)
		for _, line := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
			class := ""
			switch {
			case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
				continue
			case strings.HasPrefix(line, "+"):
				class = "dp-add"
			case strings.HasPrefix(line, "-"):
				class = "dp-del"
			case strings.HasPrefix(line, "@@"):
				class = "dp-hunk"
			case strings.HasPrefix(line, "diff "), strings.HasPrefix(line, "index "), strings.HasPrefix(line, "new file"), strings.HasPrefix(line, "deleted file"):
				continue
			}
			if class == "" {
				b.WriteString(html.EscapeString(line) + "\n")
			} else {
				b.WriteString(`<span class="` + class + `">` + html.EscapeString(line) + "</span>\n")
			}
		}
		if len(body) < len(f.body) {
			b.WriteString(`<span class="dp-hunk">… this file's change is longer than the page can show</span>` + "\n")
		}
		b.WriteString("</pre></details>")
	}
	if rest := len(files) - shown; rest > 0 {
		fmt.Fprintf(&b, `<p class="dp-note">%s more not shown here; open the run in Agent Manager to read them.</p>`, operatordecision.Count(rest, "file", "files"))
	}
	return b.String()
}

type diffFile struct {
	name           string
	body           string
	added, removed int
}

func splitDiff(unified string) []diffFile {
	var files []diffFile
	var cur *diffFile
	var body strings.Builder
	flush := func() {
		if cur != nil {
			cur.body = body.String()
			files = append(files, *cur)
		}
		body.Reset()
	}
	for _, line := range strings.SplitAfter(unified, "\n") {
		trimmed := strings.TrimRight(line, "\n")
		if strings.HasPrefix(trimmed, "diff --git ") || (cur == nil && strings.HasPrefix(trimmed, "--- ")) {
			flush()
			cur = &diffFile{name: diffName(trimmed)}
		}
		if cur == nil {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "+++ "):
			if name := strings.TrimPrefix(strings.TrimPrefix(trimmed, "+++ "), "b/"); name != "/dev/null" {
				cur.name = name
			}
		case strings.HasPrefix(trimmed, "+"):
			cur.added++
		case strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "--- "):
			cur.removed++
		}
		body.WriteString(line)
	}
	flush()
	return files
}

func diffName(header string) string {
	if fields := strings.Fields(header); len(fields) >= 4 && fields[0] == "diff" {
		return strings.TrimPrefix(fields[len(fields)-1], "b/")
	}
	return strings.TrimPrefix(strings.TrimPrefix(header, "--- "), "a/")
}

// Style is the stylesheet for the blocks above. Add it once per page with
// Page.Raw; it uses only the Hub kit's tokens, so it works in both themes.
const Style = `<style>` +
	`.dp-diff,.dp-more{margin:.5rem 0;border:1px solid var(--hub-border);border-radius:var(--hub-radius);background:var(--hub-surface)}` +
	`.dp-diff summary,.dp-more summary{min-height:2.75rem;padding:.625rem .75rem;cursor:pointer;overflow-wrap:anywhere}` +
	`.dp-diff pre,.dp-more pre{margin:0;padding:.75rem;overflow-x:auto;font:.8125rem/1.45 var(--hub-mono);white-space:pre;background:var(--hub-surface-muted)}` +
	`.dp-more pre{white-space:pre-wrap;overflow-wrap:anywhere}` +
	`.dp-add{color:var(--hub-success)}.dp-del{color:var(--hub-danger)}.dp-hunk{color:var(--hub-muted)}` +
	`.dp-quote{margin:.5rem 0 1rem;padding:.25rem 0 .25rem .875rem;border-left:3px solid var(--hub-border);color:var(--hub-fg)}` +
	`.dp-note{color:var(--hub-muted);font-size:.9375rem}` +
	`</style>`
