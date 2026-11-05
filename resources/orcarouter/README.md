# OrcaRouter Resource

Hosted OrcaRouter gateway for OpenAI-compatible access to multiple upstream
model providers through one endpoint.

## Intent

- Resource ID: `orcarouter`
- Category: `hosted-service`
- Driver: `cloud-api`
- Portability tier: `full`

## Architecture

- `resource.json` owns endpoint, credential, health, environment export, and
  lifecycle metadata.
- `model-policy.json` is the single source of truth for concrete OrcaRouter
  model ids and the roles consumers resolve.
- `cli/` is a thin adapter over `resources/internal/aiproviderapp` and
  `packages/ai-provider-go`.
- OrcaRouter is OpenAI-compatible. Provider-specific request extensions belong
  in this resource, not in AI Gateway.

## Usage

```bash
vrooli resource install orcarouter
resource-orcarouter status
resource-orcarouter policy roles --json
resource-orcarouter policy resolve --role chat.default --field model
echo "Summarize OrcaRouter routing" | resource-orcarouter generate --role chat.default --json
resource-orcarouter list-models --json --limit 20
```

Credentials are provisioned through the credential authority, never argv:

```bash
resource-orcarouter configure
vrooli credentials provision --identity vrooli/orcarouter --field api-key
```

`ensure` validates a scenario's declared roles only:

```bash
resource-orcarouter ensure --config-base64 "$(printf '%s' '{"model_roles":["chat.default"]}' | base64 -w0)"
```

## Notes

- Concrete model slugs live only in `model-policy.json`.
- The initial catalog is sourced from OrcaRouter's public docs and must be
  verified against the live `/models` catalog before promotion.
- OrcaRouter is a normal hosted candidate for AI Gateway. It is not a coding
  harness resource.
