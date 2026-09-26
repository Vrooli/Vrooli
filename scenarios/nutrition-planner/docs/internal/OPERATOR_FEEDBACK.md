# Operator feedback ledger — Nooch redesign

Durable record of every instruction and piece of feedback the operator gives
while the redesign goal runs. Session context is compacted and forgotten; this
file is not. It is a gating input to the convergence loop in
[`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §9 (`harness-goal-authoring` §6).

## Rules

1. **Capture before acting.** The moment the operator says something about this
   work, append it here **verbatim** as a new entry (split distinct asks into
   separate entries), then act on it.
2. **Re-read every pass.** Read this file at the start of every build slice and
   every review pass.
3. **Resolve with a receipt.** An entry closes only with evidence: what changed,
   where, and the command output or capture that proves it. Never edit or delete
   the operator's words.
4. **Statuses:** `open` (not yet satisfied), `resolved` (satisfied, receipt
   recorded), `standing` (a rule that applies for the whole effort; verify it every
   pass and record any violation as a new open entry).
5. **The goal is not done while any entry is `open`.**

## Entries

### OF-001 — 2026-09-22 — The quality bar

> "It should focus on mature, maintainable code, and a ux that's professional, responsive, and as close to the reference images as possible"

- **Source:** the operator's request that commissioned this documentation update
  and the redesign goal.
- **Status:** `standing` — verified in every review pass through the engineering
  bar ([`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §4) and the per-surface capture
  verdicts against [`../reference/mockups/`](../reference/mockups/README.md).

### OF-002 — 2026-09-22 — Keep going until it is perfect

> "This should be the type of goal with the adversarial checks that keeps going until everything is absolutely perfect."

- **Source:** same request.
- **Status:** `standing` — implemented as the convergence protocol: done only
  after two consecutive fresh hostile passes with zero material findings
  ([`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §9).

### OF-003 — 2026-09-22 — Generate the artwork and integrate the capability

> "We have tools in this project for you to generate them now, and for you to integrate the capability properly. Not sure if you connect to image-tools or ai-gateway, but it exists"

- **Source:** the operator's answer when asked how production artwork should be
  sourced.
- **Status:** `open` — resolved when every raster asset in the curated inventory
  in [`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §7.1 exists as a reviewed,
  manifest-tracked file generated through image-tools (rows marked code-native
  SVG are drawn in code), **and** the app's own
  generation workflow (R18) runs through image-tools/ai-gateway with the
  budget, review, and fallback behaviour the specification requires. Decision
  D-029.

### OF-004 — 2026-09-22 — No Plan Manager

> "Archive it. Note that this new approach should NOT use plan-manager"

- **Source:** the operator's answer about the earlier plan
  `implement-daily-nutrition-planner-to-production-ready-r0-r1`.
- **Status:** `standing`.
- **Receipt (archive):** 2026-09-22 `plan-manager plans archive
  63ef6942-e987-4659-9a1d-6ff98b1ab038` returned `PLAN_STATUS_ARCHIVED`
  (decision D-026). Verify every pass that no plan, execution, or plan-manager
  log was created for this effort.

### OF-005 — 2026-09-22 — Working name Nooch

> "And let's brainstorm actual names for this. The business apps all have a common theme. The personal apps do not have s theme yet, and not sure if they should. Perhaps we can just pick a name like Nooch for now, since I like nutritional yeast and it sounds neat"

- **Source:** the operator, after the documentation update; confirmed by choosing
  "Nooch" when asked.
- **Status:** `standing` — the display name is Nooch (D-042). Verify every pass
  that the UI renders the configured display name (wordmark, titles,
  `service.json`) and that no new hard-coded "Daily" product label appears.

### OF-006 — 2026-09-22 — Spending cap

> "The spending cap makes sense."

- **Source:** the operator, then choosing "$15 total" when asked for the amount.
- **Status:** `standing` — hard cap of US$15 on image generation for this effort
  (D-043). Verify every pass that the ledger's running total is current and
  below the cap; generation that would exceed it waits for the operator.
