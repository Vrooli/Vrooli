# Configuration

## Environment variables

| Variable | Purpose | Required |
|---|---|---|
| `API_PORT` | Lifecycle-assigned API port. | yes |
| `UI_PORT` | Lifecycle-assigned UI port. | yes |

## Service manifest (`.vrooli/service.json`)

The service manifest declares lifecycle, CLI, ports, dependencies, and test steps. Quality Health v1 declares no external resources:

```json
"dependencies": {
  "resources": [],
  "scenarios": []
}
```

Runtime scenario dependencies such as Code Facts are documented in [INTEGRATIONS.md](../concepts/INTEGRATIONS.md) and should become explicit manifest dependencies only when the lifecycle contract supports that mode cleanly.

## Schema bootstrap

No product persistence is required for live audits. If run history is implemented, add SQLite schemas under the owning API domain and document them in [DATA.md](../concepts/DATA.md).

## CLI config file

The future CLI should use cli-core conventions for API-base resolution and JSON/human output. Do not add command-specific config files unless a persistent operator preference is required.

## API-base resolution precedence

Use the generated cli-core scenario conventions:

1. explicit command flag,
2. environment variable if supported by cli-core,
3. lifecycle-discovered API base,
4. default local API base.

## Test/CI configuration

The scenario's `.vrooli/testing.json` must stay strict enough for Quality Health's own contracts. It is also a target of the `TESTING_CONFIG_LINT_STRICT` rule.

## Cross-references

- `.vrooli/service.json`
- [INTEGRATIONS.md](../concepts/INTEGRATIONS.md)
- [DATA.md](../concepts/DATA.md)
- [Quality Contracts](quality-contracts.md)
