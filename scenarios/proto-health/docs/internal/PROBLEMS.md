# Problems — Proto Health

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

### 2026-06-11 - Generated notes slice is temporary reference code

**Symptom:** The scaffold still contains the template `notes` CRUD
domain, its SQLite schema, CLI commands, UI pages, and proto files.

**Root cause:** `proto-health` was generated from the `react-vite`
template before its real `validation` and `protosurface` domains were
implemented.

**Workaround:** Treat notes as reference-only. Do not extend it for
product behavior. Copy patterns from it when implementing the first
real domain, then delete notes in the same change.

**Real fix:** Implement `validation` and `protosurface` with API, CLI,
UI, tests, and docs; remove notes code/protos/docs references after the
replacement pattern is green.

**Owner:** proto-health implementer.

**Refs:** `docs/concepts/DOMAINS.md`, `api/internal/notes/`,
`packages/proto/schemas/proto-health/v1/notes/`.

### 2026-06-11 - Descriptor source-info must be verified early

**Symptom:** `@stability` and other leading-comment annotations are
only available if descriptor source info is present.

**Root cause:** The planned reader consumes
`packages/proto/gen/descriptor/image.binpb`; if that image ever omits
source info, annotation-based checks would silently lose evidence.

**Workaround:** During the descriptor reader implementation, add a
fixture or smoke test that proves leading comments are readable for at
least one known proto file.

**Real fix:** If committed `image.binpb` lacks source info, build the
descriptor with `buf build --include-source-info` or generate a
scenario-scoped descriptor with source info inside the reader.

**Owner:** protosurface implementer.

**Refs:** `packages/proto/Makefile`, `packages/measures-go/paramschema.go`.

### 2026-06-11 - Generated sync finding is still a planned seam

**Symptom:** `internal/validation` defines the stable
`proto.gen_out_of_sync` code, but the production validator does not yet
run a scenario-scoped generated-artifact sync check.

**Root cause:** `packages/proto make check` currently regenerates and
diffs the whole committed `gen/` tree. The validator needs a
scenario-scoped seam before it can translate drift into a finding
without turning every API request into a fleet-wide generation pass.

**Workaround:** Continue running `cd packages/proto && make check`
manually or in CI until the `GenSyncChecker` seam is wired.

**Real fix:** Implement `internal/validation::GenSyncChecker` with a
fakeable interface and production wiring that shells to the existing
buf workflow or consumes its output, then emit `proto.gen_out_of_sync`
from `ValidateScenario`.

**Owner:** proto-health implementer.

**Refs:** `docs/internal/SEAMS.md`, `packages/proto/Makefile`,
`api/internal/validation/types.go`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Template reference | `notes` remains while real domains are not implemented. | Keeps extra product-looking surfaces in a meta scenario. | Remove after `validation` and `protosurface` replace its reference value. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
