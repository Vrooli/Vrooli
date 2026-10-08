# Progress — Architecture Cartographer

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/architecture-cartographer/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-18 | Codex | Intent alignment rollout cleanup. | Archived |
| 2026-06-18 | Codex | Intent alignment Phase 3 seam landed. | Archived |
| 2026-05-25 | Claude (Opus 4.7) | CLI conflict-workbench wiring (OT-P0-006 / OT-P0-010). | Archived |
| 2026-05-22 | Claude (Opus 4.7) | UI Phases 6–10 — apply / manifest / analytics / settings extension / quality gates. | Archived |
| 2026-05-21 | Claude (Opus 4.7) | Phases 11–12 — fixtures + integration tests + dogfood closure. | Archived |
| 2026-05-21 | Claude (Opus 4.7) | Phases 7–10 — analytics + apply + signals Connect handlers, registry wiring, codegen. | Archived |
| 2026-05-21 | Claude (Opus 4.7) | Phase 6 — conflicts service orchestration + Connect handler + flow contract. | Archived |
| 2026-05-21 | Claude (Opus 4.7) | Phase 5 — graph domain Connect surface + production adapter stubs. | Archived |
| 2026-05-21 | Claude (Opus 4.7) | Phase 4 — manifest domain landed end-to-end. | Archived |
| 2026-05-21 | Claude (Opus 4.7) | Scenario charter complete. | Archived |
| 2026-05-23 | Claude (Opus 4.7) | Dependency unblock — cartographer's two language-graph dependencies, `go-code-graph` and `typescript-code-graph`, were initialized from the `react-vite` template in dedicated session. | Archived |
| 2026-05-21 | Claude (Opus 4.7) | Phase 3 — day-one detectors + signals landed. | Archived |
| 2026-05-21 | Claude (Opus 4.7) | Phase 2 — foundational packages authored for every product domain under `api/internal/<domain>/`. | Archived |
| 2026-05-21 | Claude (Opus 4.7) | Phase 1 — proto schemas authored for all six product domains (`graph`, `manifest`, `signals`, `conflicts`, `apply`, `analytics`) under `packages/proto/schemas/architecture-cartographer/v1/<domain>/<domain>.proto`. | Archived |
| 2026-05-21 | Claude (Opus 4.7) | Phase 0 — template `notes` residue removed end-to-end: deleted proto schemas + generated go/ts/python trees, `api/handlers/notes/`, `api/internal/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `ui/src/api/notes.*`, `ui/src/pages/NotesPage.tsx`. | Archived |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

## Historical handoff leads retained during condensation

These clauses are preserved from dated entries; later work may have superseded
them. They do not assert that an old failure or gap still exists.

- **2026-06-18 — done**: `intent_alignment` now wires a `Matcher` strategy seam with deterministic `LexicalMatcher` as production default and explicit off-by-default `EmbeddingMatcher`/`LLMMatcher` stubs for the deferred semantic tiers.

- **2026-05-22 — done**: **Phase 7 — manifest**: `features/manifest/` ships `ManifestValidationReport.tsx`, `DomainInventory.tsx`, `ManifestView.tsx`, and `controllers/useManifestController.ts` (`useGetManifest`, `useListDomains`, `useValidateManifest`). v0.1 surface is read-only per the proto contract; manifest write paths blocked on backend per `PROBLEMS.md`.

- **2026-05-21 — done**: Loose semantic matching keeps fixtures resilient to cosmetic detector message tweaks; tightening to byte-exact golden comparison is a follow-up.

- **2026-05-21 — done**: **Flow contract**: new `internal/conflicts/flow/flow.json` (schema_version=v6, flow_id=conflict_lifecycle) declares the 7-state machine (detected → assigned → split → resolved → validated → committed; +force_resolved branch) and 16 transitions including ReopenConflict from any non-initial/non-terminal state. flow-verifier wiring stays deferred per docs/internal/PROBLEMS.md; the JSON serves as the readable contract and reviewer pin until the verifier scenario is in tree.

- **2026-05-21 — done**: Registry wiring (modules/registry + main.go) deferred to Phase 10.
