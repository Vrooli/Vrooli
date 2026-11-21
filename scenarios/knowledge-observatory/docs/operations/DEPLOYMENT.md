# Deployment

## Supported Tiers

Knowledge Observatory is currently designed for the Tier 1 local Vrooli stack.

## Runtime Requirements

Use the scenario lifecycle commands so ports, process tracking, logs, and
health checks remain under Vrooli control.

## Packaging

API, CLI, and UI are packaged as scenario surfaces. Required resources are
declared through scenario and project configuration.

## Release Checklist

- Run API and CLI tests.
- Run `vrooli scenario test knowledge-observatory`.
- Verify Qdrant and PostgreSQL connectivity.
- Verify documentation health for this scenario.

## Rollback

Rollback is handled by reverting the scenario code and restarting through the
scenario lifecycle.
