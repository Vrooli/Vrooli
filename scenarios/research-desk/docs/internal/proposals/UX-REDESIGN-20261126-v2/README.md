# Research Desk — Redesign 02

Built from the three reference images the operator supplied on 2026-11-26: the Insuriva
tablet dashboard, the fashion shopping phone, and the nutrition tracking phone.

- **[index.html](index.html)** — the visual study. Light canvas, white cards, electric blue,
  lime signal, black pills, big numbers, soft grey tiles. Phone-readable; the desktop previews
  scroll horizontally on a phone.
- **[PROPOSAL.md](PROPOSAL.md)** — the written study: what the references say, the token system,
  three structural models (Daylight, Index, Week), feature parity, build plan and the decisions.

## The three structures

| | What it is | Best for |
|---|---|---|
| **A · Daylight** | Blue hero, live rings, week strip, bento desktop, focus-deck inbox. | The recommended default: the fastest five-second answer and the fastest 19 decisions. |
| **B · Index** | White feed, pastel hero, black pills, two-column topic grid. | The quietest, most premium version; best for reading and browsing. |
| **C · Week** | Seven-day strip, Gantt-like commission blocks, now line. | An optional Week view for overlap and drift. |

## What is different from the first study

The first study (`../UX-REDESIGN-20261126/`) used a dark instrument / editorial serif / space-map
language. The operator rejected it as AI slop. This study keeps the product model and the 34
feature groups, and replaces the entire visual language with the one in the references.

## Honesty labels

- The phone and desktop screens in `index.html` are **`sketch`**: HTML/CSS, not RCL components.
- The build target is **`rendered`** in RCL; the proof is **`composited`** through EX11.
- The current-state captures and the 34-group inventory remain in the first study, linked from
  the footer.

## Status

**Implemented 2026-11-26 (A · Daylight).** The phone home, inbox focus/list, commission ring
hero and segmented sections, launch rail, topic grid and dossier, reviews, runs, desk and file
surfaces are built in `scenarios/research-desk/ui`. The shared-library work the redesign needed
is recorded in [DESIGN.md](../../../../DESIGN.md) and
[component-library-gaps.md](../../../reference/component-library-gaps.md): Card 1.4.0 variants,
Tabs 1.3.8 segmented surfaces, the page-section className consumer, and the dev/test resolver's
`component.json.entry` fix. The visual study in `index.html` remains `sketch`; the served UI is
the `rendered` proof and the captures under `/tmp` during delivery are composited evidence, not
a committed baseline.
