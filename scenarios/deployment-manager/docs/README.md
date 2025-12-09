# Deployment Documentation

> **This is the authoritative source for all Vrooli deployment documentation.**
> Documentation lives close to its relevant code - deployment-manager owns all deployment workflows.

## How to Use This Documentation

| If you are... | Start here |
|---------------|-----------|
| **Deploying for the first time** | [Deployment Guide](DEPLOYMENT-GUIDE.md) - Complete 8-phase walkthrough |
| **Looking for a specific command** | [CLI Reference](cli/README.md) - All commands with examples |
| **Building an integration** | [API Reference](api/README.md) - REST endpoint documentation |
| **Troubleshooting an issue** | [Troubleshooting](workflows/troubleshooting.md) - Common problems and solutions |
| **An agent helping a user** | [For Agents](#for-agents) section below |
| **Need a quick refresher** | [Cheatsheet](CHEATSHEET.md) - Copy-paste command reference |

**Documentation is organized by intent:**
- `cli/` and `api/` - **Reference** (lookup specific commands/endpoints)
- `workflows/` - **Procedures** (step-by-step, follow along)
- `guides/` - **Concepts** (understand how things work)
- `examples/` - **Patterns** (copy and adapt for your scenario)
- `tiers/` - **Platform details** (when targeting a specific tier)
- `architecture/` - **System design** (how scenarios interact)
- `decisions/` - **ADRs** (understand why things are designed this way)

---

## Quick Navigation

| I want to... | Status | Go to |
|--------------|--------|-------|
| **Deploy a scenario (start here)** | - | [Deployment Guide](DEPLOYMENT-GUIDE.md) |
| Get started in 5 minutes | - | [Quick Start](guides/quick-start.md) |
| Copy-paste common commands | - | [Cheatsheet](CHEATSHEET.md) |
| Learn CLI commands | Working | [CLI Reference](cli/README.md) |
| Call REST APIs directly | Working | [API Reference](api/README.md) |
| Deploy to desktop (Tier 2) | Partial | [Desktop Workflow](workflows/desktop-deployment.md) |
| Deploy to mobile (Tier 3) | Not Started | [Mobile Workflow](workflows/mobile-deployment.md) |
| Deploy to cloud (Tier 4) | Not Started | [SaaS Workflow](workflows/saas-deployment.md) |
| Understand deployment tiers | - | [Tier Overview](tiers/README.md) |
| Understand bundle manifests | - | [Bundle Manifest Schema](guides/bundle-manifest-schema.md) |
| See how scenarios interact | - | [Bundle Pipeline Architecture](architecture/bundle-pipeline.md) |
| Check implementation progress | - | [Roadmap](ROADMAP.md) |
| Understand design decisions | - | [Architecture Decisions](decisions/README.md) |
| Fix an error | - | [Troubleshooting](workflows/troubleshooting.md) |

---

## Why a Tiered Model?

We validated during `scenario-to-desktop` that "build an Electron app" is useless unless the UI, API, resources, and CLI dependencies all come along for the ride. That failure exposed three immovable facts:

1. **The current local stack *is* a deployment tier.** Cloudflare tunnels + app-monitor already let us access every running scenario from anywhere.
2. **Portability demands intelligence.** We have to understand dependency graphs, fitness for each platform, and offer swap suggestions before packaging.
3. **Secrets behave differently per tier.** Infrastructure credentials cannot leave the mothership, whereas per-install service secrets must be generated anew.

## Deployment Tiers

| Tier | Description | Status | Doc |
|------|-------------|--------|-----|
| 1 | Full Vrooli stack + Cloudflare tunnel | Production Ready | [Local/Dev Stack](tiers/tier-1-local-dev.md) |
| 2 | Desktop bundles (Windows/macOS/Linux) | Partial | [Desktop](tiers/tier-2-desktop.md) |
| 3 | Mobile packages (iOS/Android) | Not Started | [Mobile](tiers/tier-3-mobile.md) |
| 4 | SaaS/Cloud installs | Not Started | [SaaS/Cloud](tiers/tier-4-saas.md) |
| 5 | Enterprise/Hardware appliances | Vision | [Enterprise](tiers/tier-5-enterprise.md) |

### Status Legend

- **Production Ready** - Fully automated, tested end-to-end
- **Partial** - Core functionality works; some manual steps required
- **Not Started** - Documentation placeholder; implementation pending
- **Vision** - Future direction; no implementation yet

## CLI Cheat Sheet

```bash
# Health & Analysis
deployment-manager status                              # Check API health
deployment-manager analyze <scenario>                  # Dependency DAG
deployment-manager fitness <scenario> --tier 2        # Desktop fitness score

# Profiles
deployment-manager profiles                            # List all profiles
deployment-manager profile create <name> <scenario> --tier 2
deployment-manager profile show <id>
deployment-manager profile swap <id> add postgres sqlite
deployment-manager profile export <id> --output ./profile.json

# Swaps
deployment-manager swaps list <scenario>               # Available swaps
deployment-manager swaps analyze postgres sqlite       # Impact analysis
deployment-manager swaps apply <profile> postgres sqlite

# Secrets
deployment-manager secrets identify <profile-id>      # Required secrets
deployment-manager secrets template <profile-id> --format env

# Deployment
deployment-manager validate <profile-id> --verbose    # Pre-flight checks
deployment-manager package <profile-id> --packager scenario-to-desktop
deployment-manager logs <profile-id> --level error    # View telemetry
```

See [CLI Reference](cli/README.md) for complete command documentation.

## Documentation Structure

```
docs/
├── README.md              # YOU ARE HERE - Main hub
├── DEPLOYMENT-GUIDE.md    # Single entry point for deploying scenarios
├── CHEATSHEET.md          # Quick command reference (copy-paste friendly)
├── ROADMAP.md             # Implementation status and planned work
├── PROGRESS.md            # Implementation progress (standardized)
├── PROBLEMS.md            # Known issues (standardized)
├── RESEARCH.md            # Research notes (standardized)
├── SEAMS.md               # Integration points (standardized)
│
├── cli/                   # CLI command reference
│   ├── README.md          # CLI overview & global options
│   ├── overview-commands.md
│   ├── profile-commands.md
│   ├── swap-commands.md
│   ├── secret-commands.md
│   └── deployment-commands.md
│
├── api/                   # REST API reference
│   ├── README.md          # API overview & authentication
│   ├── bundles.md
│   ├── profiles.md
│   ├── fitness.md
│   ├── swaps.md
│   ├── deployments.md
│   └── telemetry.md
│
├── tiers/                 # Deployment tier reference
│   ├── README.md          # Tier overview
│   ├── tier-1-local-dev.md
│   ├── tier-2-desktop.md
│   ├── tier-3-mobile.md
│   ├── tier-4-saas.md
│   └── tier-5-enterprise.md
│
├── workflows/             # Step-by-step deployment guides
│   ├── README.md          # Workflow index
│   ├── desktop-deployment.md
│   ├── mobile-deployment.md
│   ├── saas-deployment.md
│   └── troubleshooting.md
│
├── guides/                # Technical deep-dives
│   ├── README.md          # Guide index
│   ├── quick-start.md     # 5-minute onboarding
│   ├── bundle-manifest-schema.md  # Bundle.json reference
│   ├── dependency-swapping.md
│   ├── fitness-scoring.md
│   ├── secrets-management.md
│   ├── deployment-checklist.md
│   ├── packaging-matrix.md
│   ├── auto-updates.md
│   └── version-matrix.md  # Version compatibility reference
│
├── architecture/          # System design documentation
│   ├── README.md          # Architecture index
│   └── bundle-pipeline.md # Multi-scenario orchestration
│
├── examples/              # Real-world case studies
│   ├── README.md
│   ├── picker-wheel-desktop.md
│   ├── picker-wheel-cloud.md
│   ├── system-monitor-desktop.md
│   └── manifests/         # Sample bundle.json files
│       ├── desktop-happy.json
│       └── desktop-playwright.json
│
├── decisions/             # Architecture Decision Records (ADRs)
│   ├── README.md          # Decision index
│   ├── 001-tiered-deployment.md
│   ├── 002-manifest-driven-bundles.md
│   ├── 003-runtime-supervisor.md
│   └── 004-secrets-classification.md
│
├── providers/             # Infrastructure-specific guides
│   ├── README.md
│   ├── digitalocean.md
│   ├── cloudflare-tunnel.md
│   └── hardware-appliance.md
│
├── scenarios/             # Per-scenario deployment notes
│   └── (scenario-specific docs)
│
└── history/               # Legacy documentation (archived)
```

## The Deployment Orchestration Loop

```
                    ┌─────────────────────────────────────┐
                    │         deployment-manager          │
                    │    (orchestrates full workflow)     │
                    └──────────────┬──────────────────────┘
                                   │
         ┌─────────────────────────┼─────────────────────────┐
         │                         │                         │
         ▼                         ▼                         ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ scenario-       │    │ secrets-manager │    │ scenario-to-*   │
│ dependency-     │    │ (classify +     │    │ (package for    │
│ analyzer        │    │  generate)      │    │  target tier)   │
│ (DAG + fitness) │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

1. **deployment-manager** drives the workflow
2. Queries **scenario-dependency-analyzer** for the full dependency DAG
3. Scores fitness for the requested tier, highlighting blockers and suggesting swaps
4. Coordinates with **secrets-manager** to classify/create secrets per tier
5. Triggers the appropriate **scenario-to-*** packager (desktop/mobile/cloud)
6. When manual work is required (e.g., swapping Postgres -> SQLite), files **app-issue-tracker** tasks

## Bundled Runtime Expectations

For Desktop/Mobile/Cloud bundles:

- **Manifest-driven**: Bundles consume `bundle.json` from deployment-manager
- **Runtime executable**: Cross-platform binary supervises processes, allocates ports, exposes control API
- **Dependency swapping**: Mandatory (postgres->sqlite, ollama->packaged models, redis->in-process)
- **Self-contained**: Ship binaries per OS/arch, resource files, migrations, data templates
- **Secrets hygiene**: No infrastructure secrets; user-prompted and local-generated only
- **Upgrade safety**: Versioned data dirs, idempotent migrations

See [Bundle Manifest Schema](examples/manifests/) for the `bundle.json` contract.

## For Agents

When helping users deploy scenarios:

1. **Start with [DEPLOYMENT-GUIDE.md](DEPLOYMENT-GUIDE.md)** - The single entry point with complete workflow
2. **Check [ROADMAP.md](ROADMAP.md)** - Know what works vs. what needs workarounds
3. **Use workflows** - [workflows/](workflows/) provides detailed step-by-step guides
4. **Reference the CLI** - [cli/](cli/) documents every command with examples
5. **Check API docs** - [api/](api/) for direct REST calls when CLI doesn't suffice
6. **Note the gaps** - Each workflow documents what's implemented vs. planned

### Key Files to Know

| Purpose | File |
|---------|------|
| CLI entry point | `scenarios/deployment-manager/cli/app.go` |
| API routes | `scenarios/deployment-manager/api/server/routes.go` |
| Bundle handler | `scenarios/deployment-manager/api/bundles/handler.go` |
| Manifest schema | `docs/examples/manifests/desktop-happy.json` |
| Runtime supervisor | `scenarios/scenario-to-desktop/runtime/supervisor.go` |
| Bundle packager | `scenarios/scenario-to-desktop/api/bundle_packager.go` |

## Related Resources

- [Bundled Desktop Runtime Plan](/docs/plans/bundled-desktop-runtime-plan.md) - Technical implementation plan
- [scenario-to-desktop](/scenarios/scenario-to-desktop/README.md) - Desktop packager
- [scenario-dependency-analyzer](/scenarios/scenario-dependency-analyzer/README.md) - Dependency analysis
- [secrets-manager](/scenarios/secrets-manager/README.md) - Secret classification

## Roadmap

1. Document current truth (this hub + spokes)
2. Extend `service.json` with `deployment.platforms` metadata
3. Upgrade `scenario-dependency-analyzer` to compute resource tallies and cascade fitness scores
4. Build the `deployment-manager` UI (dependency visualization, swap tool, secret prep)
5. Teach `scenario-to-desktop/mobile/cloud` to read deployment bundles
6. Close the loop with app-issue-tracker automation for required swaps/migrations
