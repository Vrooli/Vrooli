# Ollama Installation Guide

This guide covers installing and running Ollama for local LLM inference.

Ollama runs as a Vrooli-managed service from a checksum-verified native artifact.
There is no host-systemd install, no `/usr/local/bin/ollama` binary, no Docker
fallback, and no `manage.sh` script — the managed-service supervisor owns the
deployment. Ports, environment, artifact, and health checks are declared in
[`resource.json`](../resource.json) and applied by `vrooli resource …`.

## Prerequisites

- **Hardware**:
  - Minimum: 8GB RAM, 4GB free disk space
  - Recommended: 16GB+ RAM, 50GB+ disk space
  - GPU: NVIDIA with 8GB+ VRAM (optional but recommended)
- **Software**:
  - A staged Ollama server artifact matching the manifest checksum
  - NVIDIA driver/runtime (only for GPU acceleration)

## Install / Start / Stop

All lifecycle actions route through the managed-service driver:

```bash
vrooli resource install ollama   # verify the pinned native server artifact
vrooli resource start ollama     # supervise the native server on :11434
vrooli resource status ollama    # report Running/Healthy via the /api/tags health check
vrooli resource stop ollama      # stop the managed process (model data persists)
vrooli resource logs ollama      # read the managed-service log
```

`install` verifies the staged artifact before the supervisor grants it lifecycle
authority. The model directory is `${RESOURCE_DATA_DIR}/models`, so pulled
models survive restarts.

### Port-conflict preflight

The managed-service driver performs the normal host-port preflight and fails
fast if `:11434` is already occupied. Stop the conflicting service and retry:

```
resource "ollama" cannot start: host port 11434 is already in use. Stop and
remove the conflicting process — e.g.
`sudo systemctl disable --now ollama` or terminate whatever is listening on
:11434 — then retry.
```

## Models

There is no host `ollama` CLI. Manage models through the service API or the
resource CLI; this keeps model identity and deletion inside Ollama's ownership
boundary:

```bash
# List installed models
curl http://localhost:11434/api/tags

# Pull a model through the service API
curl http://localhost:11434/api/pull -d '{"name":"llama3.1:8b","stream":false}'
```

Scenarios should not pull models by hand. They declare model **roles** (resolved
through [`model-policy.json`](../model-policy.json)) in their ollama dependency
config, and `resource-ollama ensure` pulls any missing resolved models into the
running managed service automatically.

## Configuration

Runtime configuration lives in [`resource.json`](../resource.json). The normal
managed-service settings are under `managed_service`:

| Setting | Where | Default |
| --- | --- | --- |
| Artifact pin | `managed_service.artifact` | Ollama `0.30.10`, checksum verified |
| API port | `ports[].host` | `11434` |
| Concurrency / loaded models / flash-attn / keep-alive / CORS | `managed_service.environment` | see file |
| Model storage | `managed_service.environment.OLLAMA_MODELS` | `${RESOURCE_DATA_DIR}/models` |

To change any of these, edit `resource.json` and `vrooli resource restart ollama`.

## Post-Installation Verification

```bash
# Lifecycle health
vrooli resource status ollama        # → Running / Healthy

# API directly
curl http://localhost:11434/api/tags

# Inference smoke test (model must be pulled first)
curl http://localhost:11434/api/generate \
  -d '{"model":"llama3.1:8b","prompt":"What is 2+2?","stream":false}'
```

## GPU Acceleration

```bash
# Verify host drivers
nvidia-smi

# Verify host GPU access
nvidia-smi
```

## Troubleshooting

- **Port already in use** — see the [port-conflict preflight](#port-conflict-preflight)
  above; stop the host process holding `:11434`, then `vrooli resource start ollama`.
- **GPU not detected** — confirm `nvidia-smi` works on the host, then inspect
  `vrooli resource status ollama --json` for processor and GPU findings.
- **Insufficient memory** — prefer a smaller role target (e.g. the `chat.small`
  role; see `resource-ollama policy roles`) or lower concurrency/model residency
  in `managed_service.environment`.
- **Inspect managed logs** — `vrooli resource logs ollama`.

## Next Steps

- [Explore the model catalog](MODELS.md)
- [Operations runbook](OPERATIONS.md)
- [Embedding models](EMBEDDING_MODELS.md)
