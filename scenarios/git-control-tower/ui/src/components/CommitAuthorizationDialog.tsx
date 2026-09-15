import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { AlertTriangle, CheckCircle2, Clock3, FileCode2, GitBranch, ShieldCheck, Sparkles } from "lucide-react";
import type { MutationPreviewResponse } from "../lib/api";

export interface PendingCommitAuthorization {
  message: string;
  options: { conventional: boolean; amend: boolean; skipHooks?: boolean; authorName?: string; authorEmail?: string };
  preview: MutationPreviewResponse;
}

interface Props {
  pending: PendingCommitAuthorization;
  isAuthorizing: boolean;
  skipPrecommit: boolean;
  dontAskAgain: boolean;
  showPrecommitOption: boolean;
  onSkipPrecommitChange: (skip: boolean) => void;
  onDontAskAgainChange: (dontAskAgain: boolean) => void;
  onConfirm: () => void;
  onClose: () => void;
}

export function CommitAuthorizationDialog({
  pending,
  isAuthorizing,
  skipPrecommit,
  dontAskAgain,
  showPrecommitOption,
  onSkipPrecommitChange,
  onDontAskAgainChange,
  onConfirm,
  onClose,
}: Props) {
  return (
    <ResponsiveDialog
      open
      onClose={onClose}
      title={
        <span className="flex min-w-0 items-center gap-3">
          <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-blue-400/20 bg-blue-500/10 text-blue-300" aria-hidden="true">
            <ShieldCheck className="h-5 w-5" />
          </span>
          <span className="min-w-0">
            <span className="block text-[10px] font-semibold uppercase tracking-[0.18em] text-blue-300/80">Human approval</span>
            <span className="mt-1 block text-lg font-semibold tracking-tight text-slate-50">Confirm exact repository mutation</span>
          </span>
        </span>
      }
      ariaLabel="Confirm exact repository mutation"
      closeLabel="Close commit confirmation"
      size="lg"
      contentPadding="none"
      avoidKeyboard
      testId="commit-authorization-dialog"
      panelClassName="!border-slate-700/80 !bg-slate-900 !text-slate-200"
      footer={
        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button
            type="button"
            className="rounded-xl border border-slate-700 px-4 py-2.5 text-xs font-medium text-slate-300 transition hover:bg-slate-800 disabled:opacity-60"
            onClick={onClose}
            disabled={isAuthorizing}
          >
            Cancel
          </button>
          <button
            type="button"
            className="inline-flex items-center justify-center gap-2 rounded-xl bg-blue-600 px-4 py-2.5 text-xs font-semibold text-white shadow-lg shadow-blue-950/40 transition hover:bg-blue-500 disabled:opacity-60"
            onClick={onConfirm}
            disabled={isAuthorizing}
            data-testid="confirm-commit-authorization"
          >
            <CheckCircle2 className="h-4 w-4" />
            {isAuthorizing ? "Authorizing…" : skipPrecommit ? "Authorize & commit" : "Authorize and commit"}
          </button>
        </div>
      }
    >
      <div className="space-y-4 p-5">
        <p className="text-xs leading-relaxed text-slate-300/80">
          Review the snapshot below before authorizing this commit. The approval is single-use and expires shortly.
        </p>

        <div className="grid gap-2 sm:grid-cols-2">
          <div className="rounded-xl border border-slate-800 bg-slate-950/40 p-3">
            <div className="flex items-center gap-2 text-[10px] font-semibold uppercase tracking-wider text-slate-500">
              <FileCode2 className="h-3.5 w-3.5" /> Repository
            </div>
            <p className="mt-1 truncate text-xs font-medium text-slate-100" title={pending.preview.repositoryPath}>
              {pending.preview.repositoryPath}
            </p>
          </div>
          <div className="rounded-xl border border-slate-800 bg-slate-950/40 p-3">
            <div className="flex items-center gap-2 text-[10px] font-semibold uppercase tracking-wider text-slate-500">
              <GitBranch className="h-3.5 w-3.5" /> Branch
            </div>
            <p className="mt-1 truncate text-xs font-medium text-slate-100">
              {pending.preview.branch || "detached HEAD"}
            </p>
          </div>
        </div>

        <dl className="divide-y divide-slate-800/80 rounded-xl border border-slate-800 bg-slate-950/25 text-xs">
          <div className="flex items-start justify-between gap-4 px-3 py-2.5">
            <dt className="flex shrink-0 items-center gap-2 text-slate-500"><Clock3 className="h-3.5 w-3.5" /> Revision</dt>
            <dd className="break-all text-right font-mono text-[11px] text-slate-300">{pending.preview.expectedRevision}</dd>
          </div>
          <div className="flex items-start justify-between gap-4 px-3 py-2.5">
            <dt className="flex shrink-0 items-center gap-2 text-slate-500"><Sparkles className="h-3.5 w-3.5" /> Message</dt>
            <dd className="max-w-[68%] text-right font-medium text-slate-100">{pending.message || "(amend previous message)"}</dd>
          </div>
        </dl>

        <div className="rounded-xl border border-slate-800 bg-slate-950/50 p-3">
          <div className="flex items-center justify-between gap-3">
            <p className="text-xs font-semibold text-slate-100">Change snapshot</p>
            <span className="rounded-full bg-blue-500/10 px-2 py-1 text-[10px] font-medium text-blue-300">
              {pending.preview.fileCount} staged file{pending.preview.fileCount === 1 ? "" : "s"}
            </span>
          </div>
          <ul className="mt-2 max-h-28 space-y-1 overflow-y-auto rounded-lg border border-slate-800/80 bg-slate-950 p-2 font-mono text-[11px] text-slate-400">
            {pending.preview.stagedFiles.map((file) => <li key={file} className="truncate">{file}</li>)}
          </ul>
          <p className="mt-2 break-all text-[10px] text-slate-600">Subject digest: {pending.preview.subjectDigest}</p>
        </div>

        <div className="space-y-2">
          {showPrecommitOption && (
            <label className="flex cursor-pointer items-start gap-3 rounded-xl border border-amber-800/50 bg-amber-950/20 p-3 transition hover:border-amber-700/70">
              <input
                type="checkbox"
                checked={skipPrecommit}
                onChange={(event) => onSkipPrecommitChange(event.target.checked)}
                disabled={isAuthorizing}
                className="mt-0.5 h-4 w-4 rounded border-amber-700 bg-slate-900 text-amber-500 focus:ring-amber-500 focus:ring-offset-0"
                data-testid="skip-precommit-confirmation-checkbox"
              />
              <span>
                <span className="block text-xs font-medium text-amber-100">Skip pre-commit checks for this commit</span>
                <span className="mt-0.5 block text-[11px] leading-relaxed text-amber-200/65">Use only when you understand why the configured checks should be bypassed.</span>
              </span>
            </label>
          )}
          <label className="flex cursor-pointer items-start gap-3 rounded-xl border border-slate-800 bg-slate-950/30 p-3 transition hover:border-slate-700">
            <input
              type="checkbox"
              checked={dontAskAgain}
              onChange={(event) => onDontAskAgainChange(event.target.checked)}
              disabled={isAuthorizing}
              className="mt-0.5 h-4 w-4 rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500 focus:ring-offset-0"
              data-testid="dont-ask-commit-confirmation-checkbox"
            />
            <span>
              <span className="block text-xs font-medium text-slate-200">Don&apos;t ask again on this device</span>
              <span className="mt-0.5 block text-[11px] leading-relaxed text-slate-500">The server will still require a fresh, exact approval for every commit.</span>
            </span>
          </label>
        </div>

        <div className="flex items-start gap-2 rounded-lg border border-blue-900/60 bg-blue-950/20 px-3 py-2.5 text-[11px] leading-relaxed text-blue-200/75">
          <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-blue-300" />
          <span>If the staged files or repository revision changes, this approval will be rejected and you will be asked to review the new snapshot.</span>
        </div>
      </div>
    </ResponsiveDialog>
  );
}
