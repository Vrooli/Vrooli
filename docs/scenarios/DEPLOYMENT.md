# Direct Scenario Deployment (Tier 1)

> 📚 **[Back to Scenario Documentation](README.md)** · **[Deployment Hub](../deployment/README.md)**

This guide now focuses solely on Tier 1 — running scenarios directly from source on a full Vrooli stack. For any other deployment target (desktop, mobile, SaaS, enterprise), see the [deployment hub](../deployment/README.md).

## Why Tier 1 Matters

- It is the truth we have today: scenarios run from the `scenarios/` directory with shared resources.
- Every other deployment tier must behave like Tier 1 once bundled.
- Tier 1 environments (local machine, dev VPS) are also how we expose scenarios via app-monitor + Cloudflare tunnels.

## Running Scenarios

### Using the CLI

```bash
# List available scenarios
vrooli scenario list

# Run a scenario
vrooli scenario run picker-wheel

# Test a scenario
vrooli scenario test picker-wheel
```

### Using Scenario Makefiles

```bash
cd scenarios/picker-wheel
make start   # starts develop lifecycle (serves production bundles)
make stop
make logs
```

## Process Isolation

- **Process Home**: `~/.vrooli/processes/scenarios/<scenario>/`
- **Logs**: `~/.vrooli/logs/scenarios/<scenario>/`
- **Port Registry**: Vrooli assigns ports automatically; see `~/.vrooli/port-registry.json`.

## Resource Sharing

Scenarios reuse shared resources defined in `.vrooli/service.json` (Ollama, Postgres, Redis, Qdrant, etc.). Tier 1 assumes those resources run locally via `vrooli resource start-all` or the Vrooli lifecycle manager.

## Environment Variables

Automatically set when running within the scenario lifecycle:

- `SCENARIO_NAME`
- `SCENARIO_MODE=true`
- `SCENARIO_PATH`
- `PM_HOME`
- `PM_LOG_DIR`

## Preparing for Other Tiers

While operating in Tier 1, capture deployment metadata to help future tiers:

1. **Dependency Listings** — Keep `service.json` up to date via `scenario-dependency-analyzer`.
2. **Deployment Metadata** — Add `deployment.platforms` entries with fitness scores and requirements.
3. **Secrets Strategy** — Classify secrets using the schema outlined in [Secrets Management](../deployment/guides/secrets-management.md).

## Troubleshooting Tier 1

- Scenario won't start? Ensure service.json exists and run `vrooli scenario run <name> --dry-run`.
- Port conflicts? Inspect/remove `~/.vrooli/port-registry.json` (only if no scenarios are running).
- Resource unavailable? Run `resource-<name> start` or `vrooli resource start-all`.

## What's Next?

- For Tier 2+ deployments (desktop, mobile, SaaS, enterprise), follow the [hub-and-spokes documentation](../deployment/README.md).
- When building packagers (`scenario-to-*`), ensure they recreate Tier 1 behavior inside the bundle.
