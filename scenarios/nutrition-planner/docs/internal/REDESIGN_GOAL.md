# Redesign goal — Nooch

The goal message that drives the v2.0 redesign to done. It was authored
with the `harness-goal-authoring` skill as a **convergence goal** (§6 of that
skill, shape C: no plan, tracked in files). It carries the finish line, the
proof, the dials, and the stop rules; the design itself lives in the documents it
points at, so the goal stays short and never goes stale when the design is
refined.

## How to use

- **Claude Code or Codex session:** type `/goal ` followed by the main message
  below (3,877 characters with the prefix; Claude Code's cap is 4,000).
- **Agent Manager run or Swarm item:** pass the compact variant as `--until`
  (1,373 characters; the cap is 2,048). It points at the same documents.
- Before starting, make sure the documents it names exist and are current:
  [`REDESIGN_PLAN.md`](REDESIGN_PLAN.md), [`REDESIGN_LEDGER.md`](REDESIGN_LEDGER.md),
  [`OPERATOR_FEEDBACK.md`](OPERATOR_FEEDBACK.md),
  [`../reference/mockups/README.md`](../reference/mockups/README.md),
  [`../../DESIGN.md`](../../DESIGN.md), and
  [`../reference/product-specification.md`](../reference/product-specification.md).
- While it runs, give feedback normally. The goal instructs the agent to copy each
  item verbatim into [`OPERATOR_FEEDBACK.md`](OPERATOR_FEEDBACK.md) before acting.
  If a design decision changes, change the documents, not the goal text.

## Main message

```text
Work until this verified end state is true: Nooch (scenarios/nutrition-planner, called S; the files named below are under S) meets each item of the definition of done in S/docs/internal/REDESIGN_PLAN.md §1 — every surface works end to end on real persisted data in the running scenario and, captured at the R27.5 viewports in Light and Evening, matches its mockup in S/docs/reference/mockups/ — and two consecutive fresh, hostile review passes find nothing material.

Read first, in order: docs/internal/OPERATOR_FEEDBACK.md, docs/internal/REDESIGN_LEDGER.md, docs/internal/REDESIGN_PLAN.md, docs/reference/mockups/README.md, DESIGN.md, docs/reference/product-specification.md (R01–R30, Appendix A); then prompt-manager skill read experiential-ui-design harness-goal-authoring. The docs hold the design; do not re-derive it.

Work the plan's build spine, foundation first (D0: the 401 auth defect, persisted plans, migrations, transactions), one complete vertical slice at a time: proto, Go domain, persistence, UI, tests, docs. Mature, maintainable code only: shared components, tokens only, per-medium composition, no dead or test-only production code, no shims. Leave the code simpler and lower in debt than you found it. Generate the curated artwork with image-tools per plan §7 and integrate in-app generation the same way; that spend is approved up to a hard US$15 total (D-043): do not pause below it, and ask before exceeding it.

Proof, shown in the transcript: go test ./... in scenarios/nutrition-planner/api; pnpm -C scenarios/nutrition-planner/ui type-check and test; vrooli scenario test nutrition-planner --phases <affected>, blocking once on test-genie runs wait --json; vrooli scenario requirements validate nutrition-planner --json; experience-manager spec validate nutrition-planner --json; requirement statuses earned by sync after re-enabling each fully referenced module (D-037); real CLI/API calls against the running scenario; per-surface capture verdicts against the mockups in the ledger; and, last, a comprehensive vrooli scenario test nutrition-planner with every phase passing. Never edit a test, band, baseline, or claim to pass.

Convergence: a green suite starts a fresh hostile pass; it never ends the work. A fresh pass re-captures every surface and re-runs the checks on the current code. Each pass judges correctness, R26 edge cases, code maturity and cleanliness, accessibility, performance, and visual fidelity to the mockups, as a skeptic who wants to reject the work. You are the sole and final reviewer; anything you leave ships broken. Record every finding and resolution in the ledger. Over-engineering and new debt are findings.

Operator feedback: append each item verbatim to OPERATOR_FEEDBACK.md the moment it arrives, before acting; re-read it every pass; you are not done while any item is open.

Do not use Plan Manager; the plan and ledger are the tracking system. No git. Dependencies only through scenario-dependency-analyzer. Adjacent defects in image-tools, ai-gateway, personal-planner, or the component library: fix at the owner when understood and blocking; otherwise file with the report-bug skill and continue.

Blocked means a decision, credential, or approval you lack. Record the exact ask in the ledger and keep working; blocked items never hold the rest open. Friction you can diagnose is not blocked, and a list of remaining work is your next instruction, not a blocker.

Checkpoint to the ledger after every slice and pass (changed / verified / remaining / unverified); at done, write the R30 report there. There is no turn limit; any interrupted or new session resumes from the ledger.

Stakes: this app has never worked for its user — every request fails and the database is empty. They want a beautiful, honest meal planner they will open every day; until it is real, the whole specification is inert.
```

## Compact variant

```text
Work until this verified end state is true: Nooch (scenarios/nutrition-planner) meets every item of the definition of done in scenarios/nutrition-planner/docs/internal/REDESIGN_PLAN.md §1, and two consecutive fresh hostile review passes find nothing material. Read first (under scenarios/nutrition-planner/): docs/internal/OPERATOR_FEEDBACK.md, docs/internal/REDESIGN_LEDGER.md, docs/internal/REDESIGN_PLAN.md, docs/reference/mockups/README.md, DESIGN.md. Build the plan's spine in complete vertical slices, D0 foundation first; mature, maintainable code only; generate artwork with image-tools per plan §7 (approved up to US$15 total, D-043). Proof in the transcript: go test, pnpm type-check and test, scoped vrooli scenario test with one blocking test-genie runs wait, the requirements and experience validators, earned requirement statuses (D-037), real API calls, capture verdicts against the mockups in the ledger, and a final comprehensive run with every phase passing. A green suite starts a new hostile pass; record findings and resolutions in the ledger; you are the final reviewer. Append operator feedback verbatim to OPERATOR_FEEDBACK.md before acting; not done while any is open. No Plan Manager, no git, never weaken a test. Blocked means a missing decision, credential, or approval: name it and continue elsewhere. Checkpoint to the ledger after each slice.
```

## Slot map

How the main message covers the `harness-goal-authoring` slots, for whoever
revises it.

| Slot | Where it is |
| --- | --- |
| Destination | First paragraph: a directive toward a future state, defined by the plan's §1 definition of done plus the convergence condition. |
| Proof | "Proof, shown in the transcript" — named commands, earned requirement statuses, the ledger's capture verdicts, and a final comprehensive run. |
| Sources | "Read first, in order" — ledgers first, so feedback and open findings frame the session. |
| Boundary | The scenario, its protos and generated clients, and owner fixes in named dependencies when understood and blocking; everything else is filed. |
| Dials | Validation: targeted, blocking once on runs. Adjacent defects: fix at owner or file. Quality: mature code, no shims, less debt. |
| Blocked | The "Blocked means …" paragraph: record the exact ask, keep working, blocked items never hold the rest open, and remaining work is not a blocker. |
| Budget | No turn limit by operator intent; interrupted sessions resume from the ledger. |
| Handoff | Ledger checkpoints (changed / verified / remaining / unverified) and the R30 report at done. |
| Non-goals | No Plan Manager, no git, no weakened tests, no dependency installs outside the analyzer. |
| Convergence (§6) | Green starts a fresh pass (re-capture and re-run on current code); two consecutive empty hostile passes; sole and final reviewer; over-engineering and new debt are findings; operator feedback is a durable gating input; stakes stated truthfully. |

## Why these choices

- **No Plan Manager** — operator direction (D-026). A convergence loop has no
  phase order; the ledger and the plan's build spine carry the state.
- **Foundation first** — the 2026-09-22 audit found every RPC returning 401 and an
  empty database ([`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §3). Surfaces built on
  that foundation would look finished and prove nothing.
- **Artwork authorized with a cap** — operator direction (D-029, D-043):
  bounded by the plan's inventory and a hard US$15 total, with every cost
  recorded in the asset manifest and the ledger.
- **No downstream reviewer named** — the goal makes the agent the final reviewer
  on purpose; a named safety net makes agents stop early (`harness-goal-authoring` §6).
