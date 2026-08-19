# Progress — Notification Hub

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-17 | claude | done | Regenerated the scenario from `react-vite` 1.6.5 after the previous notification-hub was assessed as architecturally stranded: none of the template's structural markers, 6 of 20 business endpoints implemented, every send path terminating in a log line, and an empty requirements registry. Authored the charter PRD and populated the requirements registry with 23 operational targets across three modules. |
| 2026-08-17 | claude | done | Folded the monetization research back into the charter. Promoted acknowledgement and escalation from P2 to P1 and added a blocking ask primitive, on the finding that return traffic is what separates this scenario from both adjacent markets. P1 went 8 → 11, P2 went 5 → 3, P0 unchanged at 13; traceability matrix reports 27 of 27 rows linked and `business-health validate scenario notification-hub` passes. **Requirement id remap:** `NOTIFICA-P2-004` → `NOTIFICA-P1-009` (acknowledgement and response), `NOTIFICA-P2-005` → `NOTIFICA-P1-011` (escalation chains); `NOTIFICA-P1-010` (blocking ask primitive) is new. Safe to renumber because neither moved requirement was implemented or carried a `[REQ:...]` test tag. |
| 2026-08-18 | claude | done | Settled the delivery architecture and rewrote the concept docs against it. Replaced the planned push resource with Web Push from this scenario's own installed PWA, so `dependencies.resources` stays empty and OT-P0-009 is literally true; the rejected self-hosted relay is preserved as the `ntfy` resource blueprint at `status: candidate`. Added `notification-hub` to the operator core seed (and updated the coreset test that pins membership) so the public origin never expires, because a push subscription dies with its origin. Declared `tunnel-manager` a required runtime dependency. Rewrote `DOMAINS.md` with the real five-domain map, `DATA.md` with the schema and retention rules, `FLOWS.md` with three Mermaid flows and the state-machine table, and `SECURITY.md` with the threat model; promoted all four to `active`. Requirements: OT-P0-001 rewritten, OT-P0-014 and OT-P0-015 added, OT-P1-008 reframed, former OT-P2-001 promoted into P0 as the mechanism. P0 is now 15, P1 11, P2 2. Recorded five new durable decisions and superseded the cloud-api push-provider decision. Filed three new problems: the vrooli-events defects, the bridge per-scenario dispatch change, and a sensitivity gap in escalation. |
| 2026-08-18 | codex | done | Replaced the generated notification scaffold with protobuf/Connect domains for intake, recipients, routing, delivery, and conversations; added SQLite lifecycle tables, verified owner identity, RFC 8291 Web Push/VAPID transport, retry/receipt evidence, PWA push handling, and durable event fan-out with signed webhook retries in vrooli-events. Refactored vrooli-bridge to derive every run-eligible dispatch verb from the shared scope catalog and require paired effect plus verb grants. |
| 2026-08-17 | claude | done | Closed the charter-versus-scaffold contradictions found on review. Declared `scenario-authenticator` (required, `try_start`), `vrooli-bridge`, and `vrooli-events` in `.vrooli/service.json` as `runtime_only` dependencies with explicit `degraded_behavior`; rewrote `INTEGRATIONS.md` as a real dependency contract including the rejected-resource rationale and the upstream `vrooli-events` fan-out gap; backfilled ten durable decisions into `DECISIONS.md`; opened three `PROBLEMS.md` entries; and re-stamped 17 documents in `docs/manifest.json` from `active` to `stub`/`draft` so documentation health stops reporting generic scaffold as finished. |
| 2026-08-18 | codex | done | Completed the durable P0/P1 implementation pass: recipient/device/channel CRUD, per-channel sensitivity routing, quiet windows, retry receipts, scheduling, digest collapse, SMTP/macOS transport seams, escalation worker, bridge-safe base64 relay through current machine lineage, event subscription reconciliation, nested event payload intake, analytics, health authenticator reachability, Ask response projection, and browser device settings. Local API/CLI/UI suites pass; physical iPhone/Mac arrival and Test Genie notification-hub evidence remain operator-owned validation gates. |
| 2026-08-18 | codex | done | Removed the generated `notes` example domain from scenario surfaces and all Notification Hub proto outputs through template-manager; reconciled the documentation debt entry after every registered document reached active maturity. |
| 2026-08-18 | codex | done | Rebuilt the served UI after cleanup and reran the authoritative gates: notification-hub 21/21, vrooli-bridge 20/20, and vrooli-events 21/21 passed; the durable baseline comparison remains not-comparable only because its Notification Hub member has no recorded phase result. |
| 2026-08-18 | codex | blocked | Recorded the remaining operator/platform acceptance blocker: the managed public origin lacks Cloudflare credentials/DNS resolution, and real iPhone plus paired-Mac delivery evidence is still required for OT-P0-001 and OT-P0-015. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
