# Operations

`ollama` is organized as a `docker-service` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative runtime, lifecycle, port, export, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Ollama-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

There is no longer any host-systemd install path or `lib/` shell layer — Ollama runs
purely as a Docker container. All deployment/management changes localize to the Docker
driver and `resource.json`; there is no second (bash) manager to keep in sync.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized runtime shaping, model bootstrap, richer status interpretation, Ollama-specific probes, or environment derivation, grow `cli/internal/install`, `cli/internal/runtime`, `cli/internal/status`, `cli/internal/health`, or `cli/internal/env` first.

## Operator Checklist

- Keep runtime image, ports, volumes, and health checks declared in `resource.json`.
- Keep mutable model and cache state in the `${RESOURCE_DATA_DIR}` bind-mount (persists across container restarts).
- Manage models inside the container (`docker exec ollama ollama …`) or let `resource-ollama ensure` pull scenario-declared roles; there is no host `ollama` CLI.
- The `runtime.memory_limit` cap (`12g`) is the OOM-kill ceiling that replaced the old host-systemd cgroup limits — keep it set.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
- If `start` fails on a port conflict, a non-container host process owns `:11434`; the driver's preflight prints the remediation. Stop that process, then retry.
