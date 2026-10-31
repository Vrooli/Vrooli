# ai-provider-go

Shared inference-provider contract and implementation kit for Vrooli AI API
resources.

This package is the owner-side counterpart to the scenario-facing
`packages/ai-go` consumer package. Resource CLIs use it to implement one
consistent provider surface:

- `policy`: the model-role catalog schema, loading, validation, and resolution
  rules shared by hosted and local provider resources.
- `policycmd`: the generic `policy resolve|roles|models|constraints` command
  behavior. Resources wrap these handlers in their own `cliapp` command group.
- `ensure`: role-only dependency validation for `resource-<provider> ensure`.
- `openai`: the OpenAI-compatible HTTP client, model-list normalization, chat,
  image, and usage decoding.
- `providererror`: the typed failure vocabulary and
  `VROOLI_PROVIDER_ERROR` cross-process marker.
- `conformance`: reusable checks every AI provider policy must pass.

The package owns **contract and mechanics**, never provider data. Each resource
keeps its own `model-policy.json`, `resource.json`, credentials, and any
provider-specific extensions such as OpenRouter spend caps.

## Consumer classes

`resource_runtime` is the primary consumer. `internal_platform` and scenario
classes are admitted because this package also carries the contract types used
by AI Gateway and scenario clients. It must not import the control plane's
private `internal/` packages.

## Related

- [Package governance](../../docs/package-governance.md)
- [Resource architecture](../../docs/resources/architecture.md)
- [AI Gateway role policy](../../scenarios/ai-gateway/docs/reference/roles-profiles-policies.md)
