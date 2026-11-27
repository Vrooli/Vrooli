> **Superseded by [`../UX-REDESIGN-20261126-v2/`](../UX-REDESIGN-20261126-v2/README.md).** The operator rejected this study's dark instrument / editorial serif / space-map language as AI slop. It is kept for its current-state captures and the 34-group feature inventory.

# Research Desk UX redesign study — 2026-11-26

Three complete redesign directions for `research-desk`, a recommendation, a 34-group feature
parity map, and the path through the experience-engineering pipeline.

- **[index.html](index.html)** — the visual study. Open it in a browser; it is self-contained
  apart from the five `captured` PNGs in `assets/`. Designed to be read on a phone.
- **[PROPOSAL.md](PROPOSAL.md)** — the canonical written study: diagnosis, the three directions,
  screen-by-screen treatment, parity, proof plan, acceptance measures and decisions needed.

## What is in the room

| File | What it is |
|---|---|
| `index.html` | The comparison page. Decision first, today captured beside three proposals, one card per direction, parity summary, pipeline. |
| `PROPOSAL.md` | The full study. Read §2 for the diagnosis, §4 for the directions, §9 for every feature. |
| `assets/captured-now-390.png` | Current Now screen, 390×844, dark. `captured`. |
| `assets/captured-now-light-390.png` | The same screen in the light default. `captured`. |
| `assets/captured-now-1440.png` | Current Now screen, 1440×1200, dark. `captured`. |
| `assets/captured-now-light-1440.png` | The same screen at 1440×1200 in light. `captured`. |
| `assets/captured-needs-you-390.png` | Current Needs you feed. `captured`. |
| `assets/captured-topics-390.png` | Current Topics. `captured`. |
| `assets/captured-launch-390.png` | Current Launch. `captured`. |

The phone screens inside `index.html` are **`sketch`**: CSS compositions, not RCL components.
The build target is **`rendered`** in RCL, and the design proof is **`composited`** through EX11.
No picture on the page is labelled more faithfully than it is.

## Recommendation

Build **A · The Bridge**. Keep **B · The Desk**'s briefing voice inside "Since last look" and the
idle states. Ship **C · The Constellation** as an optional desktop map after the core is proved.

## Status

Proposal only. No scenario code, API, supervisor file or commission was changed. The current
desk keeps working exactly as it does now.
