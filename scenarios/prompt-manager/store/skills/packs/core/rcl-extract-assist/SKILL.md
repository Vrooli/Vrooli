## Tools focus: RCL Extract Assist

Extract one component, hook, or other catalog asset into React Component Library. Refactor source scenario code when needed, then use the RCL CLI as the only catalog mutation authority.

## Boundaries

In scope: source inspection, source refactoring, RCL ingest, and no-regression verification for each touched scenario.

Out of scope: direct catalog storage changes, claiming a catalog mutation without CLI evidence, and starting another assisted workflow.

## Workflow

1. Record the current build and test result for each scenario you will touch. Existing failures are acceptable. The final state must add no failures.
2. Inspect the requested asset and its local imports. Decide whether it is a component, hook, or another supported catalog asset.
3. Refactor the source scenario as needed. Remove scenario-specific names, references, and coupling so the source is a clean reusable library target.
4. Run `react-component-library components ingest <scenario> <source-file> <slug>`. Use the source filename without its extension as the default slug.
5. Read the CLI result. A completed agent run is not proof that ingest committed.
6. Verify the catalog through the RCL CLI. Run the recorded build and test checks again for every touched scenario. Report the before and after result, the asset kind, the CLI mutation evidence, and any blocker.

## Requested operation

{{.operation}}

## Decision table

| Condition | Action |
| --- | --- |
| Source has scenario coupling | Refactor it before ingest. |
| Ingest CLI fails | Do not claim success. Report the failure and source state. |
| A check was already failing | Preserve it. Do not classify it as a new regression. |
| Post-change check introduces a failure | Fix it or return `blocked` or `needs_review`. |

## Output expectations

Return only this JSON object: `{"summary":"non-empty string","outcome":"completed|blocked|failed|needs_review","evidence":["string evidence"]}`. Include the catalog CLI result and before/after validation evidence.

## Troubleshooting & Edge Cases

If relative imports hide required companions, identify them before ingest and use the CLI companion-file option when appropriate. If the asset kind has no supported ingest surface, stop and report `needs_review`; do not mutate catalog files directly.
