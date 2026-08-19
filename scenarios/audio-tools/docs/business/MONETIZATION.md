# Monetization — Audio Tools

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: `not-applicable` is
better than inventing a commercial story.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

> **Status: active — the three-tier chain is already built here.**
>
> This document said `deferred`; the code does not. `docs/concepts/INTEGRATIONS.md`
> documents a shipped **local → BYOK → Vrooli** chain across all three
> capabilities, with every local resource marked `required: false` so the
> scenario runs cleanly with zero local models. That is the monetization model
> in `path:docs/monetization/evidence/FINANCIAL_MODEL.md` already implemented —
> the doc was drifted, not the product.

## Role In Vrooli

- **Direct product: yes, candidate.** Local, privacy-aware dictation and TTS is
  standalone-appealing — it does not require the rest of the fleet.
- **Internal capability: yes.** Consumed by web-console today, with
  swarm-manager, agent-manager, and phone-agent named as future consumers.
- **SKU/bundle candidate: yes.** Not yet in
  `path:docs/monetization/catalogs/scenario-sku-map.json`; that file is governed
  (`catalog-strategist` proposes, human curates), so propose rather than edit.
- **Revenue line: gateway-driven variable cost**, same shape as every Tier 1/2
  line in the financial model.

## The Three-Way Chain (already implemented)

| Capability | Rung 1 — local | Rung 2 — BYOK | Rung 3 — Vrooli |
|---|---|---|---|
| STT | `whisper` batch and optional `sherpa-onnx` streaming resource | `openai-whisper`, `deepgram` | LPBS-gated |
| TTS | `sherpa-onnx` native adapter with the Kokoro voice catalogue | `openai-tts`, `elevenlabs` | LPBS-gated |
| Summarize | `ollama` resource | `openrouter` | LPBS-gated |

Credentials travel per-request in `X-Audio-BYOK-{Provider,Key}`; the Vrooli tier
authenticates with `X-Audio-LPBS-Token`. Adapters never log unredacted keys.

**Every local resource is `required: false`.** A user with no local models and
no key of their own still gets a working product through rung 3 — which is
precisely the subscriber the financial model describes.

### Open gaps

- **Rung 3 is gated off.** `AUDIO_AI_ENABLE_VROOLI=false` until
  `execute/lpbs-audio-gateway-endpoints` ships. The revenue rung is built and
  switched off, so nothing here is billable yet.
- **This chain bypasses ai-gateway.** It reaches the subscription directly
  through landing-page-business-suite, while `scenarios/ai-gateway` is the
  fleet's designated inference router and carries the route-evidence metering
  (`OT-P1-007`). Two independent paths to one subscription will produce two
  metering stories and two billing bugs. Convergence is a prerequisite for
  trustworthy unit economics — tracked in ai-gateway's monetization doc as well.
- **Audio does not meter in tokens.** STT bills per minute of audio and TTS per
  character or per generated second. A token-only credit allowance will
  mis-price audio in either direction. The allowance model needs a second unit
  before audio can be included in a Tier 1 bundle.

## Customer / Buyer

- **Primary user:** someone who dictates regularly and wants it private, local,
  and reliable across interruptions — the trust properties in `OT-P0-001`.
- **Buyer (subscription):** the same user at the moment they have no capable
  local hardware and no provider account, or who wants transcription on a phone
  where local models are not viable. Mobile is a strong rung-3 case precisely
  because rung 1 is weakest there.
- **Pain:** cloud dictation means sending voice to a third party; local
  dictation means managing models and hardware; neither travels across devices.
- **Existing alternatives:** OS built-in dictation (free, shallow, cloud-backed),
  Otter/Rev/Descript (hosted subscription, no local option), raw Whisper (free,
  requires setup). The differentiator is that **the same product runs fully
  local, fully BYOK, or fully hosted** with no change in behaviour.

## Customer / Buyer

- Primary user: define during PRD generation.
- Buyer: define during monetization review.
- Pain: define from demand evidence.
- Existing alternatives: capture through market validation.

## Packaging

| Packaging Option | Status | Notes |
|---|---|---|
| Standalone app | deferred | Revisit after first real domain is implemented. |
| Bundle component | deferred | Map in project-level monetization catalog if promoted. |
| Add-on | deferred | Use only when scenario clearly extends another SKU. |
| Service/consulting assist | deferred | Consider if this scenario accelerates done-for-you delivery. |

## Pricing Hypothesis

- **Model:** free and fully functional on rung 1 or rung 2; revenue only on
  rung 3, as part of a subscription allowance rather than a per-scenario price.
- **Comparable products:** hosted transcription services price per minute;
  consumer dictation apps price per month. Both anchors are useful, and they
  disagree — which is why the metering unit matters more here than the number.
- **Willingness-to-pay evidence: none captured.**
- **Cost drivers:** wholesale STT minutes, TTS characters, and summarize tokens.
  **Unlike LLM-only scenarios, cost is dominated by audio duration rather than
  token count.** Long-form dictation (`OT-P0-001`) is explicitly a supported
  workload, so the heavy-user tail is longer here than in text scenarios and
  needs metering before a bundle includes it.

## Validation Plan

- **Demand signal needed:** rung-3 fall-through — the share of users running
  neither local models nor BYOK. Mobile users are the expected concentration.
- Channel: define in [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- Success threshold: define from project-level monetization taxonomy.
- Revisit trigger: first real domain reaches validated scenario tests
  and has a clear user/customer.

## Current Status

`draft` — the three-tier chain is implemented and documented in
`docs/concepts/INTEGRATIONS.md`. Rung 3 is switched off pending
`execute/lpbs-audio-gateway-endpoints`. Two open questions gate inclusion in a
paid bundle: a per-minute/per-character metering unit, and convergence with
ai-gateway rather than a direct LPBS path.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- [`../../../../docs/monetization/README.md`](../../../../docs/monetization/README.md) — project-level monetization strategy
