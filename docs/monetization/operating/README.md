# Monetization Operating

This folder holds the team-level operating contract for the `monetization` team.

## Operating Documents

| Document | Purpose |
|---|---|
| [`OPERATING_MODEL.md`](OPERATING_MODEL.md) | Mission, scope, operating loops, graph, topics, decisions, external inputs, outputs, feedback loop, gaps, and validation commands. |

## Validation

When prompt-manager PoR validation is wired, this folder is validated in two layers:

- the local [`../manifest.json`](../manifest.json) requires `operating/OPERATING_MODEL.md`;
- the operating-model validator checks the document's graph, topic catalog, decision catalog, inputs, outputs, feedback loop, gaps, and adoption commands.
