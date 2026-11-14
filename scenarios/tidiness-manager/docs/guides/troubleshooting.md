# Troubleshooting

## Quality Phase Fails

Run:

```bash
quality-health audit run tidiness-manager --commands --autofix-preview --json
```

Fix the reported config contract. Do not weaken strict TypeScript, ESLint, Go lint, Makefile, or testing handler settings.

## Tidiness Phase Fails

Run:

```bash
tidiness-manager scan tidiness-manager --type tidiness --json
```

Then inspect the API logs and `docs/internal/SEAMS.md` to locate whether the failure is input normalization, scanner execution, persistence, or output mapping.

## Code Facts Evidence Is Missing

Check Code Facts and the Go/TypeScript/JavaScript graph providers through their managed lifecycle. A missing provider or incomplete graph must surface as degraded/skipped evidence, never as a clean zero-duplicate result. JavaScript has one prompt-manager census whose rates bind to the authoritative source/config hashes from its fast Code Facts `parse_units` response; full graph recommendation serving remains unverified, and the frame is not a fleet estimate. TypeScript has no source-replayed census yet.

## UI Cannot Reach API

Check lifecycle status for assigned ports and verify `ui/src/lib/api.ts` can resolve the API base through the Vrooli API base helper.

## Campaign Is Stuck

Inspect campaign status, error reason, and visited-tracker availability. Pause/resume can recover transient failures; terminate only when the campaign cannot continue safely.
