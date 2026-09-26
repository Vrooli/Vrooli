# Architecture

## Purpose Of This Document

This is the authoritative architecture map. Historical design analyses are context, not contracts.

## Scenario Shape

Browser Automation Studio is a local Vrooli scenario with Go API and CLI surfaces, a React UI, and a Node Playwright driver sidecar. It turns visual workflows into typed browser instructions and durable replay evidence.

## System Boundaries

| Surface | Responsibility | Must not own |
| --- | --- | --- |
| `ui/` | Authoring, visualization, operator feedback | Execution semantics or wire types |
| `api/` | Validation, workflow compilation, orchestration, storage | Browser-process control |
| `playwright-driver/` | Session lifecycle and typed browser actions | Workflow persistence/policy |
| `packages/proto/` | Cross-language action and execution contracts | Product business rules |

## Contracts And Data Flow

UI workflow intent is validated and compiled by the API into proto-backed instructions. The API sends those to the driver; the driver returns normalized outcomes and artifacts. The API persists evidence and exposes it to UI, CLI, and replay export consumers.

SessionManager owns the runtime page stack for every driver session, including
pages created by workflow popups while recording is off. TabHandler mutations
remain idempotent with context page lifecycle events. Recording page identity
and opener reconstruction feed the workflow generator's narrow multi-page
path: stable page IDs bind actions to deterministic tab-stack indices, a popup
may follow its opener action only when lifecycle timestamps prove that causal
order, and an independent page may be opened at first use. Ambiguous timing,
missing identities, and closed targets fail closed. Full recording-to-saved-
workflow replay across tabs/frames in a fresh context remains unqualified.

## Shared Infrastructure

SQLite is routed through the scenario database layer; artifact storage and process lifecycle are scenario-managed. Test Genie owns scenario-suite execution. Rehabilitation setpoint evidence uses the root control plane's typed scenario-status and lifecycle-freshness bindings; BAS does not duplicate the artifact fingerprint algorithm. Evidence cannot qualify against a managed candidate whose lifecycle artifacts are stale or whose freshness verdict is missing.

## Extension Rules

Add a capability end-to-end: typed proto action, compiler validation, driver handler, UI authoring, then behavior-focused tests. Never add a parallel JSON instruction dialect.

## Architecture Maturity

The V2 typed instruction path is active. Deployment/commercial hardening is tracked by the operational and business documents, not inferred from old plans.

## Intentional Deviations

The established `browser_automation_studio.v1` proto package prefix is retained because changing it would be a cross-language wire-contract migration, not a cosmetic validator cleanup.

## Documentation Architecture

This document and [Domains](DOMAINS.md) are canonical. Operational procedures live in `docs/operations`; stable facts live in `docs/reference`; old plans are historical.

## Cross-References

- [Domains](DOMAINS.md)
- [Flows](FLOWS.md)
- [Data](DATA.md)
- [Integrations](INTEGRATIONS.md)
- [Seams](../SEAMS.md)
- [Testing](../../../../docs/TESTING.md)

## Proposed browser-first target — 2026-09-21

Status: engineering target for the operator-requested rehabilitation preparation
(2026-09-22 UTC), linked by PRD OT-P0-005. This does not claim the design is
implemented. Qualification bands and decisions are in docs/internal/TESTING.md
and docs/internal/REFRACTOR_CONTRACT.json. Evidence, alternatives,
preservation matrix, proposed performance bands, and qualification methods are in
[the refactor assessment](../internal/REFRACTOR_ASSESSMENT.md). Issue state is in
[PROBLEMS.md](../PROBLEMS.md#refactor-investigation-register--2026-09-21).

The target product is a persistent browser workspace that observes ordinary
human activity and can turn selected history into validated automation. It also
serves agent browser tasks, evidence capture, and isolated web/desktop/mobile
validation. These uses share typed contracts while retaining different session,
resource, and evidence policies.

### Ownership

| Owner | Authoritative responsibility | Boundary |
| --- | --- | --- |
| Session coordinator | Admission reservation, session purpose, lease generation, cancellation and bounded resource lifetime | Reuse requires explicit release and compatible target/profile/configuration; user and test sessions are distinct. |
| Browser runtime adapter | Browser/context operations and capability negotiation | Managed Chromium, external Electron and Android WebView retain their own capability declarations and lifecycle owners. |
| Interactive transport | Ordered input receipts and current rendered state | Frames can coalesce; accepted key/button transitions cannot disappear or replay on reconnect. |
| Recording journal | Ordered observations, event identity, durability acknowledgement, reconnect recovery | Keep raw observations immutable; report gaps and pending durability explicitly. |
| Profile store | Supported authentication state, tabs, snapshot versions and recovery | Serialize writable ownership; checkpoints are atomic; failure cannot produce a persisted acknowledgement. |
| Workflow compiler/deriver | Transform selected observations or AI proposals into typed workflow candidates | Preserve source provenance, target identity, assertions and versions. One normalization policy serves API and UI. |
| Evidence service | Capture policy, completeness, integrity, retention and authorized retrieval | Heavy artifact work cannot stall interactive input; required evidence failures affect the result. |

Screenshot integrity decoding is owned by `api/automation/execution-writer`. It
checks encoded dimensions before full PNG/JPEG validation, reserves a shared
process-wide estimated raster budget across every `FileWriter`, then releases
the reservation before storage I/O. The 192 MiB budget charges ten estimated
bytes per pixel so it admits one measured 16.384 MP full-page raster at a time.
The higher estimate accounts for Go heap pages retained after earlier decodes:
ten concurrent writer calls peaked at649MiB PSS with admission effectively
unlimited,271MiB with the original five-byte estimate, and146MiB with ten bytes
per pixel. The managed16.384MP fixture remains accepted; larger estimates fail
explicitly before raster allocation. The BAS API's `.vrooli/service.json` run
environment also owns `GOMEMLIMIT=96MiB`. On Linux, ten managed concurrent
full-page PNG writes peaked363,218KiB combined API+driver PSS and returned to
206,058KiB at60seconds, below the300MiB idle target; the settled idle reading
was106,257–106,559KiB. The maintained capture cohort remained within its2s p95
band. GOMEMLIMIT is a soft Go heap target, not a hard cap: the measured active
PSS exceeded300MiB. Valid large JPEG managed capture, Windows/macOS memory, CPU,
storage backpressure and long-soak growth remain unqualified. The storage package
uses one MIME-to-extension rule across FileStorage, MemoryStorage and MinIOClient:
JPEG objects use `.jpg`, PNG objects retain `.png`, and GIF uses `.gif`; this keeps
durable object identity consistent with validated bytes and response metadata.
| Workspace UI | Browser chrome, viewer, timeline, workflow editing and agent control | Subscribe by responsibility; frame rendering does not drive whole-workspace React updates. |

These are module boundaries within the current deployment. They do not prescribe
new microservices or replace the Go/Node/React stack.

The Playwright driver serializes live input per browser page so a delayed pointer
move cannot let a later button transition overtake it. The queue has a fixed
pending-work bound, coalesces superseded adjacent pointer moves, and returns the
monotonic sequence of the applied input through HTTP and WebSocket receipts.
Each client input carries a stable ID. The driver caches a bounded set of
per-page receipts and returns the same result for duplicate delivery; reusing an
ID for different input is rejected. The UI retains unacknowledged WebSocket
inputs and replays them over HTTP in order after disconnect before sending
synthetic pointer releases. This recovers dropped acknowledgements within the
driver's retained receipt window. A full UI page reload, server-side session
cancellation, and older retries after receipt eviction remain unqualified.
The driver's frame timing counters describe processing stages only; they do
not measure input-to-paint, network transit, or UI decode/draw latency. W164
adds a 1,000-input local measurement from the workspace browser's captured
pointer event through the applied receipt to the actual canvas pixels: p50
36.70ms, p95 39.30ms, p99 40.00ms, with all samples correlated. This establishes
the local numerical band only. The independent remote cohort and governed
sensor binding remain open; earlier driver-only and command-start measurements
are diagnostic. See
`../internal/evidence/rehabilitation/interactive-feedback-browser-clock-2026-09-24.json`.

### Delivery paths

~~~mermaid
flowchart TB
    HUMAN[Human workspace] --> INPUT[Ordered input channel]
    AGENT[Authorized agent or workflow] --> INPUT
    INPUT --> SESSION[Session coordinator and runtime adapter]
    SESSION --> BROWSER[Owned browser or external target]
    BROWSER --> FRAME[Latest-frame stream with byte and age limits]
    FRAME --> HUMAN
    BROWSER --> OBS[Typed observations with page and frame identity]
    OBS --> JOURNAL[Durable recording journal]
    JOURNAL --> ACK[Commit acknowledgement and replay cursor]
    ACK --> HUMAN
    JOURNAL --> DERIVE[Deterministic or AI-assisted candidate derivation]
    DERIVE --> VALIDATE[Typed compiler and independent assertions]
    VALIDATE --> VERSION[Versioned workflow catalog]
    BROWSER --> EVIDENCE[Bounded artifact pipeline]
    EVIDENCE --> MANIFEST[Evidence manifest and retention owner]
~~~

Use one negotiated viewing route per session/client, with explicit fallback.
Separate replaceable visual frames from durable ordered actions. Bound frame
bytes, age, and decode work; isolate slow subscribers. Include session/lease
generation, page/frame identity, input sequence, event sequence, and timestamps
in correlation data. Invalidate old-generation frames and inputs on tab switch,
resize, or ownership change.

Frame diagnostics use one bounded sample window for counts, percentiles and bytes.
Throughput divides those window totals by API monotonic observation time, retaining
idle time but excluding evicted history. Lifetime counters serve cadence only.
Driver/API processing durations exclude time waiting for the next socket message;
they do not measure network transit or input-to-paint latency. Keep that limitation
visible in diagnostic labels and bottleneck descriptions. UI consumers share the
frame-streaming type contract instead of declaring the same server fields twice.

Record browser actions independently of workflow creation intent. The journal
must distinguish observed, queued, committed, and incomplete data. Commit before
claiming durability. Deduplicate retried event IDs and retain an acknowledged
cursor across reconnects. A bounded buffer can evict committed events; eviction
of uncommitted events must surface a recording gap.

The recording domain schema must include the journal tables used by its repository.
Production and leased test pools initialize the same embedded schema; repository
tests must exercise that schema instead of defining private tables. Adding a
missing journal table is declarative bootstrap and preserves existing rows.

The repository owns journal order, immutable event identity and complete query
results. A process-local tail must not allocate durable sequence numbers, mask
failed commits, alter totals or substitute for offset queries. One append path
must commit before success or broadcast, reject conflicting identity reuse, and
return the existing commit for an identical retry. HTTP ingestion must propagate
that outcome. Unused alternate recording APIs and their cache policies are removed.

Reload, back and forward share the same post-browser recording owner: update the
known page, commit the observed action, then notify listeners. A missing session
or failed commit must remain explicit after the browser effect. Journal reads
must decode an entire valid JSON document, including rejection of trailing data.

Use explicit replacement/delta input semantics. Empty values, composition,
clipboard operations, frame identity and same-selector actions in different tabs
are first-class cases. The UI may summarize events, but summaries do not alter
the immutable observation record. AI may propose parameterization, grouping,
waits, and assertions; validated versioned candidates remain the automation
artifact. Repeat-effect authorization precedes replay qualification.

Protect sensitive values at capture and export boundaries. Retain secret
references or redacted observations where appropriate. The separate
credential-use executor policy does not establish passive-recording redaction.
The passive recorder must exclude values from password, hidden, one-time-code,
and payment-autocomplete fields from both event payloads and element metadata;
preserve the action and selector so the recording remains useful. Historical
recordings must be redacted at the API persistence/read boundary without
mutating saved rows during ordinary reads. AI element extraction omits
data-attribute selectors for these fields and masks their rendered text before
its full-page screenshot is taken. Export paths and legacy at-rest cleanup still
require separate qualification.

### Recorded action conversion target (BAS-WORK-011)

The workflow deriver shall produce typed WorkflowDefinitionV2 nodes directly and
reuse the compiler's ActionDefinition builders. Remove its parallel V1 node/config
registry and second lossy V2 projection. Recording adapters only translate actual
capture fields (full text snapshots, absolute scroll coordinates and drag phases)
into the compiler's parameter vocabulary. Unknown actions fail with action index
and kind; they never become clicks. The service propagates conversion failure and
the handler passes the typed candidate to catalog validation without a JSON roundtrip.

A single recorded page binds to the fresh replay page. Frame selector paths now
produce frame-switch transitions during generation, but the full frame replay
journey is not yet qualified. Driver page identity now travels from the
page-level event route through `ActionTelemetry` and Go action conversion, so
target-aware merging and the multiple-page refusal can distinguish captured
tabs. Multi-page generation still refuses until lifecycle/opener relationships
become portable logical tab bindings and alternating-target replay passes in a
fresh context. Raw recording history is unchanged. Snapshot merging uses the
final full value within an identical page/frame/selector and does not mutate
journal payloads. Explicit submit boundaries remain distinct. Capture omission
repairs remain RF-004 and are measured separately.

### Typed action execution target (BAS-WORK-012)

Driver handlers shall apply the complete declared interaction semantics. Click
button/count/modifiers/delay/force and keyboard modifiers must reach the browser.
Input replacement, explicit append, delay and submit must have distinct effects.
Scroll targets, absolute axes and relative deltas must act on the specified element
or viewport without resetting omitted axes. The protobuf adapter owns enum spelling;
handlers own browser effects and cleanup. Modifiers acquired for an operation must
be released on success and failure, without leaking into later instructions.
Independent DOM event/state fixtures, not echoed parameters, qualify these effects.

### Runtime purposes and portability

- Interactive browsing uses an explicitly owned persistent profile and supports
  human intervention, tab/history continuity, and passive observation.
- Validation uses isolated leased state and fixtures, exact workflow revisions,
  deterministic capability declarations, and failure evidence.
- Agent tasks use task authority, budgets, observable postconditions, and
  attributable attempts; an uncertain side effect is not automatically retried.

Keep external target process, device discovery, forwarding and device recording
under scenario-to-desktop/scenario-to-android/device-control. BAS owns its
attachment and evidence correlation. Host remediation belongs to the control
plane. A missing capability is explicit, not silently emulated with another
target.

The existing bundle lists Linux/macOS/Windows x64 artifacts. An accepted release
matrix must name OS/architecture combinations and include native clean-machine
receipts. Propose macOS arm64 as an explicit row; do not infer it from a macOS
label. Package the declared Node/browser/native dependencies, resolve writable
storage from the runtime owner, and qualify upgrade/rollback and process cleanup.
Preserve saved workflow/profile data with validated versioned conversion where needed.
Keep one supported runtime path; bounded one-time converters live outside the
shipped runtime and retain recovery data until conversion is verified.

### Lifecycle and evidence invariants from the follow-up investigation

The 2026-09-22 UTC [isolated probes](../internal/REFRACTOR_ASSESSMENT.md#follow-up-isolated-reproductions--2026-09-22-utc)
strengthen these proposed boundaries. They are targets, not claims about shipped
behavior:

- **One lifecycle across decorators.** Execution sinks expose finish/drain and
  abort behavior through an interface. Wrappers propagate it. Completion,
  failure and cancellation finalize durable evidence and release queue workers,
  accumulators and lease references. A terminal event accepted before finish
  cannot disappear as a side effect of cleanup. Late writes have an explicit
  rejection/recovery policy.
- **One ordered input owner per session lease.** Each accepted input has a
  monotonic applied sequence. Replaceable pointer motion may be coalesced within
  a byte/age budget; button/key/modifier transitions retain order. Cancelling an
  HTTP request alone does not prove the browser action did not occur.
- **Observation identity differs from time.** Record unique event IDs plus
  session/page/frame epoch and sequence; timestamps describe occurrence, not
  uniqueness. A successful transport response and a durable journal commit are
  distinct states. Acknowledgements name exactly which events committed.
- **Evidence correlation uses real request identity.** Distinct request objects
  remain distinct even when URLs, methods or string representations match.
  Lost/evicted request evidence is explicit. Redact sensitive input before it
  enters recovery storage, transport, persistence or inference.
- **Caches serve durable history.** A bounded recent-events cache cannot define
  the complete event count or make older actions inaccessible. Paginated and
  filtered queries reconcile with the journal. Execution accumulators retire
  after durable finalization; retained-result policy has explicit bounds.
- **Resource ownership survives concurrent failure.** Session admission reserves
  capacity before launch; launch retries coalesce; shutdown owns in-flight
  launches. A late browser cannot escape shutdown or overwrite another live
  browser's only ownership reference.

Qualify resource targets with allocated and retained Go/Node heap, PSS, swap,
goroutine/worker counts, queue bytes and workload identity. RSS or a dependency-
health green state alone is insufficient. Baselines include running artifact
identity as well as source digest so implementation comparisons are meaningful.

### Profile commits and semantic preservation

The [profile/replay investigation](../internal/REFRACTOR_ASSESSMENT.md#profile-and-replay-investigation--2026-09-22-utc)
adds these proposed requirements to the same owners:

- The profile owner commits metadata and encrypted browser state as one coherent
  version. Failed/interrupted writes preserve the previous committed version.
  Concurrent tab, history, authentication and workflow updates serialize or
  report a version conflict. Closing a browser has an explicit commit/recovery
  result; a failed save cannot be reported as persisted or lose its only retry
  association. Route callers through one aggregate operation.
- Profile recovery distinguishes absent, intentionally empty, locked, corrupt
  and incomplete state. Listing can expose safe metadata/recovery status without
  decrypting or emitting authentication values. Default selection cannot turn a
  recovery failure into an apparently new signed-out workspace. Installation,
  upgrade and backup workflows own encryption-key continuity.
- Human input distinguishes text insertion/composition from key transitions,
  shortcuts and pointer modifier state. Qualify clipboard and IME behavior on
  each supported platform, including failure/reconnect and held-key recovery.
- Workflow derivation uses one versioned typed semantic conversion. Preserve
  modifiers, click counts, both scroll axes and supported action kinds.
  Unsupported observations remain in the journal with a clear derivation error;
  they cannot silently become executable clicks. Schema-valid output alone is
  insufficient preservation evidence.
- Replay targets are logical page/frame bindings derived with lifecycle
  observations, opener relationships and stable frame locators. Map those
  bindings to new runtime pages during execution. Captured session/page UUIDs
  are provenance, not portable replay locators. Derivation must account for the
  relevant context outside a selected action range or report missing context.

### Session isolation, frame lifetime and retention progress

The [session/frame/retention investigation](../internal/REFRACTOR_ASSESSMENT.md#session-frame-and-retention-investigation--2026-09-22-utc)
adds these proposed invariants:

- A retried start preserves an active operation. Execution ID equality does not
  prove its previous operation died. Recovery fences the prior generation only
  after explicit cancellation/expiry evidence. Admission counts reservations as
  well as inserted sessions.
- Clean reuse has one owner for the supported origin/storage boundary, active
  page selection and page maps. Its failure leaves an explicit recoverable or
  failed state. A stuck transient phase cannot become indefinitely exempt from
  resource recovery. Persistent human sessions and isolated test sessions retain
  distinct policies.
- Frame decoding has bounded admission, not only a post-decode stale check.
  Sequence and session/page generation travel through all asynchronous work.
  Disposed connections cannot revive when config/blob/decode promises finish.
  Fallback depends on frame freshness, not just socket connection state.
- Close/flush returns structured artifact and resource outcomes. Failed page,
  context or detach operations retain a bounded recovery owner. Returned evidence
  references distinguish committed bytes from requested destinations. Required
  evidence failures affect the terminal result.
- Retention computes protected history independently of batch selection and
  preview subsets. Repeated bounded work makes progress without deleting active
  or protected evidence. Keep this policy coherent across indexed artifacts and
  directory-budget enforcement.

### Instruction identity and failure finalization

The [execution investigation](../internal/REFRACTOR_ASSESSMENT.md#execution-retry-and-cancellation-investigation--2026-09-22-utc)
adds these proposed requirements:

- Instruction admission atomically validates lease ownership and allowed phase
  before any cache lookup or effect. A failed transition is a rejected command.
  Apply the same ownership envelope to reset, input and other mutating surfaces.
  Parse awaited input before reserving the session; reserve synchronously before
  dispatch. Release only an executing phase still owned by that instruction,
  preserving a concurrent close/reset. Retain a settlement promise alongside the
  in-flight reservation. Reset joins it before mutating browser state. Close
  marks a live operation interrupted, closes its active owned page (or detaches
  an external target), then joins the route's uncertain receipt before releasing
  the lease. The reservation lasts until the action settles; pooling and idle
  cleanup respect it. Live timing and external-target interruption still need
  owner qualification.
  A same-execution start retry observes the
  existing session without resetting browser state or declaring an active action
  abandoned. Recovery requires independent cancellation/expiry evidence.
- Node identity is distinct from dynamic invocation identity. Loops/subflows
  allocate invocations; declared retries allocate attempts; transport retries
  retain their operation identity. Bind each to the lease/generation and payload
  digest, with explicit conflicts for identity reuse with different content.
  The Go session assigns a monotonic transport sequence inside its admitted lease;
  start receipts restore its high-water mark and matching live wrappers are reused.
  One driver response map retains bounded receipts; the high-water mark survives
  response eviction/reset so forgetting a result cannot authorize repeating it.
  A new lease starts a new sequence. Optional idempotency headers must identify
  that exact lease and sequence; arbitrary old header keys fail explicitly.
- Record operation states that distinguish not-started, applying, completed,
  known-safe failure and uncertain effects. A lost response or unexpected throw
  cannot authorize automatic repetition. Reconcile with an independent
  postcondition or require the caller's repeat decision where uncertainty remains.
- Instruction admission requires the active execution ID and exact unreleased
  lease token after request-body parsing, before phase changes, cached outcomes or
  browser effects. Rejected old ownership cannot extend session activity. The Go
  session transports its admitted owner/token on every instruction request.
- Every execution path persists terminal evidence using a bounded context that
  survives action cancellation. Retain already-collected diagnostics before
  disposal, attempt failure capture within its budget and expose missing reasons.
  One executor outcome owner provides a 30-second persistence/event budget,
  retains request routing values, and joins storage errors with the action cause.
  Linear, graph, nested subflow/loop and synthetic outcomes use this same boundary;
  Completed execution cleanup releases outcome, timeline and configuration
  accumulators after all legitimate writes. Workflow and archive-import callers
  share this terminal cleanup contract; persisted files remain the read authority.
  Checkpoint updates apply to successful linear outcomes only. The file writer
  must acknowledge timeline, manifest and index writes before returning success;
  partial files may remain after failure and are evidence, not a completed receipt.
  Execution payloads use the shared typeconv owner to preserve structured values,
  JSON field names, exact signed integers and the outcome schema/version. Raw
  maps remain objects even when their keys resemble proto envelopes. Conversion
  failures prevent a successful write receipt; debug-string fallback is forbidden.
  Existing malformed historical strings remain unqualified evidence.
- Executor finalization joins close and artifact persistence failures with the
  action outcome. Its bounded cancellation-independent context retains routed
  storage/owner values. A missing artifact whose download fails is an error, not
  a valid path. Video, trace and HAR share one resolution/persistence policy;
  source evidence remains owned after a failed import. The artifact writer rejects
  unreadable, malformed or unstored evidence; published trace/video references
  require retained bytes. HAR publishes sanitized derivatives only, with store
  failure distinguished from a successful supported inline derivative.
- Session close and lease release share one terminal operation. Concurrent
  callers wait for its result; an in-progress marker is not acknowledgment.
  A canceled waiter stops waiting without canceling the owning request. Failure
  retains ownership for explicit retry; terminal notification follows explicit
  driver acknowledgment. HTTP200 with missing/false success cannot release ownership.
- Driver close callers share one in-flight result. Failed cleanup retains the
  session in closing, its recording buffer and completed teardown stages for an
  explicit retry. Pre-close flush failure prevents context disposal; successful
  stages and video moves are not repeated. Published artifact paths must refer to
  real files. External target shutdown detaches BAS without closing owned pages.
- Explicit screenshot actions follow the same declared retry and continuation
  policy as other actions. A reported capture failure remains a failed step even
  when the caller permits the workflow to continue; passive telemetry policy does
  not turn the action into success. When image persistence is enabled, an explicit
  screenshot requires retained image bytes and a complete storage receipt. A
  storage failure is recorded as a failed outcome and returned to the executor.
  Passive screenshots remain optional; an explicit no-artifact profile still
  discards them. Encoded images must never be truncated to meet a byte budget.
  Omission records its reason and leaves the caller's original capture unchanged.
- Required vs optional capture is explicit. Finalization combines action,
  artifact and resource outcomes without erasing a failed required capture or
  close operation. Cancellation receipts distinguish requested/accepted from
  applied/quiescent and terminal cleanup.

Qualify the wire contract across Go, driver and callers with independent effect
counters. Per-layer tests that supply successful synthetic retry responses are
useful controls but do not prove that the real driver will perform a new attempt.

### Reuse compatibility and capture ownership

The [reuse investigation](../internal/REFRACTOR_ASSESSMENT.md#reuse-profile-and-evidence-investigation--2026-09-22-utc)
shows why a released lease and matching labels are necessary but insufficient
conditions for transferring a context:

- Resolve and validate requested target, profile and required capabilities
  before selecting either a reused or fresh resource. External target identity,
  validation context and isolation lease remain mandatory on every path.
- Keep an immutable effective context descriptor separate from the new request.
  Compatibility includes target kind/identity, profile identity and committed
  version, proxy, locale, viewport/mobile traits, browser launch settings and
  creation-time evidence capabilities. A label map is a pool hint, not an
  authorization or compatibility proof. Reject incompatible requests or acquire
  a context that actually satisfies them.
- Define clean test isolation separately from continuation of a person's signed-in
  session. A reset cannot claim a new profile merely by replacing metadata. The
  reset/import contract covers the supported origin/storage matrix and permissions.
- Give each execution capture an explicit beginning and end with immutable
  artifact ownership. Finalize the prior capture before transfer. Rotate capture
  where supported; create a new context when its HAR/video setup cannot satisfy
  the new request. A returned path identifies verified evidence for that execution,
  not a retained destination from another owner.
- Treat required capabilities, collection policy and permission to retain evidence
  as distinct contracts. A false requirement does not by itself prohibit capture.
  Report effective capabilities and capture scope at admission and finalization.

Qualify same-profile continuation, cross-profile refusal/recreation, clean test
state, changed evidence requirements and managed/desktop/Android transitions.
Preserve the existing fresh-mode, unreleased-lease and distinct-label controls.
No live contamination or artifact mixing has been established by the synthetic
probes; their reproduced ownership failures define the next fixture boundary.

### Producer lifecycle, effective controls and evidence projection

The [stream/live-fixture investigation](../internal/REFRACTOR_ASSESSMENT.md#stream-lifecycle-and-live-fixture-investigation--2026-09-22-utc)
extends the generation boundary from the viewer to the capture producer:

- A stream generation owns its pending start, capture handle and transport.
  Replacement, stop and fallback use one coordinator. Cleanup removes only its
  own registry entry; a late start disposes acquired resources before returning.
  A failed start closes its connection. Every active resource remains stoppable.
- CDP acquisition/restart checks generation after each asynchronous boundary.
  Stop is terminal for that generation. Pending frames carry page identity and
  cannot survive a switch. Transport readiness delivers the latest valid pending
  frame without depending on another compositor paint.
- Expose requested, effective, pending and unsupported stream settings separately.
  A successful update has an observable application boundary. Measured current
  FPS is independent of target FPS; performance framing is negotiated rather than
  changing only a metadata field. Qualify controls on both CDP and polling.
- Evidence projection preserves structured versioned outcomes. Do not stringify
  unrecognized structs into apparently valid artifact payloads. Use one owned
  conversion policy and typed round-trip checks at persistence and export seams.
- Validation gate options must reach the server through the public CLI/program
  boundary. Qualify boolean presence/absence and failed enforcement; never treat a
  client parsing error as validation evidence or remove a required assertion to
  make authoring succeed.

These are proposed invariants. The live counter control proves ordinary typed
execution on the deployed browser; it does not certify the streaming path or its
latency, and producer race probes do not attribute the API's observed Go heap.

### Rendering and release decisions still open

Keep the existing stream as the initial baseline. Compare a native desktop
WebContentsView and a video transport only with measured input latency, text
clarity, recording fidelity, evidence support, platform packaging, and resource
cost. No framework replacement or transport preference is approved here.

Derive accepted performance limits from the assessment's proposed cohort and
bands. Preserve required unknown rows until instruments and receipts exist.
Lighthouse, file sizes, and generic green suite results do not establish the
browser-first product outcome.

### Implemented profile boundary repairs — 2026-09-22

Manual persist and browser close now share `persistSessionProfile`, which
captures storage and tabs then calls the profile service's aggregate
`PersistSessionState`. Close preserves the browser and association until that
operation succeeds; UI callers retain the workspace and allow retry on failure.
Unreadable or missing encrypted profile state now fails default resolution
explicitly; listing cannot silently hide it and cause a replacement identity.
These changes are covered by maintained fault tests (BAS-WORK-001/002).

Read/modify/save concurrency is not serialized. RF-026 remains unresolved;
atomic complete snapshots do not establish the full field-update transaction target.

The selected RF-025 replacement is a single versioned JSON envelope containing
the encrypted full profile. Only that document is authoritative; api-core owns
temporary-file sync and atomic replacement. Public service types stay unchanged.
The old paired metadata/protected format will have no runtime reader or migrator.
Before deploying this format, preserve existing data and convert offline with
verified recovery bytes. Local inspection found one older plaintext profile and
no configured environment key; credential-authority provisioning/recovery is a
required adjacent RF-012 repair, not permission to discard or re-key that profile.
Credential resolution now uses the platform authority's generated versioned
keyring with a data-owned loss witness. Reads select the document's key version;
new writes use the active version. Missing credentials never trigger re-minting
when profile data or the witness survives. Synthetic rotation, loss, provider
failure and restore tests pass. The offline converter has verified synthetic
roundtrip, rollback and refusal of changed or unknown source data. One existing
profile was converted offline with its original bytes retained for rollback.
Lifecycle restart is healthy; fresh repository reads preserve all fields and the
live metadata API returns the original identity without protected storage.
Native-platform, off-host key recovery and physical-power-loss proof remain open.

### Implemented event lifetime ownership — 2026-09-22

Workflow orchestration defers the required `Sink.CloseExecution` operation on
every exit, including compile failure. Collector forwards that operation and
clears execution-specific cursor buffers. The queue worker owns delivery of
accepted events before hub close; closure rejects new publishes. Collector
records UX evidence only after delegate admission succeeds. RF-021's wrapper
leak and close-time drop are repaired in owner tests; the contextless hub
broadcast still lacks a bounded stopped-consumer shutdown guarantee.

### Profile mutation boundary — selected RF-026 repair

The profile repository will own an atomic Update callback, reading the current
profile and publishing its changed snapshot under one stable store lock. Create and Delete acquire that same lock; reads
continue to use the atomic published file. Reuse platform-go's native advisory
lock with a bounded five-second acquisition wait. The stable lock file is control
metadata, separate from each profile's sole commit document and key-loss witness.
Do not remove it on profile deletion, which could split ownership across inodes.

Service field mutators and handler cookie/tab mutators must move their read and
write into this boundary. Preserve validation, timestamp, not-found and failure
semantics; an acknowledged Delete cannot be undone by a stale field update.
The existing context-free profile service API does not propagate caller
cancellation; bounded lock admission is the initial guarantee. Native Windows
behavior still requires target evidence. The store transaction and converted service/handler callers are implemented
and locally deployed with real-file interleaving and failure tests. This does
not qualify exclusive browser-session write ownership or the native target matrix.

### Targeted storage edits — selected RF-051 repair

The recordings owner must edit only the requested cookie or localStorage subtree
inside the profile transaction. Treat all other JSON as opaque preserved state,
including unknown root/cookie/origin fields and origin IndexedDB payloads. Clearing
localStorage may remove an origin only when no other stored field remains there.
Do not rebuild a complete browser snapshot from the narrower metadata/UI schema.
Malformed relevant JSON must reject the edit before publication. Clear-all-storage
remains the explicit operation that discards the entire snapshot.

The profile repository exposes Create/Update/Delete for mutation. The exported
unconditional full-profile Save path has no remaining runtime caller and is
removed as part of finishing the transaction replacement. Full encoded snapshot
publication remains private to the repository.

### Browser authentication capture — selected RF-011 repair

Capture must request the installed provider's IndexedDB snapshot alongside its
cookies and localStorage. Session storage types should derive from the provider's
BrowserContext storageState return type so restoration does not carry a competing
partial schema. Prove close/reopen against a local independent identity fixture
while a second browser profile stays isolated. This target does not establish
service-worker cache, sessionStorage or crash/checkpoint durability guarantees.

### Profile identifier confinement

The file repository owns identifier-to-path validation. A profile identifier is
one opaque filename component, never a relative or absolute path. Both POSIX
and Windows separators, drive/stream syntax and NUL are rejected on every host.
Reject invalid identifiers before reads, writes, deletes or write-lock admission;
transport validation provides an additional boundary, not the storage invariant.
Tests must verify that unrelated sibling files survive rejected operations.

### Vision request resources

Build and validate a gateway request before allocating its abort timer. The
network attempt owns that timer from admission through its existing finally
cleanup; local validation rejection must leave no pending request resource.


### Maintainability measurement scope

The driver is production source and must participate in applicable maintainability
metrics. RF062 found both a shared directory-name exclusion and Tidiness Manager's
second api/ui/cli-only language walk. Remove those omissions at their owners;
derive all metric families from the same filtered source inventory. CodeFacts
remains the declared-surface authority. Maintainability also covers supporting
source beyond those surfaces, so this repair reuses Tidiness Manager's existing
local inventory instead of adding remote surface discovery to every file scan.
Retain original narrower receipts, remeasure comparable expanded before/current
boundaries, and keep ratchets unchanged. More findings after correcting coverage
are revealed debt, not evidence that the scan or product should hide those files.

Current qualification limit (BAS-RF-064): the JavaScript duplication adapter
still ignores its supplied inventory and malformed output can become an empty
result. TypeScript complexity is explicitly unsupported. Neither absence
establishes clean source; retain analyzer coverage limits beside native totals.

### Instruction failure diagnostics

The instruction executor owns collectors from startup through one final cleanup.
An unexpected handler exception becomes a nonretryable uncertain result before
telemetry collection, so the failure retains enabled console/network and visual
evidence without repeating an effect. Each capture channel must preserve the
other available channels when it fails. Unexpected capture errors are recorded
in outcome notes; a previously successful handler cannot silently claim complete
execution evidence after that error. Declared handler failures retain their
original cause. Metrics observers cannot interrupt evidence collection. Explicit
handler screenshots and NEVER/ON_FAILURE capture policy remain authoritative.
Native capture timeouts remain unchanged in this slice; transport-hang behavior
requires separate qualification.

Enabled Chromium console capture owns a dedicated CDP session and acknowledges
Runtime.enable before executing the instruction. It consumes native console
events directly, without requiring Rebrowser's suppressed Page console bridge,
without page-script wrappers, and without changing process-wide patch settings.
Initialization and detach are awaited. The per-instruction collector bounds its
retained events and releases its session on every exit. This explicitly enables
Runtime observation while requested console capture is active; anti-detection
impact and cross-target worker/OOPIF coverage require their own receipts.

### Go browser execution ownership

Workflow execution uses the lease-owning GoSession and driver.Client's single
compiled-instruction transport. Recording/navigation use their maintained client
interfaces. Remove the unimported parallel driver.Driver/Session model and its
Playwright adapter/ClaudeCode stub; these do not participate in current wiring.
Their untyped plural RunInstructions endpoint payload is obsolete and has no
remaining caller once the unused adapter is removed. Keep active client/session,
recording and navigator interfaces and their behavioral regressions.

### Execution-history query contract

The execution repository owns one query for workflow, project and status filters,
ordering and pagination. The public response reports the total matching rows,
including when an offset returns an empty page; has_more follows that total.
Count and page are read from the same request-routed database transaction.
Equal start times use execution ID as a deterministic tie-breaker. The public
limit defaults to50 and accepts1–100; internal maintenance can explicitly request
an unlimited query. Retention uses the same query with oldest-first selection;
separate status-list implementations and fallback interfaces are unnecessary.
Offset pagination across separate requests still has no snapshot guarantee while
history changes; deterministic ordering does not claim cursor isolation.

### Recording delivery owner — RF002 (033)

A browser observation receives a stable identity before its first transmission.
Retries retain the same observation, identity, and driver sequence. One bounded
pending-entry owner retains delivery state; overflow, failed callbacks, destructive
reads, reinjection and teardown must not silently discard unacknowledged work.
The browser keeps its pending observation until the driver acknowledges the
configured consumer. For BAS that consumer is the Go journal, whose200 response
follows the commit. Driver routes await the delivery outcome and propagate failed
or unknown delivery rather than replying with success.

Stopping first prevents new capture, then settles admitted work and flushes pending
input. Stop, reset, and close preserve recovery ownership on failure. A repeated
stop retries unfinished work and returns the committed result; it cannot infer
success solely because capture is no longer active. The no-callback/pull capability
retains explicit pending/acknowledgement semantics; reading bytes alone does not
prove the receiving process committed them. Browser and driver process-loss
guarantees require separate native fault evidence and must not be inferred from
in-memory retention.

### AI suggestion response owner — RF068 (034)

The suggestion generator owns the response contract. It requests structured JSON
through the existing governed Ollama gateway, including the required action,
confidence and supported category fields. It validates returned suggestions
before exposing them; malformed, incomplete or out-of-range provider output is
an error, never an empty successful result or an invented replacement category.
The existing analysis operation may continue with DOM-only results on provider
failure, and must not charge for absent AI suggestions. Valid empty suggestions
remain distinguishable from malformed output. Both established array and object
response shapes remain accepted at the parser boundary. Live integration checks
must not skip response-validation failures after a healthy provider preflight.

An empty extracted element set returns an empty suggestion list locally. It has
no element-grounded action to infer and must not trigger a provider request.

### Frame execution target — RF004/RF030 (035)

A successful frame-switch changes the session-owned document target for subsequent
DOM actions. The selected document is identified by the actual frame, not its URL;
distinct siblings may share a URL. Enter resolves a real iframe relative to the
current document, parent restores its ancestor and exit restores the main document, and repeated selection of the
same target preserves depth. Detached frames and ambiguous targets fail before an
action can affect a different document. Main-page navigation and tab replacement
invalidate the selected frame path. Page-level input devices and evidence remain
owned by the page; DOM actions resolve their target explicitly and never cast a
Frame into a Page or fall back silently to the main document.

Qualification uses public driver instruction routing and independent fixture
counters in main, sibling and nested documents. The declared iframe capability
requires effect-level evidence; success metadata and a mocked stack are insufficient.
Default Rebrowser frame-context quirks require native tests with the installed
mode, without disabling runtime protections or relying on private SDK identifiers.

The Rebrowser dependency owns JavaScript realm discovery. A version-pinned pnpm
patch, installed through Scenario Dependency Analyzer, must materialize scriptless
frame realms before binding discovery, require a positive matching context ID,
join concurrent discovery for one world, and select the actual ancestor renderer
session for nested out-of-process frames. Missing/detached contexts reject; they
cannot default to the parent. No BAS runtime monkeypatch or Runtime-fix mode
change is permitted. Native SDK tests cover default and isolated renderer modes.
The patch retires persistent discovery scripts and unbounded binding listeners;
existing cross-realm binding globals and wider stealth/security fidelity require
separate qualification, beyond the narrow console-getter probe.

Download retry identity is owned by the session route's lease/operation receipt,
like other browser effects. Each new download invocation must trigger and save a
new download from the selected document. Selector/URL/TTL result reuse is invalid
across frames, document changes and loops. Handler-local cache state and its
legacy cleanup surface are retired; in-flight trackers for unrelated owners do
not acquire a second result cache.

### Arbitrary JavaScript execution — RF073 (036)

An evaluate instruction dispatches its expression once. Context destruction,
timeout, page closure and script errors do not establish that its external effects
were absent. The handler preserves the original failure but marks it non-retryable
so the executor cannot automatically repeat an uncertain arbitrary script.
The navigation owner and explicit workflow waits establish document readiness;
no handler-local retry loop may replay arbitrary code after navigation. Ordinary
read-only extraction retains its existing error classification. Qualification
uses an independent server effect count during script-triggered navigation, plus
success and error controls. This is a bounded at-most-once dispatch guarantee,
not durable exactly-once execution across process failure or a caller's new action.

### Dependency and secret-scan evidence — RF071 (037)

Scenario Dependency Analyzer owns dependency declarations, resolver overrides and
lock regeneration. A new package-wide override replaces superseded range-qualified
overrides for that same package; one declaration governs its selected version.
Other packages and parent-qualified overrides retain their explicit constraints.
Security Health owns the resulting dependency/security assessment.

Historical source digests remain immutable evidence. A reviewed secret-scanner
exception may match only the exact non-secret digest line or the exact credential
lookup-label declaration, with the reason retained beside its configuration.
Keep default detection rules and scan every file. Synthetic counterexamples must
still detect changed digest values and credential keys in those same files.

Direct URL downloads may convert a Chromium navigation into an attachment,
which reports `net::ERR_ABORTED` to the navigation call. The download handler
accepts that one transport signal only while still requiring the page's download
event and successful file persistence. An abort without a download remains a
failure; other navigation errors propagate. This rule does not apply to clicks
and does not infer a saved file from the navigation result alone.

The Rebrowser session owns one context-discovery binding registration for its
lifetime. Immediate `Runtime.removeBinding` after discovery destabilizes context
identity on scriptless documents after navigation in the supported Chromium build.
A stable random session binding name and a unique per-request receipt token
prevent registration growth and crossed acknowledgements. Transient listeners are released per request; one global binding function per
renderer realm remains available for repeated discovery. Session disposal owns
the registration. A fresh realm may require one additional installation after
the internal probe reports an absent binding; only this side-effect-free discovery
handshake is repeated, with a bounded attempt count. Arbitrary user expressions are never
retried to mask discovery failure. This retains default runtime protection mode.

### Reset admission repair — 2026-09-22

Every external reset carries its immutable execution and lease IDs. The driver
parses the full request before checking the active, unreleased lease, then joins
any in-flight reset or starts the destructive operation. Rejected ownership must
leave browser state, phase and last-used time unchanged. The Go Session supplies
its admitted identity and requires an explicit success acknowledgement. Concurrent
same-owner reset callers may share an in-flight result; elapsed time and a guessed
ready phase are not reset receipts.

RF076 separately proves the current in-place storage reset is incomplete and
fails on opaqueabout:blank. The admission repair does not qualify successful clean
reset, cross-origin storage clearance or context replacement. Its next owner repair
must preserve recording acknowledgement, capture resources and explicit failure
recovery while using a coherent browser-context reset boundary.

### Intended clean reset boundary — 2026-09-22

RF076 repair keeps the managed browser context and its original primary page so
context configuration, recording injection, tracing/HAR and primary-page capture
remain owned. Reset makes that page active and closes other live pages. Context
construction records imported and subsequently visited storage origins (including Chromium file://); reset
uses that inventory instead of serializing user IndexedDB to discover origins.
After recording delivery is acknowledged, reset closes other pages, navigates
the retained page away from application scripts, clears each origin's browser
storage through public CDP, and clears per-tab storage on intercepted empty
origin documents. These visits must make no external application requests.
Cookies and permissions clear at the context boundary; another context on the
same origins must remain unchanged. Successful reset clears page/frame selection
and pending result receipts, retaining the lease's instruction high-water mark.

The SessionManager must reserve/reset phase before awaited work, own concurrent
reset joining, retain failed reset ownership for an explicit retry, and join a
pending reset before close disposes resources. A closing session cannot return
to ready because reset completed late. Unsupported external targets still refuse
reset before effects. Neither clean profile import/reuse compatibility (RF042)
nor secondary-tab capture preservation (RF043) is qualified by this repair.


### Readiness lifetime — 2026-09-22

A readiness caller owns its deadline timer until the readiness result, rejection,
or deadline settles; every path releases that timer. Recording verification may
poll while its page exists, but page closure must end the wait and stop further
protocol calls and retries. Closed-page cancellation is a failed readiness result,
never permission to mark the pipeline ready. Keep this policy in the existing
SessionManager wait and recording verification owners, without a second lifecycle
manager. Successful live-page verification and its bounded timeout remain supported.

### Frame lifecycle ownership (rehabilitation target, 2026-09-22)

The existing frame coordinator owns one slot per session. A slot serializes
resource acquisition/disposal and invalidates its generation at replacement or
stop admission. Pending starts are joined by stop; obsolete page providers and
transport adapters cannot publish into a new generation. Failed disposal retains
its resource for explicit retry. Strategies clean up resources acquired before
startup failure. Session-start previews retain their immutable execution/lease
identity across readiness waits and subsequent page access. RF045 qualification
records the coordinator boundary; RF046 still owns internal CDP resize/tab races.

The CDP strategy's target design binds each listener, buffered frame and ACK to
the exact capture that produced it. Initial start, tab change and resize use one
serialized acquisition path. Stop invalidates the generation before joining late
acquisition; resize checks its original generation after viewport mutation. The
existing page probe delivers the current buffered frame after reconnect without
requiring another paint. Transport delivery failure still acknowledges Chrome's
frame. These are RF046 and RF047 buffer-delivery targets; effective FPS/quality
controls require their separate qualification.


### Polling capture lifetime (rehabilitation target, 2026-09-22)

The fallback strategy uses the browser SDK screenshot owner with one bounded
capture in flight. It must not keep a private global CDP session cache or a second
capture deadline. Preserve configured quality, CSS/device scale and visible caret.
Every interval wait owns exactly one timer and abort listener; either settlement
releases both. Stop invalidates publication immediately and all concurrent stop
callers join the same loop. A completed capture belongs to its original page and
viewer; only a successful send advances deduplication state. Changing page or
viewer requires a current frame even when its pixels match the previous frame.


### Recording tab identity and callback lifetime (target, 2026-09-22)

Recording page registration in recording-pages.ts returns the existing identity
for an already tracked Page and otherwise updates the page list and both maps
once. Explicit tab creation and context page discovery use this same operation.
Initial SessionManager page identities remain authoritative.

The page-events owner attaches the same navigation and close handlers to initial,
already-open and new pages. Handlers bind the exact Page/ID, never mutable active
page selection. Its cleanup removes context and page listeners and invalidates
pending callbacks before they can send or attach more listeners. Every asynchronous
listener rejection is observed. The owner returns immediate cleanup plus initial-page delivery readiness. The start
route stores cleanup before awaiting readiness; its duplicate initial-page event
and navigation implementations are removed. Previously admitted
HTTP callbacks retain their existing deadline; local cleanup cannot undo remote
commit. Recording request/lease fencing remains a separate RF038 boundary.

Session reset and teardown invoke this callback cleanup after recording acknowledgement
and before navigation or browser disposal. Failed cleanup keeps its handle for
retry; successful teardown stages use the existing completed-stage ledger.


### Effective stream controls (target, 2026-09-22)

The existing frame manager owns admission and reported settings. An update receipt
must follow application by the active capture owner; overlapping stop/replacement
must invalidate obsolete updates. Quality changes require an acknowledged capture
change when the protocol cannot update encoding in place. Both strategies must
honor the advertised delivery FPS limit and timing-header mode. Measured FPS comes
from the existing performance collector and must remain distinct from target FPS.
Preserve the current scale restart policy and supported frame decoding formats.
Prefer one settings/encoding/delivery policy over parallel metadata or schedulers;
validate numeric controls before they can corrupt capture or scheduling state.


### Frame scale capability (target, 2026-09-22)

The frame coordinator selects the existing SDK polling strategy for device-scale
requests and CDP for supported CSS-scale requests. Native Chromium in the current
cohort emits CSS-sized screencast images at DPR2 even when protocol size caps are
doubled. CDP therefore must explicitly reject device-scale startup rather than
acknowledge fidelity it cannot guarantee. Device capture uses the SDK's existing
scale contract, including at DPR1; do not add a guessed DPR probe or a second
capture implementation. This choice can trade compositor frame rate for requested
pixel fidelity. Measure that tradeoff without reducing quality or calling a target
FPS an observed rate. Native OS and release performance qualification remain
separate. Existing startup-failure fallback, settings, stop and tab ownership stay
with their present owners.


### Recording-start continuation ownership (target, 2026-09-22)

RecordingPipelineManager owns readiness, document activation and the monotonic
recording generation. The HTTP start route must not add a second DOM-load wait
after that owner has acknowledged capture startup. Admit preview immediately
through the existing frame coordinator so Stop can dispose the same owned stream.
Validate the caller execution/lease after body parsing and retain it across
every subsequent asynchronous boundary. Bind the start operation
and its frame page provider to the pipeline's next recording generation; a stop,
restart with the same public recording ID, pipeline replacement, reset or lease
handoff must invalidate the older operation. A superseded start cannot return a
successful current-recording receipt or attach new preview/page callbacks. Use
the existing generation and cleanup owners; add no parallel lifecycle registry.
Start/stop now authenticate explicit caller leases as specified below. RF038
still requires explicit envelopes on the other mutating commands and their callers.


### Recording caller ownership and receipt fidelity (051 target)

Pull acknowledgement uses the same immutable Session authority (056 target).
The API resolves that Session before reading or committing a destructive pull,
and retains it through journal commit and exact-ID acknowledgement. The raw
driver interface cannot acknowledge entries without an execution/lease pair.
The driver parses the complete request before admitting that pair; missing,
stale, released or non-operational owners leave the buffer untouched. Accepted
acknowledgement is synchronous after admission, and hides only the supplied
committed identities. Retried acknowledgements retain their exact receipt.
Navigation, preview and other interactive mutations remain separate RF038 work.

Recording start and stop carry the immutable execution ID and lease held by the
Go Session. Live capture resolves its owned Session; it must not guess a driver
lease or bypass ownership through the raw client. The driver validates the wire
lease after parsing and before activity, cached success or recording effects.
Start and stop continuations also validate the admitted session and existing
pipeline generation before mutating preview, callbacks or session phase. This
replaces 050's server-snapshot admission; its continuation fences remain.

Driver-issued recording IDs and start/stop timestamps survive Go decoding and
typed proto conversion unchanged. The driver stopped_at maps explicitly to the
canonical completed_at field. Remove handler-generated IDs, duplicate handler
response types and silent untyped success fallbacks. The canonical proto is
unchanged. Other mutating routes and operation-specific retry identity remain
RF038 boundaries; this repair does not claim to reverse already admitted effects.

UI recording responses consume canonical completed_at. A known recording-in-progress
conflict may recover only through an authoritative active status receipt with the
same session and actual recording ID/start time. Unknown conflicts or missing
status evidence are errors; never fabricate a recording identity or current time.


### Live-input caller ownership and transport order (052 target)

HTTP and WebSocket live input must use the same live-capture service and immutable
Go Session lease. Remove the WebSocket-specific HTTP transport and the driver's
obsolete raw-post helper after caller conversion. Preserve the existing WebSocket
2-second context deadline through the shared driver client, which already pools
connections and drains responses. Unowned/malformed input must fail before HTTP.

Reuse the recording route's lease admission/continuation helper for driver input;
parse before ownership lookup, refresh activity only for admitted calls, and
revalidate before any subsequent browser sub-operation and success receipt.
The WebSocket reader must await forwarding in received order and use its existing
read/backpressure boundary rather than spawn an unbounded goroutine per event.
This fixes the demonstrated single-connection reorder. Cross-client/HTTP ordering,
applied-sequence receipts, coalescing and reconnect/key-state guarantees remain
RF020 obligations; this bounded repair must not claim to satisfy that entire row.


The existing WebSocket hub mutex owns client membership, subscription fields and
Send-channel lifetime (052 RF085 extension). All mutations and channel closure
need its exclusive lock; broadcasts that cannot remove clients may share its
read lock. Subscription handling must confirm membership before sending replies.
Browser input forwarding stays outside that critical section, with the existing
per-connection read loop owning input order and the forwarder owning its deadline.


### Recording harness qualification (053 target)

The registered recording E2E harness must use the current owned session and typed
action contracts. Its own temporary HTTP fixture supplies the independent effect
counter; recorder entries cannot be the oracle for their own correctness. Stop
flushes capture before entries are inspected and explicitly acknowledged; replay
uses a fresh session and must reproduce the fixture effect. HTTP/JSON/action
errors and failed cleanup produce a nonzero result. Requests have finite deadlines.
Optional API coverage is selected at prerequisite inspection and remains visibly
unqualified if absent. Once selected, generation/persistence/read/cleanup failures
fail the run. Temporary projects/workflows belong to the harness and must be
removed. Do not loosen production lease contracts to accommodate old test calls.


### Saved recording execution evidence (054 investigation)

The next qualification boundary is the existing WorkflowsService.ExecuteWorkflow
owner, with the exact persisted revision and wait_for_completion=true. The local
recording fixture must observe one additional effect and the execution owner must
return completed state and timeline evidence. Keep native receipts before any
cleanup. Retention preview/run must be restricted to the unique temporary project
and workflow and must select only the fixture execution; do not sweep shared
history. No new production execution or wait policy is authorized by this test.


### Recording diagnostics and application console evidence (055 target)

Routine recorder initialization, activation and capture must not write to the
page console. The injected script already exposes readiness markers, handler
counts and event/delivery/error telemetry for driver diagnostics. Remove routine
console statements at their source; do not filter application console evidence.
Preserve genuine initialization failure reporting and real application errors.
Native console observations must cover both passive and active recording, with
an independent application-error sentinel and unchanged click capture/telemetry.

### Recording navigation ownership and browser history (057 deployed boundary)

URL navigation, reload, back and forward carry the Go Session's immutable lease.
HTTP recording controls, history Connect navigation and first-tab restoration
resolve that owner before the command. Remove unowned raw-client navigation
methods and their mocks. Reload/back/forward share their request, response and
commit policy; preserve the existing public endpoints and JSON shapes without
retaining duplicate types or compatibility aliases.

The driver admits after full body parsing and binds its continuation to both the
Session lease and the originally selected page. Check that authority after every
await before navigation-history, frame-cache, callback or successful response
publication. Shared completion owns that policy across the four operations.
Preserve timeouts/wait conditions, optional screenshot and successful navigation
recording/page broadcasts. The active browser page owns history: read its CDP
entries/index with a short-lived attachment, detached on every completion/error.
Remove the session-scoped history map and its cleanup/export paths. Browser
history determines bounds and movement, including same-document transitions
whose Playwright response is null and entries created by page scripts or links.
Navigation-state and popup reads bind the original Session/page throughout
awaited reads. Popup entries carry URL/title; visit timestamps are unavailable
from the browser and must not be fabricated or required by the UI. Empty titles
remain valid. The Go JSON timestamp is optional for compatibility with consumers
that can supply one; this driver does not synthesize it.
A browser navigation already admitted while owned may have external effects;
post-admission interruption/reconciliation remains an explicit RF038 boundary.

### Navigation result page attribution (058 deployed boundary)

Successful driver navigation receipts identify the exact originally selected
page through its registered driver page ID. The Go transport rejects missing
page identity. API completion retains the admitted Session and resolves this
receipt through that Session's PageTracker; it never guesses the later active
page or rebinds a completed effect to a replacement Session. A later tab switch
does not erase a valid effect on the original page. Missing/unknown page receipts
and replaced owners fail explicitly before metadata/journal publication.

Reload/back/forward commit their exact original page action before WebSocket
publication and recheck Session identity after awaited journal work. If ownership
changes during commit, retain the successful commit but suppress publication to
a replacement subscriber stream and report incomplete completion. This is not
a claim of atomic journal-generation isolation or reversal of external effects.
URL navigation retains its existing event/capture policy without adding a
second journal observation. Public recording HTTP response shapes stay stable.

058 necessary page-registration extension: session admission returns its active
registered driver page ID; the Go Session retains it to initialize its initial
PageTracker mapping before any navigation or recording callback. New-tab201 is
a valid endpoint receipt. PageTracker atomically registers each driver page once;
creation/restoration receipts and later callbacks converge on that existing
identity. A recording-start initial event describes the active page, not authority
to relabel the Session's original page. No speculative active-page fallback.

### Saved-tab restoration agreement (059 deployed boundary)

RestoreTabs applies successful browser receipts to the existing PageTracker and
returns actual restored URLs/titles. Retain the saved selected page across the
entire creation loop, including the first page. Once pages exist, use the
existing ActivatePage owner for both browser selection and API active-page state;
skip a redundant switch when already selected. Do not add another tab map,
infer selection from the last created page, or use the raw driver switch while
leaving API state stale. Failed selection must be reported instead of silently
claiming successful restoration. Additional empty-tab omission remains the existing policy; failed admission
follows the060transaction below. Full profile durability and page mutation lease
qualification remain separate requirements.

### Failed tab restoration transaction (060 deployed boundary)

New-page success means the requested navigation completed. A rejected navigation
closes that command's new page, removes its registry entry through the existing
page lifecycle owner, preserves the prior active page and returns failure. Do not
turn network failure into an acknowledged Chromium error URL. Cleanup failure
must remain visible with the original failure. No alternate registry or retry
framework. Page callbacks and explicit cleanup converge idempotently.

Restoring saved tabs is an admission transaction: a failed navigation, creation
or final selection must reach CreateRecordingSession. The handler closes the
uncommitted session through the existing service, reports failure, and leaves
the saved profile snapshot/active association unchanged. Touch/associate only
a successfully admitted session. Users can explicitly choose restore_tabs=false;
a failed automatic restore must not silently choose that behavior for them.
Existing profile storage transaction remains authoritative; do not introduce a
shadow copy or runtime migration. Raw page mutation leases and profile concurrency
are still separate unqualified boundaries.

### Tab command authority (061 deployed boundary)

Create-page and activate-page are mutations under the same immutable Session
execution/lease ownership as navigation and input. Go callers must route through
the retained Session; driver transport cannot infer authority from a session ID.
Driver admission parses the request then uses the existing recordingOwner lease
validator. Revalidate the admitted owner after each await and before selection,
frame-cache mutation or success publication. Newly created pages whose command
loses ownership must be disposed by that command; never close or mutate a
replacement Session's page. Commit selected-page state only after awaited page
metadata and ownership/page-registration checks succeed. Existing recording and
execution page switching must both carry their own lease.

Live-capture API completion retains the admitted Go Session and rejects replaced
owners before updating its PageTracker. RestoreTabs retains one owner for the
whole restoration transaction; repeated creations must not silently re-admit a
replacement with the same public ID. Share existing page registration policy and
lease validator without adding another registry, command framework or guessed
lease. Missing/wrong/expired leases must cause zero newly admitted browser effects.
Already admitted network effects during a later handoff cannot be undone; report
incomplete completion and preserve that explicit reconciliation limitation.

061 cleanup extension: internal session admission returns its bound close operation.
Failure cleanup uses that exact operation, never resolves a potentially replaced
Session by public ID. Journal-registration failure likewise closes its captured
Session. This is a narrow internal receipt capability, not another lifecycle
manager or public protocol field.

### Initial navigation receipt agreement (062 deployed)

Fresh requested initial navigation and saved first-tab restoration share one
operation in live-capture: retain the admitted Session and its initial-page
identity, apply the browser's actual URL/title receipt, and reject a replaced
owner or mismatched page. Session admission with an explicit initial URL succeeds
only after that operation succeeds. Failed admission closes its captured Session
under a bounded cleanup context that survives request cancellation; profile usage
and association are left to the handler after successful admission. A blank
initial URL remains supported. Delete the separate warn-and-continue policy and
duplicate receipt validation rather than patching the two callers independently.
The public InitialURL response field remains scoped to restoration; the page
registry is the canonical location/title read for every admitted session.

062 admission cleanup uses one deferred failure owner after Session creation.
Journal-registration and initial-navigation errors both preserve their cause and
join any bounded cleanup error. A successful admission transfers the bound close
operation to its result. No second cleanup implementation or fallback lookup.

### Detached page-state reads (063 deployed)

PageTracker owns mutable page records. Registration inputs and every returned
page must be detached, including optional opener and close-time pointers. API
page lists and profile tab snapshots obtain their pages and selected ID under
one read lock; sorting uses private copies after unlocking. Replace the two
list traversals with one snapshot operation and convert their callers. A session
with no open pages has no active page. Preserve chronological list ordering,
driver identity deduplication and the existing public JSON fields. This boundary
protects serialization from callbacks; it does not establish driver/API event
reconciliation or a transaction across profile storage and browser state.

### Browser-owned tab closure (064 deployed)

A successful close-tab command must close the actual admitted browser page and
return the driver's resulting selection. The retained Go Session supplies its
immutable execution/lease identity; validate completion before updating API state.
The page registry records browser results rather than pretending that a local
status change closed a tab. Recording callbacks and explicit commands share
closure registration and journal identity, preserving retries without duplicate
close observations. Final-page closure leaves no selected open page and permits
subsequent new-tab creation in the same browser context. Keep page cleanup and
selection policy in the existing driver page owner, with focused failure/handoff
tests and native close-before/while-recording cases.

064 qualification: explicit close commands now satisfy this boundary before and
during recording, including last-tab close and create-after-close. PageTracker's
terminal event ID/timestamp is stable across command/callback/retry; existing
journal idempotency owns commit deduplication. UI close and selection receipts
synchronize view and store, including no selected page. External callback
generation authority and arbitrary simultaneous-close reconciliation remain
unqualified; this receipt does not close those broader RF038 obligations.

### UI page admission and session ownership (065 deployed)

The page registry's detached creation receipt supplies the UI with its canonical
page and selected ID immediately, including before recording listeners exist.
Preserve existing driver identity/location fields for current API consumers.
Live-capture returns the registered Page rather than discarding it and requiring
a second lookup. The session store owns UI page/selection state; usePages observes
it and applies authoritative receipts/events without a parallel local page map.
Consumer callbacks run outside React state updaters, once for a newly observed
page. A request completing after its session is replaced cannot publish pages,
selection, errors or callbacks into that replacement. Preserve chronological
ordering, close retries, popup notifications and saved-tab restoration.

### Live preview frame wire — BAS-WORK-066 (deployed)

The Go driver client shall decode the driver's actual `image` and `mime` fields.
The HTTP handler shall forward that canonical frame payload and metadata instead
of translating a second frame DTO. Missing image data, dimensions, timestamp, content hash or
JPEG media type shall fail before success/ETag admission. Session identity shall
match the requested browser. The content hash owns conditional caching; remove
the obsolete timestamp fallback. Preserve the existing
public frame shape and conditional caching. Native colored-page pixels in the
real UI qualify the bridge; mocked screenshots alone do not. Transport ownership
and performance remain separately qualified concerns.

### Live viewer transport and lifetime — BAS-WORK-067 (deployed)

The live viewer shall use the configured, proxy-aware API WebSocket route with an
explicit recording-session subscription. A viewer-owned socket retains that
session admission; cleanup closes the socket and rejects every late callback,
config read, HTTP response and decode. The existing global WebSocket context
continues to own ordinary workspace events/input. Do not consume anonymous
binary frames from a shared subscription after changing its session.

One decode/render path serves binary frames and HTTP fallback, with one active
decode, one newest pending input and one pending bitmap. Monotonic admission
sequence replaces wall-clock identity. Polling continues until a usable socket
frame paints and resumes after a stalled/disconnected stream; a socket-open event
is not proof of rendered content. Cache a polling ETag only after successful paint.
Tab/session replacement and final-tab closure clear displayed state and dispose
owned asynchronous work. Remove unused latency-spike metrics, duplicate decode/
paint policy and the now-unused direct driver frame listener/configuration after
callers are verified. The API remains the supported web viewer route.

This local owner fencing does not prove source page identity for frames already
queued by the browser/driver. The current binary wire lacks per-frame canonical
page/lease identity; retain that RF038/J22 qualification gap explicitly and repair
it at its producer rather than stamping current page state onto stale bytes.

### Explicit navigation intent — BAS-WORK-068 (deployed)

The browser navigation hook shall distinguish displayed/observed location from
explicit navigation intent. Page selection, browser navigation notifications,
restored tab metadata and history responses update display state without sending
new navigate commands. URL-bar submission, empty-preview submission and an
explicit launch URL admit navigation. A launch request may wait for session
creation; each explicit submission retains its own identity even for the same URL.

The request lifetime owns its abort and completion admission across session
replacement/unmount. Redirect receipts update location without another command.
Initial recorder/AI readiness follows a successful launch navigation (a failed
request must remain retryable). RecordingSession wires observations and commands
to their distinct hook methods and no longer owns a second navigation effect,
URL-equality suppression or navigation-response parser. Existing browser history
commands remain authoritative; no URL observation is replayed into that history.

New-page lifecycle admission (068 deployed): register page listeners synchronously
when the context reports a page. Creation delivery gates publication of its later
navigation/close callbacks, so the API has canonical identity before consuming
them. Cleanup invalidates pending admission and disposes listeners. Remove the
post-callback attachment gap and separate pending-close repair branch; one page
listener lifetime shall cover creation, navigation and close. This does not
qualify general callback retry/durability or concurrent navigation ordering.

### Page-bound navigation intent — BAS-WORK-069 (deployed)

URL navigation, reload, back and forward from recording browser chrome shall bind
to the selected canonical page at user intent. Commands include a page precondition;
the API resolves it only within the owning session and translates it to the driver
page identity. The driver checks the precondition against its active registered
page before effects, preserving the existing post-await lease/page checks.
This closes both UI-to-API delay and API-to-driver selection races. Programmatic
callers that intentionally address the active tab may omit the precondition;
that is an explicit command semantic, not a compatibility migration.

The existing UI navigation hook shall share one request lifetime across URL and
history operations, aborting pending work and rejecting stale config, response,
error and history-popup completions on page/session replacement or unmount.
Initial launch waits for session and page admission. A queued launch binds once;
a later tab selection is not permission to replay it. Consolidate repeated history
HTTP handlers when introducing this common request owner; retain explicit repeat
URL behavior, successful launch readiness and browser-authoritative history.


### Passive tab metadata — deployed070, 2026-09-23

The browser runtime reads favicon metadata from the loaded document. Existing
page lifecycle receipts transport it through the canonical API page registry to
the workspace tab bar. Tab display must not invoke OpenGraph/document previews.
A missing metadata value preserves a same-URL icon; a changed URL clears it; an
explicit empty value removes it. Read metadata after document readiness and reject
results from a retired listener or replaced URL. Tab image failures apply to that
image URL so a later page icon can recover. Start-page preview cards retain their
separate preview capability; no private tab cache or document-fetch policy remains.

The same icon reader supplies initial/navigation/history and new-tab command
receipts, including restoration and pre-recording admissions. A document change
while creation metadata waits cannot cancel page registration or its queued
navigation observations; only the stale metadata is discarded.


### WebSocket domain event delivery — deployed071, 2026-09-23

The workspace connection validates the common JSON envelope without stripping
fields owned by domain consumers, and synchronously delivers every accepted
message to subscribed handlers. React rendering may batch visual updates but
must not coalesce lifecycle, action, terminal or selection events. Domain hooks
subscribe through the shared context and read their latest committed callback;
unmount removes delivery ownership. Replace all single-lastMessage consumers,
including recording, execution, AI, schedule, driver and export notifications.
Remove the unused binary-subscriber facade: the recording viewer owns its separate
bounded frame socket. Connection callbacks are fenced to the current socket;
disconnect/unmount must not reconnect or publish from a retired connection.

071 empty-page viewer: canonical `null` selection means there is no page;
retain it through RecordPreviewPanel and PlaywrightView. The existing frame and
input owners must stop work for explicit null. An omitted page ID still follows
the active browser page for FloatingMiniPreview. Retire the old page lifetime,
abort pending polling, close its socket and reject late decode/paint work before
a fresh explicit page starts. No parallel empty-workspace polling policy.

### Dependency availability classification — deployed072, RF103

The existing resilience breaker measures dependency availability. Answered HTTP
400–499 request rejections, except408timeout, remain returned errors but do not
open the availability breaker. Context cancellation remains caller-owned; actual
transport errors, timeout and server failures retain failure counting and
half-open recovery. The driver and storage reuse this one policy; no caller-local
exemption or bypass. Classify wrapped typed errors without importing the driver
into the resilience owner.

### Browser tab keyboard interaction — deployed073, RF101

TabBar owns semantic selection and close controls as sibling native buttons,
with one roving tab entry and a visible focus indicator. Left/Right wrap focus;
Home/End focus boundary tabs; Enter/Space activate through native button behavior.
Use manual activation because browser selection awaits a remote operation. Delete
closes the focused tab without activating another as a side effect of focus.
After the focused tab disappears, focus its following neighbor, or the preceding
one at the end, or the empty New Tab control. Failed closure retains focus;
external focus is never stolen. Existing favicon, selected state, pointer close,
new-tab and horizontal overflow behavior remain. Close controls are siblings,
not nested interactive descendants of a tab button. The compact tab bar is a
scrollable list surface; this repair changes keyboard interaction, not the
recording route composition or design tokens. Full mobile/visual qualification
remains open. Reference: https://www.w3.org/WAI/ARIA/apg/patterns/tabs/.

### Recording frame source authority — target074, RF038

Both streamed frames and HTTP previews retain the immutable execution/lease and
driver page that produced the image. Capture that source before asynchronous
metadata/screenshot work; never attach the current page to older bytes. A cached
preview belongs to that source and capture policy, and late work cannot populate
a replacement page/session cache. Optional performance data does not gate or
replace required frame identity.

The driver-to-API binary protocol uses one bounded length-prefixed JSON header
plus JPEG payload, with mandatory version/source/capture identity and optional
timing fields. Remove anonymous raw/timestamp-only acceptance once all producers
convert. API validates source against its immutable Session and active PageTracker
receipt, maps the actual driver page to the canonical page ID, and publishes that
identity to viewers. Do not expose the driver lease credential to the browser.
The viewer rejects wrong-session/page frames before decoding; implicit-active
mini preview remains supported through API validation. HTTP fallback validates
requested page and returned capture source under the same ownership rule.

The repository has no producer or consumer for the old JSON recording_frame
HTTP-push route: only its handler/hub facade, mocks, unused UI schema and docs
remain. Retire that path while preserving the actual binary stream and HTTP GET
fallback. Update its listed docs. This replaces existing wire variants rather
than adding a parallel stream. Same-page navigation/capture epoch qualification
and full source-ordering/soak evidence remain explicit RF038 follow-ups.


### Recording subscriber intent — target075, RF007

The recording subscription owns event delivery and explicitly chooses whether
it also consumes binary frames. `subscribe_recording` retains frame delivery by
default; `frames: false` selects events only. The timeline requests events only;
the canvas keeps its frame subscription and independent reconnect/fallback
lifetime. Hub keeps the choice under the existing subscription lock and gates
both binary fanout and its frame-subscriber availability query with that choice.
Page events, timeline entries and performance messages remain available to event
subscribers. Resubscription replaces the choice; unsubscription clears it.
Remove the ambiguous subscriber-query name and convert every facade/mock caller.

Proof requires a current frame to reach exactly one actual canvas socket, no
binary bytes or dropped-frame accounting for the timeline, preserved timeline
and page events, reconnect/default-frame behavior, and the normal recording
workflow. The change does not qualify slow-reader queues or a performance band.

Driver frame senders bound each WebSocket's queued outbound data to one maximum
recording frame (12 MiB plus 4 KiB for the transport envelope). CDP and polling
drop a new frame when sending it would exceed that bound; CDP continues to keep
only its newest pending frame. This caps driver-side transport backlog but does
not qualify end-to-end frame age, browser decode/render work, API relay fanout
under sustained load, or a 30 FPS session. See BAS-RF-007 for the remaining
owner matrix.

The managed viewer relay also bounds each client's queued plus actively written
binary frames to 12 MiB plus 4 KiB. It drops a frame before enqueue when the
client byte budget would be exceeded, and releases the accounted bytes only
after the socket write completes. The viewer decoder keeps one active decode
and replaces its pending frame with the newest one. The managed motion owner
measures the full driver/API/UI canvas path and deliberately slows a viewer to
verify these bounds; it does not claim remote-network qualification.

### Managed motion qualification owner

Run from the repository root against a healthy managed BAS build:

```bash
node scenarios/browser-automation-studio/api/cmd/motion-cohort/qualification.mjs
```

The owner runs the focused UI one-active/newest-pending decoder regression, a
focused Go relay-byte-budget regression, and a real managed API-to-viewer cohort.
The live fixture changes a 16-bit visual marker at 30 FPS. Its five-minute
baseline requires at least 9,000 rendered and unique fixture frames, p95 frame
age at most 100 ms, p95 decode at most 100 ms, maximum decode at most 250 ms,
and frame payloads no larger than 12 MiB plus 4 KiB. A separate 18-second slow
viewer cohort adds 250 ms main-thread stalls every five seconds and an 80 ms
decode delay; it requires one active decode, fewer decoded/rendered frames than
frames received, frame age at most one second, and both viewer and Go relay byte
queues within the same frame-size ceiling. The receipt binds all relevant
sources, the live build, and hashes of the frame samples and combined test logs.

Run only the exact `rehabilitation-evidence` phase after the owner and all other
current-build receipts pass. The setpoint reader consumes its `motion` standing.
The artificial reader delay measures BAS browser decode/backlog behavior; it
does not qualify a remote network or native device.


### Session preview scale authority — target076, RF047

The admitted driver SessionSpec retains the requested CSS/device preview scale
for its lease. Session startup, recording startup/restart and HTTP preview use
that same value; a recording transition must not reset it to an implicit CSS
choice. Session reuse adopts the new admitted spec. Frame streaming remains the
owner of active transport/quality/FPS operations; do not add a second preferences
registry or retain disposed transport slots just to remember scale.

HTTP preview keeps `width`/`height` as CSS viewport geometry, matching its existing
cache contract and input coordinate space. The encoded bitmap follows the chosen
scale; device-scale image dimensions may differ from viewport dimensions. Make
this distinction explicit in the frame type, and verify both scales independently.
Preserve full-page capture, quality, source/lease fencing and conditional caching.
The prior probe's assumption that every HTTP bitmap must match CSS dimensions
is not evidence that HiDPI itself is broken. Its CSS-policy mismatch and missing
scale admission are actionable; initial CDP height/resize timing and repeated
heading fidelity remain separate investigations.


### Renderer geometry and capture ordering — target077, RF047

The SDK Page owns viewport mutation and Screenshotter owns screenshot admission.
Viewport mutation shall join the same per-page screenshot queue so that an
in-flight page or element capture cannot restore stale Chromium geometry after
resize. Keep the existing queue; do not add BAS locks at every capture caller,
a second screenshot implementation, polling delays, or a runtime SDK monkeypatch.
Repair the existing canonical Rebrowser dependency patch through SDA and retain
its current version and context-identity protections.

Chromium viewport application shall finish window sizing before setting the
compositor visible size. Respect preserveWindowBoundaries and null/external
viewport paths; preserve mobile, DPR, orientation, browser launch and audio/SW
configuration. Remove the BAS video-only unawaited metrics override once the SDK
owner covers its geometry invariant. Keep unrelated video layout behavior until
it has independent evidence. Initial capture, viewport resize, CSS/device output,
page/element screenshot ordering, video pixels and failure/close cleanup need
qualification. Existing frame lease/source fences and profile/workflow behavior
remain mandatory. This repair does not by itself qualify viewport command leases,
callback generation, old same-page packets or sustained performance.


### Address-bar observation — implemented078, RF099

The validated session store's selected active Page is the browser-location
observation owner. Feed its URL into navigation display on snapshot admission,
selection and page metadata changes; an observed URL must not submit navigation.
Keep explicit navigation intents and their page-lifetime fences. An unchanged
observation must not reset a pending local value. Validated empty selection clears
the location; an unvalidated session must not observe a foreign cached page.

Retire the RecordingSession callback-only URL assignments, last-action fallback,
and separate restored-URL state in hook/store. Their sole consumer is replaced
by canonical page observation. Remove unused closed/selected/navigated callback
facades in usePages while retaining creation activity and authoritative store
updates. BrowserUrlBar keeps its existing local typing draft. Preserve explicit
launch/navigation, restoration, tab selection/closure and stale-response refusal.
This target repairs location display; initial Back/Forward history state and
navigation-stack read attribution still require separate behavioral evidence.

### Navigation-history observations — implemented079, RF099/RF038

The browser remains the history owner. Initial attachment and every selected
canonical Page observation refresh Back/Forward availability through an owned
read. Reset capability state on page/session handoff; stale reads cannot publish
into a new lifetime or overwrite the URL draft. Refresh reads observe capability
booleans only. Explicit navigation and history commands retain their existing
intent and response handling.

Both API history reads bind the current immutable Session and selected canonical
page. Optional public page_id must match selection; omission binds the selected
page at admission. Translate it to the registered driver identity and transport
that identity plus the immutable execution/lease through the existing Go Client.
The driver uses recordingOwner and the selected page identity across title/CDP
awaits and detach. Revalidate API Session/page before returning. Return conflict
for a stale selected page; no old direct-driver API bypass or ownerless SDK read
path remains. A read must not command navigation or extend mutation authority.

### Viewport command ownership — target080, RF038/RF047

The admitted Session/page owns the single SDK viewport mutation. Bind optional
public page_id to canonical selected-page identity, transport the immutable
execution/lease and driver page precondition, admit after complete body parsing,
and retain the captured Page through all awaited work. The API returns actual
positive dimensions from the canonical flat driver receipt; retire the unused
alternate ActualViewport response shape instead of layering a fallback over it.

Capture refresh observes the already-applied viewport on that captured Page.
Retain the existing notification entrypoint with bounded completion and current
stream/page checks. Retire CDP's second SDK mutation, small-change suppression,
pending-update dropping, public pending/result facades and polling's no-op
viewport updater. Existing capture transition generations choose the latest
refresh; existing SDK screenshot queue remains the viewport/capture serializer.
Do not introduce another queue, polling timer or parallel viewport authority.

The UI submits selected canonical page identity and bounds the request to its
session/page lifetime across config and transport. Cancel obsolete work; late
completions cannot mark another page synchronized. Preserve clamping, debounce,
bounds equality, resize feedback and explicit force/reset controls. Selecting a
new page must synchronize retained desired bounds without requiring a new
container resize. Validate actual receipts; a zero-sized or malformed success
is not synchronization. Native small/rapid resize and source-preservation checks
must distinguish effects from capture readiness and retained old frames.

### Preview capture admission — target081, RF007/RF038

The existing per-session HTTP preview cache owns its in-flight capture as well
as completed bytes. Identical concurrent reads share that promise at unchanged
quality, scale and full-page policy. Capture identity includes selected source,
viewport dimensions and URL; a resize must not return the old cached geometry.
The session cache slot is a lifetime: clear/cleanup retires pending work, which
cannot publish a response or repopulate a replacement slot. A failed capture
must leave the current slot retryable, without deleting newer pending work.
Different fidelity requests retain their own results; only the current request
may update the retained cache. Keep one slot per session, bounded to one retained
result and one shareable pending capture, with existing TTL and teardown callers.
Do not introduce a parallel browser queue, new service, weaker watchdog, lower
fidelity or per-client persistent cache. This addresses redundant admission;
browser-wide rendering cost and sustained stream throughput need separate proof.

### Capture device scale — target082, RF016

CaptureRequest dimensions own an explicitly supplied device scale on either
preset or explicit viewport paths. The capture-to-execution translation carries
that value through the existing browser-profile fingerprint into SessionSpec;
it must not silently drop it and use the driver's default. Clone the caller's
profile before applying this override, retaining its other fingerprint, identity,
proxy and behavior settings. When dimensions omit device scale, preserve the
supplied profile/default policy. No parallel driver setting, SDK patch or default
DPR change. Independently verify requested CSS viewport, renderer DPR and PNG
bitmap dimensions through both public capture surfaces before measuring latency.

### Capture request admission — target083

CaptureService validates the generated request's existing protobuf constraints
before URL discovery, execution or artifact export, including dry runs. Reuse
the repository-approved protovalidate runtime so the protobuf remains the owner
of numerical limits; do not copy those limits into a second Go policy. Initialize
the validator once when mounting the service and compile the CaptureRequest
descriptor before serving. Invalid input returns InvalidArgument; a validator
configuration failure must not masquerade as a caller error. Existing capture
normalization rejects unspecified and unrecognized enum numbers before effects.
The viewport resolver retains paired-dimension and preset-precedence semantics.
Omitted optional dimensions and browser-profile fields retain their defaults.

### AI suggestion optional text — target084, RF068

The provider schema and parser share one contract. Action, confidence and category
remain required and validated. Description, element text, selector and reasoning
are optional text: omitted or explicit null means absent metadata, represented by
the existing empty string in AISuggestion. Numbers, booleans and objects are not
text and remain invalid. Accepting absence must not manufacture content, silently
repair required fields, relax confidence/category constraints or mask a provider
failure as an empty suggestion list. Keep the existing governed local-provider
route and external API representation; verify the actual failing search fixture.

### Synchronous execution completion — target085, RF105

The existing per-execution goroutine owns completion. Its starter returns one
channel, closed after the runner and its deferred artifact/event/cancellation
cleanup return. Both saved and ad-hoc synchronous APIs wait on that channel or
their request context, then read the persisted terminal receipt once. Remove both
250 ms polling loops; do not replace them with shorter polling or a second global
completion registry. Closing the channel preserves an already-completed result
without a lost wakeup. A retired runner without a persisted terminal timestamp
is an explicit error carrying its execution ID, never an unbounded wait or a
success inferred from goroutine exit. Request cancellation retires only its wait;
the existing detached execution context and explicit stop ownership remain.
Asynchronous response shapes and terminal status/error/timestamp projection stay
unchanged. Qualify delayed teardown, failed terminal publication and cancellation
in addition to latency; retained artifacts must be available at return.

### Execution admission metadata — target086, RF106

The workflow service commits one immutable admission snapshot before publishing
the new database execution index or starting a runner. The snapshot owns resolved
workflow version, trigger and the actual input namespaces; manual flat inputs are
stored under InitialStore. Saved, manual, ad-hoc and resumed admissions use the
same commit boundary. Resolve omitted workflow versions to the actual selected
workflow version. Existing DB lifecycle fields remain authoritative in public
hydration; remove running/terminal snapshot replacements rather than merging two
mutable status stores. The artifact README must identify snapshot status as the
admission state and direct current-status consumers to the execution API.

Use the existing api-core atomic file writer for complete-file publication and
file fsync; this alone does not qualify parent-directory/power-loss durability.
Failure to commit metadata rejects admission before a browser effect or index
publication. If index creation fails after the file commit, return an error with
the attempt ID and retain its metadata: a database error can have an uncertain
commit outcome, so deleting the only recovery receipt would be unsafe. Do not
retry effects or invent a cross-store transaction framework. Historical lost
metadata cannot be reconstructed from this repair and remains explicitly absent.

Qualify parameters/version/trigger during running and terminal states, unchanged
snapshot bytes, resume version rejection, all admission callers, real filesystem
failure, index failure with retained metadata, and preserved existing lifecycle
and native workflows. Resume context propagation and terminal DB/event faults
remain separate concerns unless evidence requires extending this intervention.

### Resume continuation — target087, RF107

A completed step at index0 is a real checkpoint. Represent checkpoint presence
explicitly in the executor request; fresh execution has no checkpoint, and resume
continues after the identified step. Retire the ambiguous default-zero/-1 scalar
convention and convert all callers without a runtime compatibility branch.
Resolve continuation before tab restoration, entrypoint navigation or action
execution. Flat plans use their actual instruction order, not numeric less-than
filtering. For a graph whose reachable path is one acyclic chain without loops
or subflows, follow its edges and start at the checkpoint's successor. Preserve
original node identities, indices and full plan evidence; do not renumber steps.
A checkpoint at the end has no remaining effects.

A scalar step index cannot encode branch decisions, loop iterations or a subflow
call stack. If the graph requires those, or the checkpoint is absent/ambiguous,
return an explicit recovery error before browser effects. Never pretend to resume
by starting at the root or by skipping arbitrary lower static indices. Rich
control-flow recovery remains an open capability until a durable cursor and
state receipt are implemented; this safety boundary is not full resume
qualification. Fresh branch/loop/subflow execution stays supported.

Prove the ordinary compiled two-step workflow resumes after step0 with exactly
one original fixture effect, plus flat/graph/fresh/end/invalid/ambiguous controls.
Keep the independently observed routed-context, lineage and richer recovery-state
problems visible. Historical missing version/input data must not be invented.

### One execution admission and runner for resume — implemented088, RF108

Resume prepares an ordinary version-pinned execution request with recovered store,
parameter overrides and the original full execution settings. It then uses the
same saved-workflow admission and runner as a fresh execution, carrying explicit
resume checkpoint and original execution identity as execution options. Preserve
resumed trigger/lineage in the index and public hydration. Detach request
cancellation once while retaining routed storage metadata and the browser routing
header; explicit StopExecution owns runner cancellation, joins the runner through
terminal persistence and deferred cleanup before reporting success, and terminal
writes retain their routing. If a step after the last successful checkpoint has
`INSTRUCTION_OUTCOME_UNCERTAIN`, reject resume until the browser-side effect is
reconciled; transport failure alone does not prove that the action did not run.
Delete the private resumed starter and executor lifecycle.

Recovery must retain original browser, artifact and other execution settings,
with an explicit resume URL overriding the saved start URL. Missing original
workflow revision is an explicit non-resumable condition, not permission to guess
at the latest workflow. Preserve original snapshots and input parameters. Remove
Request.InitialVariables: its sole caller already supplies the same store via
InitialStore. Initialize state once with plan variables overridden by InitialStore,
plus the existing params/env namespaces; preserve both actual input sources.

Verify public ResumeExecution routing/header, detached caller cancellation,
explicit stop/terminal status, initial settings and lineage, original data,
missing-version rejection and the087 repeated-effect fixture. This does not add
a branch/loop/subflow recovery cursor or repair historical erased metadata.


### Checkpoint cursor and store share one persistence owner — implemented089, RF109

The executor owns the actual mutable store. After each successful step's store
mutation and outcome publication, persist its cursor and complete store together
through the execution writer. Store recovery data as a private, atomically
replaced, owner-readable execution file, independent of screenshot/extracted-data
collection policy and excluded from timeline/events. Do not reconstruct mutable
state from presentation previews or recompute prior actions during recovery.

The checkpoint reader must validate format, execution identity and agreement
with durable successful outcome evidence before admitting resume. Missing,
malformed or disagreeing state is explicitly non-resumable; never substitute
initial values for unknown completed mutations. Preserve immutable admission
metadata and original files. Historical records without this evidence cannot be
silently upgraded. Convert the existing checkpoint interface and all callers;
remove the timeline-preview accumulator and duplicate progress-only checkpoint
implementation instead of adding a parallel recovery path.

Qualify static set_variable, named storeResult, plan defaults, nested values and
collection-disabled cases with expected browser/engine inputs. Faults must not
publish a usable mismatched checkpoint or leak store values through public step
payloads. Keep branch/loop/subflow cursor recovery and atomic coordination with
external effects explicitly outside this bounded guarantee; crash-window
qualification remains required.

089 adjacent typed result contract (RF110): the existing action result-key owner
must recognize Extract.store_as and Evaluate.store_result. Apply that named store
mutation before committing the same outcome's recovery state; do not flatten raw
extracted keys into the store. Fresh execution and resume use the same semantics.

### Capture qualification through its measurement owner — target090

The rehabilitation capture outcome needs a maintained producer and an authoritative
read, not a hand-entered board value. Promote the independent 100-capture fixture
from dated experiments into BAS-owned source. Retain its first-attempt denominator,
one declared warmup, screenshot+computed snapshot fidelity, exact viewport/DPR,
independent observations, all failures and artifacts, candidate identity and raw
samples. The <=2s p95 band stays unchanged.

Performance Health owns running and evaluating declared performance workloads;
Test Genie remains the admission/evidence owner for the performance phase. Prefer
a small declared-workload adapter to a BAS-specific exception or a second BAS job
system. A maintained scenario-owned producer command may implement the fixture;
its executable/configuration digest and actual invocation belong to the owner
receipt. No API accepts a caller's pass flag or an arbitrary uploaded historical
receipt as qualification. Missing, stale, malformed, incomplete or mismatched
producer evidence must remain unknown or fail, never default to zero/success.

The governed setpoint read may consume a current, owner-produced workload receipt
only after candidate applicability and contract identity are established. Existing
mixed execution statistics and unrelated build/Lighthouse success cannot satisfy
capture. Other sixteen outcomes remain unknown until their own evidence exists.
Retain raw receipts with the producer and bounded source references in BAS progress.

### Requested capture frames — target091, RF016

CaptureService owns the requested final screenshot: append one explicit screenshot
 after readiness, direction, interaction and inline snapshot work when the request
 asks for screenshots (including the default). Preserve an explicit element selector
 and explicit screenshots inside the interaction flow. Non-image captures need no
 final screenshot unless a selector explicitly requests one.

Use the existing artifact policy and telemetry pipeline with a named capture
 profile: retain explicit image actions and failed-step diagnostics, without passive
 successful-step images. Validation's navigation/assertion checkpoints remain a
 separate setting of the validation profile. Product/replay profiles still retain
 every step. Other capture evidence remains available through its existing owner.
 Do not deduplicate exported files after browser work, weaken pixel fidelity, drop
 requested evidence, or turn an explicit screenshot failure into successful capture.

Explicit `DOM` and `DOM_TREE` capture types each add their own post-interaction
evaluate action, then materialize bounded `dom.html` and `dom-tree.json` files
before artifact storage publication and result-summary writing. These artifact
requests do not implicitly populate the larger inline response fields; callers
must opt into those fields separately. A missing evaluate result leaves that
artifact explicitly unavailable, and a truncated generated file carries
`metadata.truncated=true`.

When the driver finalizes browser recordings, workflow folder export copies each
regular `.webm` file from the execution's `artifacts/videos/` directory into
`videos/`. The VIDEO capture producer returns one artifact per exported file;
missing recordings are reported as unavailable. Device-specific video capture
still requires its own target evidence and is not implied by browser recording.

### Capture interaction boundaries — target092, RF111

Capture composes the interaction by graph boundaries, not node-array positions.
 The compiler owns interpretation of workflow topology, including loop bodies.
 Its boundary query must identify one outer entry and the outer terminal nodes
 without resolving target URLs/selectors or executing actions. A malformed or
 disconnected interaction fails before admission. Capture connects readiness to
 the identified entry and every applicable terminal to its requested postlude;
 branch/loop edges remain unchanged and loop-body terminals do not gain exits
 to the capture postlude. A single-node flow remains supported.

Reuse the compiler's planner and loop-body extraction. Avoid a second graph
 algorithm in the handler, a synthetic one-iteration loop or private runner,
 temporary saved workflow files, and a duplicate full compile with different
 target options. Preserve exact action/edge definitions and error semantics.

RF114 amendment to target092: WorkflowEdgeV2.label is the declared branch
 condition. Compiler topology projection shall preserve it in PlanEdge.Condition.
 Remove the unreachable V1 data.condition reader and internal camel/snake handle
 fallback: proto inputs are normalized once and emitted with canonical proto field
 names. External accepted protojson spellings still decode through the protobuf
 owner. No migration or secondary branch representation is needed.

### Typed assertion semantics — target093, RF113

The existing driver assertion handler owns all eight typed AssertParams modes.
Negation changes the expected predicate, never the meaning of an evaluator error.
Presence and visibility use Playwright's locator state waits for the requested
polarity; only a genuine browser timeout becomes an unsatisfied predicate. Invalid
selectors, closed pages and other evaluation failures remain errors. Explicit
timeout zero requests an immediate observation, not an unbounded browser wait.
Text and attribute comparisons retain the observed value, honor case_sensitive
(default true), and negate only after a successful read. They wait for the target
with the declared timeout; a separate Wait node governs delayed value changes.
A missing attribute differs from an empty attribute. Custom failure_message applies
to a logical mismatch, without concealing engine errors. Assertion evidence keeps
mode, expected/actual, negated, caseSensitive, success and message through the
existing typed outcome pipeline. Frame targeting remains getDocument's policy.

Remove unreachable string aliases and regex/expression paths: the supported
AssertParams enum has eight DOM modes and no regex or JavaScript expression mode.
Do not invent new wire variants or another predicate runtime. Conditional action
support (RF112) remains a separate follow-up with page-JavaScript and workflow
variable ownership preserved. Existing compiler topology cohesion debt is retained
for review; do not split fixtures merely to evade a file-length threshold.

### Typed conditional execution — target094, RF112

Conditional actions have three typed modes. Page JavaScript and element presence
belong to the existing driver and selected-frame owner; workflow-variable reads
belong to Go's execution store. A successfully evaluated false predicate is a
successful step carrying ConditionOutcome.outcome=false. An evaluation error is
a failed step with no condition truth value. Negation applies only to a completed
evaluation. A failed conditional can take an explicitly wired error/failure edge
under ordinary continuation policy, but never fall through to the first true/false
edge. An absent selected edge terminates that path rather than running its opposite.

Driver outcomes use the existing typed ConditionOutcome field through the existing
StepOutcome builder and wire decoder, including actual/expected scalar or JSON
values. Do not add another extracted-data condition convention. Go variable
conditions use the same value-comparison policy as loop conditions, the actual
execution store, and the existing outcome/event/checkpoint owner. Unknown variables
or operators are evaluation errors. False results do not fail execution.

Page expressions support expression syntax and the function-body syntax emitted
by the builder. Parse before executing; a runtime exception must not cause the
script to execute again under a syntax fallback. Preserve Promise results, frame
context, and the ordinary execution deadline/cancellation owner. Arbitrary scripts
are not automatically retryable after an error. Element mode means presence, as
in the node contract, and polls up to its declared timeout; a genuine timeout
means absent, while invalid selectors or closed pages remain errors. Zero timeout
means an immediate observation. Variable and script conditions observe once; a
Wait node governs delayed values. No duplicate browser/session or generic runtime.

Qualification includes both branch truths, both negations, variable operators and
missing variables, expression versus body syntax, exception-after-effect exactly
once, malformed selectors/closed-page errors, delayed presence, zero timeout,
selected frames, typed evidence round-trips and unsupported operators. Keep
explicit failure edges distinct from false edges and retain native action logs.

094 native amendments (RF115/RF116): iframe capability means automation inside
frames, independent of viewer transport. The managed Playwright engine must
admit its implemented frame actions. Condition evidence belongs in shared event
context as well as driver outcomes. Move the existing ConditionOutcome definition
from execution/driver.proto to base/shared.proto (same fully qualified protobuf
message and wire fields), then reference it from EventContext at new optional
tag26. Update all workspace SDK import sites; no duplicate schema or compatibility
wrapper. Move the existing condition conversions from protoconv to lower typeconv (which
does not import driver/export); reuse them at wire, telemetry, FileWriter and
export boundaries. The persistence regression must exercise the actual writer,
disk reload and export, since live telemetry conversion is a separate caller. Remove the narrower intermediate copy that loses Expression.
Generated Go/TS/Python and BAS manifest remain owned by scoped protogen generation.

### Execution status authority — target095

WorkflowService owns the pending→running→terminal index and its terminal
notifications. The running transition must persist before executor/browser
effects. Every terminal path (including compilation and target admission errors)
uses one finalization policy. Terminal notifications require successful terminal
persistence; a failed write remains an explicit error rather than a completed
receipt. Cancellation-independent persistence keeps routed context metadata and
sink retirement follows finalization. The writer owns artifacts and result-path
updates only. Remove its unused MarkCrash/status-mutation path and all no-op
implementations; this does not replace the executor's real step-failure evidence.

Validate with fault-injected running/terminal writes, independent executor effect
counts and captured lifecycle notifications, plus existing synchronous completion,
compile/target failure, cancellation, routed-storage and cleanup regressions.

### Profile snapshot ownership — target096

Before adding periodic profile checkpoints, make the existing capture-and-commit
path safe across detach/replacement. Session-profile service owns an opaque active
binding identity and serializes captures for that binding. Browser I/O remains
with the recording adapter. A complete storage/tab snapshot may commit only
while its original binding is still current; clear or replacement, including
same-profile reattachment, invalidates an older in-flight capture. Cancellation
while waiting or capturing must not acknowledge a save. Keep existing multiple
session associations and manual close/retry behavior; no new admission restriction.
Manual and future periodic callers use the same fenced aggregate transaction.
Do not introduce a timer that bypasses this owner. RF011's periodic capture and
crash recovery remain required subsequent work, not satisfied by fencing alone.

096 checkpoint extension: API lifecycle owns one joined checkpoint loop, stopped
before the driver. Every two seconds, eligible active bindings capture through
the same fenced transaction with a two-second browser-I/O deadline. Health reports
a stale (>5 seconds) or failed checkpoint. A profile shared by multiple active
browsers has no unique automatic writer: preserve existing manual behavior, skip
automatic replacement and report degraded checkpoint health rather than silently
selecting one browser's identity. Full shared-profile recovery remains an explicit
RF011 gap requiring a separately justified state-ownership design. Separate profiles
checkpoint concurrently; a slow browser must not hold another profile's capture.
Do not add per-session timer services, change saved profile format, merge opaque
storage, or claim crash qualification from an interval test.

Profile-scoped live operations follow the same ambiguity rule. Service-worker
control and history navigation accept a profile ID but act on one live browser
context. When more than one active browser is bound to that profile, the API
must return an explicit conflict and leave every browser unchanged; it must not
select an arbitrary session from registry iteration order. Keep the active
bindings intact and preserve the current single-session behavior. A future API
may accept an explicit session ID if that interaction is needed, but do not add
an admission restriction or silently apply a profile-scoped command to one of
several live contexts.
