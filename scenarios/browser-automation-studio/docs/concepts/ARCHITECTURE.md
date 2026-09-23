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

## Shared Infrastructure

SQLite is routed through the scenario database layer; artifact storage and process lifecycle are scenario-managed. Test Genie owns scenario-suite execution.

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
| Workspace UI | Browser chrome, viewer, timeline, workflow editing and agent control | Subscribe by responsibility; frame rendering does not drive whole-workspace React updates. |

These are module boundaries within the current deployment. They do not prescribe
new microservices or replace the Go/Node/React stack.

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

### Recorded action conversion target (BAS-WORK-011)

The workflow deriver shall produce typed WorkflowDefinitionV2 nodes directly and
reuse the compiler's ActionDefinition builders. Remove its parallel V1 node/config
registry and second lossy V2 projection. Recording adapters only translate actual
capture fields (full text snapshots, absolute scroll coordinates and drag phases)
into the compiler's parameter vocabulary. Unknown actions fail with action index
and kind; they never become clicks. The service propagates conversion failure and
the handler passes the typed candidate to catalog validation without a JSON roundtrip.

A single recorded page binds to the fresh replay page. Until logical page/frame
lifecycle reconstruction is implemented, a candidate containing multiple page
identities or non-main-frame observations must fail explicitly; retaining raw IDs
as metadata alone does not qualify replay. Raw recording history is unchanged.
RF-030 remains open for usable multi-context reconstruction. Snapshot merging uses the final full value within an identical page/frame/selector
and does not mutate journal payloads. Explicit submit boundaries remain distinct.
Capture omission repairs remain RF-004 and are measured separately.

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
  preserving a concurrent close/reset. The in-flight reservation lasts until the
  action settles, even after reset completes; pooling and idle cleanup respect it.
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

### Recording navigation ownership (057 target)

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
