# Vrooli Documentation

This directory is the canonical documentation hub for the Vrooli platform.

Vrooli is a local, cross-platform, Go-native control plane for orchestrating resources, running scenarios, and compounding capabilities through reusable software artifacts. The docs should reflect the platform as it exists now, while still preserving the strategic direction that gives the project its ambition.

## Start Here

- [QUICKSTART.md](QUICKSTART.md) for the first-touch setup and command flow
- [concepts/ARCHITECTURE.md](concepts/ARCHITECTURE.md) for the current platform mental model
- [concepts/GLOSSARY.md](concepts/GLOSSARY.md) for shared vocabulary
- [reference/cli-commands.md](reference/cli-commands.md) for the current CLI surface

## Strategic Context

These files are the durable project-level memory and strategy layer:

- [context.md](context.md)
- [decisions.md](decisions.md)
- [risks.md](risks.md)
- [roadmap.md](roadmap.md)
- [../VISION.md](../VISION.md)

## System Areas

- [scenarios/README.md](scenarios/README.md) explains the scenario ecosystem
- [resources/README.md](resources/README.md) explains the resource system
- [deployment/README.md](deployment/README.md) explains deployment tiers and current portability reality
- [TESTING.md](TESTING.md) points to the Test Genie documentation surface
- [repo-contract.md](repo-contract.md) defines repo-aware structural rules
- [package-governance.md](package-governance.md) defines shared-package policy

## Structure

This documentation tree is being standardized around a clearer taxonomy:

- `QUICKSTART.md` for first-touch onboarding
- `concepts/` for architecture and vocabulary
- `guides/` for task-oriented workflows
- `reference/` for CLI, config, and contracts
- `internal/` for documentation maintenance state and migration notes
- `plans/` for proposals and implementation plans

Older top-level docs remain in place where needed for compatibility, but the files above are the first-pass canonical entry layer.

## Current Priorities

The current docs rewrite is focused on:

- replacing shell-era and transitional project-level guidance
- aligning docs with the Go-native `vrooli` control plane
- separating current truth from roadmap direction
- reducing duplication between project docs, scenario docs, and resource docs
- preserving the project's ambition without overstating maturity

## Notes For Maintainers

- Prefer updating canonical docs over adding parallel explanations elsewhere.
- If a doc primarily describes one scenario, move or rewrite it under that scenario instead of expanding project-level docs.
- If a doc primarily describes one resource, move or rewrite it under `docs/resources/` or the resource itself.
- If a plan is no longer active, keep it under `plans/` and make sure canonical docs do not present it as current truth.
