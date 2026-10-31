# Production Operations Guide

This document is an operational reference for running Vrooli in real environments. It should be read alongside the [Deployment Hub](../deployment/README.md), which remains the source of truth for deployment tiers and maturity.

## Current Scope

The current production-ready path is the Tier 1 local stack. That generally means:

- a full Vrooli installation
- scenario and resource lifecycle managed through the current CLI and scenario Makefiles
- app-monitor and related remote-access surfaces where appropriate
- local or operator-controlled infrastructure as the primary deployment model

This file does not treat older multi-node Kubernetes and script-heavy server layouts as canonical production guidance.

## Operating Principles

- prefer the current `vrooli` CLI for inspection, validation, and orchestration
- prefer scenario-local `make start|test|logs|stop` for scenario lifecycle management
- keep resource and scenario manifests accurate so the platform can reason about the installation
- treat deployment docs, manifests, and runtime behavior as one system

## Pre-Production Checklist

- `make setup` completes successfully on the target host
- `vrooli resource status` shows the required local resources in a healthy state
- `vrooli scenario status <name>` works for the scenarios you intend to run
- scenario-local `make test` or `vrooli scenario test <name>` passes for the workloads you are shipping
- backup, logging, and operator access paths are defined before the deployment is treated as durable

## Day-To-Day Operations

Common operational commands:

```bash
vrooli status
vrooli doctor
vrooli resource status
vrooli scenario status <name>
vrooli scenario logs <name>
cd scenarios/<name> && make logs
```

Use the CLI first when diagnosing installation-wide issues. Drop to scenario-local logs when the problem is specific to one scenario.

## Change Management

When promoting changes into a real environment:

- update manifests and docs in the same change when shared behavior changes
- validate scenario behavior before and after deployment
- avoid undocumented direct process management or shell-era lifecycle shortcuts
- record any environment-specific override that future operators must preserve

## Logging And Troubleshooting

Start with:

- [logging.md](logging.md)
- [troubleshooting.md](troubleshooting.md)

Prefer current CLI diagnostics over legacy script procedures.

## Backups And Recovery

Backups should focus on the state that would be expensive to reconstruct:

- persisted resource data
- scenario-specific state where the scenario is stateful
- manifests and configuration that define the installation
- operator-managed secrets and environment configuration

Recovery procedures should be tested against the same Tier 1 assumptions you actually run in production.

## Historical Note

Older documents and plans may reference:

- multi-node Kubernetes topologies
- shell-driven packaging or deployment scripts
- older Node-first application layouts

Treat those as historical or forward-looking reference unless the Deployment Hub explicitly says they are active.
