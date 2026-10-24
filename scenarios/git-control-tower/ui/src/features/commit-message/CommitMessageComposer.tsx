import { useEffect, useId, useRef, useState } from "react";
import { Eye, FileText, Sparkles, X } from "lucide-react";
import { getDraftScope, parseMessage, serializeMessage, type MessageFields } from "./message";
import { useMessageSuggestions, isMessageJobActive } from "./useMessageSuggestions";
import { MessageSuggestionDialog } from "./MessageSuggestionDialog";

interface Props {
  value: string;
  onChange: (message: string) => void;
  repoId: string;
  stagedCount: number;
  statusVersion: number;
  updatingIndex: boolean;
  disabled: boolean;
  amend: boolean;
}

export function CommitMessageComposer({ value, onChange, repoId, stagedCount, statusVersion, updatingIndex, disabled, amend }: Props) {
  const id = useId();
  const [fields, setFields] = useState(() => parseMessage(value));
  const emitted = useRef(value);
  const suggestions = useMessageSuggestions(repoId, statusVersion, updatingIndex);
  useEffect(() => {
    if (value !== emitted.current) {
      emitted.current = value;
      setFields(parseMessage(value));
    }
  }, [value]);
  const change = (next: MessageFields) => {
    setFields(next);
    emitted.current = serializeMessage(next);
    onChange(emitted.current);
  };
  const active = isMessageJobActive(suggestions.job?.status);
  const hasWork = suggestions.session.phase !== "idle" || Boolean(suggestions.job) || suggestions.reconnecting || Boolean(suggestions.error);
  const unknownJob = suggestions.job && !["queued", "running", "cancelling", "succeeded", "failed", "cancelled"].includes(suggestions.job.status);
  const resultScope = suggestions.job?.result ? getDraftScope(suggestions.job.result) : undefined;
  const readyLabel = resultScope === "whole" ? "Whole-selection draft ready" : resultScope === "limited" ? "Limited draft ready" : resultScope === "unknown" ? "Draft ready · scope unavailable" : "Agent suggestion ready";
  const label = suggestions.action === "cancel" ? "Requesting cancellation…"
    : suggestions.admissionState === "starting" ? "Starting drafting agent…"
    : suggestions.admissionState === "recovering" ? "Checking launch status…"
    : suggestions.admissionState === "unknown" ? "Could not confirm launch"
    : suggestions.staleReason ? "Review staged selection · draft retained"
    : unknownJob ? "Execution status unavailable"
    : suggestions.error ? "Suggestions unavailable · message retained"
    : suggestions.reconnecting ? "Reconnecting to drafting job…"
    : suggestions.session.phase === "collecting" ? "Checking existing evidence…"
    : active && !suggestions.job?.executionId ? "Confirming agent launch"
    : active ? suggestions.job?.status === "cancelling" ? "Stopping drafting agent…" : "Drafting agent in progress"
    : suggestions.session.applied ? "Suggestion applied · editable"
    : suggestions.job?.status === "succeeded" ? suggestions.job.result ? readyLabel : "No completed suggestion"
    : suggestions.job?.status === "failed" ? "Drafting request failed"
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
          onClick={() => hasWork ? suggestions.setOpen(true) : void suggestions.collect(value)}
          className="inline-flex min-h-11 items-center gap-1.5 rounded-lg border border-cyan-800/70 bg-cyan-950/30 px-3 text-xs text-cyan-200 hover:bg-cyan-950/60 disabled:opacity-50"
          data-testid="suggest-commit-message"
        >
          <Sparkles className="h-3.5 w-3.5" aria-hidden="true" /> Suggest
        </button>
      </div>
      {hasWork && (
        <div role="status" className={`flex flex-wrap items-center gap-2 rounded-lg border px-3 py-2 text-xs ${suggestions.staleReason ? "border-amber-800 bg-amber-950/30 text-amber-200" : "border-slate-700 bg-slate-900 text-slate-300"}`} data-testid="commit-suggestion-standing">
          <span className="min-w-0 flex-1">{label}</span>
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
      <MessageSuggestionDialog suggestions={suggestions} message={value} disabled={disabled || updatingIndex} onApply={next => change(parseMessage(next))} />
    </div>
  );
}
