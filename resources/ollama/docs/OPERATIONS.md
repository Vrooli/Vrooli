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
- Manage installed models through the Ollama API. `resource-ollama ensure`
  preserves models already present. For a missing model it resolves the public
  managed resource model path through Vrooli's shared storage resolver, rejects
  an explicit `OLLAMA_MODELS` that disagrees with it, resolves the public
  registry manifest over HTTPS, pins that exact manifest behind a one-shot
  loopback registry, holds Vrooli's managed-service lifecycle lock against a
  concurrent start/restart/stop, and reserves the model-store filesystem
  through the shared capacity disk-job owner before sending `/api/pull`. The lease samples model
  store growth during transfer, verifies the installed manifest digest, removes
  the temporary alias, checks partial files, samples final growth, and only then
  completes the shared job. Unknown paths/sizes, broker denials, sample errors,
  incomplete streams, or uncertain cancellation fail closed and leave an active
  lease for owner recovery. Do not treat HTTP cancellation as proof that Ollama's
  shared blob workers stopped. To recover an uncertain lease, use the lease ID in
  the ensure error, inspect `vrooli capacity disk-list --active --json`, stop
  the managed Ollama resource, and verify the daemon is stopped and its model
  store is no longer changing. Keep installed models and resumable partials;
  account for any retained bytes in the next path baseline. Then run
  `vrooli capacity disk-reconcile --lease-id <id> --process-stopped --cleanup-verified`.
  Restart Ollama only after reconciliation succeeds. The capacity owner checks
  the ensure process identity has exited; the operator supplies the evidence
  that the separate daemon and its pull-created files are quiescent.
  Missing-model acquisition refuses private registries
  and remote Ollama endpoints; the already-installed no-pull path keeps its
  inventory behavior. The adapter also refuses model roots with
  symlinks or nested filesystems and platforms that cannot report filesystem
  identity for the full model tree.
- Shared disk-job receipts keep model-store path samples (`Paths[].Samples[].DeltaBytes`)
  separate from filesystem free-space observations (`Filesystems[].ActualDeltaBytes`
  and `ActualDeltaBasis`). The path delta is model-store growth evidence; the
  filesystem delta is host-wide context and must not be attributed to Ollama.
- The managed-service environment owns concurrency and model residency.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
- If `start` fails on a port conflict, another host process owns `:11434`; the
  driver's preflight prints the remediation. Stop that process, then retry.

For bounded one-shot work, `resource-ollama gateway generate --one-shot
--require-loopback --timeout 90s --json` first refuses an already-loaded target,
sends Ollama `keep_alive: 0`, and verifies the target is absent from `/api/ps`
before it returns a result. The JSON receipt includes
`worker_cleanup_verified: true`. A timeout or generation error still runs the
bounded cleanup path; the result is rejected if cleanup cannot be verified.

All `gateway embed`, `gateway generate`, and `gateway chat` requests verify
loopback transport by default. `--require-loopback` remains accepted for older
callers; omitting it or passing `--require-loopback=false` cannot select a remote
endpoint.

Missing-model `ensure` acquisition accepts literal loopback addresses and the exact
hostname `localhost`. For `localhost`, the client resolves once, requires every
resolved address to be loopback, and pins the selected loopback IP into the
endpoint. It still disables proxies and refuses redirects; other DNS names,
remote addresses, credentials, and non-root URL paths are rejected.
When all requested models are already installed, `ensure` retains its existing
inventory-only endpoint behavior because no disk-writing request is issued.

The model-pull peak is the sum of every pinned config/layer descriptor, with no
deduplication credit, plus 3 × 4 MiB for manifest writes and 1 MiB per descriptor
for downloader metadata (at most 64 descriptors). Before admission, ensure
checks each blob's HTTPS `HEAD` content length against its manifest descriptor;
the temporary local registry then returns that exact length to Ollama v0.30.10's
standard downloader, preventing a later upstream `HEAD` from changing the sparse
partial size. The downloader
truncates a sparse partial to that advertised length and records at most 16
range-part JSON files. Its transfer downloader uses descriptor sizes and checks
the final blob size and SHA-256 before rename. Both staging branches write inside
the configured model blob directory. This makes the payload bound conservative
even when blobs are reused or sparse files are counted by logical size. The
fixed metadata allowance is intentionally generous; manifests above the
descriptor cap or beyond the 4 MiB validation bound are refused. The
requested tag is published from the frozen manifest through an atomic
same-filesystem hard link, which fails if a competing writer created the tag
before commit. Explicit Ollama API pulls remain independent operations outside
`ensure`'s shared admission contract.

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
