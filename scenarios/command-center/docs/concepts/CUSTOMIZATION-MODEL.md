# Customization Model

**Status:** design contract for the generalization effort. Not yet implemented. This document defines how Command Center becomes fully configurable — connections, rooms, themes, backgrounds, and connector-specific visuals — **without any change to what the operator's own instance renders today**. It governs the settings surface, the catalog formats, the slot-binding schema, and the customization skill. Where it touches existing models it defers to `INSTRUMENT-MODEL.md`, `COVERAGE-MODEL.md`, `PROVENANCE-MODEL.md`, and `UI-ARCHITECTURE.md`.

## The one invariant that governs everything

**Zero visual diff.** When the generalization is complete, the operator's instance must render pixel-for-pixel identical to today — the same six rooms, the same themes, the same compositions, the same cycling — because those become the *shipped default preset set* plus the *Vrooli connector pack*, expressed as data instead of code. Generalization is proven done not by a feature list but by re-deriving today's board from the new data files and measuring **no visual difference** across sampled frames of the running scenes (the same frame-sampling discipline `UI-ARCHITECTURE.md` already requires under `CC-P1-005`).

Everything below is subordinate to that invariant. A refactor that changes the operator's board is wrong even if it is "more general."

## The seam: engine versus instance

Generalization draws one seam.

- **The instrument engine** is domain-agnostic and ships with the scenario: the rendering pipeline, the honesty model, the board cycler, the catalogs of themes/compositions/readouts, and the generic connector runtime. It knows nothing about revenue, Vrooli, or director-swarm.
- **An instance definition** is the data that makes one particular board: which connectors are wired, which signals they yield, which rooms exist, and how each room is themed, composed, and cycled. The operator's current board is **instance zero** — the first instance, not a special case.

Today these are fused: Vrooli data sources are compiled Go clients, room→theme→composition is a code map, and scene→metric bindings are hardcoded ID strings. The effort moves the instance-specific facts out of code and into data, and leaves the engine holding only what is true for every instance.

## Vocabulary

The board decomposes into seven independent, separately swappable layers. They are joined only where noted; everything else is orthogonal.

| Layer | Definition | Joined to |
|---|---|---|
| **Connector** | A live data source: a resolved base URL, an auth reference, and a transport (HTTP/JSON, Connect, GraphQL). | — |
| **Signal** | One measurement, produced by selecting a value out of a connector response (a select path + TTL). Carries the honesty axes of `COVERAGE-MODEL.md`. | a connector |
| **Theme** | A palette only: gradient, accent, foreground, glow, corner radius. **No geometry, no color logic.** | — |
| **Composition** | An animated background scene: pure geometry and motion, plus a set of declared **data slots**. **Names no color** — it receives a palette at draw time. | slots ← signals |
| **Readout** | A foreground tile that renders one signal (`hero`, `panel`, `ladder`, `funnel`, `leaderboard`, `posture`, …). Selected by the signal's `kind`. | a signal |
| **Beat** | One time-segment of a room's cycle: a hero readout, supporting readouts, a layout, and a dwell time. | signals, readouts |
| **Room** | A named board screen: a theme, a composition, and an ordered list of beats. | theme, composition, beats |

The load-bearing consequence: **a connector is plumbing, a room is presentation, and they meet only at the signal ID.** One connector feeds many signals across many rooms; one room draws signals from many connectors. Connect once, reference anywhere.

## What is already decoupled — do not rebuild it

The rendering layer is further along than the vision implies. These properties already hold in the current code and must be *preserved*, not re-engineered.

1. **Color is already theme-driven at render time, not welded to compositions.** The room's theme sets `data-theme` on `<html>`; `AmbientCanvas` re-reads the CSS variables into a `palette` object every 250ms and passes it into every scene's `draw()`. Each composition draws with `palette.primary` / `palette.accent` / etc. and names no color of its own. *Consequence:* if two rooms swap themes, their backgrounds recolor within 250ms with zero code change. This is the target invariant for compositions, already satisfied — the generalization must not regress it.
2. **Readouts are already a generic dispatch on the signal's `kind`**, not on room identity. The release-ladder is not "the forge's readout"; it is what any beat whose hero has `kind:"ladder"` renders. Room-to-readout is already a data fact.
3. **Beats are already data-driven** from the board response (count, order, hero, per-beat composition, layout, dwell), with a code fallback.
4. **Themes are already API-overridable** per room, with the code `THEMES` map only as a fallback.
5. **Data-driven backgrounds already work.** Compositions already read live signals (throughput/blocking drive the forge sparks and dam; the focus metric's rows drive the broadcast great-circle arcs; conversion ratios drive the signal constellation). The binding exists; it is the *addressing* of that binding that is welded (next section).

## What is welded — the actual refactor

Four couplings, all mechanical rather than conceptual, stand between today and a data-driven engine.

1. **Scene→signal addressing is hardcoded.** A composition reads its data by literal metric ID (`read(data, "throughput_stats")` inside `flowCurrent`). This is the central coupling. **Fix:** a composition declares abstract **slots**; a room **binds** a signal to each slot; the scene reads its slots, never a global ID. Then the same "dots-along-lines" background works for anyone's throughput-shaped signal.
2. **Room→theme→composition is a code map + a fixed scene registry** keyed on the six room names. **Fix:** rooms become preset data; the scene registry becomes a catalog keyed by composition ID.
3. **Themes and compositions are code/CSS lists.** Adding one means editing `scenes/index.ts`, adding a `.css` file, and adding a `THEMES` entry. **Fix:** theme = JSON design-token file; composition = a registered module declaring its slots.
4. **The supporting strip has inline per-`kind` rendering** in places instead of delegating to readout components. **Fix:** a single readout registry maps `kind` → component for both hero and supporting positions.

Nothing here changes what the operator sees; each is a lift from code-addressed to data-addressed with the current values as the first data.

## The catalogs

Four catalogs replace four code lists. Each ships with the current set as its defaults.

- **Theme catalog** — one file per theme carrying only tokens (gradient stops, `--color-primary`, `--color-accent`, `--color-glow`, corner radius, halo widths). Ships the six current themes. Adding a theme is a new file; no CSS authoring, no code.
- **Composition catalog** — one registered module per background, declaring: its ID, its capability tier behavior (per `UI-ARCHITECTURE.md`'s Full/Reduced/Still ladder), its **quiet-zone** contract, and its **slot manifest** (below). Ships the nine current compositions.
- **Readout catalog** — one entry per `kind` → component, for hero and supporting positions. Ships the current `hero`/`panel`/`ladder`/`funnel`/`leaderboard`/`posture`/`sky` set.
- **Room preset catalog** — a room is `{ id, theme, composition, beats[], signals[] }`. The six current rooms become six preset files. A user clones a preset, swaps the theme, reorders beats, rebinds signals — all data.

## The slot-binding schema — the core design

This is the one part with genuine design choices, because it is what turns "the background reacts to data" from bespoke wiring into configuration.

A **composition declares slots** it can consume. A slot has a name, a shape, and a degradation rule:

```jsonc
// composition: flow-current  (forge dots-along-lines)
"slots": {
  "rate": { "shape": "scalar", "role": "emission-rate", "whenUnbound": "calm" },
  "pool": { "shape": "scalar", "role": "backpressure",  "whenUnbound": "empty" }
}
// composition: meridian-arc  (broadcast earth arcs)
"slots": {
  "arcs": { "shape": "rows", "fields": ["key→destination", "share→weight"], "whenUnbound": "globe-only" }
}
```

A **room binds a signal to each slot**:

```jsonc
// room: forge
"composition": "flow-current",
"bind": { "rate": "throughput_stats", "pool": "blocking_stats" }
```

Three rules make this safe:

1. **A composition never names a global signal ID.** It reads `slot("rate")`, never `read(data, "throughput_stats")`. All addressing lives in the room's `bind`.
2. **Unbound degrades, never breaks.** Every slot declares `whenUnbound`. A composition with no bindings renders decoratively (the current graceful-`null` behavior, made explicit). This preserves the `CC-P1-003` "a scene that draws nothing is a failure" contract: a slot degrades to a defined visual, not to a blank canvas.
3. **Shape is validated at bind time.** Binding a `scalar` signal to a `rows` slot is a settings-time error surfaced in the editor, not a runtime surprise.

Shapes to support at minimum: `scalar`, `series`, `rows`, and `meta` (a small object, e.g. the offer posture's burn/revenue pair). These are exactly the shapes the current scenes already consume — the schema is a naming of what exists, not a new capability.

## Connector packs — where connector-specific visuals live

Some readouts and compositions are generic (a funnel, a leaderboard, a scalar hero, an orbital field). Others are bespoke to a data source: the "what each release opens" hive visual, the forge release-ladder, the broadcast earth-arcs. These must not live in the engine, or every instance carries dead Vrooli-shaped code; and they must not be lost, or the operator's board changes.

**Resolution:** bespoke readouts and compositions ship *with the connector that feeds them*, as a **connector pack**, and register themselves only when that connector is enabled.

- The **engine** ships the generic catalogs.
- A **connector pack** (e.g. `vrooli/offer-desk`) ships its data bindings *and* its bespoke readouts/compositions. Enabling the connector registers its extra visualization types into the catalogs, making them selectable. Disabling it removes them from the catalog — no dead code in the picker, no loss of capability.
- The operator's instance = the generic engine + every Vrooli connector pack enabled, which reconstructs today's board exactly. A third party = the generic engine + their own connectors + whatever bespoke visuals they or an agent author.

This is also the monetization seam: the free layer is the engine and the open catalogs; the paid layer is managed connector packs and hosted convenience, per `path:docs/concepts/PAID_FEATURES.md`.

## The settings surface and preview

The settings page edits the instance definition — connectors, signals, room presets, per-room theme/composition/beats, and slot bindings — and writes it back to the versioned outcome registry (not a hidden store; the instance stays inspectable and reviewable).

Preview is nearly free because **every signal already carries an authored `sample`** (value + series + basis; see `PROVENANCE-MODEL.md`). The editor renders a real room using the real `AmbientCanvas` and `BoardController` driven by sample values, in a contained frame, with **zero live data wired**. What you configure is literally what the war room shows — including the `sample` honesty ink, so preview never lies green about coverage or trust.

## The customization skill

The scenario ships a customization skill (distinct from the existing `command-center` *read* skill) that teaches an agent to extend a board, not just read it. It must cover:

1. **Connect a data source** — declare a connector, discover its response shape, define signals with select paths and TTLs.
2. **Compose a room** — clone a preset, pick a theme and composition, arrange beats, bind each beat's readouts to signals.
3. **Bind a background** — read a composition's slot manifest and bind signals to slots, with the shape rules.
4. **Author a new theme** — add a token file; no CSS or code.
5. **Author a new composition** — add a geometry module that declares slots, names no color, respects quiet zones and the tier ladder, and passes the first-frame render check (`CC-P1-003`). This is the deepest rung and the one that most needs worked examples.

The skill is the plug-in's selling point: it is what makes the instrument extensible-by-agent. It carries value the day it is written, independent of whether the plugin ramp has shipped.

## Milestones

Two shippable milestones, deliberately decoupled so the second never blocks the first.

- **Milestone 1 — data-driven engine + settings.** Draw the seam, lift the four welded couplings to data, ship the four catalogs, build the settings page with sample-driven preview. Acceptance: the **zero visual diff** gate passes on the operator's instance. Delivers all the internal "production-ready, organized, extensible" value.
- **Milestone 2 — skill + monetization packaging.** Author the customization skill, give Command Center a standalone install path (the universal prerequisite for any plugin monetization; see `path:docs/monetization/catalogs/channels/skill-registries.md`), and package for the scenario-to-plugin ramp. Gated on the ramp actually shipping — which is why it is a separate milestone.

## Resolved design decisions (2026-09-17)

These were settled with the operator and now bind implementation.

1. **Slot manifest — module export is truth; JSON is generated.** A composition exports its slot manifest next to its `draw()`; a build step generates the sidecar JSON the settings editor reads. One source of truth, no code/picker drift, and the editor never imports render code to discover slots.
2. **Signal shapes — four primitives, with typed `rows`.** The taxonomy stays `scalar | series | rows | meta`, but `rows` carries a declared column schema (e.g. `{ key, share, label? }`) so reach-map and release-ladder validate their required columns at bind time. No Vrooli-specific shape enters the engine taxonomy; bespoke structure stays inside connector packs.
3. **Connector pack format — code modules, in-tree and trusted for Milestone 1.** A pack registers its readouts/compositions into the catalogs via a code module; the Vrooli packs ship in-tree and are trusted. A sandboxed, data-only pack format for untrusted third parties is deferred to Milestone 2, where the plugin security baseline governs it. This keeps M1 unblocked without prematurely solving untrusted-code execution.
4. **Config layout — split per-catalog files.** The instance definition splits into `themes/`, `compositions/`, `rooms/`, and `connectors/` files rather than one registry blob. Each preset is independently reviewable, diffable, and clonable, which is how the catalogs are meant to grow. (If a single runtime read is later wanted, a generated merged bundle is an additive optimization, not a change to the authoring layout.)
5. **Zero-diff gate — a purpose-built harness, because the scenes are animated and seeded.** A naive pixel diff flakes: compositions animate continuously and lay out from a seed (star and orbit positions are seeded by room + metric id; see the panorama decision in `internal/DECISIONS.md`). The gate therefore **pins the seed and freezes animation time to fixed sample points**, then compares the resulting frames — reusing capture/compare plumbing from the BAS visual-check pattern (LPBS `visual-check.mjs`) but driven by this harness so "identical" is deterministic. Acceptance is worst-case across the pinned sample points (`CC-P1-005`), not an average or a single still.

### Consequences for the schema

- A composition module's public surface is `{ id, slots, tier behavior, quietZones, draw() }`; `slots` is the generated JSON's source.
- `rows` signals declare `columns`; a bind that omits a required column is a settings-time error.
- Connector packs are loaded, not sandboxed, in M1 — so pack code is reviewed like engine code until M2's data-only format exists.
- The seed and the frozen sample points are themselves pinned inputs to the gate and must be stable across runs, or the baseline is meaningless.

## Non-goals

Explicitly out of scope for this effort, to keep the seam clean and Milestone 1 shippable:

- **Multi-tenant hosting.** One instance per deployment. A second board for a second domain is a second deployment, not a tenant.
- **Arbitrary auth providers / a connector marketplace.** Connectors resolve through the existing credential authority; a marketplace waits on plugin-ramp telemetry proving demand.
- **A new visual language.** `DESIGN.md` (`vrooli-command-display`) still governs every surface. Generalization changes *how* the board is configured, never the honesty model, the layer model, or the motion registers.
