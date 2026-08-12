# Problems — Backdrop Studio

## Remaining after the treatment-layer plan

- `REL-006` remains unbuilt: sized-variant derivation with reserved-region
  preservation is a separate geometry capability and is not silently implied by
  the store/device-frame implementation.
- Asset Studio currently exposes metadata/render/release RPCs but no external
  backdrop byte-ingress RPC. Backdrop Studio therefore owns an injectable
  publisher seam and refuses model-backed release without it. This keeps
  provenance and disclosure from being duplicated or fabricated.
  **Filed 2026-08-12 as `knw-1786507241786326657`** (scenario-qa,
  `bug-inbox/code-defect/asset-studio-exposes-no-external-byte-ingress-rpc`).
- Tier-3 image treatments remain unbuilt by explicit plan decision: `glitch`,
  `kaleidoscope`, `slit_scan`, `fluted_glass`, `photomosaic`, and `resample`.

## Open after the 2026-08-12 output-quality repair

- ~~`Normalize` is not on the wire.~~ **Closed 2026-08-12.** `normalize` now
  exists on every tone-mapping proto message, along with `dark`/`light` on the
  Tier-2 ink-on-paper screens, `distance` on aberration and `spacing` on
  displacement. See the correction below — this was not merely a missing
  convenience, it was breaking renders.
- ~~Every scenario's `api/go.mod` is stale.~~ **Closed 2026-08-12.** Root cause:
  `api-core` and `packages/proto` had moved to `golang.org/x/sys v0.44.0` while
  every consumer still pinned `v0.42.0`, so MVS demanded a version the recorded
  requirement was below. Repaired through the governed gateway
  (`scenario-dependency-analyzer deps install`), never `go mod tidy`: 61 blocked
  modules down to 6. `go test` now runs normally here — the
  backup/`GOFLAGS=-mod=mod`/restore dance is no longer needed and should not be
  reintroduced. The 6 remaining are tool gaps and one governance decision, filed
  as `knw-1786539478410570891`; none of them are in backdrop-studio or
  image-tools.
- **`docs/evidence/` is 8.8 MB across 36 PNGs** now that evidence renders at
  delivery resolution. `treatments/grain.png` alone is 2.9 MB because noise does
  not compress. Delivery resolution was the point — a screen cannot be judged at
  64×48 — but whether this belongs in git or behind a blob seam is an owner
  decision that has not been made.
- **`ascii_mosaic` is the only treatment whose cell size is coupled to a
  font.** It blits a 7×13 bitmap face, so `block_size` values far from 7 resample
  the glyph. Legible, but not crisp at extremes.

## Open after the 2026-08-12 catalog seeding

- **The catalog is 16 styles across 4 scenes.** Adding a genuinely new *subject*
  (botanical, celestial, figure, industrial) needs a new scene generator, not a
  new catalog row — those subjects are only reachable through model-backed
  strategies today, and `scenePreset` refuses them procedurally rather than
  silently substituting a field.
- ~~`TreatmentParams` is unvalidated on write.~~ **Closed 2026-08-12.**
  `validateStyle` now calls `imageengine.ValidateChain`, which rejects malformed
  JSON, non-object values, parameters naming an operation the style does not
  run, and — most importantly — any field image-tools' proto will not accept.
  Both write paths are covered (`CreateStyle` and `ImportStylePack`). The
  wire-format knowledge lives in `internal/imageengine`, so the catalog asks
  "will the engine take this?" without learning protobuf.
- **Two seeded styles cannot be released.** `guided-botanical` and
  `constructivist-figure` are model-backed and blocked on asset-studio byte
  ingress (`knw-1786507241786326657`). They are seeded deliberately so the lanes
  have real coverage the moment that lands.
- **Catalog visual evidence is not committed.** The 12 rendered style previews
  produced during seeding came from a throwaway probe and were removed rather
  than shipped, because unreproducible evidence is the defect this repair
  existed to fix. A repeatable path needs a running image-tools, so it belongs
  in an integration lane, not a unit test.

## Shipped-then-caught: the wire contract (2026-08-12)

The catalog seeded earlier that day requested `normalize` and brand inks on the
Tier-2 screens. Neither existed on `ops.proto`, and `protojson.Unmarshal`
**rejects unknown fields** — so eleven of sixteen styles would have failed their
render with `400 invalid params: unknown field "normalize"`.

Both unit suites stayed green throughout, and that is the lesson worth keeping:
backdrop-studio tests against a fake executor that never reaches the REST edge,
and image-tools tests its treatments below the wire. Neither side could see the
boundary between them. The gate now lives in
`backdrop-studio/internal/imageengine/wire_contract_test.go`, which resolves
every seeded style's parameters against a real brand and asserts the exact bytes
parse as `opsv1.OpParams` — plus `image-tools/handlers/ops/wire_test.go`, which
pins every parameter the engine accepts.

**Rule for future work here: a parameter is not shipped until it round-trips
through protojson.** Adding a knob to `treatments.Params` without extending the
proto produces a loud failure in production and silence in CI.

## Corrections to the 2026-08-11 audit

The audit that drove this repair got one finding wrong, recorded here so it is
not re-derived:

- **The placement preview never applied its placement.** `PreviewPlacements`
  resized the candidate to the viewport and ignored the placement argument
  entirely, so every placement returned identical pixels. It now composites four
  real layouts with scrim and copy chrome.
- **`drawScaled` stretched instead of cover-cropping**, so a 1600x1000 backdrop
  in a tall split panel rendered a circular sun as an oval. Now scaled by the
  larger axis ratio and centre-cropped.
- **"Treatments are not reachable from the CLI" was false.** All 18 were already
  registered in `image-tools/cli/domains/ops/register.go`, with param builders
  and proto messages. `cli/manifest.json` legitimately carries only
  Connect-bound calls; REST multipart run commands are hand-appended and
  documented in the manifest's `omitted` array. The audit tested a **stale
  installed binary** (`~/.vrooli/bin/image-tools`, a day older than the source).
  Rebuild before concluding a CLI surface is missing.

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
