# Problems — Source Ledger

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

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

## Current Working Rung

**W3 — moved engine gate.** Phases 12–14 established and validated the
scenario contract, generated service boundary, migration ownership, degraded
mode, latency ceilings, and the behavior-preserving source-ledger engine move.
The next rung is corpus migration; keep the `vrooli-memory` implementation
intact until cutover proves the new authority path.

### 2026-08-07 — Phase 14 residual advisories

**Symptom:** The passing source-ledger suite still reports architecture and
proto-health advisories: moved domain packages use the existing `*sql.DB`
repository seam, capabilities is a REST probe without a local proto domain,
and code-facts proof is unavailable because the repository contract advertises
an unsupported `project` target kind.

**Root cause:** Phase 14 is intentionally a pure move. Converting every
repository to the routed seam or changing the operational probe contract would
change the move's behavior and scope; the code-facts issue is infrastructure
owned.

**Workaround:** Source-ledger storage validation passes, routed lifecycle tests
pass, and both scenario suites pass 20/20. Track routed-repository conversion
and capabilities contract cleanup as explicit follow-up work rather than
silencing the analyzers.

**Real fix:** Complete the routed-repository conversion and either add a typed
capabilities contract or classify the REST probe as a supported shared
operational surface.

**Owner:** Source Ledger extraction / API-core contract owners.

**Refs:** `api/internal/{facets,forest,journal,policy,recall}`,
`api/handlers/capabilities`, proto-health, storage-manager, Phase 14 evidence.

### 2026-08-07 — Runtime surface language advisory

**Symptom:** Scenario Dependency Analyzer reports one advisory finding for
`scenarios/source-ledger/runtime` because the generated runtime surface has no
supported language evidence yet.

**Root cause:** The scaffold's runtime directory is not an implementation
surface in this contract phase; the Go API and CLI are the authoritative
surfaces until the service boundary is built.

**Workaround:** Treat the finding as advisory while API, CLI, and lifecycle
health checks remain green. Do not add a placeholder runtime implementation to
silence the inventory.

**Real fix:** Add the actual runtime surface or an explicit supported Code
Facts classification when the service contract is implemented.

**Owner:** Source Ledger extraction work.

**Refs:** `docs/concepts/ARCHITECTURE.md`, Scenario Dependency Analyzer health
output, Phase 13 service-contract work.

### 2026-08-07 — Orientation scaffold-health freshness race

**Symptom:** `template-manager guidance next source-ledger` invokes the
declared `make test` scaffold check, but that nested run can terminate before
phases with `source tree changed after admission`; the direct comprehensive
suite remains green.

**Root cause:** The orientation command check and Test Genie admission/freshness
boundary are not coordinated for a server-owned run that creates durable test
evidence during its own lifecycle.

**Workaround:** Use the direct server-owned `vrooli scenario test source-ledger
--preset comprehensive` command and wait on its printed run handle. The latest
direct run `20260807-230918-45c5fccb` passed all 20 phases.

**Real fix:** Make the orientation scaffold check consume a completed,
server-owned run or make Test Genie admission ignore only run-owned artifacts
without weakening source freshness for real source edits.

**Owner:** Template-manager / Test Genie infrastructure.

**Refs:** `scenarios/test-genie/api/internal/orchestrator/suite_execution.go`,
`packages/freshness-go/treedigest`, Phase 12 orientation gate.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| No known architecture drift beyond the advisory recorded above. | The runtime directory remains a future implementation surface. | Advisory only. | Implement the Phase 13 service boundary and classify or replace the runtime surface. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
