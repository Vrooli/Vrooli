# Quality Phase

**ID**: `quality`  
**Timeout**: 120 seconds  
**Required**: Yes  
**Provider**: `quality-health`

The quality phase delegates static quality validation to the Quality Health scenario. It replaces the old native `lint` phase; Test Genie no longer owns component-specific lint/type command heuristics.

## What It Checks

Quality Health audits the target scenario's discovered surfaces and reports findings for:

- strict TypeScript configuration and required guardrail comments
- ESLint safety rules, typed linting setup, and required guardrail comments
- dangerous TypeScript suppression patterns
- Node build scripts that must run type checking before bundling
- strict `.vrooli/testing.json` lint policy contracts
- Go module and golangci-lint configuration
- Makefile quality gates

The phase maps Quality Health findings into Test Genie's `standards` finding channel so ecosystem-manager and maturity guidance can treat static-quality failures as standards work.

## Execution

Test Genie calls the provider through the shared validation RPC:

```text
scenario-validation/v1.ScenarioValidationService.ValidateScenario
```

Quality Health packs its native `AuditQualityResponse` into `native_detail` for its own CLI/UI. Warnings are reported as warnings. Error findings fail the phase. If the Quality Health API is unavailable, the phase fails as a missing dependency because this is now the canonical static-quality producer.

## Troubleshooting

Run the provider directly for full detail:

```bash
quality-health audit run <scenario> --json
quality-health explain finding <finding-id> --scenario <scenario>
```

Use Quality Health's fix preview before applying deterministic config repairs:

```bash
quality-health fix-config run <scenario> --dry-run
quality-health fix-config apply <scenario>
```

Do not weaken lint/type settings to make this phase pass. The guarded config comments are part of the contract and are intentionally checked so agents fix source code instead of loosening validation.
