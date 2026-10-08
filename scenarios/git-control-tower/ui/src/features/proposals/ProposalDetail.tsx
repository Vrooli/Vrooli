import { useEffect, useRef, useState } from "react";
import {
  HelpCircle,
  AlertTriangle,
  CheckCircle2,
  Loader2,
  RefreshCw,
  Save,
  ShieldCheck,
  Undo2,
} from "lucide-react";
import type { MutationPreview } from "@vrooli/proto-types/git-control-tower/v1/human_control/human_control_pb";
import {
  FreshnessState,
  IssueSeverity,
  ProposalState,
  type ApplyProposalResponse,
  type Proposal,
} from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import {
  useApplyProposal,
  useEditProposal,
  usePrepareProposalApproval,
  useRefreshProposal,
  useWithdrawProposal,
} from "../../lib/hooks-proposals";
import {
  hasPriorWork,
  epochsLabel,
  freshnessMeta,
  proposalEpochs,
  resolutionMeta,
  shortOid,
} from "./model";
import { GuidedCommitButton } from "./ProposalGuidance";
import {
  ProposalReviewGuidance,
  reviewBlocker,
} from "./ProposalReviewGuidance";
import { useSessionState } from "./sessionState";

interface ProposalDetailProps {
  proposal: Proposal;
  repoId: string | null;
  /** False when the caller cannot approve (agents, read-only sessions). */
  canApprove: boolean;
  approveDisabledReason?: string;
  /** Device preference shared with the commit dialog ("don't ask again"). */
  skipConfirmation?: boolean;
  onBack: () => void;
  onOpenRelated?: (id: string, revision?: number) => void;
  historical?: boolean;
  comparingCurrent?: boolean;
  diffDisabledReason?: string;
  onCompareCurrent?: () => void;
  onReturnToProposal?: () => void;
  onOpenWorkingChanges?: () => void;
  onRetryReview?: () => void;
  relatedProposals?: Proposal[];
  onCommitted?: (oid: string) => void;
}

/** Review, edit and approve one proposal. Approval is one human action: the
 *  server stages exactly these files and commits this message, or refuses
 *  and says why. */
export function ProposalDetail({
  proposal,
  repoId,
  canApprove,
  approveDisabledReason,
  skipConfirmation = false,
  onBack,
  onOpenRelated,
  relatedProposals = [],
  historical = false,
  comparingCurrent = false,
  diffDisabledReason,
  onCompareCurrent,
  onReturnToProposal,
  onOpenWorkingChanges,
  onRetryReview,
  onCommitted,
}: ProposalDetailProps) {
  // Unsaved edits live in a draft that survives refetches and new revisions
  // (GCT-053): a revision arriving mid-edit never discards operator text.
  const [draft, setDraft] = useSessionState<{
    subject: string;
    body: string;
    revision: number;
  } | null>(`gct.proposals.draft.${proposal.id}`, null);
  const [preview, setPreview] = useState<MutationPreview | null>(null);
  const [skipPrecommit, setSkipPrecommit] = useState(false);
  const [confirmingWithdraw, setConfirmingWithdraw] = useState(false);
  const [closeReason, setCloseReason] = useState("");
  const [replacementId, setReplacementId] = useState("");
  const replacement = relatedProposals.find(
    (item) => item.id === replacementId,
  );
  const replacementPaths = new Set(replacement?.files.map((file) => file.path));
  const omittedByReplacement = proposal.files.filter(
    (file) => !replacementPaths.has(file.path),
  );
  const [acknowledgePriorWork, setAcknowledgePriorWork] = useState(false);
  const [guide, setGuide] = useState<"blocked" | "version" | "update" | null>(
    null,
  );
  const [result, setResult] = useState<ApplyProposalResponse | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const edit = useEditProposal(repoId);
  const refresh = useRefreshProposal(repoId);
  const withdraw = useWithdrawProposal(repoId);
  const prepare = usePrepareProposalApproval();
  const apply = useApplyProposal(repoId);
  const reviewKey = JSON.stringify([
    proposal.id,
    proposal.revision,
    proposal.state,
    proposal.captured,
    proposal.proposalDigest,
    proposal.freshness?.state === FreshnessState.FRESH ||
    proposal.freshness?.state === FreshnessState.DRIFTED
      ? "reviewable"
      : proposal.freshness?.state,
    proposal.freshness?.baseChangedPaths,
    proposal.files
      .filter((file) =>
        file.flags.some((flag) => flag.code === "staged_version_differs"),
      )
      .map((file) => file.path),
    proposal.freshness?.foreignStagedPaths,
    acknowledgePriorWork,
    comparingCurrent,
    diffDisabledReason,
    historical,
    canApprove,
    draft,
  ]);
  const latestReview = useRef(reviewKey);
  latestReview.current = reviewKey;

  // A new revision invalidates any open preview: an approval always names the
  // revision on screen.
  useEffect(() => {
    setPreview(null);
    setAcknowledgePriorWork(false);
  }, [proposal.id, proposal.revision]);

  const serverSubject = proposal.message?.subject ?? "";
  const serverBody = proposal.message?.body ?? "";
  const subject = historical
    ? serverSubject
    : (draft?.subject ?? serverSubject);
  const body = historical ? serverBody : (draft?.body ?? serverBody);
  const updateDraft = (next: { subject?: string; body?: string }) =>
    setDraft({
      subject: next.subject ?? subject,
      body: next.body ?? body,
      revision: draft?.revision ?? proposal.revision,
    });

  const isOpen = proposal.state === ProposalState.OPEN && !historical;
  const epochs = proposalEpochs(proposal);
  const freshness = proposal.freshness;
  const freshnessInfo = freshnessMeta(freshness?.state);
  const isFresh = freshness?.state === FreshnessState.FRESH;
  const hasNewerEdits = freshness?.state === FreshnessState.DRIFTED;
  const affectedBase = (freshness?.baseChangedPaths.length ?? 0) > 0;
  const stagedVersionsDiffer = proposal.files
    .filter((file) =>
      file.flags.some((flag) => flag.code === "staged_version_differs"),
    )
    .map((file) => file.path);
  const foreignStaged = freshness?.foreignStagedPaths ?? [];
  const uncertainPaths = proposal.files
    .filter((file) => hasPriorWork(file))
    .map((file) => file.path);
  const dirty =
    !historical &&
    draft !== null &&
    (subject.trim() !== serverSubject || body.trim() !== serverBody.trim());
  const draftOutdated =
    dirty && draft !== null && draft.revision !== proposal.revision;
  const busy =
    edit.isPending ||
    refresh.isPending ||
    withdraw.isPending ||
    prepare.isPending ||
    apply.isPending;
  const blocker = reviewBlocker({
    canApprove,
    operatorReason: approveDisabledReason,
    comparing: comparingCurrent,
    open: isOpen,
    historical,
    captured: proposal.captured,
    diffReason: diffDisabledReason,
    dirty,
    needsAcknowledgment: uncertainPaths.length > 0 && !acknowledgePriorWork,
    differentStaged: stagedVersionsDiffer.length > 0,
    baseChanged: affectedBase || freshness?.state === FreshnessState.BASE_MOVED,
    reviewable: isFresh || hasNewerEdits,
    foreignStaged: foreignStaged.length > 0,
  });
  const approveBlocker = blocker?.message;

  const runAction = async (action: () => Promise<unknown>) => {
    setActionError(null);
    try {
      await action();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : String(error));
    }
  };

  const saveMessage = () =>
    runAction(async () => {
      if (!subject.trim()) return;
      await edit.mutateAsync({
        id: proposal.id,
        expectedRevision: proposal.revision,
        subject: subject.trim(),
        body,
      });
      setDraft(null);
      setGuide(null);
    });

  const applyWith = (approvedPreview: MutationPreview) =>
    runAction(async () => {
      const response = await apply.mutateAsync({
        proposal,
        preview: approvedPreview,
        skipPrecommit,
        acknowledgePriorWork,
        keepLaterChanges: true,
      });
      setPreview(null);
      setResult(response);
      if (response.success) onCommitted?.(response.commitOid);
    });

  const handleApprove = () =>
    runAction(async () => {
      setResult(null);
      const prepared = await prepare.mutateAsync({
        proposal,
        acknowledgePriorWork,
        keepLaterChanges: true,
      });
      if (latestReview.current !== reviewKey) return;
      if (skipConfirmation) {
        await applyWith(prepared);
        return;
      }
      setPreview(prepared);
    });

  return (
    <section
      className="flex h-full min-h-0 flex-col"
      aria-label="Commit proposal"
      data-testid="proposal-detail"
    >
      <header className="border-b border-slate-800 px-4 py-3">
        <h2 className="text-sm font-semibold text-slate-100">
          Review &amp; commit
        </h2>
      </header>

      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-3 py-3">
        <button
          type="button"
          className="inline-flex min-h-11 items-center gap-2 text-xs text-cyan-300 hover:underline"
          data-testid="proposal-version-help"
          onClick={() => setGuide("version")}
        >
          <HelpCircle className="h-4 w-4" />
          What will this commit?
        </button>
        <p className="text-sm leading-relaxed text-slate-300">
          {proposal.message?.body.split(/(?<=[.!?])\s|\n\n/)[0] ||
            "Review the proposal version and commit message. Later working edits stay uncommitted."}
        </p>
        {epochs.length > 1 && (
          <p className="text-xs text-slate-400" data-testid="proposal-folded">
            One commit covers {epochsLabel(epochs)}. Source proposals and their
            evidence remain linked below.
          </p>
        )}
        <details aria-label="Acceptance evidence" className="space-y-2">
          <summary className="cursor-pointer text-xs font-medium text-cyan-300">
            Acceptance evidence ·{" "}
            {proposal.evidence?.gateReceipts.length || "no"}{" "}
            {proposal.evidence?.gateReceipts.length === 1
              ? "receipt"
              : "receipts"}
          </summary>
          <h3 className="text-xs font-medium text-cyan-300">
            Reported acceptance evidence
          </h3>
          <p className="text-xs leading-relaxed text-slate-400">
            Receipts describe the source work. Adopting newer contents does not
            revalidate those checks.
          </p>
          {proposal.evidence?.epochFile && (
            <p className="break-all text-xs text-slate-300">
              {proposal.evidence.epochFile}
            </p>
          )}
          {proposal.evidence?.metricBefore && (
            <p className="text-xs text-slate-300">
              {proposal.evidence.metricBefore} →{" "}
              {proposal.evidence.metricAfter || "not supplied"}
            </p>
          )}
          {(proposal.evidence?.gateReceipts.length ?? 0) > 0 ? (
            <details className="text-xs text-slate-400">
              <summary className="cursor-pointer">
                {proposal.evidence?.gateReceipts.length} supplied receipts
              </summary>
              <ul className="mt-2 space-y-1">
                {proposal.evidence?.gateReceipts.map((receipt) => (
                  <li className="break-all" key={receipt}>
                    {receipt}
                  </li>
                ))}
              </ul>
            </details>
          ) : (
            <p className="text-xs text-slate-500">
              No structured acceptance receipts supplied.
            </p>
          )}
        </details>
        {isOpen && (!isFresh || !proposal.captured) && (
          <div
            className="space-y-2 rounded-md border border-amber-800/60 bg-amber-950/20 p-3 text-xs text-amber-200"
            role="status"
            data-testid="proposal-stale"
          >
            <p className="flex items-center gap-2 font-medium">
              <AlertTriangle className="h-3.5 w-3.5" />{" "}
              {proposal.captured
                ? freshnessInfo.hint
                : "This older proposal has no saved file contents. Choose the latest working files, then review them before committing."}
            </p>
            {(freshness?.driftedPaths ?? []).length > 0 && (
              <p className="font-mono text-[11px] break-all">
                Changed: {freshness?.driftedPaths.join(", ")}
              </p>
            )}
            {(freshness?.baseChangedPaths ?? []).length > 0 && (
              <p className="font-mono text-[11px] break-all">
                Committed since base: {freshness?.baseChangedPaths.join(", ")}
              </p>
            )}
            {freshness?.detail && <p>{freshness.detail}</p>}
            <button
              type="button"
              className="inline-flex items-center gap-1.5 rounded-md border border-amber-700 px-2 py-1 font-medium hover:bg-amber-900/40 disabled:opacity-60"
              onClick={() => setGuide("update")}
              disabled={busy}
              data-testid="proposal-refresh"
            >
              <RefreshCw
                className={`h-3.5 w-3.5 ${refresh.isPending ? "animate-spin" : ""}`}
              />{" "}
              Use latest working files
            </button>
            {proposal.captured && onCompareCurrent && (
              <button
                className="min-h-11 text-cyan-300 hover:underline"
                onClick={onCompareCurrent}
              >
                Compare later edits
              </button>
            )}
          </div>
        )}
        {isOpen && foreignStaged.length > 0 && (
          <p
            className="rounded-md border border-amber-800/60 bg-amber-950/20 px-3 py-2 text-xs text-amber-200"
            data-testid="proposal-foreign-staged"
          >
            Other files are staged ({foreignStaged.join(", ")}). Unstage them so
            the commit holds only this proposal.
          </p>
        )}

        {uncertainPaths.length > 0 && isOpen && proposal.captured && (
          <div className="space-y-2 border-l-2 border-amber-600 pl-3 text-xs text-amber-200">
            <p>
              {uncertainPaths.length} whole{" "}
              {uncertainPaths.length === 1 ? "file" : "files"} may include
              earlier or unattributed work.
            </p>
            <label className="flex items-start gap-2">
              <input
                type="checkbox"
                className="mt-0.5"
                checked={acknowledgePriorWork}
                onChange={(event) => {
                  setAcknowledgePriorWork(event.target.checked);
                  setPreview(null);
                }}
                disabled={busy}
                data-testid="proposal-acknowledge-prior"
              />
              <span>
                I reviewed those files and want to include their complete
                proposal contents.
              </span>
            </label>
          </div>
        )}

        <div className="space-y-2">
          <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">
            Message
          </p>
          <input
            value={subject}
            onChange={(event) => updateDraft({ subject: event.target.value })}
            disabled={!isOpen || busy}
            aria-label="Commit subject"
            className="w-full rounded-md border border-slate-700 bg-slate-800/50 px-3 py-2 text-sm text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            data-testid="proposal-subject"
          />
          <textarea
            value={body}
            onChange={(event) => updateDraft({ body: event.target.value })}
            disabled={!isOpen || busy}
            aria-label="Commit body"
            rows={5}
            className="w-full resize-y rounded-md border border-slate-700 bg-slate-800/50 px-3 py-2 text-xs text-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
            data-testid="proposal-body"
          />
          {draftOutdated && (
            <p
              className="text-[11px] text-amber-300"
              data-testid="proposal-draft-outdated"
            >
              The proposal changed to r{proposal.revision} while you were
              editing. Your text is kept; saving applies it to r
              {proposal.revision}.
            </p>
          )}
          {dirty && (
            <div className="flex gap-2">
              <button
                type="button"
                className="inline-flex items-center gap-1.5 rounded-md border border-slate-600 px-2.5 py-1 text-xs text-slate-200 hover:bg-slate-800 disabled:opacity-60"
                onClick={saveMessage}
                disabled={busy || subject.trim() === ""}
                data-testid="proposal-save"
              >
                {edit.isPending ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Save className="h-3.5 w-3.5" />
                )}{" "}
                Save edits
              </button>
              <button
                type="button"
                className="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs text-slate-400 hover:bg-slate-800"
                onClick={() => setDraft(null)}
                disabled={busy}
              >
                <Undo2 className="h-3.5 w-3.5" /> Discard
              </button>
            </div>
          )}
          <details className="text-xs text-slate-400">
            <summary className="cursor-pointer">
              Commit trailers &amp; metadata
            </summary>
            <p className="my-2 break-all text-[11px]">
              {proposal.id} · revision {proposal.revision} · {proposal.branch} @{" "}
              {shortOid(proposal.baseHead)} ·{" "}
              {proposal.createdBy?.subject || "unverified creator"}
            </p>
            <ul
              className="mt-2 space-y-1"
              aria-label="Trailers"
              data-testid="proposal-trailers"
            >
              {(proposal.message?.trailers ?? []).map((trailer, index) => {
                const meta = resolutionMeta(trailer.resolution);
                return (
                  <li
                    key={`${trailer.key}-${index}`}
                    className={`max-w-full truncate rounded-md border px-2 py-0.5 font-mono text-[10px] ${meta.className}`}
                    title={`${trailer.key}: ${trailer.value} — ${meta.label}${trailer.detail ? `: ${trailer.detail}` : ""}`}
                  >
                    {trailer.key}: {trailer.value}
                  </li>
                );
              })}
            </ul>
          </details>
          {proposal.issues.length > 0 && (
            <ul className="space-y-0.5 text-[11px]">
              {proposal.issues.map((issue, index) => (
                <li
                  key={`${issue.code}-${index}`}
                  className={
                    issue.severity === IssueSeverity.ERROR
                      ? "text-red-300"
                      : "text-amber-300"
                  }
                >
                  {issue.key}: {issue.message}
                </li>
              ))}
            </ul>
          )}
        </div>

        {(proposal.folds.length > 0 ||
          relatedProposals.length > 0 ||
          proposal.supersededBy) && (
          <details className="text-xs text-slate-400">
            <summary className="cursor-pointer">
              Sources &amp; related proposals (
              {proposal.folds.length + relatedProposals.length})
            </summary>
            <ul className="mt-2 space-y-2">
              {proposal.folds.map((source) => (
                <li key={source.id}>
                  <button
                    className="text-left text-cyan-300 hover:underline"
                    onClick={() =>
                      onOpenRelated?.(source.id, source.revision || undefined)
                    }
                  >
                    {source.epoch || "Source"} · {source.subject} · r
                    {source.revision || "legacy"}
                  </button>
                  {source.evidence?.gateReceipts.length ? (
                    <p className="mt-1 text-slate-500">
                      {source.evidence.gateReceipts.length} source receipts
                    </p>
                  ) : null}
                </li>
              ))}
              {relatedProposals
                .filter((p) => p.id !== proposal.id)
                .map((p) => (
                  <li key={p.id}>
                    <button
                      className="text-left text-cyan-300 hover:underline"
                      onClick={() => onOpenRelated?.(p.id)}
                    >
                      {p.message?.subject}
                    </button>
                  </li>
                ))}
              {proposal.supersededBy && (
                <li>
                  <button
                    className="text-cyan-300 hover:underline"
                    onClick={() => onOpenRelated?.(proposal.supersededBy)}
                  >
                    View replacement proposal
                  </button>
                </li>
              )}
            </ul>
          </details>
        )}
        <details className="text-xs text-slate-400">
          <summary className="cursor-pointer">
            Activity &amp; exclusions
          </summary>
          <ul className="mt-2 space-y-2">
            {proposal.events.map((event, index) => (
              <li key={index}>
                {event.action.replace(/_/g, " ")} · r{event.revision}
                <p className="break-words text-slate-500">{event.detail}</p>
              </li>
            ))}
          </ul>
          {proposal.excluded.length > 0 && (
            <details className="mt-3">
              <summary>Left out ({proposal.excluded.length})</summary>
              <ul>
                {proposal.excluded.map((item) => (
                  <li className="mt-1 break-all" key={item.path}>
                    {item.path} — {item.reason.replace(/_/g, " ")}
                  </li>
                ))}
              </ul>
            </details>
          )}
        </details>
        {result && <ApplyResult result={result} />}

        {actionError && (
          <p
            className="rounded-md border border-red-800/50 bg-red-950/30 px-3 py-2 text-xs text-red-300"
            role="alert"
            data-testid="proposal-error"
          >
            {actionError}
          </p>
        )}

        {preview && (
          <div
            className="space-y-2 rounded-xl border border-blue-900/70 bg-blue-950/20 p-3 text-xs text-blue-100"
            data-testid="proposal-confirm"
          >
            <p className="flex items-center gap-2 font-semibold">
              <ShieldCheck className="h-4 w-4 text-blue-300" /> Confirm this
              exact commit
            </p>
            <p className="text-blue-200/80">
              Commit the saved proposal version of exactly{" "}
              {proposal.files.length} file
              {proposal.files.length === 1 ? "" : "s"} on{" "}
              {preview.branch || "detached HEAD"} at{" "}
              {shortOid(preview.expectedRevision)} with the message above. You
              are the author. Later working edits stay uncommitted. The approval
              is single-use and expires shortly.
            </p>
            <p className="break-all text-[10px] text-blue-300/60">
              Subject digest: {preview.subjectDigest}
            </p>
            <label className="flex items-center gap-2 text-amber-200">
              <input
                type="checkbox"
                checked={skipPrecommit}
                onChange={(event) => setSkipPrecommit(event.target.checked)}
                disabled={apply.isPending}
              />
              Skip pre-commit checks for this commit
            </label>
            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <button
                type="button"
                className="rounded-lg border border-slate-700 min-h-11 px-3 py-2 text-slate-300 hover:bg-slate-800"
                onClick={() => setPreview(null)}
                disabled={apply.isPending}
              >
                Cancel
              </button>
              <GuidedCommitButton
                className="inline-flex items-center justify-center gap-1.5 rounded-lg bg-blue-600 min-h-11 px-3 py-2 font-semibold text-white hover:bg-blue-500 disabled:opacity-60"
                onClick={() => void applyWith(preview)}
                blocked={apply.isPending || Boolean(approveBlocker)}
                reason={approveBlocker || "Commit is running"}
                onExplain={() => setGuide("blocked")}
                testId="proposal-authorize"
              >
                {apply.isPending ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <CheckCircle2 className="h-3.5 w-3.5" />
                )}
                {apply.isPending ? "Committing…" : "Authorize & commit"}
              </GuidedCommitButton>
            </div>
          </div>
        )}
      </div>

      {isOpen && (
        <footer className="space-y-2 border-t border-slate-800 px-3 py-2">
          {confirmingWithdraw ? (
            <div className="flex flex-wrap items-center gap-2 text-xs text-slate-300">
              <p>
                Removes this proposal from the inbox. Your code and staged
                changes stay untouched.
              </p>
              <input
                value={closeReason}
                onChange={(event) => setCloseReason(event.target.value)}
                placeholder="Reason (optional)"
                aria-label="Reason for closing proposal"
                className="w-full rounded border border-slate-700 bg-slate-900 px-3 py-2"
              />
              {relatedProposals.some((p) => p.id !== proposal.id) && (
                <select
                  aria-label="Replacement proposal"
                  value={replacementId}
                  onChange={(event) => setReplacementId(event.target.value)}
                  className="w-full rounded border border-slate-700 bg-slate-900 px-3 py-2"
                >
                  <option value="">No replacement</option>
                  {relatedProposals
                    .filter(
                      (p) =>
                        p.id !== proposal.id &&
                        (p.state === ProposalState.OPEN ||
                          p.state === ProposalState.COMMITTED),
                    )
                    .map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.message?.subject || p.id}
                      </option>
                    ))}
                </select>
              )}
              {replacementId && (
                <div className="space-y-2">
                  <p>
                    This links the replacement; it does not combine code. Shared
                    paths may contain different versions.
                  </p>
                  {omittedByReplacement.length > 0 ? (
                    <div>
                      <p>
                        {omittedByReplacement.length}{" "}
                        {omittedByReplacement.length === 1
                          ? "path is"
                          : "paths are"}{" "}
                        absent from the selected replacement. Their code stays
                        in your workspace:
                      </p>
                      <ul className="max-h-32 overflow-y-auto break-all text-xs">
                        {omittedByReplacement.map((file) => (
                          <li key={file.path}>{file.path}</li>
                        ))}
                      </ul>
                    </div>
                  ) : (
                    <p>The replacement includes every path in this proposal.</p>
                  )}
                </div>
              )}

              <button
                type="button"
                className="rounded-md bg-red-700 px-2.5 py-1 font-medium text-white hover:bg-red-600 disabled:opacity-60"
                onClick={() =>
                  runAction(async () => {
                    await withdraw.mutateAsync({
                      id: proposal.id,
                      reason: closeReason,
                      expectedRevision: proposal.revision,
                      replacementId,
                    });
                    setConfirmingWithdraw(false);
                    onBack();
                  })
                }
                disabled={busy}
                data-testid="proposal-withdraw-confirm"
              >
                Close proposal
              </button>
              <button
                type="button"
                className="rounded-md px-2.5 py-1 text-slate-400 hover:bg-slate-800"
                onClick={() => setConfirmingWithdraw(false)}
              >
                Keep
              </button>
            </div>
          ) : (
            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-between">
              <button
                type="button"
                className="rounded-full border border-slate-700 min-h-11 px-4 py-2 text-xs text-slate-300 hover:bg-slate-800 disabled:opacity-60"
                onClick={() => setConfirmingWithdraw(true)}
                disabled={busy}
                data-testid="proposal-withdraw"
              >
                Close proposal
              </button>
              <GuidedCommitButton
                className="inline-flex items-center justify-center gap-2 rounded-full bg-blue-600 min-h-11 px-4 py-2 text-xs font-semibold text-white shadow-lg shadow-blue-950/40 hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-50"
                onClick={() => void handleApprove()}
                blocked={busy || Boolean(approveBlocker) || Boolean(preview)}
                onExplain={() => setGuide("blocked")}
                reason={
                  busy
                    ? "An action is running"
                    : approveBlocker || "Confirm the open preview"
                }
                testId="proposal-approve"
              >
                {prepare.isPending || (apply.isPending && !preview) ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <CheckCircle2 className="h-4 w-4" />
                )}
                Create commit
              </GuidedCommitButton>
            </div>
          )}
          {approveBlocker && !confirmingWithdraw && (
            <p
              className="text-right text-[11px] text-slate-500"
              data-testid="proposal-approve-blocker"
            >
              {approveBlocker}
            </p>
          )}
        </footer>
      )}
      {guide && (
        <ProposalReviewGuidance
          guide={guide}
          error={actionError ?? undefined}
          blocker={blocker}
          busy={busy}
          captured={proposal.captured}
          open={isOpen}
          priorPaths={uncertainPaths}
          stagedPaths={[...stagedVersionsDiffer, ...foreignStaged]}
          onClose={() => setGuide(null)}
          onUpdateGuide={() => setGuide("update")}
          onUpdate={() =>
            void runAction(async () => {
              await refresh.mutateAsync({
                id: proposal.id,
                expectedRevision: proposal.revision,
              });
              setGuide(null);
            })
          }
          onSave={subject.trim() ? saveMessage : undefined}
          onAcknowledge={() => setAcknowledgePriorWork(true)}
          onCompare={onCompareCurrent}
          onReturn={onReturnToProposal}
          onWorkingChanges={onOpenWorkingChanges}
          onRetry={onRetryReview}
        />
      )}
    </section>
  );
}

function ApplyResult({ result }: { result: ApplyProposalResponse }) {
  if (result.success) {
    return (
      <div
        className="rounded-md border border-emerald-800/60 bg-emerald-950/30 px-3 py-2 text-xs text-emerald-200"
        role="status"
        data-testid="proposal-committed"
      >
        <p className="font-medium">Committed {shortOid(result.commitOid)}</p>
        {result.verificationNotes.map((note) => (
          <p key={note} className="mt-1 text-amber-200">
            {note}
          </p>
        ))}
      </div>
    );
  }
  const refusal = result.refusal;
  const precommit = result.precommit;
  return (
    <div
      className="space-y-1 rounded-md border border-red-800/50 bg-red-950/30 px-3 py-2 text-xs text-red-300"
      role="alert"
      data-testid="proposal-refused"
    >
      {result.commitOid && (
        <p className="font-medium text-amber-200">
          Inspect commit {shortOid(result.commitOid)} and destination history
          before retrying.
        </p>
      )}
      <p className="font-medium">
        {refusal
          ? `Not committed: ${refusal.code.replace(/_/g, " ")}`
          : "Commit failed"}
      </p>
      {refusal?.detail && <p>{refusal.detail}</p>}
      {refusal && refusal.paths.length > 0 && (
        <ul className="font-mono text-[11px]">
          {refusal.paths.map((path) => (
            <li key={path} className="break-all">
              {path}
            </li>
          ))}
        </ul>
      )}
      {result.error && <p className="break-words">{result.error}</p>}
      {precommit && precommit.status !== "passed" && (
        <pre className="max-h-32 overflow-auto whitespace-pre-wrap rounded bg-slate-950/50 p-2 text-[11px] text-red-200">
          {[precommit.summary, precommit.stderr, precommit.stdout]
            .filter(Boolean)
            .join("\n")}
        </pre>
      )}
    </div>
  );
}
