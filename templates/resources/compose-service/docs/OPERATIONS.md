# Operations

`{{RESOURCE_NAME}}` is scaffolded as a compose-managed multi-service resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative lifecycle and orchestration metadata.
- `compose.yaml` owns the concrete service graph.
- `cli/` owns entrypoint wiring and delegated command registration.
- `internal/` owns compose-specific Go logic that cannot be expressed through the manifest, compose graph, or shared control-plane packages.

Do not turn `cli/main.go` into the primary implementation surface. If the resource needs specialized graph handling, dependency reasoning, readiness logic, or environment shaping, grow the matching package under `internal/` first.

## Operator Checklist

- Replace placeholder images in `compose.yaml`.
- Keep bind mounts rooted in canonical resource storage directories rather than repo-local `data/`.
- Define explicit dependency and readiness rules.
- Prefer shared control-plane behavior first; use `internal/compose`, `internal/topology`, `internal/runtime`, `internal/health`, and `internal/env` only for real specialization.
- Document data directories, backup, and restore behavior.
- Add teardown guidance for local development and production support.
