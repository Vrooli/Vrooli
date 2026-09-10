# Flows — Tech Tree Designer

## Purpose Of This Document

Describe existing proto planning and the target documentation-first workflow without confusing draft, approval, application and proof.

## Flow Inventory

| Flow | Current boundary | Target |
|---|---|---|
| Observe | SDA interface graph and planned overlay | Provenance-bearing bounded observed views, explicit partial/stale results |
| Design contracts | Planned scenarios and .proto files | Multi-target artifact and authored-graph proposals |
| Curate ontology | Hierarchy, mappings, coverage/focus | Reviewed revisions; contribution distinct from verified fulfillment |
| Review/apply | Proto validation/materialization | Exact-revision, owner-directed, recoverable general application |
| Experiment | Not qualified by this contract update | Explicitly granted isolated draft experiments (P1) |

## Flow Details

### Bottom-up documentation-first design

1. Start from selected repository paths, new target identities, or a bounded observed neighborhood. Query intended capability context only when useful.
2. Select existing or new scenarios, resources, shared packages and project-level artifacts. The proposal need not reproduce the official graph and may contain proposed corrections or new entities.
3. Retain references and deltas in a separate proposal workspace. Author real proposed PRDs, requirements, docs, experiences, interface schemas and skill/program definitions without publishing them.
4. Validate with artifact owners. Show coverage, unsupported validators, unresolved decisions and expected effects.
5. Produce an immutable revision. Plan Manager stores concise rationale, sequencing and links to that revision, rather than re-describing every artifact.
6. The operator reviews and authorizes the applicable mandate through Swarm. The executing agent consumes the approved target and iterates within that mandate; ordinary in-scope improvements do not each require a new backlog item.

Missing graph coverage does not prevent drafting an otherwise supported artifact.
Retain unknown source relationships and proposal-local corrections explicitly;
do not require a full graph refresh or an ontology visit before creating a draft.
The graph-backed journey in `experience/` is one entry path, not a mandatory wizard.

### Top-down design and bottom-up learning

Curate sectors, capabilities, decomposition and relationships as a revisable intended model. Link proposed/implemented contributions with provenance. Evidence can justify a fulfillment judgment, but a scenario mapping alone cannot. Propose ontology corrections from implementation lessons; review them as authored changes. The civilization-scale horizon guides exploration, not an exhaustive task list or execution grant.

### One apply experience, multiple authoritative effects

1. Resolve the exact revision and selected entries; preview effects and dependencies read-only.
2. Recheck grants, owner support, required validation and relevant base identities. Hold dependent entries when prerequisites fail.
3. Persist operation intent before effects; delegate artifact writes/generation/publication to qualified owners.
4. Apply authorized authored-ontology deltas through TTD's own domain with equivalent conflict checks.
5. Record per-entry receipts and overall complete, partial or conflicted state. Re-derive observed facts from their sources after actual changes; never promote proposed dependencies into observed truth.
6. Resume an interrupted operation with the same identity and reconcile owner receipts before retrying. Recovery or compensation requires applicable authority and must preserve newer unrelated edits.

Application and observation refresh have separate standing. If selected writes
succeed but the source refresh fails, retain the successful owner receipts and
show observation as stale or unavailable. Retry the read without replaying writes.
Neither successful application nor successful refresh establishes capability fulfillment.

### Qualified experiments

A draft can describe code and experiments, but opening it grants no execution. If explicitly authorized, reuse qualified workspace operations and control-plane runtime isolation. Capture exact bytes, runtime identities, databases, network/billing allowances, fixtures and evidence. Experiment success is not permission to apply or deploy.

## State Machines

Target proposal lifecycle: draft → immutable review revision → authorized selection → applying → applied or partial/conflicted. Amendment creates a new revision; approval does not silently follow it. Rejected/abandoned proposals retain pinned reviewed content according to retention policy.

Revision-bound application approval is not a per-edit development gate. After
initial target application, an approved development mandate may authorize code,
documentation and experiment changes without another bundle approval. Protected
target changes and effects outside the mandate still require an amendment. Preserve
both the approved target and subsequent implementation checkpoints.

Per-entry outcomes include pending, held, applied, conflicted, failed and operator-disposed. An operation is not complete while a selected entry has an unresolved outcome. Cancellation stops new permitted work but does not erase effects or receipts. Do not claim global atomicity across owners.

## Maturity Ladder

Contract clarity precedes obligations, evidence and implementation maturity. Each failure/retry state needs deterministic tests and owner-path qualification. Existing proto tests cannot establish generic bundle safety.

## Production Shape

API, CLI and UI share typed behavior. Owner operations and durable results support agent use without UI automation. TTD is not an execution scheduler, authorization system or private sandbox.

## Deferred / Unmodeled Flows

P1 alternatives and experiments follow the safe core. P2 strategic AI and civilization-scale simulation remain optional expansion. Exact retention durations, numerical performance thresholds and remote access guarantees remain qualification decisions.

## Cross-References

- [Architecture](ARCHITECTURE.md)
- [Data](DATA.md)
- [Testing](../internal/TESTING.md)
- [Experiences](../../experience/README.md)
