---
name: experiential-ui-design
description: Design a scenario UI to be genuinely beautiful, intuitive, cohesive, and market-standout across desktop and mobile as co-equal media — design the medium (not the screen size), model content once and compose it twice, work mockup-first, and unify the surfaces under one concept. Not token/primitive infrastructure (ui-design-system-migration), not i18n (ui-i18n-adoption), not usability heuristics alone (ux).
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [steer, ux]
  targetDimensions: [visual, ui, accessibility]
  tags: [ui, ux, visual, mobile, responsive, design, mockup, layout]
  icon: sparkles
  status: active
  revision: 6
  createdAt: "2026-09-21T00:00:00Z"
  updatedAt: "2026-10-25T00:00:00Z"
  requires:
    scenarios: [prompt-manager]
    commands: [prompt-manager skill read]
  origin:
    kind: authored
---

## Steer focus: Experiential UI Design

Prioritize making the UI in `scenarios/{{TARGET}}/ui/` genuinely beautiful, intuitive, and cohesive across desktop and mobile as co-equal media — not a desktop layout reflowed to fit a phone.

Your goal is to design the medium and the experience — the information architecture, the composition, the interaction model, and the unifying concept per device — rather than to restyle components or add breakpoints to an unchanged layout.

Required reading:
- `prompt-manager skill read ux` — usability and responsive tactics (the mobile viewport test, touch targets, safe-area, spacing). This skill does not restate them; it decides the layout those tactics then refine.
- `prompt-manager skill read ui-design-system-migration` — the token and primitive substrate. It owns the design system; this skill composes with those primitives, it does not define them.
- `prompt-manager skill read polish` — final finish and consistency at R3, after the design holds.
- `prompt-manager skill read harness-goal-authoring` §6 "Convergence goals" — how to drive the build to genuine excellence without stopping at first green. Beauty has no mechanical oracle; the convergence loop plus the human checkpoints below are how "done" is decided.

### 1. Scope

In scope, within `scenarios/{{TARGET}}/ui/`:
- The information architecture of each surface: what data and actions it holds, and their priority.
- The layout composition for desktop and for mobile, chosen per medium.
- The interaction model per medium (bars, sheets, drill-in, command palette, full-screen).
- The unifying visual concept that makes the surfaces cohere and stand out.
- Mockups, and the procedural-versus-illustration asset strategy.

Out of scope (hand off to the named owner):
- Design tokens, primitives, and the design-system migration → `ui-design-system-migration`.
- Multi-language strings → `ui-i18n-adoption`.
- Pure usability heuristics, contrast ratios, and touch-target sizes → `ux`.
- Rough-edge finish once the design is settled → `polish`.
- Product capabilities, API, and data → implementation belongs to their owners. Identify
  missing behavior that prevents the approved task; include the enabling change when the
  assignment authorizes it, otherwise name the precise dependency. This boundary does not
  make a visually polished but incomplete workflow acceptable.

### 2. Core doctrine

**Design the medium, not the screen size.** Desktop and mobile are different media, not one screen at two widths. A responsive reflow that keeps the same component tree and only rearranges it is level-1; it cannot change the interaction model, and that is the ceiling most disappointing UIs hit.

| | Desktop | Mobile |
|---|---|---|
| Spatial budget | Wide, shallow — room across, one screen of height | Narrow, deep — one column, scroll is the main axis |
| Attention | Simultaneous — many panels at once | Sequential — one thing at a time |
| Input & reach | Pointer, hover, keyboard; uniform reach | Thumb, tap, swipe; bottom easy, top hard |
| Natural metaphor | Dashboard — see everything, act anywhere | Stack of cards — focus one, drill for more |

Desktop composes in **space** (put things side by side); mobile composes in **sequence** (put things in order, reveal on demand, keep primary actions in the thumb zone). Hover, side-by-side panels, and always-visible toolbars map to bottom sheets, tabs or swipe, and a FAB or thumb-bar.

**Model the content once, compose it twice.** List a surface's data and actions device-independently first, with no layout commitment. Both media must serve that whole list. Never remove content on mobile; re-sequence it. Rank the list by "what is the user here for, right now?" — the highest-ranked item, and where it fits the next one or two, must be reachable without scrolling on the mobile viewport that `ux` defines; the fold is a pressure to demote, not a countable gate. Demote the rest to scroll-below, a collapsed section, persistent chrome, or an on-demand sheet.

**Compose with components, not only CSS.**

```
                 Does only the arrangement change?
                            │
             ┌──────────────┴───────────────┐
           YES                              NO
            │                    (interaction model or the set
            ▼                     of components changes)
   CSS media / container query           │
   — reflow is correct here              ▼
                             Conditional composition: a breakpoint
                             hook (useBreakpoint / useIsMobile,
                             matchMedia, SSR-safe) selects different
                             components off the SAME data hook
```

Check whether the design-system library already exports such a hook before writing one. A bottom sheet that a next-step bar opens cannot be pure CSS; it is a different component, so it needs conditional composition.

**Work mockup-first, and reserve exactly two human checkpoints.** Produce side-by-side desktop and mobile mockups in `scenarios/{{TARGET}}/docs/mockups/` before writing product code. A human ratifies the **concept** and the **mockups** — those two only. Everything after is autonomous; drive it to genuine excellence with the convergence loop in `harness-goal-authoring` §6. Do not add a third human gate into the goal text (`harness-goal-authoring` §6 explains why a named reviewer makes agents stop early).

**Unify the surfaces under one concept.** A saturated market is not won with prettier cards; it is won with a coherent world. Choose one concept or metaphor and make every surface a vantage point within it. The concept is structure, not decoration — it drives copy, iconography, and layout, not just a color.

**Layer the assets: procedural first, illustration sparingly.** Star fields, gradients, glow, and theme washes belong in code or SVG — crisp at any size, animatable, theme-aware, tiny, and about ninety percent of the felt beauty. Build that layer before generating any art. Generated hero illustrations are a small set — one motif in a few framings — and are a nice-to-have, not a blocker. Keep glow procedural; it does not survive baking into a raster.

### 2a. Build toward the current design target

Keep one current target in the scenario's existing durable design document (§6):
selected mockups, applicable amendments, retained capabilities and verified refinements.
An atlas specifies the selected concept and its unamended details; it is not a command
to reproduce an awkward artifact or remove useful behavior missing from a screenshot.
Within product, safety and authority constraints, the latest applicable explicit user
decision takes precedence over an older design choice. Resolve an actual conflict with
a governing constraint explicitly. A question or agent preference is not user approval.

| Input | Action and authority |
| --- | --- |
| Explicit user amendment, such as retaining bottom navigation | Preserve the request and provenance; update the current target and affected assignments/review criteria. No replacement atlas or repeat concept ratification is required for a clear in-scope amendment. |
| Ordinary reversible refinement within the chosen concept and workflows | Use design judgment and verify it below. Record it as agent-authored; adopt when the evidence supports it. Do not require an operator decision for every spacing or component choice. |
| Ambiguous alternatives with a consequential tradeoff, changed concept, removed capability or changed authority | Preserve the working behavior, investigate and present the specific unresolved choice to its owner. Do not silently select by taste or claim human approval. |

Judge a proposed refinement against the same task, state, theme and viewport:

- **Task and hierarchy:** Does it make the main task clearer and reduce competing controls?
  Keep primary and frequent actions easy to find; secondary tools can use a labelled menu.
- **Task completeness and recovery:** Can users complete the approved task with realistic
  data, see state and recover errors? For searchable lists, distinguish a bounded page from
  an arbitrary total cap: search/filter the full authorized scope before pagination, expose
  continuation and an honest end, and cover important lifecycle states including outstanding
  work. A limitation banner does not repair unreachable records. Keep selection, query and
  return context through drill-in. A collapsed editor and its expanded controls may be
  complementary states, not alternative features.
- **Composition:** Does the whole screen have a clear attention order? Prioritize the task
  and distinguishing record content; subordinate routine freshness/scope metadata, use warning
  emphasis for actionable exceptional conditions, and group related filters. Match spacing,
  typography and density to that hierarchy across the product's surfaces.
  Use familiar icon buttons for unambiguous compact actions such as menu, close and
  overflow, with accessible names and adequate targets. Do not replace clear primary,
  destructive or unfamiliar action labels with ambiguous symbols merely to reduce text.
- **Component fit:** Prefer a suitable governed shared component over custom CSS copying
  an obvious defect. Inspect its actual contract and behavior; library membership alone
  does not establish suitability. Change published assets through their owner workflow.
- **Access and device fit:** Verify keyboard/focus, labels, contrast, touch targets, zoom,
  responsive behavior and safe areas. Inspect the actual scrolling panes with long real
  content and machine identifiers; document width alone can miss inner overflow. Verify
  a coherent focus indicator on composed controls. A flush drawer removes an unnecessary *outer gutter*;
  it still needs readable inner padding and viewport inset handling.
- **Evidence and tradeoff:** Show the concrete gain and any cost with before/after captures
  and operated behavior. A cleaner screenshot alone cannot prove a better interaction. Operate a realistic
  large dataset, older match, outstanding item, no-match and continuation failure when those
  cases apply; a small attractive fixture and tests that enshrine a cap are insufficient.

Use the existing target's decision/amendment section, not another feedback manager. Give
each material amendment/refinement a stable ID, source/author/date, affected surfaces and
states, intended change and preserved behavior, rationale/tradeoff, evidence, disposition
(proposed, adopted, rejected or superseded) and any superseded ID. Use descriptive or
author-scoped IDs, check for duplicates during parallel edits, and preserve concurrent
authored entries; do not race to allocate one shared sequential number. Preserve whether a
decision is user-approved or agent-authored; do not relabel an untested proposal adopted.
Record routine finishing adjustments together when they share a rationale. Use independent
review for material refinement judgments within the existing acceptance route; don't add
a human gate to routine execution. If verification disproves the benefit, revise or revert.

Send the current target and applicable IDs to the actual implementer and reviewer. Reconcile
active briefs and eligible native completion predicates when an amendment changes acceptance;
verify receipt and application, rather than counting a feedback-file write as adoption.
Keep worker-owned and native authority boundaries intact.

Compare the served result against this effective target on desktop and phone for responsive
web UIs, with applicable themes, states and accessibility checks. The existing comparison
ledger records the source/build identity, reference, amendment/refinement ID, expected
difference and remaining defect. An implemented, verified amendment or refinement is an
expected difference and does not block completion. A missing requirement, unverified change
or genuine defect remains open. Keep original images and decision history; update only
representative mockups when useful. Do not require a full atlas regeneration or relax the
whole product's quality bar merely to close a mismatch count.

### 3. Archetype model — classify each surface, then compose

Walk this table per surface. Two agents reading the same surface must land in the same row.

| Surface archetype | Signal | Dominant risk | Mobile composition |
|---|---|---|---|
| Dashboard / multi-panel | Several panels side by side | Worst desktop→mobile mismatch | Bespoke mobile tree: fold-first content, thumb-zone action bar, overflow into sheets |
| List / feed | Repeating rows | Low — already mobile-native | CSS reflow to one column plus a FAB; do **not** over-build |
| Form / settings | Many fields | A long-scroll wall of inputs | Guided sequence, or summary rows that drill into sub-screens |
| Immersive single-task | One focal action | Desktop wastes space | Mobile may exceed desktop: full-screen, chrome removed |
| Data table | Columns | Horizontal-scroll trap | Card list; surface the one key metric per row |

Effort is proportional to the mismatch: dashboards and immersive tasks earn bespoke mobile trees; lists and stable forms do not. Recognizing which surfaces need less is part of the skill.

A **primary surface** is a top-level route the user navigates to directly — an entry in the app's main navigation or router — not a modal, drawer, or sub-screen reached only by drill-in. The maturity ladder (§4) gates on primary surfaces.

### 4. Experiential maturity ladder

This skill owns the ladder. Every rung is gated by a verifiable artifact, run against `scenarios/{{TARGET}}/`. Record the current rung and evidence in the durable doc (§6).

| Rung | What exists (verify) | When to stop here |
|---|---|---|
| **L0 Reflow-only** | `rg -l -e useBreakpoint -e useIsMobile -e useMediaQuery scenarios/{{TARGET}}/ui/src` is empty **and** `scenarios/{{TARGET}}/docs/mockups/` is absent. One component tree, CSS media queries only. | Never — this is the disappointing baseline. |
| **L1 Mockups + concept** | Side-by-side desktop and mobile mockups for every primary surface under `scenarios/{{TARGET}}/docs/mockups/`; the unifying concept is named in the scenario's durable design doc (§6). | The design is agreed but not built. Stop here only to get the two human ratifications. |
| **L2 Medium-composed** | The breakpoint hook exists (`rg -e useBreakpoint -e useIsMobile scenarios/{{TARGET}}/ui/src` is non-empty) and every Dashboard/Immersive/Table archetype surface renders distinct desktop and mobile component trees, not one reflowed tree. | Composition is real but the concept and assets are not yet applied everywhere. |
| **L3 Concept + asset system** | The unifying concept is applied on every primary surface; the procedural visual layer exists as a dedicated component or stylesheet under `scenarios/{{TARGET}}/ui/src/` (for example a `*Backdrop`, `*Scene`, or `theme/*.css` module the primary surfaces render), confirmable by `rg` or `ls`; an image brief exists at `scenarios/{{TARGET}}/docs/mockups/image-generation-brief.md` when illustration is used. | The world is coherent; visual parity is not yet verified against the mockups. |
| **L4 Beauty verified** | Desktop and mobile captures (and Day/Night when themed) exist and are compared to the current design target (§2a), with expected differences linked to adopted amendments/refinements and no unresolved required visual or behavior findings; the selected concept and mockups carry their human ratification. | Ceiling. Stop. Adding ornamentation past the concept is a regression (`improvement-do-and-dont`), not a higher rung. |

The terminal "is it genuinely beautiful?" judgment has no mechanical oracle. The two human checkpoints establish the concept and selected mockups; §2a governs later amendments and refinements. L4 verifies the current target, operated behavior and comparison evidence; exact pixel-match totals are not a substitute for judgment.

This L0–L4 ladder is the *experiential* axis; it is distinct from the R0–R4 scenario maturity ladder (`scenario-maturity-ladder`). A surface can be R3-mature yet L1 on experience.

### 5. Anti-gaming

- Reflow is not composition. Do not claim a mobile design from added breakpoints while the component tree and interaction model are unchanged (that stays L0/L1).
- Do not remove content to "fit" mobile. Missing content is a different, worse product; re-sequence instead.
- Do not add visual ornamentation unmoored from the concept to look "designed". Decoration that the concept did not call for is a finding, not progress.
- A screenshot that renders is not a design that is beautiful. Judge captures against the current design target (§2a), not against "it loads". Neither personal taste nor a shared-component label excuses lost behavior or an unverified deviation. The gaming definitions live in `improvement-do-and-dont`; cite it.

### 6. Memory management

Findings, the chosen concept, and the current maturity rung land in the scenario's durable design doc — `scenarios/{{TARGET}}/docs/ARCHITECTURE.md` or the closest existing design document — via `knowledge-observatory-tools`. Mockups and the image brief live under `scenarios/{{TARGET}}/docs/mockups/`. Do **not** create a standalone `*_AUDIT.md`; it freezes one session's view and rots. Read the durable doc and the existing mockups at session start so you extend the design rather than restart it.

### 7. Output expectations

You may:
- Add or update mockups under `scenarios/{{TARGET}}/docs/mockups/` and the concept in the scenario's design doc.
- Add a breakpoint hook and per-medium component trees in `scenarios/{{TARGET}}/ui/src/`.
- Add the procedural visual layer and an image-generation brief.

You must:
- Design the content model once and compose it for both media before writing product code.
- Classify each surface with the archetype table and match effort to the mismatch.
- Keep the guidance transferable — use `{{TARGET}}`, never a hardcoded scenario name.

You must NOT:
- Restyle components without addressing composition and interaction model.
- Edit tokens or primitives owned by `ui-design-system-migration`, add features, or change data contracts.
- Add a third human review gate into a convergence goal's text.

### Troubleshooting & Edge Cases

- **"It's responsive but doesn't change enough."** The component tree is frozen and only CSS reflows. This is L0/L1. Add the breakpoint hook and compose the high-mismatch surfaces as distinct trees (L2).
- **A utility surface feels over-themed.** The concept was applied as decoration to a surface that did not need it. Lists and stable forms take restraint, not the full treatment (§3).
- **The concept reads as a skin, not a structure.** It was added as color and imagery only. Push it into copy, iconography, and layout, or drop it.
- **Beauty stalls because tests are green.** Green is not done for a design (§4, L4). Drive the build with the `harness-goal-authoring` §6 convergence loop.
