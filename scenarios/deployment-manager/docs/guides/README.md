# Guides

Technical deep-dives and how-to documentation for deployment topics.

## Quick Start

| Guide | Status | Time | Description |
|-------|--------|------|-------------|
| [Quick Start](quick-start.md) | Working | 5 min | Deploy your first scenario |

## Core Concepts

| Guide | Status | Description |
|-------|--------|-------------|
| [Bundle Manifest Schema](bundle-manifest-schema.md) | Working | Complete bundle.json reference |
| [Fitness Scoring](fitness-scoring.md) | Working | How scenarios are scored for each tier |
| [Dependency Swapping](dependency-swapping.md) | Working | Replace heavy deps with portable alternatives |
| [Secrets Management](secrets-management.md) | Working | Infrastructure vs service vs user secrets |

## Deployment Preparation

| Guide | Status | Description |
|-------|--------|-------------|
| [Deployment Checklist](deployment-checklist.md) | Working | Pre-deployment readiness check |
| [Packaging Matrix](packaging-matrix.md) | Working | What scenario-to-* can produce today |

## Post-Deployment

| Guide | Status | Description |
|-------|--------|-------------|
| [Auto-Updates](auto-updates.md) | Working | Update channels and provider configuration |

### Status Legend

- **Working** - Fully documented and implemented
- **Partial** - Documentation exists; implementation has gaps
- **Planned** - Documentation placeholder; implementation pending

## By Task

### "I want to..."

| Task | Guide |
|------|-------|
| Get started quickly | [Quick Start](quick-start.md) |
| Understand bundle.json structure | [Bundle Manifest Schema](bundle-manifest-schema.md) |
| Understand why my scenario can't deploy | [Fitness Scoring](fitness-scoring.md) |
| Make my scenario portable | [Dependency Swapping](dependency-swapping.md) |
| Handle secrets properly | [Secrets Management](secrets-management.md) |
| Set up auto-updates | [Auto-Updates](auto-updates.md) |
| Check if I'm ready to deploy | [Deployment Checklist](deployment-checklist.md) |

## For Agents

When helping users:

1. **Start with Quick Start** for first-time deployments
2. **Use Fitness Scoring** to explain why scores are low
3. **Reference Dependency Swapping** when blockers need resolution
4. **Check Secrets Management** for secret-related errors
5. **Consult Auto-Updates** for post-deployment configuration

## Related

- [Deployment Guide](../DEPLOYMENT-GUIDE.md) - Single entry point for deploying scenarios
- [Roadmap](../ROADMAP.md) - Implementation status and planned work
- [CLI Reference](../cli/README.md) - Command documentation
- [API Reference](../api/README.md) - REST API documentation
- [Workflows](../workflows/README.md) - Step-by-step deployment guides
- [Examples](../examples/README.md) - Real-world case studies
