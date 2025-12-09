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

## Service.json Schema Draft

Deployment metadata lives under a single `deployment` key inside `.vrooli/service.json`. The analyzer owns everything except the optional overrides.

```json
{
  "deployment": {
    "metadata_version": 1,
    "last_analyzed_at": "2025-02-05T18:22:11Z",
    "analyzer": {
      "name": "scenario-dependency-analyzer",
      "version": "0.3.0"
    },
    "aggregate_requirements": {
      "ram_mb": 1536,
      "disk_mb": 4200,
      "cpu_cores": 3,
      "gpu": false,
      "network": "required",
      "storage_mb_per_user": 25,
      "startup_time_ms": 18000,
      "bucket": "medium",
      "source": "calculated",
      "confidence": "medium"
    },
    "tiers": {
      "tier-1-local": {
        "status": "ready",
        "fitness_score": 1.0,
        "constraints": ["requires_network"],
        "notes": "Runs in Tier 1 today via app-monitor",
        "requirements": {
          "ram_mb": 1024,
          "disk_mb": 1800,
          "cpu_cores": 2,
          "bucket": "medium",
          "source": "calculated"
        },
        "adaptations": [],
        "secrets": [
          {
            "secret_id": "picker-wheel_api_key",
            "classification": "service",
            "strategy_ref": "file:.vrooli/secrets-deployment.json#picker-wheel"
          }
        ],
        "artifacts": [
          {
            "type": "proxy-access",
            "producer": "app-monitor",
            "status": "shipping"
          }
        ]
      },
      "tier-2-desktop": {
        "status": "blocked",
        "fitness_score": 0.3,
        "constraints": ["requires_always-on_api", "no_offline_mode"],
        "notes": "Needs sqlite + secrets rewiring",
        "requirements": {
          "ram_mb": 1536,
          "disk_mb": 4200,
          "cpu_cores": 3,
          "bucket": "large",
          "source": "estimated"
        },
        "adaptations": [
          {
            "dependency": "postgres",
            "swap": "sqlite",
            "impact": "functional",
            "effort_days": 2
          },
          {
            "dependency": "ollama",
            "swap": "openrouter",
            "impact": "ai-capability",
            "effort_days": 1
          }
        ],
        "secrets": [
          {
            "secret_id": "ollama_api_key",
            "classification": "user",
            "strategy_ref": "file:.vrooli/secrets-deployment.json#ollama"
          }
        ],
        "artifacts": [
          {
            "type": "desktop-installer",
            "producer": "scenario-to-desktop",
            "status": "thin-client"
          }
        ]
      }
    },
    "dependencies": {
      "resources": {
        "postgres": {
          "deployment": {
            "resource_type": "database",
            "footprint": {
              "ram_mb": 512,
              "disk_mb": 1024,
              "cpu_cores": 1,
              "bucket": "medium",
              "source": "calculated"
            },
            "platform_support": {
              "tier-1-local": {
                "supported": true,
                "fitness_score": 0.95
              },
              "tier-2-desktop": {
                "supported": false,
                "fitness_score": 0.2,
                "reason": "Requires Docker + network access",
                "alternatives": ["sqlite"],
                "requirements": {
                  "ram_mb": 512,
                  "disk_mb": 512
                }
              },
              "tier-4-saas": {
                "supported": true,
                "fitness_score": 0.9,
                "requirements": {
                  "ram_mb": 512,
                  "disk_mb": 512,
                  "cpu_cores": 1
                }
              }
            },
            "swappable_with": [
              {
                "id": "sqlite",
                "relationship": "drop-in",
                "notes": "Needs migration of schema + data"
              }
            ],
            "packaging_hints": ["requires_persistent_storage"]
          }
        }
      },
      "scenarios": {
        "secrets-manager": {
          "deployment": {
            "resource_type": "scenario",
            "footprint": {
              "ram_mb": 256,
              "disk_mb": 256,
              "cpu_cores": 1,
              "bucket": "small"
            },
            "platform_support": {
              "tier-2-desktop": {
                "supported": false,
                "fitness_score": 0.1,
                "reason": "Depends on Postgres + Vault"
              },
              "tier-4-saas": {
                "supported": true,
                "fitness_score": 0.85
              }
            }
          }
        }
      }
    },
    "overrides": [
      {
        "tier": "tier-2-desktop",
        "field": "fitness_score",
        "value": 0.3,
        "reason": "Manual downgrade until sqlite migration ships",
        "expires_at": "2025-03-01"
      }
    ]
  }
}
```

### Field Reference

- `metadata_version` — bump when the schema changes so analyzers can rehydrate.
- `aggregate_requirements` — rolled-up RAM/disk/CPU/network/gpu for the entire bundle plus `bucket` (small/medium/large), `source` (calculated/estimated/manual), and `confidence`.
- `tiers.<tier>` — canonical keys (`tier-1-local`, `tier-2-desktop`, `tier-3-mobile`, `tier-4-saas`, `tier-5-enterprise`). Each tier stores status (`ready|limited|blocked`), `fitness_score` (single numeric), `constraints`, optional `notes`, per-tier `requirements`, `adaptations` (recommended swaps/changes), `secrets` (references into the secrets config), and `artifacts` describing what scenario-to-* can emit.
- `dependencies.resources|scenarios` — attaches deployment metadata to each dependency so deployment-manager can reason about individual blockers.
- `overrides` — manual adjustments when calculated data is wrong or incomplete. Include `expires_at` to force revalidation.

### Dependency Metadata Expectations

- `resource_type` — categorize the dependency (`database`, `vector-db`, `llm-provider`, `workflow-engine`, `scenario`, etc.) for swap suggestions.
- `footprint` — per-dependency resource usage; use estimations + `bucket` when telemetry is missing.
- `platform_support` — nested per tier; `supported`, `fitness_score`, optional `reason`, `requirements`, and `alternatives` help power the swap UI.
- `swappable_with` — list of viable replacements referencing resource IDs or scenario IDs, plus notes about migration effort.
- `packaging_hints` — free-form array for things scenario-to-* must know (needs GPU, has native addons, requires Docker, etc.).

### Secrets Config (separate file)

Secrets get their own `.vrooli/secrets-deployment.json` so lifecycle metadata stays tidy. `service.json` only references `secret_id` + `strategy_ref`.

```json
{
  "secrets": [
    {
      "id": "postgres_connection",
      "description": "Internal DB connection string",
      "classification": "infrastructure",
      "tiers": {
        "tier-1-local": {
          "action": "reuse",
          "source": "secrets-manager:vault://secret/data/vrooli/dev/postgres"
        },
        "tier-2-desktop": {
          "action": "generate_sqlite_path",
          "template": "file://{{USER_DATA}}/picker-wheel.db"
        },
        "tier-4-saas": {
          "action": "environment_variable",
          "template": "${POSTGRES_URL}",
          "rotation": "per-deployment"
        }
      }
    },
    {
      "id": "ollama_api_key",
      "description": "AI provider key",
      "classification": "user",
      "tiers": {
        "tier-2-desktop": {
          "action": "prompt_user",
          "prompt": "Provide your OpenRouter API key to unlock AI features",
          "storage": "electron.safeStorage"
        }
      }
    }
  ]
}
```

`strategy_ref` strings inside `service.json` point to anchors (e.g., `file:.vrooli/secrets-deployment.json#postgres_connection.tier-2-desktop`) so deployment-manager knows how to prep secrets for each tier.

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
