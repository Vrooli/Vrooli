# Swap Commands

Commands for analyzing and applying dependency swaps.

**Dependency swapping** replaces heavy or non-portable dependencies with bundle-safe alternatives. For example, swapping `postgres` for `sqlite` enables desktop deployment.

## swaps list

List available swap suggestions for a scenario.

```bash
deployment-manager swaps list <scenario> [--format json|table]
```

**Arguments:**
- `<scenario>` - Scenario name

**Flags:**
- `--format` - Output format (default: json)

**Example:**

```bash
deployment-manager swaps list picker-wheel
```

**Output:**

```json
{
  "scenario": "picker-wheel",
  "suggestions": [
    {
      "id": "swap-postgres-sqlite",
      "from": "postgres",
      "to": "sqlite",
      "impact": {
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
        "Zero configuration"
      ],
      "cons": [
        "Single-user access only",
        "No replication support",
        "Limited concurrent writes"
      ],
      "migration_effort": "medium",
      "applicable_tiers": [2, 3]
    },
    {
      "id": "swap-redis-inprocess",
      "from": "redis",
      "to": "in-process",
      "impact": {
        "tier_2": 10,
        "tier_3": 15
      },
      "pros": [
        "No external process",
        "Reduced memory footprint"
      ],
      "cons": [
        "Memory-only, no persistence",
        "Single-instance only"
      ],
      "migration_effort": "low",
      "applicable_tiers": [2, 3]
    },
    {
      "id": "swap-ollama-packaged",
      "from": "ollama",
      "to": "packaged-models",
      "impact": {
        "tier_2": 30
      },
      "pros": [
        "Offline capable",
        "No Ollama server required"
      ],
      "cons": [
        "Large bundle size (models)",
        "Limited to bundled models"
      ],
      "migration_effort": "high",
      "applicable_tiers": [2]
    },
    {
      "id": "swap-browserless-playwright",
      "from": "browserless",
      "to": "playwright-driver",
      "impact": {
        "tier_2": 20
      },
      "pros": [
        "Bundled Chromium",
        "No external service"
      ],
      "cons": [
        "Larger bundle size (~200MB)"
      ],
      "migration_effort": "medium",
      "applicable_tiers": [2]
    }
  ]
}
```

**Table Format:**

```bash
deployment-manager swaps list picker-wheel --format table
```

```
FROM          TO                 TIER 2    TIER 3    EFFORT
postgres      sqlite             +25       +40       medium
redis         in-process         +10       +15       low
ollama        packaged-models    +30       -         high
browserless   playwright-driver  +20       -         medium
```

---

## swaps analyze

Analyze the impact of a specific swap.

```bash
deployment-manager swaps analyze <from> <to>
```

**Arguments:**
- `<from>` - Original dependency name
- `<to>` - Replacement dependency name

**Example:**

```bash
deployment-manager swaps analyze postgres sqlite
```

**Output:**

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
    "Zero configuration"
  ],
  "cons": [
    "Single-user access only",
    "No replication support",
    "Limited concurrent writes",
    "No LISTEN/NOTIFY"
  ],
  "migration_steps": [
    "Update DATABASE_URL format from postgres:// to file://",
    "Convert schema to SQLite dialect (no SERIAL, use INTEGER PRIMARY KEY)",
    "Remove connection pooling configuration",
    "Update any Postgres-specific queries (RETURNING, JSON operators)",
    "Run data migration script"
  ],
  "affected_files": [
    "api/database/connection.go",
    "api/database/migrations/*.sql",
    ".vrooli/service.json"
  ],
  "estimated_effort": {
    "code_changes": "medium",
    "testing": "high",
    "data_migration": "medium"
  }
}
```

**Fitness Delta Interpretation:**
- Positive values = improved fitness for that tier
- Negative values = reduced fitness (trade-off)
- `0` = no change

---

## swaps cascade

Detect cascading impacts of a swap on dependent scenarios.

```bash
deployment-manager swaps cascade <from> <to>
```

**Arguments:**
- `<from>` - Original dependency name
- `<to>` - Replacement dependency name

**Example:**

```bash
deployment-manager swaps cascade postgres sqlite
```

**Output:**

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
      "notes": "Uses postgres for data storage"
    },
    {
      "scenario": "shared-auth",
      "impact": "indirect",
      "notes": "picker-wheel depends on shared-auth which uses postgres"
    }
  ],
  "cascade_depth": 2,
  "total_affected": 2,
  "recommendation": "Apply swap to shared-auth first, then picker-wheel"
}
```

**Impact Types:**
- `direct` - Scenario directly uses the dependency
- `indirect` - Scenario uses another scenario that uses the dependency

---

## swaps info

Get detailed information about a specific swap suggestion.

```bash
deployment-manager swaps info <swap_id>
```

**Arguments:**
- `<swap_id>` - Swap ID from `swaps list` output

**Example:**

```bash
deployment-manager swaps info swap-postgres-sqlite
```

**Output:**

```json
{
  "id": "swap-postgres-sqlite",
  "from": {
    "name": "postgres",
    "type": "resource",
    "description": "PostgreSQL relational database",
    "requirements": {
      "ram_mb": 256,
      "disk_mb": 500,
      "network": true
    }
  },
  "to": {
    "name": "sqlite",
    "type": "resource",
    "description": "Embedded SQLite database",
    "requirements": {
      "ram_mb": 10,
      "disk_mb": 50,
      "network": false
    }
  },
  "compatibility": {
    "tier_1": "compatible",
    "tier_2": "recommended",
    "tier_3": "recommended",
    "tier_4": "not_recommended",
    "tier_5": "not_recommended"
  },
  "documentation": "/docs/guides/dependency-swapping.md#postgres-to-sqlite"
}
```

---

## swaps apply

Apply a swap to a deployment profile.

```bash
deployment-manager swaps apply <profile_id> <from> <to> [--show-fitness]
```

**Arguments:**
- `<profile_id>` - Target profile ID
- `<from>` - Original dependency name
- `<to>` - Replacement dependency name

**Flags:**
- `--show-fitness` - Show updated fitness scores after applying

**Example:**

```bash
deployment-manager swaps apply profile-123 postgres sqlite --show-fitness
```

**Output:**

```json
{
  "status": "applied",
  "profile_id": "profile-123",
  "swap": {
    "from": "postgres",
    "to": "sqlite"
  },
  "new_version": 4,
  "fitness_after": {
    "tier_2": {
      "overall": 70,
      "previous": 45,
      "delta": 25
    }
  }
}
```

**Notes:**
- Swaps are recorded in the profile only
- Does not modify the actual scenario code
- Use `profile swap remove` to undo

---

## Common Swaps Reference

| From | To | Best For | Fitness Impact |
|------|-----|----------|----------------|
| `postgres` | `sqlite` | Desktop, Mobile | +25/+40 |
| `redis` | `in-process` | Desktop, Mobile | +10/+15 |
| `ollama` | `packaged-models` | Desktop | +30 |
| `browserless` | `playwright-driver` | Desktop | +20 |
| `vault` | `file-secrets` | Desktop | +15 |

See [Dependency Swapping Guide](../guides/dependency-swapping.md) for detailed migration instructions.

---

## Related

- [Profile Commands](profile-commands.md) - Manage profiles that hold swaps
- [Overview Commands](overview-commands.md) - Check fitness before/after swaps
- [Dependency Swapping Guide](../guides/dependency-swapping.md) - Detailed swap strategies
