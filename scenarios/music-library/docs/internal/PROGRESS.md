# Progress — Music Library

Historical details: `plan-artifacts/docs-html-progress-20260908/scenarios/music-library/docs/internal/PROGRESS.md`. See the [runtime-home retention contract](../../../../docs/reference/storage-retention.md#control-plane-runtime-home-contract) for backup custody and readiness semantics.
Copied July 7 template history was removed from this scenario’s log.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append concise milestones when work lands. Keep detailed execution receipts
with the owning plan.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-11-16 | author | partial | Added immutable release drafts, ordered exact-hash occurrences, catalog-derived take/receipt snapshots, versioned external artist credits, owner-verified cover/lineage seams, append-only rights attestations, digest-pinned operator reviews, REST/CLI/UI workflows, and a revision-bound batch listening room. Draft provenance and rights claims cannot qualify as verified; releases must use liked catalog hashes, exact receipt paths, confirmed use scope, and the owner derivation verifier. Tests prove the exact-hash multi-track contract, missing-proof rejection, rights and review immutability; test adapters do not represent real eligible recordings or operator taste. The six safe judged instrumentals support individual liking and track hashes only; EP continuity, full-song form, actual cover/artist/lineage proof, commercial rights and operator batch judgment remain unproven. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

## 2026-11-15 — MQ4 taste-model foundation

Extended the durable MQ4 taste slice with hash-matched decomposition source artist metadata for display/diversity, explicit `genre:<name>` catalog tags, and compact duration/tempo/key features projected only from the latest operator-confirmed exact-hash decomposition job. Newer analysis jobs and later non-confirming reviews suppress stale projections. The candidate path reads only those compact rows, not full analysis JSON or audio. Recommendations exclude the seed artist and cap known artists and genres at two; score explanations include only exact catalog/decomposition feature provenance. The MQ2 brief names the confirmed-feature boundary and uncertainty limits. A Connect fixture with ten candidates, ten artists, and five explicit genre groups returns ten diverse rows with reconstructable explanations; this is contract evidence from test fixtures, not live catalog yield or taste success. Focused API, preference, decomposition projection and module tests pass. The regular UI type-check still waits on the governed stale package link; temporary source-mapped TypeScript checking passed earlier. No live catalog or MQ1 reference audio was accessed. See the music-quality MQ4 checkpoint for limits and gates.
