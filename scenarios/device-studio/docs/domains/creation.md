# AI model and scene creation

## Scope

Foundation contract, 2026-10-19. Product behavior below is specified for implementation; the generated reference scaffold does not implement it. Research demonstrates individual techniques, not a released scenario.

AI creation/refinement is an explicit first-release workflow. It uses existing/imported candidates when suitable and creates reference-based alternatives when needed. It is not hidden inside ordinary editing or rendering.

## Initiating action and policy

Before starting, label AI use and agent launch separately. Show local/BYOK/Vrooli funding source, selected account where relevant, estimate or unknown cost, enforceable total ceiling, time/iteration/concurrency limits, permitted sources/workspace/tools and data leaving the machine. A project external-app-media opt-in is independent of account consent and spending authority. Default external references are public hardware/scene references and blank/redacted screens; remote vision/critique obeys the same policy.

Admission must establish an enforceable upper bound for each paid operation and reserve it against the whole parent budget, including children and retries. If the runner/provider cannot enforce the requested hard cap, refuse that route with a concrete reason. Current soft token admission cannot be advertised as hard in-flight containment. Unknown actual costs remain pending and retain their hold until reconciled.

## Creation stages

1. Resolve exact product/generic identity, target shot/view envelope, dimensions and permitted reference sources.
2. Search zero-purchase sources and reject incompatible rights or low-quality preview-only leads with reasons.
3. Preserve originals; import/adapt or construct geometry in an isolated candidate workspace.
4. Lock known dimensions, calibrate reference cameras and compare silhouette/parts/materials using held-out views.
5. Apply a bounded edit and retain source code/change log, tool/model identity when reported and before/after outputs.
6. Evaluate against unchanged criteria; iterate within the parent budget or return needs-input/budget-exhausted.
7. Present candidate lineage, remaining uncertainty, scope and comparisons for review. Publish only through approved domain and shared owner gates.

Model identity/seed/cost absent from a tool result remain unknown. Do not invent them to satisfy an upstream admission field. A generated photographic result cannot pass a geometry-fidelity review for the underlying mesh.

## Autonomy and intervention

One disclosed bounded workflow can perform authorized iterations without repeated modal confirmation. Activity lists AI calls, child agents, tool stages, file changes, elapsed resources, costs and outputs. Expanded privacy, funding, authority or budget scope requires a revised explicit choice. Normal asset reuse, camera controls, scrubbing and local deterministic rendering do not spawn agents. AI denoisers/upscalers, including local ones, are accurately labeled if used.

Cancellation prevents new child dispatch, stops owned local work, requests remote cancellation and preserves candidate checkpoints. Already consumed external work may remain billable. Restart resumes recorded state and reconciles uncertain effects instead of blindly regenerating. A partial or rejected candidate is retained as evidence and never promoted automatically.

## Review and release

Keep technical validity, dimensional fidelity, intended motion support, rights and craft as separate review dimensions. Human approval names exact revision and supported use envelope. Source/model/profile changes invalidate affected evidence. Device Studio's review does not grant missing license rights or bypass Asset Studio's human-only decisions. An unavailable shared release adapter produces release-pending while local private use remains possible under its permissions.

## Implementation seams and first bounded slice — 2026-10-20

The current implementation has useful owners, but no creation workflow owner yet:

- `internal/library` canonicalizes bundle manifests and review bindings, streams
  and verifies artifact bytes, publishes immutable reviewed bundle revisions,
  and exposes bounded metadata reads. `LibraryCatalogService` is read-only.
  `Publish` is an internal trusted call: a caller-provided `ReviewBinding` is
  not a safe public review endpoint.
- `internal/projects` provides staged, digest-verified project imports for
  PNG/JPEG and inspected video. It deliberately does not accept GLB geometry.
- `internal/devices`, `internal/scenes`, and `internal/motions` own canonical,
  immutable candidate revisions with expected-head writes, revision reads and
  bounded head lists. A device candidate must reference an exact reviewed
  Library artifact; candidate state is not geometry qualification.
- `internal/physical` resolves exact bundle manifest/review/artifact digests
  and current `render` use rights. The shot resolver repeats those checks when
  a saved physical setup is compiled. `internal/physicalruntime` only seeds
  packaged reviewed starter assets and candidate revisions.
- The UI can browse candidate heads and show Library rights/review metadata in
  `PhysicalShotWorkspace`. The CLI can save and inspect component candidates,
  list/get Library revisions and initialize the packaged starter. Neither
  imports an arbitrary model or submits qualification review.
- There is no Device Studio agent/provider admission or creation-job owner.
  The capability registry currently describes optional Audio Tools only.
  Creation policy in this document is therefore a contract, not an existing
  runtime control.

### E5-A: local geometry intake candidate

E5-A provides a local-only path that imports one GLB as a durable,
inspectable **unreviewed intake candidate**. It exercises original-byte
custody, provenance, bounded GLB inspection and review status without making
an import publishable or renderable.

The owner is `api/internal/library/intake.go`. It uses the routed blob store,
the existing `revisionstore` SQLite repository, streaming size and digest
verification, and embedded schema registration. The API, multipart handler,
CLI, and physical workbench expose revision-1 import, bounded list, metadata
read, and verified original-content read. User claims remain separate from
inspector observations and owner-assigned review/render status. Each import
writes original bytes under a unique routed key. Same-ID exact replays
converge; conflicting claims or bytes reject. Parent lineage resolves and
pins an exact previous candidate revision digest before staging. Ambiguous
metadata commits retain any object that might be referenced. Projections do
not expose storage keys or host paths.

The import request accepts user claims only. It cannot set inspection,
`unreviewed`, or `not_reviewed` facts. `rendering.InspectPhysicalGLB` records
format/profile observations; unsupported results remain inspectable facts and
do not become renderable. The UI labels candidates unreviewed and does not
show them in renderable selectors. There is no `ArtifactRef`, `ReviewBinding`,
`UseRender` grant, qualification result, derived GLB, DeviceCandidate save,
metadata-correction writer, archive extraction, external fetch, or native
script execution in this slice. The endpoint creates revision 1 for a new
candidate ID; exact same-ID replays converge and differing content/claims
conflict. It records exact parent lineage but does not yet create a later
revision. Intake records are not Library bundle revisions and cannot be
resolved by `OpenArtifact`.

The implementation boundaries are:

| Change | Owner surface |
|---|---|
| Intake value, bounds, repository, embedded schema and verified object read | `api/internal/library/intake.go`, `api/internal/library/schema.sql`, and `api/internal/revisionstore` |
| Multipart upload, typed list/get metadata, verified content read and safe errors | `api/handlers/library` with generated Connect messages for metadata |
| Import/list/get CLI | `cli/domains/library`, `cli/manifest.json`, and generated Connect clients |
| Workbench intake/status view | `ui/src/api/library-intake.ts` and `ui/src/features/projects/LibraryIntakePanel.tsx`; intake remains separate from renderable selectors |
| Inspector observation | `api/internal/rendering/physical_glb.go:InspectPhysicalGLB`, called with owner bounds |

The focused verification covers upload/content byte equality, malformed and
oversized GLB rejection, stable pagination and cursor validation, exact replay
and conflicting metadata, cancellation and ambiguous-commit object retention,
exact parent digest resolution, routed storage access, and absence of private
storage keys in projections. Handler tests exercise multipart import and
verified content retrieval. The CLI test uses the registered application and
typed Connect clients for import/list/get. UI tests cover successful import
and catalog refresh, unsupported status, and failure without a fabricated
catalog row. This is not yet a full live public journey across a restarted
service.

### Later creation/review boundary

The next slice can bind an accepted import to a device definition and create
immutable correction revisions with parent digest, edits, tool identity,
before/after observations and held-out reference evidence. Human review must
approve the exact bundle digest and allowed uses through the established
Library review authority; saving a component candidate never does this.
Reference search and AI construction/refinement remain an explicit workflow.
The local-only construction slice is now admitted below. Paid/BYOK inference,
child-agent delegation, external reference fetching, and remote vision remain
unavailable until their owning authority, egress, hard-spend, and
uncertain-charge reconciliation contracts are integrated. Public references
can be recorded locally without remote upload; external media remains opt-in
and `external_ai_upload` is not an import right.

### E5-A UI correction and next owner decision — 2026-10-20

The focused workbench correction now resets the browser file input after a
successful upload (so the same file can be selected again), keeps the generated
candidate ID under collapsed Technical details, and loads additional catalog
pages through the existing typed candidate cursor. The owner-generated ID stays
stable when an upload fails or its result is uncertain; a new ID is assigned
only after confirmed success. Focused UI evidence: the intake panel's three
behavior tests and UI type-check pass. This is source evidence only; the full
live journey and parent-owned gates remain open.

The earlier read-only integration review correctly found that raster generation
and existing media receipts do not themselves provide geometry construction.
Its media-only next-slice conclusion is superseded by the admitted E5 local
construction workflow below. The supported typed Gateway route is distinct
from its raster media executor:

- AI Gateway owns provider-neutral inference/routing and a durable media receipt.
  Its concrete OpenRouter media executor supports only one image-generation
  output, writes to a same-host caller path, and does not populate actual cost;
  its current receipt therefore cannot prove that generation was free or enforce
  a Device Studio parent budget. The route's role and privacy policy are useful
  owner contracts, but a remote role being available is not consent or spend
  authority.
- Image Tools owns raster generation and enhancement jobs, model selection,
  and cancellation. It does not produce geometry. Its Gateway bridge exposes
  image output, not a portable artifact or GLB result. Image outputs may inform
  appearance review but cannot qualify an underlying mesh.
- Agent Manager's typed investigation contract has bounded diagnostic budgets
  and durable operation receipts. It is not an admitted model-construction
  worker or a Device Studio delegation grant. Its presence does not authorize
  child dispatch or aggregate paid spending.
- LPBS account and credit owners provide account-scoped balance/usage and
  metered inference contracts, while the Device Studio security review records
  unresolved callback and scoped lease-to-spend gaps. No verified account-bound
  operation currently reserves a hard, aggregate ceiling across Device Studio
  work and children. Report unknown cost as pending and preserve its hold; do
  not present the Gateway's zero-valued image cost as free.

### E5 admitted slice: local reference-based geometry construction

Deliver one real local AI-to-geometry path, not reference custody alone.
Device Studio calls AI Gateway's typed `InferenceService.Run` with existing
role `extract.structured`, a strict bounded recipe schema, source-asset digests,
reference image attachments, an explicit output-token cap, and
`PROFILE_LOCAL_ONLY`. It does not select a model or credential. Read-only
Ollama inventory and route preview identified a local candidate, but neither
proves inference ran or that the effective endpoint is local. Gateway filters
hosted candidates for this profile, while the Ollama resource accepts
`OLLAMA_BASE_URL`/`OLLAMA_HOST` overrides inherited from its scenario. Typed
inference also permits schema repair and candidate fallback, so profile and
adapter labels alone prove neither endpoint locality nor a one-call bound.
The shared Gateway schema owner enforces the keywords used by this contract,
including exclusive numeric lower bounds, array/string size limits and closed
objects; a digest-pinned copy of the actual geometry schema guards this cross-
owner contract without importing Device Studio's private package.

Add generic `max_provider_calls` to `RunRequest` (zero preserves legacy
behavior; positive values are validated against a Gateway-wide maximum) and
`provider_calls_made` to `RunResponse`. Device Studio sends 1. This total
spans candidate fallback and every validation/repair attempt; count each
adapter `Execute` attempt, including errors, and stop before a further attempt
once spent. The response count is not a receipt of provider HTTP requests
completed because the resource may refuse in local preflight. Preserve existing
fallback and repair behavior when callers omit the limit. Do not add a
role-specific retry switch.

Enforce locality at Ollama's actual `gateway generate` HTTP dispatch boundary
using the effective endpoint resolved once by the resource owner. Extend the
existing provider adapter to pass fixed `--require-loopback` for local-only
generation. The Ollama resource's existing `gateway generate`
owner resolves its effective base URL once and, before any HTTP request,
requires an HTTP(S) URL with no credentials or non-root path and a literal
loopback IP host (`127.0.0.0/8` or `::1`); reject DNS hostnames, non-loopback
addresses and malformed URLs, and use a direct HTTP transport with no proxy.
The Ollama owner must use that checked client/endpoint for the request. Device
Studio must not read provider environment or infer locality from model
identity. If this owner check is unavailable, remote, or uncertain, refuse
dispatch with a truthful unavailable reason; do not fall back to a hosted
provider. An older resource CLI that lacks the flag must fail before dispatch,
which is a safe unavailable outcome.

The finite `device_geometry_recipe.v1` contract has one `profile` enum
(`detailed_device_v1`), one `operation` enum (`construct` or `correct`), a
`parts` array of 1–32 items, and a required `screen` mapping. Each part has a
unique stable id (1–48 lowercase ASCII characters), a `kind` enum (`box`,
`rounded_box`, or `cylinder`), an allowed semantic role (`housing`, `bezel`,
`display`, `button`, `camera`, `port`, `speaker`, `control`, or `detail`), a
`center_mm` triple, a positive `size_mm` triple, `rotation_deg` triple, bevel
radius in millimeters (zero unless the kind is `rounded_box`, where it must be
positive), and one fixed
material enum (`matte_polymer`, `gloss_polymer`, `rubber`, `glass`,
`anodized_metal`, or `neutral_metal`). The origin is the envelope center; x is
width, y is height and z is depth. Parts are axis-aligned except for right-angle
rotations: each `rotation_deg` component is exactly 0, 90, 180 or 270.
Cylinders use local z as their long axis. The screen mapping names one
`display` part and supplies a face enum (`front:+z`, `back:-z`, `left:-x`,
`right:+x`, `top:+y`, `bottom:-y`), a normalized face-center pair, width and
height in normalized 0..1 face coordinates, plus a rotation in 90-degree steps.
The compiler creates the screen as a distinct UV-mapped primitive in the
display part's mesh, so the existing display-node selector resolves it through
the normal candidate binding. A generic box alone cannot satisfy screen
binding. For the canonical front face, horizontal maps to +x and vertical to
+y. The compiler offsets the surface outward by 0.02 mm and rejects it if the
result exceeds the locked outer envelope tolerance. The screen rectangle must
also lie on the actual planar portion of the selected display surface: a
rounded-box screen cannot overlap its rounded edge/corner, and a cylinder's
curved side cannot host a flat screen plane. Unsupported mappings are rejected
with a specific recipe-validation reason before output is compiled.

Material enums resolve to compiler-owned fixed PBR presets, never model-authored
shader/color values: `matte_polymer` is neutral charcoal (base RGB
0.16/0.17/0.18, metallic 0, roughness 0.82); `gloss_polymer` is neutral
charcoal (0.12/0.13/0.14, metallic 0, roughness 0.24); `rubber` is dark gray
(0.07/0.075/0.08, metallic 0, roughness 0.94); `glass` is neutral dark
(0.025/0.03/0.035, metallic 0, roughness 0.08); `anodized_metal` is neutral
silver (0.48/0.50/0.52, metallic 0.82, roughness 0.32); and `neutral_metal`
is gray (0.28/0.29/0.30, metallic 0.68, roughness 0.48). No texture or
external material asset is loaded.

Locked outer dimensions (width, height, depth in millimeters) are immutable
request facts, not model output. Every part must remain inside that exact
envelope with a 0.1 mm tolerance; the recipe cannot change the envelope,
locked screen mapping, or parent dimensions. Reject duplicate ids, missing or
mis-role-bound display parts, out-of-range coordinates, rotations outside the
right-angle set, degenerate solids,
unsupported fields, and extra fields. After conversion to GLB float32 positions,
reject triangles that have actually collapsed or have non-finite geometric area;
do not impose a generic physical minimum that would exclude small buttons,
ports or speaker details. Limit canonical recipe JSON to 32 KiB,
at most 100,000 output triangles and 8 MiB GLB. Set the generic Gateway total
provider-call limit to one for each AI job; invalid output fails cleanly rather
than triggering a hidden repair or candidate retry. Corrections use the same
finite recipe
with operation `correct`, exact parent revision digest, and the same locked
envelope; they create a child candidate and preserve the parent. They cannot
delete or replace the parent or reference arbitrary paths/scripts.

Reference inputs are limited to four caller-selected, immutable, project-local
images and 10 MiB total; accept only supported PNG/JPEG media with bounded
dimensions and verified digest. No URL fetching is allowed. The request also
includes explicit dimension facts and screen-face locks where known. Attach
images to the typed Gateway call only after verifying the expected bytes and
digest. Keep user-supplied claims separate from extracted facts.

The new geometry owner is a pure, deterministic recipe-to-GLB compiler in
Device Studio's geometry domain. It triangulates the three allowed primitive
kinds; `rounded_box` uses an 8-segment rounded-cuboid profile (not a chamfer),
while boxes retain crisp face normals and cylinder caps retain their planar
normals. It creates rounded edges only within part bounds, applies fixed local
PBR material presets, and maps the independently specified screen face. It has no
filesystem, network, command, or subprocess capability. Reuse existing
physical GLB inspection/validation after compilation; never execute
model-authored Python, Blender scripts, shell commands, URLs, or file paths.

The creation job uses the existing `internal/jobs` durable ledger and the
existing `internal/renderjobs.Coordinator` single dispatch/execution owner via
a new typed subject adapter. That adapter validates and stages the immutable
input closure, calls the typed Gateway route with the bounded timeout, compiles
and inspects output, then supplies completion evidence only after verifying
all staged/result bytes. The jobs ledger remains authoritative for state,
attempt, idempotency, cancel intent and recovery; the coordinator remains the
only dispatcher/process owner. Do not add a second queue, worker supervisor or
per-request goroutine lifecycle. Persist intent and input pins before dispatch.
After restart, reconcile the recorded attempt; never automatically repeat an
uncertain Gateway effect. If inference outcome is uncertain, retain job and
input evidence and mark it for reconciliation rather than generate again.

The current construction surface is mounted at
`POST /api/v1/projects/{project_id}/geometry-creations`, with durable status at
`GET /api/v1/geometry-creations/{job_id}` and versioned cancellation at
`POST /api/v1/geometry-creations/{job_id}/cancel`. An uncertain client submit
is recoverable through read-only
`GET /api/v1/projects/{project_id}/geometry-creations/operations/{operation_key}`;
the UI stores the exact request and operation key per project before dispatch,
restores by owner readback after reload, and exposes retry only with the same
immutable request. It never resubmits automatically. A new operation identity
is created only after terminal status and an explicit new-draft action. The project detail UI uses this recovery route; the UI and
`creation start|get|cancel` CLI commands use the create/status/cancel routes. Start requires
an explicit local-AI confirmation, one to four ready project PNG/JPEG images,
locked outer dimensions and a construction brief. The UI enforces the same
four-image and 10 MiB limits and makes no manufacturer-fidelity claim. A
completed job persists an unreviewed intake candidate plus recipe, reference,
provider, model, call and usage lineage. This is source and fake-provider test
evidence until a new lifecycle build and live local inference/native review.

Before POST, the browser must durably persist the immutable request identity.
If local persistence is unavailable, the UI refuses to dispatch and explains
that it cannot safely recover an uncertain submit. After an accepted submit or
cancel, the handler activates the existing render-job pump with the request's
pinned routed context. Activation failure is reported as an accepted/recorded
operation that can be reconciled using the same key; it never creates a new
operation key or advances work from a GET request. The pump remains responsible
for lease validation and bounded progress.

Persist each result through existing Library Intake and revision-store
owners: immutable original and result blobs, expected-head revision write,
exact parent revision digest for correction, reference/source digests, recipe
digest, and reported provider/model/applied settings/usage. A generated
revision remains unreviewed and non-renderable until existing review-binding
workflow qualifies it. A stale-head conflict must be reconciled by reading
back the revision and referenced blob, as E5-A does. Keep public reference
claims and rights separate from approval. Report absent model, usage, or cost
fields as unknown; local-only routing means no hosted-provider spend, not that
every local resource is available or free of compute cost.

The workflow is one explicitly disclosed local-only Gateway call plus local
validation/compilation, not an Agent Manager child run. Project external egress
remains opt-in and is not needed for this path. A project-level AI disclosure,
local-only label, progress/cancel/status/result API and equivalent CLI/UI
projection are required. Cancellation prevents later local stages where
possible; it cannot claim to undo a Gateway call already submitted. If
completion races with cancellation, reconcile job and candidate readback and
retain any result blob already referenced. Do not claim a hard paid cap for a
future hosted path: provider charge is not a hard in-flight reservation. Keep
hosted inference unavailable until there is an enforceable operation-level
spend ceiling and cancellation/reconciliation across all work.

Agent Manager is the canonical owner only if a later workflow requires tool use
or delegated, multi-step work. Its declared workflow/run path carries verified
human or exact parent agent identity, enforces policy admission, and applies
attenuated child grants. For this single recipe operation it adds no value and
its aggregate token/charge budgets are usage accounting, not hard in-flight
provider spend caps. Agent Manager's existing Gateway structured-result
extractor also does not set local-only profile or a caller output cap, so it
must not be used as a substitute for the direct typed Gateway call without an
owner enhancement.

Focused validation covers the generic total call limit across candidate fallback
and repair attempts (including failed-dispatch accounting), unchanged default
retry behavior when omitted,
local-only profile propagation and remote-candidate refusal,
Gateway→Ollama `--require-loopback` propagation, and Ollama owner
endpoint policy (loopback accepted; remote, DNS, malformed, or credential-
bearing `OLLAMA_BASE_URL`/`OLLAMA_HOST` refused before HTTP, with no proxy),
strict recipe/dimension/triangle/output
caps, deterministic compiler output and screen binding, physical GLB validation,
source/reference/recipe/output lineage, immutable revision/head conflicts,
dispatch cancellation and uncertain readback, and API/CLI/UI disclosure plus
unavailable-local-provider behavior. API/CLI/UI should reuse existing Device
Studio and Gateway clients. Additive typed Gateway fields for a generic total
provider-call ceiling and actual call count are required; they do not change
legacy callers when omitted. Root owns
real inference and lifecycle proof; focused tests use fakes. This admitted
slice does not close the iterative held-out-reference, photographic-scene,
device-family qualification, or paid-service obligations.

## Cross-References

[security](../internal/SECURITY.md), [monetization](../business/MONETIZATION.md), [devices](devices.md), [acceptance](../internal/ACCEPTANCE.md), [flows](../concepts/FLOWS.md).

### Inference outcome provenance repair

Gateway `Usage` retains its legacy numeric fields and adds per-value
provenance: unknown, provider-reported, estimated by the Gateway, or mixed
across attempts. Numeric zero means an observed zero only when provenance says
provider-reported; unknown cost is not “free”. Prompt-token estimates stay
labelled as estimates, and metered-provider credits are not USD cost. Aggregated
usage must keep absent contributions unknown rather than converting them to
zero.

Device Studio persists the actual Gateway `AppliedSettings` from a typed
response, not request values echoed into lineage. Immediately after receiving
any typed response, the worker writes a bounded canonical outcome sidecar before
validation or compilation. It contains bounded job/attempt/input/worker
identity, provider/model, applied settings, usage provenance, response digest
and byte count, adapter execution-attempt count, and typed error metadata. It
contains no prompt, raw response, reference bytes, or source data. The existing
native operation evidence owner binds the sidecar on both success and failure.
A transport error with no typed response leaves response outcome and usage
unknown; an adapter execution attempt is not proof of HTTP completion. No
automatic retry or promotion is implied.
For cancellation after native work was stopped, the persisted quiescence record
may bind a complete canonical response sidecar. An absent, partial or identity-
mismatched sidecar is recorded as not bound and never prevents cancellation;
that state makes no claim about whether the provider returned a response.
