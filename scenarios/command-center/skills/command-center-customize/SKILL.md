---
name: "command-center-customize"
description: "Extend a Command Center board — connect a data source, compose a room, bind a background, author a theme or composition — as data, under the zero-visual-diff invariant."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  status: "active"
  revision: 1
  createdAt: "2026-09-17T00:00:00Z"
  updatedAt: "2026-09-17T00:00:00Z"
  requires:
    scenarios: ["command-center"]
    commands: ["command-center board", "command-center room", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Practice focus: Command Center Customization

Extend a Command Center board by editing data, never by editing the engine. Every board is a set of split config files — connectors, signals, rooms, themes, compositions, readouts — plus optional connector packs that carry bespoke visuals. The shipped settings surface edits the board's real section model: rooms cycle through ordered beats, each beat has a hero, dwell, layout, and compatible metric bindings. Add capability by adding a file and binding it; prove you added nothing to the operator's own board by the zero-visual-diff gate.

This is the *extend* skill. The base `command-center` skill *reads* the instrument; this one *changes what it renders*. Read the customization contract before you start: it owns the seam, the vocabulary, and the resolved design decisions.

Required reading:
- `path:scenarios/command-center/docs/concepts/CUSTOMIZATION-MODEL.md` — the seam (engine vs. instance), the seven layers, the slot-binding schema, the resolved decisions. This skill executes that contract; it does not restate it.

Read when the task reaches them:
- `path:scenarios/command-center/docs/concepts/COVERAGE-MODEL.md` and `PROVENANCE-MODEL.md` — the honesty axes a signal carries (coverage, trust, sample). A new signal that skips these lies green.
- `path:scenarios/command-center/docs/concepts/UI-ARCHITECTURE.md` — the Full/Reduced/Still tier ladder and the quiet-zone contract a new composition must respect.
- `prompt-manager skill read command-center` — to read the instrument you are extending.

---

## Scope boundaries

**In scope:** adding or editing files under `path:scenarios/command-center/config/` (connectors, signals, rooms, themes, compositions, readouts); adding a scene module and its bespoke readout under `path:scenarios/command-center/ui/src/scenes/` and `ui/src/components/`; declaring a connector pack under `ui/src/packs/`; running the local build gates that validate those changes.

**Out of scope:**
- The rendering engine, the honesty model, the board cycler, the tier ladder, the motion registers. You configure the engine; you do not rewrite it.
- The visual language. `path:scenarios/command-center/DESIGN.md` (`vrooli-command-display`) still governs every surface. Customization changes *how a board is configured*, never the honesty ink or the layer model.
- Multi-tenant hosting, a connector marketplace, and untrusted data-only pack execution — deferred, per the contract's Non-goals.
- Improving this instrument's own effectiveness — that is `prompt-manager skill read command-center-improve`.

---

## The one invariant that governs everything

**Zero visual diff on instance zero.** The operator's own board is the first instance, expressed as the shipped default config plus the enabled Vrooli pack. Any change you make must leave that board rendering pixel-for-pixel as before. A change that is "more general" but alters what the operator sees is wrong.

This is a destination, not a style. It is verified — not judged — by the seed-pinned, time-frozen harness at `path:scenarios/command-center/ui/src/lib/visualDiff.ts` (`VISUAL_DIFF_PIN` pins the seed and the frozen sample timestamps). Adding a new theme, room, signal, or composition **as a new file** cannot regress instance zero, because instance zero binds none of them. Editing an *existing* default file can. Route every change down the first column of this table:

| Your change | Touches instance zero? | Gate |
|---|---|---|
| Add a new `config/<catalog>/<id>.json` the default rooms do not reference | No | Schema-valid file + preview renders |
| Add a new scene module / bespoke readout in a **new** pack | No | `pnpm slots:check` + preview |
| Add a beat, signal, or bind to a **cloned** room preset | No | Bind validates + preview |
| Edit a **default** theme/room/composition/signal file, or the engine | **Yes** | Zero-visual-diff harness must hold before the change is accepted |

If a change lands in the bottom row, stop and treat the harness result as the acceptance gate, not an afterthought.

---

## The layers, in one orientation table

The contract defines seven separately-swappable layers. This table is orientation only — the contract is authoritative. The load-bearing fact: **a connector is plumbing, a room is presentation, and they meet only at a signal id.**

| Layer | File you author | Joined to |
|---|---|---|
| Connector | `config/connectors/<id>.json` — base URL + auth ref + transport | — |
| Signal | `config/signals/<id>.json` — one measurement (a connector + select path + TTL + shape) | a connector |
| Theme | `config/themes/<id>.json` — palette tokens only, no geometry | — |
| Composition | `config/compositions/<id>.json` (+ generated `.slots.json`) — animated background; declares slots, names no color | slots ← signals |
| Readout | `config/readouts/<id>.json` — a foreground tile keyed by a signal's `kind` | a signal |
| Room | `config/rooms/<id>.json` — theme + composition + ordered beats + `bind` | theme, composition, signals |

Persistence: the Settings page writes each edit through `PUT /api/v1/catalogs/{catalog}/{id}`, which validates and atomically rewrites `config/<catalog>/<id>.json`. Editing the file directly and editing it through Settings are the same act on the same source of truth. `config/catalog.schema.json` is the shape contract for every entry.

## Settings-to-agent handoffs

The operator-facing flow deliberately stops at trusted code boundaries. When a request needs a new layout, custom readout/component, or upstream source, use the **Hand to an agent** control. It copies a prompt that names the target and this skill. The agent must preserve the section model, use compatible shapes, keep source attribution and honesty fields, and run `pnpm --dir ui slots:check`, `node ui/scripts/verify-room-bindings.mjs`, focused tests, and the visual gate before handing back the change.

---

## The extension process

Five phases, ordered by depth. Each is independently valuable and stops cleanly. Do only the phases your task needs; a new room from existing signals needs only Phase B.

```
Connect ──▶ Compose ──▶ Bind ──┬──▶ (need a new palette?) ──▶ Author theme
 (A)         (B)        (C)     │
                                └──▶ (need a new motion?) ──▶ Author composition (E)
                                                                   │
                                                       (rejected: color-in-scene,
                                                        unbound crash, tier regress)
                                                                   ▼
                                                            back to E, fix
```

### Phase A — Connect a data source

**Entry:** a live endpoint exists that returns the value you want to show.

**Actions:**
1. Add `config/connectors/<id>.json`: the transport (`http`, `connect`, or `graphql`), the base-URL resolution, and the auth reference. Resolve credentials through the existing authority; never inline a secret.
2. For each measurement the connector yields, add `config/signals/<signalId>.json` with `source.binding`, `source.read` (the path), `source.select` (the value to pull out), `source.ttlSeconds`, and the `shape` (`scalar | series | rows | meta`).
3. Set the honesty axes: `coverage` (per `COVERAGE-MODEL.md`) and `source.sourceTimePolicy`. A signal without a producer time policy cannot support a current-success claim.
4. Author a `sample` on the signal (value + series/rows + `basis`). This is not optional decoration — the Settings preview renders the real engine from `sample` with zero live data, so a signal with no sample cannot be previewed and its honesty ink cannot be checked.
5. For a `rows` signal, declare `columns` (e.g. `{ key, value, share }`); a room that binds it will validate those columns.

**Exit:** the signal file passes `config/catalog.schema.json`; the connector resolves when enabled; the signal appears in the Settings signals catalog with its sample rendering.

**Artifacts:** `config/connectors/<id>.json`, one or more `config/signals/<id>.json`.

### Phase B — Compose a room

**Entry:** the signals the room will show already exist as files.

**Actions:**
1. Clone the closest existing preset in `config/rooms/` rather than starting empty — it carries the working `title`, `category`, `theme`, `composition`, and `beats` shape. Give the clone a new `id`.
2. Pick a `theme` (a themes-catalog id) and a `composition` (a compositions-catalog id).
3. List the room's signals in `metricIds`, and order the `beats`. Each beat names a `hero` signal id and a `dwellSeconds`; a beat may override `composition` for its segment (as the Forge's release-ladder beat swaps to `funnel-cascade`).
4. Leave `bind` for Phase C.

**Exit:** the room renders in preview through the real `BoardController` and `AmbientCanvas`, cycling its beats, driven by each hero signal's sample.

**Artifacts:** `config/rooms/<id>.json`.

### Phase C — Bind a background to data

**Entry:** the room names a composition, and that composition declares slots.

This is the core act: it turns "the background reacts to data" from bespoke wiring into configuration. Three rules, enforced by `validateRoomBinding` (`ui/src/lib/catalogs.ts`):

1. **The room binds; the composition never names a signal.** In `config/rooms/<id>.json`, set `bind: { "<slotName>": "<signalId>", … }`. Read the composition's slots from `config/compositions/<comp>.slots.json` to learn the slot names, shapes, and required columns.
2. **Shape must match.** Binding a `scalar` signal to a slot whose shape is `rows` is a save-time error, not a runtime surprise. Match the signal's `shape` to the slot's `shape`.
3. **Every required column must exist.** If a slot declares `columns`, the bound signal must declare each non-`optional` column. Missing one is a save-time error naming the column.

An unbound slot does not break — it degrades to its `whenUnbound` visual (`decorative`, `calm`, `empty`, `globe-only`, …). A composition with no bindings renders as decoration, never as a blank canvas.

**Exit:** `validateRoomBinding` returns `null` for the room (the Settings Save button enables only when it does), and the background visibly reacts to the bound signals' samples in preview.

**Artifacts:** the `bind` map inside `config/rooms/<id>.json`.

### Phase D — Author a theme

**Entry:** no existing palette fits, and the gap is color only — not geometry.

**Actions:**
1. Add `config/themes/<id>.json` carrying only tokens: `gradient`, `primary`, `accent`, `glow`, `cornerRadius`, `haloWidths`, and the `tokens` map of `--color-*` CSS variables. Name no geometry and no motion.
2. Run `pnpm theme:generate` (from `ui/`) to regenerate the CSS from the token file. The theme is not live until this runs.
3. Reference the theme id from a room's `theme` field.

**Exit:** `pnpm theme:generate` succeeds; a room set to the new theme recolors in preview; compositions in that room recolor within one palette read (~250 ms) with no scene edit, because color is read from CSS variables at draw time.

**Artifacts:** `config/themes/<id>.json` + regenerated theme CSS.

### Phase E — Author a composition (the deepest rung)

**Entry:** you need a *new motion*, not a new palette — an animated background no existing composition provides.

A composition is a geometry module. Its slot manifest JSON is **generated from the module**, so the module — not the JSON — is the source of truth.

**Actions:**
1. Add a scene module under `ui/src/scenes/` that exports a `draw()` and a slot manifest (`{ <slotName>: { shape, role, whenUnbound, columns? } }`). Register it in `ui/src/scenes/index.ts`.
2. **Name no color.** Draw with `palette.primary` / `palette.accent` / etc. passed in at draw time. A composition that hardcodes a color breaks the theme invariant (see anti-patterns).
3. **Read only slots, never a global signal id.** Read `slot("<name>")`; never `read(data, "throughput_stats")`. All addressing lives in the room's `bind`.
4. **Declare a `whenUnbound` for every slot** so an unbound composition degrades to a defined visual, not a blank frame.
5. Respect the quiet-zone contract and the Full/Reduced/Still tier ladder in `UI-ARCHITECTURE.md`. The scene must render a correct first frame with no animation elapsed.
6. Regenerate and verify the sidecar: `pnpm slots:check` (runs `slotManifests.test.ts`) proves the generated `.slots.json` matches the module's exported manifest. Drift here fails the build.
7. **Bespoke vs. generic — decide before you register.** Use the table below. A composition specific to one connector's data (an earth-arc map, a release ladder) ships **inside that connector's pack** (`ui/src/packs/<pack>.ts` — its `compositions` and `readouts` maps), so it registers only when that connector is enabled and disappears from the picker when it is disabled. A generic composition (an orbital field, a funnel) ships in the engine catalog.

| Question | If YES | If NO |
|---|---|---|
| Does it only make sense for one specific connector's data shape? | Ship it in that connector's pack | Continue |
| Is the motion reusable for any signal of its slot's shape? | Ship it in the engine catalog | Ship it in a pack, conservatively |

**Exit (falsifiable — any failure routes back into Phase E):**
- `pnpm slots:check` passes (manifest matches).
- The scene renders a correct first frame with no bound slots (proves `whenUnbound` degradation).
- Swapping the room's theme recolors the scene with no scene edit (proves color-at-draw-time).
- `pnpm test` and `pnpm type-check` pass.

**Artifacts:** the scene module, its registration, its generated `.slots.json`, and — if bespoke — the pack entry.

---

## Anti-patterns

| Anti-pattern | Why it fails | Better approach |
|---|---|---|
| A composition hardcodes `read(data, "throughput_stats")` | Welds one background to one signal; unusable by any other board | Declare a slot; let the room `bind` a signal to it |
| A composition names a color (`#8aa8ff`, `"cyan"`) | Breaks the zero-diff theme invariant; the room can no longer recolor it | Draw with the `palette` passed at draw time; name no color |
| A new signal ships with no `sample` | Cannot be previewed; its coverage/trust ink cannot be checked; hides behind live data | Author a `sample` with a truthful `basis`; preview is the honesty check |
| Binding a `scalar` signal to a `rows` slot to "make it fit" | Save-time validation error; or worse, a scene that reads a shape it cannot use | Match the signal shape to the slot shape; add a correctly-shaped signal if none fits |
| Editing the `.slots.json` by hand | It is generated; the next build regenerates over your edit | Edit the module's exported manifest, then `pnpm slots:check` |
| Adding a Vrooli-shaped composition to the engine catalog | Every third-party board then carries dead Vrooli code | Ship bespoke visuals in a connector pack that registers only when enabled |
| "Improving" a default room's colors or beats while generalizing | Regresses instance zero; conflates two efforts | Keep generalization pixel-identical; take visual changes to a separate, DESIGN-governed effort |

---

## Knowledge capture

Customization is investigative — you discover a connector's response shape, a composition's slot contract, a binding's failure. Preserve what you learn so the next agent does not re-derive it.

- Record durable findings (a connector's real response shape, a non-obvious bind, a tier-ladder constraint) in `path:scenarios/command-center/docs/internal/` alongside the existing `DECISIONS.md` / `PROBLEMS.md`. Do not create standalone `*_AUDIT.md` files.
- New authored artifacts (a signal's provenance, a composition's slot rationale) belong in the file itself: a signal's `basis`, a `description`, a protective comment on a non-obvious slot.
- When a customization pattern recurs across agents, that is the signal to promote it — a new preset, a CLI verb, or a pack — per `path:docs/agent-system/PROMOTION_LADDER.md`. Today there is no customization CLI verb; steps are file edits and local build gates. A recurring, stable edit path is the evidence that earns one.

---

## Output expectations

You may add or edit files under `config/`, `ui/src/scenes/`, `ui/src/components/`, and `ui/src/packs/`, and run the local build gates.

You must:
- Keep instance zero pixel-identical unless the task explicitly owns a DESIGN-governed visual change; route any default-file edit through the zero-diff harness.
- Give every new signal an honest `coverage`, a `sourceTimePolicy`, and a `sample`.
- Make every composition read slots and name no color.
- Regenerate derived artifacts (`pnpm theme:generate`, `pnpm slots:check`) rather than hand-editing them.
- Ship connector-specific visuals in a pack, not the engine.

You must not:
- Edit the rendering engine, the honesty model, or the tier ladder to make one board work.
- Hand-edit generated files (`.slots.json`, theme CSS).
- Fake coverage or trust to make preview read green.

---

## Troubleshooting & Edge Cases

| Symptom | Cause | Fix |
|---|---|---|
| Settings "Save" stays disabled with a red message | `validateRoomBinding` rejected the room: a bound signal is unknown, its shape mismatches the slot, or a required column is missing | Read the message; it names the slot, signal, and reason. Match the shape or add the column to the signal. |
| A new composition renders a blank background | A slot has no `whenUnbound`, or `draw()` threw on the first frame (e.g. a negative radius from a negative clock) | Declare `whenUnbound` for every slot; clamp time to non-negative; the engine falls back to the composed still on a failed draw. |
| `pnpm slots:check` fails after editing a scene | The generated `.slots.json` no longer matches the module's exported manifest | Regenerate; never edit the sidecar by hand. The module export is the source of truth. |
| A new theme does not change any color | `pnpm theme:generate` was not run after adding the token file | Run it from `ui/`; the theme is inert until the CSS is regenerated. |
| A bespoke readout/composition never appears in the picker | Its pack is not enabled | Enable the pack via `configureEnabledPacks` in `ui/src/scenes/index.ts`; a pack's visuals register only when its connector is enabled. |
| A background stops reacting after a signal rename | The room's `bind` still points at the old signal id | Update the `bind` map; a composition addresses data only through the room's binding. |
| The zero-diff harness reports a diff you did not intend | You edited a default file that instance zero renders | Revert to a pixel-identical form, or move the visual change to a separate DESIGN-governed effort; generalization must not alter the operator's board. |
