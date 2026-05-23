# Problems — TypeScript Code Graph

Persistent register of known issues, tech debt, and deferred work specific to **this** scenario. Future agents read this file to avoid re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from the code

## What does NOT belong here

- **Generic template issues** — those go in [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

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

**Symptom:** No `graph`, `rewrite`, or `sidecar` domain code exists. `Extract` and `Rewrite` Connect-RPC services are not registered. The Node sidecar (`sidecar/`) is not yet present. The `notes` example domain from the react-vite template is still present.

**Root cause:** The scenario was freshly generated from the `react-vite` template. PRD, requirements, and docs were authored in the initialization session, but no product implementation has been done yet.

**Workaround:** Consumers (architecture-cartographer, react-component-library) that depend on typescript-code-graph remain blocked. Cartographer's `e2e_lang_graph_test.go` is build-tagged-off precisely for this reason; rcl's regex parser is still in production.

**Real fix:** Implement the `sidecar`, `graph`, and `rewrite` domains per the PRD operational targets. Start with the sidecar bootstrap (REQ-P0-009) because both `graph` and `rewrite` depend on it. Replace the `notes` example domain. See Gate 6 + Gate 7 in [`../START-HERE.md`](../START-HERE.md).

**Owner:** Next implementation agent.

**Refs:** `requirements/`, `../../PRD.md`, cartographer's `docs/internal/PROBLEMS.md` "typescript-code-graph does not exist yet" entry (now obsolete — this scenario exists, just not implemented).

### 2026-05-23 — Template `notes` test fails linter on generation

**Symptom:** During `vrooli scenario generate react-vite --run-hooks`, the `golangci-lint --fix` post-hook reports `handlers/notes/module_test.go: cannot use d (variable of type *sql.DB) as *RoutedDB` errors.

**Root cause:** Template-level type mismatch in the `notes` example domain. This is a template defect, not a scenario-level issue.

**Workaround:** The lint failure is non-fatal to generation. The scenario itself is correctly created. The `notes` example will be removed in Gate 7.

**Real fix:** File a template-level fix against the `react-vite` template. Not in scope for typescript-code-graph implementation.

**Owner:** react-vite template maintainer.

**Refs:** `templates/scenarios/react-vite/`. Same issue surfaces in `go-code-graph` (sibling scenario generated minutes earlier).

### 2026-05-23 — Node sidecar bootstrap directory does not exist

**Symptom:** No `sidecar/` directory at the scenario root. There is no `package.json`, no `tsconfig.json`, no `src/index.ts` for the sidecar.

**Root cause:** The `react-vite` template does not anticipate language-sidecar scenarios. It seeds `api/`, `cli/`, `ui/`, but not a separate Node code tree alongside them.

**Workaround:** None. The sidecar must be hand-authored as part of REQ-P0-009 (Node Sidecar With Lifecycle Supervision).

**Real fix:** Create `sidecar/` with `package.json` (pinning `ts-morph`), `tsconfig.json`, and `src/` directory containing the IPC protocol implementation. The sidecar build step needs wiring into the scenario's `Makefile` and `.vrooli/service.json`. See SEAMS for the planned `SidecarClient` interface contract.

**Owner:** Next implementation agent (Phase 2 of launch sequencing per PRD).

**Refs:** PRD OT-P0-009, REQ-P0-009.

## Architecture Drift

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| `notes` example domain | Present in the scaffold; not part of product scope. | Counts against domain map clarity; surfaces in DOMAINS.md as "template starter only". | Remove per Gate 7 in `START-HERE.md` once the first real domain is green. |
| `sidecar/` directory absent | Required by REQ-P0-009; not seeded by template. | Blocks `graph` and `rewrite` implementation. | Hand-author as the first implementation phase. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
