# Testing

## TL;DR — the canonical examples

Use fakeable interface graph clients for actual graph tests and in-memory SQLite for store tests.

## API testing

Run `go test ./...` under `api`.

## UI testing

Run `pnpm run lint` and `pnpm run build` under `ui`.

## CLI testing

Run `go test ./...` under `cli`.

## How to add a new proto

Add the schema under `packages/proto/schemas`, regenerate artifacts, run `make lint` in `packages/proto`, and add API/CLI adapter tests.

## E2E binary smoke gate

Use lifecycle-managed scenario tests, not direct binary execution.

## Coverage thresholds

Coverage is tracked by test-genie; low coverage findings should be handled as a campaign when broad.

## Common patterns and anti-patterns

Prefer fake clients and fixture facts over source-file scanning in SDA tests.

## Cross-references

- `../concepts/FLOWS.md`
- `SEAMS.md`
