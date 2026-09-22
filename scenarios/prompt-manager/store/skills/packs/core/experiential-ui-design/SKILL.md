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
  revision: 1
  createdAt: "2026-09-21T00:00:00Z"
  updatedAt: "2026-09-21T00:00:00Z"
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
- New product capabilities, API, and data → out of scope; a design skill changes form, not features.

### 2. Core doctrine

**Design the medium, not the screen size.** Desktop and mobile are different media, not one screen at two widths. A responsive reflow that keeps the same component tree and only rearranges it is level-1; it cannot change the interaction model, and that is the ceiling most disappointing UIs hit.

| | Desktop | Mobile |
|---|---|---|
| Spatial budget | Wide, shallow — room across, one screen of height | Narrow, deep — one column, scroll is the main axis |
| Attention | Simultaneous — many panels at once | Sequential — one thing at a time |
| Input & reach | Pointer, hover, keyboard; uniform reach | Thumb, tap, swipe; bottom easy, top hard |
| Natural metaphor | Dashboard — see everything, act anywhere | Stack of cards — focus one, drill for more |

Desktop composes in **space** (put things side by side); mobile composes in **sequence** (put things in order, reveal on demand, keep primary actions in the thumb zone). Hover, side-by-side panels, and always-visible toolbars map to bottom sheets, tabs or swipe, and a FAB or thumb-bar.

**Model the content once, compose it twice.** List a surface's data and actions device-independently first, with no layout commitment. Both media must serve that whole list. Never remove content on mobile; re-sequence it. Rank the list by "what is the user here for, right now?" — the top one to three items must fit above the mobile fold without scrolling; demote the rest to scroll-below, a collapsed section, persistent chrome, or an on-demand sheet.

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

### 4. Experiential maturity ladder

This skill owns the ladder. Every rung is gated by a verifiable artifact, run against `scenarios/{{TARGET}}/`. Record the current rung and evidence in the durable doc (§6).

| Rung | What exists (verify) | When to stop here |
|---|---|---|
| **L0 Reflow-only** | `rg -l "useBreakpoint\|useIsMobile\|useMediaQuery" scenarios/{{TARGET}}/ui/src` is empty **and** `scenarios/{{TARGET}}/docs/mockups/` is absent. One component tree, CSS media queries only. | Never — this is the disappointing baseline. |
| **L1 Mockups + concept** | Side-by-side desktop and mobile mockups for every primary surface under `scenarios/{{TARGET}}/docs/mockups/`; the unifying concept is named in `scenarios/{{TARGET}}/docs/ARCHITECTURE.md` (or the scenario's design doc). | The design is agreed but not built. Stop here only to get the two human ratifications. |
| **L2 Medium-composed** | The breakpoint hook exists (`rg "useBreakpoint\|useIsMobile" scenarios/{{TARGET}}/ui/src` is non-empty) and every Dashboard/Immersive/Table archetype surface renders distinct desktop and mobile component trees, not one reflowed tree. | Composition is real but the concept and assets are not yet applied everywhere. |
| **L3 Concept + asset system** | The unifying concept is applied on every primary surface; the procedural visual layer is present in the UI source; an image brief exists at `scenarios/{{TARGET}}/docs/mockups/image-generation-brief.md` when illustration is used. | The world is coherent; visual parity is not yet verified against the mockups. |
| **L4 Beauty verified** | Desktop and mobile captures (and Day/Night when themed) exist and are compared to the mockups with no open parity or visual findings in the durable doc; the concept and mockups carry a human ratification. | Ceiling. Stop. Adding ornamentation past the concept is a regression (`improvement-do-and-dont`), not a higher rung. |

The terminal "is it genuinely beautiful?" judgment has no mechanical oracle; it routes to the two human checkpoints (concept, mockups). L4 gates only the verifiable process artifacts around that judgment — captures exist, compare to the mockups, and leave no open findings.

### 5. Anti-gaming

- Reflow is not composition. Do not claim a mobile design from added breakpoints while the component tree and interaction model are unchanged (that stays L0/L1).
- Do not remove content to "fit" mobile. Missing content is a different, worse product; re-sequence instead.
- Do not add visual ornamentation unmoored from the concept to look "designed". Decoration that the concept did not call for is a finding, not progress.
- A screenshot that renders is not a design that is beautiful. Judge captures against the mockups, not against "it loads". The gaming definitions live in `improvement-do-and-dont`; cite it.

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
