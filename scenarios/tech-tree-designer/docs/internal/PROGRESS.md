# Progress — Tech Tree Designer

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-14 | Codex | done | Greenfield-regenerated Tech Tree Designer from `react-vite`, removed the template notes domain, kept the scaffold health surface green, and recorded the graph/planning/roadmap target domains. |
| 2026-06-14 | Codex | done | Added Phase 2 graph/planning/roadmap proto contracts, expanded graph CLI manifest coverage with deferred handlers, and implemented a tested `api/internal/graph` `ProtoHealthSource` seam over `DescribeScenariosProtos`. |
| 2026-06-14 | Codex | done | Implemented Phase 3 graph Connect handlers, graph query/export service logic, endpoint manifest entries, and runnable graph CLI commands over the generated `GraphService` client. |
| 2026-06-14 | Codex | done | Implemented Phase 4 planning API/CLI: SQLite planned proto file tree, protocompile validation, materialization, planned graph overlay, endpoint regeneration, and cli-health-clean manifest bindings. |
| 2026-06-14 | Codex | done | Implemented Phase 5 roadmap API/CLI: overlay storage, Connect handlers, progress rollup from graph node stability, manifest bindings, endpoint regeneration, and focused tests. |
| 2026-06-14 | Codex | done | Implemented Phase 6 UI surfaces: React Flow + ELK scenario graph canvas, planned proto editor with validation/materialization actions, roadmap progress overview, route-level code splitting, and navigation/test coverage. |
| 2026-06-14 | Codex | done | Implemented Phase 7 integration hardening: localized graph/planning/roadmap feature copy, fixed planned-proto validation for live imports with Google well-known dependencies, self-validated graph/planning/materialization flows, and restarted the scenario healthy. |
| 2026-06-14 | Codex | done | Cleaned up post-integration docs drift: README, quickstart, architecture, domains, flows, testing, monetization, API reference, and template docs now describe the shipped graph/planning/roadmap surface and pass docs validation without warnings. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

## 2026-09-09 — documentation-first ecosystem design target

Updated the PRD through Business Health, expanded requirements while preserving foundation records, and authored five draft pages plus four draft journeys. Replaced stale template guidance with the three-view architecture, repository-wide proposals, immutable review, owner-directed apply/recovery, authority/isolation, scale qualification and fresh-agent development obligations. No implementation code was changed by this task; concurrent UI work was preserved.

Experience specification validation passed on the installed CLI. A subsequent call warned that current-source auto-rebuild failed and used a stale binary, limiting qualification. Business Health accepts PRD shape but reports historical unearned completion claims for TTD-GRAPH-001, TTD-ONTOLOGY-001 and TTD-SDA-001; requirements validation reports the same debt. No runtime certification, attestations or evidence sync were fabricated.

## 2026-09-09 — final documentation consistency review

Reviewed the expanded target against the operator's observed/intended/proposal model.
Clarified path-first drafting without graph coverage, read-only refresh recovery after
successful application, and in-mandate editing without per-edit bundle approval.
Added these cases to the existing qualification protocols. Replaced stale notes-domain
error examples with current planning/ontology mappings and corrected quickstart paths,
managed health discovery and focused test guidance. Preserved prior edits and code.

Local checks passed for 47 documentation/contract files, 129 local link destinations,
18 PRD target mappings across 32 requirements, and page/state references for five draft
pages and four journeys. JSON parsing and scoped `git diff --check` passed. These checks
do not validate prose semantics or runtime behavior by themselves.

Business Health and requirements validation still report the same three historical
unearned completion claims. Experience validation reports no findings through its
stale installed CLI; current-source rebuild still fails on a missing protovalidate
checksum. Retain both limitations. No product suite, service launch, source implementation,
dependency repair or evidence attestation was performed during this final review.

## 2026-09-09 — development setup implementation and validation

This is the subsequent authorized implementation pass. Added scenario-owned
usage/improve skills, declared them in service.json, and added the read-only
setpoint program with an outcome inventory and isolated diagnostic joins.
START-HERE is the entrypoint. TESTING owns the evidence matrix; PERFORMANCE
and DECISIONS hold explicit unapproved qualification/authority proposals.
The preview JSON selects P0 and the proposed ontology/UI/parity P1 subset without
approving or launching anything. No full TTD product implementation is claimed.

Corrected the historical requirement assertions through the Business Health /
requirements-traceability method: preserved their history, demoted unsupported
completion and repaired stale test function references. Existing tests retain
their assertions and now carry the corresponding requirement tags. Repaired the
API protobuf validation checksum through the approved SDA dependency gateway.
The lifecycle starts TTD; optional Prompt Manager / Program Runtime fresh-build
attempts reported degraded dependencies, although their existing read endpoints
were available for the successful diagnostic run. No private process launch.

### Validation evidence

| Check | Result / limit |
| --- | --- |
| Local reader regressions | `python3 -m unittest discover -s scenarios/tech-tree-designer/test -p test_setpoint_read.py`: num[sot]:10 tests pass. Real Program Runtime helper, substituted owner boundaries; not product or live-fixture proof. |
| Foundation tests | Focused graph/ontology Go regressions pass, including the corrected registry references; not a comprehensive requirements-sync snapshot. |
| Registry | `prompt-manager skill-set validate tech-tree-designer` passes; both skills are readable and the child registry reader returns complete diagnostics. |
| Live reader | `prog_43e720a6-3edc-41f3-81a4-d6d71abe328d`, digest `dd6402b7294767a743ac7f4dfa49e2fc281c11374b882a3e777683afb9229ed2`, runtime/envelope `ok`, 563 ms. All outcome readings remain unknown. |
| Initial Test Genie run | `20260909-163227-f7c7c663`: skill-set/business pass; programs fails during unavailable startup and unit fails on existing UI debt. |
| Follow-up Test Genie run | `20260909-163608-84ddcbd5`: skill-set/business pass; programs remains failed on historical budget evidence. Overall PASS reflects advisory phase policy, not successful programs qualification. Source inspection confirms the owner skipped new fixtures after that finding. |
| Documentation | `20260909-163717-2a0a71c9` found local manifest/table/example defects, now repaired. Remaining owner alias/path issues are in PROBLEMS. Required docs and local links remain represented; no waivers were introduced. |
| Final scoped sweep | `20260909-164615-224d76d4`: skill-set/business pass; docs retains the shared README-alias error and reference warnings. Local manifest/table errors are closed. This is not full documentation certification. |
| Experience contract | Installed CLI reports no findings, with explicit stale-binary warning after current-source build failure. Not fresh-owner proof. |
| Development preview | Full request rejected for DESIGN.md; bug filed, file retained. No review-complete, approval, acceptance or launch claim. |

The initial live program timeout was not a nested-library incompatibility:
independent plan/ontology reads and the child program succeeded after startup;
the unchanged concurrent composition then succeeded. Startup/read availability
is the supported explanation; no unobserved internal transport cause is claimed.
The failed runtime had no JSON envelope, so the usage skill explicitly retains
that failure and the outcome denominator instead of treating it as empty work.

### Skill validation report — usage and improve roles

#### Summary

Scenario declaration and command/path existence checks pass. The skills expose
current operations and development repair selection without authorizing new
effects. Fresh-agent end-to-end execution remains unqualified; local review
does not substitute for that proof.

#### Findings

| Capability | Primary path | Verification | Failure path |
| --- | --- | --- | --- |
| Select existing operations | Usage decision table and real CLI group help | CLI help, live plan/ontology reads | Central troubleshooting table |
| Read targets without false completion | Setpoint contract and owner reads | Local adversarial tests and live program identity above | Null outcomes, child validity and runtime-failure handling |
| Select successive authorized repairs | Improve routes plus campaign owner | Divergence probes below; not an agent-run claim | Amendment/checkpoint branch |
| Complete a whole development mandate | Shared Swarm/Agent Manager owner | Not executed; launch/resolver/approval prerequisites remain | Explicit launch boundary |

#### Divergence probe

| Probed instruction | Attempted divergent executions | Result / decision |
| --- | --- | --- |
| Generic proposal request | Publish via proto materialize vs report unsupported general operation | Materialize substitution explicitly prohibited; only the latter complies. |
| Board ok, outcome rows unknown | Claim completion vs select missing evidence repair | Completion explicitly requires selected owner evidence; only repair within authority or read-only reporting complies. |
| Successive in-scope repair | File an approval item for each repair vs continue under mandate | Per-repair approval is explicitly excluded; protected-target amendment still requires review. |
| Missing grant | Repair dependency because a sensor named it vs checkpoint | Sensor grants no effects; only checkpoint/report complies. |

No Critical or Major divergence found in these probes. This is a same-session
contract probe, not independent-agent agreement evidence.

#### Contract integrity

No primary skill workflow uses parser pipelines, private HTTP or shell mutation
glue. CLI syntax remains owned by command help. Machine JSON in this validation
pass was used to inspect exact provider findings/IDs; not installed as a user
workflow. The Python test runner is a focused regression exception, not a
replacement production program. Referenced campaign, ladder, traceability,
measures-adoption, debugging and Memory skills were resolved through the registry.

#### Validation findings and cross-skill coherence

- Gap: product acceptance sensors and actual fresh-agent development proof are
  absent. The skills declare these limitations and route them to their owners.
  Resolution is authorized owner implementation and TESTING's fresh-session
  protocol, not additional claims of readiness.
- Gap: owner docs validation disagrees with canonical path/basename semantics.
  Filed owner defects are retained in PROBLEMS; no source-reference suppression.
- Shared execution authority, learning and evidence semantics are cited, not
  reimplemented. No additional feature role is claimed or waived.

#### Prose Retirement Map

| Instruction / Gate | Disposition (Keep/Collapse/Delete) | Rationale | Prerequisite contract | Risk |
| --- | --- | --- | --- | --- |
| Operation argument syntax | Collapse | CLI already owns syntax | TTD CLI help/manifest | Stale copied flags |
| Bounded diagnostic collection | Collapse | Repeated joins belong in the program | tech-tree-designer.setpoint-read | Child failure must remain visible |
| Generic proposal versus proto publication | Keep | Safety and scope | Usage role boundary | Accidental canonical publication |
| Evidence selection and repair authority | Keep | Judgment and ownership | Shared development contract | False acceptance or unauthorized scope |
| Per-repair approval queue | Delete | Contradicts authorized iterative development | Shared development contract | Reintroducing a queue for every local repair |

#### Recommendations and notes

R1A (recommended): repair the named owner validation/preview gaps, then execute
the preserved scoped gates and fresh-agent protocol through the qualified Swarm
path. R1B: retain this setup as unqualified development scaffolding while owner
work continues; do not claim a launch-ready scenario. Neither option authorizes
new budgets or changes to the selected product contract. Revisit promotion only
after repeated usage establishes another stable multi-operation workflow.

### 2026-09-09 — shared validation and review blockers repaired

**Trigger:** Operator approved extending the setup pass to the named shared-owner
blockers. This did not authorize the full TTD product-development mandate.

**Changes:** Swarm review accepts and fingerprints DESIGN.md and canonical PRD
outcome IDs. Knowledge Observatory separates explicit identifiers from ambiguous
basename aliases and resolves marked paths from the repository root. Program
Runtime runs fresh fixtures despite historical portfolio debt, while retaining
that debt in qualification. Experience Manager's CLI checksum was installed
through the approved dependency gateway. TTD UI test setup now preserves the
canonical per-render query client; leaf input tests use that same helper.

**UI investigation:** Prior recall returned the existing TTD testing contract and
regeneration history, not a confirmed matching fix. Hypothesis A was a shared
pending query shadowing the canonical client; hypothesis B was an incorrect API
mock or graph state renderer. A probe compared the component's useQueryClient
identity with the helper's returned client and failed before repair. Removing
the extra shared provider made the probe and unchanged graph assertions pass.
The API mocks and production GraphPage did not change. The setup still supplies
the scenario theme and locale providers. This supports A over B for this failure.

| Evidence | Observed result and limit |
| --- | --- |
| Swarm development package | Full package Go tests pass, including DESIGN fingerprint invalidation, canonical outcome IDs, malformed identifiers, scope, symlink, and authority regressions. |
| Knowledge Observatory | doccontract, dochealth service, and dochealth handler Go packages pass. Regressions cover order-independent basename ambiguity, explicit collisions, scenario/skill roots, missing paths, and repository escapes. |
| Program Runtime | Focused handler regressions pass: historical budget debt plus fresh success, fresh failure, static-only inspection, and invalid static contract. Actual fixture admission remains digest-pinned and test-provenance. |
| Fresh governed fixtures | Run `20260909-170118-74e98643`: inventory execution `prog_0b5b49d5-38d8-44a4-9375-6da123156084` succeeded in 1394 ms; live execution `prog_0f50e60b-7d70-4b77-9e43-f56fe22b7dcc` succeeded in 6811 ms. Both use digest `dd6402b7294767a743ac7f4dfa49e2fc281c11374b882a3e777683afb9229ed2`. Invalid-mode admission matched its declared rejection. No fixture finding remains. |
| Qualification distinction | That run passed docs, skill-set, and business. Programs still failed on `programs.budget_exceeded`; overall PASS is advisory-phase policy, not a clean program portfolio. Failed historical runs were not erased and the budget was not increased. |
| UI assertions | num[sot]:81 tests in num[sot]:19 files pass. The new client-identity/isolation test demonstrably failed before repair. Graph error/empty/warning tests now pass without longer test deadlines. |
| UI coverage | All assertions pass under coverage, but measured lines/statements are 63.82%, branches 66.41%, functions 50.75%, below the existing 85% floor. No exclusions or thresholds changed. |
| Final scoped sweep | `20260909-170629-0d0e0747`: docs passed, unit failed only on the coverage command's error; render-policy drift is gone. Coverage/seam warnings remain. This is not full scenario certification. |
| Live doc owner | After the text-diagram correction, docs health reports no failure-severity findings and no broken marked paths. Existing command validation warnings remain, with no reference waivers. |
| Experience owner | Current-source CLI rebuilt at fingerprint `5fe8d25d5d191ab679b09791226fc737cba247f948b5a206dccf244d724dfd78`; spec validation passed without stale fallback. |
| Full review packet | Live preview returned digest `b4739c453fd712aff9140b6eaec0716958f4a957ec370335db72cdd5392cb20f`, including DESIGN.md hash `3f71fbd9632ee8e6787772d34b94edfdbfbb3e86d2c3eb0da1bf086d0d21b4c8`. It retained effects/evidence findings and launch blockers. This is an observation before this log append, not an approval or a claim that the mutable packet still has that digest. |

**Remaining work:** Actual product implementation and owner-backed acceptance
measurements; explicit mandate/grant/budget review; fresh-agent execution through
qualified Swarm; UI coverage; existing dependency governance and Memory startup
debt. Numerical performance, retention, backup, and spending proposals remain
unapproved. The diagnostics still retain every outcome as unknown.

**Owner reports:** Existing repaired bug IDs are retained in PROBLEMS.md's earlier
table. CLI Health's group-help false negatives are filed as
`knw-1788973641904063371`. Test Genie's registry manifest mismatch is filed as
`knw-1788973716593366903`. Neither issue is hidden by relabeling current commands
as future ones.

**Evidence applicability:** This was a shared mutable worktree. A concurrent
Program Runtime binding-digest compile error disappeared after an intervening
owner edit; this pass did not alter that code. Package assertions and live
observations stand as recorded, not as an isolated exact-revision certification.
