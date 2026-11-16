# Advisory operations

## Startup and health

Start and stop Git Control Tower through the Vrooli lifecycle. Health means the
API process and database are reachable; it does not mean Workspace Sandbox,
Search Hub, Integration Hub, Test Genie, or host publication are available.
The UI must display each dependency's capability standing independently.

## Durable jobs and recovery

Review and mention rows are SQLite-owned by their domains. A request key is
idempotent. On restart, queued/running rows are rehydrated and provider
execution identities remain attached. A failed requested check remains a
failed check; a newer unrelated provider run cannot replace it. Retention must
not remove active jobs or evidence referenced by a result.

## Evidence and privacy

Change subjects and evidence bundles are bounded by repository identity,
revision/digest, scope, coverage, and omissions. `available=false` is distinct
from an available empty result. Public mention output must omit private work
references and internal artifact paths. A provenance path match is not
committed authorship.

## Staged commit messages

The Commit panel separates Title, Description, and optional References and
trailers. Git stores these as one message separated by blank lines. Suggest
first collects evidence without starting a model. A reusable message is offered
only when its retained changes survive in the staged selection. Partial coverage
is disclosed; accepting a suggestion fills the editable fields. Committing
remains a separate operator action. A confirmed UI commit (success plus a commit
hash) clears the completed suggestion session and its browser recovery handle,
so Suggest starts a fresh evidence check next time. Acceptance alone, refused or
uncertain commits, and ordinary staging changes preserve the draft. Running or
unconfirmed jobs retain their handle for cancellation/reconnect. A commit cannot
clear a newer investigation or a different repository's session; server-side job
history remains available.

History **Continue as** adds continuation references to the current message. It
prefills the continuation title/message only when the editor is empty. Existing
titles, descriptions and custom trailers survive; selecting the same history
item again does not duplicate its references. AI adoption also retains these
trailers, so either action can come first.

`AdvisoryService.CollectStagedMessage` retains an immutable selection using HEAD
and the index's path, mode, and blob identities. Working-tree content is never
substituted for staged bytes. `CheckStagedMessage` compares this identity before
starting or applying a suggestion. An edit outside the index does not invalidate
it. A changed staged selection keeps the old result visible but requires fresh
evidence before use. Binary, large, unavailable, and budget-limited content has
explicit omissions.

Selections can contain up to 4,096 staged files. That limit applies to the
complete changed index manifest, independently of detailed evidence size. Large
selections use a bounded sample across affected scopes, prioritizing source over
obvious generated and lockfile bulk. Missing patch detail does not remove files
from the selection; exact retained blob/mode correspondence can still support
reusable proposal text without rereading every file.

The complete subject remains in GCT's durable job store. New drafting jobs send
a compact summary and representative detail to Agent Manager: at most 48 KiB
of draft context and 56 KiB for its input envelope, within the existing 64 KiB
structured-input limit. Whole patches are preferred over incomplete excerpts.
Stable file references map detailed inspection back to the complete subject;
unspecified or unsupported file coverage remains uninspected. New deeper drafts
report message scope separately: a whole-selection synthesis uses the complete
inventory and selected evidence without claiming every patch was read. The agent
need not echo thousands of full paths. Older admitted jobs
retain their input format and owner execution rather than changing subjects or
launching replacement work.

Durable `DraftInput` JSON has one current schema, format 4. Formats 2 and 3 are
read-only compatibility inputs: format 2 retains the original representative
draft and omission rules; format 3 retains deep evidence without synthesis
scope claims. The persisted-input decoder accepts exactly formats 2, 3, and 4;
missing, zero, negative, and unsupported future versions are invalid. The
`DraftInput.semantics()` adapter maps accepted stored versions to runtime
capabilities. New deep investigations and continuations write format 4. Never
rewrite a stored input, snapshot, or format number during recovery or retry;
retry must preserve the original owner key and payload. Remove a legacy mapping
only after durable-job retention and backup evidence establish that no format
2 or 3 row can be restored, and remove its oldest-format regression in the same
change.

The dialog labels the pinned input as "Staged selection: N files". This count
identifies the files selected for the commit, not the completeness of a draft.
Whole-selection and limited draft scope, detailed inspection, and evidence
limits are shown separately. A limited draft puts its supplied reason under
"Why this draft is limited", including concrete evidence-access failures. Groups and long evidence/path lists use expandable
previews. A detail sample is an evidence budget, never a filter on what the
eventual commit contains.

An agent's quick draft uses a representative sample and may be partial. Choose
**Investigate remaining changes** beside **Use partial suggestion** to start one
explicit deeper investigation. It inherits the completed partial job's exact
staged subject, selected context and operator note. It first obtains a bounded
inventory of the complete selection, then chooses representative or important
patches from recorded Git blob IDs. Full patch traversal is optional. It does not
recollect the index or treat later working-tree content as staged evidence. No matching saved
message or related run does not mean the staged-diff evidence is absent.

The previous draft remains available during investigation and after failure,
cancellation, or another partial result. Adoption stays blocked while execution
or admission is unresolved, and still verifies the staged digest before filling
the editor. Binary, oversized and unexamined patches remain disclosed inspection
limits. They do not automatically make the message incomplete. A draft with
unresolved change themes stays limited and explains the uncertainty. Message
scope is the agent's synthesis assessment, not a certification of exhaustive review.
Legacy results retain their original coverage meaning.

Workflow token limits count cumulative input, cached input and output across
model calls. They are not context-window limits. The quick workflow permits
262,144 cumulative tokens; deeper investigation permits 2,000,000. Time, charge,
tool and child limits still apply. Budget exhaustion remains a failed workflow
even when its child completed a draft. The drawer shows that failure before the
previous draft. Older failed jobs with no saved explanation show an explicit
missing-explanation message; they are not silently retried or rewritten.

Result `source_ids` can cite only selected related-work `contexts[].id`; return
an empty array when no related context was selected. Stable file IDs belong in
`covered_file_ids`. Unsupported source references fail validation. Prompt and
schema descriptions explain this distinction but do not guarantee model
compliance; evidence/result validation remains authoritative.

After changing a workflow declaration or its resolved skill, use the canonical
`agent-manager declarations reconcile-scenario --scenario git-control-tower`
operation and inspect the active revision, budget, source hash and prompt
freshness. Restarting GCT alone updates its API/UI but does not reconcile AM's
catalog. AM's canonical startup also performs the owner reconciliation sweep;
use lifecycle safeguards before restarting it. A healthy API does not prove
that the new workflow revision is active.

The CLI equivalent uses a fresh key and parent job:

```bash
git-control-tower message start --repo-id REPO --continue-from PARTIAL_JOB --snapshot DIGEST --request-key NEW_KEY --json
git-control-tower message evidence DEEP_JOB --repo-id REPO --snapshot DIGEST --inventory --cursor 0 --max-bytes 16384 --json
git-control-tower message evidence DEEP_JOB --repo-id REPO --snapshot DIGEST --file-id 17 --file-id 83 --max-bytes 16384 --json
```

The evidence command returns one bounded page. Inventory reads expose stable
file IDs and metadata without loading Git blobs; targeted patch reads use those
IDs. Follow inventory `nextCursor` until `complete` to account for the selection,
then select detail according to importance and uncertainty. For inventory or
sequential reads, `complete` means the manifest ended. For targeted reads, it
means all requested IDs were supplied; retry important unsupplied IDs explicitly.
`completePatch` describes each file independently. An absent/default edit count
may mean the initial sample did not measure the file, rather than zero changes. Verified owning workflow children require only
`git-control-tower:read`. Whole-patch supply receipts stay with the existing
durable job and do not change its admission/cancellation revision. Human reads
do not create agent coverage receipts. The AM operator's delegated scope ceiling
must include that same read capability; a profile declaration alone cannot grant
it. Deployment uses the existing exact owner binding, never a private token or
an ambient server identity. Verified workflow coordinates use the canonical
Agent Manager metadata contract and retained workflow membership. An agent's
environment labels alone do not establish ownership. Before GCT has saved the
execution handle, it explicitly reads the retained execution input to verify
the same immutable job; the metadata-only execution read cannot provide that
proof. First-tool requests remain valid before the workflow launch ACK.

The drawer puts the draft or current step before supporting evidence. Optional
context and a single Review evidence section keep the main action visible on
mobile. Starting and cancellation show feedback immediately. A saved GCT request
without an Agent Manager execution handle is shown as confirming launch, not as
an agent already running. Technical error details remain expandable beside the
human-readable failure and recovery action.

Related Agent Manager events are optional context. Path and changed-text
matches, plus activity near file modification hints, guide bounded retrieval.
File times can change without content changes, disappear for partial staging,
or describe a different working-tree version. They are retrieval hints, never
proof of staging time or authorship. Imported-session freshness and repository
scope remain visible limitations. Selecting context does not create an
attribution trailer.

Start drafting agent explicitly admits a durable message-only job through
`StartStagedMessage`. Retain its job ID and request key. `GetStagedMessageJob`
reattaches by either handle; `WaitStagedMessageJob` follows the same job with a
bounded server wait. Hiding the panel or disconnecting does not cancel it.
`CancelStagedMessageJob` requests cancellation; the UI waits for confirmed
terminal standing. A lost start response must be reconciled using the original
request key, never by creating a fresh execution.

Public `GetStagedMessageJob` is observational: it projects the last durable
receipt and does not dispatch, reconcile with Agent Manager, or mutate job
state. The server-owned settle worker advances active owner executions and
recovers queued/dispatching rows left by a process restart before dispatch. It
uses the immutable input and the same durable owner idempotency key; dispatch
claim and lifecycle revision checks make it safe alongside a live Start or a
cancellation request. A row with a confirmed maintenance refusal remains
paused until an explicit operator retry.

The browser retains the exact admission intent before Start. A temporary
NotFound after a lost response does not authorize a new identity: reconnect or
retry the same frozen request. Late observer responses cannot clear an observed
cancellation request or revive a confirmed terminal job.

GCT forwards managed Cloudflare or sign-in proof to Agent Manager when present.
Without verified proof, the caller is programmatic; it is never labeled human.
Agent Manager permits this fallback only for Connect starts of the two GCT
staged-message workflow keys. It does not rely on peer IP because proxies and
container networking can change that address. It strips caller-supplied
initiator and agent claims, then applies the workflow revision's trigger policy,
input validation, budget, and self-trigger checks. Other Agent Manager
operations remain under their regular identity policy. The workflow only drafts
message text; applying it fills the editor and committing remains a separate
operator action. The personal OS provider abstains for verified agents so
shared middleware retains their bounded capabilities.

Agent Manager owns the declared `git-control-tower/staged-message` workflow and
its configured model. The protected draft profile cannot publish repository
changes and receives no API mutation scopes. The workflow returns structured
message fields and coverage; GCT validates them against the retained input.
Results are previewed for Use or Discard. If the operator edited the message
meanwhile, the UI compares the two before replacing text and preserves existing
references and unknown trailers.

The declaration assets live under `.vrooli/agent-manager/`; scenario lifecycle
setup reconciles them through Agent Manager. An unavailable or denied context
source does not prevent local retained-message suggestions. An unavailable
drafting owner is shown explicitly; GCT does not substitute another identity or
invent a result.

## Shutdown and backup

Before shutdown, allow the lifecycle owner to finish its server-owned operation
or leave a durable running/interrupted row. SQLite backups must include the
domain tables and be restored into a disposable validation instance before
claiming recovery. Provider caches and Search Hub indexes are projections and
must be rebuilt from owner evidence.

## Reliability verification map

### Failure classification and recovery matrix

The persisted job is authoritative for what GCT has confirmed. `Known` fields
carry evidence, not defaults: unknown start, token, or charge values stay
unknown until the owner supplies a receipt. A request or owner-read failure
does not by itself prove that work did not start. Historical rows without a
durable cause remain unclassified.

| Failure category | Origin and persisted standing | User-visible explanation and recovery | Source regressions |
|---|---|---|---|
| Provider unavailable or closed | An owner-issued Workspace Sandbox maintenance refusal with explicit prelaunch/no-work evidence becomes `awaiting_provider`; the durable receipt proves no start, zero tokens, and zero charge. A generic unavailable response after dispatch remains uncertain with its original request identity. | The UI explains the provider wait and only enables a new job after a separate readiness read is freshly known open. Generic outages reconnect to the existing job. | `TestStagedMessageAgentReadinessAcceptsOnlyFreshOwnerObservation`, `TestStagedMessageAgentProjectsMaintenanceRefusalAsDefinitiveNoStart`, `TestMaintenanceRefusalPersistsAwaitingProviderAndRequiresExplicitNewJobRetry`, `TestOwnerReadAuthenticationFailureDoesNotInventTerminalExecution` |
| Uncertain dispatch acknowledgement | The job remains `dispatching` with `dispatch_unconfirmed`, no invented execution handle, and no asserted work or charge certainty. Cancellation resolves the same owner key through its fence. | Keep the frozen request; reconnect by its original request key. Do not submit a fresh Start while acceptance is unresolved. | `TestLostAdmissionAcknowledgementReconcilesSameDurableKey`, `TestLostAdmissionAcknowledgementSurvivesLaterAuthenticationFailure`, `TestCancelledAmbiguousDispatchAdoptsHandleAndSettleFinishesCancellation`; UI `recovers an uncertain launch by reconnecting to the same request without redispatch` |
| Authentication or policy denial | A definitive first admission denial is terminal `failed` with `authentication_required` or `policy_denied`; work and charge remain unknown unless receipts say otherwise. If access fails while reading an already-attached execution, the job stays attached and records the access cause without claiming that execution stopped. | For a first-admission denial, restore access and explicitly start a new suggestion. For an attached execution, restore access and reconnect to the same job; do not create a replacement. | `TestClassifyDispatchErrorsByRecoveryAction`, `TestFirstDefinitiveAdmissionRejectionRemainsTerminal`, `TestOwnerReadAuthenticationFailureDoesNotInventTerminalExecution`; UI `explains persisted authentication_required` and `explains persisted policy_denied without implying the saved execution stopped` |
| Budget exhaustion | The owner terminal reason is retained as `budget_exhausted` with its budget name/message. Start, tokens, and charge are shown only from measured owner receipts. | The failed job retains its explanation and evidence. Start a new suggestion only after reviewing the limit; no same-key redispatch is implied. | `TestStagedMessageBudgetFailureExplainsActualOwnerReason`, `TestStagedMessageAgentProjectsChargeOnlyFromMeasuredReceipt` |
| Cancellation | `cancelling` persists until owner confirmation. A fenced pre-start cancellation records known no-start, zero tokens, and zero charge; an accepted execution retains owner work and charge receipts. | The UI says cancellation is pending until confirmation, then shows `cancelled`; reconnect continues the same owner identity. | `TestCancelBeforeOwnerStartDoesNotDispatch`, `TestCancelFenceWinsAgainstBlockedStartBeforeOwnerAdmission`, `TestCancelRacingAcceptedStartAdoptsAndCancelsOwner`, `TestCancelledAmbiguousDispatchAdoptsHandleAndSettleFinishesCancellation`; UI `keeps the owner job while hidden and cancellation pending until owner confirmation` |
| Invalid output | Malformed or out-of-scope structured output becomes terminal `failed` with `invalid_structured_output`; any known work, token, and measured charge receipt is preserved. No transcript fallback manufactures a result. | The UI shows the failure and offers a separate new suggestion. Deep retries use the retained immutable input; applying a result remains a separate operator action. | `TestMalformedStructuredResultFailsWithoutTranscriptFallback`, `TestResultCoverageRejectsMissingDuplicatedAndInventedScope`; UI `shows failures with retry evidence collection and never restarts an agent automatically`, `retries a failed deep draft as a new job over the same saved selection` |
| Result or evidence read failure | A typed owner result/evidence read error persists as `evidence_unavailable`. When the owner response includes its handle and receipts, GCT saves them; otherwise it preserves the prior certainty. A successful reconnect clears the transient category. | The active job shows the read failure and a reconnect action that reads the same job; it never creates a replacement execution. | `TestStagedMessageAgentClassifiesResultReadFailureAsEvidenceUnavailable`, `TestEvidenceReadFailureKeepsJobRecoverableAndClearsOnReconnect`, `TestEvidenceReadFailureAtStartPersistsHandleAndWorkReceipt`; UI `surfaces a persisted active-job evidence error and reconnects to that same job`, `shows a persisted active-job failure explanation with same-job recovery` |
| Stale execution or selection | A saved execution is marked `stale` only when Agent Manager returns typed NotFound. A changed captured selection cannot start or be applied as though still current. | The UI says the saved execution is missing or the selection changed; capture and queue the current files as a new request. | `TestJobStatusMarksStaleOnlyOnOwnerNotFound`, `TestStaleCollectionCannotStartAgent`; UI `retains an inapplicable result as stale and explains how to queue the current files` |
| Queue conflict | An ownership or revision mismatch returns a conflict and preserves the current queue owner; it does not replace an active job. | Refresh the queue and act on its current entry. A failed/orphaned entry is retried only after its old job is terminal or missing. | `TestWorktreeRefreshReplacesDurableQueueEntryAtomicallyAndReplays`, `TestFailedQueueEntryCanRetryWhenItsSuggestionJobWasDeleted`, `TestWorktreeQueueRejectsPathOwnershipConflictsTransactionally`; UI `surfaces a persisted queue failure without attempting a job lookup` |
| Storage failure | GCT returns a safe internal error when durable storage fails. No durable failure classification can be promised while the storage owner is unavailable; the last committed row, if any, is the limit of known standing. | Reconnect by the original job ID or request key after storage recovers. If dispatch may have happened, reconcile that identity before considering a new request. | `TestStagedMessageStorageErrorsAreSafeInternalFailures`, `TestQueuedSelectionCanReplaceOrphanedSuggestionJob`, `TestLostAdmissionAcknowledgementReconcilesSameDurableKey` |

The UI regression names in this matrix refer to `CommitMessageComposer.test.tsx`
or `QueuedStageSuggestionStatus.test.tsx` under `ui/src/`. These tests cover the
operator-facing path; API and storage tests remain authoritative for durable
classification and certainty.

These tests are the owning regressions for the staged-message agent reliability
contract. Run the Agent Manager and Git Control Tower `unit` Test Genie phases
after changes to these paths. Focused package tests provide faster local
feedback; the Test Genie receipt is the scenario-level validation record.

| Requirement | Authoritative regression coverage |
|---|---|
| Fresh provider readiness, a typed maintenance refusal, and an operator-controlled retry after readiness opens | `TestStagedMessageAgentReadinessAcceptsOnlyFreshOwnerObservation`, `TestStagedMessageAgentProjectsMaintenanceRefusalAsDefinitiveNoStart`, `TestMaintenanceRefusalPersistsAwaitingProviderAndRequiresExplicitNewJobRetry`, `TestMaintenanceRetryRequiresKnownOpenProviderReadiness` |
| Failure classification, budget feedback, and truthful work/result evidence | `TestClassifyDispatchErrorsByRecoveryAction`, `TestFirstDefinitiveAdmissionRejectionRemainsTerminal`, `TestLostAdmissionAcknowledgementReconcilesSameDurableKey`, `TestLostAdmissionAcknowledgementSurvivesLaterAuthenticationFailure`, `TestOwnerReadAuthenticationFailureDoesNotInventTerminalExecution`, `TestStagedMessageBudgetFailureExplainsActualOwnerReason`, `TestMaintenanceCodeWithStartedReceiptDoesNotClaimProviderWaitOrZeroCharge`, `TestMalformedStructuredResultFailsWithoutTranscriptFallback`, `TestResultCoverageRejectsMissingDuplicatedAndInventedScope`, `TestStagedMessageAgentClassifiesResultReadFailureAsEvidenceUnavailable`, `TestEvidenceReadFailureKeepsJobRecoverableAndClearsOnReconnect`, `TestStagedMessageStorageErrorsAreSafeInternalFailures` |
| Observational reads, idempotent reconnect, and retry from the saved input | `TestJobGetIsObservationalAndAdvanceDispatches`, `TestDurableAdmissionIsIdempotentAndReattachesByRequestKey`, `TestRetryDraftUsesSavedInputWithoutBrowserAdmission`, `TestLegacyAdmissionRetryUsesOriginalDurableInputAndRejectsChangedIntent` |
| Stale snapshots, queue ownership, and concurrent admission | `TestStaleCollectionCannotStartAgent`, `TestStagedDriftRetainsResultAndDoesNotRestart`, `TestWorktreeRefreshReplacesDurableQueueEntryAtomicallyAndReplays`, `TestConcurrentAdmissionRetainsOneJobAndOneOwnerIdempotencyIdentity`, `TestConcurrentRetryAdmissionCannotBeTerminatedByOlderAuthenticationRefusal` |
| Agent Manager run action availability, request validation, owner failures, and response projections | `TestAgentRunEventsBoundsAndProjection`, `TestAgentRunDiffProjectsFiles`, `TestAgentRunEventsRejectInvalidQueryBounds`, `TestAgentRunActionsRejectUnavailableAndMissingRunID`, `TestAgentRunActionResponseProjections`, `TestAgentRunActionsMapOwnerErrors` |
| Advisory Connect request validation and nested evidence projection | `TestAdvisoryConnectDraftValidatesAndReturnsDraft`, `TestAdvisoryConnectDraftRejectsMissingAndInvalidRequests`, `TestAdvisoryProtoConversionsPreserveNestedEvidenceAndIgnoreNilEntries` |
| Durable recovery queue paging, cancellation filtering, and stored-state validation | `TestListRecoverableAdvisoryFiltersCancelledRowsBeforePaging`, `TestListAdvisoryUsesRecentFirstAndValidatesStoredState` |
| Cancellation before, during, and after owner admission, including terminal receipts | `TestCancelBeforeOwnerStartDoesNotDispatch`, `TestCancelAfterAttemptMarkerUsesOwnerFenceWithoutStarting`, `TestCancelFenceWinsAgainstBlockedStartBeforeOwnerAdmission`, `TestCancelRacingAcceptedStartAdoptsAndCancelsOwner`, `TestCancelledAmbiguousDispatchAdoptsHandleAndSettleFinishesCancellation`, `TestCancelFencedStartAdoptsTerminalOwnerReceiptWithoutRecancel`, `TestWorkflowExecutionCancellationFenceSerializesStartAdmission`, `TestCancelWorkflowExecutionByKeyReturnsTerminalStartReceipt` |
| UI state, elapsed time, and operator-visible failure | `ui/src/components/QueuedStageSuggestionStatus.test.tsx`, `ui/src/features/commit-message/CommitMessageComposer.test.tsx`; the latter also runs under `pnpm type-check` |
| Durable failure and stuck-work signals | `TestMetricsAggregateDurableLifecycleEvidence`, `TestMetricsTracksDispatchingUntilOwnerHandleIsSaved`, `TestMetricsCountsProviderWaitWithoutCallingItFailedOrStuck`, `TestMetricsClassifiesOnlyProvenLegacyMaintenanceRowsAsProviderWait` |
| Exact executor identity, bounded inventory, and fail-closed compatibility with an older root CLI | `TestControlPlaneExecutorScopeUsesDurableIdentityAndCompleteProof`, `TestControlPlaneExecutorScopeKeepsOrdinaryCoverageFromOldIncompleteOwner`, `TestControlPlaneExecutorScopeBatchesHistoricalReferences`, `TestControlPlaneExecutorScopeOldCountLimitFallbackIsNonAtomic`, `TestControlPlaneExecutorScopeSplitsOversizedInventoryByJSONBytesAndMarksNonAtomic`, `TestExecutorScopePaginatesReferencesAgainstOneHostSnapshot` |
| Workspace Sandbox fence recovery and PID-reuse protection | `TestProviderLifecycleFencePersistsAcrossReplacementAndRequiresExactResume`, `TestProviderLifecycleDrainIgnoresExitedProcessWhosePIDWasReused`, `TestStartOperationFenceOwnerFindsExactSuccessfulOwnerAcrossLaterUnlinkedStart` |

These regressions do not make uncertain runtime evidence complete. The installed
root CLI includes bounded executor-reference paging and one complete systemd
scope snapshot for gone-scope checks and Web Console session correlation on
attached runs (binary SHA-256
`817b5600853d5fa5ab890aac4b1fe4dd19d9c69cb600b353c6a0a67b46a87a16`). A live
read of 70,187 retained references completed in 3.2 seconds: 29,207 were proven
absent and 40,980 remain unknown because their old attached-session bindings
lack current process/scope proof. The former timeout is fixed; the historical
unknowns are not timeout errors and must not be backfilled as dead.

The current Agent Manager maintenance projection is narrower and reports one
unresolved attached run, `fffdf89f-ca80-4d20-bbc9-da2908b5a741`. Its durable run
is complete, its recorded leader PID is absent, and its event stream records an
exit, but the old record has no established containment provenance to prove
that descendants are gone. Maintenance therefore still fails closed. Keep the
run and its uncertainty intact until the lifecycle owner has an authoritative
scope proof or a supported, compare-and-swap recovery for this legacy case.
This limits claims that a physical drain is complete; it does not block the
operator's normal supported Agent Manager restart, which preserves detached
executors and does not require Agent Manager to self-drain. Record the original
run IDs and verify their recovery after that restart.
Historical failures with no durable cause remain unclassified; do not backfill
a guessed cause.

## Known external activation gaps

Live Integration Hub host credentials, provider publication receipts, and
Workspace Sandbox committed-content evidence are not fabricated by GCT. The
activation packet must show configured, probed, scoped, and human-authorized
standing separately.
