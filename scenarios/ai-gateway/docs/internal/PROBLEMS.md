# Problems — AI Gateway

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

### 2026-07-06 — Security Health Hardening Follow-up

**Symptom:** Phase 8 security validation initially reported findings in AI Gateway command execution, path handling, inventory error conversion, scanner file reads, and shared/toolchain surfaces. Focused product security now reports zero findings, while comprehensive Test Genie still includes advisory Go toolchain and pnpm audit observations.

**Root cause:** Product-code command/path findings came from normal scaffold expansion before the scenario's real resource-command boundaries and scanner rules were hardened.

**Workaround:** No AI Gateway product-code workaround is currently required. Treat comprehensive security observations that point at Go standard-library or JS toolchain advisories as dependency/toolchain governance follow-up.

**Real fix:** Keep future provider/resource additions behind the existing command-runner seams, rerun focused security health before marking new slices complete, and address Go/pnpm advisories through normal dependency governance.

**Owner:** ai-gateway maintainers.

**Refs:** `api/internal/providers/runner.go`, `api/internal/conformance/scanner.go`, `api/handlers/inventory/connect_handler.go`, `api/main.go`, `docs/internal/SECURITY.md`.

### 2026-07-06 — Performance Example Flow Produces Workflow Selector Warnings

**Symptom:** `test-genie execute ai-gateway workflow --wait --json` passes but reports advisory `workflow.selector_unregistered` warnings for `bas/flows/perf-example-scroll.json`.

**Root cause:** The performance example intentionally uses literal `[data-testid=...]` selectors because the performance capture path documents literal selector usage. Workflow Health still audits `bas/flows/**` and reports the literal selector as an advisory.

**Workaround:** Do not bind the example flow as functional validation evidence. The real Phase 7 operator workflows under `bas/cases/operator-ui/` pass and use `@selector` references.

**Real fix:** Either teach Workflow Health to treat `metadata.labels.intent = "performance"` flows with the performance-health selector rule, or replace the example with a product-specific performance flow once performance budgets are introduced.

**Owner:** workflow-health/performance-health follow-up.

**Refs:** `bas/flows/perf-example-scroll.json`, `bas/cases/operator-ui/dashboard-readiness.json`, `bas/cases/operator-ui/route-preview-workflow.json`.

### 2026-07-06 — Route Evidence Has A Temporary Measures Waiver

**Symptom:** `measures-health validate scenario ai-gateway --json` passes, but reports an advisory `measures.undeclared-substrate` warning for `route_events`.

**Root cause:** AI Gateway persists redacted route evidence for operator audit and debugging, but does not yet expose a descriptor-backed analytical measure endpoint for that table. A malformed manifest-only measure was avoided because measures-health must assemble measures against the committed proto descriptor.

**Workaround:** Keep the `routing` domain waiver in `cli/manifest.json` until route evidence is promoted from audit metadata to a real federated measure.

**Real fix:** Add a descriptor-backed measure endpoint for route evidence counts or trends after the shared proto layout is stable, then remove the `routing` waiver.

**Owner:** ai-gateway follow-up.

**Refs:** `cli/manifest.json`, `api/internal/routing/schema.sql`, `packages/proto/schemas/ai-gateway/v1/shared/gateway.proto`.

### 2026-07-06 — Tidiness Still Reports Info-level Hardening Debt

**Symptom:** The Test Genie `tidiness` phase passes, but still reports info-level duplication and complexity findings across scaffold tests, generated-style CLI domain handlers, and shared test utilities.

**Root cause:** The hard Phase 8 blocker was fixed by shrinking duplicated manifest/no-production-import helpers. The remaining findings are lower-severity scaffold and test-support hardening debt.

**Workaround:** Treat these as non-blocking while `test-genie execute ai-gateway tidiness --wait --json` passes and product validation remains clean.

**Real fix:** Run a dedicated tidiness cleanup campaign after the scenario API/CLI surfaces stabilize; avoid mixing those refactors with feature phases.

**Owner:** ai-gateway/platform tidiness follow-up.

**Refs:** `api/internal/testutil/no_prod_import_test.go`, `coverage/logs/*/tidiness.log`.

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
