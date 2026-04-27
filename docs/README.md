# Vrooli Documentation

This directory is the canonical documentation hub for the Vrooli platform.

Vrooli is a local, cross-platform, Go-native control plane for orchestrating resources, running scenarios, and compounding capabilities through reusable software artifacts. The docs should reflect the platform as it exists now, while still preserving the strategic direction that gives the project its ambition.

## Looking for a description of Vrooli?

If you (or an AI assistant reading this) are trying to understand or describe what Vrooli is, the canonical answers live in [`narrative/`](narrative/):

- [`narrative/PITCH.md`](narrative/PITCH.md) — motto, one-line through 2-minute elevator pitches, audience-tailored leads, key positioning lines, what-Vrooli-is-NOT
- [`narrative/NARRATIVE.md`](narrative/NARRATIVE.md) — the project story at 4 depths (1-line, 1-paragraph, 1-page, deep-vision bracketed for vision-aligned audiences only)
- [`narrative/FAQ.md`](narrative/FAQ.md) — canonical Q&A (what is it, how does it make money, how is it different from other agent frameworks, status, contribution paths)
- [`narrative/PRESS_KIT.md`](narrative/PRESS_KIT.md) — composition skeleton for journalists and external publications
- [`narrative/PITCH_DECK.md`](narrative/PITCH_DECK.md) — slide-by-slide deck outline

For the long-term philosophical thesis (recursive intelligence, evolution timeline, compound-intelligence effect), see [`../VISION.md`](../VISION.md). For the marketing voice canon (positioning principles, voice samples, anti-patterns, dev-log narrative principles), see [`marketing/STRATEGY.md`](marketing/STRATEGY.md).

## Start Here (technical / contributor onboarding)

- [QUICKSTART.md](QUICKSTART.md) for the first-touch setup and command flow
- [concepts/ARCHITECTURE.md](concepts/ARCHITECTURE.md) for the current platform mental model
- [concepts/GLOSSARY.md](concepts/GLOSSARY.md) for shared vocabulary
- [reference/cli-commands.md](reference/cli-commands.md) for the current CLI surface

## Canonical Areas

- [narrative/](narrative/) — project-identity canon (pitch, story, FAQ, press kit, pitch-deck outline) consumed across teams
- [marketing/](marketing/) — voice, audiences, channels, campaigns, brand assets, image-style guide
- [guides/README.md](guides/README.md) for contributor and operator workflows
- [reference/cli-commands.md](reference/cli-commands.md) for CLI and control-plane reference
- [operations/README.md](operations/README.md) for live operational guidance
- [deployment/README.md](deployment/README.md) for deployment tiers and maturity
- [scenarios/README.md](scenarios/README.md) for the scenario ecosystem
- [resources/README.md](resources/README.md) for the resource ecosystem
- [strategy/README.md](strategy/README.md) for project framing, decisions, risks, and roadmap
- [monetization/README.md](monetization/README.md) for the revenue / bundle / SKU canon

## Structure

The project-level docs are organized around a stable taxonomy:

- `QUICKSTART.md` for first-touch onboarding
- `narrative/` for project-identity canon (pitch, story, FAQ, press kit, deck outline) — cross-team consumed
- `marketing/` for voice canon, audiences, channels, campaigns, brand assets, image-style guide
- `concepts/` for architecture and vocabulary
- `guides/` for task-oriented workflows
- `reference/` for CLI, contracts, and shared policy
- `operations/` for live operational guidance
- `deployment/` for tier and target deployment guidance
- `scenarios/` for the scenario system
- `resources/` for the resource system
- `strategy/` for durable project framing and direction
- `monetization/` for revenue / bundle / SKU canon
- `meta-optimization/` for self-improvement framework
- `director-swarm/` for portfolio philosophy / roadmap / outcomes charter
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
