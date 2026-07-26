# Deployment Tiers

Vrooli supports 5 deployment tiers, each with different characteristics and requirements.

> **Ready to deploy?** Start with the [Deployment Guide](../DEPLOYMENT-GUIDE.md) for a complete walkthrough.

## Tiers And Ramps Are Different Axes

A tier describes where a scenario runs. A ramp is the `scenario-to-*` scenario that packages it for that destination. One tier can have several ramps. Tier strings such as `tier-2-desktop` are contract values passed to bundle export, so the mapping below is normative.

## Tier Overview

| Tier | Name | Description | Ramp | Status |
|------|------|-------------|------|--------|
| 1 | [Local/Dev Stack](tier-1-local-dev.md) | Full Vrooli + Cloudflare tunnel | none; the platform runs the scenario directly | Production Ready |
| 2 | [Desktop](tier-2-desktop.md) | Windows/macOS/Linux apps | `scenario-to-desktop` | Partial |
| 3 | [Mobile](tier-3-mobile.md) | iOS/Android apps | `scenario-to-ios`, `scenario-to-android` | Not Started |
| 4 | [SaaS/Cloud](tier-4-saas.md) | DigitalOcean/AWS/bare metal | `scenario-to-cloud` | Not Started |
| 5 | [Enterprise](tier-5-enterprise.md) | Hardware appliances | none yet | Vision |

## Unassigned Ramps

`scenario-to-extension` and `scenario-to-mcp` package a scenario to run inside a host application rather than on a tier of its own. Their tier assignment is not decided. Do not pass a tier string for them until it is.

### Status Legend

- **Production Ready** - Fully automated, tested end-to-end
- **Partial** - Core functionality works; some manual steps required
- **Not Started** - Documentation placeholder; implementation pending
- **Vision** - Future direction; no implementation yet

See [ROADMAP.md](../ROADMAP.md) for detailed implementation status.

## Tier Characteristics

### Tier 1: Local/Dev Stack

- **Baseline**: All fitness scores measured against this
- **Resources**: Full access to all local resources
- **Secrets**: Infrastructure secrets available
- **Fitness**: 100 (no constraints)

### Tier 2: Desktop

- **Packaging**: Electron wrapper with bundled UI/API/resources
- **Resources**: Must be swapped to portable alternatives
- **Secrets**: No infrastructure secrets; generated + user-prompted only
- **Fitness baseline**: 75 (lightweight deps required)

### Tier 3: Mobile

- **Packaging**: Native iOS/Android apps
- **Resources**: Heavy swaps required (all deps → cloud APIs or embedded)
- **Secrets**: Device-local only
- **Fitness baseline**: 40 (significant architecture changes)

### Tier 4: SaaS/Cloud

- **Packaging**: Container/VM deployment
- **Resources**: Web-native preferred
- **Secrets**: Cloud vault integration
- **Fitness baseline**: 85 (mostly compatible)

### Tier 5: Enterprise

- **Packaging**: Hardware appliance
- **Resources**: Full local stack
- **Secrets**: On-premise vault
- **Fitness baseline**: 60 (compliance friction)

## Fitness Scoring

Each tier has baseline fitness requirements. See [Fitness Scoring Guide](../guides/fitness-scoring.md) for the detailed rubric.

## Choosing a Tier

| If you need... | Choose |
|----------------|--------|
| Development/testing | Tier 1 |
| Offline desktop app | Tier 2 |
| Mobile app | Tier 3 |
| SaaS product | Tier 4 |
| Enterprise appliance | Tier 5 |

## Related

- [Deployment Guide](../DEPLOYMENT-GUIDE.md) - Single entry point for deploying scenarios
- [Workflows](../workflows/README.md) - Step-by-step deployment guides
- [Dependency Swapping](../guides/dependency-swapping.md) - Make scenarios portable
- [Fitness Scoring](../guides/fitness-scoring.md) - Scoring rubric
- [Roadmap](../ROADMAP.md) - Implementation status and planned work
