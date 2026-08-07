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

## GPU access and processor diagnosis

`vrooli resource status ollama --json --no-stale-check` includes the
container-scoped `raw.gpu_state`, `raw.gpu_reason`, and `raw.processor` fields.
The GPU state is based on opening `/dev/nvidiactl` inside the running container,
not merely on host `nvidia-smi`. `revoked` is degraded and `unknown` is not
healthy.

To inspect the live Ollama `/api/ps` processor placement directly:

```text
resource-ollama health-gpu --json
resource-ollama capacity plan --scenario <name> --json
```

The first command reports `gpu`, `cpu`, `mixed`, or `unknown` placement and
returns a failure when the host has an NVIDIA GPU while a loaded model is on
CPU. Capacity output keeps host GPU facts separate from the live processor
observation. Repair container access with `vrooli resource restart ollama`.
