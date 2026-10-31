# Testing — Secrets Manager

## Test Surfaces

API and CLI use Go tests. UI uses Vitest with jsdom, a setup file, `renderWithProviders`, and V8 coverage. Test Genie owns the scenario suite.

## Required Commands

- `go test ./...` from `api/` and `cli/`
- `pnpm test`, `pnpm run lint`, and `pnpm run type-check` from `ui/`
- `vrooli scenario test secrets-manager` for lifecycle and cross-surface evidence

## Coverage Policy

The UI declares an 85 percent V8 coverage threshold. API and CLI coverage are evaluated by Unit Health. Do not lower a threshold to hide uncovered behavior.

## Test Architecture

`api/internal/testutil` and `cli/internal/testutil` guard against production imports. UI tests use `src/test-utils/renderWithProviders.tsx`. Direct non-deterministic dependencies should be moved behind injectable seams.

## Cross-References

- [Architecture](../concepts/ARCHITECTURE.md)
- [Performance](PERFORMANCE.md)
