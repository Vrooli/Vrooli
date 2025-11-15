# Deployment Fitness Scoring

Fitness scores quantify how ready a resource or scenario is for a deployment tier. They combine measured requirements with subjective readiness so deployment-manager can highlight risk.

## Score Definition

- **Range:** 0.0 – 1.0
- **Meaning:**
  - `>= 0.9` → Ship-ready
  - `0.6 – 0.89` → Works with caveats or optional tweaks
  - `< 0.6` → Needs swaps or engineering work

## Inputs

1. **Measured Requirements** — memory, disk, CPU, GPU, network, background services.
2. **Dependency Availability** — does the dependency itself depend on unsupported services?
3. **Offline Capability** — required for desktop/mobile tiers.
4. **Security/Secrets Posture** — can secrets be regenerated per deployment?
5. **Operational Complexity** — how difficult is it to install, monitor, upgrade?

## Schema Extension

```json
{
  "deployment": {
    "platforms": {
      "desktop": {
        "supported": true,
        "fitness_score": 0.6,
        "requirements": {
          "ram_mb": 512,
          "disk_mb": 100,
          "cpu_cores": 1,
          "gpu": false,
          "network": "optional"
        },
        "notes": "Requires 512MB RAM for embeddings"
      }
    },
    "offline_capable": true,
    "complexity": "medium"
  }
}
```

## Calculation Strategy

1. **Baseline Score** — Derived automatically from requirements vs tier budget (e.g., Tier 2 desktop budget = 2GB RAM, 2 cores, 1GB disk). Score = remaining budget ratio.
2. **Penalty Adjustments** — Apply penalties for blockers (no offline mode, needs root privileges, sensitive secrets, etc.).
3. **Manual Override** — Developers can override via `manual_score`/`manual_reason` fields when data is incomplete.
4. **Historical Data** — Later we can feed telemetry (actual runtime stats) to refine scores.

## Usage

- `scenario-dependency-analyzer` aggregates scores across the dependency DAG to produce an overall fitness grade per tier.
- `deployment-manager` shows the grade and indicates which dependencies drag the score down.
- `scenario-to-*` packagers refuse to build when fitness < threshold unless the builder acknowledges the risk.

## Next Steps

- Define tier budgets (RAM/CPU/disk/network) in a shared config so scoring stays consistent.
- Instrument resources to capture runtime metrics for smarter scoring.
- Teach CI to fail when deployment metadata is missing for required tiers.
