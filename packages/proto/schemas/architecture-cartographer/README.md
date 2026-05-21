# Template proto sources

This folder is **not** a regular template directory.

At scenario generation time, `vrooli scenario generate` reads the
`relocations` block in `template.json` and copies this entire `proto/`
tree into `packages/proto/schemas/<your-scenario>/`, substituting
`architecture-cartographer` and `architecture_cartographer` in both path components
and file content. The `proto/` folder does **not** appear inside the
generated scenario.

After relocation, the generator runs `make generate` in
`packages/proto/` so the scenario's `api/`, `ui/`, and `cli/` can
import generated Go and TypeScript types immediately.

## What's here

- `v1/health/health.proto` — wire contract for the `/health` endpoint.
  Mirrors `packages/api-core/health/health.go`'s `Response` and
  `DependencyStatus` types field-for-field with matching JSON names so
  the api-core handler chain produces JSON that decodes into the
  generated proto type without translation.

  After relocation this lands at
  `packages/proto/schemas/<your-scenario>/v1/health/health.proto`. The
  namespace comes from the relocation `to` path in `template.json`,
  not from a directory inside `proto/` — matching the convention used
  by every existing scenario in `packages/proto/schemas/`.

## Adding a new schema

After the scenario is generated, add new `.proto` files under
`packages/proto/schemas/<your-scenario>/v1/` (or a `v1/<domain>/`
subdirectory if the domain warrants its own folder). Then run
`cd packages/proto && make generate && make lint` and commit the
regenerated artifacts in `packages/proto/gen/`.

## Why this layout

`packages/proto/schemas/` is the canonical source of truth for every
scenario's wire contracts. Keeping the template's protos here as a
relocation source — rather than directly in `packages/proto/schemas/` —
prevents the template's protos from leaking into builds: a scenario's
generated artifacts always come from its own substituted copy at
`packages/proto/schemas/<your-scenario>/`, never from this template tree.

See `templates/scenarios/react-vite/template.json::relocations` for the
generator wiring and `internal/cli/scenariohandlers/template_runtime.go`
for the relocation implementation.
