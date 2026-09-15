# swarm-manager persisted-state inventory (Phase 1)

Read-only snapshot for the declarative-operations state migration (Phase 8).
Deterministic and byte-stable: no timestamps; two runs over unchanged state match.

## Roots

- resolved from: `default(~/.vrooli)`
- data:  `~/.vrooli/data/vrooli/swarm-manager` (exists=true)
- state: `~/.vrooli/state/vrooli/swarm-manager` (exists=true)
- cache: `~/.vrooli/cache/vrooli/swarm-manager` (exists=true)
- config file: `~/Vrooli/scenarios/swarm-manager/config/settings.json` (exists=true)

## Totals

- files scanned: 3389
- bytes: 13834697
- primary objects: 2312
- anomalies: 0
- referential findings: 14
- master content hash: `sha256:11a83c6b3563da19a7d5204340297bf6a2d6b6374234875dd3765beaa1f79c44`

## Object classes

| class | kind | count | bytes | content hash |
|---|---|---:|---:|---|
| acceptance_validation | artifact | 56 | 5208 | `sha256:ade03841705670302b82adc6a5e9f4f74f23d0af8b25ffdd6aaa19cd04453307` |
| agent_activities | state | 1 | 64883 | `sha256:464c842e7608bd765c18e8f76293663f6cab57b888947e710f74b3ac7113d132` |
| backlog_clarify_artifact | artifact | 1 | 4376 | `sha256:0bb3cd5554fe79b1fd22a077fb3e74063687c427f8c4325c7f1b89645a59cc56` |
| backlog_evidence_artifact | artifact | 9 | 36373 | `sha256:ff775d6536ecdbbf489991a3d33e02b264c30c54493dcaeac331a0ab8d1e1478` |
| backlog_item | primary | 600 | 1102468 | `sha256:3be47666b22128961e61e20432cff15ec79a49c2d1dc85be11d783fc136d4cf1` |
| backlog_review_artifact | artifact | 185 | 758582 | `sha256:00ad7583d7f0383c1c287f1b9885d74bd5b49df0c2e3f20a36edfe88b82974a1` |
| capture | primary | 9 | 7254 | `sha256:26f5aa0badf1ccc3075d3872e8e69d6341f3688f956e56d10588f92b1ec3a266` |
| eventlog_sqlite | opaque | 3 | 958544 | `sha256:ee3606e427638e9fc434086b7ac260437894b2b81ed8d0e6bb84c7bee4531648` |
| execution_runs | state | 1 | 2083 | `sha256:04d8117704b2d7dff4375050d8f2e48da82bc1678427ea17255abce1df9e3a96` |
| foreign_deployment_report | foreign | 1 | 283273 | `sha256:86b8fc24b38b731426b5a62e95bf7b076321a96fb406acfc15ae04562263d089` |
| goal | primary | 5 | 4242 | `sha256:ccbf1bc8414a8efaa0c37d95cb96eda8929de9b76acf19d4d0643f16d2a76a34` |
| initiative | primary | 68 | 122606 | `sha256:17f8e56c64073182a70227cd2c93cf65dc8efec8b797876d6c35e199691a1661` |
| initiative_context_file | artifact | 84 | 477463 | `sha256:d47dc6002f3e6198e01838e8d6ad6df11d9805db0cbd37289784364d7fc1f13b` |
| initiative_graph | artifact | 68 | 152717 | `sha256:9e67bcf05274bd1618ba88deae977e46477251b95506cedbcb469a5377836a54` |
| initiative_review_round | artifact | 1 | 241 | `sha256:e93685d339e66e1db9a35f84fd2e3f1b2865a69cdba43730e0c4d293bb6cab75` |
| item_doc | artifact | 229 | 1892867 | `sha256:d6ccd5a1e9f3c1351cddfc05156a5f2e7fb486bc7e1db11f0cdc4bdb04b32d1b` |
| item_swarm_artifact | artifact | 136 | 3374709 | `sha256:ff137e72f693845c0b51f5ac19d13413ee57df83388033e9c3baa4b1e7a774c3` |
| om_execution_manifest | primary | 1 | 38399 | `sha256:9976c5b455d050f8610bdb83162039f49506d0c59ee95b45d7bf81ee98a6c653` |
| om_global_run_owners | state | 1 | 599 | `sha256:e3a7d431fecb448a1d4b288728df6f5713b8b148cd6607c6f5b131e8daf0df9c` |
| om_round | artifact | 4 | 53202 | `sha256:5262e602f04a6bb5ddaf9a0a14e2f5958cd0b35c627ee79d66e9d241e468ddcb` |
| om_scope_run_owners | state | 1 | 152 | `sha256:2b6f25c48c324d8bbff29f1fce956f249bbab6c8bd9fafa90aab03486d7aa9bc` |
| plan_ref_sweep_manifest | artifact | 1 | 54865 | `sha256:eb3ef007bd0264ad1f0cf8ae114747783aa986cd48ee1f8c680b34ca48053329` |
| record | primary | 1629 | 2499845 | `sha256:d1974e3ef905092abc84953ed8f0203c0832202f05ba7497b1a2427b7b4dbe1e` |
| settings_config | artifact | 1 | 1402 | `sha256:7fd900c068e0954e65d4befa1dd3cfa468c3fff63c896a7fc47fdc5a939fabc6` |
| unclassified | artifact | 8 | 54142 | `sha256:5fa8702ab0f70b7f9b00d0a1f52891e6b30c0b36a3f2a6090abb28e2e7685cad` |
| workshop_clarification | artifact | 3 | 21871 | `sha256:56e51f7cfb8c3855cafdd3f65032df2d36d926f4bd5dc9a6f4e56c213a9bb048` |
| workshop_round | artifact | 283 | 1862331 | `sha256:f3ad6ce3457e59b79429c6cdde9722f96e6ab326dd224e8baba6c0f60f3efd73` |

## Status distributions

- **backlog_item**: backlog=452, completed=135, failed=6, needs_followup=1, queued=4, review_pending=2
  - by kind: chore=21, execute=318, fix=131, idea=73, research=57
- **capture**: failed=9
- **execution_runs**: failed=1, pending=3
- **goal**: active=5
- **initiative**: active=65, completed=2, review_pending=1
- **om_execution_manifest**: needs_attention=1
- **record**: <empty>=3, abandoned=1, partial=343, shipped=1282
  - by kind: chore=45, execute=1291, fix=203, research=85

## Plan-ref usage

- total: 144, managed: 144, **unmanaged: 0**

## Ownership

- global run-owner index present: true
- scope run-owner indexes: 1
- engagement-owners present: false
- ambiguous run owners: 0

## Expected-but-absent state

- `data/auto-drain.json` — auto-drain flag; absent means disabled (default)
- `data/autofiler/dismissed_findings.json` — auto-filer dismissals; absent means none dismissed
- `state/circuit-breaker.json` — per-item failure circuit breaker; absent means no trips recorded
- `state/engagement-owners.json` — engagement/run-owner exclusivity index; absent means no open engagements
- `state/queue.json` — execution queue snapshot; absent means empty queue

## Referential findings

- ambiguous_ownership: 1
- dangling_dependency: 2
- dangling_initiative_item: 3
- unclassified_artifact: 8

<details><summary>all findings</summary>

- [ambiguous_ownership] `data/deployment/deployment-report.json` → `` foreign artifact written by scenario-dependency-analyzer under swarm-manager data root; not swarm-manager-owned state
- [dangling_dependency] `fix/dtv-cli-validate-and-report` → `fix/dtv-validation-api` depends_on target not found on disk
- [dangling_dependency] `fix/dtv-report-generation` → `fix/dtv-validation-api` depends_on target not found on disk
- [dangling_initiative_item] `initiative/dtv-meta-optimization-readiness` → `fix/dtv-validation-api` initiative.items references a non-existent backlog item
- [dangling_initiative_item] `initiative/swarm-manager-graph-workspace` → `chore/swarm-manager-remove-legacy-tabbed-surfaces` initiative.items references a non-existent backlog item
- [dangling_initiative_item] `initiative/swarm-manager-graph-workspace` → `execute/swarm-manager-mobile-graph-interaction` initiative.items references a non-existent backlog item
- [unclassified_artifact] `data/plan-executions/32da39ad-96c4-44eb-851f-2473b6d109b1/agentops/executions/exec-2c8e3b6cad6bfec1ff9ba9c8.json` → `` file matched no known swarm-manager storage pattern; investigate before migration
- [unclassified_artifact] `data/plan-executions/32da39ad-96c4-44eb-851f-2473b6d109b1/agentops/workflow.json` → `` file matched no known swarm-manager storage pattern; investigate before migration
- [unclassified_artifact] `data/plan-executions/768abed4-8ce9-48db-84fd-e08f77b8785d/agentops/executions/exec-9dd73f45c7c89386244750cd.json` → `` file matched no known swarm-manager storage pattern; investigate before migration
- [unclassified_artifact] `data/plan-executions/768abed4-8ce9-48db-84fd-e08f77b8785d/agentops/workflow.json` → `` file matched no known swarm-manager storage pattern; investigate before migration
- [unclassified_artifact] `data/plan-executions/e214a5ab-3f94-4d67-996e-627f71abbda4/agentops/executions/exec-3cd32c57b2d1e4ead7e4428b.json` → `` file matched no known swarm-manager storage pattern; investigate before migration
- [unclassified_artifact] `data/plan-executions/e214a5ab-3f94-4d67-996e-627f71abbda4/agentops/workflow.json` → `` file matched no known swarm-manager storage pattern; investigate before migration
- [unclassified_artifact] `data/plan-executions/f5de76eb-513e-4ce6-affd-80c438f2759c/agentops/executions/exec-3968789f881f54ab844ab708.json` → `` file matched no known swarm-manager storage pattern; investigate before migration
- [unclassified_artifact] `data/plan-executions/f5de76eb-513e-4ce6-affd-80c438f2759c/agentops/workflow.json` → `` file matched no known swarm-manager storage pattern; investigate before migration

</details>

## Anomalies (unreadable / invalid state — reported, never dropped)

- none
