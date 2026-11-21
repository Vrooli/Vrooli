# Private managed process boundary

7 November 2026. Implemented and tested as a disabled code candidate. No production provider, credentials, subscription, service start, deployment or transmission was configured or performed.

## Owner and activation state

Program Runtime main registers exactly `/internal/program-runtime/local-work/events` and lifecycle Start/Stop hooks with `Enabled: false`. Disabled calls return 404 without body reads, storage, credential access, processes or tick goroutines. There is no environment enable flag or key-presence switch. A future enabled configuration requires reviewed concrete code/configuration that supplies an approved `BindingProvider` and bounded Host options.

The native route admits only a numeric loopback RemoteAddr; forwarding headers cannot establish it. Existing shared middleware remains in place. It caps header bytes at 8 KiB, native body at 64 KiB and one admitted receipt at a time. Native body read plus child invocation have at most eight seconds, below the producer's ten-second timeout. A native HMAC is verified before opening the private outbox. For an in-scope native event, 204 means authenticated durable acceptance or an exact durable duplicate. Authenticated events outside the exact allowlist also receive 204 but are explicitly ignored without storage; this avoids retries for unselected event types. A tick may return 204 for an idle or completed bounded cycle. Status 204 alone is not proof of queued work, source authority or successful reconciliation.

## Binding provider contract

`BindingProvider.Resolve(ctx)` returns only non-secret `ApprovedBinding`. It must name an explicit already-prepared CPython 3.12 `pyenv.Spec`, reviewed lock digest, absolute `ipc_entry.py` path, entry/config digests, exact module digest map, non-secret configuration and a current approval window of at most seven days. No runtime provisioning, dependency installation, PATH fallback, shell or environment lookup occurs.

`BindingProvider.Credential(ctx, purpose)` is called lazily for an exact finite purpose. The provider owns approval, principal identity and scope and must honor cancellation. No production provider is implemented or installed by this slice. It must not substitute a generic user principal, cookie, MCP token, Events fleet key or incidental ambient credential for a missing capability.

Prepared-runtime verification checks approved lock bytes, existing bounded sentinel and executable existence. This proves freshness only. The provider must establish runtime provenance and immutable runtime/module storage for the invocation. Go verifies all seven application files before launch; Python later reopens source paths. Hashing does not prevent concurrent source replacement. Do not activate against the mutable shared checkout and claim executed-byte integrity.

Pinned application closure: `__init__.py`, `adapter.py`, `projection.py`, `owners.py`, `managed.py`, `ipc.py`, `ipc_entry.py`. The map must contain exactly these files. Missing/extra entries, digest mismatch, module symlinks/nonregular files, unbounded input or changed identity fail closed before credential access. Source modules are at most 1 MiB each. Python's restricted source-only loader compiles these exact `.py` sources without cached bytecode, extension modules, competing package directories or global package-parent sys.path insertion. The trusted interpreter/standard library remains part of the approved runtime.

## Concrete provider follow-up

An inert Linux sealed-descriptor provider candidate now exists; see [the concrete binding contract](PROVIDER-BINDING.md). It is not installed in main and cannot qualify the mutable shared host. The trusted launcher, immutable image, dedicated owner scopes and live joined runtime remain unimplemented/unqualified prerequisites. It does not turn metadata hashes into authority or retrieve values from a general credential backend.

## Private protocol

The initial stdin packet is bounded to 192 KiB, schema `flywheel.local-work-native-ipc.v1`, with exactly `schema`, `operation`, `request`, `config`.

- `operation`: `accept_native` or `tick`.
- `request`: exactly `body_base64`, `signature`, `event_id`, `idempotency_key`. Tick requires empty values. Body decodes to at most 64 KiB; each identity/signature field is at most 512 characters.
- Enabled `config`: exactly `enabled`, `storage_root`, `policy`, `owners`, `export`. The storage root is an approved absolute normalized path; filename is always `local-work-outbox.sqlite3`. Policy/owner/export fields map to their strict existing data classes. No source event may choose a path, executable, shell command, collector argument, URL or secret field. Those non-secret scope fields come only from the pinned approved binding.
- Disabled config returns 404 before opening inherited credential descriptors or storage.

Secrets do not travel in initial stdin, argv, environment, stdout or diagnostic errors. The Go parent clears the entire child environment and passes inherited anonymous pipes: child FD3 writes one compact JSON line `{"purpose":"..."}`; child FD4 reads exactly `{"value_base64":"..."}` or `{"error":"unavailable"}`. Secret responses are bounded to 16 KiB and decoded values to 8 KiB. At most four unique purposes are allowed and duplicate requests fail closed at the parent; the child caches each value only for its one invocation.

- `accept_native`: only `native_webhook_verify`.
- `tick`: only `events_reader`, `runtime_collector`; `flywheel_submit` additionally requires explicit export enablement.
- Local reader/collector values must be scoped Bearer authorizations; cloud signing is a separate 32-byte key. The native verification key never leaves the local child boundary.

Stdout is one compact JSON object with only `http_status`, restricted to 204, 400, 401, 403, 404, 409, 413, 429 or 503. The parent caps stdout at 128 bytes and discards/caps stderr at 4 KiB. Extra/duplicate fields, alternate output shapes, overflow and worker errors become a fixed unavailable error. Source data, credentials and worker diagnostics are never echoed to HTTP callers or owner logs.

## Process and lifecycle

Linux uses one finite child with its own process group. The exact approved Python path runs `-I -S ipc_entry.py`; no shell is involved. Operation deadline is at most 180 seconds and further bounded by current approval expiry and the host's native receipt deadline. Cancellation kills the process group, closes private pipes and waits for the child and cooperating credential provider. Output overflow also cancels. Other platforms compile but enabled process execution returns unavailable without provider access or process I/O.

One host serializes receipt/tick work. A tick does not run at startup. Tick errors or non-204 results stop the managed loop, report a fixed scrubbed OnFailure error promptly and stop further admission. Cleanup cancels and joins the loop and active call. A timed-out uncooperative provider/runner is a failure, not permission to launch another instance. Production providers must cooperate with contexts; this code cannot safely force-cancel arbitrary provider code.

## Qualification and remaining gate

107 Python component tests pass, including actual finite CPython child/FD dispatch, invalid HMAC before storage creation, disabled/idle no-secret behavior, malformed/oversized IPC, and valid-timestamp malicious bytecode/competing package-extension/parent-shadow fixtures. Go race tests qualify managed helper processes, cancellation/reaping, bounds, digest refusal, lifecycle failure reporting and disabled admission. Prepared-runtime package tests and API-main tests pass. Windows package cross-compilation passes; no Windows process execution is claimed.

These tests use public fixture keys, temporary storage and injected transports. They create no listener and invoke no live source/cloud endpoint. Go's process tests use a finite helper executable; Python tests exercise its real entry point separately. A live joined runtime/provider and full scenario gate remain unqualified.

Activation requires action-time approval of the actual native owner/route/runtime/storage, immutable bundle and digests, source identity/epoch and exact three event types, fixed collector inputs/digest, scoped local persistent access, dedicated expiring cloud credential and placements, exact private Site access and destination, narrow data scope, subscription, service enablement/deployment and named transmissions. Then separately verify source → native queue → cloud receipt → journal → handler pickup → source-backed many-to-many reconciliation → readback, including two separated later authorized changes. Unknown checkpoint/source coverage, permission holds, pause/mute and valid local-only/cloud-only work must remain explicit. Continuation of implementation is not activation approval.

## Inactive source-broker pipe transport continuation

The Linux `SourceTransport` component is separate from the older credential-pipe
`ProcessRunner` above. It is not wired to that runner, Python, main or a launcher.
It accepts two pre-supplied, distinct anonymous pipe ends with checked direction
and deadline support. No secret, URL, socket address, executable or environment
configuration is carried by the worker protocol. Disabled construction and use
perform no descriptor or dependency I/O.

Each request is a four-byte big-endian length followed by 1–4096 bytes of the
existing finite SourceBroker JSON. Each response has the same framing and at
most 1 MiB of canonical owner output. Invalid requests, authority failures,
truncated frames and write failures close the session without an error payload.
A trusted configuration limits the session to 1–64 operations, at most 180
seconds and at most five seconds per read frame/release. There is no retry loop,
listener, socket, process launch or admission-key renewal.

The broker retains its busy slot, exact permit and operation deadline through
the write. The supplied `SourceReleaseBoundary` must use the real source
revocation and launch-invalidation serialization, clip the callback to current
authority expiry, and permit fresh rechecks without recursive lock acquisition.
It invokes the callback exactly once synchronously. The transport rejects a
missing/repeated callback and records callback failure even if the boundary
swallows its return value. No production implementation of this boundary is
supplied. A callback interface or unrelated mutex is not proof of shared ownership.

Release means enqueueing bytes, not worker receipt, consumption or cloud
delivery. Writes are bounded to 4 KiB and recheck current authority before each
chunk. Cancellation closes descriptors before joining owner work, the context
watcher and already-started cancellation callbacks. One transport permits only
one Serve lifetime, including after failure. A timed-out Stop does not prove
owner or worker termination and must not permit replacement. Actual worker
process stop/join remains the control-plane launcher's separate responsibility.

A complete or partial frame may already have entered the pipe before a later
failure or revocation. Those bytes cannot be retracted. Partial frames are
unusable responses, never evidence that admission did not occur; reconcile the
same durable selection/request key rather than admitting a fresh attempt.
Pipe closure or EOF does not prove worker exit, because inherited duplicates
can keep descriptors alive. Deadlines and fresh rechecks do not establish a
strict no-byte-after-expiry kernel linearization point. Strict IPC expiry and
expiry after SQL driver commit begins remain unqualified gates.

Offline tests exercise real anonymous pipes, a finite test-owned subprocess
roundtrip and a real helper exit during a truncated request, malformed frames,
revocation before release, missing/repeated/swallowed callbacks, backpressure,
partial writes, cancellation and uncooperative-owner join behavior. They use
fixture source authority and owner responses. Existing canonical SQLite owner
regressions are rerun separately; there is no claim of a joined production
owner/transport/process path. No confinement, credential custody, filesystem
immutability, network denial, live source access or cloud transmission is
established by these tests. See the retained transport evidence summary at
`.vrooli/evidence/local-work-bridge-20261106/transport-20261107/SUMMARY.md`.
