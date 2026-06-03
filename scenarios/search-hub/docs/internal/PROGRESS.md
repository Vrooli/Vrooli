# Progress — Search Hub

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-03 | agent (Phase 1) | done | Scaffold generated; PRD + 11 requirement modules + proto (registry/routing) landed; deps set (postgres/ollama required, 5 soft providers); green end-to-end. See plan Appendix B. |
| 2026-06-03 | agent (Phase 2) | done | Orientation docs authored: real `DOMAINS.md` (registry/providers/routing/rerank/metrics, mapped to requirement modules + build phases); `DATA.md` (postgres registry+telemetry, no corpus tables); `ARCHITECTURE.md` (thin-router invariant + storage diagram + intentional deviations); `INTEGRATIONS.md` (dependency-decisions rationale). 6/8 orient gates green. |
| 2026-06-03 | agent (Phase 2) | deferred | `--finalize` deferred to end of Phase 3: the two remaining gates (`example-domain-removed`, `scaffold-health`) require the first real domain (`registry`) to replace `notes`, per START-HERE ("build a real domain, prove green, then remove the example"). `scaffold-health` is also blocked by a scaffold-template **false-positive** standards critical (`api/internal/httpc/doer.go:34` — a compile-time `var _ Doer = (*http.Client)(nil)` assertion, not a real timeout-less client). See plan Appendix C. |
| 2026-06-03 | agent (Phase 4) | done | Router core: explicit-type fan-out. `routing` domain (`internal/routing/`): `Router.Query` selects providers by explicit `--type`/`--all`/`--group` (no classifier yet — a query with no selector is rejected, not silently widened), resolves each provider's live base URL via api-core/discovery's cross-scenario resolver, fans out over a 30s-timeout `httpc` client (bounded concurrency 6, per-provider 10s timeout), maps every response through the generic `providers.MapResults`, and returns honest by-provider grouping (`reranked=false` until Phase 6). A degraded/unreachable provider returns a noted group, never failing the query. `handlers/routing/` Connect handler (`Query` live; `Status` Unimplemented → Phase 7). CLI `query` group (`search-hub query "<text>"` via DefaultSubcommand; `--type/--all/--group/--limit/--explain`; operator-friendly grouped output + `--json`). Registered 2nd+3rd live providers — `ui-health.surfaces` + `swarm-manager.records` (seeds added; A.5 mappings corrected against real wire shapes — ui-health is camelCase Connect-JSON with no `id` field, swarm records carry no `title`). **Live 3-provider fan-out verified** (cli-health + ui-health + swarm-manager). Fixed a real gosec **G109 ERROR** the new `--limit` parse introduced (`strconv.ParseInt(.,.,32)` instead of `int32(atoi)`). `make test` (test-genie comprehensive) exits 0. See plan Appendix E. |
| 2026-06-03 | agent (Phase 3) | done | Provider registry + registration contract shipped. **Storage decision reversed: SQLite, not PostgreSQL** (template grain + every sibling RoutedDB scenario uses SQLite + no new test dependency; postgres dep removed from `service.json`; concept docs updated). `registry` domain: `internal/registry/` (SQLite store persisting descriptors as protojson blobs + projected filter columns, descriptor validation, Normalize/Validate) + `handlers/registry/` Connect handlers (`RegisterProvider`/`ListProviders`/`DeregisterProvider`). `providers` domain: generic descriptor→`SearchHit` result-mapping adapter (`internal/providers/`) with the first leaf seed `cli-health.commands`, unit-tested against a captured fixture. Added `filter_field`/`filter_value` to `ResultMapping` proto. CLI `providers register\|list\|remove`. **`notes` worked example removed** across api/cli/ui/proto/manifest/endpoints. Fixed a pre-existing UI-scaffold defect surfaced by running vitest: React Router v6 future-flag `console.warn`s (now opted-in) + duplicate nav-landmark a11y label. api+cli `go build`/`go test`/`golangci-lint`/`gofumpt` green; UI `tsc`/`vitest` (127 tests) green. See plan Appendix D. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
