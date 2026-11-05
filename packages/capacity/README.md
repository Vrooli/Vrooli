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
policy or preemption. Its `enforce` posture is advisory: callers receive the
verdict and choose their fallback. This guarantee covers `packages/capacity`
broker admissions; direct ledger and resource-specific admission writers remain
outside this broker lock.

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
