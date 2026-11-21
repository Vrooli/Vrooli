# Problems — Music Library

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

### 2026-08-19 — Every capability depends on a scenario with no implementation

**Symptom:** This scenario runs no models by design; all attributes come from
`music-tools`. That scenario is documentation-only today, and neither scenario
declares dependencies in `.vrooli/service.json`.

**Root cause:** Documentation-first sequencing across a dependency pair.

**Workaround:** None needed. Library scan, playback, and the elicitation surfaces
can be built against a stubbed attribute provider before `music-tools` executes
anything real.

**Real fix:** Declare the `music-tools` scenario dependency and `qdrant`, then
build against the real contract.

**Owner:** unassigned.

**Refs:** `.vrooli/service.json`, `docs/concepts/INTEGRATIONS.md`.

### 2026-08-19 — Section-level skip attribution has no prior art

**Symptom:** The design attributes a skip to the structural section that was
playing. A literature search found no prior work on within-track feedback
attribution; published work treats skips as a track-level signal and reports them
as noisy.

**Root cause:** Genuinely novel mechanism, unvalidated.

**Workaround:** It is gated by construction — attribution requires repetition
across plays before it updates the profile, and is corrected for position bias.
No other requirement depends on it.

**Real fix:** Evaluate against accumulated listening history once enough exists.
If it does not hold up, delete the mechanism; the product does not need it.

**Owner:** unassigned.

**Refs:** `docs/internal/DECISIONS.md`, requirements module `07` signals.

### 2026-08-19 — The preference model is useless before it has comparisons

**Symptom:** A Bayesian preference model over an embedding space starts with near-
uniform uncertainty. Early recommendations will be close to arbitrary.

**Root cause:** Inherent cold start for a single-listener content-based model. It
cannot be borrowed from other users because there are none.

**Workaround:** Seed from existing listening history where available, and lead new
listeners into the comparison surface rather than the queue.

**Real fix:** Quantify how many comparisons are needed before ranking beats a
random baseline, and make that the first-run experience target.

**Owner:** unassigned.

**Refs:** requirements modules `03` preference-model and `04` elicitation.

### 2026-08-19 — Player fundamentals are real unbuilt work

**Symptom:** Building the interface rather than adopting an existing self-hosted
music server means this scenario owns transcoding, format coverage, gapless
playback, library scan, and offline behaviour.

**Root cause:** Deliberate — see `DECISIONS.md`. The differentiating surfaces have
no representation in existing players' data models.

**Workaround:** None. This was accepted with the decision.

**Real fix:** Sequence these as explicit requirements rather than discovering them
during implementation. None is novel; all are real.

**Owner:** unassigned.

**Refs:** `docs/internal/DECISIONS.md`, requirements module `01` library-playback.

### 2026-08-19 — Not yet mapped to a bundle SKU

**Symptom:** This scenario is described as the first Lifestyle bundle scenario, but
it has no entry in the monetisation catalogue's scenario-to-SKU map. The lifestyle
bundle itself is `candidate`, not `active`.

**Root cause:** That map is operator-curated monetisation canon. Agents do not edit
it; membership is an operator decision.

**Workaround:** None needed — the claim is documented here and in
`docs/business/MONETIZATION.md` as intent, not as an existing mapping.

**Real fix:** Operator adds the mapping when the bundle is promoted.

**Owner:** operator.

**Refs:** `docs/business/MONETIZATION.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Template contract | The generated `react-vite` manifest requires README headings (`What You Get`, `Customize Safely`) and reference-doc headings (`Notes (CRUD reference)`) that the same template version does not emit. `audio-tools` shows the identical mismatch, so this is template-wide, not local. | Fails doc validation on every newly generated scenario. | Upstream: reconcile `templates/scenarios/react-vite` manifest `requiredHeadings` with what the template emits. Locally the README headings were aligned; the `notes` example headings were left alone because `detemplate` removes them. |
| Dependency wiring | Docs declare a `music-tools` scenario dependency and `qdrant`; `.vrooli/service.json` declares none. | Blocks `implemented`. | Declare both. |
| Blindness boundary | `DOMAINS.md` states ranking and generation are blind to monetisation, enforced by package boundary. No package boundary exists yet. | Blocks `implemented`; the claim is load-bearing for trust. | Create the boundary and a test that fails if `ranking` can import `offers`. |
| Domain scaffolding | `DOMAINS.md` declares eleven domains; the generated `notes` example domain is still present and none of the eleven exist in code. | Expected at `generated`; blocks `implemented`. | Detemplate, then build in launch order. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
