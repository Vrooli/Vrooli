# Progress — Document Manager

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |

| 2026-08-05 | claude | done | **Scenario re-created from scratch.** Generated from `react-vite` v1.6.5 with the `vrooli-default` design kit, replacing three retired scenarios (`document-manager`, `secure-document-processing`, `data-structurer` — see [`DECISIONS.md`](DECISIONS.md)). The prior `document-manager` tree is parked at `scenarios/document-manager-retired/` pending removal. Charter is local-first document ingestion and understanding: tiered parsing, citable anchors, sensitivity-aware routing, and a per-document custody receipt. Generation with `--run-hooks` failed on the template's own golangci-lint post-hook (unused `fileRootPath` seam); worked around with `//nolint:unused` and filed upstream as scenario-qa `knw-1785974545127093568`. Remaining five post-hooks run by hand, all green. Orientation gates 0–5 complete: scaffold healthy on first boot, PRD authored through the `business-health` wizard (business phase L0 → L3, "PRD clean"), 53 requirements across three modules (`DOC-P0-001`…`DOC-P2-009`) validating PASSED with zero findings, eight-domain map written, dependency decisions recorded, design adaptation noted. Docs filled: `DOMAINS.md`, `DATA.md`, `FLOWS.md`, `INTEGRATIONS.md`, `MONETIZATION.md`, `GO-TO-MARKET.md`, `DECISIONS.md`. No product code written — this is a documentation-first base. Known upstream blockers recorded in [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md): AI Gateway has no vision role (blocks tier-3), `RouteEvidence` has no caller correlation key (weakens the attestation claim), OpenRouter embeddings unverified, and `vrooli-memory` has no scope CRUD yet. |

| 2026-08-06 | claude | done | **Boundary correction: this scenario is the ledger's sibling, not its ingestion half.** A design review found the 08-05 charter had the storage boundary wrong in a way that would have been expensive after code existed. The ledger's defining behavior is compaction — unbounded storage, bounded ambient attention — which is right for a stream of findings and wrong for a shelf of documents: a source has no frontier, and a page of source text must never reach an agent's session context. Corrected: documents never enter the ledger; this scenario owns its own embeddings and its own retrieval; neither scenario depends on the other; a consumer joins them and `search-hub` federates. Seven decisions added and two superseded in [`DECISIONS.md`](DECISIONS.md). New ninth domain `retrieval` (the old "no search endpoint" rule had hidden the gap by forbidding the capability). Targets moved: ledger handoff P0→P1 (`OT-P1-020`, `OT-P1-023`), search-hub P1→P0 (`OT-P0-018`); four new P0s for retrieval, privacy-filtered retrieval, collection-level privacy inheritance and the gateway-request choke point; `OT-P0-019` reissued for anchor kinds. **This removes every P0 dependency on ledger scope CRUD, so the engine extraction is now off this scenario's critical path.** Also landed from the same review: two anchor kinds (`geometric` durable by construction, `logical` needing alignment maps) resolving a contradiction between `OT-P0-009`'s guarantee and parse outputs being prunable; a single `internal/gatewayreq` construction site with an AST check, because gateway fail-closed attaches to the *profile* not the privacy class and the guarantee actually lives here; two new load-bearing seams in [`SEAMS.md`](SEAMS.md). Requirements now 58 across three modules (26 P0, 23 P1, 9 P2), `business-health validate` PASSED with zero findings and full `OT ↔ DOC` traceability. Experience specs replaced: the template's dashboard/notes/settings gave way to the three real surfaces (Corpus, Reader, Receipt) plus an `ingest-to-citation` journey, all validating against `scenario-experience-spec/v1`. Four entries added to [`PROBLEMS.md`](PROBLEMS.md), including the removed BAS cases and the `storage-manager` append-only prune trap. Still no product code. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
