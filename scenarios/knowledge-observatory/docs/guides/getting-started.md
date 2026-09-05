# Getting Started Guide

This guide walks through the UI, CLI, and API to confirm Knowledge Observatory is working end to end.

## End-to-End Flow
1. Start the scenario (`make start`).
2. Confirm API health from the UI dashboard or CLI.
3. Select an existing repository document.
4. Run a search and inspect metadata.
5. Explore graph and metrics panels.

## UI Walkthrough
- **Dashboard**: run quick search, confirm knowledge + documentation health, review activity feed.
- **Search**: select semantic, file, text, unified, or deep search modes.
- **Explorer**: browse scenario docs, tree warnings, health summary, and healing controls.
- **Viewer**: open document content with preview/mermaid rendering and reset support.
- **Graph**: explore relationships around a center concept.
- **Metrics**: review coherence/freshness/redundancy scores.

[CODE: ui/src/surfaces/dashboard/DashboardPage.tsx]
[CODE: ui/src/surfaces/search/SearchPage.tsx]
[CODE: ui/src/surfaces/explorer/ExplorerPage.tsx]
[CODE: ui/src/surfaces/viewer/ViewerPage.tsx]
[CODE: ui/src/surfaces/graph/GraphPage.tsx]
[CODE: ui/src/surfaces/metrics/MetricsPage.tsx]

## CLI Walkthrough

```text
knowledge-observatory status
knowledge-observatory knowledge-base search --query "architecture" --scope scenario --target knowledge-observatory --limit 5 --json
knowledge-observatory knowledge-base inspect --path scenarios/knowledge-observatory/README.md --limit 3000 --json
knowledge-observatory knowledge-base health --scope path-exact --path scenarios/knowledge-observatory/docs/guides --checks links,refs --skip-external-links --json
```

These commands read existing sources. Repository documentation is discovered by
the documentation indexer; there is no `knowledge-observatory ingest` command.

[CODE: cli/app.go]
[CODE: cli/domains/knowledgebase/register.go]

## API Walkthrough

Resolve the running port with `vrooli scenario port knowledge-observatory API_PORT`.
Send JSON with `Content-Type: application/json` to the Connect endpoint
`/knowledge_observatory.v1.KnowledgeBaseService/SearchDocuments`:

```json
{"query":"architecture","scope":"scenario","target":"knowledge-observatory","limit":5}
```

Use `InspectDocument` on the same service with a repository-relative `path` and
bounded `limit`. Read `sha256`, `metadata`, and `truncated` before using a result.
The CLI uses these same typed operations and avoids shell-specific HTTP examples.

[CODE: api/knowledge_base.go]

## Related Reference
- [DOC: docs/reference/api-endpoints.md]
- [DOC: docs/reference/cli-commands.md]

## Agent knowledge workflows

Load `knowledge-observatory` through Prompt Manager for ordinary knowledge tasks;
load `knowledge-observatory-improve` when improving this capability. Skills select
routes and assess evidence. Programs perform bounded composition. The scenario
owns stable source inspection and retrieval operations; accepted facts remain
with document owners, plan artifacts with Plan Manager, and usage evidence with
Vrooli Memory.

The `knowledge-base` CLI group exposes governed read operations: `search`,
`inspect`, `review`, `status`, and `health`. Its manifest binds to
KnowledgeBaseService and the existing documentation-health service. Each command
supports JSON output. `inspect` returns a SHA-256 of exact file bytes, a character
page, manifest metadata, and bounded references. `expected-sha256` rejects a
changed source before returning content. Paths are repository-relative with `/`
separators; traversal, drive/UNC paths and symlink escape are rejected.

The scenario programs are:

| Program | Inputs and evidence |
|---|---|
| `collect-context` | Task query, scope/target, retrieval mode, up to five sources and two followed references; returns bounded excerpts, source hashes and explicit gaps. |
| `prepare-change` | One to eight source paths, intent, scan directory and file budget; returns revision preconditions, duplicate/backlink findings, portability cues, related sources and editorial decisions. |
| `verify-change` | Intended post-edit source hashes, up to three reviewed query/expected-path cases, and scan directory; checks source revisions, retrieval and local/reference health. |
| `learning-read` | Fixed from/to window, operation and context_key; returns comparable outcome/effort cohorts, with missing or incomparable observations unbanded. |
| `setpoint-read` | Scan directory and optional immutable retrieval eval run; reads index, reference and binding health and points to usage-learning/external-friction evidence. |

Use `program-runtime library run knowledge-observatory.<program> --input key=value`.
The command's input parser accepts structured JSON values for arrays. Inspect the
program contract for exact input shapes and ceilings. API bounds are eight selected
sources, 1 MiB per readable UTF-8 source, 12,000 characters per inspection page,
128 references per source, 1,000 scanned files, 10,000 walked entries and 100 review
observations. Programs narrow these further to fit a 64 KiB response. Every scan
reports truncation; reference extraction currently covers inline Markdown links
and typed path references, not every Markdown/HTML reference syntax.

These programs do not edit, relocate, delete, or archive documents. A caller may
apply reviewed changes through the source owner's authorized workflow after
rechecking the preparation hashes. Verification accepts intended post-change
hashes; it is not a substitute for checking preconditions before a write. Source
reads are individual snapshots, not a transaction over an entire document family.
No checkout, reset, clean, branch switch or destructive filesystem operation is
part of these workflows.

### Source authority and applicability metadata

A document manifest may declare `knowledgeStatus` (`unknown`, `accepted`, `draft`,
`historical`, `superseded`, `supplemental`), `operatingSystems` (intended scope),
`verifiedOperatingSystems` (evidence-backed support), `machineScope`, and
`supersededBy` (successor paths). Existing `aliases`, `ownerSkills`, `canonicalFor`
and `maturity` remain supported. None of these optional declarations is inferred
from recency or search rank. Unregistered sources remain searchable with unknown
status; plan-like directories default to supplemental. HTML is plan material and
is available for explicit inspection, not ordinary semantic indexing.

Metadata travels with semantic chunks and the Search Hub provider mapping. Search
responses overlay live manifest declarations, identified by knowledge_metadata_basis,
so pending reconciliation cannot retain a superseded authority label. Chunk hashes
still describe indexed content; live manifest hashes describe the declaration. Text
fallback uses the same repository-relative document identity and resolves live
manifest declarations. Metadata changes invalidate source indexing even when file
bytes do not change. An index can lag live sources; inspect before acting.

### OS and machine independence

Vrooli intends OS and machine independence. Keep intended support distinct from
verified Linux/macOS/Windows support and from observations on a particular machine.
Review cues such as `/proc/`, `systemctl`, or a named host require context; an
occurrence is not automatically a defect. Generated multi-platform bindings or
successful cross-compilation do not establish runtime support on those systems.

### Learning and knowledge preservation

The usage caller recalls advice once and writes one completed attempt through
Vrooli Memory's learning API in scope `knowledge-observatory-usage`. Programs never
copy the corpus into memory or record success before the caller verifies the task.
Unknown, failed, unavailable and verified-success outcomes remain distinct. Keep
source hashes in evidence references and comparable task attributes in context_key.
Synthetic/test attempts do not establish an operator baseline. The learning read
uses Memory's existing Source Ledger projection; it creates no second journal.

Measure representative source retrieval and supported task outcomes, not files
removed. The provider-owned evaluation remains `knowledge-observatory.docs.starter`;
direct and federated runs must be compared separately with their configuration,
case denominator, degraded state and immutable run IDs. Improvement requires
comparable operator cohorts; initial learning targets remain null.

### Supplemental artifacts and documentation authority

Generated plan HTML, reports, and evidence are supplements, not official
documentation. Inspect them when they may contain useful knowledge. Promote unique
accepted knowledge into an existing authoritative document, retain necessary
rationale/evidence with the plan owner, and retire redundancy only after
preservation and references are verified. Search results, exact duplicates and
successful program execution do not authorize retirement. Semantic contradiction
resolution remains an editorial decision.

Memory setup: if the owner reports that `knowledge-observatory-usage` is not
provisioned, create it once with `vrooli-memory scopes create
knowledge-observatory-usage --label "Knowledge Observatory agent usage"`.
Keep the declared default budgets; a read program never provisions or mutates a scope.

Verification uses documentation health scope `path-exact`: generic checks stay
inside the requested file/directory. Existing scope `path` retains its legacy
behavior of promoting paths inside a scenario to the full scenario.
