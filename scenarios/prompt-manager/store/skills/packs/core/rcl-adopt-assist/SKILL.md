---
name: "rcl-adopt-assist"
description: "Adopt a React Component Library asset into a scenario with integration and baseline-aware verification."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["contract"]
  tags: ["react-component-library","catalog","adopt"]
  icon: "package"
  status: "active"
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
## Tools focus: RCL Adopt Assist

Adopt one React Component Library component, hook, or other catalog asset into a target scenario. Integrate scenario code freely, but use the RCL CLI as the only catalog mutation authority.

## Boundaries

In scope: target integration, local usage replacement, imports and theme wiring, RCL adoption, and no-regression verification.

Out of scope: direct catalog storage changes, claiming adoption without CLI evidence, and starting another assisted workflow.

## Workflow

1. Record build and test results for every scenario you will touch. Existing failures are a baseline, not a success condition.
2. Inspect the selected catalog asset and target scenario integration points.
3. Update target scenario code. Replace local usages when appropriate. Keep imports, theme wiring, and asset-specific integration correct.
4. Run `react-component-library adoptions preflight <component-id> <scenario>` and inspect the complete adoptability verdict. If tokens are missing, run `react-component-library adoptions tokens-sync <scenario> --dry-run`, review collisions, then apply the sync before retrying preflight.
5. Link the selected pinned version with `react-component-library adoptions link <component-id> <scenario>`. The link must install the managed locale bridge and derived English keys, and must generate and compose `ui/src/consts/selectors.library.ts` into the adopter selector registry.
6. If the published contract and `className` restyle seam cannot express a deliberate local behavior, stop for review and use `react-component-library adoptions eject <component-id> <scenario> --reason "..."`; ejection is the only source-writing exception.
7. Read the CLI result. A completed agent run is not proof that adoption committed. Run `react-component-library adoptions obligations <scenario> --json` and verify the token, locale-bridge, and selector dimensions.
8. Verify the adoption through the RCL CLI. Re-run the recorded checks and show no new build or test failures in touched scenarios.

## Requested operation

{{.operation}}

## Decision table

| Condition | Action |
| --- | --- |
| Target has a compatible local use | Replace it and verify imports. |
| Target needs theme or provider wiring | Add the integration before final validation. |
| Adoption CLI fails | Do not claim success. Return evidence for the failure. |
| Published contract is insufficient | Require a reasoned `adoptions eject`; never fall back to an untracked copy. |
| Preflight has unsatisfied tokens | Run `react-component-library adoptions tokens-sync <scenario>`, resolve collisions, and rerun preflight; do not hide the finding. |
| Post-change check adds a failure | Fix it or return `blocked` or `needs_review`. |

## Output expectations

Return only this JSON object: `{"summary":"non-empty string","outcome":"completed|blocked|failed|needs_review","evidence":["string evidence"]}`. Include the adoption CLI result and before/after validation evidence.

## Troubleshooting & Edge Cases

If a target file already exists, use explicit CLI overwrite controls only when the request authorizes replacement. If the requested catalog asset cannot be integrated without an unreviewed behavioral change, return `needs_review`; do not bypass the RCL CLI.
