# HTML explainers for decisions

An explainer is one HTML page that the Hub shows at the top of a decision, in a
sealed frame, above the answer form. It exists for one reason: some decisions
are easier to see than to read. Everything else about the decision (question,
big picture, options, recommendation) still goes in the cover fields from
`SKILL.md`, because the push and the decision list never show the page.

Send it with `notification-hub conversations ask ... --explainer ./page.html`.
The contract (sealing, limits, bridge, check codes) is in
`docs/concepts/DECISION_PACKETS.md`, section "HTML explainers".

## 1. When to use HTML at all

Most decisions do not need an explainer. A clear question, a short big picture
and good option labels are faster to read than any page. Use HTML only when the
layout itself carries the meaning:

| The operator must... | Use | Example |
|---|---|---|
| Compare options side by side | `.hub-pair` | [choose-between-drafts.html](../examples/choose-between-drafts.html) |
| Weigh several options against a few criteria | `.hub-tradeoffs` | [plan-tradeoffs.html](../examples/plan-tradeoffs.html) |
| See what a change looks like before and after | `.hub-compare` | [compare-before-after.html](../examples/compare-before-after.html) |
| Follow steps in time, or a rollout | `.hub-timeline` | [rollout-timeline.html](../examples/rollout-timeline.html) |
| Watch, look at or turn something | `.hub-figure` and `data-hub-open` | [media-walkthrough.html](../examples/media-walkthrough.html) |

If a bulleted list in the big picture says the same thing, skip the page. If
the operator needs a file as evidence, attach it with `--attach`; do not build
a page to display it.

## 2. Hard rules

These are the only strict part. Each one maps to a packet-check code. Every
explainer rule is advice (severity `advice`): none refuses the ask, and none
removes the explainer. The Hub still shows the page, and the check names what will
probably break inside the sealed frame. The only change the Hub makes to the
explainer itself is for count and kind: a second explainer, or an explainer that
is a folder, is kept as ordinary evidence (`attachment_explainer_limit`). A
missing attachment file or a broken form question is a different part of the packet;
the Hub leaves those out with a `dropped` warning (see `SKILL.md`).

| Rule | Why | Check code |
|---|---|---|
| Design for a 390 px wide phone first. Add `<meta name="viewport" content="width=device-width, initial-scale=1">`. | The operator reads on a phone. | `explainer_no_viewport` |
| Make the page self-contained. Load nothing from the network: no `http:`, `https:` or `//` address in an element's `src`, `poster` or `srcset`, a `<link href>`, a CSS `url()` or `@import`, or a `fetch()` call. Text, comments and plain `<a href>` links are not checked. | The sealed frame blocks the network, so such a resource never loads. | `explainer_network` |
| Show an attachment by its name: `<img src="chart.png">`. Attach every file the page names. | The Hub swaps names for the files. A name with no attachment shows nothing. | `explainer_unknown_attachment` |
| Work in light and dark. Use the kit's colors (`var(--hub-fg)`) or `color-scheme`; never fix a white page background. | The Hub has both themes. | none |
| Put no form control in the page: no `<form>`, `<input>`, `<select>`, `<textarea>`. A `<button type="button">` is fine. | Only the Hub's form answers. | `explainer_form` |
| Never ask for the answer itself on the page. To help, call `hub.prefill`; the operator still presses send in the Hub. | A page can inform. The Hub answers. | none |
| Keep the HTML under 2 MiB. Put large images, video and models in attachments. | The Hub does not read a larger page for the check, and the frame may not show it. | `explainer_too_large` |
| Send one explainer per ask, as an `.html` file. | A second explainer or a folder becomes plain evidence (`attachment_explainer_limit`). A file that is not HTML is flagged and will not display as a page. | `explainer_not_html` |

## 3. Basics

This is advice. The kit already follows it, and you may bend all of it.

- **Lead with the decision.** The first screen says what is being chosen and
  what you recommend. Put detail after it.
- **One visual idea per page.** A page that compares two options does not also
  carry a timeline. If you need two ideas, you probably need two decisions.
- **Use plain words.** Write for a reader who has none of your context. Keep
  internal IDs, paths and jargon out of the page, as in the cover fields.
- **Use real content.** Real numbers, real names, real screens. Never ship
  placeholder text or "Lorem ipsum".
- **Space things evenly.** The kit sets spacing in `rem` on one scale. Keep to
  it, and leave room around tap targets (44 px high or more).
- **Say what happens next** and what happens if the operator does not answer.
  The cover already says it; repeat it only if the page would be confusing
  without it.
- **Check it narrow.** Look at the page at 390 px and at 320 px wide, in both
  themes, before you send.

The pages in `examples/` look different on purpose. They are techniques, not
templates. Change fonts, color, density and layout when the content asks for it.

## 4. The starter kit

The Hub adds the kit to every page. It is a stylesheet in `@layer hub-kit`, so
any style you write yourself wins, plus a small script for `.hub-compare`.
Opt out with `<meta name="hub-kit" content="off">`.

Tokens (CSS custom properties, light and dark): `--hub-bg`, `--hub-surface`,
`--hub-surface-muted`, `--hub-fg`, `--hub-muted`, `--hub-border`,
`--hub-primary`, `--hub-on-primary`, `--hub-accent`, `--hub-success`,
`--hub-warning`, `--hub-danger`, `--hub-font`, `--hub-mono`, `--hub-radius`,
`--hub-gap`. Override any of them in your own `:root`.

If you write your own dark styles, key them to `:root[data-theme="dark"]`, which the
Hub always sets. Do not use `light-dark()`: iPhones older than iOS 17.5 drop it and
the text or background disappears. The kit itself uses plain colors for this reason.

Plain HTML is already styled: `h1` to `h3`, `p`, lists, `a`, `code`, `img`,
`video`, `table`, `button` (add `class="is-quiet"` for a quiet one).

| Block | Use | Snippet |
|---|---|---|
| `.hub-pair` | Two things side by side. Stacks under 560 px. | `<div class="hub-pair"><section>...</section><section>...</section></div>` |
| `.hub-compare` | Before and after, with a slider. The first two children are the two sides. The kit adds the slider. Leave room at the top for the labels. | `<div data-hub-compare data-before-label="Today" data-after-label="Proposed"><div>...</div><div>...</div></div>` |
| `.hub-timeline` | Ordered steps. Mark one `is-done` or `is-now`. | `<ol class="hub-timeline"><li class="is-now"><span class="hub-when">Today</span><h3>You decide</h3><p>...</p></li></ol>` |
| `.hub-tradeoffs` | Pros and cons cards (a `div`), or options by criteria (a `table`) that stacks into cards on a phone. Add `class="is-recommended"` to the favoured card or row. | `<div class="hub-tradeoffs"><section><h3>Option</h3><ul class="pro"><li>...</li></ul><ul class="con"><li>...</li></ul></section></div>` |
| `.hub-callout` | One thing to notice. Add `is-success`, `is-warning` or `is-danger`. | `<div class="hub-callout is-warning"><strong>By 20 November.</strong> ...</div>` |
| `.hub-figure` | An image or video with a caption. | `<figure class="hub-figure"><img src="before.png" alt="..."><figcaption>...</figcaption></figure>` |
| `.hub-chip` | A short label. Add `is-primary`, `is-success`, `is-warning` or `is-danger`. | `<span class="hub-chip is-success">Recommended</span>` |

For a table, give each cell a `data-label` so the phone layout can name it:
`<td data-label="Cost">$500</td>`. Always give images an `alt` text. Use color
together with words, never alone.

## 5. The bridge

The page talks to the Hub only through these calls. The Hub ignores any other
message. `window.hub` can be missing when the page is opened outside the Hub, so
check for it.

| Call | Effect |
|---|---|
| `hub.open(name, path?)` | Opens an attachment (or a file inside an attached folder) in the Hub viewer. Any element with `data-hub-open="name"` does this on tap. Use it for videos, 3D models (`.glb`), PDFs and long files that should not sit in the page. |
| `hub.prefill(questionId, response)` | Fills one form question in the draft. `hub.prefill("answer", "key")` fills the main option with an option key. For a form question, give a choice key or an answer in the same JSON form the form uses. The operator still presses send. |
| `hub.theme` | `"light"` or `"dark"`. The page root also gets `data-theme`, so CSS can use `:root[data-theme="dark"]`. |

```html
<button type="button" id="pick">Choose B</button>
<script>
  document.getElementById("pick").addEventListener("click", function () {
    if (window.hub && window.hub.prefill) window.hub.prefill("answer", "second-opinion");
  });
</script>
```

Use the same keys as your `--option` and `--form` question ids.

## 6. Worked examples

Each file is a complete page. Its first comment names the technique and holds
the `conversations ask` command that sends it. They are based on real pending
decisions; where a decision has no real media, the comment says which files the
agent would attach.

| File | Technique |
|---|---|
| [choose-between-drafts.html](../examples/choose-between-drafts.html) | Two finished options side by side, with buttons that pre-fill the answer and open the full drafts. |
| [plan-tradeoffs.html](../examples/plan-tradeoffs.html) | Three options against a few criteria, as a table that becomes cards on a phone. |
| [compare-before-after.html](../examples/compare-before-after.html) | A change shown as before and after with a slider. No images needed: both sides are drawn in HTML. |
| [media-walkthrough.html](../examples/media-walkthrough.html) | A video, a photo, and a 3D model that opens in the Hub viewer, with steps. |
| [rollout-timeline.html](../examples/rollout-timeline.html) | A plan over time with a size comparison, a stop rule and a pre-filled answer. |

## After you send

The CLI prints the packet check to stderr. Each explainer line is advice, not an
error. To fix the page, send only what changes: run `ask --subject-key <key>
--explainer ./page.html` again. A revision keeps everything you leave out
(attachments, form, texts), and a sent explainer replaces the current one. If the
page names a new file, add it with `--attach`. To swap an attachment for one with a
different name, use `--remove-attachment old --attach new`; a file with a new name
alone is added next to the old one, and the page keeps pointing at the old name.
If an operator asks for more context, see "Decision packets" in `SKILL.md`.
