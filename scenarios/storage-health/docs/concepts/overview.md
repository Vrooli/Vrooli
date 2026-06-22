# Concepts Overview

`storage-health` owns all storage judgment in Vrooli — schema layout,
migration hygiene, persistence-seam adoption, and (most importantly)
test-isolation safety. It is a test-genie delegated producer that emits
findings on the `storage` dimension via `FINDING_SOURCE_STORAGE`.

The safety throughline: test-genie's destructive playbooks phase drives a
real browser issuing real DB mutations. Whether those land in an isolated
test database is — today, before this scenario — an unenforced, Go-only
convention. `storage-health` makes isolation a **statically-proven,
fail-closed** precondition for destructive E2E. When isolation can't be
proven, test-genie refuses destructive playbooks.

## Durable concept docs

| Doc | What it covers |
|---|---|
| [`storage-model.md`](storage-model.md) | The engines (SQLite / Postgres / Qdrant / Redis / file), the api-core namespace seams (`Collection` / `RedisKey` / `RedisPrefix` / `ScenarioNamespace`, `VROOLI_STORAGE_NAMESPACE`), the per-domain embedded idempotent schema convention, the `storage_stage` greenfield signal, and the L0–L4 maturity ladder. |
| [`test-isolation-contract.md`](test-isolation-contract.md) | The canonical test-isolation contract: header semantics (`X-Vrooli-Test-Mode: 1`), the four-seam qualification cookbook (`database.Open → *RoutedDB`, `EnsureSchemas`, `TestModeMiddleware`, `devrouting.Register`), the `RoutingService` lease contract, mode-flag defaults, and the `primary_during_test_mode_requests` leak counter as in-run defense-in-depth. **Static proof, fail-closed gate.** |

The standard template concept docs (`ARCHITECTURE.md`, `DOMAINS.md`,
`DATA.md`, `FLOWS.md`, `INTEGRATIONS.md`, `UI-ARCHITECTURE.md`) cover the
scenario's own structure; the two docs above are the domain knowledge it
exists to enforce.

## Cross-References

- `../../PRD.md` — operational targets.
- `.vrooli/maturity.json` — authoritative ladder + finding catalog.
- `docs/plans/storage-health-scenario-and-test-genie-producer-plan.md` (repo root) — the source plan.
