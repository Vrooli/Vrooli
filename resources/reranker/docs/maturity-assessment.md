# Reranker managed-service maturity assessment

Evaluation date: 2026-08-15. This assessment uses the resource maturity and
deployment contracts in [`docs/resources/maturity-migration.md`](../../../docs/resources/maturity-migration.md)
and [`docs/resources/deployment-contract.md`](../../../docs/resources/deployment-contract.md).

## 1. Current maturity

Reranker is **M5 (maximal feasible maturity)**. The claim is bounded to the
explicitly declared deployment profile: Linux amd64 is the only staged native
bundle, while macOS and Windows are deliberately unsupported until delivery
and target evidence exist.

| Dimension | Score | Evidence |
|---|---:|---|
| Contract | 2 | [`resource.json`](../resource.json) declares the TEI artifact, model/cache storage, exports, port, readiness, capacity, and lifecycle profile. |
| Archetype | 2 | The shared `managed-service` template supervises the pinned TEI server; no custom container or shell lifecycle is needed. |
| Operator surface | 2 | `resource-reranker` and `vrooli resource ... reranker` own install, lifecycle, status, logs, health, gateway reranking, info, and capacity operations. |
| Configuration | 2 | Typed manifest environment and relocation metadata preserve the model cache; the model policy and gateway keep the served model and endpoint explicit. |
| Runtime and health | 2 | TEI `1.7.4` is checksum-pinned and `/health` is a readiness probe that waits for the model-serving process. GPU/CPU behavior is reported as a runtime capability, not inferred from a scheduler hint. |
| Tests | 2 | `resources/reranker/Makefile` Go gates pass; client, gateway, config, CLI, and capacity-sync tests pass; Search Hub exercises the shared consumer path. |
| Portability | 2 | Linux amd64 is conditional with an explicit CUDA/CPU limitation; macOS and Windows are explicitly unsupported because no native bundle is staged. |
| Legacy debt | 2 | No resource Bash layer, Docker runtime, or fallback shim exists in the manifest or normal operator path. |
| Deployment readiness | 2 | The profile names the pinned artifact, architecture, readiness evidence, host limitation, and clear unsupported reasons for every desktop target. |

## 2. Active contract and consumers

The active contract is `RERANKER_URL` and `POST /rerank` on port 11453, with
the TEI model cache under the resource-owned data directory. Search Hub and
the shared AI search package consume the resource through the reranker
interface and fall back to Ollama/fused retrieval when it is unavailable.
The Go resource tests cover the operator and gateway contract; Search Hub
provides the consumer validation.

## 3. Target archetype

The selected archetype is `managed-service`: Vrooli owns the supervised TEI
server, cache placement, readiness, logs, and health. A Docker runtime would
add a daemon requirement without improving the single-process lifecycle.
Native macOS and Windows delivery is not inferred from upstream source
availability; those targets remain rejected before runtime execution until
Vrooli-owned artifacts are published and tested.

## 4. Deployment profile

| Target | Delivery | Result and limitation |
|---|---|---|
| Linux amd64 | Pinned TEI `1.7.4` server artifact with SHA-256 | Conditional; manifest validation, checksum verification, readiness contract, and consumer evidence are recorded. The staged target requires CUDA 8.9+; the CPU image remains explicitly unsupported until a relocatable, closure-verified bundle is published. |
| macOS amd64/arm64 | No staged native TEI bundle | Unsupported, with a clear pre-runtime reason. |
| Windows amd64/arm64 | No staged native TEI bundle | Unsupported, with a clear pre-runtime reason. |

## 5. Gap list

- Validate CPU fallback for the selected model on each supported Linux host
  class before changing the conditional tier (owner: reranker resource
  maintainers).
- Publish a signed, checksum-paired macOS bundle only if TEI provides a
  supported native backend and the model has a readiness smoke path (owner:
  resource-platform maintainers).
- Publish and exercise a Windows bundle before enabling Windows (owner:
  resource-platform maintainers).
- First start requires network access to populate the regenerable model cache;
  warm starts are local (owner: reranker resource maintainers).

## 6. Migration plan

The managed-service migration is complete for Linux amd64. Future target work
is additive and reversible: publish the target artifact, add the profile,
validate checksum/readiness, run reranking and Search Hub smoke, then promote
the target. The existing model cache path is retained for restart and rollback
continuity.

## 7. Validation matrix

| Gate | Evidence |
|---|---|
| Manifest and artifact | Fleet manifest validation; pinned TEI artifact metadata and checksum tests. |
| Unit/type/lint | `make check` in `resources/reranker`. |
| Runtime | Managed-service readiness on `/health`, gateway health/info, and model-serving contract. |
| Capacity | `vrooli capacity reconcile`; multi-step VRAM profile and capacity-sync tests. |
| Consumer | Search Hub server-owned suite and reranker gateway contract tests. |
| Platform gates | Explicit Linux amd64 conditional profile and unsupported macOS/Windows gates. |

## 8. Risks and decisions

The model cache is regenerable but large, so it is retained and relocated
across restarts. The native TEI artifact is preferred over Docker for local
portability. GPU acceleration is conditional on the host driver and the model
backend; the resource must not report GPU readiness solely from host inventory.
