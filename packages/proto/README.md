# Vrooli Proto Contracts

This package hosts Protocol Buffers schemas for inter-scenario contracts and their generated artifacts (Go, TypeScript, Python). Source files live under `schemas/`; generated code lives under `gen/` and is committed.

## Quickstart

- Edit `.proto` files in `schemas/`.
- Regenerate: `cd packages/proto && make generate`
- Lint: `cd packages/proto && make lint`
- Breaking check (against `master` by default): `cd packages/proto && make breaking`
- Keep `gen/` in sync with `schemas/` before committing.
- JSON serialization: BAS endpoints use snake_case; when marshaling with Go's `protojson` set `UseProtoNames: true` (and the equivalent options in TS/Python) so you don't emit lowerCamel JSON.

## Workspace usage

- Go: use the repo `go.work` so scenarios can import `github.com/vrooli/vrooli/packages/proto/gen/go/...`.
- TypeScript/JavaScript: `@vrooli/proto-types` is published from `gen/typescript` (ESM JS lives under `js/`) and linked via the pnpm workspace.
- Python: install `packages/proto/gen/python` in editable mode for local development.
