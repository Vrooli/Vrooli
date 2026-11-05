# Abliteration Resource

Hosted abliteration.ai models exposed through OpenAI-compatible chat,
responses, and model endpoints.

## Intent

- Resource ID: `abliteration`
- Category: `hosted-service`
- Driver: `cloud-api`
- Portability tier: `full`

## Architecture

- `resource.json` owns endpoint, credential, health, environment export, and
  lifecycle metadata.
- `model-policy.json` is the single source of truth for concrete abliteration.ai
  model ids and roles.
- `cli/` is a thin adapter over `resources/internal/aiproviderapp` and
  `packages/ai-provider-go`.
- This provider is intentionally **not** part of the default fleet routing set.
  Use it only through an explicit scenario dependency or research role.


## Models

| Model | Context | Input | Output |
|---|---:|---|---|
| `abliterated-model` | 262K | Text + image | $1.00 / $3.00 per 1M |
| `abliterated-model-large-flash` | 1M | Text + image | $1.00 / $2.00 per 1M |
| `abliterated-model-large-v2` | 1M | Text only | $3.00 / $5.00 per 1M |
| `abliterated-model-large` | 1M | Text only | $3.00 / $5.00 per 1M (deprecated) |

These models reason before answering. A small `max_tokens` cap can be consumed
entirely by hidden reasoning and leave no visible answer; leave at least a few
hundred output tokens for short tasks.

Abliteration is text-output only. It accepts image input on `abliterated-model`
and `abliterated-model-large-flash`, but it does not generate images. Use
OpenRouter or OrcaRouter for text-to-image.

## Usage

```bash
vrooli resource install abliteration
resource-abliteration status
resource-abliteration policy roles --json
resource-abliteration policy resolve --role research.unrestricted.default --field model
echo "Research prompt" | resource-abliteration generate --role research.unrestricted.default --json
resource-abliteration list-models --json
```

Credentials are provisioned through the credential authority, never argv:

```bash
resource-abliteration configure
vrooli credentials provision --identity vrooli/abliteration --field api-key
```

## Notes

- `abliterated-model` and `abliterated-model-large-flash` accept image input.
  The two `large` models are text-only and reject images upstream.
- Concrete model slugs live only in `model-policy.json`.
- The initial catalog is sourced from abliteration.ai's public docs and should
  be verified against the live `/models` catalog before promotion.
