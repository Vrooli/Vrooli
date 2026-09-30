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
([`../reference/product-specification.md`](../reference/product-specification.md) —
redesign R27 and Appendix A §21), not the current implementation. Verify risky behavior at the lowest
useful layer, then exercise complete user journeys through the real UI and
persistence boundary.

## Scenario-specific testing

### Current reality (audit 2026-09-22)

Validation today proves almost nothing about product behavior. Read this before
trusting any green result (details in [`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §3):

- **Requirements:** all 161 are `planned` (the audit found 83 of the then 84
  `planned`); no validation entry references a real test, and no test in `api/`,
  `cli/`, or `ui/src` carries a `[REQ:<ID>]` tag. A comprehensive run's sync on
  2026-09-22 nonetheless promoted 93 requirements from phase-level passes; they
  were reverted and auto-sync is off (D-037, F-015).
- **API:** 57 Go test files cover `internal/*` packages against in-memory SQLite;
  **no Connect handler outside `health` and `capabilities` has a test**, so
  authorization, revision checks, idempotency, and transactions are unproven at
  the boundary. `api/coverage.out` (2026-09-18) reports **48.8 %** statement
  coverage against the **75 %** policy in `.vrooli/testing.json`.
- **UI:** 36 test files mock the API modules; none exercises a running API. The
  reported ~94 % line coverage is inflated because each page is a few very long
  JSX lines.
- **BAS:** the three `bas/cases/r0/*` cases navigate and assert that a page
  container is visible — they pass on the `401 unauthenticated` error screen.
  `bas/cases/routed-database/proves-test-pool-routing.json` is a scaffold that
  cannot pass, and `bas/cases/experience-spec/dashboard.json` targets a retired
  page. Replace them with journeys that assert persisted outcomes.
- **Experience contract:** none of the bound `data-testid`s in `experience/`
  existed in the UI at the audit, so machine-tier claims cannot run yet.
- `.vrooli/testing.json` still carries the template's "Sample" description, and
  `.vrooli/lighthouse.json` gates only the template `/` page.

### Verification commands

Choose scope with repository `docs/TESTING.md`: targeted checks by default,
heavier runs when a milestone or certification needs them.

| Check | Command | Passes when |
| --- | --- | --- |
| Go domain and handler tests | `cd scenarios/nutrition-planner/api && go test ./...` | All pass; fixture arithmetic is exact; handler tests cover auth, revisions, idempotency, and transactions. |
| UI types and tests | `pnpm -C scenarios/nutrition-planner/ui type-check` and `pnpm -C scenarios/nutrition-planner/ui test` | Zero type errors; interaction semantics are tested. |
| Scoped Test Genie phases | `vrooli scenario test nutrition-planner --phases <phases>`, then block once on `test-genie runs wait --json nutrition-planner <run-id>` | The named phases pass for the right reasons (structure, unit, lint, business, experience, ui-health, performance, docs, storage, contracts, security as relevant). |
| Requirements | `vrooli scenario requirements validate nutrition-planner --json` (structure only; it stamps `last_validated_at`) plus the sync that follows a Test Genie run | Statuses are earned by `[REQ:ID]` sync; nothing is hand-set. Auto-sync is off until every validation in a module carries a `ref` (D-037). |
| Contract | `business-health validate scenario nutrition-planner` | Clean. |
| Experience contract | `experience-manager spec validate nutrition-planner --json` | Clean; bound test ids exist; machine claims pass in the experience phase. |
| Documentation | `knowledge-observatory docs audit nutrition-planner --json` | No new findings in authored docs. |
| Runtime | `vrooli scenario status nutrition-planner` plus real CLI and API calls | Healthy, and a real workspace flow succeeds (not a 401). |

**Wait protocol.** A Test Genie run is server-owned and survives a cancelled
client. Start it, then block **once** with
`test-genie runs wait --json nutrition-planner <run-id>`; never poll status in a
loop. Cancelling your client is not an abort — use
`vrooli scenario test abort …` to stop a run. Never rerun unchanged validation to
get a greener result, and never edit a band, baseline, test, or claim to make a
check pass.

### Requirement tagging

Put `[REQ:<ID>]` in the test name or a leading comment of every test that proves a
requirement (for example `func TestShopping_CheckDoesNotPurchase_REQ_GROC_005`
with `// [REQ:GROC-005]`, or `it("[REQ:COOK-002] Next never completes a step", …)`).
Point the requirement's validation `ref` at that test. Requirement status then
comes from requirements sync after a qualifying run — but auto-sync is currently
off in every module (D-037) because Test Genie promotes any validation without a
`ref` from its phase's overall pass. Re-enable a module's `auto_sync_enabled`
only when every validation in it carries a `ref` or an attested manual record. A
hand-edited `"status": "complete"` is a defect.

### Redesign acceptance mapping (R27)

| Area | Acceptance cases | Fixtures | Lowest useful layer |
| --- | --- | --- | --- |
| Shell, appearance, empty accounts | AT-001–AT-003, AT-052 | Fresh workspace; appearance stored locally and in the account | UI tests + BAS journey with reload |
| Today media and context | AT-004–AT-008 | R27.3 artwork fixtures (scene, ordinary photo, no photo, incompatible revision, rejected, load failure, no evening asset) | Pure chooser tests in Go; UI captures per treatment |
| Week and planning | AT-009–AT-015 | FIX-04, FIX-06, FIX-10; DST/local-midnight fixtures | Go planner and scope tests; UI agenda at 390 px |
| Meals, Explore, authoring | AT-016–AT-023 | FIX-01, FIX-06, R27.2 recipe fixture | Handler tests for idempotent save and add-to-week; UI for context banner |
| Recipe and cooking | AT-024–AT-030 | R27.2 four-step fixture, FIX-03, FIX-05, FIX-07 | Go graph/scaling; UI timer tests with fake clocks; two-tab expiry in the browser |
| Groceries and kitchen | AT-031–AT-039 | FIX-03, R27.3 equipment fixtures | Go shopping diff and ledger; UI Review/Shop; offline replay in the browser |
| Generation and budgets | AT-040–AT-043 | Recorded image-tools responses | Go reservation concurrency test; real image-tools run when configured |
| Access, calendar, portability | AT-044–AT-051 | FIX-08, FIX-09 | Handler auth tests; calendar adapter fake plus a real run when configured; PDF bytes inspected |

AT-040–AT-043 and AT-045–AT-046 need real configured integrations for end-to-end
evidence. Deterministic adapter tests alone must be labelled as such; if access
is absent, record the case as blocked with manual fallback evidence rather than
passing it (R27.4).

### Visual acceptance (R27.5)

Capture every primary surface plus Explore, recipe detail, recipe map, focused
cooking, inventory, equipment, Preferences, and one long editor at **360×800,
390×844, 768×1024, 1024×768, and 1440×1000** CSS px, plus **320 px width** and
**200 % zoom**, in **Light and Evening**, with real fonts and assets. Include the
nonideal media states (ordinary photo, no photo, failed load). Put each capture
beside its mockup in [`../reference/mockups/`](../reference/mockups/README.md)
and judge hierarchy, composition, spacing, type, image treatment, and control
language — not pixels. Record per-surface verdicts and residual gaps in
[`REDESIGN_LEDGER.md`](REDESIGN_LEDGER.md). A screenshot proves appearance only;
it never proves persistence, arithmetic, eligibility, spending controls, or
integrations (R30).

### Journeys assert outcomes

A BAS journey passes only when it verifies the **persisted effect** of each step
(a saved meal reappears after reload; a checked row stays checked on a second
device; a timer resumes at the right remaining time). Asserting that a page
container is visible is not a journey. The R27.6 integrated review journey is the
top-level browser acceptance: clean account → setup → name-only meal → enriched
recipe → map/reading consistency → week draft → lock → Explore add → leftovers →
groceries → offline partial shopping → reconnect → purchases → cooking with a
timer → batch and one consumed serving → full-day nutrition with unknowns →
recipe revision → export and restore.

### What must be proven

The redesign acceptance cases (AT-001–AT-052, R27.4) and the retained
baseline matrix (ACT-001–ACT-063, Appendix A §21.2) are the source of truth for
release-gated behavior; each ACT row is tagged to a release (R0/R1/R2/R3) and
a requirement area. The highest-value baseline test families:

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

Use the embedded synthetic fixtures from Appendix A §20 (FIX-01 through FIX-11),
the redesign's four-step Sesame tofu bowl fixture (R27.2), the artwork and
equipment fixtures (R27.3), and the vegan starter collection. Demo fixtures load
only through the explicit demo seed path into a separate demo workspace. All numbers and product identities there are
synthetic test data — never a recommended diet, verified price, or a personal
target. A fresh real workspace must not inherit sample pantry stock, the
fictional user's targets, or any demo totals.

### End-to-end review scripts

Appendix A §21.4 scripts A–H (cold start to a real week; incompatible settings;
tired-evening swap; batch and actual use; data portability; document quality;
access and failure; low-maintenance assistance) are the browser-level acceptance
scripts. Run them through the real persistence boundary, not mocked stores.

### Where tests live and how to run them

- Unit/integration: `api/internal/<domain>/..._test.go`, `cli/..._test.go`,
  `ui/src/**/*.test.tsx`.
- Scenario suites: `vrooli scenario test nutrition-planner --phases <phases>`
  (focused) or the `test-genie.iterate` program. The run is server-owned;
  block once with `test-genie runs wait` rather than polling.
- Tag requirement-linked tests with `[REQ:<ID>]`, set the validation `ref`, and
  re-enable the module's auto-sync once all of its validations have refs (D-037).

## Binary startup

`api/main_e2e_test.go` uses `api-core/boottest` to build this API, verify its
health identity in isolated storage, and check shutdown. Keep the expected
service name in the scenario test. Shared process machinery and failure
regressions live in `packages/api-core/boottest`; see its package README section
for configuration and evidence limits. The existing E2E gate runs this test.
