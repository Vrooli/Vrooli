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
- **Explorer**: browse scenario docs, tree warnings, and health summary; open document content with code, preview, or split rendering.
- **Documentation reset**: use `knowledge-observatory docs reset` for manifest-declared append logs; reset is not part of the current Explorer viewer.
- **Graph**: explore relationships around a center concept.
- **Metrics**: review coherence/freshness/redundancy scores.

[CODE: ui/src/surfaces/dashboard/DashboardPage.tsx]
[CODE: ui/src/surfaces/search/SearchPage.tsx]
[CODE: ui/src/surfaces/explorer/ExplorerPage.tsx]
[CODE: ui/src/surfaces/explorer/components/DocViewer.tsx]
[CODE: ui/src/shared/controllers/viewerController.ts]
[CODE: ui/src/App.tsx]
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

For Connect method paths, request fields, revision checks, and bounded review
semantics, use the canonical [Knowledge Base API contract](../reference/api-endpoints.md#knowledge-base-connect-service). The CLI calls these same typed read operations and is the portable terminal workflow.

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

For cleanup, consolidation, moves, and retirement, use the
[maintenance workflow](document-maintenance.md) and `knowledge-observatory-maintenance`.

The scenario programs, inputs, purposes, and output signals are generated from the
authoritative Program Runtime contracts. After changing those contracts, run
`python3 scenarios/knowledge-observatory/tools/generate_program_table.py --write`;
use `--check` in validation to detect drift.

<!-- BEGIN GENERATED PROGRAM TABLE source=.vrooli/program-runtime/*.json source_sha256=fc454bb013c975e6edd7d105eac04ecae245fce28abb69d33ae0a5d4950511cd block_sha256=10e188070fccb88906c4e294dadd2d0758d450ecc530357d0a30fa235aaf6f92 -->
| Program | Contract inputs | Purpose | Output signals |
|---|---|---|---|
| `knowledge-observatory.candidate-inventory` | `paths` (array, required), `base_path` (string, required), `intent` (string, optional), `max_files` (integer, default 100) | Produce bounded, provenance-aware documentation candidates for owner review without editing source files. | **candidates**: Bounded source candidates and owner hints. |
| `knowledge-observatory.candidate-proposals` | `candidates` (array, required), `reader_task` (string, optional) | Turn bounded candidate observations into review-only disposition proposals. | **proposals**: Bounded review-only proposals and count. |
| `knowledge-observatory.collect-context` | `query` (string, required), `scope` (string, default "global"), `target` (string, optional), `mode` (string, default "auto"), `limit` (integer, default 3), `follow_references` (integer, default 1) | Retrieve and inspect bounded task context with source revisions, applicability, and explicit evidence gaps. | **documents**: At most 7 inspected excerpts of 3000 characters with 12 references each.; **gaps**: Explicit incompleteness and editorial review needs.; **capture_required**: Caller captures once; task_success_verified is always false. |
| `knowledge-observatory.learning-read` | `from` (string, optional), `to` (string, optional), `operation` (string, optional), `context_key` (string, optional) | Read comparable knowledge-task outcomes and effort from the existing Memory learning projection. | **rows**: Rows with full cohort identities; multiple cohorts require operation/context selectors and remain unreliable until narrowed.; **reliability**: Shared Memory reliability, scan counters, and cohort truncation; status and resolved inputs are preserved. |
| `knowledge-observatory.prepare-change` | `paths` (array, required), `base_path` (string, required), `intent` (string, required), `max_files` (integer, default 300), `dispositions` (array, optional) | Prepare documentation maintenance: cleanup, consolidation, reorganization and artifact promotion with source dispositions, preservation obligations, revision preconditions and a scoped reference baseline. | **preconditions**: Exact source byte hashes; recheck before caller writes.; **observations**: At most 100 backlink, duplicate, missing-reference, or portability cues.; **decisions_required**: Editorial decisions; never an automatic deletion instruction.; **editorial_review_complete**: Caller-supplied editorial review status; never proof of task completion.; **dispositions**: Selected source/action/targets/reason/preserve/decision/evidence_refs with observed source hashes; missing decisions remain unresolved.; **reference_baseline**: Complete bounded owner findings and exact selectors for before/after comparison. |
| `knowledge-observatory.setpoint-read` | `base_path` (string, default "scenarios/knowledge-observatory/docs"), `eval_run_id` (string, optional) | Read index availability, scoped documentation findings, a selected immutable retrieval evaluation, and routes to comparable usage evidence. | **rows**: Index availability target true; scoped reference findings target zero; retrieval and learning comparisons require comparable windows.; **capture_required**: Caller records improve-cycle work evidence. |
| `knowledge-observatory.verify-change` | `expectations` (array, required), `queries` (array, required), `base_path` (string, required), `mode` (string, default "auto"), `absent_paths` (array, optional), `preservation` (array, optional), `reference_baseline` (object, optional) | Verify documentation cleanup, consolidation, moves and retirement: intended revisions, absent paths, preserved excerpts with editorial evidence, retrieval expectations and reference regressions. | **checks**: Explicit revision, retrieval, and reference checks; task_success_verified remains false.; **reference_findings**: Bounded sample with full count; caller distinguishes new failures from pre-existing debt.; **editorial_review_complete**: Caller-supplied editorial review status; never proof of task completion.; **reference_delta**: Full finding identity comparison with identical complete baseline selectors; known debt remains visible. |
<!-- END GENERATED PROGRAM TABLE -->

Use `program-runtime library run knowledge-observatory.<program> --input key=value`.
The command's input parser accepts structured JSON values for arrays. Inspect the
program contract for exact input shapes and ceilings. API bounds are eight selected
sources, 1 MiB per readable UTF-8 source, 12,000 characters per inspection page,
128 references per source, 1,000 scanned files, 10,000 walked entries and 100 review
observations. Programs narrow these further to fit a 64 KiB response. Every scan
reports truncation; reference candidate extraction covers inline Markdown links, link definitions,
HTML href/src, DOC/CODE markers and typed path references, with additional
source-code/configuration consumers. Dynamic references and arbitrary manifest
paths still require owner checks.

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
