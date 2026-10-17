# Operations

`ollama` is organized as a `managed-service` resource. The control plane
supervises the checksum-verified native server artifact and keeps model state
under `OLLAMA_MODELS`.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative runtime, lifecycle, port, export, artifact,
  and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Ollama-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

There is no host-systemd install path or `lib/` shell layer. The normal operator
path is the shared managed-service driver; there is no second (bash) manager to
keep in sync.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized runtime shaping, model bootstrap, richer status interpretation, Ollama-specific probes, or environment derivation, grow `cli/internal/install`, `cli/internal/runtime`, `cli/internal/status`, `cli/internal/health`, or `cli/internal/env` first.

## Operator Checklist

- Keep ports, model storage, artifact, and health checks declared in
  `resource.json`.
- Keep mutable model and cache state under `${RESOURCE_DATA_DIR}/models` (persists across managed-service restarts).
- Manage models through the Ollama API or let `resource-ollama ensure` pull
  scenario-declared roles; there is no host package-manager install path.
- The managed-service environment owns concurrency and model residency.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
- If `start` fails on a port conflict, another host process owns `:11434`; the
  driver's preflight prints the remediation. Stop that process, then retry.

## GPU access and processor diagnosis

`vrooli resource status ollama --json --no-stale-check` includes
`raw.gpu_state`, `raw.gpu_reason`, and `raw.processor` fields. The managed
service reports native host GPU access and `/api/ps` processor placement;
`revoked` is degraded and `unknown` is not healthy.

To inspect the live Ollama `/api/ps` processor placement directly:

```text
resource-ollama status --json
resource-ollama capacity plan --scenario <name> --json
```

The first command reports `gpu`, `cpu`, `mixed`, or `unknown` placement and
returns a failure when the host has an NVIDIA GPU while a loaded model is on
CPU. Capacity output keeps host GPU facts separate from the live processor
observation. Repair managed-service state with `vrooli resource restart ollama`.
