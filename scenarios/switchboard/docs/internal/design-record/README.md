# Design Record

## Purpose Of This Document

Preserve the design investigation that produced this scenario, so a later agent
can recover *why* a decision was made without the conversation that made it.
These are primary sources. They are dated, they are not maintained, and where
they disagree with `path:../../../PRD.md`, `path:../../concepts/EXPERIENCE.md`,
or `path:../../../experience/`, those win.

## Contents

| File | What it records | Dated |
|---|---|---|
| `strategy-and-architecture.html` | The reframe from a voice-only phone scenario to a multi-channel presence plane. Channel economics with measured costs, the descriptor-and-adapter extensibility model, the group-chat hazard set, the channel-versus-tool split, ecosystem-fit placement, monetisation posture, and the name decision. | 2026-09-01 |
| `experience-design-spec.html` | The UX design spec: the two-posture thesis, information architecture, the three visual encodings, high-fidelity mockups of every surface, the state and refusal-copy contracts, and the react-component-library gap analysis. | 2026-09-01 |

Open either file in a browser. They are self-contained HTML with no external
dependencies beyond web fonts.

## Provenance

Both were authored during the design conversation that produced this scenario,
published as Claude Artifacts, and copied here so the reasoning survives the
conversation. The published copies are at:

- `https://claude.ai/code/artifact/dfc3ec00-398a-43a6-b2aa-d6f5bd14b193`
- `https://claude.ai/code/artifact/7a71ac3d-b4d3-4794-a13c-5c4275d82fec`

## What They Are Not

They are not a plan, and they are not a contract. Three claims inside them were
already superseded during the same conversation and are corrected here so nobody
re-derives the wrong conclusion from a primary source:

1. **`notification-hub` is not an anti-pattern.** Its hardcoded channel switches
   are the deliberate result of a dated decision — *2026-08-17: "Ship one real
   channel end to end before building any abstraction"* — and its own decision
   log already names the target: *"Adding a channel touches this scenario's
   adapter registry and nothing else."* The registry does not exist in code yet.
   This scenario is the trigger that builds it.
2. **`prompt-injection-arena` is not a usable dependency.** It is stale, and it
   is shaped as an offline tournament that ranks models on a leaderboard, not as
   a runtime guard. Any use of it requires its own redesign with its own
   operational targets.
3. **The `Message` component in react-component-library is not a chat message.**
   Its catalog id is `ai.message` and its contract is an AI-console transcript
   card. See `path:../../reference/component-library-gaps.md`.

## Cross-References

- `path:../../concepts/EXPERIENCE.md` — the maintained UX contract
- `path:../../reference/component-library-gaps.md` — the maintained gap report
- `path:../../../experience/README.md` — the machine-readable experience contract
