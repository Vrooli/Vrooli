# Summarize Provider Chain

This document is the canonical architecture reference for text
summarization inside audio-tools. It explains how a `Summarize`
request is routed through the three provider tiers (BYOK →
Vrooli/LPBS → Local Ollama) and where the chain wires into the
usage-reporting pipeline.

Read this first when:

- adding a new BYOK summarize adapter beyond the OpenRouter starter,
- swapping the Local backend (Ollama → a different local LLM),
- changing summarization level wording ("light" / "moderate" /
  "heavy") or the default Local model,
- debugging "why is summarize falling back to Local when I set an
  OpenRouter key?".

The chain shape deliberately mirrors `ttschain` and `sttchain`; see
[`../tts/synthesis-pipeline.md`](../tts/synthesis-pipeline.md) and
[`../stt/streaming-pipeline.md`](../stt/streaming-pipeline.md). The
three are intentionally kept structurally identical so the next AI
capability slots in by copy-paste of the chain skeleton.

## Purpose

`summarizechain.Chain`
(`api/internal/ai/summarizechain/chain.go:10`) is the single home for
the question *"which provider summarizes this request?"*. It owns:

- tier precedence (fixed: BYOK → Vrooli/LPBS → Local),
- per-tier availability caching with the same TTLs as `ttschain`
  (BYOK 5 min, Vrooli 30 s),
- short-circuit error semantics for credit and configuration errors,
- routing of `req.Level` and `req.Model` overrides to the chosen
  provider's prompt template.

There is no streaming path; summarization is unary only. Unlike
`ttschain` there is no `Stream()` entry point because the wire shape
(`SummarizeService.Summarize`) returns a single payload.

## Inputs

`Summarize` Connect-RPC method
(`api/handlers/summarize/handler.go:46`) translates
`summv1.SummarizeRequest` into `summarizechain.Request`
(`api/internal/ai/summarizechain/interface.go:22`):

| Field | Source | Notes |
|---|---|---|
| `Text` | request body | Required. No client-side normalization. |
| `Level` | request body | `light` \| `moderate` \| `heavy`. Maps to a system-prompt variant per adapter (e.g. `summarizationSystemPrompt`, `api/internal/byok/openrouter.go:34`). |
| `Model` | request body | Optional override. BYOK adapters use it as the model id; Local passes it to Ollama. Blank → adapter default. |
| `TimeoutSeconds` | request body | Currently advisory; HTTP clients (`api/internal/byok/openrouter.go:25`) carry their own timeout. |
| `BYOKProvider`, `BYOKKey` | envelope | Skipped if blank. |
| `LPBSToken`, `UserIdentity` | envelope | Skipped if blank. |

The handler also reads an `X-Audio-Operation-ID` request header
(`api/handlers/summarize/handler.go:61`) and synthesizes a UUID when
missing, so every chain invocation produces exactly one
`store.UsageRow`.

## Outputs

`SummarizeResponse` carries the summary plus trace fields:
`Text`, `PromptTokens`, `OutputTokens`, `ProviderTier`, `ProviderId`,
`ModelId`, `LatencyMs`. The handler maps `summarizechain.Result`
(`api/internal/ai/summarizechain/interface.go:34`) directly.

A `store.UsageRow` is enqueued through
`usagereport.Recorder.Enqueue` (`api/handlers/summarize/handler.go:67`)
regardless of success — error rows carry the `Error` field populated
and a blank `ProviderTier`. See [`../usage/reporting.md`](../usage/reporting.md)
for the persistence pipeline.

Two admin methods round out the surface:

- `GetSummarizeConfig` (`api/handlers/summarize/handler.go:101`) returns
  the persisted defaults (auto-enabled, char threshold, level, model,
  timeout) from `intsumm.SummarizeConfig`.
- `UpdateSummarizeConfig` (`api/handlers/summarize/handler.go:109`)
  applies a `SummarizeConfigPatch` and persists via the injected
  setter.

## Internal Chain

```
SummarizeRequest (Connect)
        │
        ▼
envelope.FromConnectRequest          ← BYOK header parse, LPBS token, user
        │
        ▼
operation-id resolve (header or new UUID)
        │
        ▼
summarizechain.Chain.Execute         ← tier ordering + availability gates
        │
        ├──► BYOKProvider.Summarize           (api/internal/ai/summarizechain/provider_byok.go:27)
        │       │
        │       ▼
        │   adapter[req.BYOKProvider]         (api/internal/byok/registry.go:36)
        │       e.g. NewOpenRouterSummarize() (api/internal/byok/openrouter.go:22)
        │
        ├──► VrooliProvider.Summarize         (api/internal/ai/summarizechain/provider_vrooli.go:29)
        │       │
        │       ▼
        │   LPBS chat with Operation: audio.summarize
        │
        └──► LocalProvider.Summarize          (api/internal/ai/summarizechain/provider_local.go:27)
                │
                ▼
            summarize.Summarizer.Summarize    (api/internal/summarize/summarizer.go)
                │
                ▼
            Ollama HTTP endpoint
        │
        ▼
usagereport.Recorder.Enqueue         ← bounded async queue
        │
        ▼
SummarizeResponse                    ← trace fields + tokens
```

Execution order (`api/internal/ai/summarizechain/chain.go:59`):

1. **BYOK** iff `enableBYOK && req.BYOKKey != "" && availFor(BYOK)`.
   `ErrUnknownBYOKProvider` / `ErrMissingBYOKProvider` short-circuit;
   other errors fall through.
2. **Vrooli** iff `enableVrooli && req.LPBSToken != "" && availFor(Vrooli)`.
   `ErrInsufficientCredits` short-circuits; other errors fall through.
3. **Local** iff `enableLocal && local.IsAvailable(ctx)`. The local
   provider's availability check is `summarizer != nil && BaseURL != ""`
   (`api/internal/ai/summarizechain/provider_local.go:23`) — i.e. it
   trusts that Ollama is reachable when its URL is configured. A
   reachability ping happens lazily on the first call.

The Vrooli tier short-circuit on `ErrInsufficientCredits` is the
same design decision as `ttschain`: the user is owed an explicit
price-gate error, not a silent demotion to Local.

### Level → prompt mapping

The OpenRouter adapter (the only BYOK starter) maps `Level`
to a one-line system prompt
(`api/internal/byok/openrouter.go:34`):

| `Level` | System prompt |
|---|---|
| `light` | "Summarize in one sentence. No preamble. Just the summary." |
| `moderate` (default) | "Summarize in 2–4 sentences. No preamble. Just the summary." |
| `heavy` | "Summarize in 1–2 short sentences. Aggressive compression. No preamble." |

Local and Vrooli providers carry their own level handling
(`internal/summarize.Summarizer` passes `level` to the Ollama prompt;
LPBS hosts its own template).

## Seams

| Seam | Interface | Production | Test fake |
|---|---|---|---|
| Summarize provider | `summarizechain.Provider` (`api/internal/ai/summarizechain/interface.go:44`) | `LocalProvider`, `BYOKProvider`, `VrooliProvider` | Stubs returning canned `Result` per chain test (`api/internal/ai/summarizechain/chain_test.go`) |
| BYOK adapter | `summarizechain.BYOKAdapter` (`api/internal/ai/summarizechain/provider_byok.go:8`) | `NewOpenRouterSummarize()` (`api/internal/byok/registry.go:36`) | Per-test fakes |
| Vrooli client | `summarizechain.VrooliClient` (`api/internal/ai/summarizechain/provider_vrooli.go:8`) | LPBS HTTP client | Per-test fake |
| Local Ollama | `summarize.Summarizer` (`api/internal/summarize/summarizer.go`) | Wired with `BaseURL` from `ProviderConfig.OllamaURL` | Function-literal injection in service tests |
| Usage recorder | `usagereport.Recorder` (`api/internal/usagereport/recorder.go:37`) | `AsyncRecorder` backed by `store.UsageStore` | Synchronous fake in handler tests |
| Config accessors | `Deps.GetSummarizeConfig` / `SetSummarizeConfig` (`api/handlers/summarize/handler.go:30`) | Closures over the persisted `SummarizeConfig` | Per-test closures |

The settings domain reconfigures the chain at runtime via
`chains.Coordinator.Reconfigure` (same coordinator that drives
`ttschain` and `sttchain`); see
[`../settings/byok-and-voice-overrides.md`](../settings/byok-and-voice-overrides.md).

## Failure Modes

| Cause | Symptom | Chain behavior | Wire mapping (`mapChainError`, `api/handlers/summarize/handler.go:149`) |
|---|---|---|---|
| BYOK key present, `BYOKProvider` blank | Misconfigured envelope | Short-circuit | `CodeInvalidArgument` (`ErrMissingBYOKProvider`) |
| Unknown `BYOKProvider` | Not in registry | Short-circuit | `CodeInvalidArgument` (`ErrUnknownBYOKProvider`) |
| OpenRouter 401 / 403 | Invalid key | Adapter error; falls through | `CodeInternal` if no tier succeeds |
| OpenRouter 429 / timeout | Rate limit / slow vendor | Falls through to Vrooli then Local | Usage row records the error string |
| LPBS insufficient credits | `ErrInsufficientCredits` | Short-circuit | `CodeResourceExhausted` |
| LPBS transport error | Unreachable | Falls through to Local | `CodeInternal` if Local also fails |
| Ollama unreachable | Connection refused | Local returns error | `CodeInternal` |
| All tiers disabled | No eligible provider | `ErrAllProvidersFailed` | `CodeUnavailable` |
| Chain not wired | Bootstrap regression | Handler returns early | `CodeUnavailable` ("summarize chain not configured") |

Every failure path also enqueues a usage row with the error string
populated (`api/handlers/summarize/handler.go:75`) so failure rates
show up in the usage summary alongside success counts.

## Capacity Notes

Summarization is unary and stateless; concurrency is bounded by the
upstream provider, not by the chain. The BYOK starter
(`NewOpenRouterSummarize`) uses a 120s HTTP timeout
(`api/internal/byok/openrouter.go:25`); the Local Ollama path inherits
the timeout of `summarize.Summarizer.HTTPClient`.

The async usage queue has a fixed depth of 1024
(`api/internal/usagereport/recorder.go:34`); under sustained burst the
recorder drops rows newest-first rather than blocking the request
path. `summarize.Summarize` latency is therefore decoupled from
storage availability.

`availFor` caching means a freshly-rotated BYOK key keeps succeeding
on a stale "available=true" reading for up to 5 minutes; broken keys
keep being tried until the TTL expires. Settings updates that toggle
tiers invoke `Chain.Reconfigure`
(`api/internal/ai/summarizechain/chain.go:95`) which invalidates the
cache immediately.

There is no token-budget enforcement inside the chain. Adapters set
their own ceilings (OpenRouter caps at `max_tokens: 512`,
`api/internal/byok/openrouter.go:59`); operators who need a global
budget should add it at the handler edge, not inside the chain.

## Cross-References

- [`../../internal/SEAMS.md`](../../internal/SEAMS.md) — full seam registry
- [`../../internal/DECISIONS.md`](../../internal/DECISIONS.md) — durable decisions
- [`../../internal/PROBLEMS.md`](../../internal/PROBLEMS.md) — current drift
- [`../../reference/configuration.md`](../../reference/configuration.md) — operator-tunable levers
- [`../tts/synthesis-pipeline.md`](../tts/synthesis-pipeline.md) — sibling chain with identical tier shape
- [`../usage/reporting.md`](../usage/reporting.md) — where the usage rows go
- [`../settings/byok-and-voice-overrides.md`](../settings/byok-and-voice-overrides.md) — credential storage
- `packages/proto/schemas/audio-tools/v1/summarize/summarize.proto` — wire shape
