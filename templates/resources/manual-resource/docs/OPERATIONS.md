# Operations

`{{RESOURCE_NAME}}` is scaffolded as a manual resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative validation, lifecycle limits, and operator-facing metadata.
- `docs/` owns the real setup contract.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` stays intentionally small and only exists for validation or environment logic that cannot live purely in docs and manifest metadata.

Do not turn `cli/main.go` into the primary implementation surface. If the resource needs code at all, start with `cli/internal/validate` or `cli/internal/env` and keep the rest manual and explicit.

## Operator Checklist

- Document every prerequisite explicitly.
- Capture the manual install process step by step.
- Keep any local state in canonical resource storage directories, not repo-local `data/`.
- Define the validation probes Vrooli can run after setup.
- Prefer docs and manifest contracts first; use `cli/internal/validate` and `cli/internal/env` only for real specialization.
- State what Vrooli does not own.
