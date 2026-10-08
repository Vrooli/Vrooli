import { useEffect, useState } from "react";
import { AlertTriangle, ArrowLeft, CheckCircle2, Loader2, RefreshCw, Save, ShieldCheck, Undo2 } from "lucide-react";
import type { MutationPreview } from "@vrooli/proto-types/git-control-tower/v1/human_control/human_control_pb";
import {
  FreshnessState,
  IssueSeverity,
  ProposalState,
  type ApplyProposalResponse,
  type Proposal,
} from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import { Badge } from "../../components/ui/badge";
import {
  useApplyProposal,
  useEditProposal,
  usePrepareProposalApproval,
  useRefreshProposal,
  useWithdrawProposal,
} from "../../lib/hooks-proposals";
import { effortLabel, flagMeta, freshnessMeta, kindMarker, resolutionMeta, shortOid } from "./model";
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
  onOpenDiff: (path: string) => void;
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
  onOpenDiff,
  onCommitted,
}: ProposalDetailProps) {
  // Unsaved edits live in a draft that survives refetches and new revisions
  // (GCT-053): a revision arriving mid-edit never discards operator text.
  const [draft, setDraft] = useSessionState<{ subject: string; body: string; revision: number } | null>(`gct.proposals.draft.${proposal.id}`, null);
  const [preview, setPreview] = useState<MutationPreview | null>(null);
  const [skipPrecommit, setSkipPrecommit] = useState(false);
  const [confirmingWithdraw, setConfirmingWithdraw] = useState(false);
  const [result, setResult] = useState<ApplyProposalResponse | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const edit = useEditProposal(repoId);
  const refresh = useRefreshProposal(repoId);
  const withdraw = useWithdrawProposal(repoId);
  const prepare = usePrepareProposalApproval();
  const apply = useApplyProposal(repoId);

  // A new revision invalidates any open preview: an approval always names the
  // revision on screen.
  useEffect(() => {
    setPreview(null);
  }, [proposal.id, proposal.revision]);

  const serverSubject = proposal.message?.subject ?? "";
  const serverBody = proposal.message?.body ?? "";
  const subject = draft?.subject ?? serverSubject;
  const body = draft?.body ?? serverBody;
  const updateDraft = (next: { subject?: string; body?: string }) =>
    setDraft({ subject: next.subject ?? subject, body: next.body ?? body, revision: draft?.revision ?? proposal.revision });

  const isOpen = proposal.state === ProposalState.OPEN;
  const freshness = proposal.freshness;
  const freshnessInfo = freshnessMeta(freshness?.state);
  const isFresh = freshness?.state === FreshnessState.FRESH;
  const foreignStaged = freshness?.foreignStagedPaths ?? [];
  const dirty = draft !== null && (subject.trim() !== serverSubject || body.trim() !== serverBody.trim());
  const draftOutdated = dirty && draft !== null && draft.revision !== proposal.revision;
  const busy = edit.isPending || refresh.isPending || withdraw.isPending || prepare.isPending || apply.isPending;
  const approveBlocker = !canApprove
    ? approveDisabledReason || "Only the operator can approve proposals"
    : !isOpen
      ? "This proposal is closed"
      : dirty
        ? "Save your edits first"
        : !isFresh
          ? "Refresh the proposal first"
          : foreignStaged.length > 0
            ? "Unstage the other staged files first"
            : undefined;

  const runAction = async (action: () => Promise<unknown>) => {
    setActionError(null);
    try {
      await action();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : String(error));
    }
  };

  const applyWith = (approvedPreview: MutationPreview) =>
    runAction(async () => {
      const response = await apply.mutateAsync({ proposal, preview: approvedPreview, skipPrecommit });
      setPreview(null);
      setResult(response);
      if (response.success) onCommitted?.(response.commitOid);
    });

  const handleApprove = () =>
    runAction(async () => {
      setResult(null);
      const prepared = await prepare.mutateAsync(proposal);
      if (skipConfirmation) {
        await applyWith(prepared);
        return;
      }
      setPreview(prepared);
    });

  return (
    <section className="flex h-full min-h-0 flex-col" aria-label="Commit proposal" data-testid="proposal-detail">
      <header className="flex items-center gap-2 border-b border-slate-800 px-3 py-2">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1 rounded px-1.5 py-1 text-xs text-slate-400 hover:bg-slate-800 hover:text-slate-200"
          data-testid="proposal-back"
        >
          <ArrowLeft className="h-3.5 w-3.5" /> Proposals
        </button>
        <span className="ml-auto font-mono text-[11px] text-slate-500">{proposal.id} · r{proposal.revision}</span>
        <Badge variant={freshnessInfo.tone} title={freshnessInfo.hint}>{freshnessInfo.label}</Badge>
      </header>

      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-3 py-3">
        <div className="flex flex-wrap items-center gap-1.5 text-[11px]">
          {proposal.work?.epoch && <span className="rounded border border-blue-900/70 bg-blue-950/40 px-1.5 py-0.5 font-mono text-blue-200">{proposal.work.epoch}</span>}
          {proposal.work?.effortRef && <span className="rounded border border-slate-700 bg-slate-950 px-1.5 py-0.5 text-slate-300" title={proposal.work.effortRef}>{effortLabel(proposal.work.effortRef)}</span>}
          {proposal.work && proposal.work.runIds.length > 0 && <span className="text-slate-500">{proposal.work.runIds.length} run{proposal.work.runIds.length === 1 ? "" : "s"}</span>}
          <span className="text-slate-500">by {proposal.createdBy?.subject || "unknown"}{proposal.createdBy?.kind ? ` (${proposal.createdBy.kind})` : ""}</span>
          {proposal.branch && <span className="text-slate-500">on {proposal.branch} @ {shortOid(proposal.baseHead)}</span>}
        </div>

        {isOpen && !isFresh && (
          <div className="space-y-2 rounded-md border border-amber-800/60 bg-amber-950/20 p-3 text-xs text-amber-200" role="status" data-testid="proposal-stale">
            <p className="flex items-center gap-2 font-medium"><AlertTriangle className="h-3.5 w-3.5" /> {freshnessInfo.hint}</p>
            {(freshness?.driftedPaths ?? []).length > 0 && <p className="font-mono text-[11px] break-all">Changed: {freshness?.driftedPaths.join(", ")}</p>}
            {(freshness?.baseChangedPaths ?? []).length > 0 && <p className="font-mono text-[11px] break-all">Committed since base: {freshness?.baseChangedPaths.join(", ")}</p>}
            {freshness?.detail && <p>{freshness.detail}</p>}
            <button
              type="button"
              className="inline-flex items-center gap-1.5 rounded-md border border-amber-700 px-2 py-1 font-medium hover:bg-amber-900/40 disabled:opacity-60"
              onClick={() => runAction(() => refresh.mutateAsync({ id: proposal.id, expectedRevision: proposal.revision }))}
              disabled={busy}
              data-testid="proposal-refresh"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${refresh.isPending ? "animate-spin" : ""}`} /> Refresh to current content
            </button>
          </div>
        )}
        {isOpen && foreignStaged.length > 0 && (
          <p className="rounded-md border border-amber-800/60 bg-amber-950/20 px-3 py-2 text-xs text-amber-200" data-testid="proposal-foreign-staged">
            Other files are staged ({foreignStaged.join(", ")}). Unstage them so the commit holds only this proposal.
          </p>
        )}

        <div>
          <p className="mb-1.5 text-[11px] font-medium uppercase tracking-wide text-slate-500">Files ({proposal.files.length})</p>
          <ul className="divide-y divide-slate-800/80 rounded-md border border-slate-800" data-testid="proposal-files">
            {proposal.files.map((file) => {
              const marker = kindMarker(file.kind);
              return (
                <li key={file.path} className="flex flex-wrap items-center gap-x-2 gap-y-1 px-2 py-1.5">
                  <span className={`w-3 font-mono text-xs ${marker.className}`} title={marker.label}>{marker.letter}</span>
                  <button
                    type="button"
                    className="min-w-0 flex-1 truncate text-left font-mono text-xs text-slate-200 hover:text-blue-300 hover:underline"
                    onClick={() => onOpenDiff(file.path)}
                    title={`Show diff for ${file.path}`}
                  >
                    {file.path}
                  </button>
                  {file.drifted && <Badge variant="warning" className="px-1.5 text-[10px]">changed</Badge>}
                  {file.flags.map((flag) => {
                    const meta = flagMeta(flag.code);
                    return (
                      <Badge key={`${flag.code}-${flag.detail}`} variant={meta.tone} className="px-1.5 text-[10px]" title={flag.detail ? `${meta.hint} (${flag.detail})` : meta.hint}>
                        {meta.label}
                      </Badge>
                    );
                  })}
                </li>
              );
            })}
          </ul>
          {proposal.excluded.length > 0 && (
            <details className="mt-2 text-xs text-slate-400">
              <summary className="cursor-pointer">Left out ({proposal.excluded.length})</summary>
              <ul className="mt-1 space-y-0.5">
                {proposal.excluded.map((item) => (
                  <li key={item.path} className="font-mono text-[11px] break-all">{item.path} <span className="text-slate-500">— {item.reason.replace(/_/g, " ")}</span></li>
                ))}
              </ul>
            </details>
          )}
        </div>

        <div className="space-y-2">
          <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">Message</p>
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
            <p className="text-[11px] text-amber-300" data-testid="proposal-draft-outdated">
              The proposal changed to r{proposal.revision} while you were editing. Your text is kept; saving applies it to r{proposal.revision}.
            </p>
          )}
          {dirty && (
            <div className="flex gap-2">
              <button
                type="button"
                className="inline-flex items-center gap-1.5 rounded-md border border-slate-600 px-2.5 py-1 text-xs text-slate-200 hover:bg-slate-800 disabled:opacity-60"
                onClick={() => runAction(async () => { await edit.mutateAsync({ id: proposal.id, expectedRevision: proposal.revision, subject: subject.trim(), body }); setDraft(null); })}
                disabled={busy || subject.trim() === ""}
                data-testid="proposal-save"
              >
                {edit.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Save className="h-3.5 w-3.5" />} Save edits
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
          <ul className="flex flex-wrap gap-1.5" aria-label="Trailers" data-testid="proposal-trailers">
            {(proposal.message?.trailers ?? []).map((trailer, index) => {
              const meta = resolutionMeta(trailer.resolution);
              return (
                <li key={`${trailer.key}-${index}`} className={`max-w-full truncate rounded-md border px-2 py-0.5 font-mono text-[10px] ${meta.className}`} title={`${trailer.key}: ${trailer.value} — ${meta.label}${trailer.detail ? `: ${trailer.detail}` : ""}`}>
                  {trailer.key}: {trailer.value}
                </li>
              );
            })}
          </ul>
          {proposal.issues.length > 0 && (
            <ul className="space-y-0.5 text-[11px]">
              {proposal.issues.map((issue, index) => (
                <li key={`${issue.code}-${index}`} className={issue.severity === IssueSeverity.ERROR ? "text-red-300" : "text-amber-300"}>
                  {issue.key}: {issue.message}
                </li>
              ))}
            </ul>
          )}
        </div>

        {result && <ApplyResult result={result} />}
        {actionError && <p className="rounded-md border border-red-800/50 bg-red-950/30 px-3 py-2 text-xs text-red-300" role="alert" data-testid="proposal-error">{actionError}</p>}

        {preview && (
          <div className="space-y-2 rounded-xl border border-blue-900/70 bg-blue-950/20 p-3 text-xs text-blue-100" data-testid="proposal-confirm">
            <p className="flex items-center gap-2 font-semibold"><ShieldCheck className="h-4 w-4 text-blue-300" /> Confirm this exact commit</p>
            <p className="text-blue-200/80">
              Stage exactly {proposal.files.length} file{proposal.files.length === 1 ? "" : "s"} and commit them on {preview.branch || "detached HEAD"} at {shortOid(preview.expectedRevision)} with the message above. You are the author. The approval is single-use and expires shortly.
            </p>
            <p className="break-all text-[10px] text-blue-300/60">Subject digest: {preview.subjectDigest}</p>
            <label className="flex items-center gap-2 text-amber-200">
              <input type="checkbox" checked={skipPrecommit} onChange={(event) => setSkipPrecommit(event.target.checked)} disabled={apply.isPending} />
              Skip pre-commit checks for this commit
            </label>
            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <button type="button" className="rounded-lg border border-slate-700 px-3 py-1.5 text-slate-300 hover:bg-slate-800" onClick={() => setPreview(null)} disabled={apply.isPending}>
                Cancel
              </button>
              <button
                type="button"
                className="inline-flex items-center justify-center gap-1.5 rounded-lg bg-blue-600 px-3 py-1.5 font-semibold text-white hover:bg-blue-500 disabled:opacity-60"
                onClick={() => void applyWith(preview)}
                disabled={apply.isPending}
                data-testid="proposal-authorize"
              >
                {apply.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <CheckCircle2 className="h-3.5 w-3.5" />}
                {apply.isPending ? "Committing…" : "Authorize & commit"}
              </button>
            </div>
          </div>
        )}
      </div>

      {isOpen && (
        <footer className="space-y-2 border-t border-slate-800 px-3 py-2">
          {confirmingWithdraw ? (
            <div className="flex flex-wrap items-center gap-2 text-xs text-slate-300">
              <span>Withdraw this proposal? Its history is kept.</span>
              <button
                type="button"
                className="rounded-md bg-red-700 px-2.5 py-1 font-medium text-white hover:bg-red-600 disabled:opacity-60"
                onClick={() => runAction(async () => { await withdraw.mutateAsync({ id: proposal.id, reason: "withdrawn by operator" }); setConfirmingWithdraw(false); onBack(); })}
                disabled={busy}
                data-testid="proposal-withdraw-confirm"
              >
                Withdraw
              </button>
              <button type="button" className="rounded-md px-2.5 py-1 text-slate-400 hover:bg-slate-800" onClick={() => setConfirmingWithdraw(false)}>Keep</button>
            </div>
          ) : (
            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-between">
              <button
                type="button"
                className="rounded-full border border-slate-700 px-4 py-2 text-xs text-slate-300 hover:bg-slate-800 disabled:opacity-60"
                onClick={() => setConfirmingWithdraw(true)}
                disabled={busy}
                data-testid="proposal-withdraw"
              >
                Withdraw
              </button>
              <button
                type="button"
                className="inline-flex items-center justify-center gap-2 rounded-full bg-blue-600 px-4 py-2 text-xs font-semibold text-white shadow-lg shadow-blue-950/40 hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-50"
                onClick={() => void handleApprove()}
                disabled={busy || Boolean(approveBlocker) || Boolean(preview)}
                title={approveBlocker}
                data-testid="proposal-approve"
              >
                {prepare.isPending || (apply.isPending && !preview) ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle2 className="h-4 w-4" />}
                Approve &amp; commit
              </button>
            </div>
          )}
          {approveBlocker && !confirmingWithdraw && <p className="text-right text-[11px] text-slate-500" data-testid="proposal-approve-blocker">{approveBlocker}</p>}
        </footer>
      )}
    </section>
  );
}

function ApplyResult({ result }: { result: ApplyProposalResponse }) {
  if (result.success) {
    return (
      <div className="rounded-md border border-emerald-800/60 bg-emerald-950/30 px-3 py-2 text-xs text-emerald-200" role="status" data-testid="proposal-committed">
        <p className="font-medium">Committed {shortOid(result.commitOid)}</p>
        {!result.commitVerified && result.verificationNotes.map((note) => <p key={note} className="mt-1 text-amber-200">{note}</p>)}
      </div>
    );
  }
  const refusal = result.refusal;
  const precommit = result.precommit;
  return (
    <div className="space-y-1 rounded-md border border-red-800/50 bg-red-950/30 px-3 py-2 text-xs text-red-300" role="alert" data-testid="proposal-refused">
      <p className="font-medium">{refusal ? `Not committed: ${refusal.code.replace(/_/g, " ")}` : "Commit failed"}</p>
      {refusal?.detail && <p>{refusal.detail}</p>}
      {refusal && refusal.paths.length > 0 && (
        <ul className="font-mono text-[11px]">{refusal.paths.map((path) => <li key={path} className="break-all">{path}</li>)}</ul>
      )}
      {result.error && <p className="break-words">{result.error}</p>}
      {precommit && precommit.status !== "passed" && (
        <pre className="max-h-32 overflow-auto whitespace-pre-wrap rounded bg-slate-950/50 p-2 text-[11px] text-red-200">{[precommit.summary, precommit.stderr, precommit.stdout].filter(Boolean).join("\n")}</pre>
      )}
    </div>
  );
}
