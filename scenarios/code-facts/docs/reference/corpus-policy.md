# Corpus Policy

Policy version: `code-facts-corpus-v1`

Code Facts maintains one authoritative catalog for the governed `scenarios/`,
`packages/`, `resources/`, `internal/`, and `cmd/` roots. Catalog identity is a
normalized repository-relative path plus a content hash. Derived document
identity is a language-qualified symbol identity when an analyzer supplies one;
otherwise it is a content anchor derived from its semantic owner and normalized
declaration signature. Line ranges are mutable provenance and never identity.

## Roles

| Role | Default retrieval policy | Semantic-card policy | Authority |
|---|---|---|---|
| `implementation` | Include | Eligible for meaningful declarations and file-purpose cards | Canonical implementation evidence |
| `contract` | Include | Eligible for service, RPC, message, endpoint, and policy cards | Canonical declared contract evidence |
| `generated_alias` | Include compact contract aliases; exclude generated source bodies | Exclude implementation bodies | Alias evidence linked to its contract |
| `test` | Include when requested; demote by default | Exclude by default; a future evaluated policy may opt in named behavioral intent | Behavioral evidence, not production authority |
| `fixture` | Exclude by default | Exclude | Test input only |
| `documentation` | Exclude from the code leaf; allow explicit scope | Exclude from code cards | Explanatory evidence only |
| `transient` | Exclude | Exclude | No authority |

Generated protobuf verification trees, dependency/build output, VCS metadata,
editor state, coverage output, caches, and vendored dependency bodies are
`transient` and do not enter the catalog. A generated client committed under
`packages/proto/gen/` is classified as `generated_alias` for inventory, but its
source body is not searchable. Descriptor extraction creates compact method
aliases that point to the `.proto` contract instead.

## Language Capabilities

| Language | Catalog | Lexical | Declarations | Semantic cards | Graph | Contract proof |
|---|---|---|---|---|---|---|
| Go | Required | Required | Analyzer-backed | Evaluated symbols and file purpose | Analyzer-backed | Imports, calls, routes |
| TypeScript/TSX | Required | Required | Analyzer-backed | Evaluated symbols and file purpose | Analyzer-backed | Imports, calls, routes |
| Protobuf | Required | Required | Descriptor image plus source join | Service, RPC, message, endpoint | Descriptor relationships | Descriptor digest and `.proto` provenance |
| Shell/JSON/YAML/Markdown | Required when governed | Path/content exact lookup | File-level only | Disabled by default | Unsupported | Metadata provenance only |

## Evidence And Freshness

Every returned item carries the active catalog generation, source hash, current
range, role, scope, retrieval regime, score components, and analyzer provenance.
`proven`, `missing`, `contradicted`, `unsupported`, and `unknown` describe the
relationship evidence only. Retrieval score never changes proof status.

SQLite owns catalog identity, role, scope, freshness, and the active generation.
FTS, vector, graph, and contract candidates must match that active generation
and source hash before they can reach a response. When optional retrieval stages
are unavailable, lexical search remains available and the response names the
degraded stages.

## Semantic Card Policy

`code-card-v1` admits only file-purpose, symbol, contract, route, persistence,
and relationship documents whose role is `implementation`, `contract`, or a
compact `generated_alias`. It rejects fixture, test, documentation, transient,
and generated implementation bodies. Display evidence remains source-backed;
embedding text is a separate bounded composition of kind, title, path, scope,
contract, aliases, and evidence. Each vector point uses the stable card ID and
carries source hash, generation, embedding hash, policy, model, role, scope,
kind, and display text.

The selected profile is `nomic-embed-text:latest`, 768 dimensions, cosine
normalization, on-disk Qdrant vectors and payloads, and scalar quantization.
The installed-host bake-off compared 384 and 768 dimensions across five
implementation, contract, routing, persistence, and cache queries. Both
profiles achieved recall@5 1.00 and MRR@3 1.00 with every expected card at
rank one. The full dimension remains selected because it is the governed
Ollama default and the projected 80,000-card profile remains below 1.5 GiB.
The durable report is
`api/internal/retrieval/testdata/model-bakeoff-v1.json`; its limitation field
requires the final full-corpus provider evaluation before cutover.

## Evaluation And Scale Fixtures

`api/internal/facts/testdata/retrieval-eval-v1.json` contains reviewed query
categories and repository-relative authoritative locators. The fixture does not
encode line numbers. `scale_multiplier: 3` defines the deterministic scale
fixture: each source document is repeated in three isolated namespaces while
content, role distribution, and expected-query mapping remain unchanged.

Cutover requires no exact-category regression, direct recall@5 of at least 95%,
MRR@3 of at least 85%, exact p95 at most 100 ms on the current corpus and
200 ms at three times scale, and hybrid p95 at most 500 ms and 750 ms
respectively. Old and new engines run against the same fixture; unexplained
accuracy, provenance, freshness, latency, CPU, memory, or storage regressions
block promotion.

## Execution-Start Inventory

The execution-start inventory used the initial v1 path-role analysis over all
non-ignored files under the governed roots. It includes transient observations
for baseline diagnosis. The production catalog drops those paths.

| Role | Files | Bytes | Default searchable |
|---|---:|---:|---|
| `implementation` | 22,752 | 121,437,126 | yes |
| `contract` | 1,733 | 7,335,458 | yes |
| `generated_alias` | 4,724 | 63,572,270 | exact-only / demoted |
| `test` | 12,123 | 62,311,513 | no |
| `fixture` | 504 | 3,979,181 | no |
| `documentation` | 4,229 | 75,446,326 | no |
| `transient` | 5,665 | 90,840,179 | no |

The initial default lexical population is therefore 29,209 files and
192,344,854 source bytes before document extraction. These figures describe
the dirty execution-start worktree and are baseline context, not an invariant
that future repository revisions must match.

The Phase 4 repository-aware catalog pass ran after the policy became
executable. It streamed tracked and non-ignored regular files, skipped dirty
tracked deletions and symlinked directories, and dropped transient trees. The
pass cataloged 53,445 files: 30,837 implementation, 840 contract, 5,203
generated-alias inventory rows, 11,916 test, 495 fixture, and 4,154
documentation files. Governed-root counts were: scenarios 44,977; packages
6,594; resources 853; root internal 1,017; and cmd 4. These are reproducible
snapshot evidence, not permanent repository-size assertions.
