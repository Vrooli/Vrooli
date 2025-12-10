# Deployment Workflows

Step-by-step guides for deploying Vrooli scenarios to different platform tiers.

> **New to deployment?** Start with the [Deployment Guide](../DEPLOYMENT-GUIDE.md) for a streamlined walkthrough.

## Purpose

The reference documentation in this hub explains *what* each component does. These workflow guides explain *how* to accomplish specific deployment goals by orchestrating those components in the correct sequence.

## Available Workflows

| Workflow | Tier | Status | Time Estimate | Guide |
|----------|------|--------|---------------|-------|
| Desktop Deployment | 2 | Partial | 30-60 min | [desktop-deployment.md](desktop-deployment.md) |
| Mobile Deployment | 3 | Not Started | TBD | [mobile-deployment.md](mobile-deployment.md) |
| SaaS/Cloud Deployment | 4 | Not Started | TBD | [saas-deployment.md](saas-deployment.md) |

### Status Legend

- **Production Ready** - Fully automated, tested end-to-end
- **Partial** - Core functionality works; some manual steps required
- **Not Started** - Documentation placeholder; implementation pending

### Current Gaps (Desktop)

The desktop workflow is **Partial** because:

| Gap | Workaround | Tracking |
|-----|------------|----------|
| ~~No CLI `bundle export` command~~ | ~~Use REST API directly~~ **RESOLVED** - CLI commands available | - |
| No automated binary compilation | Manual cross-compile per platform | [ROADMAP.md](../ROADMAP.md) |
| No asset procurement automation | Manual download/placement | [ROADMAP.md](../ROADMAP.md) |
| Code signing not wired | Manual certificate config | [ROADMAP.md](../ROADMAP.md) |

See [ROADMAP.md](../ROADMAP.md) for full implementation status.

## Workflow Structure

Each workflow follows a consistent 8-phase structure:

| Phase | Purpose | Key Tools |
|-------|---------|-----------|
| 1. Prerequisites | Verify environment and dependencies | `vrooli`, `deployment-manager status` |
| 2. Compatibility | Check scenario fitness for target tier | `deployment-manager fitness` |
| 3. Issues | Address blockers via dependency swaps | `deployment-manager swaps` |
| 4. Configuration | Create and configure deployment profile | `deployment-manager profile` |
| 5. Secrets | Configure tier-appropriate secret strategy | `deployment-manager secrets` |
| 6. Build | Generate deployment artifacts | `scenario-to-*` packagers |
| 7. Distribution | Publish and deliver to users | Platform-specific |
| 8. Monitoring | Collect analytics and handle updates | `deployment-manager logs`, telemetry |

## Quick Navigation

### By Task

- "I want to create a desktop app" → [Desktop Deployment](desktop-deployment.md)
- "I want to deploy to iOS/Android" → [Mobile Deployment](mobile-deployment.md)
- "I want to deploy to the cloud" → [SaaS Deployment](saas-deployment.md)
- "Something isn't working" → [Troubleshooting](troubleshooting.md)

### By Component

- **deployment-manager CLI** → [CLI Reference](../../scenarios/deployment-manager/README.md#cli-cheat-sheet-agent-friendly)
- **Fitness Scoring** → [Fitness Scoring Guide](../guides/fitness-scoring.md)
- **Dependency Swapping** → [Dependency Swapping Guide](../guides/dependency-swapping.md)
- **Secrets Management** → [Secrets Management Guide](../guides/secrets-management.md)
- **Auto-Updates** → [Auto-Updates Guide](../guides/auto-updates.md)
- **Bundle Manifests** → [Example Manifests](../examples/manifests/)

### By Tier

- **Tier 1 (Local)** → [Local Dev Stack](../tiers/tier-1-local-dev.md) (baseline, no workflow needed)
- **Tier 2 (Desktop)** → [Tier 2 Reference](../tiers/tier-2-desktop.md) | [Workflow](desktop-deployment.md)
- **Tier 3 (Mobile)** → [Tier 3 Reference](../tiers/tier-3-mobile.md) | [Workflow](mobile-deployment.md)
- **Tier 4 (SaaS)** → [Tier 4 Reference](../tiers/tier-4-saas.md) | [Workflow](saas-deployment.md)
- **Tier 5 (Enterprise)** → [Tier 5 Reference](../tiers/tier-5-enterprise.md) (requires Tier 4 first)

## For Agents

When helping users deploy scenarios:

1. **Start with the workflow** - These guides provide the exact command sequence
2. **Reference the guides** - For deeper explanation of any step
3. **Check troubleshooting** - For common errors and solutions
4. **Note the gaps** - Each workflow documents what's implemented vs. planned

### Key Files to Know

```
docs/deployment/
├── workflows/           # YOU ARE HERE - Task-oriented guides
│   ├── README.md        # This index
│   ├── desktop-deployment.md
│   ├── mobile-deployment.md
│   ├── saas-deployment.md
│   └── troubleshooting.md
├── tiers/               # Reference docs per deployment tier
├── guides/              # Deep-dive technical guides
├── scenarios/           # Per-scenario deployment notes
├── examples/            # Real-world case studies and sample manifests
└── README.md            # Deployment Hub (start here for overview)
```

## Contributing

When adding or updating workflows:

1. Follow the 8-phase structure for consistency
2. Include exact commands with expected output
3. Mark implementation status clearly (Working/Partial/Planned)
4. Link to reference docs rather than duplicating content
5. Document known gaps with links to tracking issues
6. Test the workflow end-to-end before marking "Production Ready"

## Related Documentation

- [Deployment Guide](../DEPLOYMENT-GUIDE.md) - Single entry point for deploying scenarios
- [Deployment Hub](../README.md) - Overview of the tiered deployment model
- [Roadmap](../ROADMAP.md) - Implementation status and planned work
- [Bundled Runtime Plan](/docs/plans/bundled-desktop-runtime-plan.md) - Technical plan for offline bundles
- [deployment-manager Scenario](../scenarios/deployment-manager.md) - Orchestration scenario details
