# CLI Commands

Phase 6 will replace the generated notes CLI with Code Facts commands backed by generated Connect clients.

| Command | Purpose |
|---|---|
| `code-facts describe <target> --include <families> --json` | Return selected fact families. |
| `code-facts surfaces <target> --json` | Inspect target context, surfaces, and parse units. |
| `code-facts proto-adoption <target> --json` | Inspect proto adoption evidence. |
| `code-facts endpoint-proof <target> <endpoint-id> --json` | Inspect endpoint proof evidence. |
| `code-facts cache status --json` | Show cache diagnostics. |
| `code-facts cache clear <target> --dry-run` | Preview cache invalidation. |

The CLI must remain a thin API translation layer; proof and cache logic belongs in the API.
