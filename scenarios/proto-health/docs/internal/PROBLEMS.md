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

### 2026-06-11 - Generated notes slice is temporary reference code — RESOLVED 2026-11-24

**Disposition:** Resolved. The finding described an obsolete scaffold state:
the notes API, schema, handlers, CLI commands, and UI feature are absent, and
validation, proto-surface, and impact capabilities now have active API/CLI/UI
surfaces. Corrected the current domain, data, architecture, API, CLI, README,
and Start Here references so they no longer instruct maintainers to implement
or delete those already-completed domains. Older seam/tutorial mentions remain
only as explicitly historical examples, not live product contracts.

**Owner evidence:** `api/internal/validation`, `api/internal/protosurface`,
`api/handlers/validation`, `cli/domains/{validate,describe,impact}`, and
`ui/src/features/proto-health/ProtoHealthPanel.tsx` exist. Full API and CLI Go
suites and the focused proto-health panel Vitest suite pass. Focused proof is
not a full scenario `make test` receipt.

**Owner:** proto-health implementer.

**Refs:** `docs/concepts/DOMAINS.md`, `docs/concepts/DATA.md`,
`docs/reference/api-endpoints.md`, `docs/reference/cli-commands.md`,
`api/internal/validation/`, `api/internal/protosurface/`.

### 2026-06-12 - Implementation proof tier is deferred

**Symptom:** Implementation proof can degrade when `code-facts` or one
of its graph providers is not running.

**Root cause:** `proto-health` intentionally consumes source evidence
through `code-facts` instead of parsing Go or TypeScript directly. That
keeps proto policy separate from analyzer implementation, but the proof
tier now depends on code-facts lifecycle discovery and graph-provider
availability.

**Workaround:** Treat `proto.code_facts_unavailable` and unsupported
proof findings as degraded implementation evidence. Descriptor and
static contract findings remain authoritative.

**Real fix:** Keep code-facts healthy through scenario lifecycle, and
extend code-facts provider coverage when a surface returns unsupported
proof for a language or framework it should understand.

**Owner:** unassigned.

**Refs:** `api/internal/codefacts/`, `api/internal/validation/`,
`docs/internal/SEAMS.md`.

### 2026-06-18 - Descriptor staleness is backstopped outside proto-health

**Symptom:** Structural checks read the committed descriptor image, so
a hand-edited schema can leave descriptor-derived facts stale until
generation runs.

**Root cause:** proto-health intentionally does not rebuild descriptors
on the hot path. The manifest verifier hashes source closures and
generated outputs, but descriptor regeneration remains owned by
`packages/proto`.

**Workaround:** Treat `proto.gen_out_of_sync` or
`proto.gen_manifest_missing` as a signal to run
`cd packages/proto && make generate`.

**Real fix:** Keep `make verify-committed-gen` as the authoritative
fleet gate for generated outputs, manifests, and
`gen/descriptor/image.binpb`.

**Owner:** packages/proto maintainer.

**Refs:** `packages/proto/Makefile`, `packages/proto/genmanifest/`,
`api/internal/validation/gen_sync.go`.

## Architecture Drift

Use this section for deferred findings from `architecture-lenses`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None unresolved._ | The stale notes scaffold record was resolved 2026-11-24 after verifying the active replacement domains and updating current owner documentation. | No current template-domain drift is recorded. | Keep canonical domain, API, CLI, and data docs aligned with runtime. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
