# Problems — Chain Forensics

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file records only unresolved scenario-specific blockers. Generated
template problems that are owned upstream stay with their owning route.

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

### 2026-11-09 — Generated UI dependency graph has two blocking advisories

**Symptom:** the first comprehensive Test Genie run (`20261109-214830-675f059d`)
failed the `security` phase on `GHSA-27r3-jxmh-p828` (`body-parser@1.20.6`) and
`GHSA-jqcg-44mw-7w3h` (`proxy-addr@2.0.7`). The phase also reported inherited
security warnings across the generated package lock and template example code.

**Root cause:** `express@4.22.2` is resolved through the approved first-party
`@vrooli/api-base` package and pins vulnerable transitive versions in the
current lock. The Scenario Dependency Analyzer exposes governed install and
lock resync routes but no reviewed transitive override/lock-only update path;
adding unused direct dependencies or hand-editing the lock would obscure the
owner and bypass that contract.

**Workaround:** do not treat scenario security as passing. Do not deploy or
describe the scaffold as production-ready. Continue package and local evidence
work that does not depend on the UI lock.

**Real fix:** resolve the vulnerable versions through the dependency owner's
governed JavaScript update route (and propagate any accepted dependency
decisions into the React-Vite template), then rerun the focused security phase.

**Owner:** Scenario Dependency Analyzer and Template Manager owners.

**Refs:** `ui/package.json`, `ui/pnpm-lock.yaml`, Test Genie run
`20261109-214830-675f059d`.

### 2026-11-09 — Scenario transport and UI are not yet joined to the Go package

**Symptom:** the exported `github.com/vrooli/chain-forensics` evidence package
and its offline inspect/packet CLI are independently testable, but the
generated scenario's Connect/API, registered scenario CLI, and React routes
still serve the Notes example and do not load this case.

**Root cause:** the Scenario Dependency Analyzer refuses to install the new
first-party Go module as unrecorded. The current plan also explicitly excludes
hand-editing the global approved-dependency registry. A local consumer import
would therefore bypass dependency governance.

**Workaround:** use `go test ./...` in `packages/chain-forensics` to validate
the reusable model and replay. Do not count this as API/CLI/UI integration or
CI2 adoption.

**Real fix:** route a first-party module approval through the SDA's governance
owner and install it through SDA, or establish another approved monorepo
composition route; then wire Connect, the registered scenario CLI, local packet
storage/export, and the case-review UI. CI2 adoption requires a real CI2 import.

**Owner:** Chain Forensics owner with Scenario Dependency Analyzer, then
Crypto Infrastructure owner for CI2.

**Refs:** `packages/chain-forensics/README.md`, Plan Manager plan
`cf6-chain-forensics-capability`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| React-Vite starter remains in API, CLI, and UI sources pending a green case slice. | Product domain not yet wired through generated surfaces. | First-slice maturity remains incomplete. | Build and validate the case domain across API, CLI, and UI, then use `template-manager detemplate chain-forensics`. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
