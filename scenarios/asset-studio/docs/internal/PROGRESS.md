# Progress — Asset Studio

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative. The
2026-07-28 entry records design and contract authoring only; no implementation
code exists.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |

| 2026-07-28 | claude | done | Scenario generated from `react-vite` 1.6.5 with the `vrooli-default` design kit. PRD authored (15 P0 / 11 P1 / 4 P2 operational targets). Requirements registry rebuilt as three modules — `01-must-ship`, `02-post-launch`, `03-future` — with 30 requirements, every one linked to a PRD target and declaring intended evidence, all statuses `planned` and no fabricated refs; `vrooli scenario requirements validate asset-studio` reports PASSED with no findings. Concept docs written: DOMAINS (5 product domains plus health, with build order and four rejected domain candidates), DATA (ownership, blob fork with content-desk, reproducibility contract, import idempotency), FLOWS (6 flows, 4 state machines, the render job at L5), INTEGRATIONS (dependency and failure contracts, and the one-way direction with content-desk), ARCHITECTURE (production layering and four invariants). DECISIONS records 14 durable decisions. PROBLEMS records 5 open constraints, including that the conformance comparison is unvalidated. **No implementation code written.** |

| 2026-07-28 | claude | done | Contract amendments from a design review, all pre-implementation. Two refinements to earlier positions: **D-015** pins the identity-block freeze trigger to *release* rather than *render*, so authoring iteration carries no versioning cost; **D-016** has import update an unreferenced head version in place and create a version only when the head is released-referenced, removing catalogue-edit churn. Three additions from a review of the character-consistency product category: **D-017** adds conditioning artifact references to the identity block — schema and provenance at P0 (`ASSET-P0-016`), rendering wiring at P1 (`ASSET-P1-012`) — closing a real gap where the block assumed prose traits alone reproduce an identity, which they do not; **D-018** allows one render job to produce several candidates with operator selection, reusing the existing asset lifecycle and correcting the spend-per-released-artifact model (`ASSET-P0-017`); **D-019** routes regional refinement through image-tools and requires a refined artifact to re-enter conformance rather than inherit its parent's verdict (`ASSET-P1-013`). Registry grew from 30 to 34 requirements across 17 P0 / 13 P1 / 4 P2; `ASSET-P0-002`, `ASSET-P0-003`, `ASSET-P0-007`, and `ASSET-P0-008` were amended rather than duplicated. Validation: `vrooli scenario requirements validate` and `business-health validate scenario` both PASSED with no findings; matrix shows 34 rows with every target traced. **Still no implementation code.** |

| 2026-07-28 | claude | done | Third review pass, still pre-implementation, correcting one factual error and two design gaps. **The rich-media catalogue is empty** — `characters/`, `scenes/`, and `products/` hold a README and a `_template.json` each and zero authored records — so `ASSET-P0-003` would have imported nothing and the P0 slice had no subject. `PROBLEMS.md` previously asserted those records "were authored", which made a corpus problem read as a validation problem; the entry is rewritten. Consequences: **D-021** makes the P0 slice subject a *product* identity authored in the workbench rather than an imported character, which also drops the persona, AI-UGC-disclosure, and `channel-strategy-update` dependencies that sit outside this scenario and are blocked while the marketing team is paused; **`ASSET-P0-018`** adds identity authoring as a declared ingress, a capability D-015 and D-016 both already assumed; and import is re-sequenced after authoring as the migration path. **D-020** adds a `basis` field to every conformance verdict (`reference-sheet`, `reference-image-set`, `conditioning-artifact`, `prose-only`), closing a gap where the release gate was P0 while the reference material that makes it judgeable was P1 — a gate with nothing to compare against passes everything and reads as a control. Also mechanical: decision rows reordered by ID (they ran D-013 → D-017 → D-015 → D-014) and the log preamble corrected to account for D-015 through D-021 and to mark which are refinements versus corrections. ARCHITECTURE gains a *What P0 pays forward* table separating the two schema-shape targets (`ASSET-P0-016`, `ASSET-P0-017`) from slice work, so a plan does not budget columns as features. Registry grew from 34 to 35 requirements across 18 P0 / 13 P1 / 4 P2. **Still no implementation code.** |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
