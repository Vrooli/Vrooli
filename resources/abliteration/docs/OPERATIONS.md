# Operations

`abliteration` is a `cloud-api` resource.

## Boundaries

- `resource.json` owns endpoint, credential, export, lifecycle, and health metadata.
- `model-policy.json` owns role and concrete model selection.
- `cli/` owns only provider identity and wiring over `resources/internal/aiproviderapp`.
- AI Gateway owns routing across providers; this resource owns only its own
  credentials, endpoint, and policy.

## Required surfaces

- `policy roles|resolve|models|constraints`
- `ensure --config-base64`
- `generate --role <role>`
- `list-models`
- `show-config`
- `configure`

## Failure handling

Provider failures are emitted as typed `VROOLI_PROVIDER_ERROR` stderr lines
from the shared `ai-provider-go/providererror` package, preserving HTTP status
and `Retry-After`.
