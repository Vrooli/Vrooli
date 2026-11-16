package operatordecision

import (
	"fmt"
	"strconv"
	"strings"
)

// The data parts of a decision page. Each one renders markup styled by
// Notification Hub's starter kit (classes hub-proportion, hub-groups,
// hub-command, hub-recurrence, hub-matrix in the Hub's kit.css), so they look
// right in the Hub's light and dark themes with no style of their own. A page
// written by hand uses the same classes; the Hub's skill has the snippets.
//
// Reach for these before Table: a list of items is usually a few groups and a
// few sizes, and the operator decides faster from a bar and three groups than
// from forty rows.

// Share is one part of a whole in Proportion: a label, its size as a number
// (any unit; only the ratio is drawn), the size as the operator reads it, and
// an optional note under the label.
type Share struct {
	Label   string
	Value   float64
	Display string
	Note    string
}

// Proportion adds a bar split by share with a legend: where the space, money,
// time or count goes. heading names the whole ("Where the 3.1 GiB is") and
// total is the whole as the operator reads it ("3.1 GiB"; "" to leave out).
// List the largest shares first and fold the long tail into one share
// ("40 others"); the kit colours up to six shares apart. A share too small to
// see still gets a sliver, and the legend carries every exact value.
func (p *Page) Proportion(heading, total string, shares ...Share) *Page {
	sum := 0.0
	for _, share := range shares {
		if share.Value > 0 {
			sum += share.Value
		}
	}
	if len(shares) == 0 || sum <= 0 {
		return p
	}
	var bar, legend, spoken strings.Builder
	for index, share := range shares {
		percent := 0.0
		if share.Value > 0 {
			percent = share.Value / sum * 100
		}
		tone := strconv.Itoa(index%6 + 1)
		width := strconv.FormatFloat(percent, 'f', 2, 64)
		bar.WriteString(`<span data-tone="` + tone + `" style="--share:` + width + `%"></span>`)
		legend.WriteString(`<li data-tone="` + tone + `"><span class="hub-proportion__label">` + esc(share.Label))
		if share.Note != "" {
			legend.WriteString("<small>" + esc(share.Note) + "</small>")
		}
		legend.WriteString(`</span><span class="hub-proportion__value">` + esc(share.Display) + `<small>` + esc(percentText(percent)) + "</small></span></li>")
		if index > 0 {
			spoken.WriteString(", ")
		}
		spoken.WriteString(share.Label + " " + share.Display + " (" + percentText(percent) + ")")
	}
	p.body.WriteString(`<figure class="hub-proportion"><figcaption><span>` + esc(heading) + "</span>")
	if total != "" {
		p.body.WriteString("<strong>" + esc(total) + "</strong>")
	}
	p.body.WriteString(`</figcaption><div class="hub-proportion__bar" role="img" aria-label="` + esc(spoken.String()) + `">` + bar.String() + "</div>")
	p.body.WriteString(`<ol class="hub-proportion__legend">` + legend.String() + "</ol></figure>")
	return p
}

// percentText is a share as a reader says it: "68%", "4%", "<1%".
func percentText(percent float64) string {
	switch {
	case percent <= 0:
		return "0%"
	case percent < 1:
		return "<1%"
	}
	return strconv.FormatFloat(percent, 'f', 0, 64) + "%"
}

// Group is one group of items in Groups. Key is the choice key of the form
// question the groups fill (see Groups); Title and Summary say what the group
// is and why it matters; Meta is a short count or size ("12 items · 2.9 GiB");
// Items are listed under a fold. Selected marks the groups the sender
// recommends; the page starts with them switched on.
type Group struct {
	Key, Title, Summary, Meta string
	Items                     []string
	Selected                  bool
}

// Groups adds grouped items the operator can switch on and off by group. With
// a questionID it is tied to the ask's multi_choice form question of that id
// (choice keys = group keys): each tap fills that question in the Hub's form
// through hub.prefill, and the operator still sends. With questionID "" it
// only shows the groups. A page cannot hold form controls, so the switches are
// buttons; the answer always comes from the Hub's form.
func (p *Page) Groups(questionID string, groups ...Group) *Page {
	if len(groups) == 0 {
		return p
	}
	p.body.WriteString(`<div class="hub-groups"`)
	if questionID != "" {
		p.body.WriteString(` data-hub-groups="` + esc(questionID) + `"`)
	}
	p.body.WriteString(">")
	for _, group := range groups {
		p.body.WriteString(`<section class="hub-group" data-key="` + esc(group.Key) + `">`)
		p.body.WriteString(`<div class="hub-group__head"><h3>` + esc(group.Title) + "</h3>")
		if questionID != "" {
			pressed := "false"
			if group.Selected {
				pressed = "true"
			}
			p.body.WriteString(`<button type="button" class="hub-group__toggle" aria-pressed="` + pressed + `" aria-label="` + esc("Include "+group.Title) + `"></button>`)
		}
		p.body.WriteString("</div>")
		if group.Meta != "" {
			p.body.WriteString(`<p class="hub-group__meta">` + esc(group.Meta) + "</p>")
		}
		if group.Summary != "" {
			p.body.WriteString("<p>" + esc(group.Summary) + "</p>")
		}
		if len(group.Items) > 0 {
			p.body.WriteString(fmt.Sprintf(`<details><summary>Show the %d items</summary><ul>`, len(group.Items)))
			for _, item := range group.Items {
				p.body.WriteString("<li>" + esc(item) + "</li>")
			}
			p.body.WriteString("</ul></details>")
		}
		p.body.WriteString("</section>")
	}
	p.body.WriteString("</div>")
	return p
}

// Command is something the operator must run themselves: the exact command,
// where to run it, what it does in order, what it shows before it changes
// anything, and how to undo it. Duration is how long it takes ("about 10
// minutes"); every field but Line is optional, but a command without Where
// and Undo is a command the operator cannot judge.
type Command struct {
	Line     string
	Where    string
	Steps    []string
	Shows    string
	Undo     string
	Duration string
}

// Command adds a command block on the page's first screen: where to run it,
// the command with a copy button, what it does, what it shows first, and the
// undo. A decision that asks the operator to run something shows it here, not
// in the technical details the phone never shows.
func (p *Page) Command(c Command) *Page {
	if strings.TrimSpace(c.Line) == "" {
		return p
	}
	p.body.WriteString(`<section class="hub-command">`)
	where := c.Where
	if where == "" {
		where = "Run this in a terminal"
	}
	p.body.WriteString(`<p class="hub-command__where">` + esc(where))
	if c.Duration != "" {
		p.body.WriteString(" <small>" + esc(c.Duration) + "</small>")
	}
	p.body.WriteString("</p>")
	p.body.WriteString(`<div class="hub-command__line"><code>` + esc(c.Line) + `</code><button type="button" class="is-quiet" data-hub-copy>Copy</button></div>`)
	if len(c.Steps) > 0 {
		p.body.WriteString(`<ol class="hub-command__steps">`)
		for _, step := range c.Steps {
			p.body.WriteString("<li>" + esc(step) + "</li>")
		}
		p.body.WriteString("</ol>")
	}
	if c.Shows != "" {
		p.body.WriteString(`<p><strong>Before it changes anything:</strong> ` + esc(c.Shows) + "</p>")
	}
	if c.Undo != "" {
		p.body.WriteString(`<p><strong>Undo:</strong> ` + esc(c.Undo) + "</p>")
	}
	p.body.WriteString("</section>")
	return p
}

// Recurrence is how often this decision has come back: Count asks (this one
// included) since Since ("4 Nov"), an optional Marks list with one short label
// per ask, oldest first ("4 Nov · keep on"), and Stop, the sentence that says
// which answer ends the repeats ("“Keep it on and stop asking” ends the daily
// question.").
type Recurrence struct {
	Count int
	Since string
	Marks []string
	Stop  string
}

// Recurrence adds a "you have seen this before" strip. Use it whenever the
// operator has been asked about the same thing before: it names the repeat,
// and Stop points at the option that fixes the cause.
func (p *Page) Recurrence(r Recurrence) *Page {
	if r.Count < 2 {
		return p
	}
	p.body.WriteString(`<aside class="hub-recurrence"><strong class="hub-recurrence__count">` + strconv.Itoa(r.Count) + `×</strong><div>`)
	line := "You have been asked this " + strconv.Itoa(r.Count) + " times"
	if r.Since != "" {
		line += " since " + r.Since
	}
	p.body.WriteString("<p>" + esc(line+".") + "</p>")
	if r.Stop != "" {
		p.body.WriteString(`<p class="hub-recurrence__stop">` + esc(r.Stop) + "</p>")
	}
	if len(r.Marks) > 0 {
		p.body.WriteString(`<ol class="hub-recurrence__marks">`)
		for _, mark := range r.Marks {
			p.body.WriteString(`<li title="` + esc(mark) + `"><span>` + esc(mark) + "</span></li>")
		}
		p.body.WriteString("</ol>")
	}
	p.body.WriteString("</div></aside>")
	return p
}

// Tones of a matrix cell: how the cell reads for the operator. The kit shows
// a mark and a colour, and the text always carries the meaning too.
const (
	ToneGood = "good"
	ToneWarn = "warn"
	ToneBad  = "bad"
)

// Cell is one cell of a Matrix row: its text and its tone ("", ToneGood,
// ToneWarn or ToneBad).
type Cell struct{ Text, Tone string }

// MatrixRow is one option (or item) in a Matrix: its key, the label the
// operator reads, and one cell per criterion.
type MatrixRow struct {
	Key, Label string
	Cells      []Cell
}

// Matrix adds options against criteria: one row per option, one column per
// criterion, each cell marked good, warn or bad. On a phone each row becomes a
// card with the criteria as labelled lines. recommended is the key of the row
// to highlight ("" for none).
func (p *Page) Matrix(criteria []string, recommended string, rows ...MatrixRow) *Page {
	if len(rows) == 0 || len(criteria) == 0 {
		return p
	}
	p.body.WriteString(`<table class="hub-matrix"><thead><tr><td></td>`)
	for _, criterion := range criteria {
		p.body.WriteString(`<th scope="col">` + esc(criterion) + "</th>")
	}
	p.body.WriteString("</tr></thead><tbody>")
	for _, row := range rows {
		if row.Key != "" && row.Key == recommended {
			p.body.WriteString(`<tr class="is-recommended" data-key="` + esc(row.Key) + `">`)
		} else {
			p.body.WriteString(`<tr data-key="` + esc(row.Key) + `">`)
		}
		p.body.WriteString(`<th scope="row">` + esc(row.Label) + "</th>")
		for index, criterion := range criteria {
			cell := Cell{}
			if index < len(row.Cells) {
				cell = row.Cells[index]
			}
			tone := ""
			switch cell.Tone {
			case ToneGood, ToneWarn, ToneBad:
				tone = ` data-tone="` + cell.Tone + `"`
			}
			p.body.WriteString(`<td data-label="` + esc(criterion) + `"` + tone + ">" + esc(cell.Text) + "</td>")
		}
		p.body.WriteString("</tr>")
	}
	p.body.WriteString("</tbody></table>")
	return p
}
