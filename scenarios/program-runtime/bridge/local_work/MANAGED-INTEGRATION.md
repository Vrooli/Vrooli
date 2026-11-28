# Disabled managed integration and recovery

7 November 2026. This extends the disabled native candidate. The route and lifecycle hooks are now registered **default-disabled**. It is **not started, configured, deployed or production-certified**. See [private process boundary](PRIVATE-IPC.md) for the latest implementation and activation contract.

## Implemented components

- `api/internal/localworkbridge/` is an isolated Go host seam for the existing Program Runtime owner. It provides `/internal/program-runtime/local-work/events`, bounded native request reading, serialized runner calls, a caller-owned blocking tick loop, cancellation and bounded shutdown. Disabled construction does no I/O/background work. Only runner HTTPStatus 204 succeeds; other tick results stop for owner supervision. Main.go now calls the lifecycle Register/Start/Stop hooks with Enabled left false and no runner/provider; this creates no child or tick goroutine.
- `managed.py` implements the Python injected runner ABI, `accept_native` and `tick` only. The finite `ipc_entry.py` binds it through strict private pipes; it has no listener or ambient credential discovery. Disabled dispatch returns 404 without storage/key/transport access. In-scope native 204 follows authenticated durable acceptance; authenticated out-of-scope events are explicitly ignored with 204 without opening storage. Idle ticks do not request keys or owner data. Persistent holds return fixed 503 states; arbitrary errors and credentials are never echoed.
- `owners.py` implements exact-ID lookup and declared collector calls using verified existing APIs. It admits explicit numeric loopback origins, exact selected paths, injected approved bearer principals, no proxies/redirects, bounded responses and current approval. No shell or operator impersonation is used. The complete owner cycle has a 150-second budget, under the Go host's at-most-180-second deadline and 300-second collection lease.
- Collection freezes the exact pending event set before calling the owner. A scope digest, immutable idempotency key and fixed ten-minute admission deadline commit before admission. The v2 program digest and inputs are pinned. The client first checks GetDeclaredExecution contract v2, then admits or resumes that original run through one bounded WaitForProgram. Lost responses, restart and newly arrived events do not silently create another collection. Failed/interrupted originals are retained, not automatically repeated. The verified principal must remain the same owner across calls.
- `recovery.py` stores local baseline/recovery records guarded by source/epoch, a current-state digest and a short-lived explicit recovery approval reference. Baseline never creates an event or claims complete coverage. Observed permission denial becomes a durable hold. `resume_same_collection` clears only held local backoff, preserving the exact event set, original admission and safety holds. Active leases and identity conflicts refuse recovery. Neither operation is exposed by webhook/tick dispatch.

## Verified owner calls

- Events: `GET /api/v1/events?event_id=<exact encoded ID>&limit=2`. Zero rows means retained canonical evidence is unavailable; one must match; duplicates/mismatch hold. The outbox separately verifies source/type/occurrence.
- `POST /vrooli.program_runtime.v1.library.LibraryService/GetDeclaredExecution`
- `POST /vrooli.program_runtime.v1.library.LibraryService/RunDeclaredProgram`
- `POST /vrooli.program_runtime.v1.programs.ProgramService/WaitForProgram`

Paths/JSON fields match current source/protobuf. Tests use injected in-memory transports. No endpoint was called by this implementation work.

## Checkpoint limits

The public Events query returns envelopes, not a certified stream watermark or contiguous source cursor. Selected Agent Manager publication is best-effort. Absence of an event does not prove absence of work. A local commit, highest ID, baseline or later event cannot establish completeness.

Thus `unknown_checkpoint` remains. Retention/replay/source-reset gaps and permission holds are not cleared by baseline or retry recovery. Checkpoints say `local_durable_observation_only` and `historical_coverage: unknown`. New source epochs, held export recovery, retention cleanup, verified gap closure and intentionally new admissions after failed original runs still require separate governed transitions. There is no generic reset/delete operation.

## Implemented process and lifecycle boundary

The earlier private IPC and startup placeholders are implemented. `ProcessRunner` resolves only injected approved configuration, verifies the prepared runtime/lock and exact seven-module source closure, invokes CPython 3.12 with `-I -S`, clears the child environment, supplies separate inherited credential pipes, bounds all input/output and kills/reaps the process group on cancellation. Python imports only the reviewed source modules, never their cached bytecode or competing packages/extensions. Non-Linux enabled execution fails closed.

The exact route is wrapped in numeric loopback admission, header/concurrency bounds and the existing shared middleware. Python verifies native HMAC before opening storage. In-scope native 204 requires authenticated durable acceptance; authenticated events outside the exact allowlist receive 204 and are ignored without storage. The lifecycle reports fixed scrubbed failures promptly, stops admission on loop failure and joins cleanup. The default main configuration never resolves a provider, opens storage, reads keys or starts a child/goroutine.

## Remaining concrete binding and qualification gate

1. Implement/select the production `BindingProvider` under the existing native authority. None is installed. It must return explicit expiring approved configuration and honor cancellation; it must not synthesize principals or load incidental ambient credentials.
2. Approve an immutable prepared CPython 3.12 runtime and module bundle, storage root and scope. Module/lock/config digests are checked, but the child reopens the verified source paths. Concurrent file replacement is not prevented. Trusted immutability during invocation is therefore a required binding property, not a claim established by the hash checks. The pyenv sentinel checks freshness, not interpreter provenance.
3. Bind exact credential purposes to their authority. Native webhook verification uses only the local Events key, which never goes to cloud. Local scoped principals must authorize the exact Events read and pinned collector observation/admission/wait calls. Dedicated cloud signing and private Site admission remain separate gates.
4. After approval, qualify the real immutable runtime, concrete provider, actual source/storage targets and private Flywheel access path. Offline Go helper-process tests and real Python pipe tests exercise both sides; they do not certify a deployed joined process or live native/cloud integration.

## Action-time activation gate

Before provisioning or enabling the concrete binding, obtain approval naming native owner/route/runtime/storage; source instance/epoch and three exact event types; local scoped access and fixed collector inputs/digest; dedicated expiring cloud key and secret placements; destination `https://decision-inbox-experiment.mdhalloran.chatgpt.site/api/bridge/local-work-events`; narrow data scope; subscriptions; startup/enablement; and named synthetic/real transmissions. Code-task approval or key presence does not authorize activation. No credential, subscription, service config or deployment changed.

Then verify actual source → durable native queue → cloud receipt → journal → handler pickup → useful many-to-many reconciliation → readback, followed by two separated authorized later changes without manual pickup. Preserve pause/mute, uncertain native liveness, valid local-only/cloud-only work and separate outcome acceptance.

## Validation

Python standard-library tests cover frozen selection, completion-time freshness, exact API shapes, uncertain admission, one-wait resume, expiry, scope/identity drift, CAS recovery and disabled defaults. Go tests use finite helper processes, fake runners and in-memory HTTP recorders with race/cancellation coverage; Python tests exercise actual finite CPython pipe dispatch and source-loader attack fixtures. No real listeners/endpoints are used. These are component checks, not managed scenario or live end-to-end certification.
