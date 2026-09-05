# Integrations — Asset Studio

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | identities, specs, renders, assets, conformance | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Filesystem BlobStore | embedded storage | yes | assets | BlobStore seam in the assets module | Render completes but the asset is not stored; the job fails rather than recording an artifact it cannot serve. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario must be started through lifecycle commands. |
| ai-gateway | scenario | yes | renders (image and video generation) | scenario CLI/API | **Renders queue rather than fail.** A submitted job stays submittable and reports the gateway as unavailable; nothing already produced is affected, and no vendor fallback is attempted. |
| image-tools | scenario | yes | assets (derived variants), conformance (scoring, P1) | scenario CLI/API | Variants are not produced; the parent asset is unaffected and can still be released. Automated scoring degrades to operator-only judgement, which is the P0 behaviour anyway. |
| browser-automation-studio | scenario | no (P1) | specs, renders (capture kind) | scenario CLI/API | Capture specs cannot render; generative specs are unaffected. |
| search-hub | scenario | no (P1) | federation registration | `.vrooli/search.json` descriptor + `RegisterProvider` | Local library queries keep working; only federated reach is lost. Registration retries. |
| agent-manager | scenario | no (P1) | UI (commissioned runs) | scenario CLI/API | The workbench loses the commission action; every manual path is unaffected. |
| audio-tools | scenario | no (P2) | persona voice | scenario CLI/API | Not wired. |
| vrooli-events | scenario | no | all | api-core receipt publication (automatic) | Run correlation is absent; nothing else is affected. |
| marketing rich-media catalogue | repository files | no | identities (import) | Read-only path glob under `docs/marketing/catalogs/rich-media/` | Import reports the unreadable source and imports nothing from it; records already imported are unaffected. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| SQLite (embedded) | active | Identity, spec, job, asset, and verdict tables are single-writer, local, and modest. Artifact bytes go to the filesystem, not to a database. | If the scenario ever becomes multi-host. |
| Filesystem (BlobStore) | active | Produced artifacts are opaque bytes with a local consumer. The template's filesystem BlobStore is the correct default and needs no external resource. | If artifact volume outgrows local disk, or if a hosted deployment is ever chosen. |
| ollama | indirect | Reached **through ai-gateway**, never directly. Generation is an inference call, and ai-gateway owns model policy, routing, and capacity. | Never call ollama directly from this scenario — that would duplicate policy the gateway owns. |
| minio | not-applicable | Object storage is unnecessary while delivery is local-first. Introducing it would add an operational dependency for a capability the filesystem already provides. | If hosted delivery is chosen, which is currently explicitly not the plan. |
| qdrant | not-applicable | Conformance scoring compares a frame to a small set of reference images for one identity; that is a direct comparison, not a vector search over a corpus. | If identity lookup ever becomes semantic search over thousands of records. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| ai-gateway | required | Every image and video generation call. Model policy, routing, and capacity stay in the gateway. | Scenario CLI/API. This scenario never speaks a model vendor's protocol and holds no vendor credential. |
| image-tools | required | Four distinct uses, all through the same dependency: deterministic operations (resize, crop, format) for derived variants; **identity conditioning** — LoRA, IP-Adapter, ControlNet, img2img — at render time (`ASSET-P1-012`); **regional refinement** via inpainting (`ASSET-P1-013`); and image analysis for conformance scoring (`ASSET-P1-005`). | Scenario CLI/API. **No image processing, conditioning, or adapter logic is implemented here.** The conditioning machinery already exists in image-tools; this scenario contributes identity, versioning, and provenance on top of it. |
| browser-automation-studio | P1 | Executes capture specs. This scenario never drives a browser itself. | Scenario CLI/API through the capture-executor seam. |
| content-desk | consumer | It references released assets by identifier. **The dependency runs one way** — this scenario does not call content-desk and must not learn about drafts, campaigns, or publication state. | Asset reference surface (`ASSET-P0-014`). |
| search-hub | P1 | Federated retrieval over identity and asset metadata, never bytes. | `.vrooli/search.json` validated against `.vrooli/schemas/search.schema.json`; boot self-registration. |
| agent-manager | P1 | Commissioned spec composition and conformance triage from the workbench. | Scenario CLI/API; results carry recorded provenance and never satisfy the operator gate. |
| audio-tools | P2 | Persona voiceover bound to an identity record. | Scenario CLI/API. |
| vrooli-events | automatic | Run correlation for work done inside an agent run. **No integration work** — `api-core` wraps every handler in the automatic receipt runtime. | Correlation ids only; run payloads are never copied. |
| scenario-to-desktop | P2 | Tier 2 desktop packaging, matching content-desk's delivery direction. | Packaging pipeline; no code dependency. |

### Direction of dependency with content-desk

Worth stating explicitly because it is easy to get backwards. `content-desk`
depends on `asset-studio`; `asset-studio` does not depend on `content-desk`.

| Concern | Owner | The other scenario's role |
|---|---|---|
| Artifact bytes | asset-studio | content-desk stores a reference and no bytes at all |
| Alt text, dimensions, disclosure state | asset-studio | content-desk reads them with the reference |
| Which draft uses which asset | content-desk | asset-studio never learns this |
| Whether an artifact was published | content-desk | asset-studio never learns this |

A `campaign_ref` on a spec is a **label for cost reporting**, not a foreign key.
It is a free-text reference this scenario never resolves, so the two scenarios
can be developed, tested, and run entirely independently.

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | Every inference call routes through ai-gateway and all storage is local. This scenario reaches no external network service and holds no vendor credential. | Add only if a hosted generation provider is ever introduced — which would go through ai-gateway, not here. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| BlobStore | write error | The render job fails and records the failure. **No asset record is created for bytes that were not stored** — a metadata row pointing at nothing is worse than no row. | `ASSET-P0-009` |
| ai-gateway (generation) | call error or timeout | The job records a failed attempt with whatever cost was consumed and remains retryable. No partial asset is written, and no vendor fallback is attempted. | `ASSET-P0-006`, `ASSET-P0-008` |
| ai-gateway (long-running video job) | connection lost mid-job | The job is resumed by identifier rather than orphaned or duplicated. | `ASSET-P1-002` |
| image-tools (variants) | call error | The parent asset is stored and releasable; the variant is retryable and its absence is reported rather than silently skipped. | `ASSET-P0-009` |
| image-tools (scoring, P1) | call error | The frame keeps its unresolved verdict and the operator judges without a recommendation. **The gate does not weaken** because scoring is unavailable. | `ASSET-P1-005` |
| browser-automation-studio | unreachable, or step timeout | The capture job fails with the failing step named. Generative specs are unaffected. | `ASSET-P1-003` |
| Catalogue source unreadable | path missing or permission denied | Import skips that source and reports it; other sources import normally. A partial sweep is never recorded as complete. | `ASSET-P0-003` |
| Catalogue source shape changed | schema validation yields zero valid items from a non-empty file | **Import aborts for that item rather than importing nothing silently.** A changed shape must surface as a failure, not as an empty diff that reads as "nothing new". | `ASSET-P0-003` |
| search-hub | registration failure at boot | Scenario starts and serves local queries; registration retries. | `ASSET-P1-008` |
| Bound identity version missing at regeneration | provenance references a version that no longer resolves | Regeneration reports the missing version **by name** rather than substituting the current one. Silent substitution would produce a different artifact under the same provenance. | `ASSET-P1-010` |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
