# Problems — Workflow Health

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

### 2026-07-02 — Generator design alias mismatch

**Symptom:** `vrooli scenario generate react-vite --design default --run-hooks` failed with `design kit not found: default`.

**Root cause:** The template metadata describes `default: vrooli-default`, but the generator currently expects the concrete kit id.

**Workaround:** Use `--design vrooli-default` when generating workflow-health.

**Real fix:** Align generator flag resolution with template metadata, or update the plan/template docs to use concrete kit ids.

**Owner:** Vrooli generator maintainers.

**Refs:** `templates/scenarios/react-vite/template.json`, `vrooli scenario template show react-vite`, scenario-qa bug `knw-1783012148307954507`.

### 2026-07-02 — API generated module needed cli-core replace

**Symptom:** The workflow-health generation post-hook failed during generated module validation while resolving `github.com/vrooli/cli-core v0.0.0` from the network.

**Root cause:** `workflow-health/api/go.mod` indirectly imports cli-core through local api-core code, but Go module replace directives are not transitive.

**Workaround:** Add `replace github.com/vrooli/cli-core => ../../../packages/cli-core` to the generated API module.

**Real fix:** Update the react-vite template or module validator so generated API modules include local replaces required by local package dependencies.

**Owner:** Scenario template maintainers.

**Refs:** `api/go.mod`, `packages/api-core/preflight`, `packages/api-core/staleness`, scenario-qa bug `knw-1783012148329275426`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Template UI/API example surfaces | The scaffold still contains generic starter UI/API behavior until implementation phases replace it. | Keeps workflow-health at contract/scaffold maturity only. | Build catalog, validation, execution, search, remediation, and UI domains, then detemplate/remove sample surfaces. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
