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

- files scanned: 3354
- bytes: 13614763
- primary objects: 2293
- anomalies: 0
- referential findings: 6
- master content hash: `sha256:0b4030439dfc806c71aa37533445f77525f0241769d4b4323742726d231f8f57`

## Object classes

| class | kind | count | bytes | content hash |
|---|---|---:|---:|---|
| acceptance_validation | artifact | 56 | 5208 | `sha256:ade03841705670302b82adc6a5e9f4f74f23d0af8b25ffdd6aaa19cd04453307` |
| agent_activities | state | 1 | 62708 | `sha256:ddc25fe3156b131973d89c08b0b2fc02c17a901bef0ccee00a0641c91d8660f5` |
| backlog_clarify_artifact | artifact | 1 | 4376 | `sha256:0bb3cd5554fe79b1fd22a077fb3e74063687c427f8c4325c7f1b89645a59cc56` |
| backlog_evidence_artifact | artifact | 9 | 36373 | `sha256:ff775d6536ecdbbf489991a3d33e02b264c30c54493dcaeac331a0ab8d1e1478` |
| backlog_item | primary | 600 | 1102464 | `sha256:69082eb87210bc9644745157a7a6482635366cf11ec61ac28330ac2c95d5d60f` |
| backlog_review_artifact | artifact | 185 | 758582 | `sha256:00ad7583d7f0383c1c287f1b9885d74bd5b49df0c2e3f20a36edfe88b82974a1` |
| capture | primary | 9 | 7254 | `sha256:26f5aa0badf1ccc3075d3872e8e69d6341f3688f956e56d10588f92b1ec3a266` |
| eventlog_sqlite | opaque | 3 | 921600 | `sha256:aa8df649f94377ea2498b145a247ee8b7b15067e897dadede4a29704f43591df` |
| foreign_deployment_report | foreign | 1 | 276526 | `sha256:8995ed0f9512855160831bd347c7d03f2986429a03555210ee6f54fd23b8b67f` |
| goal | primary | 5 | 4242 | `sha256:ccbf1bc8414a8efaa0c37d95cb96eda8929de9b76acf19d4d0643f16d2a76a34` |
| initiative | primary | 68 | 122606 | `sha256:17f8e56c64073182a70227cd2c93cf65dc8efec8b797876d6c35e199691a1661` |
| initiative_context_file | artifact | 84 | 477463 | `sha256:d47dc6002f3e6198e01838e8d6ad6df11d9805db0cbd37289784364d7fc1f13b` |
| initiative_graph | artifact | 68 | 152712 | `sha256:84b05e78d8b54698ac6107f3f0500ebb8f782ce4e5864aec060ee7f6715e972e` |
| initiative_review_round | artifact | 1 | 241 | `sha256:e93685d339e66e1db9a35f84fd2e3f1b2865a69cdba43730e0c4d293bb6cab75` |
| item_doc | artifact | 225 | 1852207 | `sha256:4de9a03034f542feb4204bd17efc1ca71b956c4fed6638721565ac79fdda9766` |
| item_swarm_artifact | artifact | 136 | 3374709 | `sha256:ff137e72f693845c0b51f5ac19d13413ee57df83388033e9c3baa4b1e7a774c3` |
| om_round | artifact | 3 | 43804 | `sha256:29015ee9285467d516f96d069cc1429d2a39e7e80bacf587eea30273fd45076e` |
| plan_ref_sweep_manifest | artifact | 1 | 54865 | `sha256:eb3ef007bd0264ad1f0cf8ae114747783aa986cd48ee1f8c680b34ca48053329` |
| record | primary | 1611 | 2471219 | `sha256:88fac4953245590ee406e2bbed089586cb184efbd7712f3465823a231564784b` |
| settings_config | artifact | 1 | 1402 | `sha256:7fd900c068e0954e65d4befa1dd3cfa468c3fff63c896a7fc47fdc5a939fabc6` |
| workshop_clarification | artifact | 3 | 21871 | `sha256:56e51f7cfb8c3855cafdd3f65032df2d36d926f4bd5dc9a6f4e56c213a9bb048` |
| workshop_round | artifact | 283 | 1862331 | `sha256:f3ad6ce3457e59b79429c6cdde9722f96e6ab326dd224e8baba6c0f60f3efd73` |

## Status distributions

- **backlog_item**: backlog=456, completed=135, failed=6, queued=1, review_pending=2
  - by kind: chore=21, execute=318, fix=131, idea=73, research=57
- **capture**: failed=9
- **goal**: active=5
- **initiative**: active=65, completed=2, review_pending=1
- **record**: abandoned=1, partial=343, shipped=1267
  - by kind: chore=44, execute=1283, fix=199, research=84

## Plan-ref usage

- total: 144, managed: 144, **unmanaged: 0**

## Ownership

- global run-owner index present: false
- scope run-owner indexes: 0
- engagement-owners present: false
- ambiguous run owners: 0

## Expected-but-absent state

- `data/auto-drain.json` — auto-drain flag; absent means disabled (default)
- `data/autofiler/dismissed_findings.json` — auto-filer dismissals; absent means none dismissed
- `data/operating-mode-run-owners/run-owners.json` — global operating-mode run-owner index; absent means no mode-round runs indexed
- `state/circuit-breaker.json` — per-item failure circuit breaker; absent means no trips recorded
- `state/engagement-owners.json` — engagement/run-owner exclusivity index; absent means no open engagements
- `state/execution-runs.json` — item-level execution run log; absent means no runs recorded (or recently pruned)
- `state/queue.json` — execution queue snapshot; absent means empty queue

## Referential findings

- ambiguous_ownership: 1
- dangling_dependency: 2
- dangling_initiative_item: 3

<details><summary>all findings</summary>

- [ambiguous_ownership] `data/deployment/deployment-report.json` → `` foreign artifact written by scenario-dependency-analyzer under swarm-manager data root; not swarm-manager-owned state
- [dangling_dependency] `fix/dtv-cli-validate-and-report` → `fix/dtv-validation-api` depends_on target not found on disk
- [dangling_dependency] `fix/dtv-report-generation` → `fix/dtv-validation-api` depends_on target not found on disk
- [dangling_initiative_item] `initiative/dtv-meta-optimization-readiness` → `fix/dtv-validation-api` initiative.items references a non-existent backlog item
- [dangling_initiative_item] `initiative/swarm-manager-graph-workspace` → `chore/swarm-manager-remove-legacy-tabbed-surfaces` initiative.items references a non-existent backlog item
- [dangling_initiative_item] `initiative/swarm-manager-graph-workspace` → `execute/swarm-manager-mobile-graph-interaction` initiative.items references a non-existent backlog item

</details>

## Anomalies (unreadable / invalid state — reported, never dropped)

- none
