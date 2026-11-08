# Disabled managed local-work host

This package is now registered by Program Runtime main with Enabled left false and no runner/provider. Startup and cleanup hooks are present; they perform no credential reads, provider resolution, storage access, network calls, process starts or tick goroutines in that configuration. Disabled HTTP requests receive 404 without reading their body. No service configuration or deployment changed.

The host serializes `accept_native` and `tick`; native raw bytes and identity headers are preserved. Only status 204 is acknowledged: in-scope native events require authenticated durable receipt/duplicate; authenticated out-of-scope events are ignored without storage. Tick 204 can also mean an idle bounded cycle. Errors use fixed strings and never echo worker output. Native receipt has an at-most-eight-second whole-operation budget and 64 KiB body cap; tick has at most 180 seconds, below the durable 300-second lease. The route wrapper requires numeric loopback RemoteAddr, caps headers at 8 KiB and permits one admitted receipt. It does not trust forwarded identity. Existing shared API middleware remains outside the route.

`ProcessRunner` implements Linux-only finite private IPC with approved prepared CPython 3.12. It uses injected expiring `BindingProvider` configuration, exact seven-module hashes, private lazy credential pipes, an empty child environment, isolated Python flags, output caps and process-group cancellation/reaping. Non-Linux enabled execution fails closed. The lifecycle owns and joins its tick loop, reports scrubbed failures promptly and stops admission on failure.

No provider is installed in production. An inert Linux `SealedProvider` candidate now accepts only pre-delivered sealed anonymous capability descriptors and an already-isolated read-only whole-root image; see [the concrete binding contract](../../../bridge/local_work/PROVIDER-BINDING.md). Its trusted authority-delivery launcher, server-enforced narrow scopes and actual isolated runtime are still prerequisites. The owner must provide a trusted immutable prepared runtime/module bundle and approved private storage/scope. Hash verification followed by child path reopening does not prevent concurrent source replacement; the pyenv sentinel proves freshness, not runtime provenance. Provider methods must cooperate with cancellation. There is no ambient credential or PATH fallback, provisioning, automatic subscription or enablement switch.

See [the private IPC contract](../../../bridge/local_work/PRIVATE-IPC.md) and [managed owner/recovery contract](../../../bridge/local_work/MANAGED-INTEGRATION.md) for exact wire shapes, limitations and the action-time activation gate.

Focused qualification from the API directory:

`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -race -count=1 ./internal/localworkbridge`

Go tests use finite helper processes, fake runners and in-memory HTTP recorders. Python's 107 component tests separately exercise real finite CPython child dispatch and private pipe behavior. Main package and pyenv-go checks pass. Windows cross-compilation passes, not execution. These are offline component checks, not a deployed joined runtime/provider, scenario gate, source coverage proof or live end-to-end acceptance.

## Inactive canonical owner continuation

The finite broker now has canonical owner components: Events selected-record
reads, archived collector source/input resolution, canonical one-shot sessions,
durable per-selection gate binding, terminal reconciliation and replay-safe
tombstones. Explicit recovery requires a real stopped/joined execution boundary.
These components are unregistered; live authority, production transport composition and worker
confinement are still missing. An unregistered Linux SourceTransport now owns
bounded anonymous-pipe framing and joined cancellation with a supplied trusted
release boundary; see PRIVATE-IPC.md for its exact offline scope and limits. Collector terminal status is not cloud delivery.
See [the current ownership contract](../../../bridge/local_work/OWNERSHIP.md) for
exact guarantees, qualified test counts and remaining activation gates.
