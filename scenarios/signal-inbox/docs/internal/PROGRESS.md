# Progress — Signal Inbox

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-27 | claude | done | Scenario created from `react-vite` 1.6.5 with the `vrooli-default` design kit, replacing `bookmark-intelligence-hub` (moved to `/tmp/bookmark-intelligence-hub`, implementation discarded per D-004). Authored the full product contract from an operator design workshop: `PRD.md` with 28 operational targets across P0/P1/P2; `requirements/` restructured into `01-must-ship` (14), `02-post-launch` (9), and `03-future` (5) with 1:1 OT traceability and the template's starter `01-foundation` module removed; `DECISIONS.md` recording D-001 to D-025 plus four withdrawn positions; `DOMAINS.md` with six product domains and their acyclic reference order; `DATA.md` with the full ownership, schema, import/export, and retention model; `INTEGRATIONS.md` with the enrichment-delegation boundaries, risk-tier table, search-hub descriptor contract, and consumption contract; `FLOWS.md` inventory; `UI-ARCHITECTURE.md` product surfaces; `PROBLEMS.md` with nine open design gaps. Ported the one salvaged predecessor artifact to `docs/reference/conversation-extraction.md` and registered it in `docs/manifest.json`. Validation: `business-health validate scenario signal-inbox` PASSED; traceability matrix reports 28 rows with every `OT-*` mapped 1:1 to a `SIG-*`. **No implementation exists** — every requirement is `status: planned`, which is accurate rather than pending cleanup. |
| 2026-07-27 | codex | partial | Replaced the `notes` template example with the real `signals` vertical slice: generated SignalsService contract, append-only SQLite journal, content-hash deduplication and tracking-parameter normalization, capture-time annotation rows, Connect handlers, CLI commands, and the URL/text/image capture screen. Image uploads use a BlobStore multipart exception with content-addressed payload references; image bytes stay out of Connect requests. Focused API, CLI, and UI suites pass. Requirements remain planned until their complete validation evidence is registered. |
| 2026-07-27 | codex | partial | Began the enrichment vertical slice with an append-only `signal_enrichment` sidecar (D-027), so URL/image extraction runs after journal append without ever mutating a signal. URL extraction uses a bounded, browser-identifying fetch and rejects client-rendered shell/chrome as zero-content; image OCR delegates through the measured `image-tools analyze ocr <input> --json` contract after materializing only a temporary copy of BlobStore bytes. Focused extraction tests cover zero-content, unavailable delegate, and delegated OCR. |
| 2026-07-27 | codex | partial | Built the Phase 5 categories vertical slice: generated CategoriesService contract, SQLite category/taxonomy/classification storage, runtime create/rename/retire semantics, reserved fallback, append-only advisory proposal/confirmation history, ai-gateway degradation queue, Connect handlers, CLI verbs, and category/confirmation UI. D-028 closes the annotation-record collision by making classification history authoritative and rendering it later through triage. Focused API, CLI, and UI tests pass; requirement status remains evidence-driven until the whole phase validation is registered. |
| 2026-07-27 | codex | partial | Started the Phase 6 triage slice: generated the TriageService contract; added schema-owned current disposition and append-only annotation/outcome-link records; and registered the schema in application startup. Core tests prove invalid transitions are rejected, `dropped` does not alter the immutable journal, and annotations preserve multiple identifier-only outcomes. Connect, CLI, detail UI, and retrieval-facing validation remain unfinished. |
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
