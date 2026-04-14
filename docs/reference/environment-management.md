# Environment Management

This page describes the current project-level environment and secrets posture.

## Current Truth

Environment and secrets handling in Vrooli is a mix of:

- project-level configuration under `.vrooli/`
- project and scenario manifests
- host-tool and setup requirements declared in manifests and setup flows
- deployment-tier-specific secrets behavior described in the Deployment Hub

This is not a stable “one environment model for everything” story yet, so avoid treating older environment docs as universally canonical.

## What To Use Today

For setup and development:

```bash
vrooli setup --help
vrooli develop --help
vrooli status
vrooli doctor
```

These commands are the best current entrypoints for understanding environment expectations, tool availability, and setup posture.

For deployment-tier-specific secrets thinking:

- [../deployment/README.md](../deployment/README.md)

For scenario-specific runtime expectations:

- the scenario's own `.vrooli/service.json`
- scenario-local docs when the behavior is specific to that scenario

## Guidance

- Treat project configuration under `.vrooli/` as part of the project-level operational truth.
- Treat scenario-local `.vrooli/service.json` files as scenario-specific operational truth.
- Avoid documenting environment behavior in old shell-script terms unless that specific path still exists and is intentionally supported.
- Be careful with secrets claims: current behavior varies by setup path and deployment tier.
- Keep host requirements, setup behavior, and secret handling distinct. They overlap, but they are not one single unified concern.

## Status

This page is intentionally conservative until a fuller environment-and-secrets rewrite happens, but it is still the canonical project-level reference for the current posture.
