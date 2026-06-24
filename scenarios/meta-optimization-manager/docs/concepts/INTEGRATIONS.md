# Integrations

## Purpose Of This Document

The dependency contract for **meta-optimization-manager**: the resources, scenarios, and services it reads. Because it is a thin aggregator, almost every dependency is a **read** of another scenario's typed output, and every one is **soft** — it degrades gracefully rather than failing.

## Dependency Inventory

| Kind | Dependency | Why |
|---|---|---|
| Resource | SQLite (`api-core/storage`) | Owned state (gaps, trials history, fitness index, snapshots). |
| Scenario (denominator owner) | `search-hub`, `test-genie`, `prompt-manager` | The `space --projection` verb (denominators) + their registries (numerators). |
| Scenario (read) | `completeness-scoring`, `code-facts`, `architecture-cartographer`, `scenario-auditor` | Maturity score; structural counts + clean scans for convergence. |
| Scenario (trials) | `agent-manager`, `workspace-sandbox` | Run local-model SWE tasks with diff attribution + isolation. |

## Vrooli Resources

- **SQLite** via `api-core/storage` — the only resource. No Ollama/Qdrant: this scenario aggregates typed JSON; it does not embed or search. (Any local model used by `trials` is mediated by `agent-manager`'s runner config, not a direct resource of this scenario.)

Environment + service-manifest details: see [../reference/configuration.md](../reference/configuration.md).

## Scenario Dependencies

All **soft / degrade gracefully**:
- **search-hub / test-genie / prompt-manager** — must expose `space --projection <p> --json` (the shared denominator-read contract) and their numerator registries (`providers list`, `health`/`fleet status`, `graph health`).
- **completeness-scoring** — `GetScore` for the maturity dimension.
- **code-facts / architecture-cartographer** — structural data for convergence fitness counts (read, not re-run).
- **scenario-auditor / test-genie** — clean-on-all-tools status for reference health (read, not re-run).
- **agent-manager / workspace-sandbox** — `trials` dispatch (runner selection + local-model config) and sandboxed diff attribution.

## Third-Party Services

None directly. The local model exercised by `trials` is reached through `agent-manager`'s runner (e.g. opencode + a local backend); this scenario never calls an external LLM provider itself.

## Failure Modes

- **A read source is down** → that projection/section reports "unavailable"; the rest of the snapshot still computes. Never a false-fail.
- **An owner lacks the `space` verb** → that projection is "uncomputable"; surfaced as an explicit integrity finding, not a crash.
- **agent-manager / workspace-sandbox unavailable** → `trials run` is skipped with a clear reason; the historical trend is unaffected.
- **Stale upstream registry** → coverage is computed against whatever the registry reports, always paired with denominator-confidence.

## Cross-References

- [../reference/configuration.md](../reference/configuration.md) — environment + service.json shape.
- [ARCHITECTURE.md](ARCHITECTURE.md) — the read-client contracts.
- [DOMAINS.md](DOMAINS.md) — which domain consumes each dependency.
- [../internal/SEAMS.md](../internal/SEAMS.md) — the per-scenario read-client seams + test doubles.
