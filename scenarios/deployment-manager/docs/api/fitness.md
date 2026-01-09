# Fitness Endpoints

Endpoints for analyzing scenario fitness for deployment tiers.

## GET /health

Check API and dependency health.

**Request:**

```bash
curl "http://localhost:${API_PORT}/api/v1/health"
```

**Response:**

```json
{
  "status": "healthy",
  "api_version": "1.1.0",
  "dependencies": {
    "scenario-dependency-analyzer": "healthy",
    "secrets-manager": "healthy",
    "postgres": "healthy",
    "redis": "healthy"
  },
  "uptime_seconds": 3600
}
```

---

## GET /dependencies/analyze/{scenario}

Analyze a scenario's full dependency graph.

**Request:**

```bash
curl "http://localhost:${API_PORT}/api/v1/dependencies/analyze/picker-wheel"
```

**Response:**

```json
{
  "scenario": "picker-wheel",
  "dependencies": {
    "resources": [
      {
        "name": "postgres",
        "type": "resource",
        "required": true,
        "requirements": {
          "ram_mb": 256,
          "disk_mb": 500
        }
      },
      {
        "name": "redis",
        "type": "resource",
        "required": true,
        "requirements": {
          "ram_mb": 64,
          "disk_mb": 50
        }
      }
    ],
    "scenarios": [
      {
        "name": "shared-auth",
        "type": "scenario",
        "required": true
      }
    ]
  },
  "circular_dependencies": false,
  "depth": 2,
  "resource_requirements": {
    "ram_mb": 512,
    "disk_mb": 650,
    "gpu": false
  },
  "tiers": {
    "1": { "overall": 100 },
    "2": { "overall": 45 },
    "3": { "overall": 25 },
    "4": { "overall": 85 },
    "5": { "overall": 60 }
  }
}
```

**Notes:**
- Calls scenario-dependency-analyzer internally
- Aggregates resource requirements across all dependencies
- Returns fitness scores for all tiers

---

## POST /fitness/score

Calculate detailed fitness scores for specific tiers.

**Request:**

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/fitness/score" \
  -H "Content-Type: application/json" \
  -d '{
    "scenario": "picker-wheel",
    "tiers": [2, 3]
  }'
```

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `scenario` | string | Yes | Scenario name |
| `tiers` | array[int] | No | Tiers to score (default: all) |
| `with_swaps` | object | No | Hypothetical swaps to apply |

**Response:**

```json
{
  "scenario": "picker-wheel",
  "scores": {
    "2": {
      "overall": 45,
      "portability": 30,
      "resources": 60,
      "licensing": 100,
      "platform_support": 50
    },
    "3": {
      "overall": 25,
      "portability": 20,
      "resources": 40,
      "licensing": 100,
      "platform_support": 20
    }
  },
  "blockers": {
    "2": [
      "postgres requires swap to sqlite for desktop bundling",
      "redis requires swap to in-process cache"
    ],
    "3": [
      "postgres not available on mobile",
      "redis not available on mobile",
      "Heavy dependencies exceed mobile resource limits"
    ]
  },
  "warnings": {
    "2": ["Large dependency tree may increase bundle size"],
    "3": []
  }
}
```

**With Hypothetical Swaps:**

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/fitness/score" \
  -H "Content-Type: application/json" \
  -d '{
    "scenario": "picker-wheel",
    "tiers": [2],
    "with_swaps": {
      "postgres": "sqlite",
      "redis": "in-process"
    }
  }'
```

**Response:**

```json
{
  "scenario": "picker-wheel",
  "scores": {
    "2": {
      "overall": 75,
      "portability": 70,
      "resources": 85,
      "licensing": 100,
      "platform_support": 70
    }
  },
  "blockers": {
    "2": []
  },
  "warnings": {
    "2": ["Bundle size estimated at 180MB"]
  },
  "swaps_applied": {
    "postgres": "sqlite",
    "redis": "in-process"
  }
}
```

---

## Fitness Scoring Rubric

### Score Components

| Component | Weight | Description |
|-----------|--------|-------------|
| `overall` | - | Weighted average of all components |
| `portability` | 30% | Cross-platform availability |
| `resources` | 30% | RAM/CPU/storage fit |
| `licensing` | 20% | Commercial/OSS compatibility |
| `platform_support` | 20% | OS/architecture coverage |

### Baseline Scores by Tier

| Tier | Baseline | Description |
|------|----------|-------------|
| 1 (Local) | 100 | No constraints |
| 2 (Desktop) | 75 | Lightweight deps required |
| 3 (Mobile) | 40 | Heavy swaps needed |
| 4 (SaaS) | 85 | Web-native preferred |
| 5 (Enterprise) | 60 | Compliance friction |

### Score Ranges

| Range | Status | Meaning |
|-------|--------|---------|
| 80-100 | Ready | Deploy without changes |
| 60-79 | Good | Minor optimizations suggested |
| 40-59 | Fair | Blockers need resolution |
| 20-39 | Poor | Significant work required |
| 0-19 | Blocked | Major changes needed |

### Blocker vs Warning

- **Blocker**: Prevents deployment to the tier
- **Warning**: Non-blocking concern to consider

---

## Score Modifiers

Swaps affect scores:

| Swap | Tier 2 | Tier 3 | Tier 4 | Tier 5 |
|------|--------|--------|--------|--------|
| postgres → sqlite | +25 | +40 | -5 | -10 |
| redis → in-process | +10 | +15 | 0 | 0 |
| ollama → packaged | +30 | - | - | - |
| browserless → playwright | +20 | - | - | - |

---

## Related

- [CLI Overview Commands](../cli/overview-commands.md) - CLI fitness commands
- [Swap Endpoints](swaps.md) - Analyze swap impacts
- [Fitness Scoring Guide](../guides/fitness-scoring.md) - Detailed rubric
