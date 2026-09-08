# Portal Everywhere planning evidence

Authoring date: 2026-09-05. Repository: /home/matthalloran8/Vrooli.

This directory preserves planning sources and authoring inputs. The authoritative plan is the finalized Plan Manager record, and its markdown is rendered by that owner. These files do not claim implementation or live product validation.

| Artifact | Purpose |
|---|---|
| conversation-context.md | User intent, design evolution, rejected alternatives, authority boundaries. |
| technical-decisions.md | Detailed architecture, diagrams, example contracts/configuration, performance budgets. |
| research-notes.md | Primary-source references and bounded paraphrases. |
| research-sources.json | Machine-readable URL/provenance catalog. |
| source-manifest.json | Current source locations, immutable local copies, hashes, and capture time. |
| source-snapshots/ | Selected local source/doc snapshots; do not edit these as implementation. |
| related-plans.json | Related owner records captured for reuse and status interpretation. |
| acceptance-cases.json | Required case identities and Gherkin outcomes. |
| acceptance-corpus.md | Expanded case specification. |
| acceptance-ledger.template.json | Deliberately pending execution ledger template. |
| verify-evidence-ledger.py | Offline completeness check; never substitutes for producer validation. |
| authoring-inputs/ | Exact field submissions used by the guided authoring runtime. |
| phase-inputs.json | Phase ordering, titles, owners, dependencies, and input file paths. |
| authoring-validation.json | Owner plan-structure validation result, written after validation. |
| preservation-audit.md | Material user-intent mapping and plan review findings. |

## Execution evidence destination

Create /home/matthalloran8/Vrooli/scenarios/portal/docs/internal/plans/artifacts/portal-everywhere-20260905/execution-evidence/ during implementation. Store artifact references and redacted evidence there. Keep private screenshots and credentials in their authoritative artifact/secret stores. The completion ledger uses absolute local artifact paths plus producer receipt IDs.

## Handoff

Read the finalized plan through Plan Manager. Read implementation-plan-execution before beginning. Capture fresh baselines at execution start. Do not rerun historical implementation from source snapshots or claim the pending ledger passed.
