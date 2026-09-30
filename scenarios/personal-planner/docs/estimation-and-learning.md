# Learning From Planning — Estimation & Calibration System (design)

**Status:** design / not yet built. Seeds requirements already on the roadmap:
`P1-006 evidence-linked learning`, `P2-001 calibrated statistical forecasts`,
`P2-003 richer learning controls`.

**One-line thesis:** the planner should not only *record* what happened — it should help the user
**get better at planning itself**. Estimation accuracy becomes a first-class, visible, improvable
skill. This is the app's core differentiator in a saturated market: most planners are schedulers;
this one is a *calibration instrument*.

---

## 1. Why this matters

Everyone estimates badly and never finds out by how much, because the feedback is scattered and
implicit. The app already holds both halves of the signal (what you *planned* and what you *actually*
did) but never closes the loop. Closing it turns every finished task and goal into a small lesson,
and — because this is part of the Vrooli ecosystem — into **structured data an agent can read** when
it helps you plan future work.

Two design commitments, non-negotiable:

1. **Learning, not scolding.** Surface *positive* feedback as loudly as negative — "9 days early,"
   "15% more accurate this quarter," "you nail routines." Overruns are shown in **warm amber, never
   red**. The tone is a coach reviewing tape, not a manager flagging failure.
2. **No new tab.** History and learning live *inside the pages the user already visits* — an
   Active/All filter on Goals, a Calibration card on Review. The app has enough pages.

---

## 2. The core loop

```
   ESTIMATE            COMMIT            OBSERVE            RECONCILE           LEARN
 task minutes    →   place on the   →   focus timer /  →  actual vs planned →  bias + trend
 goal target         schedule           manual actual     delta computed       surfaced back
 milestone due                          + completion      + slip reasons       into estimates
```

Every arrow already partly exists; the gaps are in **OBSERVE→RECONCILE** (we don't reliably capture
completion or link plan↔actual) and **LEARN** (we compute nothing durable). See §4.

---

## 3. What we can build on today (data audit)

The scenario's SQLite-per-domain schema already has a **strong foundation** — this is mostly a
*connect and compute* job, not greenfield:

| Signal | Where it lives | State |
|---|---|---|
| Actual time + full correction audit trail | `api/internal/focus/schema.sql` (`manual_actuals`, `actual_corrections`) | ✅ richest data we have |
| Precise timer data (active vs wall seconds) | `focus/schema.sql` (`focus_sessions`) | ✅ |
| Planned blocks (date/start/duration/state) | `api/internal/calendar/schema.sql` (`calendar_allocations`) | ✅ |
| Reschedule (carry-forward) | `calendar/schema.sql` (`allocation_carry_forwards`) | ⚠️ last move only, no reason |
| Commitment deadline + revision history | `api/internal/commitments/schema.sql` (`commitment_revisions`) | ✅ good pattern to copy |
| Forecast snapshots (central/cautious finish) | `api/internal/forecasts/schema.sql` + `kernel.go` | ✅ durable history |
| Planned-vs-recorded compare (computed, not stored) | `api/internal/review/service.go` | ✅ compute exists |
| Goal / milestone progress + status | `api/internal/goals/schema.sql` | ✅ progress; ❌ no dates |

**The audit-trail pattern** in `actual_corrections` and `commitment_revisions` (previous value / new
value / reason / timestamp) is exactly the shape the new tables should follow. Reuse it.

---

## 4. What's missing (the gaps to fill)

Ordered by leverage. None of these is large; they're mostly *records we forget to keep*.

1. **Completion truth.** No timestamp/status when a work item, goal, or milestone is *done*.
   Inferring "done" from `remaining_minutes = 0` is fragile. → add explicit completion records.
2. **Original-estimate memory.** `work_items.remaining_minutes` is mutated in place, so the first
   estimate is lost — you can't compare "what I first thought" to "what it took." → capture estimate
   changes with a reason (mirror `actual_corrections`).
3. **Plan↔actual link.** `manual_actuals` has no FK to the allocation it fulfills, so "this 30-min
   actual belongs to that 60-min block" must be guessed by (work_item, date). → add an explicit link.
4. **Reschedule reasons + count.** `allocation_carry_forwards` keeps only the last move and no *why*.
   → append-only reschedule log with a `reason_code` (`interrupted` / `underestimated` / `blocked` /
   `deprioritized` / `external`).
5. **Goal/milestone target dates.** No `target_date` or `completed_date` on goals/milestones, so
   "9 days early / 3 weeks over" can't be computed. → add both.
6. **Durable learning aggregates.** Nothing stores bias or trend. → a small aggregate the review
   pass writes (avg error %, over/under ratio, per-category, sample size, accuracy-over-time).

### Minimal schema additions (follow existing per-domain placement)

```sql
-- goals/  — give goals & milestones real deadlines and completion truth
ALTER TABLE goals      ADD COLUMN target_date TEXT;     -- estimate
ALTER TABLE goals      ADD COLUMN completed_date TEXT;  -- actual
ALTER TABLE milestones ADD COLUMN completed_date TEXT;
ALTER TABLE milestones ADD COLUMN original_due_date TEXT; -- first commitment, for slip

-- work/  — remember the first estimate and every change (mirrors actual_corrections)
CREATE TABLE work_item_estimation_changes (
  id TEXT PRIMARY KEY, work_item_id TEXT NOT NULL,
  previous_minutes INTEGER, new_minutes INTEGER,
  reason_code TEXT, changed_at TEXT NOT NULL);

CREATE TABLE work_item_completion_history (
  id TEXT PRIMARY KEY, work_item_id TEXT NOT NULL,
  original_estimate_minutes INTEGER, final_actual_minutes INTEGER,
  variance_minutes INTEGER, created_date TEXT, completed_date TEXT);

-- calendar/  — why did it move? (append-only, keep all)
CREATE TABLE reschedule_history (
  id TEXT PRIMARY KEY, allocation_id TEXT NOT NULL,
  from_date TEXT, to_date TEXT, reason_code TEXT, rescheduled_at TEXT NOT NULL);

-- focus/  — bind an actual to the block it fulfilled
ALTER TABLE manual_actuals ADD COLUMN allocation_id TEXT;   -- FK to calendar_allocations

-- review/ (or a new learning/ domain)  — durable, queryable calibration
CREATE TABLE estimation_bias (
  period TEXT PRIMARY KEY,          -- 'all_time' | 'last_30d' | 'last_90d'
  avg_error_percent REAL,           -- +over / -under
  over_ratio REAL, under_ratio REAL,
  by_category_json TEXT,            -- {"deep_work":+0.30,"routine":+0.02,...}
  accuracy_trend_json TEXT,         -- weekly error series, for the sparkline
  sample_size INTEGER, updated_at TEXT NOT NULL);
```

Storage doctrine: keep each table **in the domain that owns it** (schema next to the code that
interprets it — the `SchemaProvider`/`EnsureSchemas` substrate), per the scenario's storage steer.
`estimation_bias` is a cross-domain *projection*; put it with review/learning and let it read the
others, don't scatter it.

---

## 5. The surfaces (where it shows up — no new tab)

### Goals — per-item variance (the detail view)
- Filter chips **Active · All · Completed**. `All`/`Completed` reveal finished goals.
- Each completed goal carries a **variance badge**: `▲ 9 days early` (green), `on time` (cyan),
  `3 wks over` (amber). One-line lesson beneath: *"You pad creative work"* / *"weather slips added
  ~40%."* (See the Goals page in `mockups/redesign-desktop-and-mobile.html`.)

### Review — calibration reading (the aggregate)
- A **Calibration card**: your headline bias (`+22% avg over-run`), where it's worst/best (*"you
  underestimate deep work ~30% but nail routines"*), and a **trend** (*"15% more accurate over 6
  weeks"*) drawn as a small descending error sparkline. This is the emotional payoff of the loop.

### Focus — honest capture at the source
- The **countdown timer** commits to the planned duration; when it hits 0 it **rolls into overtime**
  (counts up, amber) instead of stopping — so the overrun is captured truthfully rather than the
  user quietly resetting. This is the cleanest OBSERVE→RECONCILE feed.

### Everywhere friction happens — cheap reason capture
- When a task is rescheduled or a goal slips, offer a **one-tap reason chip** (never a mandatory
  form). Over time these `reason_code`s are what make the lessons specific.

---

## 6. Ecosystem angle (why this is more than a feature)

Because `estimation_bias` (and the underlying history) is **structured, queryable data**, any agent
in the ecosystem that helps the user plan — draft a project, size a goal, lay out a week — can read
it first: *"this user underestimates deep work by ~30%; pad accordingly."* The planner stops being a
private app and becomes a **calibration source for the whole system**. That is squarely the Vrooli
self-improving thesis: a scenario that makes future agents (and the user) measurably better at a
recurring judgment. Expose it via the API as a read model; do not make agents recompute it.

---

## 7. Suggested build order

1. **Completion truth + original-estimate memory** (§4.1, §4.2) — nothing works without these.
2. **Plan↔actual link + countdown/overtime timer** — closes OBSERVE→RECONCILE; timer is also a P0 UX
   win on its own.
3. **Goal target/completed dates + variance badges** — first visible learning, low effort.
4. **Calibration aggregate + Review card** — the payoff surface.
5. **Reschedule reason codes** — enriches lessons; can trail the rest.
6. **Agent read model** — expose `estimation_bias` over the API once it's populated.

Keep each step shippable and validated on its own; don't batch the schema changes into one migration.
