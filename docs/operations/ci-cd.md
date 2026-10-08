# CI/CD

This page describes the current CI/CD stance at the project level.

## Scope

Treat this page as guidance for:

- project and scenario validation in CI
- controlled Tier 1 deployment automation
- preserving alignment between CI behavior and current supported workflows

For packaging and portability beyond Tier 1, use [../deployment/README.md](../deployment/README.md).

## Current Truth

The safest current CI posture is:

- validate the project control plane
- validate repo contract and package governance
- run scenario-aware tests deliberately
- avoid reviving legacy package-and-ship assumptions

## Recommended CI Baseline

For project-level CI, start with:

```bash
make test
```

Useful focused targets:

```bash
make hygiene
make validate-package-governance
```

For scenario-focused validation:

```bash
vrooli scenario test <name>
```

## Guidance

- Prefer explicit scenario selection over “test everything blindly” unless the workflow is designed for it.
- Treat Tier 1 deployment as the current mature deployment target.
- Do not assume Kubernetes, desktop, or SaaS packaging flows are canonical unless the Deployment Hub says so.

## Legacy Material

Older CI/CD examples in the repository may still be useful as reference, but they should not be treated as canonical if they:

- assume old package-and-ship deployment outputs
- use stale command paths
- imply equal maturity across deployment tiers

Cross-check deployment automation against:

- [../deployment/README.md](../deployment/README.md)
- [../reference/cli-commands.md](../reference/cli-commands.md)
