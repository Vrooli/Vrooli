# Maintain documentation without losing knowledge

Use `knowledge-observatory-maintenance` for cleanup, consolidation, correction,
reorganization, artifact promotion, and retirement. Use `documentation-health` for
placement and traceability rules. Use the [usage guide](getting-started.md) for
source authority, OS applicability, and the single Memory learning loop.

## Prepare a document family

Select sources by the reader task. Include known consumers when choosing the scan
root. For example, consolidating two setup guides must preserve prerequisites,
portable commands, OS-specific exceptions, recovery steps, and accepted rationale.
A generated HTML comparison may hold a useful explanation without being policy.

Run the existing program:

```text
program-runtime library run knowledge-observatory.prepare-change --input 'paths=["scenarios/knowledge-observatory/docs/guides/getting-started.md","scenarios/knowledge-observatory/docs/reference/cli-commands.md"]' --input base_path=scenarios/knowledge-observatory --input 'intent=Consolidate current agent CLI usage and correct obsolete command guidance' --json
```

The response includes observed source hashes, bounded excerpts, reference candidates,
related sources, a `reference_baseline`, and one `dispositions` row per source.
Missing editorial decisions remain unresolved. Program `ok` means collection ran;
it does not mean that the proposed edits are ready or authorized.

For a larger candidate set, use the read-only inventory before selecting sources:

```text
knowledge-observatory inventory --roots=docs,scenarios --max-files=100
git-control-tower repo blame scenarios/knowledge-observatory/docs/guides/document-maintenance.md --enrich --json
knowledge-observatory proposals --input=inventory.json
```

If inventory returns an HTTP error, report the owner service failure and inspect
the selected source files directly. An unavailable inventory does not establish
that no candidates exist. See the [tracked maintenance availability issue](../internal/PROBLEMS.md#maintenance-inventory-availability--2026-09-07).

Inventory artifact classes and portability signals are hypotheses. Git author
identity is not maintenance ownership. Exact-content provenance requires a full
content digest and matching commit; path overlap, run IDs, and trailers remain
downgraded. Keep private evidence redacted and route only owner-authorized,
revision-pinned proposals. `knowledge-observatory route` defaults to a
dry-run and returns an idempotent receipt; Plan Manager owns the eventual artifact
operation.

Read remaining pages with `knowledge-base inspect --offset` and the same
`--expected-sha256`. Preparation reads only the first 2,000 characters per source.
Keep the exact preparation response in the assigned plan artifact location when
there is a plan. Do not add an evidence/report directory to documentation.

Supply reviewed `dispositions` on a subsequent preparation when useful. Each row
has this shape (illustrative repository paths):

```json
{
  "source": "docs/old-setup.md",
  "action": "consolidate",
  "targets": ["docs/guides/setup.md"],
  "reason": "Keep one current setup procedure while preserving the OS qualification",
  "preserve": ["Resource prerequisites", "Windows qualification", "Failure recovery"],
  "decision": "resolved",
  "evidence_refs": ["path:docs/old-setup.md#prerequisites"]
}
```

Actions are `correct`, `consolidate`, `promote`, `relocate`, `retain`, and `retire`.
Decisions supplied by callers are `proposed` or `resolved`. A resolved decision is
an attributed editorial assertion, not a policy grant. Preparation adds the source
hash. Use at most eight source dispositions, eight destinations per row, and twelve
preservation obligations/evidence references per row. Consolidate, promote, and
relocate require a destination. Retirement with no destination needs an explicit
preservation rationale, such as demonstrated redundancy and no remaining evidence
obligation.

## Review and apply

For each important knowledge unit, name its source revision and final destination.
Retain useful exceptions, rejected-alternative rationale, and unresolved issues.
Confirm canonical ownership from the manifest and owner contract. Distinguish
outdated implementation descriptions from intended requirements that code violates.

Recheck the preparation hashes immediately before editing. Apply the authorized
changes through the source owner's workflow. Update links, heading references,
manifest entries, aliases, navigation, and supersession declarations together.
Plan Manager retains ownership of artifact placement and closeout.

## Verify the result

`knowledge-observatory.verify-change` keeps its original required inputs:
`expectations` (one to eight post-edit path/SHA-256 pairs, optional knowledge_status),
`queries` (one to three query/expected_paths cases), and `base_path`.
Its optional maintenance inputs are:

| Input | What it proves or supplies |
|---|---|
| `absent_paths` | Up to eight old paths must be absent. A typed missing observation distinguishes absence from read errors. |
| `queries[].forbidden_paths` | Old sources must not appear in the returned top-five results for that case. This is not proof of absence from the entire index. |
| `preservation` | Up to twelve revision-pinned exact excerpts plus caller editorial review and evidence references. |
| `reference_baseline` | The unchanged, complete preparation baseline using identical base_path, links/refs checks, and external-link policy. |

Each preservation row has `id`, `path`, character `offset` (default zero), `excerpt`,
`review` (`preserved` or `unresolved`), and `evidence_ref`. Path must be in the
post-edit expectations. Excerpts are at most 1,000 characters and must match exactly
at the selected offset. Supply an excerpt that includes the critical qualification,
not just a heading. A changed revision fails before reading the excerpt.

For a move, expect the destination revision, list the old path in `absent_paths`,
exclude it from representative retrieval cases, and verify the preserved content.
For a retained supersession stub, expect the old source's new revision and
`knowledge_status` instead of absence; inspect its successor declaration as part
of the editorial review. Absence alone cannot prove preservation.

Reference comparison retains full owner finding identity. Existing identical debt
is reported as retained; new findings fail; resolved counts are reported only for
complete reads. Both baseline finding arrays are limited to 100. Incomplete or
mismatched baselines are rejected. Without a baseline, zero reference findings are
required. Never replace a pre-edit baseline with post-edit findings to manufacture
a pass. A shifted line number may conservatively count as a new finding.

`editorial_review_complete` reports caller review declarations. `task_success_verified`
remains false: the agent must compare all obligations and answer the reviewed reader
questions from final sources. An unresolved preservation review returns `partial`
even when its mechanical checks pass. A missing excerpt, remaining old file, stale
retrieval result, or new reference finding fails verification.

## Coverage and limits

Review finds exact duplicates and reference candidates in supported prose, common
source-code, and configuration file types. It recognizes inline Markdown links,
link definitions, HTML href/src, DOC/CODE markers, and typed path references.
Definitions and examples may be illustrative. Dynamic references, arbitrary
manifest path fields, heading validity, external consumers, and semantic overlap
need owner checks. The separate documentation-health validator covers its own
reference and contract rules; these surfaces do not promise identical coverage.

Review prunes `.git`, dependency/build directories, generated `gen` directories,
and `.vrooli` subtrees unless one is the explicit scan root. Account for those
consumers separately before retiring a source. Scans remain bounded and are not a
transaction over a document family. Reinspect
live sources before relying on indexed excerpts. A changed corpus or retrieval
mode cannot establish a comparable before/after improvement.

The provider's program fixtures exercise a portable authored source with intended
revisions, a deliberately absent old path, and preserved content. Behavioral tests
also check missing claims, unresolved review, stale retrieval, unretired paths,
incompatible baselines, and new versus retained debt. These prove workflow mechanics;
real maintenance outcomes must be measured separately in Memory.

Keep the observed source revisions and the authorizing task reference with the
preparation baseline so a later reviewer can identify the change it precedes.
