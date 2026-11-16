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
  streaming chat, image, and usage decoding. Streaming chat is consumed one SSE
  event at a time with `ChatStream.Next`; callers must close early-terminated
  streams, and request context cancellation interrupts body reads.
- `providererror`: the typed failure vocabulary and
  `VROOLI_PROVIDER_ERROR` cross-process marker.
- `conformance`: reusable checks every AI provider policy must pass.

The package owns **contract and mechanics**, never provider data. Each resource
keeps its own `model-policy.json`, `resource.json`, credentials, and any
provider-specific extensions such as OpenRouter spend caps.

## Model-rights impact

`cmd/rights-impact` evaluates the checked-in Ollama, Ollaya, image, and music
registries against the canonical trusted business context. Use `--json` for a
machine-readable path list. `--revenue-status`, `--annual-revenue-usd`, and
repeatable `--monthly-active-users PERIOD=COUNT|STATUS` flags construct a hypothetical
target; they do not edit the canonical context. Exact annual revenue is needed
when registries have different revenue thresholds. Add matching
`--compare-from-revenue-status`, `--compare-from-annual-revenue-usd`, or
repeatable `--compare-from-monthly-active-users PERIOD=STATUS` flags to emit
only paths whose rights decision changes.

Monthly-active-user periods are calendar months in `YYYY-MM` form. Nonnegative
integer counts are evaluated directly against each license threshold;
`at_or_below_threshold` and `above_threshold` remain available as explicit
threshold-only facts. Missing periods and unrecognized facts remain unknown.
Numeric facts in trusted context must carry per-period source evidence. For
example, the Llama 3.2 community license evaluates the preceding month
(`2024-08`) for its September 25, 2024 release date.

`annual_revenue_usd` is an optional exact value in the trusted business context;
when present its `revenue_status` must be `exact_value`. It evaluates each
license's own threshold, including when two models use different thresholds.
Do not add an amount to the canonical context unless the operator has supplied
it; hypothetical CLI values are comparison inputs only.

```sh
go run ./cmd/rights-impact --root /path/to/Vrooli \
  --compare-from-monthly-active-users 2024-08=0 \
  --monthly-active-users 2024-08=700000001 --json

go run ./cmd/rights-impact --root /path/to/Vrooli \
  --compare-from-annual-revenue-usd 0 \
  --annual-revenue-usd 1000000 --json
```

## Consumer classes

`resource_runtime` is the primary consumer. `internal_platform` and scenario
classes are admitted because this package also carries the contract types used
by AI Gateway and scenario clients. It must not import the control plane's
private `internal/` packages.

## Related

- [Package governance](../../docs/package-governance.md)
- [Resource architecture](../../docs/resources/architecture.md)
- [AI Gateway role policy](../../scenarios/ai-gateway/docs/reference/roles-profiles-policies.md)
