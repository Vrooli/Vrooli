# Operations

`{{RESOURCE_NAME}}` is scaffolded as a desktop application resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative platform, lifecycle, and detection metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns desktop-app-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the primary implementation surface. If the resource needs specialized detection, install translation, platform gating, or health verification, grow the matching package under `cli/internal/` first.

## Operator Checklist

- Document supported OS versions and install paths.
- State clearly which platforms are unsupported.
- Keep config/cache/log state in canonical resource storage directories, not repo-local `data/`.
- Prefer shared control-plane behavior first; use `cli/internal/discovery`, `cli/internal/install`, `cli/internal/platform`, and `cli/internal/health` only for real specialization.
- Capture plugin/config/profile setup requirements.
- Write manual verification steps for operators and agents.
