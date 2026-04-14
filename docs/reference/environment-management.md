# Environment Management

This page describes the current project-level environment and secrets posture.

## Current Truth

Environment and secrets handling in Vrooli is a mix of:

- project-level configuration under `.vrooli/`
- resource and scenario manifests
- host-tool and setup requirements declared in project manifests
- deployment-tier-specific secrets behavior described in the Deployment Hub

This is not a stable “one environment model for everything” story yet, so avoid treating older environment docs as universally canonical.

## What To Use Today

For setup and development:

```bash
vrooli setup --help
vrooli develop --help
```

For deployment-tier-specific secrets thinking:

- [../deployment/README.md](../deployment/README.md)

## Guidance

- Treat `.vrooli/service.json` as part of the current project-level operational truth.
- Treat scenario-local `.vrooli/service.json` files as scenario-specific operational truth.
- Avoid documenting environment behavior in old shell-script terms unless that specific path still exists and is intentionally supported.
- Be careful with secrets claims: current behavior varies by setup path and deployment tier.

## Status

This page is intentionally conservative until a fuller environment-and-secrets rewrite happens.
