# CLI Commands

The CLI is a thin wrapper over `CodeFactsService`. Human output is summarized; `--json` emits the proto response shape.

| Command | Purpose |
|---|---|
| `code-facts facts describe <target> --include <families> --json` | Return selected fact families. |
| `code-facts facts search <query> --target <target> --limit <n> --json` | Search indexed node evidence while preserving provenance. |
| `code-facts facts surfaces <target> --json` | Inspect target context and surfaces. |
| `code-facts facts proto-adoption <target> --json` | Inspect proto adoption evidence. |
| `code-facts facts endpoint-proof <target> --endpoint <id> --json` | Inspect endpoint proof evidence. |
| `code-facts cache status <target> --json` | Show cache diagnostics. |
| `code-facts cache inspect <target> --cache-key <key> --json` | Inspect matching cache entries. |
| `code-facts cache clear <target> --dry-run` | Preview cache invalidation. |

The CLI must remain a thin API translation layer; proof and cache logic belongs in the API.
