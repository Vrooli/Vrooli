# Asset derivation

Behavioral claims are current only within the specific checks in the register. Other guidance and unverified descriptions below are **design intent**, not claims of current implementation. See the [behavior claim register](../internal/TESTING.md#behavior-claim-register).

The catalog records desired capabilities. Versioned library source records implementations. `component.json` holds stable library identity and lifecycle pointers; authoring operations maintain it. A source stamp identifies the library asset and does not replace its manifest or its catalog declaration.

## Current and target ownership

| Surface | Owner | Enforcing check |
| --- | --- | --- |
| `catalog/assets/` | Authored desired intent | Catalog schema and graph gates |
| `library/**/component.json` | Manifest identity and governed lifecycle operations | `api/internal/components/indexer_test.go` |
| Released version source and stories | Immutable release, changed through a new draft/release | Indexer release-hash checks; `api/internal/components/indexer_restored_test.go` |
| Active draft | `components draft-begin`, `content-set`, `draft-publish` | Authoring and indexing tests in `api/internal/components/` |
| Exact dependency locks | Generated for the owning version; existing release choices remain pinned | `packages/react-component-library/tooling/generate-locks.test.mjs` |
| Package export map and compiled output | Package tooling | `sync-exports.test.mjs`, `build-boundary.test.mjs` in that tooling directory |
| Story contract generated from an executable story module | Story generator where that source format is used | `generate-story-contracts.test.mjs` |
| Source-adjacent application page stories | Authored workspace JSON, no release version | `api/internal/components/page_stories_test.go` |
| Shared visual token values | Authored BaseStyles release; generated consumer CSS/theme and Tokens projection | `node ui/scripts/design-token-generate.mjs --check` |
| Scenario layout tokens | Authored `ui/src/app-tokens.css`, disjoint from shared tokens | Same token check rejects collisions |

`catalog build` coordinates its declared generation stages; it does not replace the package compilation or UI rebuild. Do not delete arbitrary derived-looking files: immutable releases, historical mirrors and locks have retention and recovery contracts. The former claim that one generator recreates every artifact from two authored surfaces was too broad.

## Decisions

Keep intent and observation separate when they carry different meaning and a check compares them. Derive same-meaning copies where the owner supports it. This is design intent outside the enforced boundaries listed above; it is not a claim that the whole historical corpus has converged.

## Repairing a released version after a toolchain change

Use the [asset update loop](../guides/asset-update-flow.md). Recover a missing historical release through materialization; repair its behavior by publishing a successor. Do not refresh hashes to legitimize modified released bytes. Targeted authoring indexing preserves already-recorded unrelated history; full indexing still reports mismatches.

See [token ownership](../reference/token-contract.md) and [checked behavior](../internal/TESTING.md#behavior-claim-register).
