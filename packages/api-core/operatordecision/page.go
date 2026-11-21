package operatordecision

import (
	"html"
	"strings"
	"time"
)

// Page builds a decision's explainer (Facts.Explainer): one phone-first HTML
// page that Notification Hub shows above the answer buttons, styled by the
// Hub's starter kit (light and dark, 390 px first). Every text is escaped;
// only Compare and Raw take markup.
//
//	page := operatordecision.NewPage("Delete 4.2 GB of old build caches?").
//		Lead("The disk is 93% full. These caches rebuild on their own.").
//		Table([]string{"Folder", "Size", "Last used"}, rows).
//		Options(facts, map[string]string{"delete": "Frees 4.2 GB now."}).
//		IfNoAnswer("Nothing is deleted; the disk keeps filling.")
//	facts.Explainer = page.HTML()
//
// Lead with what is happening and why it is the operator's to decide, show
// the one thing that makes the choice clear (a table, a before/after, a
// picture), then the options with what each one does.
type Page struct {
	title, generator string
	body             strings.Builder
}

// Row is one labelled value in Facts.
type Row struct{ Label, Value string }

// NewPage starts a page whose heading is the decision in plain words.
func NewPage(title string) *Page { return &Page{title: title} }

func esc(s string) string { return html.EscapeString(s) }

// paragraphs turns blank-line separated text into <p> elements.
func paragraphs(text string) string {
	var b strings.Builder
	for _, part := range strings.Split(strings.TrimSpace(text), "\n\n") {
		if part = strings.TrimSpace(part); part != "" {
			b.WriteString("<p>" + strings.ReplaceAll(esc(part), "\n", "<br>") + "</p>")
		}
	}
	return b.String()
}

// Lead adds the opening paragraphs: what is happening and why it matters now.
func (p *Page) Lead(text string) *Page { p.body.WriteString(paragraphs(text)); return p }

// Section adds a heading, and optional paragraphs under it.
func (p *Page) Section(heading, text string) *Page {
	p.body.WriteString("<h2>" + esc(heading) + "</h2>" + paragraphs(text))
	return p
}

// Facts adds a short list of labelled values (what, where, how big, since when).
func (p *Page) Facts(rows ...Row) *Page {
	if len(rows) == 0 {
		return p
	}
	p.body.WriteString(`<dl class="od-facts">`)
	for _, r := range rows {
		p.body.WriteString("<dt>" + esc(r.Label) + "</dt><dd>" + esc(r.Value) + "</dd>")
	}
	p.body.WriteString("</dl>")
	return p
}

// Table adds a table that stacks into labelled cards on a phone.
func (p *Page) Table(headers []string, rows [][]string) *Page {
	if len(rows) == 0 {
		return p
	}
	p.body.WriteString(`<table class="hub-tradeoffs"><thead><tr>`)
	for _, h := range headers {
		p.body.WriteString("<th>" + esc(h) + "</th>")
	}
	p.body.WriteString("</tr></thead><tbody>")
	for _, row := range rows {
		p.body.WriteString("<tr>")
		for i, cell := range row {
			label := ""
			if i < len(headers) {
				label = headers[i]
			}
			p.body.WriteString(`<td data-label="` + esc(label) + `">` + esc(cell) + "</td>")
		}
		p.body.WriteString("</tr>")
	}
	p.body.WriteString("</tbody></table>")
	return p
}

// Callout adds one thing to notice. Tone is "", "success", "warning" or "danger".
func (p *Page) Callout(tone, strong, text string) *Page {
	class := "hub-callout"
	switch tone {
	case "success", "warning", "danger":
		class += " is-" + tone
	}
	inner := esc(text)
	if strong != "" {
		inner = "<strong>" + esc(strong) + "</strong> " + inner
	}
	p.body.WriteString(`<div class="` + class + `">` + inner + "</div>")
	return p
}

// Image adds a picture with a caption. src is a data: URI (the page loads
// nothing from the network) or the name of an attachment.
func (p *Page) Image(src, alt, caption string) *Page {
	p.body.WriteString(`<figure class="hub-figure"><img src="` + esc(src) + `" alt="` + esc(alt) + `">`)
	if caption != "" {
		p.body.WriteString("<figcaption>" + esc(caption) + "</figcaption>")
	}
	p.body.WriteString("</figure>")
	return p
}

// Compare adds a before/after slider. The two sides are trusted markup the
// producer built (escape any text you put in them).
func (p *Page) Compare(beforeLabel, beforeHTML, afterLabel, afterHTML string) *Page {
	p.body.WriteString(`<div data-hub-compare data-before-label="` + esc(beforeLabel) + `" data-after-label="` + esc(afterLabel) + `"><div class="od-side">` + beforeHTML + `</div><div class="od-side">` + afterHTML + "</div></div>")
	return p
}

// Options adds one card per option of f: its label, what choosing it does
// (effects, by option key), the recommendation and its reason, and a button
// that fills in the answer (the operator still presses send in the Hub).
func (p *Page) Options(f Facts, effects map[string]string) *Page {
	p.body.WriteString(`<h2>Your choices</h2><p class="od-how">Tap “Choose this”, then send.</p><div class="hub-tradeoffs">`)
	for _, o := range f.Options {
		class := ""
		if o.Key == f.Recommended {
			class = ` class="is-recommended"`
		}
		p.body.WriteString("<section" + class + ">")
		if o.Key == f.Recommended {
			p.body.WriteString(`<span class="hub-chip is-primary">Recommended</span>`)
		}
		p.body.WriteString("<h3>" + esc(f.Label(o.Key)) + "</h3>")
		if effect := effects[o.Key]; effect != "" {
			p.body.WriteString(paragraphs(effect))
		}
		if o.Key == f.Recommended && f.RecommendationReason != "" {
			p.body.WriteString(`<p class="od-why">` + esc(f.RecommendationReason) + "</p>")
		}
		p.body.WriteString(`<button type="button" data-od-answer="` + esc(o.Key) + `">Choose this</button></section>`)
	}
	p.body.WriteString("</div>")
	return p
}

// WhenApproved says what happens when the operator approves: who acts and
// from what. briefAttachment, when not empty, names the attached brief the
// work starts from, and the page offers a button that opens it.
func (p *Page) WhenApproved(text, briefAttachment string) *Page {
	p.body.WriteString("<h2>What happens when you approve</h2>" + paragraphs(text))
	if briefAttachment != "" {
		p.body.WriteString(`<p><button type="button" data-hub-open="` + esc(briefAttachment) + `">Read the prepared plan</button></p>`)
	}
	return p
}

// IfNoAnswer says what happens when nobody answers.
func (p *Page) IfNoAnswer(text string) *Page {
	return p.Callout("warning", "If you do not answer:", text)
}

// Raw appends trusted markup the producer built.
func (p *Page) Raw(markup string) *Page { p.body.WriteString(markup); return p }

const pageStyle = `.od-facts{display:grid;grid-template-columns:auto 1fr;gap:.25rem .75rem;margin:0 0 1rem}` +
	`.od-facts dt{font-weight:600;color:var(--hub-muted)}.od-facts dd{margin:0;overflow-wrap:anywhere}` +
	`.od-why{color:var(--hub-muted);font-size:.9375rem}.od-side{padding:2.5rem 1rem 1rem}` +
	`.hub-tradeoffs section button{margin-top:.5rem}.od-how{color:var(--hub-muted);margin:-.25rem 0 .75rem}`

const pageScript = `document.addEventListener("click",function(e){var b=e.target.closest&&e.target.closest("[data-od-answer]");` +
	`if(b&&window.hub&&window.hub.prefill)window.hub.prefill("answer",b.getAttribute("data-od-answer"))});`

func (p *Page) meta() string {
	if p.generator == "" {
		return ""
	}
	return `<meta name="generator" content="` + esc(p.generator) + `">`
}

// CoverPageGenerator marks a page CoverPage built, so a reader can tell the
// floor page from one the sender made.
const CoverPageGenerator = "operatordecision.CoverPage"

// CoverPage is the page a decision gets when its sender made none: the
// question, the big picture, one card per option with the recommendation and
// its reason, what happens on approval (when there is a packet) and what
// happens with no answer. Notification Hub attaches it to every ask that has
// no page of its own. It is a floor, not a good page: it can only re-show the
// cover, and the sender is still advised to make a real one.
//
// briefAttachment names the attached brief, when there is one; deadline is
// when the ask closes (zero when unknown).
func CoverPage(f Facts, briefAttachment string, deadline time.Time) string {
	page := NewPage(f.Question)
	page.generator = CoverPageGenerator
	page.Lead(f.BigPicture).Options(f, nil)
	if f.Packet != "" {
		page.WhenApproved("The work is already planned. The agent that asked starts it from the prepared plan, so nothing you decide here has to be explained again.", briefAttachment)
	}
	return page.IfNoAnswer(noAnswerText(f, deadline)).HTML()
}

// noAnswerText says what an unanswered decision does, from its default and deadline.
func noAnswerText(f Facts, deadline time.Time) string {
	when := ""
	if !deadline.IsZero() {
		when = deadline.UTC().Format("Mon Jan 2, 15:04 UTC")
	}
	switch {
	case f.DefaultAnswer != "" && when != "":
		return "At " + when + " “" + f.Label(f.DefaultAnswer) + "” is chosen for you."
	case f.DefaultAnswer != "":
		return "“" + f.Label(f.DefaultAnswer) + "” is chosen for you when the decision runs out of time."
	case when != "":
		return "Nothing is chosen for you. The decision stays open until " + when + ", then closes unanswered, and the work that asked does not go ahead."
	}
	return "Nothing is chosen for you, and the work that asked does not go ahead until you answer."
}

// HTML is the finished page.
func (p *Page) HTML() string {
	return `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width, initial-scale=1">` +
		p.meta() + "<title>" + esc(p.title) + "</title><style>" + pageStyle + "</style></head><body>" +
		"<h1>" + esc(p.title) + "</h1>" + p.body.String() +
		"<script>" + pageScript + "</script></body></html>"
}
