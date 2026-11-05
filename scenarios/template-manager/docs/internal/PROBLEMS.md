# Problems — Template Manager

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Append entries as they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-07-09 — comprehensive maturity gates remain red

**Symptom:** Focused Template Manager phase validations pass, but the default
`vrooli scenario test template-manager` comprehensive run still reports broader
maturity-provider failures.

**Root cause:** The current plan slices intentionally shipped domain behavior
before final hardening. Remaining findings include structure/profile evidence,
architecture ownership, docs snippets, duplication, security advisories, and
other provider-level maturity gaps.

**Workaround:** Use the focused phase validations recorded in the plan evidence
for Phase 1 through Phase 7 while continuing the remaining plan phases.

**Real fix:** Complete Phase 8 through Phase 11, then run and close the full
comprehensive suite and baseline diffs.

**Owner:** Template Manager implementation agents.

**Refs:** operator-local plan `template-manager-scenario-owning-the-template-domain.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| comprehensive suite | Focused plan slices are green while broader maturity gates remain unresolved. | Template Manager cannot claim final DoD until the default suite is green. | Finish hard cutover and final hardening phases, then update this entry with the closing run id. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

## 2026-09-07 — Shared selector consolidation

W3 implementation review under the existing selector contract; no W0–W2 maturity promotion is claimed. Both React/Vite templates use @vrooli/ui-selectors and exporter scripts with governed package declarations and locks. Focused generation/copy checks pass. Unit run 20260907-213240-57f1e0c5 could not acquire unit-health. Full templateengine testing additionally found an invalid compose-service resource driver (knw-1788818286865080380), outside this selector change. Shared evidence and limitations: `packages/ui-selectors/README.md`.

## 2026-11-02 — Transitional resource-template drivers

The four supported resource archetypes are now canonical. Retained templates
with the old `compose-service`, `docker-service`, `desktop-app`, and `manual`
drivers are explicitly marked transitional, remain visible for inspection, do
not block canonical validation, and cannot generate new invalid resource
manifests. This fixes the earlier compose-service validator failure at the
source owner. `go test ./internal/templateengine` passes. Test Genie run
`20261102-204531-37acc077` passed its API Go workspace; the whole unit phase
remains failed by the existing UI suite and coverage gate. The current four
canonical template generation checks still report three independent issues;
this change does not claim the full resource-template validation gate is green.
No live API restart occurred. Work record:
`e6774cd6-04af-4a02-ab4c-1935e9bcee84`.

Follow-up on 2026-11-02: repaired the four current archetype templates too.
`cloud-api` no longer treats prose labels for network/credentials as registered
host tools; outbound network and credential requirements are explicit in its
profile limitations and credential descriptors. `external-cli` uses its
`binary` field instead of treating a variable executable name as a registered
tool. The generated Go CLI modules now explicitly replace the transitive
in-repository modules used by `cli-core`, because Go does not inherit replace
directives from dependencies. `go test -count=1 ./internal/templateengine`
passes and exercises canonical generation validation. Test Genie unit run
`20261102-204832-d5d12413` still fails only in the existing UI coverage command
(27/140 UI tests fail; global coverage is below threshold); no UI changes were
made. The API/source suite and canonical template validation are green. No live
API restart occurred. Work record: `9c6746dc-1ee6-4742-831f-05ac720d16cc`.

## 2026-09-07 — Unused health fixture cleanup

W3 scoped implementation cleanup: removed 67 unused scenario health fixture
pairs after checking that builder and option symbols had no external Go
consumers. The canonical template example remains linked from the test-authoring
guide; manifest copy exclusions prevent its propagation to new scenarios.
Focused generation/copy checks and Proto Health handler/testutil tests pass.
Test Genie unit run: `20260907-222237-5e59d117` (terminal result recorded in
the work-record journal). No W0–W2 readiness claim is made.

## Work ladder — shared gamepad ownership

- Rung: W3 (scoped implementation change; W0–W2 not re-certified).
- Evidence: scenario entry points called `initSpatialNav()`, while local React
  hooks created independent managers; Swarm Manager's graph used both paths.
- Repair: shared application controller and React adapters, scoped action
  routing, modal registration cleanup, and template/provider migration.
- Measured: 2026-09-07. Package and scenario validation recorded in the work journal.

## Work ladder — shared binary boot harness

- Rung: W3, scoped consolidation under the existing binary-startup contract.
- Evidence: 67 scenario boot tests and the canonical template duplicated process
  setup and weak health/shutdown checks. `packages/api-core/boottest` now owns
  those mechanics; scenarios retain service identity and necessary startup inputs.
- Measured: 2026-09-07. Shared regression tests pass with the race detector. Real
  boot checks pass for Proto Health, Unit Health, Vrooli Bridge, and Template Manager.
- Limitation: Test Genie unit run `20260908-032406-c3fc9bec` returned FAIL with
  `UNIT_REQUIRED_ROLE_MISSING`: Code Facts did not observe the required UI role.
  This is separate from the passing focused boot checks; no W0–W2 claim is made.

## Work ladder — locale catalog ownership

- Rung: W3, repair within the established locale-parity contract; W0–W2 were
  not re-certified.
- Evidence: Test Genie run `20261102-204832-d5d12413` retained a locale test
  finding that English had 63 root keys absent from both Arabic and Japanese.
  Those keys came from React Component Library story/package metadata, not
  Template Manager translations; all three catalogs shared the same 19 app
  sections.
- Repair: removed the 63 metadata entries from the canonical English catalog
  and regenerated the owned TypeScript string map with `node
  scripts/gen-strings.mjs`.
- Validation: `pnpm exec vitest run src/i18n/locales/locales.test.ts` passes all
  five tests; `pnpm strings:check` passes. Test Genie run
  `20261102-210659-a76db834` remains FAIL, but the locale parity failure is
  absent: UI failures decreased from 27/140 to 26/140. Eleven UI test files
  still fail and function coverage remains 81.94% against 85%; no cause is
  assigned to those remaining failures. No live Template Manager restart.

## Work ladder — Template Manager list test isolation and Select contract

- Rung: W3, repair within the existing UI/test-harness contracts; no broader
  requirement, lifecycle, or runtime status was re-certified.
- Evidence: Test Genie run `20261102-210659-a76db834` retained 26 failing of 140
  UI tests. `api-base/testing.renderWithProviders` already created a fresh
  QueryClient per render, but Template Manager's test setup added a singleton
  QueryClientProvider that shadowed it and leaked query results between tests.
  Removing that provider eliminated the loading/error failures. The next run
  `20261102-212918-ea38cf63` retained four failing list-filter tests: they sent
  `change` events to RCL Select buttons, while the pages used the synthetic
  native-event callback instead of RCL's `onValueChange` value contract.
- Repair: removed the duplicate process-wide query provider; wired Template,
  Debt, and Run list filters through `onValueChange`; changed filter tests to
  open and select visible listbox options. The test console guard now ignores
  only jsdom 25's stylesheet-parser diagnostic for RCL's modern token sheet,
  identified by its exact message and jsdom parser stack; other console output
  remains an error. This is limited to test-environment evidence and does not
  qualify browser rendering or runtime behavior.
- Validation: focused list tests pass 15/15. Test Genie unit run
  `20261102-213250-97d38f21` passes; the preceding run
  `20261102-213159-174b25a9` had four failures. The first run had 26 failing
  tests. The passing run retains 49 nonblocking coverage/injectable-seam
  advisories; no live service restart occurred.
