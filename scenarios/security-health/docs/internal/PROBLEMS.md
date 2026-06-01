# Problems — Security Health

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

### 2026-06-01 — Example `notes` domain test does not compile

**Symptom:** The `golangci-lint --fix` generation hook reported
`handlers/notes/module_test.go: cannot use d (*sql.DB) as *database.RoutedDB`.

**Root cause:** Template drift — the react-vite `notes` example test predates
the api-core `RoutedDB` change. Affects only the example's `_test.go`; the API
itself builds clean (`GOWORK=off go build ./...` is green).

**Workaround:** None needed for the architectural core. The `notes` domain is
slated for removal at START-HERE Gate 7 anyway.

**Real fix:** Delete the `notes` example (Gate 7) when the first real domain is
green, or update the template's notes test to the `RoutedDB` signature.

**Owner:** unassigned (handle during Gate 7).

**Refs:** `api/handlers/notes/module_test.go`.

### 2026-06-01 — Architectural core only; producer/handlers not yet built

**Symptom:** The scenario has proto contracts, PRD, requirements, and the
domain map, but no scanner runners, handlers, CLI commands, UI, the test-genie
`security` phase, or EM dimension wiring.

**Root cause:** Intentional staging. This session scoped to scaffold + charter +
requirements + architectural core (per the source plan, Phases A–B and the
contract surface). Phases C–G are deferred to follow-up sessions.

**Workaround:** Follow the source plan phase order: validation core (scanners +
`ValidateScenario`) → CLI `validate scenario --json` → test-genie `security`
phase + `FINDING_SOURCE_SECURITY` + EM `dimensions.json` map → dependency
intelligence (search/reindex) → UI → live A/B proof.

**Real fix:** Implement Phases C–G; delete this entry when the producer flows
end-to-end into the EM R1 gate.

**Owner:** unassigned.

**Refs:** `docs/plans/security-health-scenario-and-test-genie-producer-plan.md`
(repo root), `PRD.md`, `requirements/`.

### 2026-06-01 — Optional scanners (govulncheck, osv-scanner) not installed

**Symptom:** `govulncheck` and `osv-scanner` are absent on the host; only
`gitleaks`, `gosec`, and `pnpm audit` are present.

**Root cause:** Installing them is a `go install` behind the "never install
without explicit permission" rule (source plan §8-R1 STOP-and-ask gate).

**Workaround:** v1 is fully functional on the present subset; absent scanners
register as INFO observations (REQ-P0-009), never failures.

**Real fix:** With user approval, `go install golang.org/x/vuln/cmd/govulncheck@latest`
and `go install github.com/google/osv-scanner/cmd/osv-scanner@latest`.

**Owner:** unassigned (needs user approval).

**Refs:** `docs/concepts/INTEGRATIONS.md` (Dependency Inventory).

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
