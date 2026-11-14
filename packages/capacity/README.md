# Capacity broker

`packages/capacity` is the public adapter for scenario and resource Go modules
that cannot import the repository's `internal/capacity` package. It opens the
shared SQLite ledger, collects host capacity through an injectable source, and
uses the repository-owned `Decide` policy for RAM, CPU, and VRAM claims.

The broker serializes admission across processes that open the same SQLite
ledger, using a per-ledger OS lock before SQLite schema initialization and
around stale-claim cleanup plus the read-decide-write admission sequence.
Separate ledger paths use separate locks. The lock identity canonicalizes an
existing ledger or its existing parent directory when the ledger is new.
It provides renewable operation leases and stale-claim cleanup, with explicit
grant/degrade/queue/deny verdicts. It deliberately does not implement a second
policy or preemption. The public broker, strict no-reclaim control-plane CLI,
resource lifecycle, and operation reservation paths use the same canonical
ledger lock. Advisory reclaim paths must not hold it across external callbacks.
Direct ledger writers that bypass admission remain outside that guarantee.
Sensing, locking and ledger errors are returned as errors; a non-grant verdict
without an error means an evaluated capacity refusal that a scheduler may retry.

Operation leases expire after 15 minutes unless renewed through
`RenewableLease.Heartbeat`. Renewal is conditional on the captured claim
generation and cannot recreate an expired, released, or superseded claim.

For production RAM admission, `NewBroker` reads fresh Linux memory PSI and
uses the `substrate/SB20` threshold resolved by `internal/setpoint`. PSI at or
above that bar denies RAM admission. A known below-bar sample bypasses only the
legacy swap-occupancy hard gate; the ordinary RAM-fit and ledger checks still
run. If PSI is unknown, the configured swap threshold remains a conservative
fallback. An invalid active setpoint is surfaced as a denial. Injected
`NewBrokerWithSource` brokers do not read the real host pressure sensor; tests
can provide an explicit pressure reading.

All resource capacity adapters use the shared `companion` verb dispatcher for
label validation and the `capacity degrade --to <label>` / `capacity upshift
--to <label>` contract. The resource tier has one domain-specific degrade
implementation, Ollama's VRAM-aware planner; Whisper, Reranker, Kokoro, Kyutai
STT, and speaker verification delegate their step dispatch to the shared
companion verbs.

Companion safety contract: only a successful, well-formed ledger list can
establish no claim exists. Unknown/error/ambiguous lists skip all mutations and
are not cached as empty. Runner and request-edge Reporter share exact resource
owner, VRAM, GPU and active-status selection; duplicate matching rows are refused.
A successful claim/resize/heartbeat response supplies the validated current
identity/generation for an immediate activity report. Release never reports
activity; mutation errors or malformed responses skip subsequent activity.
A list snapshot and mutations can still race external writers: CAS rejection
remains fail-open and retries on a later real observation; no identity guessing
or compatibility payload fallback is allowed.

The selector validates the existing owner-kind enum (resource/scenario/op) and
resource-kind enum (vram/ram/cpu). Unknown enum values are malformed evidence,
not legitimate foreign rows; they cannot authorize a new claim. Known foreign
rows remain valid and are ignored by exact resource/VRAM/device selection.

`AcquireResources` reserves RAM/CPU under the caller's interactive, service or
batch priority. `AcquireScoped` also owns the connection and heartbeat lifecycle.
A `Scope` cancels work on renewal loss, but retains its claims until the caller
has reaped the worker and explicitly calls `Release`; cleanup is independent of
request cancellation and failed releases may be retried.

Disk-producing scenarios use `DiskJobAdmitter.ReserveDiskJob` to atomically
reserve each canonical filesystem peak for the full durable job lifetime. The
owner can read `GetDiskJob` and `ListDiskJobs` receipts scoped to its owner ID.
After a crash, `ReconcileDiskJob` accepts explicit process-stop and cleanup
evidence, then the shared store independently checks that the recorded process
identity is gone before releasing the reservation. This receipt path does not
grant admission; new work still requires `ReserveDiskJob`.

Terminal `actualDeltaBytes` reports the filesystem available-byte decrease
since reservation and may include unrelated host activity. Receipts label this
as `filesystem_available_bytes_decrease`; use the owner-supplied monotonic path
samples for job-attributed evidence, and report both measurements separately.

`DiskJobPath.DockerRootDir` binds a pre-provision reservation to the Docker
daemon's authoritative storage root, before a volume or container exists.
Callers can type-assert the optional `DiskDockerPathSampler` interface to use
`SampleDockerPath` with the same selector when they have a repeatable byte-count
method for that root. Prefer exact Docker mount and upperdir selectors for
per-path samples once those paths exist. Ordinary `DiskJobLease` implementers
do not need Docker-specific methods.

`UseGate{Root: collectionRoot}` protects model artifacts independently of RAM
accounting. Readers call `Acquire(id)` and installers/removers call
`AcquireExclusive(id)`. All processes must use the same collection root and ID.
The lock files are never unlinked by artifact removal, and the operating system
releases a crashed process's ownership. Warm workers hold both artifact-reader
and memory ownership through residency; per-operation CPU/GPU claims end after
worker execution/reap. Music Tools and Image Tools share the scope lifecycle.
