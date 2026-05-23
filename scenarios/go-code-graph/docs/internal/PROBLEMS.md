# Problems — Go Code Graph

Persistent register of known issues, tech debt, and deferred work specific to **this** scenario. Future agents read this file to avoid re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from the code (e.g., "this resource needs warm-up before the first call; see commit X")

## What does NOT belong here

- **Generic template issues** — those go in [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a comment there is more discoverable
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

### 2026-05-23 — Implementation has not started

**Symptom:** No `graph` or `rewrite` domain code exists. `Extract` and `Rewrite` Connect-RPC services are not registered. The `notes` example domain from the react-vite template is still present.

**Root cause:** The scenario was freshly generated from the `react-vite` template. PRD, requirements, and docs were authored in the initialization session, but no product implementation has been done yet.

**Workaround:** Consumers (architecture-cartographer) that depend on go-code-graph remain blocked. Cartographer's `e2e_lang_graph_test.go` is build-tagged-off precisely for this reason.

**Real fix:** Implement the `graph` and `rewrite` domains per the PRD operational targets, starting with `Extract` against fixture Go modules. Replace the `notes` example domain. See Gate 6 + Gate 7 in [`../START-HERE.md`](../START-HERE.md).

**Owner:** Next implementation agent.

**Refs:** `requirements/`, `../../PRD.md`, cartographer's `docs/internal/PROBLEMS.md` "go-code-graph and typescript-code-graph do not exist yet" entry (now obsolete — these scenarios exist, just not implemented).

### 2026-05-23 — Template `notes` test fails linter on generation

**Symptom:** During `vrooli scenario generate react-vite --run-hooks`, the `golangci-lint --fix` post-hook reports `handlers/notes/module_test.go: cannot use d (variable of type *sql.DB) as *RoutedDB` errors.

**Root cause:** Template-level type mismatch in the `notes` example domain between the test helper signature and the `notes.ModuleWithBlobStore` constructor. This is a template defect, not a scenario-level issue.

**Workaround:** The lint failure is non-fatal to generation. The scenario itself is correctly created. The `notes` example will be removed in Gate 7 along with the lint issue.

**Real fix:** File a template-level fix against the `react-vite` template in the `templates/scenarios/react-vite/` directory. Not in scope for go-code-graph implementation.

**Owner:** react-vite template maintainer.

**Refs:** `templates/scenarios/react-vite/`, scenarios that have hit the same issue (`typescript-code-graph` shows the identical generation log).

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`. Do not create a standalone architecture-audit report unless the work is a migration handoff with a planned retirement path back into `ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| `notes` example domain | Present in the scaffold; not part of product scope. | Counts against domain map clarity; surfaces in DOMAINS.md as "template starter only". | Remove per Gate 7 in `START-HERE.md` once the first real domain (`graph`) is green. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
