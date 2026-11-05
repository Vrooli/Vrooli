# Problems — Asset Studio

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Entries below were recorded during design, before any implementation. They are
constraints already known to be real — not speculation — and each names what
would let it be deleted. Append new entries as they appear.

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

### 2026-07-28 — the conformance comparison is unvalidated and it is the whole product

**Symptom:** Nobody has rendered the same identity twice through this pipeline,
so no one knows whether comparing a frame to a reference sheet actually catches
the drift that matters. Both failure directions are plausible: too weak, and
visibly wrong frames pass; too strict, and nothing passes and the gate is
switched off.

**Root cause:** The scenario was designed before any render existed, which was
the right order for the boundaries and the wrong order for the tolerance.

**Workaround:** `ASSET-P0-011` keeps judgement human. An operator comparing a
frame to a character sheet is a calibrated instrument even when the automated
one is not.

**Real fix:** Render one identity repeatedly during the P0 slice and record what
the operator accepted and rejected. That corpus is the calibration set for
`ASSET-P1-005`, and it does not exist until the loop runs.

**Partial mitigation, added later the same day:** D-017 lets an identity carry a
conditioning artifact. Judging a frame against that artifact's characteristic
output is stronger evidence than judging it against a hand-authored sheet, so
`ASSET-P1-012` may narrow this problem considerably. It does not close it —
conditioning improves what is being compared, not whether the comparison
discriminates — and the calibration corpus is still the thing that settles it.

**Second mitigation, from the same review:** D-020 makes the verdict record
name its **basis** — `reference-sheet`, `reference-image-set`,
`conditioning-artifact`, or `prose-only`. This does not make a weak comparison
strong, but it makes the calibration corpus interpretable: a pass judged on
prose and a pass judged on a sheet are different evidence, and without the field
they would be indistinguishable in exactly the dataset meant to settle this.

**Owner:** unassigned.

**Refs:** `ASSET-P0-010`, `ASSET-P0-011`, `ASSET-P1-005`, `ASSET-P1-007`,
`ASSET-P1-012`, D-006, D-017, D-020.

### 2026-07-28 — the rich-media catalogue is empty, so import has nothing to import

**Symptom:** `docs/marketing/catalogs/rich-media/characters/`, `scenes/`, and
`products/` each contain exactly two files: a `README.md` and a
`_template.json`. There are **zero authored characters, scenes, or products**,
and `assets/character-sheets/` holds a README and no sheets. `ASSET-P0-003`
imports from this catalogue; run today it would import nothing and report
success.

**Root cause:** The catalogue was authored as a schema and a convention, and no
record was ever written against it. An earlier revision of this entry asserted
that entries "were authored … so shape drift between entries is likely" — that
was wrong, and it mattered, because it made a corpus problem look like a
validation problem. Unvalidated JSON is a discovery exercise; an empty directory
is a blocked P0.

**Workaround:** `ASSET-P0-018` gives the registry an ingress that does not
depend on canon, and D-021 makes the P0 slice subject a product identity
authored in the workbench. Import is re-sequenced after authoring as the
migration path.

**Real fix:** Two independent things, and only the first is on this scenario's
critical path. (1) Author one product identity in the workbench during the P0
slice — that is the slice, not a prerequisite to it. (2) Author the first
persona in canon by operator decision, which unblocks import having anything to
carry and is what makes character conformance testable. Schema validation of
catalogue entries remains correct and remains untested until (2) happens; expect
the first real import to surface shape problems, and fix the sources rather than
loosening the schema.

**Owner:** operator for (2) — authoring a persona is operator-curated canon
(D-011) and is not an agent's to raise while the marketing team is paused.

**Refs:** `ASSET-P0-003`, `ASSET-P0-018`, D-011, D-021,
`docs/marketing/catalogs/rich-media/`.

### 2026-07-28 — produced artifacts have nowhere to be published

**Symptom:** Every image and video post type in the marketing catalogue is
inactive (`v0`, no paired skill), no persona exists in canon, and no persona
account exists on any platform. An artifact released by this scenario cannot
currently reach an audience.

**Root cause:** Four separate gates sit outside this scenario, and they are
ordered: a persona must be authored in canon, paired post-type skills must be
authored, a `channel-strategy-update` decision must activate persona accounts
with a slate and disclosure protocol, and account warming must run. The persona
gate is upstream of the account gate and is the one most easily overlooked,
because "no account" reads as the blocking fact when "no persona" precedes it.

**Workaround:** None. This is a sequencing fact, not a defect — the scenario is
still worth building because it is on the critical path to all three, and its
first artifact can be consumed by `content-desk` without being published.

**Real fix:** Outside this repository directory. Track it where the gates live.

**Owner:** operator (the channel decision is not an agent's to raise while the
marketing team is paused).

**Refs:** `docs/marketing/strategy/CHANNELS.md`,
`docs/marketing/catalogs/post-types/`, D-014.

### 2026-07-28 — the `video-studio` skill points at a scenario that does not exist

**Symptom:** `scenarios/prompt-manager/store/skills/packs/core/video-studio/`
is marked `active` with `scenario: null`. It describes browser recording,
desktop recording, and FFmpeg compositing — scope that now belongs to
`ASSET-P1-003` and `ASSET-P1-004`.

**Root cause:** The capability was recognised and never built. The skill was
authored anyway.

**Workaround:** None. It misleads any agent that discovers it, in the same way
the retired `campaign-content-studio` skill did.

**Real fix:** When `ASSET-P1-003` lands, either retarget the skill at this
scenario or retire it. Decide deliberately rather than leaving it dangling —
that is the same cleanup `content-desk` D-002 left open for its predecessor.

**Owner:** unassigned.

**Refs:** `ASSET-P1-003`, `ASSET-P1-004`, D-009.

### 2026-07-28 — no artifact-pruning policy, and generation volume is unknown

**Symptom:** Asset bytes accumulate with no deletion path. `DATA.md` records
that deleting a released asset's bytes while retaining its record is
unspecified.

**Root cause:** Volume cannot be estimated before the loop runs. Video at any
meaningful frame count is orders of magnitude larger than the still images P0
produces, so an estimate made now would be wrong in whichever direction video
lands.

**Workaround:** Local filesystem storage with no quota. Acceptable at P0 volume.

**Real fix:** Decide the pruning policy once real volume exists, and make it
preserve provenance and verdicts — a removed artifact must still be explicable.

**Owner:** unassigned.

**Refs:** `DATA.md` § Retention And Deletion, `ASSET-P1-002`.

## Architecture Drift

## Work ladder

- Rung: W3
- Evidence: the approved plan and `OT-P0-001` through `OT-P0-018` agree on the
  identity-to-released-artifact spine; the requirements registry is structurally
  present, but the scenario remains a template scaffold and has no implemented
  product domains.
- Blocker: P0 implementation and server-owned baseline capture are in progress
  (Test Genie run `20260728-212652-b99af4dd`).
- Measured: 2026-07-28

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Docs vs code | The full concept and internal doc set describes five product domains that do not exist in code. This is expected for a designed-not-implemented scenario, but it is drift until the first slice lands, and a reader must not mistake `ARCHITECTURE.md` for a description of what runs. | Docs report `active`; API, UI, CLI, and contracts report `Scaffold` in the maturity table. | Build the P0 slice. The maturity table in `ARCHITECTURE.md` is the honest record until then. |
| Example domain | Resolved 2026-07-28: the template `notes` domain, generated contracts, UI route, CLI surface, and BAS examples were removed after the Studio identity-to-release spine became operational. | The detemplating orientation gate is clear. | Keep future domain references scenario-owned and run `template-manager orient asset-studio` after structural changes. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
