# Problems — Music Tools

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

### 2026-08-19 — Declared resources do not exist yet

**Symptom:** `PRD.md`, `ARCHITECTURE.md`, and `INTEGRATIONS.md` describe `ace-step`
and `music-mir` as required managed-service resources. Neither exists under
`resources/`, and `.vrooli/service.json` declares no dependencies at all.

**Root cause:** Documentation-first sequencing. The docs describe the intended
target; the wiring has not been created.

**Workaround:** None needed — but do not read the docs as a description of running
software. Nothing in this scenario executes a model today.

**Real fix:** Create both resources via `template-manager resource-template
generate`, then declare them in `.vrooli/service.json` alongside `qdrant`.

**Owner:** unassigned.

**Refs:** `.vrooli/service.json`, `docs/concepts/ARCHITECTURE.md`.

### 2026-08-19 — Upstream disagrees with itself about track separation

**Symptom:** The composition model's README feature matrix marks track separation,
layer addition, and continuation as supported on all DiT variants. Its inference
reference states these are base-model only. The variant that fits the reference
host's free VRAM is the turbo variant.

**Root cause:** Unresolved upstream documentation inconsistency.

**Workaround:** Treat separation as unavailable from the composition runtime.
`music-mir` owns it unconditionally.

**Real fix:** Test both variants directly and record the measured answer here. The
dedicated separator stays regardless — it is also the better separator.

**Owner:** unassigned.

**Refs:** `docs/reference/model-registry.md` — Operation availability.

### 2026-08-19 — The best structure-and-beats tool declares no licence

**Symptom:** The joint structure, beat, and downbeat model has no licence statement
in its repository.

**Root cause:** Upstream omission.

**Workaround:** Default-to-restricted places it in the non-commercial lane, so a
permissive-lane build already excludes it. No action needed for personal use.

**Real fix:** Obtain a licence statement upstream, or find a permissively licensed
equivalent. Until then the permissive lane has no structure analysis at all, which
is a real capability gap in a commercial build.

**Owner:** unassigned.

**Refs:** `docs/reference/model-registry.md` — Supporting tools.

### 2026-08-19 — Host capacity ledger under-reports GPU consumers

**Symptom:** `vrooli capacity reconcile` reports an unclaimed GPU consumer holding
~3.79 GiB, and four resources that declare GPU usage while holding no active claim.

**Root cause:** Pre-existing host hygiene issue, not caused by this scenario.

**Workaround:** Treat free-VRAM figures as advisory and verify with
`vrooli capacity reconcile` before sizing an operation.

**Real fix:** Bring the existing GPU resources into the claim ledger. This matters
here because this scenario will be the largest GPU consumer on the host, and its
admission verdicts inherit the ledger's accuracy.

**Owner:** unassigned — control-plane concern.

**Refs:** `vrooli capacity reconcile`, `docs/concepts/ARCHITECTURE.md` — Residency policy.

### 2026-08-19 — No throughput figure has been measured on target hardware

**Symptom:** Every performance number in this scenario's documentation is labelled
`vendor` or `estimated`. None was produced on the reference host.

**Root cause:** No implementation exists to measure.

**Workaround:** Treat `docs/internal/PERFORMANCE.md` budgets as hypotheses.

**Real fix:** Run the publisher's own profiling entrypoint and a structure-analysis
pass over a sample of real tracks, then replace the estimates and relabel them
`measured`.

**Owner:** unassigned.

**Refs:** `docs/internal/PERFORMANCE.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Template contract | The generated `react-vite` manifest requires README headings (`What You Get`, `Customize Safely`) and reference-doc headings (`Notes (CRUD reference)`) that the same template version does not emit. `audio-tools` shows the identical mismatch, so this is template-wide, not local. | Fails doc validation on every newly generated scenario. | Upstream: reconcile `templates/scenarios/react-vite` manifest `requiredHeadings` with what the template emits. Locally the README headings were aligned; the `notes` example headings were left alone because `detemplate` removes them. |
| Resource wiring | Docs declare two required resources; `.vrooli/service.json` declares none. | Blocks `implemented`. | Create the resources and declare them. |
| Domain scaffolding | `DOMAINS.md` declares twelve domains; the generated `notes` example domain is still present and none of the twelve exist in code. | Expected at `generated`; blocks `implemented`. | Detemplate, then build domains in the launch order in `PRD.md`. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
- [`../reference/model-registry.md`](../reference/model-registry.md) — model evidence and confidence labels
