# Vrooli Documentation

This directory is the canonical documentation hub for the Vrooli platform.

Vrooli is a local, cross-platform, Go-native control plane for orchestrating resources, running scenarios, and compounding capabilities through reusable software artifacts. The docs should reflect the platform as it exists now, while still preserving the strategic direction that gives the project its ambition.

## Start Here

- [QUICKSTART.md](QUICKSTART.md) for the first-touch setup and command flow
- [concepts/ARCHITECTURE.md](concepts/ARCHITECTURE.md) for the current platform mental model
- [concepts/GLOSSARY.md](concepts/GLOSSARY.md) for shared vocabulary
- [reference/cli-commands.md](reference/cli-commands.md) for the current CLI surface

## Canonical Areas

- [guides/README.md](guides/README.md) for contributor and operator workflows
- [reference/cli-commands.md](reference/cli-commands.md) for CLI and control-plane reference
- [operations/README.md](operations/README.md) for live operational guidance
- [deployment/README.md](deployment/README.md) for deployment tiers and maturity
- [scenarios/README.md](scenarios/README.md) for the scenario ecosystem
- [resources/README.md](resources/README.md) for the resource ecosystem
- [strategy/README.md](strategy/README.md) for project framing, decisions, risks, and roadmap

## Structure

The project-level docs are organized around a stable taxonomy:

- `QUICKSTART.md` for first-touch onboarding
- `concepts/` for architecture and vocabulary
- `guides/` for task-oriented workflows
- `reference/` for CLI, contracts, and shared policy
- `operations/` for live operational guidance
- `deployment/` for tier and target deployment guidance
- `scenarios/` for the scenario system
- `resources/` for the resource system
- `strategy/` for durable project framing and direction
- `internal/` for docs-maintenance notes
- `plans/` for proposals and implementation plans

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
- If a doc is historical but still useful, fold the important parts into a maintained canonical doc or keep it only when it still has real maintenance value.
- If a plan is no longer active, keep it under `plans/` and make sure canonical docs do not present it as current truth.
