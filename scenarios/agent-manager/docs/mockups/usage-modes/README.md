# Usage modes: design mockups (phase 4 review)

Open `index.html` (self-contained, no dependencies). Sections A to F each show desktop and phone. All numbers are invented sample data and are labelled as such. Tokens are copied from `ui/src/styles/global.css`. The app has only a dark theme, so there is no light variant.

Screenshots: `A-1280.png` ... `F-1280.png` and `A-390.png` ... `F-390.png`, one per section.

| Section | What it shows |
|---|---|
| A | Mode sheet: preset, duration, scope, unproven effect range, reach, future-only default with a separate "also switch running sessions" choice and its re-read cost |
| B | Usage page banner: what, until when, burn since start, extend or end |
| C | Web Console chip, "full power for this launch", model a session will use and why |
| D | Evidence: conclusion first; break-even bar; shaded burn chart; cost ranges; plain-language quality rates; coverage; four states (not enough data, observational, randomized, no mode run) |
| E | Phone notifications: about to run out (decision), ending soon, until-off reminder, can't reach work, quality drop |
| F | Learn mode setup with an honest, ranged time to verdict |

## Decisions for the operator

1. **Preset-first sheet vs full form.** Alternative: show every control at once. Recommend preset-first; advanced scope is folded away.
2. **Sheet vs full page** for starting a mode. Recommend sheet for modes, full page for Learn mode.
3. **Effect shown as a range with "not measured"** vs a single "saves X%". Recommend the range; a point estimate would imply unearned certainty.
4. **Active runs.** Alternative: switch running sessions by default. Recommend leaving them alone, with the re-read cost shown on the opt-in.
5. **Break-even bar vs a table of ratios.** Recommend the bar plus the one-sentence form; keep the table for the CLI.
6. **Evidence strength label** as a pill beside the conclusion vs a footnote. Recommend the pill.
7. **Per-model series** on the same chart as the mode bands vs small multiples. Recommend one chart; switch if more than three models.
8. **Quality-drop notification** wording admits small samples. Alternative: stay silent until the signal is solid. Recommend off until the signal passes its threshold, as the plan says.
9. **Learn mode default split** 50/50 (fastest) vs 80/20. Recommend 50/50 with the cost stated.
10. **Launcher "full power"** as a second button vs a toggle. Recommend the button: single-use, cannot be left on by mistake.
11. **Sample counts** (about 20 outcomes to show a result, about 40 per side for Learn mode) were placeholders. Replaced on 2026-10-30 by the power simulation in `docs/reference/operational-stats.md` ("Learn mode"): 75 accepted outcomes per arm, a 30% effect, and a time to a verdict computed per role from its own pace and variance; the setup page reads it from the API instead of printing a fixed number.

## Not verified

Not seen on a physical phone or in a real iOS PWA. Not wired to data; copy was not through the i18n owner.
