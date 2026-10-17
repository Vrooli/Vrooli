# Capacity broker

`packages/capacity` is the public adapter for scenario and resource Go modules
that cannot import the repository's `internal/capacity` package. It opens the
shared SQLite ledger, collects host capacity through an injectable source, and
uses the repository-owned `Decide` policy for RAM, CPU, and VRAM claims.

The broker promises serialized admission, lease heartbeats, stale-claim
cleanup, and explicit grant/degrade/queue/deny verdicts. It deliberately does
not implement a second policy or preemption. Its `enforce` posture is
advisory: callers receive the verdict and choose their fallback.

All resource capacity adapters use the shared `companion` verb dispatcher for
label validation and the `capacity degrade --to <label>` / `capacity upshift
--to <label>` contract. The resource tier has one domain-specific degrade
implementation, Ollama's VRAM-aware planner; Whisper, Reranker, Kokoro, Kyutai
STT, and speaker verification delegate their step dispatch to the shared
companion verbs.
