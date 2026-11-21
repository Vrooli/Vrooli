# Local work observations for the cloud bridge

This guide owns the repeatable collection method. The [Local Work Bridge in Google Drive](https://docs.google.com/document/d/152DaKTk-lBAiy2GXHf-1Bpp3GalY0xXj8EolzhYECrc/edit?tab=t.fmv6uwwand9x) owns the human reconciliation workflow. The source owners remain authoritative. Neither guide is a second work tracker.

## One bounded observation

```sh
program-runtime library run program-runtime.local-work-snapshot \
  --provenance agent \
  --input research_root=/home/matthalloran8/.vrooli/research \
  --json > /tmp/local-work-snapshot.json
```

Use a caller-authorized absolute research root on another machine; omit the input to leave research explicitly `not_requested`. This command creates and reclaims its own Runtime session. It does not start agents, modify work, schedule collection, infer Flywheel matches or accept outcomes. Read the execution record and then parse its `program.stdout` JSON envelope. Preserve program ID, session ID, provenance, `program_digest`, inputs and timestamps together. Runtime `SUCCEEDED` is not the domain verdict.

The default is four examples per source and one read for each of pending, starting, running, needs-review, parked, unknown and complete run statuses. Failed and cancelled runs are not checked by default. The default is a small briefing, not an exhaustive activity inventory. `status: partial` is intentional: native process liveness, queues/task boards outside explicitly selected teams, Swarm execution records, research acceptance and Flywheel links remain uncollected. Version 2 can additionally read execution ownership and one task-board page for each of up to four explicit `team_ids`; omission leaves these reads `not_requested`. Program Runtime usage is a sixth **capability context**, never another work count.

The contract/source pair is `scenarios/program-runtime/.vrooli/program-runtime/local-work-snapshot.{json,py}`. This is a regular declared program, not a global collector, scheduler or event-delivery mechanism. Adding selected-team coverage does not install continuous local-to-cloud pickup. Reuse it at the existing Flywheel review or on-demand reconciliation boundary. No background schedule was installed.

## Read each source on its own terms

| Source | What it observes | What it does not prove |
| --- | --- | --- |
| `runs/<status>` | One explicit Agent Manager status page, exact run/parent/task identities, execution mode and total | A retained `running`, imported or attached record is not live-process proof; terminal is not accepted |
| `teams` and `objectives` | Registered standing/finite team containers and declared objective attachments | Enabled, served and purpose labels are not queue state or outcome achievement |
| `team_execution/<team_id>` | One explicitly selected team’s owner-reported queue, retained-slot classifications and exact member/task/run links; class counts overlap | Retained `runningAgentIds` may include paused, uncertain or review-blocked owners. Idle is not proof of no work and may be the owner’s absent-execution-store fallback. No native process or outcome acceptance is verified |
| `team_tasks/<team_id>` | One bounded page for the same selected team, exact task/member identities, totals and continuation | Empty does not erase delivery history; `done` does not prove accepted delivery. Producer `updatedAt` is not refreshed by retrieval |
| `efforts` | One owner board page, enrollment and subject links, runtime state, outcome standing, discovery dates/findings | Page completion is not complete discovery; fresh collection does not refresh old scans or self-report |
| `swarm_goals` | Goal declarations, active/archived counts over at most 200 examined rows; active-first examples | Active goal labels do not imply running executors |
| `swarm_backlog` | One owner-ordered bounded backlog sample and total | It is not necessarily the active/newest work; no continuation is advertised by this binding |
| `swarm_executions` | Explicit `not_collected` marker and owner CLI locator | There is no hidden shell fallback or manufactured executor evidence |
| `research_catalog` | Up to 1 MiB of the selected CATALOG, retained topic/run counts, selected guides and workspaces, producer date, file date and hash | A catalog entry is not a live commission; indexed paths are not followed or used as authority |
| `research_workspace` | Existing explicit-root research review child, status, artifact identity, directory/file-presence signals | Directory count and catalog count have different denominators; neither proves research quality or scientific acceptance |
| `program_usage` | One declared usage window, identity gaps and program examples | Runtime success or usage is not successful work delivery |

Every source includes its owner locator, selected scope, collection timestamp, available producer timestamp and uncertainty. `query_complete` refers only to that specific owner query. `output_complete` and continuations describe returned examples, not world coverage. Some source types use named counts rather than `response_rows`; null means not supplied, not zero. Error/unknown/unavailable sources must never be converted to empty successes. Omitted non-optional protobuf numeric/boolean scalars use their wire-defined zero only after a successful typed owner response; explicit null or malformed pagination is rejected.

Text and supporting lists are deliberately shortened with omission markers. Inspect the original owner for decisive evidence. The parent Runtime receipt provides this program's digest. Child metadata retains the invoked research program's version/digest; parent digest alone does not pin transitive child source. Selected projection hashes identify the returned sample, not the entire owner database.

## Follow the gap, rather than repeating the whole inventory

- For a run status page, pass `run_offsets` for the selected status and the returned `next_offset`. For example: `--input 'run_statuses=["running"]' --input 'run_offsets={"running":4}'`. Other source families are still read once. Offsets are not transactionally stable while owner state changes; retain query times and deduplicate by exact native identity.
- For team, objective, goal or usage examples, use `sample_offsets` with the corresponding source name and `next_sample_offset`. At most 200 rows are inspected per list. A reached inspection bound is explicit, and no unusable continuation is emitted. Goal examples prioritize active declarations within that inspected slice, then descending `updated` date. Counts retain archived goals.
- For efforts, pass the exact owner-issued `nextPageToken` as `effort_page_token`. Preserve `partial`, discovery findings, `lastScanAt` and `lastSuccessfulScanAt` even when pagination ends. Inspect selected `enrollment.destinationRef`, home and authoritative contract before interpreting acceptance or next action.
- For research, select the catalog topic/run guide, canonicalize aliases and inspect its commission, current checkpoint, producer dates, parent/support role and closure evidence. Use `program-runtime.research-supervision-read` only after explicit native thread/run identities and any required authorized Web Console session are known. That child does not discover identities.
- For Swarm execution evidence, use `swarm-manager execution list --json` separately, then retain exact goal/backlog/execution/run links. The governed catalog inspected during implementation did not expose execution listing. Do not invent a callable path.
- For selected teams, add `--input 'team_ids=["vega-redesign-delivery","meta-optimization"]'` after verifying those native IDs in the current owner. Up to four unique IDs are accepted; they are never inferred from the team examples. The program reads `prompt-manager/team/execution-status` and `prompt-manager/team/task-list` once each per team. Read `team show` separately for the current charter, enabled/archived state and lifetime.
- To continue one selected team board, pass its returned `next_offset` in `team_task_offsets`, for example `--input 'team_task_offsets={"vega-redesign-delivery":4}'`. Every selected team still receives one queue read and one board page. Queue owner examples and membership samples are capped at `row_limit`, with omission counts; for an omitted decisive owner use the supported `team execution-status` CLI directly rather than inferring absence.
- Team task-board and queue bodies are JSON Structs, not typed scalar messages. Missing/null arrays or counts are rejected as unknown, not defaulted to zero. Fresh retrieval carries no producer observation timestamp for queue status. Class counts overlap, so never sum them into work or executor counts.
- If `output_truncated` is true, affected sources name omitted examples/details. Re-run a smaller `row_limit` at the **same** relevant offset before advancing: a query continuation can point past examples omitted from output.

## Reconcile without forcing one-to-one matches

Keep local-native entities and cloud/Flywheel outcomes separate. Every proposed edge needs both identities, evidence location and time, confidence, owning outcome and reviewer. Allowed relation vocabulary is `contributes_to`, `depends_on`, `duplicates`, `validates`, `informs`, `follow_up_candidate`, `local_only`, `cloud_only`.

No match is not missing work. Local-only and cloud-only work can both be valid. A proposed follow-up stays a proposal until an uncovered need, existing owner, current evidence and authority are checked. Never close an outcome from a stale report, successful program, terminal agent, prepared research brief or active goal label.

At a material source or workflow change, update this method and the existing Drive guide together. Keep dated observations in the existing evidence series rather than overwriting history or creating another dashboard.

## Verification and evidence

The focused tests are `scenarios/program-runtime/.vrooli/program-runtime/tests/test_local_work_snapshot.py`. They exercise input admission, independent owner failure, protobuf null/zero distinction, pagination, finite/standing distinction, stale discovery, research root escapes and different denominators, active-goal ordering, inspection bounds and oversized results. Version 2 adds explicit team selection, independent queue/board failures, Struct null/missing rejection, mismatched team rejection, task pagination, preserved long identities and stale dates, overlapping ownership classes and manifest read-effect regression checks.

```sh
python3 -m pytest scenarios/program-runtime/.vrooli/program-runtime/tests/test_local_work_snapshot.py -q
program-runtime programs validate scenarios/program-runtime/.vrooli/program-runtime/local-work-snapshot.json --json
vrooli scenario test program-runtime --phases programs
# Attach exactly once to the returned server-owned run:
test-genie runs wait --json program-runtime <run-id>
```

Retained implementation receipts: `.vrooli/evidence/local-work-bridge-20261106/implementation-20261107/`. These are dated observations, not continuing monitoring. The live default verification retained all source families without output clipping; the explicit uncollected evidence remains a limit.


### Version 2 selected-team qualification

On 7 November 2026 at 01:00:42–01:00:43 UTC, final read-only execution `prog_17e3278d-e6c2-4496-b7df-671140442649` collected four explicit teams at digest `c8a93f6bc4903ebfba55c175c6b63278e31c968cde5d161e3b122572f77a3a0f`. It returned 50,568 bytes, no collection errors and no clipping, with domain `partial`. Normal `programs get` retrieval returned byte-identical stdout. Nooch retained uncertain ownership and Notification Hub retained paused ownership; neither was labeled a live native executor. All four boards were empty without being labeled no work.

All 26 focused snapshot regressions and two owner CLI queue-read regressions passed. Protobuf freshness and contract validation passed. Shared-package baselines were green before and after the manifest change. Final managed programs run `20261107-010101-b6f71f1f` passed its programs-only phase, but five neighboring warnings and overall contract-integrity L0 remain. This is not full-scenario certification. Exact receipts and source backups are in `.vrooli/evidence/local-work-bridge-20261106/team-coverage-20261107/`.

## Terminal run evidence qualification (7 November 2026)

The bounded `agent-manager.investigation-evidence` projection is useful diagnostic
context, but **does not qualify a terminal run for the Flywheel resolver**. Its
`coverage: complete` means both requested report/accounting reads were available.
`source_completeness` and `source_freshness` remain `unknown`, even for a completed
run. The native selected Events reader proves the persisted event and its exact
bytes; it does not supply the missing run/Work evidence.

The native owner trace establishes these limits:

- `agent-manager/api/internal/orchestration/run_outcome_events.go` publishes
  terminal notifications best-effort. The payload correlates `run_id`, but carries
  no immutable run revision or Work binding. Publication returns only an error;
  the run row retains no terminal event-ID backlink. Notification time alone does
  not identify a specific completed invocation after continuation.
- `packages/api-core/eventbus/domain_event.go` derives the event ID from the
  complete domain event. Reconstructing it later from a mutable run, task label,
  UI base or current clock is not evidence of the persisted event.
- `agent-manager/api/internal/domain/run_types.go` defines `LifecycleVersion` as
  internal status/continuation fencing. `UpdatedAt` also changes on ordinary row
  updates and heartbeats. Neither is a retained immutable historical snapshot.
- Run `WorkReferences` are retained creation declarations. The creation path
  bounds and clones them; repository updates prevent replacement. This does not
  independently prove an arbitrary caller-provided `verified` flag, Flywheel
  `work_id`, `scope_version`, or whole-work/component relationship. The narrower
  native effort admission/inheritance checks do not establish that cross-system
  mapping.

The cloud resolver requires an exact selected terminal event backlink, the
commissioned source instance, a fresh owner observation, an immutable run
reference and an independently source-bound Work/scope/relationship. The existing
Work must reciprocally retain the same executor anchor. It has no partial-success
reader result. Therefore the current investigation adapter must retain
`run_unqualified`; a guessed link, an updated timestamp, a report hash or a new
read wrapper cannot close this gate. No additional runtime route was added.

### Prospective owner work, not historical reconstruction

The required owner design uses a canonical **prospective** run-owner receipt that
binds the terminal invocation/revision to the exact emitted event identity and
retains the immutable minimized observation bytes. Its transaction/replay rules
must survive continuation, duplicate finalization and response loss, and must
separate a recorded emission intent from confirmed Events persistence. The
receipt must state its actual capture coverage; older rows remain unqualified.
Current owner reads must recheck permission, source instance and observation
freshness without rewriting that historical identity.

Work lineage additionally needs a separately authorized, source-backed binding
of the native run to the existing Flywheel Work/scope/component executor anchor.
A caller's title, prompt, declaration or `verified` field is insufficient. Missing,
ambiguous or stale reciprocal evidence remains a hold. This is an engineering
requirement, not authorization to create a mapping, backfill receipts, activate a
route, transmit live data, provision credentials or change access.

### Offline evidence

The existing investigation fixture had two failures because it omitted the now
required accounting read. Its harness now supplies explicit accounting evidence
and explicitly fails both owners for the unavailable-subject case. Nine focused
regressions pass, including unknown freshness/completeness despite complete read
coverage, report/accounting sibling failure retention, null versus known-zero
accounting, and refusal to project injected revision/terminal-event/Work metadata.
No production projection, native publication, run persistence or source semantics
changed. These are deterministic fixture checks, not a live joined reader or
full-scenario qualification.

Reproduce with:

```sh
python3 -m pytest -q scenarios/agent-manager/.vrooli/program-runtime/tests/test_investigation_evidence.py
```

### Disabled native terminal capture implementation (7 November 2026)

Agent Manager now has an opt-in owner constructor,
`database.NewRepositoriesWithTerminalReceipts`. The production composition still
uses `NewRepositories`, which performs no terminal-receipt reads or writes. No
service was restarted, no live schema was migrated, and no publisher, external
reader, credential, grant or bridge enablement was installed.

The opt-in path joins the **existing** `runRepository.updateTx` transaction and
the direct pending `CancelTaskAttempt` transaction. It reads the prior native
row and the successfully fenced row in the same transaction. Only a valid
nonterminal-to-terminal transition produces a receipt. Terminal means complete,
failed, cancelled or unknown; unknown stays unknown. Needs-review, same-status
replay, terminal correction, terminal Create, imported rows and historical
migration do not gain a receipt. Two terminal invocations separated by a valid
continuation retain different receipts. A capture, source mismatch or Task
projection failure rolls back the entire owning transaction.

The run-model schema owns append-only `run_terminal_receipts`, unique by exact
run ID/lifecycle version and event-intent ID. Stored bytes and hashes, not the
mutable lifecycle counter alone, identify the historical observation. The
receipt has no cascading dependency on mutable run/task rows. It contains only
source instance/epoch, native run/task identities, lifecycle/owner epoch,
execution mode, terminal status, invocation/end timestamps and capture time.
It deliberately excludes prompt, title, transcript, errors, credentials and
caller-declared Work references. `capture_coverage` is `terminal_state_only`;
`work_lineage` remains `unverified`. Capture can be disabled between transitions,
so its existence does not prove a complete source inventory.

The explicit source configuration is copied on construction and bound to the
store on the **first successful capture**, in that transaction. A later instance
or epoch mismatch refuses capture/read rather than relabeling history. The
constructor itself does not contact storage or assert that the installation was
commissioned. Provisioning a real source identity, runtime adoption and rotation
remain owner decisions before activation. Schema application does not backfill
this table. Normal SQL updates/deletes of receipts and the source binding are
rejected; this is owner-storage immutability, not protection against a privileged
database administrator. A future retention policy must be separately designed.

The retained `agent_manager.run.terminal.v1` domain event is a **prospective
intent**, distinct from the existing best-effort notification. Its ID is the
SHA-256 of the exact stored `DomainEvent` JSON seed used by the current eventbus
client, not a digest of the eventual protobuf wire envelope. Its payload binds
the observation digest and source/run/lifecycle identities. Every receipt says
`publication_state: not_dispatched` as immutable capture-time state. The
separately composed native outbox below retains Events acceptance separately; a
successful/no-op best-effort publisher return is never substituted for that
evidence. Current canonical retention and permission still require a selected
Events read. Old notification IDs cannot be backfilled into these receipts.

The internal exact-receipt read returns retained bytes after integrity/source
checks; it does not reserialize a current Run. It is not a remotely authorized
reader. The dormant selected reader described below supplies bounded source-bound
selection through a trusted authority callback; a deployed handler and its actual
permission composition remain absent. The existing investigation adapter remains
`run_unqualified` and cannot substitute for this separate receipt/acknowledgment path.

### Separate reciprocal Work-binding gate

No Work mapping was created. The next owner protocol must name an existing
Flywheel Work ID, its exact scope/version and whole-work or component executor
anchor, together with the native source/run/immutable receipt and relationship.
The user must authorize the specific mapping; titles, apparent similarity,
`WorkReferences.Verified`, installation identity and a terminal receipt do not
provide that authority. Relationships may remain many-to-many or local-only.

Each owner must retain the same binding identity and exact anchors under its own
current revision/permission checks. A failed second write leaves a pending,
unverified binding, not an atomic cross-system success. Reconciliation must
re-read both sides, reject mismatched/stale/revoked scope, and reuse an exact
idempotency identity after response loss. Source observations do not grant Work
effect authority or accepted completion. Implementing/activating this protocol
requires the cloud Work owner and approved source transport; it is deliberately
not replaced with a local `verified` flag or a second tracker.

Focused offline receipts are retained under
`.vrooli/evidence/local-work-bridge-20261106/terminal-receipts-20261107/`.
These cover real isolated SQLite transactions, immutable read replay, file
reopen, concurrent writers/handles, injected statement/deferred-COMMIT failures,
and finite test-process exits before COMMIT or after COMMIT before an owner
response. Reopening after the committed exit retains the exact receipt; stale
retry does not duplicate it. This does not prove power-loss/filesystem durability,
external delivery acknowledgment, live Events delivery, reciprocal Work binding,
full-scenario certification or unattended pickup.

### Dormant native outbox and Events acknowledgment (7 November 2026)

`database.NewTerminalOutbox` requires both the explicit receipt-enabled repository
and an explicit trusted Events acceptor. Nothing calls it in production. Its
finite `Dispatch` selects at most 100 pending immutable receipts, releases its DB
readers, and passes the original seed bytes and event ID to that acceptor. It
creates no transport, worker, timer, route, grant or configuration toggle. A
caller still needs authorized source/transport composition before any real use.

The shared eventbus retained-intent codec rejects missing occurrence, changed
identity, oversized/ambiguous/noncanonical seeds and malformed acknowledgment.
It deterministically encodes both the Struct inside Any and the outer protobuf.
The event ID remains the SHA-256 of the original captured JSON, while the returned
canonical digest names the exact Events protobuf bytes. These hashes are distinct.

Events' inactive `acceptTerminalEventIntent` joins canonical event insertion,
existing subscription matching/fan-out, and a durable replay receipt in the
existing owner transaction. It returns no success before COMMIT. A retained replay
receipt binds the exact intent and canonical bytes and original store ID. It is
independent of canonical-event retention, so a retry after pruning does not
reinsert or fan out the old event. A generic duplicate event without the receipt
is a conflict, not acceptance. The receipt is not proof the canonical event is
still readable, that a subscriber received it, or that work was accepted.

Agent Manager records a matching acknowledgment in an append-only ledger separate
from the observation. Errors, response loss, cancellation, invalid acknowledgments
and failed local COMMIT leave the original intent pending. Retry reuses its ID
and bytes. Concurrent callers can send duplicates; both owner ledgers reconcile
them without rewriting source history. Uncertain outcomes never invent success.
An acknowledged intent is no longer selected after reopen, even if its mutable
Run was deleted. The original receipt retains its capture-time state.

The outbox itself does not activate the Flywheel wake selector or any subscription.
The subsequent dormant projection below adds explicit terminal selectors without
authorizing delivery. No native terminal event is sent to Flywheel and no selected
terminal-evidence route is authorized. The existing investigation adapter stays
`run_unqualified`; the distinct terminal reader can reach only an unverified-lineage
hold. There is no historical backfill. Automatic ledger reclamation is not
implemented; it requires an explicit replay-retirement protocol.

Offline qualification and exact cross-owner fixture receipts are retained under
`.vrooli/evidence/local-work-bridge-20261106/terminal-outbox-20261107/`. Fixture
success is not live operational acceptance, power-loss qualification, or full
scenario certification.


### Dormant selected terminal evidence handoff (7 November 2026)

`database.NewTerminalEvidenceReader` extends Agent Manager's existing receipt and
acknowledgment owner. It accepts one exact native source instance/epoch, run UUID,
decimal lifecycle version, observation digest, terminal event ID and canonical
Events digest. A trusted composition must supply current read authority for that
exact selection. The reader checks it before storage access and again before
release; denial, revocation or cancellation returns no evidence. This is an
inactive seam, not a deployed authorization service or externally reachable route.
Selection is bounded by identity and output shape, not an internal wall-clock
timeout. A transport composition must set a caller deadline and use a
context-respecting authority callback; the reader rejects cancellation on callback
return but cannot interrupt a callback that ignores context and never returns.

Selection verifies the persisted native installation binding, original immutable
receipt bytes and separately retained Events acknowledgment. It additionally
checks all six event-intent payload facts against the receipt, so individually
valid but inconsistent hashes cannot join. The result schema is
`agent-manager.selected-terminal-run-evidence.v1`. It contains native identity,
terminal status, fresh selection `observed_at`, a digest-versioned
`agent-manager://runs/<run>/terminal-receipts/<version>` reference and explicit
`lineage: null`. Task identifiers, transcript, labels, paths and mutable run
metadata are not released. `complete`, `failed`, `cancelled` and `unknown` remain
observations, not accepted work. This read does not claim current-run status,
process liveness, complete source coverage or a current invocation after a later
continuation. Historical/imported runs without a prospective receipt remain
unqualified, and missing acknowledgments cannot qualify a receipt.

The existing Events bridge now recognizes `agent_manager.run.terminal.v1` through
an additive `agent-manager.run-terminal.v1` locator. The locator carries the native
source instance/epoch, run, lifecycle and observation digest. Its outer Events
source binding remains distinct: an Agent Manager installation identity cannot
stand in for Events read authority. The canonical selected-event projection
returns the terminal status and exact protobuf digest; the wake omits status.
Legacy category-based notifications do not acquire terminal-selector authority.
No production selector registration or subscription activation is implied.

The cloud terminal branch first re-reads the exact selected Events evidence, then
passes its verified canonical digest and terminal locator to the native reader.
It compares the returned receipt, event, source and immutable reference before
considering Work lineage. The isolated three-owner fixture uses the original
Agent Manager SQLite receipt, actual Events SQLite admission/acknowledgment and
unmodified native selected evidence. The cloud's 18 focused fixture checks passed:
the exact join reaches `lineage_unverified`, with no Work read, Work mutation or
accepted-completion claim. Unknown terminal status remains a hold. The legacy
investigation adapter continues to return `run_unqualified`. Broader cloud
aggregate/build/readback verification is tracked separately by the coordinated
qualification and must not be inferred from these 18 checks.

Native focused race evidence is retained under
`.vrooli/evidence/local-work-bridge-20261106/terminal-reader-20261107/`.
The selected native fixture is
`terminal-outbox-20261107/final-cross-owner-fixture/selected-terminal-evidence.json`
under the same local-work bridge evidence root; its fresh selection timestamp is
fixture evidence, not ongoing monitoring. Events canonical/selected/wake fixture
bytes are retained in `scenarios/vrooli-events/bridge/testdata/`. Their timestamp
alignment uses test evaluation clocks, not rewritten source observations.

This closes a scoped W3 offline selected-evidence handoff. It does not establish
W0–W2 readiness, full-scenario certification, deployed read permissions, real
native export, reciprocal Work bindings or unattended pickup. Activation still
requires separately authorized owner/transport composition. No credential, grant,
service lifecycle, native source backfill or real Work mutation was performed.
