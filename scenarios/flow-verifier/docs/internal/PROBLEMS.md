# Problems — Flow Verifier

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

### Template defects surfaced during Phase A scaffold (2026-05-12)

**Problem:** Fresh `vrooli scenario generate react-vite` produced a scenario that could not pass `make setup` or `make test` without intervention. Several template-level defects in `templates/scenarios/react-vite/` required parallel fixes both in the template and in this scenario's scaffolded copies.

**Workaround:** Fixes applied in this session, mirrored to template + scenario:
- Codegen emitted literal `{{SCENARIO_ID}}` in Go replay-helper imports. Added `Options.GoModulePath` to `codegen.Render` plus `detectGoModulePath` in pipeline that reads `<root>/api/go.mod`. Local scenario's `tools/temporal-model/internal/codegen/go.go` patched to match (substituted module name).
- Generated `Transition` Go type was `func(...)` (non-generic) and could not be passed to `modeltest.AssertFormalTransitionsReplay[S,E]`. Changed codegen to emit `type Transition = modeltest.Transition[StatusT, EventT]` (type alias to the generic).
- `formalArtifactSchemaVersion` was 4 in Go and TS modeltest helpers, but codegen now writes schema v6. Bumped to 6 in `api/internal/testutil/modeltest/formal.go`, `formal_test.go`, and `ui/src/test-utils/modeltest/formal.ts`, `formal.test.ts`, `formal.node.test.ts`.
- `LoadFormalArtifact` resolved `artifact.json` relative to CWD, but generated `replay.go` is in `generated/` while `go test` CWD is the parent flow package. Switched to `runtime.Caller(1)` to resolve relative to the calling source file.
- UI test-utils used `node:crypto`, `node:fs`, `Buffer`, `process` but `@types/node` was missing from `ui/package.json` and `tsconfig.json#types` did not include `"node"`. Added both.
- `no_prod_import_test.go` rejected generated `replay.go` importing `testutil/modeltest`. Added `isInGeneratedDir` exemption (mirrors the existing `isInMocksDir` exemption).
- ESLint `no-restricted-imports` rejected `ui/.../flow/generated/replay.helper.ts`. Added `src/**/flow/generated/**` to the eslint `ignores` list.
- Revive flagged the error string in `tools/temporal-model/internal/contract/contract.go` for trailing punctuation; rephrased.

**Real fix:** All template-side fixes are in place; future scaffolds should not hit these. The local scenario's `tools/temporal-model/` was patched as a one-off bridge until Phase G deletes it entirely.

**Owner:** unassigned (template hardening).

**Refs:** `templates/scenarios/react-vite/tools/temporal-model/internal/codegen/{codegen.go,go.go}`, `templates/scenarios/react-vite/api/internal/testutil/modeltest/formal.go`, `templates/scenarios/react-vite/ui/{package.json,tsconfig.json,eslint.config.js,src/test-utils/modeltest/formal.ts}`.

---

### `scenario-auditor` `prd-linkage` standards on initial PRD/requirements (2026-05-12)

**Problem:** Even after the PRD and requirements registry were rewritten with PRD-specific content, scenario-auditor's `prd-linkage` rule reports HIGH+ violations: `P1 target missing requirements` for each of the 4 P1 targets in `PRD.md`, and `Requirement … references missing PRD content` for several requirements. This blocks `make test` and therefore `make orient` Gate 0 (scaffold-health) when test is re-run.

**Workaround:** Direct `make test` (without orient) currently passes the unit/lint phases; the scaffold-health gate intermittently passes vs fails depending on whether the auditor runs. Requirement `prd_ref` values were normalized from `PRD.md#OT-P0-NNN` to `OT-P0-NNN` (matching test-genie format) to clear part of the warnings.

**Real fix:** Add one requirement module per P1 target (state-graph visualizer, trace player, counterexample diff, verification timeline) referencing `OT-P1-001..004`. Defer until Phase H or as part of post-MVP scoping.

**Owner:** continuation of the flow-verifier extraction plan.

**Refs:** `requirements/0[1-4]-*/module.json`, `PRD.md`.

---

### Deferred UI screens (carry-over backlog)

**Problem:** OT-P1-001 (state-graph visualizer), OT-P1-002 (trace player), OT-P1-003 (counterexample diff), OT-P1-004 (verification timeline) are explicitly deferred from the MVP plan.

**Workaround:** None needed; documented as P1 in PRD.

**Real fix:** Schedule follow-up plans after Phase F lands.

**Owner:** unassigned.

**Refs:** `PRD.md` P1 section; `docs/concepts/DOMAINS.md` Deferred Domains table.

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
