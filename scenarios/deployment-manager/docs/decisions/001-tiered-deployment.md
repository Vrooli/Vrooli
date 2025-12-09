# ADR-001: Tiered Deployment Model

## Status

Accepted

## Context

Vrooli scenarios need to run on multiple platforms: local development servers, desktop applications, mobile devices, cloud infrastructure, and enterprise hardware appliances. Each platform has fundamentally different constraints:

- **Local dev**: Full access to all resources (Postgres, Redis, Ollama, etc.)
- **Desktop**: Must be self-contained, work offline, fit in reasonable disk/RAM
- **Mobile**: Severe resource constraints, app store requirements
- **Cloud**: Scalability concerns, cost optimization, compliance
- **Enterprise**: Security compliance, air-gapped networks, hardware constraints

Early attempts to "just package" scenarios for desktop failed because:
1. UIs bundled without APIs couldn't function
2. APIs bundled without databases couldn't persist
3. Heavy dependencies (Postgres, Ollama) made bundles impractical

## Decision

Adopt a **5-tier deployment model** where each tier represents a distinct deployment target with its own constraints and scoring system:

| Tier | Name | Description |
|------|------|-------------|
| 1 | Local/Dev | Full Vrooli stack with all resources |
| 2 | Desktop | Windows/macOS/Linux standalone apps |
| 3 | Mobile | iOS/Android native applications |
| 4 | SaaS/Cloud | Cloud-hosted multi-tenant deployments |
| 5 | Enterprise | Hardware appliances with compliance |

Each scenario receives a **fitness score** (0-100) for each tier, based on:
- Resource requirements vs. tier constraints
- Dependency portability
- Licensing compatibility
- Platform support availability

Scenarios with low fitness for a tier receive **swap suggestions** - alternative dependencies that are tier-compatible (e.g., postgres → sqlite for desktop).

## Consequences

### Positive

- **Clear mental model**: Developers understand exactly what "deploy to desktop" means
- **Actionable guidance**: Fitness scores tell developers what to fix before packaging
- **Incremental adoption**: Scenarios can start at Tier 1 and gradually add tier support
- **Automation-ready**: Fitness scoring enables automated CI/CD pipelines to gate releases
- **Shared vocabulary**: All documentation, CLI, and UI use consistent tier terminology

### Negative

- **Complexity**: 5 tiers × many scenarios = large compatibility matrix to maintain
- **Rigidity**: Some deployment targets don't fit cleanly into tiers (e.g., Raspberry Pi)
- **Overhead**: Every scenario needs tier metadata even if only targeting Tier 1

### Neutral

- Tier 1 (local dev) becomes the baseline reference implementation for all scenarios
- Tier numbers are arbitrary but memorable (smaller = more portable)

## Alternatives Considered

### Single "Portable" Flag

A boolean "portable: true/false" per scenario.

**Rejected because**: Too coarse. A scenario might work on desktop but not mobile. Need granular per-platform scoring.

### Platform-Specific Branches

Separate git branches for each platform (main, desktop, mobile, cloud).

**Rejected because**: Maintenance nightmare. Changes would need cherry-picking across branches. Divergence inevitable.

### No Tier Model (Ad-Hoc Packaging)

Let developers manually configure packaging per scenario with no standardization.

**Rejected because**: No consistency. Each scenario would reinvent its deployment strategy. Knowledge wouldn't transfer.

### Three Tiers (Dev/Edge/Cloud)

Simplified model with just development, edge devices, and cloud.

**Rejected because**: Doesn't distinguish desktop (Electron) from mobile (native apps) - very different constraints. Enterprise needs dedicated handling for compliance.

## References

- [Tier Reference Documentation](../tiers/README.md)
- [Fitness Scoring Guide](../guides/fitness-scoring.md)
- [Dependency Swapping Guide](../guides/dependency-swapping.md)
