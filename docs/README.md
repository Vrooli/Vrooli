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
- [concepts/RECURSIVE_SELF_IMPROVEMENT.md](concepts/RECURSIVE_SELF_IMPROVEMENT.md) for the self-improvement loop and the four projections (Answer/Validate/Guide/Act)
- [concepts/GLOSSARY.md](concepts/GLOSSARY.md) for shared vocabulary
- [reference/cli-commands.md](reference/cli-commands.md) for the current CLI surface
- [reference/health-maturity-assessments.md](reference/health-maturity-assessments.md) for provider-owned health maturity reports and the human-output-first contract
- [reference/machine-readable-references.md](reference/machine-readable-references.md) for `[CODE:]` / `[DOC:]` traceability references and marked inline references such as `path:...` and `topic:...`

## How docs/ is organized: three pillars

Most folders here are one of three canon types. Knowing which you're in tells you who may edit it and how:

1. **Identity & concept canon** — *what Vrooli is and how its pieces fit.* [`../VISION.md`](../VISION.md) (the why), [`narrative/`](narrative/) (the story), and [`concepts/`](concepts/): [`ARCHITECTURE.md`](concepts/ARCHITECTURE.md) (technical how), [`RECURSIVE_SELF_IMPROVEMENT.md`](concepts/RECURSIVE_SELF_IMPROVEMENT.md) (the self-improvement loop that ties the why and how together), [`ECOSYSTEM.md`](concepts/ECOSYSTEM.md) (how a scenario fits the whole), [`MEASURES.md`](concepts/MEASURES.md) (the federated metrics layer), [`PAID_FEATURES.md`](concepts/PAID_FEATURES.md) (paid-feature contract), [`PUBLIC_ASSETS.md`](concepts/PUBLIC_ASSETS.md) (the `/public/*` world-readable-asset convention + its security contract), [`GLOSSARY.md`](concepts/GLOSSARY.md). Operator-curated.

2. **Team plan-of-records (PoR)** — *each agent team's durable, accepted truth.* One folder per team, all following the shared contract in [`agent-system/team-plan-of-record.manifest.json`](agent-system/team-plan-of-record.manifest.json): a README hub → `operating/` → optional `strategy/` `evidence/` `catalogs/` `taxonomies/` → `governance/`. **Agents never edit PoR canon directly** — changes flow through operator-approved decisions; see [`agent-system/TEAM_DOCS_PATTERNS.md`](agent-system/TEAM_DOCS_PATTERNS.md) for the write boundary. The teams with a PoR (authoritative set: the `docs/<team>/` folders carrying a `manifest.json`):
   - [`monetization/`](monetization/README.md) — revenue, SKUs, delivery tiers, pricing, funnel
   - [`marketing/`](marketing/) — voice, audiences, channels, campaigns, brand assets
   - [`director-swarm/`](director-swarm/) — portfolio philosophy, roadmap, outcomes charter
   - [`infra-health/`](infra-health/) — platform reliability, instrumentation, portability
   - [`meta-optimization/`](meta-optimization/) — friction-report taxonomy + self-improvement
   - [`scenario-qa/`](scenario-qa/) — bug taxonomy, investigation + audit methods

3. **Agent-system framework canon** — *the rules the teams themselves run on* (skills, agents, teams, decisions, topics, and the PoR contract above). Lives in [`agent-system/`](agent-system/README.md); it is itself a plan-of-record, edited via `meta-optimization` decisions.

Everything else (`guides/`, `reference/`, `operations/`, `deployment/`, `scenarios/`, `resources/`, `strategy/`, `design/`, `skills/`, `development/`, `internal/`, `plans/`) is supporting documentation, not canon in the above sense.

## Canonical Areas

- [narrative/](narrative/) — project-identity canon (pitch, story, FAQ, press kit, pitch-deck outline) consumed across teams
- [marketing/](marketing/) — voice, audiences, channels, campaigns, brand assets, image-style guide
- [design/](design/) — canonical `DESIGN.md` governance for scenario UI design languages and generation adapters
- [guides/README.md](guides/README.md) for contributor and operator workflows
- [reference/cli-commands.md](reference/cli-commands.md) for CLI and control-plane reference
- [reference/health-maturity-assessments.md](reference/health-maturity-assessments.md) for health-provider maturity assessment ownership and JSON automation rules
- [reference/machine-readable-references.md](reference/machine-readable-references.md) for machine-readable reference syntax used by docs scanners and agent instructions
- [operations/README.md](operations/README.md) for live operational guidance
- [deployment/README.md](deployment/README.md) for deployment tiers and maturity
- [scenarios/README.md](scenarios/README.md) for the scenario ecosystem
- [resources/README.md](resources/README.md) for the resource ecosystem
- [strategy/README.md](strategy/README.md) for project framing, decisions, risks, and roadmap
- [monetization/README.md](monetization/README.md) for the revenue / bundle / SKU canon
- [skills/](skills/) for the external Claude Skills publishing pipeline (publishing guide, security baseline). The publication source itself lives at the repo top-level `skills/` folder; this `docs/skills/` directory holds the **how-to-publish** docs only.

## Structure

The project-level docs are organized around a stable taxonomy:

- `QUICKSTART.md` for first-touch onboarding
- `narrative/` for project-identity canon (pitch, story, FAQ, press kit, deck outline) — cross-team consumed
- `marketing/` for voice canon, audiences, channels, campaigns, brand assets, image-style guide
- `design/` for application UI design-language governance, design-kit registry, adapters, and `DESIGN.md` rules
- `concepts/` for architecture and vocabulary
- `development/` for development pipelines (e.g. [`development/proto.md`](development/proto.md) — proto codegen)
- `guides/` for task-oriented workflows
- `reference/` for CLI, contracts, and shared policy
- `operations/` for live operational guidance
- `deployment/` for tier and target deployment guidance
- `scenarios/` for the scenario system
- `resources/` for the resource system
- `strategy/` for durable project framing and direction
- `monetization/` for revenue / bundle / SKU canon
- `skills/` for external-skills publishing pipeline (security baseline, publishing guide); the publication source itself lives at `<repo>/skills/`
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
