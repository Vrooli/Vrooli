# Overview Commands

Commands for health checks, dependency analysis, and fitness scoring.

## status

Check the health of the deployment-manager API and its dependencies.

```bash
deployment-manager status
```

**Output:**

```json
{
  "status": "healthy",
  "api_version": "1.1.0",
  "dependencies": {
    "scenario-dependency-analyzer": "healthy",
    "secrets-manager": "healthy",
    "postgres": "healthy"
  }
}
```

**Exit Codes:**
- `0` - All systems healthy
- `3` - API connection failed
- `1` - One or more dependencies unhealthy

---

## analyze

Analyze a scenario's dependencies and compute aggregate requirements.

```bash
deployment-manager analyze <scenario-name>
```

**Arguments:**
- `<scenario-name>` - Name of the scenario to analyze

**Example:**

```bash
deployment-manager analyze picker-wheel
```

**Output:**

```json
{
  "scenario": "picker-wheel",
  "dependencies": {
    "resources": ["postgres", "redis"],
    "scenarios": ["shared-auth"]
  },
  "circular_dependencies": false,
  "resource_requirements": {
    "ram_mb": 512,
    "disk_mb": 100,
    "gpu": false
  },
  "tiers": {
    "1": {"overall": 100, "portability": 100, "resources": 100, "licensing": 100, "platform_support": 100},
    "2": {"overall": 45, "portability": 30, "resources": 60, "licensing": 100, "platform_support": 50},
    "3": {"overall": 25, "portability": 20, "resources": 40, "licensing": 100, "platform_support": 20},
    "4": {"overall": 85, "portability": 90, "resources": 80, "licensing": 100, "platform_support": 85},
    "5": {"overall": 60, "portability": 50, "resources": 70, "licensing": 100, "platform_support": 60}
  }
}
```

**Key Fields:**
- `circular_dependencies` - If `true`, deployment is blocked
- `resource_requirements` - Aggregate RAM/disk/GPU needs across all dependencies
- `tiers` - Fitness scores for each deployment tier

**Notes:**
- Calls `scenario-dependency-analyzer` internally
- Blocks if circular dependencies detected

---

## fitness

Calculate fitness scores for a specific tier with detailed breakdown.

```bash
deployment-manager fitness <scenario-name> [--tier <tier>]
```

**Arguments:**
- `<scenario-name>` - Name of the scenario to score

**Flags:**
- `--tier <tier>` - Target tier (1-5 or name). Defaults to all tiers.

**Tier Names:**
| Input | Tier |
|-------|------|
| `local`, `1` | Tier 1 |
| `desktop`, `2` | Tier 2 |
| `mobile`, `ios`, `android`, `3` | Tier 3 |
| `saas`, `cloud`, `web`, `4` | Tier 4 |
| `enterprise`, `on-prem`, `5` | Tier 5 |

**Example:**

```bash
deployment-manager fitness picker-wheel --tier desktop
```

**Output:**

```json
{
  "scenario": "picker-wheel",
  "tier": 2,
  "scores": {
    "overall": 45,
    "portability": 30,
    "resources": 60,
    "licensing": 100,
    "platform_support": 50
  },
  "blockers": [
    "postgres requires swap to sqlite for desktop bundling",
    "redis requires swap to in-process cache"
  ],
  "warnings": [
    "Large dependency tree may increase bundle size"
  ]
}
```

**Score Interpretation:**

| Range | Meaning | Action |
|-------|---------|--------|
| 80-100 | Excellent | Ready to deploy |
| 60-79 | Good | Review warnings, proceed |
| 40-59 | Fair | Address blockers via swaps |
| 20-39 | Poor | Significant work required |
| 0-19 | Critical | Major architecture changes needed |

**Score Components:**
- `overall` - Weighted aggregate of all factors
- `portability` - Cross-platform availability of dependencies
- `resources` - RAM/CPU/storage fit for tier
- `licensing` - Commercial/OSS compatibility
- `platform_support` - OS/architecture coverage

**Example - All Tiers:**

```bash
deployment-manager fitness picker-wheel
```

```json
{
  "scenario": "picker-wheel",
  "tiers": {
    "1": {"overall": 100, "blockers": [], "warnings": []},
    "2": {"overall": 45, "blockers": ["postgres requires swap..."], "warnings": []},
    "3": {"overall": 25, "blockers": ["postgres requires swap...", "heavy dependencies"], "warnings": []},
    "4": {"overall": 85, "blockers": [], "warnings": ["Consider CDN for static assets"]},
    "5": {"overall": 60, "blockers": [], "warnings": ["Compliance review required"]}
  }
}
```

---

## Related

- [Fitness Scoring Guide](../guides/fitness-scoring.md) - Detailed scoring rubric
- [Dependency Swapping Guide](../guides/dependency-swapping.md) - How to resolve blockers
- [Tier Reference](../tiers/README.md) - What each tier means
