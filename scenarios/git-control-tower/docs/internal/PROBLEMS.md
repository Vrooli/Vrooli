# Internal Problems: git-control-tower

## Stability Issues

- ESLint React stability config added with safety-critical rules and import cycle detection.
- TypeScript safety rules enforced (strict + noUncheckedIndexedAccess) with protective comments.
- Guarded array indexing and optional values in App selection logic, FileList grouping, GitHistory scopes, file search helpers, mobile search selection, and bottom sheet touch handling.
- No remaining lint or type-check failures after `pnpm lint` and `pnpm type-check` (2026-02-04).

## Notes

- See `docs/PROBLEMS.md` for broader scenario-level issues outside React stability.
