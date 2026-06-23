# Ollama Installation Guide

This guide covers installing and running Ollama for local LLM inference.

Ollama runs **exclusively as a Docker container** managed by the Vrooli
docker-service driver. There is no host-systemd install, no `/usr/local/bin/ollama`
binary, and no `manage.sh` script — the container *is* the deployment. The runtime
image, memory cap, ports, environment, and health checks are declared in
[`resource.json`](../resource.json) and applied by `vrooli resource …`.

## Prerequisites

- **Hardware**:
  - Minimum: 8GB RAM, 4GB free disk space
  - Recommended: 16GB+ RAM, 50GB+ disk space
  - GPU: NVIDIA with 8GB+ VRAM (optional but recommended)
- **Software**:
  - Docker (the only hard requirement)
  - NVIDIA Container Toolkit (only for GPU acceleration)

## Install / Start / Stop

All lifecycle actions route through the Docker driver:

```bash
vrooli resource install ollama   # pull the runtime image (ollama/ollama:<pin>)
vrooli resource start ollama     # run the container, bind :11434, apply env + memory cap
vrooli resource status ollama    # report Running/Healthy via the /api/tags health check
vrooli resource stop ollama      # stop the container (data volume persists)
vrooli resource logs ollama      # tail container logs (or: docker logs ollama)
```

`install` is image-only; `start` creates the container with the declared ports,
environment, `--memory 12g` cap, and the `${RESOURCE_DATA_DIR}:/root/.ollama`
bind-mount volume so pulled models survive restarts.

### Port-conflict preflight

Because Ollama now runs as a container, a **host** process already listening on
`:11434` (for example a leftover host-systemd Ollama from before the Docker
migration) would make `docker run -p 11434:11434` fail. The driver detects this
*before* starting and fails fast with an actionable message instead of crash-looping:

```
resource "ollama" cannot start: host port 11434 is already in use by a
non-container process. … Stop and remove the host process — e.g.
`sudo systemctl disable --now ollama` or terminate whatever is listening on
:11434 — then retry.
```

## Models

There is no host `ollama` CLI. Manage models inside the container or via the
resource CLI:

```bash
# List installed models
docker exec ollama ollama list

# Pull a model
docker exec ollama ollama pull llama3.1:8b

# Remove a model
docker exec ollama ollama rm llama3.1:8b
```

Scenarios should not pull models by hand. They declare model **roles** (resolved
through [`model-policy.json`](../model-policy.json)) in their ollama dependency
config, and `resource-ollama ensure` pulls any missing resolved models into the
running container automatically.

## Configuration

Runtime configuration lives in [`resource.json`](../resource.json) under `runtime`:

| Setting | Where | Default |
| --- | --- | --- |
| Image pin | `runtime.image` | `ollama/ollama:0.30.10` |
| Memory cap | `runtime.memory_limit` | `12g` (container OOM-kill ceiling) |
| API port | `ports[].host` / `.container` | `11434` |
| Concurrency / loaded models / flash-attn / keep-alive / CORS | `runtime.env` | see file |
| Model storage | `runtime.volumes` bind-mount | `${RESOURCE_DATA_DIR}` → `/root/.ollama` |

The `memory_limit: 12g` cap is the Docker equivalent of the cgroup `MemoryMax`
limit the old host-systemd unit used to render — it keeps an embeddings burst from
exhausting host RAM (the failure mode behind the 2026-05-07 host-stability incident)
by letting Docker OOM-kill the container instead of hanging the host.

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

# Verify Docker GPU passthrough
docker run --rm --gpus all nvidia/cuda:12-base-ubuntu22.04 nvidia-smi

# Install the NVIDIA Container Toolkit if missing
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey \
  | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
```

## Troubleshooting

- **Port already in use** — see the [port-conflict preflight](#port-conflict-preflight)
  above; stop the host process holding `:11434`, then `vrooli resource start ollama`.
- **GPU not detected** — confirm `nvidia-smi` works on the host and the NVIDIA
  Container Toolkit is installed; restart the container.
- **Insufficient memory** — prefer a smaller role target (e.g. the `chat.small`
  role; see `resource-ollama policy roles`) or raise `runtime.memory_limit` in
  `resource.json`.
- **Inspect the container** — `docker logs ollama`, `docker inspect ollama`.

## Next Steps

- [Explore the model catalog](MODELS.md)
- [Operations runbook](OPERATIONS.md)
- [Embedding models](EMBEDDING_MODELS.md)
