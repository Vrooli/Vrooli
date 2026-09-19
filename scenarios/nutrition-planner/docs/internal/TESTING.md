# Testing — Nutrition Planner

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.
- [Flows](../concepts/FLOWS.md): the journeys and lifecycle state machines these
  tests must exercise.

Test the **desired/expected behavior** from the canonical specification
([`../reference/product-specification.md`](../reference/product-specification.md) §21),
not the current scaffold implementation. Verify risky behavior at the lowest
useful layer, then exercise complete user journeys through the real UI and
persistence boundary.

## Scenario-specific testing

### What must be proven

The spec's acceptance matrix (ACT-001–ACT-063) is the source of truth for
release-gated behavior; each row is tagged to a release (R0/R1/R2/R3) and a
requirement area. The highest-value test families:

- **Exact arithmetic and units (R1).** FIX-02 exact protein/energy/cost totals at
  recipe and serving scale; unit changes never alter underlying quantities;
  FIX-04 distinguishes unknown, zero, partial lower-bound pass, and unresolved
  upper bounds. Property tests (§21.3): scaling a linear recipe scales
  contributions; splitting/recombining an ingredient across steps preserves
  totals; supported unit conversions round-trip within a declared precision
  policy.
- **Required constraints (R0).** FIX-06: required exclusions/equipment cannot be
  overcome by any preference weight; no-candidate outcomes name blockers and
  offer repair without silently relaxing rules.
- **Planning determinism (R1).** Same input snapshot, seed, and algorithm version
  produce a reproducible result; FIX-10 scores match; a search timeout is
  distinguished from proven infeasibility; partial plans never claim all targets
  met.
- **Inventory and batch accounting (R1).** FIX-03: preparation consumes raw
  ingredients once and creates a batch; later portions consume the batch only;
  replanning excludes its own replaced reservations; intake undo/correction
  restores stock exactly once.
- **Persistence, concurrency, recovery (R0).** FIX-01 name-only draft persists
  with unknown values preserved; ACT-034/035 saved edits survive refresh and
  concurrent writes cannot silently overwrite; FIX-07 historical snapshots
  survive recipe edits and idempotency-key reuse behaves as specified.
- **Portability and documents (R0).** FIX-08 legacy migration limitations;
  FIX-09 native `daily.recipes`/`daily.workspace` semantic round-trip; real PDF
  bytes inspected for clipping, fonts, page breaks, and A4/Letter; CSV escaping
  and formula-neutralization.
- **Accessibility (R0).** Main flows at mobile and desktop widths with keyboard
  navigation, visible focus, contained/restored dialog focus, and non-color
  status cues.
- **Providers and AI (R2).** Ordinary suites never depend on live provider calls;
  adapters are contract-tested against recorded responses and separately
  smoke-tested when configured. Malicious page instructions cannot mutate
  settings or trigger actions; prohibited fetch destinations are blocked;
  malformed model output stays a proposal/error and never overwrites newer edits.

### Fixtures

Use the embedded synthetic fixtures from spec §20 (FIX-01 through FIX-11) and the
starter collection in §20.11. All numbers and product identities there are
synthetic test data — never a recommended diet, verified price, or a personal
target. A fresh real workspace must not inherit sample pantry stock, the
fictional user's targets, or any demo totals.

### End-to-end review scripts

The spec's §21.4 scripts A–H (cold start to a real week; incompatible settings;
tired-evening swap; batch and actual use; data portability; document quality;
access and failure; low-maintenance assistance) are the browser-level acceptance
scripts. Run them through the real persistence boundary, not mocked stores.

### Where tests live and how to run them

- Unit/integration: `api/internal/<domain>/..._test.go`, `cli/..._test.go`,
  `ui/src/**/*.test.tsx`.
- Scenario suites: `vrooli scenario test nutrition-planner --phases <phases>`
  (focused) or the `test-genie.iterate` program. The run is server-owned;
  block once with `test-genie runs wait` rather than polling.
- Tag requirement-linked tests with `[REQ:<ID>]` so requirements sync can update
  status.

## Binary startup

`api/main_e2e_test.go` uses `api-core/boottest` to build this API, verify its
health identity in isolated storage, and check shutdown. Keep the expected
service name in the scenario test. Shared process machinery and failure
regressions live in `packages/api-core/boottest`; see its package README section
for configuration and evidence limits. The existing E2E gate runs this test.
