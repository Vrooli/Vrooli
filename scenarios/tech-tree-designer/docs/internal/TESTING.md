# Testing — Tech Tree Designer

## Purpose Of This Document

Turn the expanded contract into falsifiable evidence. Schema validation proves document shape only; it does not qualify proposal behavior, performance or autonomous execution.

## Current Scaffold

Existing tests cover graph mapping/queries, planned proto storage/validation/materialization and ontology storage/coverage, plus API/CLI/UI behavior. requirements/01-foundation preserves historical evidence. Those results do not establish the September 2026 general proposal target. Newly planned requirements and draft experiences are deliberately unproven.

## Commands

Use repository docs/TESTING.md to select scope. Documentation checks:

```bash
business-health validate scenario tech-tree-designer
vrooli scenario requirements validate tech-tree-designer --json
experience-manager spec validate tech-tree-designer
```

For implementation, discover supported Test Genie phases and run only relevant phases unless certification is requested. Runs are server-owned; wait once with test-genie runs wait --json tech-tree-designer <run-id>, never poll. Client cancellation is not run abortion. Record exact run/revision and do not force-sync statuses green.

## Future Coverage

The following are **planned protocols**, not completed tests or invented evidence references. Requirement validation notes identify the applicable protocol.

| Protocol | Required falsification cases |
|---|---|
| SOURCE | Live vs cached vs unavailable SDA; provenance, partial coverage, unsupported entity kinds; unavailable is not healthy empty |
| VIEWS | Same entity in observed/intended/proposal views; draft corrections do not mutate observed facts; mapping alone is not fulfillment |
| SCOPE | One proposal spans an existing scenario, new scenario, resource, shared package and root docs; include PRD, requirements, experiences, proto and draft skill/program artifacts; start from paths or new identities when graph coverage is absent |
| REVISION | Immutable reviewed bytes survive edits/deleted compute; exact plan links; selected entries and changed permissions/deletions visible |
| APPLY | Relevant base changes conflict; unrelated concurrent edits survive; missing/expired grant denies; generated artifacts route to owner |
| RECOVERY | Duplicate delivery, crash before/after owner effect, lost acknowledgement, unavailable owner, dependency hold and partial result; resume reconciles without duplicate effects; failed observation refresh after successful apply retries only the read |
| ISOLATION | Escaping/symlink paths, overlaps, secrets, oversized content, draft publication, denied network/billing/commit/deploy and revoked grants |
| SCALE | Approved PERFORMANCE cohorts; cold/warm, skew, large artifacts/revisions, concurrent requests, cancellation, retention pins and foreground responsiveness |
| EVIDENCE | Fresh agent discovers target, gaps and governed sensors; missing/stale/inapplicable evidence remains unresolved; Swarm retains acceptance ownership; ordinary in-mandate edits need no new bundle approval, while protected-target amendments do |
| ONTOLOGY | Reviewed top-down amendments, bottom-up contribution, evidence invalidation, unmapped/uncertain capabilities and no exhaustive coverage claim |
| UX | Draft experience journeys, keyboard/text graph alternative, compact layout, error/empty/partial/stale states, conflicts and honest recovery |
| LIFECYCLE | Alternatives, amendment, abandonment, export/import integrity, retention and backup/restore with referenced revisions |
| EXPERIMENT | Explicit grant, isolated ports/database/process identity, exact inputs, effect denials, cleanup and immutable evidence; optional P1 |
| PARITY | Same identities/errors/authorization across API, CLI and UI; no UI-only business rule |
| STRATEGY | Optional suggestions retain source and uncertainty; rejected suggestions have no canonical effect |

### Evidence layers and development proof

Use focused deterministic tests for invariants and adapter contract tests for owner boundaries. Add real owner-path integration tests, not only mocks, for plan references, grants, apply receipts and validation. Run operator journeys against the real UI once implemented, including constrained device/browser conditions. Simulated fixtures must state their fidelity and cannot stand in for unsupported real owner behavior.

The second-target proof is an authorized development mandate through Swarm and Agent Manager: a fresh agent discovers this target, establishes missing sensors, implements bounded work, obtains applicable durable evidence and returns an acceptance-ready result without repeated conversational clarification. This documentation task does not start that mandate.

Record each sensor's owner, discoverable invocation, input cohort, freshness/applicability, result schema and evidence link. The scenario now declares usage/improve skills and a read-only setpoint program. Their presence does not implement product sensors. Do not invent successful readings or duplicate owner workflows locally.

### Outcome evidence inventory

Inventory revision: `ecosystem-design-v1`, 2026-09-09. Row IDs in the setpoint
program are the PRD IDs below. Every product reading is currently `null` with
`pending_telemetry`; no outcome has an implemented acceptance resolver. Targets
remain in the PRD, not derived from current counts. All numeric bands are null
until cohort approval. P1/P2 become required only by explicit mandate selection.

| Outcome | Existing evidence / missing acceptance measurement | Protocol | Repair owner |
| --- | --- | --- | --- |
| OT-P0-001 | Graph/SDA foundation tests; no complete source-validity receipt | SOURCE | graph + SDA |
| OT-P0-002 | Proto planning tests; no revision/cohort-bound acceptance receipt | SCOPE, APPLY | planning |
| OT-P0-003 | No measured three-view isolation | VIEWS | graph + ontology + planning |
| OT-P0-004 | Proto-only inventory is diagnostic; generic artifact proposal absent | SCOPE | planning + artifact owners |
| OT-P0-005 | No immutable reviewed artifact/graph receipt | REVISION | planning + Plan Manager |
| OT-P0-006 | No multi-owner idempotent apply/recovery receipt | APPLY, RECOVERY | planning + workspace/artifact owners + Swarm |
| OT-P0-007 | No generic-draft authority/isolation measurement | ISOLATION | planning + workspace owner + control plane |
| OT-P0-008 | Numeric profile proposed below PERFORMANCE; no accepted scale corpus | SCALE | graph/planning + Test Genie |
| OT-P0-009 | Skills/board setup exists; no fresh-agent development/acceptance proof | EVIDENCE | TTD + Swarm + Agent Manager |
| OT-P1-001 | Ontology foundation tests; no full product acceptance receipt | ONTOLOGY | ontology |
| OT-P1-002 | Draft experience intent; not verified browser journeys | UX | UI + Experience Manager + BAS |
| OT-P1-003 | No contribution-versus-fulfillment evidence join | ONTOLOGY | ontology + evidence owners |
| OT-P1-004 | No qualified retained-alternative lifecycle | LIFECYCLE | planning + storage/workspace owners |
| OT-P1-005 | No qualified isolated draft experiment | EXPERIMENT | workspace owner + control plane |
| OT-P1-006 | Existing typed interfaces; no generic operation parity proof | PARITY | API/CLI/UI |
| OT-P2-001 | Existing SDA adapter tests, not new entity/source coverage | SOURCE | SDA + graph |
| OT-P2-002 | No qualified strategic suggestion evidence | STRATEGY | ontology + inference owner |
| OT-P2-003 | No exhaustive-horizon or simulation evidence; optional vision | STRATEGY, SCALE | ontology + future qualified owners |

The board's three diagnostics are not outcome sensors:

| Read | Validity and limit | Interpretation |
| --- | --- | --- |
| `tech-tree-designer plan list` | Current owner response; count only, no artifact bodies | Proto-plan inventory, never generic workspace coverage |
| `tech-tree-designer ontology coverage` | Owner `graphError` invalidates coverage; omitted protobuf zero remains zero only on a successful response | Authored capabilities/scenario counts, not fulfillment |
| `prompt-manager.skill-set-read` | Inspect child status, seven row identities, missing readings and artifact metadata | Registry presence, not skill quality or agent behavior |

Until acceptance resolvers exist, the board cannot consume an old test receipt
and declare it fresh. No caller-supplied receipt, timestamp or band can make an
outcome green. Future owner joins must reject stale, wrong-revision, wrong-cohort,
failed, canceled and incomplete evidence; these are separate from availability.

### Setup qualification

Run `python3 -m unittest discover -s scenarios/tech-tree-designer/test -p 'test_setpoint_read.py'`
from the repository root for adversarial reader tests using the real runtime
helper and substituted owner boundaries. This is a focused test invocation,
not a private production workflow. Run `vrooli scenario test tech-tree-designer
--phases programs,skill-set` for governed fixtures and declaration checks.
Inventory-only proves permanent gaps survive execution; live-diagnostics must
pass against healthy owners. Do not accept `partial` merely because an owner is
currently broken. Admission rejection and local mocks are not live qualification.

Fresh-session qualification uses only START-HERE and the approved contract:

1. Given a new agent without this conversation, when it discovers TTD, then it
   finds both skills, the program, all selected targets and explicit exclusions.
2. Given all outcome readings are unknown, when diagnostics are healthy, then it
   selects missing owner telemetry or a dependency-blocking gap, not completion.
3. Given one selected repair and explicit development authority, when the agent
   completes it, then it records tests and a checkpoint and proceeds to the next
   in-scope repair without creating another approval item.
4. Given an absent grant, changed target or expired budget, when the agent would
   cross that boundary, then it checkpoints for amendment without performing it.
5. Given interruption and a fresh continuation, when it resumes, then it recovers
   evidence/usage through the owners rather than resetting the mandate.

Items 3–5 require the qualified Swarm/Agent Manager development path. A local
skill divergence probe can qualify routing text, but cannot replace this run.

## Requirement Tags

Preserve existing IDs. Add [REQ:<id>] tags to actual tests and map exact existing test references as implementation lands. Planned manual protocols are specifications, not logged evidence. Do not weaken requirements or declare draft aspirational experience claims verified to make a gate pass.

## Cross-References

- [Requirements](../../requirements/README.md)
- [Performance](PERFORMANCE.md)
- [Security](SECURITY.md)
- [Test substitution seams](SEAMS.md)
- [Shared testing protocol](../../../../docs/TESTING.md)
