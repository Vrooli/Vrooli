# Browser Automation Studio verification

How BAS changes are verified. The goal, queue and epochs live in the
[goal home](goal/GOAL.md); the epoch standard (brief, slice log, step-back,
acceptance) lives only in the `large-effort-orchestration` skill. The 24
preservation journeys are the requirements in
[`requirements/08-rehabilitation`](../../requirements/08-rehabilitation/module.json);
quality bands are in [TARGETS.md](goal/TARGETS.md).

## Journeys and the fixture site

Epoch gates are the preservation journeys an epoch touches, run against a local
fixture site so no external account or network is needed. Each journey asserts
given/when/then behavior through the product surfaces, tagged `[REQ:BAS-RH-Jnn]`.
Journeys that need native machines or external accounts (J09, J10, J12, J14) stay
unverified until a target exists (see [TARGETS.md](goal/TARGETS.md)).

Run the suite from `scenarios/browser-automation-studio/playwright-driver`
against the `bas-goal` shadow:

```bash
pnpm test:journeys --json --outputFile=<path>
```

It fails fast with "shadow unavailable" when the shadow is down, serves the
local fixture site for the duration of the run, and reports one result per
`tests/journeys/J*.test.ts`. There is no Test Genie journeys phase; this command
is the suite. J11 (AI navigation with a stubbed model) runs as a driver
integration test for now; its journey coverage is pending (QUEUE.md).

## Focused checks

- API: `cd api && go test ./<changed packages>/...`
- UI: `cd ui && pnpm type-check && npx vitest run <changed domain>`
- Driver: `cd playwright-driver && npx jest <changed area>`
- Net lines: `python3 docs/internal/refactor_inventory.py --no-git` before and
  after a work unit (never git).
- Scoped Test Genie phases, one at a time, blocking once on the run:
  `vrooli scenario test browser-automation-studio --phases <phase>` with
  `tidiness` or another phase the change touches.

Run changed BAS code on the goal's shadow, never by restarting the live BAS
mid-run. The shadow, daily qualification and unavailable targets are described
once, in [TARGETS.md](goal/TARGETS.md). Never weaken an assertion or count a
local diagnostic as proof.
