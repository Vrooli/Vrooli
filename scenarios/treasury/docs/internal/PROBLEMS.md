# Problems — Treasury

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

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

### 2026-08-18 — Generated scenario cannot complete Gate 0 (scaffold health)

**Symptom:** `make setup` / `vrooli scenario start treasury` fails at the
`build-ui` step with exit code 2. `tsc --noEmit` rejects the generated file
`ui/src/test-utils/renderWithProviders.test.tsx` with TS2322: two different
`QueryClient` types, one from `@tanstack/query-core` 5.100.9 and one from
5.59.0.

**Root cause:** a version-range mismatch between two governed sources. The
`react-vite` template pins `"@tanstack/react-query": "^5.59.0"` in
`templates/scenarios/react-vite/ui/package.json`, which resolves to 5.100.9
and brings `query-core` 5.100.9. The shared package `packages/api-base`
pins `@tanstack/react-query` at exactly 5.59.0, bringing `query-core`
5.59.0. Both land in the generated scenario's lockfile and the generated
test file mixes them. **This is inherited template debt, not specific to
this scenario** — any scenario generated from `react-vite` today hits it.
`money-ledger` and `offer-desk` carry no tanstack dependency, which is why
the fleet has not surfaced it.

**Workaround:** none for the UI build. The Go API builds and the
documentation gates are unaffected, which is why gates 1-5b completed while
gate 0 did not.

**Real fix:** align the template range with the `api-base` pin, or widen
the `api-base` pin — through `scenario-dependency-analyzer`, never by hand.
Not attempted here: shared-package version changes ripple across the whole
fleet and are outside the scope of authoring one scenario's documentation.

**Owner:** unassigned; filed to scenario-qa.

**Refs:** scenario-qa bug `knw-1787080554065889915`
(`bug-inbox/code-defect/every-newly-generated-react-vite-scenario-fails-build`);
`/home/matthalloran8/.vrooli/logs/treasury.log`.

### 2026-08-18 — Experience pages have no BAS observer cases

**Symptom:** `experience-manager spec validate treasury` reports four
`experience.route_unspecced` warnings for `approvals`, `budgets`,
`mandates`, `activity` and `rails`.

**Root cause:** the experience contract now describes the intended product
routes, but no UI exists yet, so there are no selectors for a BAS case to
wait on. The generated `dashboard` observer smoke was retargeted to
`approvals` (which owns `/`), and its selector reference
`@selector/pages.approvals` is a placeholder until the page is built.

**Workaround:** none needed. The warnings are accurate.

**Real fix:** add one observer smoke per page as each page is implemented,
with real selectors. Also raise the page claims from `aspirational` to
`machine` tier once stable roles and `data-testid` selectors exist — the
claims are currently stated intent that the validator cannot check.

**Owner:** whoever implements the UI.

**Refs:** `experience/pages/`, `bas/cases/experience-spec/approvals.json`.

### 2026-08-18 — Every security mitigation is designed, not verified

**Symptom:** `docs/internal/SECURITY.md` describes a complete threat model
in which every mitigation carries status `designed`.

**Root cause:** no implementation exists. The structural claims — that the
agent-facing service declares no policy-mutating method, that identity
fails closed, that the schema refuses a third-party beneficiary — are
architectural intentions until a test earns them.

**Workaround:** treat the security posture as a specification to build
against, not as a description of the system. Do not repeat any claim from
that document as fact in a plan, report, or external statement until its
requirement's validation is earned by a passing test.

**Real fix:** implement, and let requirement sync flip the statuses from
evidence rather than assertion.

**Owner:** whoever implements the P0 spine.

**Refs:** `docs/internal/SECURITY.md`, `requirements/01-must-ship/module.json`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
