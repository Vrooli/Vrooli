# Problems — Backdrop Studio

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
  the code

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

> **Note on scope.** Every entry below is work in *another* scenario that
> Backdrop Studio depends on. They are recorded here because they were
> discovered while designing this scenario and they gate its delivery.
> Each still needs filing against its owning scenario through
> `prompt-manager skill read report-bug` before it becomes scheduled work —
> this file is the design-time record, not the work queue.

---

### 2026-08-11 — asset-studio spec composition may be specialised to character media

**Symptom:** The spec path was reviewed for identity coupling. `asset-studio`'s
prompt and reference-image composition path is built around binding *identity
records* — characters, scenes, products — into a prompt template. Backdrop Studio
binds a scaffold and a palette instead. If the composition path assumes an
identity binding the way the verdict table does, the handoff needs more
generalization than the verdict fix alone.

**Root cause:** Identity-version resolution is optional; the dispatcher accepts
resolved creative intent and generic conditioning references. What *is* established: the asset and
disclosure tables are identity-free (see the entry above), and `OT-P0-016`
already models conditioning artifacts generically enough to include "a trained
adapter, a reference image set, **or a look**" — which suggests the design
anticipated non-character conditioning. That is encouraging but not proof.

**Workaround:** Backdrop Studio composes its own plan (`compose` domain) and hands
`asset-studio` a *result* to release rather than a spec to resolve. If the
composition path turns out to be character-coupled, this boundary keeps the
blast radius to the release call.

**Real fix:** Keep identity-version fields optional for generic creative
specifications and add a regression test whenever a new conditioning kind is
introduced. The model-backed handoff is identity-free and needs no workaround.

**Additional context:** `asset-studio` has now been exercised by conformance and dispatcher tests.
Backdrop Studio is an early generic consumer, so any new conditioning kind still
Backdrop Studio is an early generic consumer, so any new conditioning kind still
needs an explicit contract test before it is treated as stable.

**Owner:** unassigned — `asset-studio`

**Refs:** `scenarios/asset-studio/api/internal/studio/{studio.go,dispatcher.go}`,
`scenarios/asset-studio/PRD.md` OT-P0-004, OT-P0-005, OT-P0-016

---

### 2026-08-11 — two recipe catalogs risk divergence with image-tools Looks

**Symptom:** Not a defect; a design tension worth recording before it becomes
one. `image-tools` already has a **Look** — a prompt template plus ordered AI and
deterministic steps with merged parameters, and a documented `Compile()` seam.
Backdrop Studio's **Style** is a superset of that shape. Two catalogs of
"recipes" in one repository can drift into two answers for the same question.

**Root cause:** The abstractions genuinely differ in scope. A Look is a
*rendering recipe* with no opinion about layout. A Style adds classification,
placement, reserved-region geometry, gates, and lineage — the layout judgement that is
this scenario's whole reason to exist. Collapsing them would push landing-page
concerns into a general-purpose image toolbox.

Worth noting the current seed pack is not a conflict in practice: `image-tools`
ships eleven Looks and all of them are consumer photo filters — Polaroid 600,
Noir, Golden Hour, Anime, Vivid Pop. None is a backdrop recipe. The shapes
overlap; the content does not.

**Workaround:** None needed. Keep Style as the outer record and compile *down* to
a Look or a step list when submitting to `image-tools`, so `image-tools` stays
the single authority on what a rendering step means.

**Real fix:** Revisit if a third consumer needs classified recipes. At that point
the classification layer may deserve promotion out of Backdrop Studio. Until
then, one consumer does not justify a shared abstraction.

**Owner:** unassigned — design watch item

**Refs:** `scenarios/image-tools/api/internal/looks/{compiler.go,seed.go}`

---

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet — scenario is documentation-only._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
