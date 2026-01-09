# Swap Endpoints

Endpoints for analyzing dependency swaps.

## GET /swaps/analyze/{from}/{to}

Analyze the impact of swapping one dependency for another.

**Request:**

```bash
curl "http://localhost:${API_PORT}/api/v1/swaps/analyze/postgres/sqlite"
```

**Response:**

```json
{
  "from": "postgres",
  "to": "sqlite",
  "fitness_delta": {
    "tier_1": 0,
    "tier_2": 25,
    "tier_3": 40,
    "tier_4": -5,
    "tier_5": -10
  },
  "pros": [
    "No server process required",
    "Single file database",
    "Cross-platform native support",
    "Zero configuration",
    "Reduced memory footprint"
  ],
  "cons": [
    "Single-user access only",
    "No replication support",
    "Limited concurrent writes",
    "No LISTEN/NOTIFY support",
    "No JSON operators"
  ],
  "migration_steps": [
    "Update DATABASE_URL format from postgres:// to file://",
    "Convert schema to SQLite dialect (INTEGER PRIMARY KEY vs SERIAL)",
    "Remove connection pooling configuration",
    "Update Postgres-specific queries (RETURNING, JSON ops)",
    "Run data migration script"
  ],
  "migration_effort": "medium",
  "applicable_tiers": [2, 3],
  "not_recommended_tiers": [4, 5]
}
```

**Path Parameters:**

| Parameter | Description |
|-----------|-------------|
| `from` | Original dependency name |
| `to` | Replacement dependency name |

---

## GET /swaps/cascade/{from}/{to}

Detect cascading impacts on dependent scenarios.

**Request:**

```bash
curl "http://localhost:${API_PORT}/api/v1/swaps/cascade/postgres/sqlite"
```

**Response:**

```json
{
  "swap": {
    "from": "postgres",
    "to": "sqlite"
  },
  "affected_scenarios": [
    {
      "scenario": "picker-wheel",
      "impact": "direct",
      "dependency_path": ["picker-wheel", "postgres"],
      "notes": "Uses postgres for primary data storage"
    },
    {
      "scenario": "shared-auth",
      "impact": "direct",
      "dependency_path": ["shared-auth", "postgres"],
      "notes": "Uses postgres for user accounts"
    },
    {
      "scenario": "analytics-dashboard",
      "impact": "indirect",
      "dependency_path": ["analytics-dashboard", "picker-wheel", "postgres"],
      "notes": "Depends on picker-wheel which uses postgres"
    }
  ],
  "cascade_depth": 2,
  "total_affected": 3,
  "recommendation": "Apply swap to shared-auth and picker-wheel first, then analytics-dashboard will inherit the change"
}
```

**Impact Types:**

| Type | Description |
|------|-------------|
| `direct` | Scenario directly depends on the resource |
| `indirect` | Scenario depends on another scenario that uses the resource |

---

## Common Swap Pairs

### postgres → sqlite

**Best for:** Desktop (Tier 2), Mobile (Tier 3)

```json
{
  "fitness_delta": { "tier_2": +25, "tier_3": +40 },
  "migration_effort": "medium",
  "key_changes": [
    "Connection string format",
    "Auto-increment syntax",
    "Type mappings (TEXT vs VARCHAR)",
    "No RETURNING clause"
  ]
}
```

### redis → in-process

**Best for:** Desktop (Tier 2), Mobile (Tier 3)

```json
{
  "fitness_delta": { "tier_2": +10, "tier_3": +15 },
  "migration_effort": "low",
  "key_changes": [
    "Cache API remains similar",
    "No persistence (memory only)",
    "Single-instance only"
  ]
}
```

### ollama → packaged-models

**Best for:** Desktop (Tier 2)

```json
{
  "fitness_delta": { "tier_2": +30 },
  "migration_effort": "high",
  "key_changes": [
    "Bundle specific model files",
    "Different inference API",
    "Larger bundle size (models)"
  ]
}
```

### browserless → playwright-driver

**Best for:** Desktop (Tier 2)

```json
{
  "fitness_delta": { "tier_2": +20 },
  "migration_effort": "medium",
  "key_changes": [
    "Bundled Chromium (~200MB)",
    "Playwright API instead of puppeteer",
    "Local process instead of remote service"
  ]
}
```

### vault → file-secrets

**Best for:** Desktop (Tier 2)

```json
{
  "fitness_delta": { "tier_2": +15 },
  "migration_effort": "low",
  "key_changes": [
    "Encrypted local file storage",
    "No remote vault connection",
    "Per-install encryption keys"
  ]
}
```

---

## Swap Compatibility Matrix

| From | To | T1 | T2 | T3 | T4 | T5 |
|------|-----|:--:|:--:|:--:|:--:|:--:|
| postgres | sqlite | - | Rec | Rec | NR | NR |
| redis | in-process | - | Rec | Rec | - | - |
| ollama | packaged | - | Opt | - | - | - |
| browserless | playwright | - | Opt | - | - | - |
| vault | file-secrets | - | Rec | Rec | NR | NR |

**Legend:**
- `Rec` = Recommended
- `Opt` = Optional
- `NR` = Not Recommended
- `-` = No change

---

## Error Responses

**Unknown swap pair:**

```json
{
  "error": "swap_not_found",
  "details": "No known swap from 'unknown-dep' to 'other-dep'"
}
```

**Incompatible swap:**

```json
{
  "error": "swap_incompatible",
  "details": "Cannot swap 'postgres' to 'redis' - different resource types"
}
```

---

## Related

- [CLI Swap Commands](../cli/swap-commands.md) - CLI interface
- [Profile Endpoints](profiles.md) - Apply swaps to profiles
- [Dependency Swapping Guide](../guides/dependency-swapping.md) - Migration strategies
