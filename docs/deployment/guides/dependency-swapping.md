# Dependency Swapping Guide

Deployment succeeds only when every dependency fits the target platform. This guide explains how `deployment-manager` will propose swaps and how to annotate scenarios today.

## Metadata Schema

Extend `.vrooli/service.json` (resource or scenario) with deployment metadata:

```json
{
  "deployment": {
    "resource_type": "database",
    "platforms": {
      "desktop": {
        "supported": false,
        "fitness_score": 0.3,
        "requirements": {"ram_mb": 1024, "disk_mb": 512, "network": "required"},
        "alternatives": ["sqlite"]
      },
      "mobile": {
        "supported": false,
        "reason": "Requires always-on server",
        "alternatives": ["sqlite"]
      },
      "cloud": {
        "supported": true,
        "fitness_score": 0.95
      }
    },
    "offline_capable": false,
    "complexity": "high"
  }
}
```

Key fields:
- `resource_type`: Helps find peers (database, vector-db, llm, automation, etc.).
- `platforms.<tier>.alternatives`: Candidate replacements when `supported=false` or score < threshold.
- `requirements`: Input for sizing (RAM, disk, CPU, GPU, network, OS).

## Swap Workflow (future `deployment-manager` UI)

1. **Target Selection** — Choose deployment tier (Desktop, Mobile, SaaS).
2. **Fitness Overview** — Table summarizing each dependency with fitness score and blockers.
3. **Swap Suggestions** — For each blocker, show alternatives + estimated work (duration, impacted files, scenario owners).
4. **Decision Capture** — Accept swaps to generate app-issue-tracker tasks or mark "won't fix" with rationale.
5. **Bundle Manifest** — Export JSON describing the chosen dependency set for `scenario-to-*` packagers.

## Manual Process (Today)

Until the UI exists:

1. Run `scenario-dependency-analyzer` to refresh `service.json` dependency listings.
2. Manually evaluate each dependency against the target tier.
3. Document swap ideas in the scenario README + app-issue-tracker issues.
4. Update deployment metadata so future runs surface the same guidance automatically.

## Example Table

| Dependency | Current | Desktop Fitness | Suggested Swap | Notes |
|------------|---------|-----------------|----------------|-------|
| Database | Postgres | 0.3 ⚠️ | SQLite | Embed file DB, drop multi-tenant features |
| Vector DB | Qdrant | 0.5 ⚠️ | ChromaDB | Use lightweight embeddings, limit dataset |
| LLM | Ollama | 0.0 ❌ | OpenRouter | Requires user API key prompt |
| Automation | n8n | 0.2 ⚠️ | Compiled workflows | Convert to native code (future `n8n-to-code`?) |

This is the experience `deployment-manager` will automate. Documenting it now keeps everyone aligned.
