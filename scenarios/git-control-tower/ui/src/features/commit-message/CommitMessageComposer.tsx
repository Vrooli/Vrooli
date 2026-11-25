import { useEffect, useId, useRef, useState } from "react";
import { Eye, FileText, RotateCw, Sparkles, X } from "lucide-react";
import { getDraftScope, parseMessage, serializeMessage, type MessageFields } from "./message";
import { useMessageSuggestions, isMessageJobActive } from "./useMessageSuggestions";
import { workflowExecutionLink } from "./workflowExecutionLink";
import { formatMicroUSD } from "./charge";
import { MessageSuggestionDialog } from "./MessageSuggestionDialog";
import { usePrecommitCandidateCheck } from "./usePrecommitCandidateCheck";
import type { QueuedStageSelection } from "./stageQueue";
import type { StagedMessageMode } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";

interface Props {
  value: string;
  onChange: (message: string) => void;
  repoId: string;
  stagedCount: number;
  statusVersion: number;
  updatingIndex: boolean;
  disabled: boolean;
  amend: boolean;
  stagedPaths?: readonly string[];
  onUnstage?: (paths: string[]) => Promise<void>;
  onQueueStagedExtras?: (paths: string[]) => Promise<boolean>;
  /** Changes only when the staged candidate changes, not on every poll. */
  candidateVersion?: string;
  queuedGroups?: QueuedStageSelection[];
  startingQueuedSuggestionIds?: string[];
  onStartQueuedSuggestion?: (selection: QueuedStageSelection, mode: StagedMessageMode) => void;
  /** Close the dialog and reveal the follow-up queue behind it. */
  onShowQueue?: () => void;
}

export function CommitMessageComposer({ value, onChange, repoId, stagedCount, statusVersion, candidateVersion, updatingIndex, disabled, amend, stagedPaths, onUnstage, onQueueStagedExtras, queuedGroups, startingQueuedSuggestionIds, onStartQueuedSuggestion, onShowQueue }: Props) {
  const id = useId();
  const [fields, setFields] = useState(() => parseMessage(value));
  const emitted = useRef(value);
  const prefetchTimer = useRef<ReturnType<typeof setTimeout>>();
  const suggestions = useMessageSuggestions(repoId, statusVersion, updatingIndex, stagedCount);
  const precommit = usePrecommitCandidateCheck(repoId, true, candidateVersion ?? String(statusVersion), updatingIndex, stagedCount > 0);
  useEffect(() => {
    if (value !== emitted.current) {
      emitted.current = value;
      setFields(parseMessage(value));
    }
  }, [value]);
  useEffect(() => () => {
    if (prefetchTimer.current) clearTimeout(prefetchTimer.current);
  }, [repoId, statusVersion, stagedCount, updatingIndex]);
  const prefetchOnIntent = () => {
    if (prefetchTimer.current) return;
    prefetchTimer.current = setTimeout(() => {
      prefetchTimer.current = undefined;
      suggestions.prefetch();
    }, 180);
  };
  const openSuggestions = () => {
    if (prefetchTimer.current) {
      clearTimeout(prefetchTimer.current);
      prefetchTimer.current = undefined;
    }
    if (hasWork) suggestions.setOpen(true);
    else void suggestions.collect(value);
  };
  const change = (next: MessageFields) => {
    setFields(next);
    emitted.current = serializeMessage(next);
    onChange(emitted.current);
  };
  const active = isMessageJobActive(suggestions.job?.status);
  const hasWork = suggestions.session.phase !== "idle" || Boolean(suggestions.job) || suggestions.reconnecting || Boolean(suggestions.error);
  const precommitResult = precommit.check?.result;
  const precommitFailed = ["failed", "timeout", "incomplete", "error"].includes(precommit.state);
  const precommitOutput = [precommitResult?.stderr, precommitResult?.stdout].filter(Boolean).join("\n").trim();
  const unknownJob = suggestions.job && !["queued", "dispatching", "running", "awaiting_input", "awaiting_provider", "cancelling", "succeeded", "failed", "cancelled", "stale"].includes(suggestions.job.status);
  const resultScope = suggestions.job?.result ? getDraftScope(suggestions.job.result) : undefined;
  const readyLabel = resultScope === "whole" ? "Whole-selection draft ready" : resultScope === "limited" ? "Limited draft ready" : resultScope === "unknown" ? "Draft ready · scope unavailable" : "Agent suggestion ready";
  const terminalJob = suggestions.job && ["failed", "cancelled", "stale", "awaiting_provider"].includes(suggestions.job.status) ? suggestions.job : undefined;
  const failureSummary = terminalJob?.error?.trim().replace(/\s+/g, " ");
  const elapsedSeconds = terminalJob ? Math.max(0, Math.floor((Date.parse(terminalJob.updatedAt) - Date.parse(terminalJob.createdAt)) / 1000)) : undefined;
  const label = suggestions.action === "cancel" ? "Requesting cancellation…"
    : suggestions.admissionState === "starting" ? "Starting drafting agent…"
    : suggestions.admissionState === "recovering" ? "Checking launch status…"
    : suggestions.admissionState === "unknown" ? "Could not confirm launch"
    : suggestions.staleReason ? "Review staged selection · draft retained"
    : unknownJob ? "Execution status unavailable"
    : suggestions.job?.status === "awaiting_provider" ? "Waiting for provider admission"
    : suggestions.error ? "Suggestions unavailable · message retained"
    : suggestions.reconnecting ? "Reconnecting to drafting job…"
    : suggestions.session.phase === "collecting" ? "Checking existing evidence…"
    : suggestions.job?.status === "dispatching" ? "Dispatching to Agent Manager…"
    : suggestions.job?.status === "awaiting_input" ? "Agent is waiting for input"
    : active && !suggestions.job?.executionId ? "Confirming agent launch"
    : active ? suggestions.job?.status === "cancelling" ? "Stopping drafting agent…" : "Drafting agent in progress"
    : suggestions.session.applied ? "Suggestion applied · editable"
    : suggestions.job?.status === "succeeded" ? suggestions.job.result ? readyLabel : "No completed suggestion"
    : suggestions.job?.status === "failed" ? "Drafting request failed"
    : suggestions.job?.status === "stale" ? "Saved execution is missing"
    : suggestions.job?.status === "cancelled" || suggestions.session.phase === "cancelled" ? "Suggestion work cancelled"
    : suggestions.session.collection?.suggestions.length ? "Reusable suggestion available" : "No reusable message found";
  return (
    <div className="space-y-3" data-testid="commit-message-composer">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium text-slate-300">Commit message</span>
        <button
          type="button"
          aria-label="Suggest commit title and description"
          disabled={stagedCount === 0 || updatingIndex || disabled}
          onPointerEnter={prefetchOnIntent}
          onFocus={prefetchOnIntent}
          onClick={openSuggestions}
          className="inline-flex min-h-11 items-center gap-1.5 rounded-lg border border-cyan-800/70 bg-cyan-950/30 px-3 text-xs text-cyan-200 hover:bg-cyan-950/60 disabled:opacity-50"
          data-testid="suggest-commit-message"
        >
          <Sparkles className="h-3.5 w-3.5" aria-hidden="true" /> Suggest
        </button>
      </div>
      {precommit.state !== "idle" && (
        <div role={precommit.state === "failed" || precommit.state === "incomplete" || precommit.state === "error" ? "alert" : "status"}
          className={`rounded-lg border px-3 py-2 text-xs ${precommit.state === "passed" ? "border-emerald-800 bg-emerald-950/30 text-emerald-200" : precommit.state === "failed" || precommit.state === "incomplete" || precommit.state === "error" ? "border-amber-800 bg-amber-950/30 text-amber-200" : "border-slate-700 bg-slate-900 text-slate-300"}`}
          data-testid="precommit-candidate-status">
          <div className="font-medium">
            {precommit.state === "loading" ? "Preparing staged checks…"
              : precommit.state === "disabled" ? "Staged checks are disabled"
              : precommit.state === "queued" ? "Commit checks queued · message drafting can continue"
              : precommit.state === "running" ? "Commit checks running · message drafting can continue"
              : precommit.state === "cancelling" ? "Cancelling commit checks…"
              : precommit.state === "cancelled" ? "Commit checks were cancelled"
              : precommit.state === "passed" ? "Checks passed for this staged candidate"
              : precommit.state === "failed" || precommit.state === "timeout" ? "Checks did not pass for this staged candidate"
              : precommit.state === "incomplete" ? "Check result is incomplete"
              : precommit.state === "error" ? "Could not start or reattach to staged checks"
              : "Staged checks unavailable"}
          </div>
          {precommit.check?.candidateDigest && <div className="mt-1 text-[11px] opacity-80">Candidate {precommit.check.candidateDigest.slice(0, 12)} · final commit checks this identity again</div>}
          {(precommit.error || precommitResult?.summary) && <div className="mt-1 break-words opacity-90">{precommit.error || precommitResult?.summary}</div>}
          {precommitFailed && !precommit.error && !precommitResult?.summary && (
            <div className="mt-1 opacity-90">The runner returned no failure summary. Retry checks; if it repeats, the pre-commit job did not report a result.</div>
          )}
          {precommitResult?.command && (
            <div className="mt-1 break-all text-[11px] opacity-80">Command: <code>{precommitResult.command}</code>{typeof precommitResult.exitCode === "number" ? ` · exit ${precommitResult.exitCode}` : ""}</div>
          )}
          {precommitOutput && (
            <details className="mt-1">
              <summary className="cursor-pointer text-[11px] opacity-80">Check output</summary>
              <pre className="mt-1 max-h-40 overflow-auto whitespace-pre-wrap break-words rounded bg-slate-950/50 p-2 font-mono text-[11px]">{precommitOutput.slice(-2000)}</pre>
            </details>
          )}
          {precommit.stale && <div className="mt-1">Staged files changed; checking the new candidate.</div>}
          {["queued", "running", "cancelling"].includes(precommit.state) && (
            <button type="button" onClick={() => void precommit.cancel()} disabled={precommit.state === "cancelling"}
              className="mt-2 inline-flex min-h-9 items-center rounded-md border border-slate-600 px-3 text-slate-200 hover:bg-slate-800 disabled:opacity-50"
              data-testid="precommit-candidate-cancel">
              {precommit.state === "cancelling" ? "Cancelling…" : "Cancel checks"}
            </button>
          )}
          {["failed", "timeout", "incomplete", "cancelled", "error"].includes(precommit.state) && (
            <button type="button" onClick={() => void precommit.retry()} disabled={precommit.state === "loading"}
              className="mt-2 inline-flex min-h-9 items-center rounded-md border border-amber-700 px-3 text-amber-100 hover:bg-amber-950/50 disabled:opacity-50"
              data-testid="precommit-candidate-retry">
              Retry checks
            </button>
          )}
        </div>
      )}
      {hasWork && (
        <div role="status" className={`flex flex-wrap items-center gap-2 rounded-lg border px-3 py-2 text-xs ${suggestions.staleReason ? "border-amber-800 bg-amber-950/30 text-amber-200" : "border-slate-700 bg-slate-900 text-slate-300"}`} data-testid="commit-suggestion-standing">
          <span className="min-w-0 flex-1">{label}</span>
          {active && suggestions.job?.error && <span className="w-full text-amber-200" role="alert">{suggestions.job.error}</span>}
          {active && suggestions.job?.error && <button type="button" disabled={suggestions.reconnecting} onClick={() => void suggestions.reconnect()} className="inline-flex min-h-9 items-center gap-1.5 px-2 text-cyan-200 disabled:cursor-wait disabled:opacity-60"><RotateCw className={`h-3.5 w-3.5 ${suggestions.reconnecting ? "animate-spin" : ""}`} aria-hidden="true" />{suggestions.reconnecting ? "Reconnecting…" : "Reconnect to job"}</button>}
          {terminalJob && <span className="w-full truncate text-amber-200" title={failureSummary || undefined}>{failureSummary || "Failure details are unavailable."}</span>}
          {terminalJob?.errorCode && <span className="w-full text-slate-400">Failure category: {terminalJob.errorCode}</span>}
          {terminalJob?.tokensKnown && <span className="w-full text-slate-400">Agent tokens used: {String(terminalJob.tokensUsed)}</span>}
          {terminalJob && <span className="w-full text-slate-400">{terminalJob.chargeKnown ? `Marginal charge: ${formatMicroUSD(terminalJob.chargeMicroUsd)} USD` : "Marginal charge unknown; token counts are not a charge estimate"}</span>}
          {terminalJob && terminalJob.executionStartedKnown === false && <span className="w-full text-slate-400">Agent start status unknown</span>}
          {terminalJob && terminalJob.executionStartedKnown && terminalJob.executionStarted === false && <span className="w-full text-slate-400">Agent did not start</span>}
          {terminalJob?.errorCode === "provider_admission_maintenance" && <span className="w-full text-slate-400">Retry is paused until Workspace Sandbox reports ready.</span>}
          {terminalJob?.phase && <span className="w-full text-slate-400">Last stage: {terminalJob.phase}{elapsedSeconds !== undefined && Number.isFinite(elapsedSeconds) ? ` · elapsed ${elapsedSeconds}s` : ""}</span>}
          {suggestions.job?.executionId && <a className="w-full text-cyan-200 underline" href={workflowExecutionLink(suggestions.job.executionId, window.location.hostname, window.location.protocol)} target="_blank" rel="noreferrer">Open workflow execution</a>}
          <button type="button" onClick={() => suggestions.setOpen(true)} className="inline-flex min-h-9 items-center gap-1.5 px-2 text-cyan-200"><Eye className="h-3.5 w-3.5" aria-hidden="true" />View</button>
          {active && <button type="button" onClick={() => void suggestions.cancelAgent()} disabled={suggestions.job?.status === "cancelling" || suggestions.action === "cancel"} className="inline-flex min-h-9 items-center gap-1.5 px-2 disabled:opacity-50"><X className="h-3.5 w-3.5" aria-hidden="true" />Cancel agent</button>}
        </div>
      )}
      <div>
        <div className="mb-1.5 flex items-center justify-between gap-2 text-xs">
          <label htmlFor={`${id}-title`} className="font-medium text-slate-300">Title</label>
          <span className="text-[10px] text-slate-500">{fields.title.length} characters</span>
        </div>
        <textarea id={`${id}-title`} rows={2} value={fields.title}
          onChange={event => {
            // Pasting a full Git message into Title retains the remaining
            // paragraphs and unknown trailers in their respective fields.
            const pasted = event.target.value;
            if (pasted.includes("\n")) {
              const parsed = parseMessage(pasted);
              change({ title: parsed.title, description: parsed.description || fields.description, trailers: parsed.trailers || fields.trailers });
            } else change({ ...fields, title: pasted });
          }}
          placeholder={amend ? "Leave empty to keep the previous message" : "Summarize what changed"}
          disabled={disabled}
          className="w-full resize-y rounded-lg border border-slate-700 bg-slate-800/50 px-3 py-2 gct-commit-input text-slate-100 outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 disabled:opacity-60"
          data-testid="commit-message-input"
        />
      </div>
      <div>
        <label htmlFor={`${id}-description`} className="mb-1.5 block text-xs font-medium text-slate-300">Description <span className="font-normal text-slate-500">· optional</span></label>
        <textarea id={`${id}-description`} rows={5} value={fields.description} onChange={event => change({ ...fields, description: event.target.value })}
          placeholder="Explain why, or add useful details…" disabled={disabled}
          className="w-full resize-y rounded-lg border border-slate-700 bg-slate-800/50 px-3 py-2 gct-commit-input leading-relaxed text-slate-100 outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 disabled:opacity-60"
          data-testid="commit-message-description"
        />
      </div>
      <details className="text-xs text-slate-400">
        <summary className="flex cursor-pointer items-center gap-2 py-2"><FileText className="h-4 w-4" aria-hidden="true" />References and trailers · optional{fields.trailers ? " · kept" : ""}</summary>
        <label htmlFor={`${id}-trailers`} className="sr-only">References and trailers</label>
        <textarea id={`${id}-trailers`} rows={3} value={fields.trailers} onChange={event => change({ ...fields, trailers: event.target.value })} disabled={disabled}
          placeholder="Reference-Key: value" className="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 font-mono gct-commit-input gct-commit-input-compact text-slate-200 outline-none focus:border-cyan-500"
          data-testid="commit-message-trailers" />
      </details>
      <p className="text-[11px] leading-relaxed text-slate-500">{stagedCount === 0 ? "Stage files to enable suggestions." : updatingIndex ? "Finishing staging before checking suggestions…" : "Suggest checks existing evidence first. An agent starts only when you choose."}</p>
      <MessageSuggestionDialog suggestions={suggestions} message={value} disabled={disabled || updatingIndex} onApply={next => change(parseMessage(next))} stagedPaths={stagedPaths} onUnstage={onUnstage} onQueueStagedExtras={onQueueStagedExtras} queuedGroups={queuedGroups} startingQueuedSuggestionIds={startingQueuedSuggestionIds} onStartQueuedSuggestion={onStartQueuedSuggestion} onShowQueue={onShowQueue} />
    </div>
  );
}
