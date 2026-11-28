# Notification Hub chart kit and page templates

Six pages the operator reads on a phone are built from a template and one JSON
object, never from hand-written HTML. A producer sends the template name and the
data; the Hub embeds the data and shows the page sealed above the answer bar.

| Template | Kind | Producer | Schema | Sample data |
|---|---|---|---|---|
| `evidence-decision` | Decision: one reusable image, audio, video, text, fields, UI or measurement comparison | `schemas/evidence-decision.schema.json` |
| `usage-mode` | decision | Agent Manager usage forecast | `schemas/usage-mode.schema.json` | `samples/usage-mode.json` |
| `daily-briefing` | status | Portfolio steward, daily | `schemas/daily-briefing.schema.json` | `samples/daily-briefing.json` |
| `opportunity-roadmap` | status | Portfolio steward, weekly | `schemas/opportunity-roadmap.schema.json` | `samples/opportunity-roadmap.json` |
| `structural-change` | decision | Portfolio steward, when friction recurs | `schemas/structural-change.schema.json` | `samples/structural-change.json` |
| `research-report` | status | Research supervisor, when a commission ends or pauses | `schemas/research-report.schema.json` | `samples/research-report.json` |

Every field is documented in its schema (`description` on each property). The
samples are illustrative and say so on the page (`meta.notice`); never send them
as real data.

## Send one

```bash
notification-hub conversations ask --kind status \
  --subject-key portfolio-steward:daily:2026-11-16 \
  --question "Today's briefing" \
  --template daily-briefing --data-file daily.json
```

Decisions (`usage-mode`, `structural-change`) also need the cover: `--question`,
the `--option key=label` values (the same keys as the data's `options[].key`),
and `--recommend`. To look at a page before sending, render it:

```bash
node kit/tools/render.mjs daily-briefing daily.json --out page.html   # validates first
node kit/tools/render.mjs daily-briefing --sample --out page.html     # with the sample
notification-hub pages check page.html
```

**Render contract** (the Hub does the same): the template has exactly one
`<script type="application/json" id="hub-data">`; its body is replaced with the
data JSON, with `<`, `>`, `&`, U+2028 and U+2029 escaped as `<` and so on
(Go `json.Marshal`'s default), so data can never close the element. The data must
validate against the schema, and the page must stay under 2 MiB.

## Honest data

- Each section has a `source: {status, label, note, asOf}`. `real` and `partial`
  show a badge; `unavailable` (or a missing section) replaces the chart with a
  plain notice naming what is missing. The page never draws an estimate in place
  of missing data.
- Leave a field out when it is unknown. Do not send zeros for unknown numbers.
- The only computed estimate is the roadmap's weight preview, which the page
  labels as an estimate and the steward replans for real.

## What the operator's choices send

Pages record choices through the Hub bridge; the Hub's answer bar holds Send,
Mark read, a note and images. Without the bridge (a file opened outside the Hub)
every page still renders and says that nothing is sent.

| Template | Answer keys (`hub.answer.set`) |
|---|---|
| `usage-mode` | `answer` = option key (also `hub.prefill("answer", key)`) |
| `daily-briefing` | `pace` = tier key (removed when back at the current tier); `steer` = steer key (removed for "As planned"). A steer's `weights` are what the producer applies. |
| `opportunity-roadmap` | `weight.<line id>` = weight key; `ask.<line id>` = `true` |
| `structural-change` | `answer` = option key; `checks` = list of `guardId`s to loosen, set only when it differs from the recommended set. Hard stops and checks without a `guardId` can never be selected. |
| `evidence-decision` | `answer` = selected option key (`hub.prefill`); evidence can be rendered with `window.HubCharts.evidence.render(el, records)` in other kit pages |
| `research-report` | `weight.<workstream id>` = selected weight key (default choices `hold`, `normal`, `more`); selecting the recorded current weight removes the answer. If the current weight is omitted or unknown, every choice starts unselected; the page says so, and **Clear choice** removes a pending answer. The operator's note comes from the Hub's answer bar. |

The research report's scorecard levels are the supervisor's judgment: every
dimension carries `level` and `reason` (shown when the operator taps the row).
Include both `was` and `now` on the same scale only when both numbers are
recorded. Omit both when numeric progress is unavailable; that row says so and
shows the judged level without drawing a progress bar. If any dimension lacks a
numeric pair, the report does not calculate an overall numeric average or
change. Workstreams are grouped by state, the groups that need the operator open
first.

`usage-mode` shows how often the question has come back from `hub.context.recurrence`
(the Hub's history for the subject key: `count` includes this ask, `dates` lists
every ask with this one last, `previousAnswers` are `{answer, note, answeredAt}`).
The data's own `recurrence` uses the same shape and is used only when the Hub has
no history.

The daily page switches live between precomputed contingency schedules:
`schedules` holds one schedule per `pace.tiers[].key`, each with the slot count of
its tier, its `blocks`, its `movedLater` list and, per steer, the schedule under
that steer.

## Components (`hub-charts.js`, `window.HubCharts`)

Charts measure their container and redraw on resize, so text stays 11 px or more
at phone width. Colours come only from Hub tokens (`--hub-*`) and the kit's own
tokens (`--kc-*`), with light and dark values; fonts are system stacks (the Hub
forbids web fonts).

| Function | Draws |
|---|---|
| `burnChart(el, spec)` | Usage readings, projections with range band, safe-pace line, reset and now markers, low-power spans, crosshair tooltip |
| `modeStrips(el, axisEl, spec)` | Full / low power / stopped strip per option |
| `dumbbell(el, spec)` | Planned against actual share per lane |
| `slotSchedule(el).update(spec)` | Gantt of agent slots; extra slots shaded; two stacked bands under 560 px |
| `roadmapMarks`, `refLines`, `hatchDefs` | Timeline rows with phases, gate ranges, no-plan hatching, reference lines |
| `slipChart(el, spec)` | A gate forecast at each review, with the proposed point |
| `checkMap(el, spec)` | Checks by hours lost (log) against harm, with a highlighted zone; locked checks are dashed |
| `funnel`, `proportion`, `shareBars`, `stackedBars`, `targetChart`, `sparkline` | Funnel, proportion bar with groups, share comparison, daily stacks, target pace, small trend |
| `stats`, `tiles`, `legend`, `pill`, `badge`, `seg` | Numbers, tiles, legends, pills, provenance badge, segmented switch |
| `answerCards(el, spec)`, `recurrence(el, spec)`, `answerNote(el, kind)` | Answer cards with "Choose this", recurrence block, answer status line |
| `measureCards(el, items)` | Measured results as before/after bar pairs coloured by verdict, with the change stated (`622× lower`) |
| `progressRows(el, spec)` | Scorecard dimensions on a labelled scale: where it was, gain or loss, target, judged level; tap for reason and today |
| `rangeBars(el, spec)` | Low-to-high estimate ranges on one axis, with a reference marker (e.g. this commission's hours) |
| `HubCharts.evidence.uiViewer(spec)` | UI screenshot comparison with side-by-side, slider and pixel difference modes; fidelity labels on each side (`captured`, `composited`, `rendered`, `sketch`) |
| `mountSource`, `unavailable`, `bindText` | Provenance badge per section, the unavailable notice, `data-text` binding |

## Change the kit

Edit `hub-charts.css` and `hub-charts.js` only; each template carries a copy so it
is self-contained. Then:

```bash
node kit/tools/sync.mjs            # copy the kit into every template
node --test kit/tools/kit.test.mjs # schemas, samples, render contract, page contract, browser behaviour
```

The browser tests run every template with its sample in headless Chrome (skipped
when Chrome is missing): no script errors, read-only without the Hub, pace switches
the schedule, hard stops stay locked, recurrence from the Hub, empty data shows
"unavailable". Run `notification-hub pages check` on a rendered page too.
