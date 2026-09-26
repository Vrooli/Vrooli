# Ollama managed-service maturity assessment

Evaluation date: 2026-08-22. This assessment uses the resource maturity and
deployment contracts in [`docs/resources/maturity-migration.md`](../../../docs/resources/maturity-migration.md)
and [`docs/resources/deployment-contract.md`](../../../docs/resources/deployment-contract.md).

## 1. Current maturity

Ollama is **M5 (maximal feasible maturity)**. M5 here means that the resource
has a complete, tested managed-service contract for the targets it claims; it
does not mean that an unverified native bundle exists for every operating
system.

| Dimension | Score | Evidence |
|---|---:|---|
| Contract | 2 | [`resource.json`](../resource.json) declares storage, exports, managed-service artifact, ports, lifecycle, readiness, GPU, capacity, and degradation profiles. |
| Archetype | 2 | The resource uses the shared `managed-service` template and provider policy; the native Ollama server is the runtime. |
| Operator surface | 2 | `resource-ollama` and `vrooli resource ... ollama` own install, lifecycle, status, logs, health, model ensuring, gateway, policy, and capacity operations. |
| Configuration | 2 | `model-policy.json`, typed dependency validation, role resolution, relocation keys, and the Go `ensure`/policy commands preserve existing model state while migrating callers to roles. |
| Runtime and health | 2 | The server artifact is version- and checksum-pinned; `/api/tags` is readiness and the control plane reports processor placement rather than trusting a scheduler hint. |
| Tests | 2 | `resources/ollama/Makefile` Go gates pass; focused capacity, health, policy, gateway, and ensure tests pass; Search Hub consumer smoke passed with the native service and Docker stopped. |
| Portability | 2 | Linux amd64 and Windows amd64 have staged, checksum-verified bundles; macOS amd64/arm64 remains conditional pending target smoke. |
| Legacy debt | 2 | No resource shell layer or Docker fallback is present in the manifest or normal operator path. Historical migration notes remain explicitly historical. |
| Deployment readiness | 2 | The managed-service profile names artifacts, checksums, architectures, health evidence, and unsupported/conditional reasons for every desktop target. |

## 2. Active contract and consumers

The active runtime contract is the Ollama HTTP API on port 11434, with model
state under `OLLAMA_MODELS`, plus the `resource-ollama` gateway and policy
surfaces. Search Hub, Meta Optimization Manager, and other scenarios consume
the exported URL and role contract; they do not own Ollama lifecycle. The
resource's Go tests exercise policy, gateway, health, capacity, and ensure
behavior. Older Docker wording in historical PRD material is not an active
runtime dependency.

## 3. Target archetype

The selected archetype is `managed-service`: Vrooli supervises a pinned native
Ollama server and owns its readiness, health, logs, environment, and data
relocation. Docker was rejected because the upstream server is already a
portable executable and a Docker daemon is an unnecessary desktop runtime
requirement. A native bundle is not claimed on targets where its extracted
artifact or target smoke evidence is absent.

## 4. Deployment profile

| Target | Delivery | Result and limitation |
|---|---|---|
| Linux amd64 | Pinned native `0.30.10` server artifact with SHA-256 | Conditional; manifest validation, checksum verification, native restart, readiness, GPU health, and Search Hub consumer evidence are recorded. |
| Linux arm64 | No matching Vrooli bundle | Conditional in the deployment contract; do not claim support until an artifact is published and exercised. |
| macOS amd64/arm64 | Pinned universal upstream server artifact | Conditional; checksum evidence exists, but native target smoke is still outstanding. |
| Windows amd64 | Pinned upstream standalone ZIP, extracted tree digest, amd64 manifest route | Build-verified; target smoke is still outstanding. |
| Windows arm64 | No matching upstream standalone bundle | Unsupported, with a clear pre-runtime reason. |

## 5. Gap list

- Produce and exercise a Vrooli Linux arm64 bundle if that target becomes a
  release requirement (owner: resource-platform maintainers).
- Record native macOS lifecycle and readiness evidence before upgrading its
  tier (owner: release qualification maintainers).
- Exercise the staged Windows amd64 server bundle on a real target before
  promoting Windows beyond build-verified (owner: release qualification
  maintainers).
- Keep model registry/network failures visible; model downloads remain a
  first-run operational dependency even though the server itself is bundled
  (owner: Ollama resource maintainers).

## 6. Migration plan

The migration is complete for the current target set. Future portability work
is additive and reversible: publish a checksum-paired artifact, add its
deployment profile, run manifest and target gates, run consumer smoke, and only
then promote that target from conditional or unsupported. Existing model data
stays under the same relocation key throughout.

## 7. Validation matrix

| Gate | Evidence |
|---|---|
| Manifest and artifact | Fleet manifest validation; pinned artifact metadata and checksum tests. |
| Unit/type/lint | `make check` in `resources/ollama`. |
| Runtime | Native managed-service restart, `/api/tags` readiness, and status processor mode. |
| Capacity | `vrooli capacity reconcile`; multi-step VRAM degradation profile and zero-claim finding tests. |
| Consumer | Search Hub server-owned suite and native Ollama smoke with Docker stopped. |
| Platform gates | Explicit Linux amd64, macOS conditional, Linux arm64 conditional, Windows amd64 build-verified, and Windows arm64 unsupported profiles. |

## 8. Risks and decisions

Model files are large but regenerable and remain in the existing resource data
root. The native artifact is preferred because it removes the Docker daemon
from the local inference path. Conditional and unsupported targets are
deliberate evidence boundaries, not implicit promises.
