# Decisions — API Health

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-07-03 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs keep manifest-backed headings and validation metadata while the content is API Health-specific. | Revisit when scenario adopts a different template or doc contract. |
| 2026-07-03 | Treat scenario-auditor as migration input, not a runtime dependency. | API Health exists to retire the API-rule residue in scenario-auditor. | `.vrooli/service.json` lists scenario-auditor disabled/ignore; implementation must read old rule intent from source and produce a ledger instead of calling scenario-auditor. | Remove once the migration ledger and cutover are complete. |
| 2026-07-03 | Disable requirements auto-sync until implementation tests exist. | The foundation registry contains planned validation refs. | Requirement statuses cannot be accidentally marked complete by refless/planned tests. | Re-enable when real `[REQ:*]` tests exist and validation refs point at concrete files. |
| 2026-07-03 | Redesign legacy HTTP semantics rules around low-ambiguity evidence. | Phase 4 classified `http_status_codes.go`, `content_type_headers.go`, the `content_type_*` companion rules, and `versioned_endpoints.go` as redesigned rather than copied. | API Health reports raw status literals, explicit stdlib JSON error-success, missing content type for obvious writer patterns, and unversioned REST feature routes; it exempts tests, Connect/framework-owned responses, ops probes, and endpoint metadata `rest_exception` paths. Runtime hygiene (`http_client_timeout.go`, `http_response_close.go`, `file_close.go`, `goroutine_context.go`, `application_logging.go`) remains for Phase 5, while security headers stay delegated to security-health. | Revisit when Phase 7 builds the full migration matrix and parity fixtures. |
| 2026-07-03 | Scope runtime hygiene to API-runtime AST evidence. | Phase 5 migrated runtime intent from legacy scenario-auditor rules without preserving regex/window behavior. | API Health reports only production API Go files for unbounded/default HTTP clients, response bodies without local Close, request context drops, long-lived goroutine loops without cancellation, and raw fmt/log operational logging. Broad disk `file_close.go` remains outside API Health unless tied to HTTP request/response lifecycle. | Revisit when Phase 7 completes the full migration matrix or runtime false-positive fixtures expose a broader ownership boundary. |
| 2026-07-03 | Keep auto-fix apply limited to local mechanical edits. | Phase 6 introduced the shared Fix RPC implementation and CLI fix commands. | Auto-fix apply supports service health metadata normalization, standard `/health` endpoint descriptor insertion, raw HTTP status constant replacement, simple JSON content-type insertion, and local outbound response body close insertion. Preflight/server-run migration, health payload repair, endpoint versioning, context propagation, timeout policy, and logging architecture remain manual because they require design context. | Revisit only after a fixture proves one of the manual classes has a deterministic, idempotent rewrite with no product semantics. |
| 2026-07-03 | Account for legacy API rules with a provider-owned ledger. | Phase 7 needed migration evidence without making scenario-auditor a runtime dependency. | API Health stores the migration ledger in `api/internal/migration` and mirrors it in `docs/reference/scenario-auditor-api-migration.md`; tests compare ledger files against the legacy rule directory and keep security headers plus broad file close delegated. | Revisit when Test Genie cutover removes the last scenario-auditor API-rule dependency. |
| 2026-07-03 | Cut Test Genie API readiness over through a dedicated `api` phase. | Phase 9 needed API Health adoption without pretending API Health replaces every remaining scenario-auditor standard. | Test Genie now registers `api` as an api-health-backed ScenarioValidationService phase before `standards`; curated presets include `api`, while `standards` keeps scenario-auditor's REST client for non-API residue. API findings temporarily use the existing standards finding source because the shared architecture enum has no dedicated API source. | Revisit when the shared finding-source contract grows an API source or the remaining scenario-auditor standards residue migrates to focused providers. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| _None yet._ | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
