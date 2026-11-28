# HTML explainers: the page contract

Every decision gets a page: one HTML file the Hub shows at the top of the decision,
in a sealed frame, above the answer form. How to make a good one is in `SKILL.md`
("Make the page"); this file is the contract and the kit. The cover fields still
carry the whole decision, because the push and the decision list never show the page.

Send it with `notification-hub conversations ask ... --explainer ./page.html`.
The contract (sealing, limits, bridge, check codes) is in
`docs/concepts/DECISION_PACKETS.md`, section "HTML explainers".

## 1. Hard rules

Each rule maps to a packet-check code. Two refuse an agent's ask (severity
`blocking`, nothing is stored): a page that only restates the cover, and a page
with text the operator cannot read. The rest are advice: the Hub still shows the
page, and the check names what will probably break inside the sealed frame.

| Rule | Why | Check code |
|---|---|---|
| Design for a 390 px wide phone first. Add `<meta name="viewport" content="width=device-width, initial-scale=1">`. | The operator reads on a phone. | `explainer_no_viewport` |
| Make the page self-contained. Load nothing from the network: no `http:`, `https:` or `//` address in an element's `src`, `poster` or `srcset`, a `<link href>`, a CSS `url()` or `@import`, or a `fetch()` call. Text, comments and plain `<a href>` links are not checked. | The sealed frame blocks the network, so such a resource never loads. | `explainer_network` |
| Show an attachment by its name: `<img src="chart.png">`. Attach every file the page names. | The Hub swaps names for the files. A name with no attachment shows nothing. | `explainer_unknown_attachment` |
| Work in light and dark. Colour with the kit's tokens. Every rule that sets a background sets `color` in the same rule; never a literal `white` or `#fff` background alone. | The Hub has both themes. Text under 3:1 contrast in either is refused. | `page_unreadable` (blocking) |
| Add what the cover cannot hold: the situation as a picture, chart, grouped list or table, the command, the cause, what each answer leads to. | The operator already read the question and options. A page that repeats them is refused. | `explainer_restates` (blocking) |
| Show the situation with at least one picture: a kit decision block (`.hub-proportion`, `.hub-groups`, `table.hub-matrix`, `.hub-timeline`, `.hub-compare`, `.hub-recurrence`, `.hub-command`), a chart with `role="img"`, a table, or a screenshot. | Option cards and callouts repeat the cover in sentences. | `page_no_picture` |
| Crop each screenshot to the part that changes (`height: 220px; object-fit: cover`). | A whole phone screenshot takes a whole screen. | `page_picture_uncropped` |
| A command the operator runs is on the page, in `.hub-command`, not inside a sentence. | `--technical-details` is not shown on the phone. | `command_hidden` |
| Put no form control in the page: no `<form>`, `<input>`, `<select>`, `<textarea>`. A `<button type="button">` is fine. | Only the Hub's form answers. | `explainer_form` |
| Never ask for the answer itself on the page. To help, call `hub.prefill`; the operator still presses send in the Hub. | A page can inform. The Hub answers. | none |
| Keep the HTML under 2 MiB. Put large images, video and models in attachments. | The Hub does not read a larger page for the check, and the frame may not show it. | `explainer_too_large` |
| Send one explainer per ask, as an `.html` file. | A second explainer or a folder becomes plain evidence (`attachment_explainer_limit`). A file that is not HTML is flagged and will not display as a page. | `explainer_not_html` |

## 2. Layout

Beyond `SKILL.md`'s rules (lead with the decision, real content, plain words, check
at 390 and 320 px):

- **One visual idea per page.** If you need two, you probably need two decisions.
- **Space things evenly.** The kit spaces in `rem` on one scale; tap targets 44 px or more.
- **Leave the page gutter to the Hub.** The kit gives `body` one `1rem` gutter, the
  same as the question above it. Do not add body padding or margin, a side-margin
  wrapper, a `max-width`, or a box around the whole page; pad the cards, not the
  page. A full-bleed band is fine: `margin: 0 -1rem; padding: 0 1rem`.
- The starters look different on purpose. Change fonts, colour, density and layout
  when the content asks for it.

## 3. The starter kit

The Hub adds the kit to every page. It is a stylesheet in `@layer hub-kit`, so
any style you write yourself wins, plus a small script for `.hub-compare`.
Opt out with `<meta name="hub-kit" content="off">`. Even then the Hub zeroes the
browser's 8 px `body` margin (in its own layer, so anything you write wins), so a
page with no kit still starts flush against the Hub's frame.

Tokens (CSS custom properties, light and dark): `--hub-bg`, `--hub-surface`,
`--hub-surface-muted`, `--hub-fg`, `--hub-muted`, `--hub-border`,
`--hub-primary`, `--hub-on-primary`, `--hub-accent`, `--hub-success`,
`--hub-warning`, `--hub-danger`, `--hub-series-1` to `--hub-series-6` (chart
colours), `--hub-font`, `--hub-mono`, `--hub-radius`, `--hub-gap`. Override any of them in your own `:root`.

If you write your own dark styles, key them to `:root[data-theme="dark"]`, which the
Hub always sets. A page whose own literal colours only work in light (or dark) is
shown in that theme only; tokens are the way to follow the operator's theme. Do not use `light-dark()`: iPhones older than iOS 17.5 drop it and
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
| `.hub-proportion` | A whole split into shares: a bar plus a legend. Largest first; fold the long tail into one share. `data-tone` 1–6. | see `examples/batch-by-group.html` |
| `.hub-groups` | Many items as a few groups, each with a switch. `data-hub-groups="ID"` fills the `--form` multi_choice question `ID` with the keys of the groups switched on; each `data-key` is a choice key. | see `examples/batch-by-group.html` |
| `.hub-command` | A command the operator runs: where, the line with a Copy button (`data-hub-copy`), steps, "Before it changes anything", "Undo". | see `examples/run-a-command.html` |
| `.hub-recurrence` | "You have been asked this N times": the count, the dates as marks, and the answer that stops it. | `<aside class="hub-recurrence"><strong class="hub-recurrence__count">3×</strong><div><p>You have been asked this 3 times since 13 Nov.</p><p class="hub-recurrence__stop">"Always back up secrets" ends it.</p></div></aside>` |
| `table.hub-matrix` | Options against criteria, with `td data-tone` marks (`good`, `warn` or `bad`) and `tr.is-recommended`. Rows become cards on a phone; give every cell a `data-label`. A cell holds a word, a number or a mark, not a sentence, so eight rows still fit a few screens. | `<table class="hub-matrix"><thead><tr><td></td><th scope="col">Cost</th></tr></thead><tbody><tr class="is-recommended"><th scope="row">Option A</th><td data-label="Cost" data-tone="good">Free</td></tr></tbody></table>` |

Scenario code builds the same blocks with `operatordecision.Page`: `Proportion`,
`Groups`, `Command`, `Recurrence`, `Matrix`.

For a table, give each cell a `data-label` so the phone layout can name it:
`<td data-label="Cost">$500</td>`. Always give images an `alt` text. Use color
together with words, never alone.

## 4. The bridge

The page talks to the Hub only through these calls. The Hub ignores any other
message. `window.hub` can be missing when the page is opened outside the Hub, so
check for it.

| Call | Effect |
|---|---|
| `hub.open(name, path?)` | Opens an attachment (or a file inside an attached folder) in the Hub viewer. Any element with `data-hub-open="name"` does this on tap. Use it for videos, 3D models (`.glb`), PDFs and long files that should not sit in the page. |
| `hub.prefill(questionId, response)` | Fills one form question in the draft. `hub.prefill("answer", "key")` fills the main option with an option key. For a form question, give a choice key or an answer in the same JSON form the form uses. The operator still presses send. |
| `hub.theme` | `"light"` or `"dark"`. The page root also gets `data-theme`, so CSS can use `:root[data-theme="dark"]`. |
| `hub.answer.set(key, label, value)` | Records one choice made on the page; the Hub's answer bar shows `label` and sends `{changes: {key: value}}` with the answer. Check that `hub.answer` exists first. |
| `hub.answer.remove(key)` | Takes that choice back out (for example when the operator returns to the starting state). |
| `hub.note(key, label)` | Opens a Hub-owned note sheet for one item. The saved `{label, text, attachments}` is sent under `changes.notes[key]`; attached image or file bytes are stored by the Hub and the report page never handles them directly. |
| `hub.openAsk(id)` | Navigates to the named Hub ask at `/asks/<id>`. Ask ids must be included in the page data's `asks[]` list for live context. |
| `await hub.asset("folder-name/relative/path")` | Resolves a file from an attached folder to a browser-loadable data URL. The Hub checks the folder entry and caps dynamic files at 12 MiB; a missing, invalid or larger file returns `null`. |
| `hub.context` | Read-only `{kind, recurrence, asks, asksTruncated}`. `recurrence` has `{count, dates, previousAnswers}`. `asks` maps up to 20 distinct ids from page data `asks[]` to `{state, askedAt, answeredAt, question}`; inaccessible ids are present with `state: "unknown"`. `asksTruncated` is true when the page named more ids. |

```html
<button type="button" id="pick">Choose B</button>
<script>
  document.getElementById("pick").addEventListener("click", function () {
    if (window.hub && window.hub.prefill) window.hub.prefill("answer", "second-opinion");
  });
</script>
```

Use the same keys as your `--option` and `--form` question ids.

## 5. Templates and the chart kit

The usage-mode decision, the daily briefing, the opportunity roadmap, the
structural-change decision and the research report are templates in `kit/templates/`, drawn by the chart
kit `kit/hub-charts.js` from one JSON object (`kit/schemas/<template>.schema.json`).
Send them with `--template NAME --data-file data.json`; do not write their HTML.
Each template keeps the rules above: no network, no form controls, Hub tokens in
both themes, under 2 MiB, and it renders read-only when the bridge is missing.
Components, answer keys and the render contract: `kit/README.md`.

## After you send

Check with `--dry-run` first: a `blocking` finding refuses the real send. After a
send, the packet check on stderr names what still reads poorly. Fix the page and
send only it: `ask --subject-key K --explainer page.html`. If it names a new
file, add `--attach`; to swap a file for one with another name, `--remove-attachment
old --attach new` (see `cli.md`).
