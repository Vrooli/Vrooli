# DevOps Documentation

This section documents project-level development, operations, and deployment workflows.

The canonical project-level control surface is the Go-native `vrooli` CLI. DevOps docs should reinforce that model, not regress to older script-first or package-and-ship assumptions.

## Start Here

- [../QUICKSTART.md](../QUICKSTART.md) for first-touch platform setup
- [../reference/cli-commands.md](../reference/cli-commands.md) for current CLI reference
- [../deployment/README.md](../deployment/README.md) for deployment tiers and maturity

## Current DevOps Truth

At the project level:

- `make setup` and `vrooli setup` are the primary setup entrypoints
- `vrooli develop` is the project-level development workflow
- individual scenarios should generally be operated through their local `make start|test|logs|stop` surfaces
- Tier 1 local/dev-stack deployment is the current mature operational story

## Recommended Workflow

### Project-Level

```bash
make setup
vrooli develop
vrooli status
```

### Scenario-Level

```bash
cd scenarios/<scenario-name>
make start
make test
make logs
make stop
```

### Validation

```bash
make test
make validate-repo-contract
make validate-package-governance
```

## What This Section Covers

- development environment setup and troubleshooting
- logging and operations notes
- CI/CD and server-deployment reference material
- historical and transitional operational material that still matters for research or migration

## What To Treat Carefully

Some files in this directory still contain mixed-current or historical content. Until they are rewritten, do not assume every page here is equally canonical.

In particular:

- pages that describe old package-and-ship deployment flows should be treated as reference only
- pages that depend on old target matrices, shell-first automation, or outdated environment assumptions should be treated as transitional
- deployment decisions should be cross-checked against the Deployment Hub

## Useful Pages

- [development-environment.md](development-environment.md)
- [troubleshooting.md](troubleshooting.md)
- [logging.md](logging.md)
- [ci-cd.md](ci-cd.md)
- [server-deployment.md](server-deployment.md)

## Documentation Intent

This directory should eventually converge on the same taxonomy used elsewhere:

- quickstart
- guides
- reference
- internal notes

That reorganization is still pending. For now, use this README as the canonical entrypoint and treat deeper pages with maturity awareness.
