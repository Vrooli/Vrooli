# Personal Planner — Feature Backlog & Gaps

Working list of missing features, UX gaps, and design debt for the personal-planner scenario.
Point an improvement agent here. Items are grouped by theme and tagged with a rough priority
(**P0** = changes how the app fundamentally feels/works · **P1** = strong lift · **P2** = polish).

**Companion docs & artifacts (read alongside this):**
- `docs/mockups/redesign-desktop-and-mobile.html` — desktop + mobile redesign mockups for all 6 pages.
- `docs/mockups/today-mobile-nextstep.html` — standalone interactive Today/next-step + sheet.
- `docs/estimation-and-learning.md` — full design of the "learning from planning" system (§ referenced below).
- `docs/mockups/image-generation-brief.md` — ChatGPT-ready brief for generating per-page observatory art.

> Existing roadmap requirements that overlap live in `requirements/02-post-launch/` and
> `requirements/03-future/` (e.g. `P1-006 evidence-linked learning`, `P2-001 calibrated forecasts`).
> Reconcile with those rather than duplicating.

---

## A. Frictionless capture & steering  *(highest leverage — makes it delightful vs. tedious)*

- [ ] **P0 · Natural-language capture.** A single box that parses *"lunch with Sam tue 1pm 1h"* into
  a placed item (title, time, duration, source). The one feature that most separates a delightful
  planner from a chore.
- [ ] **P0 · Global quick-capture.** ⌘K command palette on desktop (from any page); the FAB on
  mobile. Capture must be reachable everywhere, always in the thumb zone on mobile. *(The buried
  "add task" in the first Today mockup was the anti-pattern this fixes.)*
- [ ] **P1 · Snooze / defer / "not now"** on the next step and any scheduled item — gracefully push,
  don't just start-or-ignore. Feeds reschedule-reason capture (§D).
- [ ] **P1 · Reminders / notifications.** A planner that can't nudge is a diary. Target-reached
  chime for focus, upcoming-block and overdue nudges.

## B. Focus timer  *(anchor example — partly mocked up already)*

- [ ] **P0 · Countdown mode**, first-class, pre-filled from the task's planned duration. Toggle with
  Stopwatch. *(In mockup.)*
- [ ] **P0 · Overtime roll-over.** At 0 the countdown flips to counting *up* in amber instead of
  stopping — commit to a target AND capture the honest overrun. Directly feeds the learning loop
  (`estimation-and-learning.md` §5). *(In mockup.)*
- [ ] **P1 · Quick-pick durations** (25 / 45 / 90) and Pomodoro cycles with break prompts. *(Picks in mockup.)*
- [ ] **P1 · End-of-session note** ("what did you get done?") flowing into Review.
- [ ] **P2 · Distraction / pause reason** one-tap logging.

## C. Learning from planning (estimation & calibration)  *(the differentiator)*

See `docs/estimation-and-learning.md` for the full design, data audit, and schema. Summary tasks:

- [ ] **P1 · Completion truth** — timestamps/status on work items, goals, milestones (foundational).
- [ ] **P1 · Original-estimate memory** — capture estimate changes with reason (mirror `actual_corrections`).
- [ ] **P1 · Plan↔actual link** — FK from `manual_actuals` to the allocation it fulfilled.
- [ ] **P1 · Goal/milestone target + completed dates** — enables "9 days early / 3 weeks over."
- [ ] **P1 · Variance badges on completed goals** — Goals `Active/All/Completed` filter; green early,
  amber (never red) over; one-line lesson. Positive AND negative framing. *(In mockup.)*
- [ ] **P1 · Calibration card on Review** — aggregate bias ("underestimate deep work ~30%") + accuracy
  trend. The learning payoff surface, no new tab. *(In mockup.)*
- [ ] **P2 · Agent read model** — expose `estimation_bias` over the API so ecosystem agents planning
  for the user can read their calibration first.

## D. Per-page functional gaps

**Plan**
- [ ] **P1 · Conflict / overbook warnings** when placement exceeds capacity or collides.
- [ ] **P1 · "Protect breathing room"** as an enforced guardrail, not just a displayed number.
- [ ] **P2 · Energy-aware suggestions** (deep work → mornings).
- [ ] **P2 · Reschedule reason codes** (`interrupted/underestimated/blocked/deprioritized/external`) —
  one-tap, feeds §C.

**Goals**
- [ ] **P1 · Link tasks → milestones** so doing the work auto-advances the goal (progress looks manual today).
- [ ] **P1 · Drift nudge** when a goal falls behind pace.

**Review**
- [ ] **P1 · Guided reflection prompts** (2–3 questions) instead of a blank textarea.
- [ ] **P2 · Wins / gratitude** line — reinforce the positive-feedback commitment.
- [ ] **P2 · Trend sparklines** over weeks (active time, accuracy).

**Today**
- [ ] **P1 · Overdue awareness** — surface what slipped, not just what's next.
- [ ] **P2 · Streak / momentum** signal.

## E. Mobile adaptation  *(architecture, not just CSS)*

Current responsiveness is CSS-reflow only (media queries + `clamp()`), with a frozen component tree —
"responsive but doesn't change enough." Lift to conditional composition.

- [ ] **P0 · Breakpoint hook** (`useBreakpoint`/`useIsMobile`, matchMedia, SSR-safe). Check whether the
  react-component-library already exports one before writing a new one. Enables everything below.
- [ ] **P1 · Today** → timeline owns the fold; next-step becomes a thumb-zone bar + bottom sheet;
  capacity moves to Plan; capture FAB. *(In mockup.)*
- [ ] **P1 · Plan** → collapse 8 view modes to Day/Week + capacity ring + schedule; placement primary,
  rest behind `⋯` sheets. *(Worst desktop→mobile offender. In mockup.)*
- [ ] **P1 · Focus** → full-screen distraction-free timer, no chrome, controls in thumb zone. *(In mockup.)*
- [ ] **P1 · Review** → week **table becomes a card list** (never horizontal scroll). *(In mockup.)*
- [ ] **P2 · Goals** → single-column + FAB (correct *restraint* — a list is already mobile-native). *(In mockup.)*
- [ ] Use **container queries** for genuinely layout-only cases; reserve the JS hook for interaction-model changes.

## F. Settings redesign  *(the one genuine rebuild — "functional but generic")*

- [ ] **P1 · Reframe as "Calibrate your instrument."** Theme = live sky previews (not radio buttons),
  availability = "observatory hours," integrations = "feeds." Same card language/tokens as other pages.
- [ ] **P1 · Mobile drill-in pattern** — summary rows that open sub-screens (iOS Settings style),
  instead of one long wall of stacked form fields. *(In mockup.)*

## G. Visual identity & imagery  *(the market differentiator)*

- [ ] **P0 (concept) · Celestial-navigation throughline across ALL pages.** Today already speaks it;
  extend the world: Plan = star chart, Goals = constellations, Focus = through the scope,
  Review = star trails, Settings = the instrument. Coherent world > prettier cards.
- [ ] **P1 · Procedural visual layer everywhere** — star field, constellation lines, glow, day/night
  wash drawn in code/SVG/canvas (not generated images): crisp, animatable, theme-aware, tiny.
  This is ~90% of the "beautiful" feeling and is cheap to spread. **Do this before generating art.**
- [x] **P0 · Compositing contract L0/L1/L2 + Today seam (DONE).** Generated art is foreground ONLY;
  sky is always procedural. Today now **derives** the sky seam colour + band aspect from the image at
  runtime (sampled top strip) and feathers the band into it — no hard-coded colours, robust to image
  changes. Mechanism: `docs/visual-compositing.md`. Files: `ui/src/theme/observatoryAppearance.ts`
  (`sampleSceneAsset`/`useSampledSceneColors`), `ui/src/pages/DashboardPage.tsx`, `ui/src/styles.css`.
- [ ] **P1 · Generalize compositing to all pages** — factor the sampled-seam sky into a shared
  hook/component; apply Shape A (exterior band) / Shape B (interior aperture) per page per
  `mockups/image-generation-brief.md` §3.
- [ ] **P1 · Ambient sky FX (procedural, sparse + slow)** — on the procedural sky layer, confined to
  keyed sky regions, gated by appearance: **comets/shooting stars at night**, **occasional hot-air
  balloons by day**, subtle parallax. Works on every page that exposes procedural sky. Keep it calm —
  one balloon now and then, not a fleet.
- [ ] **P1 · Plan window diorama** — nested composite: star-chart desk plate (keyed window) → lake
  landscape view → procedural sky. Day: opaque landscape sky + balloons in the window. Night: keyed
  upper-sky band reveals procedural stars/comets. Assets in `mockups/source-art/`. Only the interior
  desk plate needs two sizes (desktop+mobile, both present); the through-window landscape needs one
  size (crop/pan within the window). Full layer stack in `visual-compositing.md` "Worked example —
  the Plan page diorama".
- [ ] **P1 · Focus "through the scope"** — Shape B interior, desktop + mobile plates captured
  (`mockups/source-art/focus-scope-{desktop,mobile}.greenscreen.png`). Composite the focus
  timer/ring + a procedural focus-star (and comets) inside the keyed circular aperture; procedural sky
  behind. One dim mood (day/night via the aperture). Engineering pending — reuse the generalized
  Shape-B compositor.
- [ ] **P1 · Settings "the instrument"** — Shape B interior, desktop + mobile plates captured
  (`mockups/source-art/settings-instrument-{desktop,mobile}.greenscreen.png`). Brass mechanism is the
  backdrop; arched green window → procedural sky. Engineering pending — reuse the Shape-B compositor.
- **Art status: COMPLETE.** Today ✅ (live) · Plan ✅ captured · Focus ✅ captured · Settings ✅
  captured · Goals/Review = procedural (no new art). All remaining work is engineering.
- [ ] **P2 · Per-page hero illustrations** — one observatory/telescope motif, ~6 framings, Night first
  then Day variants. Generate via `docs/mockups/image-generation-brief.md`. Glow is added in-app
  (never baked into the illustration — it doesn't survive well).

---

## Notes / cautions carried from prior work
- Presentation colors are shared light/dark; the visual gate is the scenario's `visual-check` flow.
- Generated hero art must come from an illustration model with the sibling brand as style ref;
  vector-native marks read as primitive; glow never survives tracing → keep glow procedural.
- Validate each change with scoped Test Genie phases; don't batch schema migrations.
