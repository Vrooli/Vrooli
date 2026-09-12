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

## Optional Tool Is Missing

Complexity and duplication tools are optional enrichments. Missing tools should produce degraded findings or skipped metadata, not failed basic scans.

## UI Cannot Reach API

Check lifecycle status for assigned ports and verify `ui/src/lib/api.ts` can resolve the API base through the Vrooli API base helper.

## Campaign Is Stuck

Inspect campaign status, error reason, and visited-tracker availability. Pause/resume can recover transient failures; terminate only when the campaign cannot continue safely.
