# AI model and scene creation

## Scope

Foundation contract, 2026-10-19. The requirement statements below remain the target. The implementation status has changed since this historical foundation note; see “Current implementation status — 2026-10-21” below. The bounded local geometry path is implemented, but the complete reference loop and creation requirements are not accepted as complete.

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

At the start of the 2026-10-20 first implementation slice, the current implementation had useful owners but no creation workflow owner. The following inventory is historical context; current status is recorded below:

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
- At that point there was no Device Studio provider admission or creation-job
  owner. That statement is no longer current; bounded local creation sessions
  and a typed geometry job owner now exist. Agent Manager remains unused for
  this single deterministic recipe workflow.

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

### Explicit physical qualification workflow

Qualification is a separate operator action over an exact immutable device
candidate revision. The review records the source bundle manifest/review/artifact
digests and permitted uses, rights claims, dimensions and uncertainty, display
views/masks, motion and pose scope, mount/support interfaces, and an inspected
native-output job and SHA-256. The shared device revision owner validates the
evidence and appends a qualified revision bound to the candidate's exact parent
revision and digest. Any stale or changed input, missing evidence, incompatible
rights/use, or unsupported scope blocks promotion. Rejection is retained as a
review decision without changing or deleting the candidate. Import, deterministic
correction, starter initialization, AI generation, and renderer success do not
promote automatically. The action is available through the same public owner API,
CLI, and workbench, and does not make a legal rights guarantee or bypass Library
review.

### E5-C: deterministic imported-model material correction

E5-C adds a local deterministic edit to an already imported GLB. It does not
call AI, launch an agent, fetch a URL, or execute content from the model. The
operator first reads a bounded material inventory from the exact candidate
revision, then selects a material by both its observed index and name and
supplies one or more supported PBR values: `baseColorFactor` (four finite
components), `metallicFactor`, `roughnessFactor`, or `emissiveFactor` (three
finite components). Values are bounded to 0..1. Unknown fields, malformed or
oversized JSON, non-finite values, out-of-range values, mismatched material
identity, and no-op edits are refused.

The request binds the exact parent candidate ID, revision and revision digest,
plus a caller-stable child candidate ID. An exact retry of the same child ID
and canonical request returns the same child; a different request under that
ID conflicts. A correction reads and verifies the parent's retained GLB bytes,
parses only the bounded GLB JSON and embedded BIN chunks, and changes only the
selected PBR fields. GLB BIN bytes remain byte-identical, and geometry,
accessor, node, mesh, buffer, and buffer-view JSON values remain unchanged
(the GLB JSON chunk may be reserialized); no external URI is fetched or resolved, and any
unsupported material extension prevents editing that material. The result is
inspected before it is committed as a new immutable child candidate. The parent
is never replaced or deleted.

The child retains the exact parent digest, selected material identity, a
field-level before/after change log, whole-GLB before/after SHA-256 values, and
the unchanged embedded BIN SHA-256. It inherits the parent's unverified rights
claims as claims only. The child remains `unreviewed` and `not_reviewed`; this
edit does not create a Library bundle, review binding, render grant, geometry
qualification, or product-fidelity claim. The workbench labels this as a local
deterministic material edit and displays the lineage and hashes. The CLI
exposes the same exact-parent and material-edit operation. Focused checks must
prove bounds and selector validation, exact replay/conflict behavior,
byte-stable BIN and geometry, immutable parent bytes, and unreviewed child
status through API, CLI and UI owners.

The implementation uses the existing Library intake revision and blob owners,
with exact-parent material inventory/correction REST routes and equivalent
`library intake-materials` / `library intake-material-correct` CLI commands.
The panel stores a pending exact correction request in browser storage before
dispatch and reuses it after an uncertain response or remount. The child
persists a typed material-change lineage; the existing Connect metadata
projection stays compatible, while REST correction and material inventory
responses expose that lineage. Focused owner, handler, CLI, panel and API-client
checks are the source checkpoint; root owns the live import/correction and
before/after native review.

The implementation's corrected gross runtime/config census is 1,028 lines,
128 above the originally admitted 900-line cap. This includes the authored
English, Japanese, and Arabic locale JSON; generated strings, tests, and docs
are counted separately. An earlier 848-line receipt omitted those locale
files and was superseded without raising the cap. Focused checks pass, while
live correction and native visual review remain pending.

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

For construction inference, include explicit geometry guidance in the fixed
system instruction: center the housing at `[0,0,0]`, keep every rotated part's
full extents within `[-W/2,+W/2] × [-H/2,+H/2] × [-D/2,+D/2]`, and bind the
screen to a planar face of its display part with the required stand-off inside
that same envelope. Do not rely on generated output to restate these constraints.
Immediately after a typed Gateway response, persist the exact returned JSON
value bytes in the private job workspace before semantic recipe validation,
provided the value is at most 1 MiB. Bind the sidecar to those bytes with SHA-256
and byte count. If it exceeds the retention bound, preserve only its digest and
size with an explicit `omitted_over_limit` disposition; never truncate and
present a partial value as exact. The retained value is inert evidence and is
never executed. Invalid recipes remain inspectable for an explicit later
correction; the worker does not retry or modify them automatically. A
recognized envelope validation failure records only a typed safe issue
(validated part ID, axis enum, issue code and bounded explanation) beside the
exact retained response. After the native operation stops, the existing
render-jobs owner verifies and stores that exact recipe as an attempt-bound
artifact referenced by native outcome evidence. The failed job's evidence
digest remains the authority for reading it; no host path is exposed. A
correction read returns the prior recipe only after checking job, attempt,
outcome, artifact size and digest. Missing, corrupt or unsupported evidence is
reported unavailable without reconstructing a response.

The UI may present that verified recipe and semantic envelope issue in an
explicit correction draft. It preserves the original project references and
funding route, but creates new operation identities and resets the required
AI confirmation. The operator reviews and submits the new draft. Status reads
never dispatch, and failure handling never retries or submits automatically.

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
provider, model, call and usage lineage. This sentence described the earlier
source/fake-provider-only stage. Subsequent live evidence is summarized below;
it does not qualify the candidate or complete the full reference-loop
requirement.

Create replay resolves an existing idempotency key before reading current
reference metadata or pinning the current executable. It returns the existing
job only when the normalized project, candidate ID, exact brief, locked
dimensions and ordered reference media IDs match the stored canonical subject.
The old worker and source digests remain pinned across application rebuilds;
they are never silently replaced by current bytes. A changed or deleted
reference may make a still-queued historical job fail its pinned-source
verification, but it does not turn a same-key retry into a new operation. A
semantically different request, non-geometry key owner, invalid stored subject,
or route mismatch remains a conflict or refusal. On a concurrent insert
conflict, the coordinator performs one same-key readback and applies this
comparison; it never submits twice.

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

The original one-operation construction slice used one explicitly disclosed
local-only Gateway call plus local validation/compilation, not an Agent Manager
child run. The current parent-session limit permits at most two local calls;
see the current status section below. Project external egress remains opt-in
and is not needed for this path. A project-level AI disclosure, local-only
label, progress/cancel/status/result API and equivalent CLI/UI projection are
required. Cancellation prevents later local stages where possible; it cannot
claim to undo a Gateway call already submitted. If completion races with
cancellation, reconcile job and candidate readback and retain any result blob
already referenced. Do not claim a hard paid cap for a future hosted path:
provider charge is not a hard in-flight reservation. Keep hosted inference
unavailable until there is an enforceable operation-level spend ceiling and
cancellation/reconciliation across all work.

The typed inference operation runs under an allowlisted environment. It
receives the operator's resolved `HOME`, the canonical `VROOLI_BIN` path, and
attempt-private `TMPDIR`; it does not inherit the API server's environment or
secrets. This is required because `api-core/discovery` resolves the local-only
AI Gateway endpoint through the runtime home and canonical scenario-port
owner. This inference operation is separate from recipe compilation: the
deterministic compiler executes through workspace-sandbox as described below.
Blender-specific settings are not part of either geometry worker profile.

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

For a supported envelope rejection, the worker adds a bounded semantic issue
to that sidecar and the native outcome owner stores the exact retained recipe
as a digest-bound job artifact. The creation read verifies the routed job,
attempt, outcome digest, artifact owner and exact bytes before returning it; an
unavailable or corrupt artifact yields no recipe. In the original single-job
flow, the UI presented a new draft using that verified recipe and issue,
preserved references and funding choice, reset AI confirmation, and waited for
an explicit submission. The current local session instead uses the persisted
parent confirmation for unchanged scope and requires a new explicit choice for
a linked changed-scope session. Focused worker, artifact binding, API, panel and type checks
pass. Source/config growth is 624 gross lines against the original 550-line
slice cap (74 over); the separately admitted 80-line correctness allowance
covers atomic outcome replacement and its regression coverage. No live
correction inference or native quality result is claimed.

### E5-D: explicit local photographic scene generation

The existing Image Tools AI, jobs and blob owners are the only inference and
job authorities for this slice. Device Studio resolves the Image Tools API
through `api-core/discovery`; the adapter submits one `text_to_image` request
and forwards job lifecycle/content operations through the existing owner APIs. It does not execute a
model, queue a shadow job, or create a provider fallback path.

The accepted local selection is the installed `sd-1.5` model (local weights
SHA-256 `6ce0161689b3853acaa03779ec93eafe75a02f4ced659bee03f50797806fa2fa`,
4,265,146,304 bytes). Its current explain route is local GPU through Diffusers
because stable-diffusion.cpp is unavailable. This is a readiness fact, not an
inference or quality result. Requests must force that model with
`fallback_policy=local_only`, `allow_byok=false`, `allow_reclaim=false`, one
variation, prompt <= 800 UTF-8 bytes, and one of the two source dimensions
768x512 or 512x768. No hosted retry, account, wallet or paid path is part of
this slice. The UI states that the prompt is sent to the local Image Tools
service and that the output is model-created; an unavailable local model is a
truthful unavailable/error state.

Image Tools persists an optional caller operation ID and server-computed
request fingerprint in its existing jobs domain. A repeated operation ID and
identical fingerprint resolves to the original durable job; a changed
fingerprint is a conflict. Device Studio writes the operation ID and exact
request to browser persistence before dispatch, and after reload it reads the
owner status DTO by ID. It never automatically repeats a request. An explicit
retry sends the same request under the same owner idempotency key. Status
refresh reads an owner snapshot; the explicit wait action uses the existing
block-once job verb. Cancellation displays the returned owner state and does
not claim that a running job has already stopped.

For these idempotent photographic requests, the Image Tools job owner enforces
a 30-minute execution timeout after a lane worker starts the job. Queue wait is
not included in that execution limit.

A successful owner result must have a safe managed `out/` blob ref, PNG MIME,
<= 12 MiB bytes, and the requested dimensions. Device Studio copies that
verified byte stream into the existing project media import with
`device-studio-generated-{job-id}.png` as its filename. This binds the source
media row to a durable Image Tools job ID without presenting a generated
image as reviewed geometry or licensed public content. The source digest and
owner job/model/tier are shown as provenance; absent facts remain unknown.
The saved image can be selected in the existing photographic still workflow.
Users must adjust any screen-entry geometry or mask manually; generation does
not calculate calibration. This contract does not qualify pixel accuracy or
photographic appearance. Root owns the real local generation, resource,
output review and native rendering proof.

### E6 admitted slice: explicit funding for geometry chat

Geometry construction may expose three caller-selected routes: the existing
local-only route, a BYOK route owned by the OpenRouter resource credential
authority, and a Vrooli-credits route owned by LPBS metered inference. The
selected route is immutable job input and participates in idempotent replay;
each remote route resolves to exactly one provider candidate and one provider
call, with no cross-funding fallback or automatic retry. Remote submission
requires a separate project-level disclosure and confirmation that selected
reference images and the geometry brief will be sent to that provider. The
Device Studio browser never receives credentials or session tokens.

OpenRouter credentials remain configured through the OpenRouter resource
owner. LPBS credit access requires an explicit desktop-owner action to share
the signed-in LPBS refresh credential with the Gateway credential authority;
the desktop entitlement lease is not an inference credential. Clearing the
shared credential is local owner state and does not claim remote revocation of
already-issued access tokens. Account, funding, or network unavailability
must leave the local route usable and must not trigger a hidden fallback.
Photographic scene generation remains explicitly local-only until a metered
media owner contract exists; this geometry-chat route does not fund it.

#### E6 spend-ceiling continuation: owner-enforced admission

The remote geometry route accepts one explicit total ceiling in USD micro-units
as immutable job input. It is an upper bound, not an estimate or an instruction
to spend that amount. Gateway admission derives a finite prompt-token bound from
current owner-resolved endpoint context metadata and the output cap, and bounds
attached images by their existing request count. If any of these limits, a
required provider price class, or the provider's request cap cannot be
enforced, the request is refused before provider dispatch.

For BYOK, the OpenRouter owner resolves a pinned model and a bounded set of
eligible endpoints immediately before dispatch, then builds a short-lived
cap authorization bound to the exact immutable input fingerprint, reference
SHA-256 set, funding route, configured credential authority/account identity,
model/provider policy, input-token bound, image count and output cap. This is a
hard-cap authorization, not a priced quote. The UI may show `Maximum spend: known`,
`Estimated price: Unknown`, and `Current pricing: unavailable`; it must not
invent a quote. The owner authorization is checked at dispatch and rejected if
the bound inputs or endpoint context limits changed. It applies
`require_parameters`, disables implicit fallbacks, restricts the allowed
endpoint set, and sends `provider.max_price` for prompt, completion,
per-request, and per-image prices. Those four caps are budget
allocations whose worst-case aggregate (bounded prompt tokens times prompt
price, output cap times completion price, image count times image price, and
one request fee) cannot exceed the user's ceiling. The underlying provider
filter is documented by OpenRouter's [Provider Routing reference](https://github.com/OpenRouterTeam/docs/blob/main/guides/routing/provider-selection.mdx#max-price).
Endpoint prompt/completion rates arrive in USD per token while OpenRouter's
corresponding `max_price` values are USD per million tokens; request and image
caps remain per unit. Endpoints without separate request/image fees receive
explicit zero caps for those categories. A lower input-cache-read rate is
accepted only when it does not exceed the prompt rate; unknown/additive fee
classes, non-finite prices, and unbounded context metadata exclude that
endpoint before dispatch.
Current model context limits must come from owner-resolved endpoint metadata;
stale local policy alone cannot establish the input bound. OpenRouter's own
`max_price` filter enforces the unit limits and excludes endpoints whose
required price categories are unavailable. If an endpoint has an unsupported
fee class or cannot be restricted by all four caps, the remote route refuses
before dispatch.

For Vrooli credits, LPBS owns USD-to-credit conversion and reservation. The
request carries a USD ceiling, never a caller-computed credit amount. LPBS
resolves the authenticated account's applicable cost-based limit and current
`cost_multiplier`, creates the same exact-input-bound cap authorization,
reserves the ceiling before calling OpenRouter, applies the
same enforceable provider price limits, and settles only against
provider-reported actual USD cost. Authorization is bound to the authenticated
account and the current limit conversion used for reservation. A provider error releases the
reservation only when the owner proves no billable dispatch occurred. A
dispatched request with unknown cost keeps its reservation pending for
reconciliation; it is never reported as zero-cost or silently charged above
the ceiling. Actual usage above the ceiling becomes an owner
reconciliation error; it must not increase the user's charge.

The durable Device Studio job binds the selected route, USD ceiling, operation
and exact source references in its immutable fingerprint. It reports the
owner cap-authorization identity, current context snapshot/freshness, applied
cap policy, and reservation status after dispatch; current price may remain
unavailable. Queued work shows estimate `Unknown`, cost `pending`; a
provider-reported cost is `known`; a dispatched operation without authoritative
cost is `known`; a dispatched operation whose owner status is
`pending_reconciliation` stays visibly pending; missing or unrecognized
accounting remains `unknown` and retains its applicable hold. Settled Vrooli
credits use an optional lineage value so an observed zero remains distinct
from absent accounting. This accounting stays
visible after reload and does not expose credentials. Local generation remains
available offline and does not acquire, reserve, or settle a remote funding
route.

## Current implementation status — 2026-10-21

The earlier “no creation workflow owner” and “source/fake-provider only” notes
above are historical. The current local route has a durable typed job owner and
a durable parent session owner in `api/internal/renderjobs` and `api/internal/jobs`.
Session policy pins project, candidate, brief, envelope, reference-media IDs,
local-only funding, and 4096 output tokens per call. The session admits at most
two child operations/provider calls. SQLite admission reserves each operation
atomically; a start claim checks that the scope remains active. Cancel closes
admission before requesting child cancellation and leaves unresolved cleanup
visible for owner-driven resumption. Session policy and children survive
SQLite reopen. A scope change is returned as `needs_confirmation` without
dispatch; an explicit confirmation creates a separate linked session instead
of mutating the original scope.

The supported creation workflow is local-only and uses one disclosed parent
confirmation for unchanged-scope iterations. The durable owner readback, not
browser local storage, determines whether that confirmation still applies.
An unchanged-scope refinement therefore does not ask for another checkbox.
Changing brief, dimensions, references, funding, or other pinned scope requires
a fresh explicit choice and a linked session. This is a bounded local workflow,
not an Agent Manager agent run, and it does not imply hosted provider spending
is capped or supported.

The recipe compiler is a separate fixed Go executable through workspace-sandbox.
Its current profile requests 4 GiB virtual address space, 45 seconds CPU,
60 seconds wall time, 32 processes/tasks, and 64 open files; it sets
`GOMAXPROCS=2` and the soft `GOMEMLIMIT=256MiB` target. The address-space cap
does not claim physical RAM or RSS containment. The parent job's limits and
remote sandbox process limits are separately owned and are not an aggregate
memory guarantee. Network and writes outside the mounted workspace are
denied. OS-read confinement is not established. Arbitrary scripts, imported
Blender execution, shell commands, URLs, or model-authored paths are
unsupported and are never executed.

Focused durable-session tests cover atomic two-child admission,
concurrent-admission bounds, cancel/claim race, and SQLite reopen/resume in
`api/internal/jobs/scopes_test.go`; UI tests cover owner-readback continuation
and confirmed scope expansion. The current WSS live compiler smoke is recorded
in [the 2026-10-21 evidence receipt](../internal/goal/evidence/E5/current-creation-session-build.md).
The smoke has validated deterministic compilation and sandbox cleanup, but its
temporary GLB was not retained as a product artifact. Earlier local visual
baseline/refinement candidates remain partial and unreviewed. The fresh
parent-session baseline path has not yet been exercised against the current
served build, so do not mark DS-CREATION-AUTONOMY or the overall P0 creation
requirements complete. Held-out reference comparison, photographic scene
generation acceptance, family qualification, and human review remain open.
