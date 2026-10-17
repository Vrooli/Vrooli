# Declared audience matches the build

Status: v1

Checklist item: `audience-matches-build`.

## What this checks

Whether the release satisfies the acceptance criterion for **Declared audience matches the build** in the deployment-manager readiness checklist.

## When it applies

Run it for a scenario and commit being prepared for sale. Use declared evidence and an attributable run or artifact.

## When it backfires

Do not infer a pass from a declaration alone, reuse evidence for another commit, or turn an unavailable producer into a pass.

## Failure mode

Record the mismatch, source, run identifier, timestamp, and exact checklist item. Unanchored observations remain unknown until directly evidenced.

See: `scenarios/deployment-manager/config/readiness-checklist.json`.

