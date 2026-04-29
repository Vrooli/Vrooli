# Video Generation Providers

This page describes how to wire third-party AI video generation providers (Seedance, Veo, Sora, Kling, etc.) for use in marketing AI-UGC and other video-producing scenarios.

## Status

**Provider integration is not currently wired in any scenario or resource.** This page describes the *intended* configuration shape using existing schema (`secretDescriptor`), so when the first scenario consumes a provider, the operator-facing surface is already defined.

The closest existing primitive is the `comfyui` resource for *local* image / video model hosting. That's distinct from the hosted-API providers covered here.

## Provider summary

| Provider | Model | Surface | Pricing model | API key naming |
|---|---|---|---|---|
| ByteDance | Seedance 1.0 (Lite, Pro) | BytePlus, fal.ai | Pay-per-second of generated video | `BYTEPLUS_API_KEY` or `FAL_API_KEY` |
| Google | Veo 3 / 3.1 | Vertex AI, Gemini API (paid tier) | Pay-per-second | `GOOGLE_GENAI_API_KEY` (via Vertex) |
| OpenAI | Sora 2 | OpenAI API | Pay-per-generation | `OPENAI_API_KEY` (existing) |
| Kuaishou | Kling | fal.ai, KlingAI direct | Pay-per-generation | `FAL_API_KEY` or `KLING_API_KEY` |

For pay-per-use without subscription, **fal.ai** is currently the cleanest aggregator: it hosts Seedance, Kling, and several others under a single `FAL_API_KEY`. Pricing as of 2026: Seedance Lite ~$0.18/sec, Seedance Pro ~$0.62/sec; an 8-second test video costs ~$1.50–$5.

## Recommended resource shape

When the first consuming scenario is built (likely `rich-media-studio` per [`docs/marketing/rich-media/README.md`](../../marketing/rich-media/README.md)), wire a provider resource using `secretDescriptor`:

```json
{
  "$schema": "../../.vrooli/schemas/resource.schema.json",
  "resource_id": "fal-ai",
  "category": "hosted-service",
  "credentials": {
    "env": [
      {
        "env": "FAL_API_KEY",
        "label": "fal.ai API Key",
        "description": "Aggregator for pay-per-use AI video and image generation models including Seedance, Kling, and Flux. Single key gives access to multiple providers.",
        "classification": "user",
        "required": true,
        "obtain_url": "https://fal.ai/dashboard/keys",
        "default_hint": "Starts with 'fal-...'"
      }
    ],
    "secret_ref": "secret/vrooli/fal"
  }
}
```

The same shape applies for `byteplus`, `vertex`, etc. — one resource per provider, each with its own `secretDescriptor` describing what to obtain and where.

## Why API-direct rather than SaaS UIs

Tools like HeyGen, Captions, Arcads, and Creatify wrap video models with subscription-billed UIs. They're useful for one-off testing without an API key but lock the workflow into their UI. For a programmable system that composes character / scene / product JSON ([rich-media architecture](../../marketing/rich-media/README.md)) and submits to a video model, direct API access via fal.ai or similar pay-per-use surfaces is the right shape.

This is the wrap-not-use principle (see project memory): wrap the underlying capability in a Vrooli scenario rather than depending on an external SaaS surface.

## Single-test cost path (no subscription)

For "does this even work" testing without committing to anything:

1. Sign up at fal.ai, top up $5 credit.
2. Add `FAL_API_KEY` to Vault under `secret/vrooli/fal` via secrets-manager.
3. Submit one composed prompt JSON (per the rich-media video-prompt template) directly via curl or a one-off script.
4. Receive an MP4. Total cost: ~$1.50–$5 for an 8-second clip.

The fully-integrated path through a `rich-media-studio` scenario is future work. Until then, the resource shape above is the operator-facing contract.

## See also

- [`../secrets.md`](../secrets.md) — `secretDescriptor` and Vault layout
- [`../../marketing/rich-media/README.md`](../../marketing/rich-media/README.md) — character / scene / product JSON that feeds video prompts
- [`../../marketing/rich-media/templates/video-prompt.template.json`](../../marketing/rich-media/templates/video-prompt.template.json) — Veo/Seedance-compatible prompt schema
- [`../../marketing/strategies/ai-ugc-personas.md`](../../marketing/strategies/ai-ugc-personas.md) — disclosure rules and what's allowed
