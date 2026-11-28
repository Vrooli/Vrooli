# Deployment

## Supported Tiers

Knowledge Observatory runs as a local Vrooli scenario. The scenario lifecycle
owns process tracking, assigned ports, logs, and storage namespace resolution.

## Runtime Requirements

Use the scenario lifecycle commands so ports, process tracking, logs, and
health checks remain under Vrooli control.

## Packaging

API, CLI, and UI are packaged as scenario surfaces. Qdrant is the external
required resource; operational metadata uses scenario-scoped SQLite, so no
PostgreSQL service or credentials are required. See the
[integration contract](../concepts/INTEGRATIONS.md).

## Release Checklist

- Run API and CLI tests.
- Run `vrooli scenario test knowledge-observatory`.
- Verify Qdrant connectivity and scenario database startup.
- Verify documentation health for this scenario.

## Rollback

Rollback is handled by reverting the scenario code and restarting through the
scenario lifecycle.
