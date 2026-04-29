# Configuring Secrets

Secrets handling in Vrooli is layered. The architecture pieces are distinct and each does a real job — there's no redundancy, only discoverability gaps that this page exists to close.

## The layering

```
┌────────────────────────────────────────────────────────────────┐
│  vrooli-onboarding wizard       (first-run operator surface)   │
│  secrets-manager scenario       (ongoing audit/admin surface)  │
└──────────────────────────┬─────────────────────────────────────┘
                           │ reads + writes
                           ▼
┌────────────────────────────────────────────────────────────────┐
│  resources/<name>/resource.json                                │
│    credentials.env       — secretDescriptor[] (what & how)     │
│    credentials.secret_ref — Vault path (where stored)          │
└──────────────────────────┬─────────────────────────────────────┘
                           │ runtime reads via
                           ▼
┌────────────────────────────────────────────────────────────────┐
│  packages/api-core/secrets    — Go package; the code seam      │
└──────────────────────────┬─────────────────────────────────────┘
                           │ pulls values from
                           ▼
┌────────────────────────────────────────────────────────────────┐
│  resources/vault              — HashiCorp Vault, storage layer │
└────────────────────────────────────────────────────────────────┘
```

Each layer has one job. **Vault stores; resource manifests declare what is needed and where; api-core/secrets reads at runtime; secrets-manager audits and provisions; onboarding configures first-time.** New surfaces consume these layers; they do not duplicate them.

## The `secretDescriptor` shape

Defined in [`common.schema.json`](../../.vrooli/schemas/common.schema.json#/definitions/secretDescriptor). Used in `resources/<name>/resource.json` `credentials.env`:

| Field | Required | Purpose |
|---|---|---|
| `env` | yes | Environment variable name the secret is exposed as at runtime (e.g. `GEMINI_API_KEY`) |
| `label` | no | Human-readable display name shown in onboarding (e.g. "Gemini API Key") |
| `description` | no | Why this secret is needed and what it unlocks |
| `classification` | no | `infrastructure` (system-generated), `service` (scenario-specific), or `user` (operator-provided). Default: `user` |
| `required` | no | Whether the resource refuses to start without it. Default: `true` |
| `default_hint` | no | Hint shown next to the input (e.g. "Starts with 'AIza...'"). Not a default value — never auto-filled |
| `obtain_url` | no | URL where the operator can sign up or generate the credential. Onboarding renders this as a "Get one" link |
| `validation_pattern` | no | Regex for client-side typo-check. Lightweight only — not a security control |

The `env` field is the only required one — minimum viable descriptor is `{"env": "MY_KEY"}`, equivalent to the legacy bare-string form. Add fields as you have meaningful values; do not stuff placeholder content.

## Vault layout

By convention, Vrooli secrets are stored at:

```
secret/vrooli/<resource-or-scenario-name>
```

Each resource's `credentials.secret_ref` points at exactly one Vault path; the keys within that path correspond to the env var names declared in `credentials.env[].env`. Example for `resources/gemini/resource.json`:

```json
"credentials": {
  "env": [{"env": "GEMINI_API_KEY", ...}],
  "secret_ref": "secret/vrooli/gemini"
}
```

At Vault path `secret/vrooli/gemini` you'd have key `GEMINI_API_KEY` with the value the operator provided.

## End-to-end: adding a new credential

Follow this sequence when wiring a new third-party API key. Walk it once for any new integration; the steps are the same regardless of provider.

1. **Decide on a Vault path.** Convention: `secret/vrooli/<short-provider-name>`.
2. **Edit the consuming resource's `resource.json`.** Add or extend `credentials.env` with a `secretDescriptor` entry. Set `obtain_url` so the operator knows where to get the key. Set `secret_ref` to the Vault path.
3. **Read it from code via `packages/api-core/secrets`.** Do not read environment variables directly when secret values are involved — go through the package so audit trails and rotation work.
4. **Provision the value.** Two paths:
   - During first-run setup: the onboarding wizard's secrets step prompts for it.
   - Later or out-of-band: secrets-manager scenario provides an admin UI; alternatively, write to Vault directly using vault CLI.
5. **Verify.** Use `secrets-manager`'s validation flow, or hit the resource's health check, which should now pass.

## What about non-paste secrets?

OAuth tokens, device-flow credentials, and coding-agent sign-ins do not fit the paste-string shape `secretDescriptor` describes. They are handled separately — see [`integrations/external-auth.md`](integrations/external-auth.md). The schema for those is intentionally deferred until the first concrete integration lands.

## Rotation

Rotation is currently manual: update the value in Vault (via secrets-manager UI or vault CLI), then restart the consuming resource or scenario. Automated rotation is not in scope for the current configuration substrate; if it lands, it lands as a feature of secrets-manager and references this same `secretDescriptor` shape.

## Distinct from `secretReference`

`common.schema.json` also defines `secretReference` (with `name`, `key`, `namespace`, `source`). That's about *referencing a secret across backends* (Kubernetes, AWS Secrets Manager, etc.) and is used in deployment-time config. `secretDescriptor` is about *operator-facing metadata* — what to ask the user, how to label it, where to obtain it. Different purpose, different schema.

## See also

- [`resources.md`](resources.md) — `credentials.env` mixed-form usage
- [`integrations/external-auth.md`](integrations/external-auth.md) — non-paste credentials
- [`architecture.md#resolution-order`](architecture.md#resolution-order) — how credential metadata resolves to wizard UI
- [`secrets-manager`](../../scenarios/secrets-manager/) — the ongoing audit and admin surface
