import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { ArrowLeft, Check, ChevronDown, Eye, FileSearch, Info, Loader2, MessageSquarePlus, RefreshCw, Search, Sparkles, Trash2, X, type LucideIcon } from "lucide-react";
import { Button } from "../../components/ui/button";
import { isMessageJobActive, type useMessageSuggestions } from "./useMessageSuggestions";
import { getDraftScope, isLimitedDraft, mergeSuggestionTrailers, parseMessage, serializeMessage } from "./message";
import type { StagedMessageSuggestion } from "./api";

type Suggestions = ReturnType<typeof useMessageSuggestions>;
const actionClass = "inline-flex items-center justify-center gap-2 min-h-11 rounded-lg border border-slate-600 px-4 py-2 text-xs font-medium text-slate-200 hover:bg-slate-800 disabled:opacity-50";
const primaryClass = "inline-flex items-center justify-center gap-2 w-full rounded-lg bg-cyan-300 text-slate-950 hover:bg-cyan-200 sm:w-auto";
const count = (value: number) => value.toLocaleString("en-US");

function PreviewList<T>({ items, label, pageSize = 20, children }: {
  items: readonly T[]; label: string; pageSize?: number; children: (item: T, index: number) => ReactNode;
}) {
  const [limit, setLimit] = useState(pageSize);
  if (!items.length) return null;
  return <div className="space-y-2">
    {items.length > pageSize && <p className="text-[11px] text-slate-400">Showing {count(Math.min(limit, items.length))} of {count(items.length)} {label.toLowerCase()}.</p>}
    <ul aria-label={label} className="space-y-2">{items.slice(0, limit).map((item, index) => <li key={index}>{children(item, index)}</li>)}</ul>
    {limit < items.length && <button type="button" className="inline-flex min-h-11 items-center gap-2 rounded border border-slate-600 px-3 text-xs text-cyan-200" onClick={() => setLimit(current => current + pageSize)}><ChevronDown className="h-4 w-4" aria-hidden="true" />Show {Math.min(pageSize, items.length - limit)} more {label.toLowerCase()}</button>}
  </div>;
}

function useElapsed(createdAt?: string) {
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    if (!createdAt) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [createdAt]);
  const seconds = createdAt ? Math.max(0, Math.floor((now - Date.parse(createdAt)) / 1000)) : Number.NaN;
  return Number.isFinite(seconds) ? `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}` : "—";
}

function SuggestionPreview({ suggestion, agent }: { suggestion: StagedMessageSuggestion; agent: boolean }) {
  const coverage = suggestion.coverage;
  const scope = getDraftScope(suggestion);
  const scoped = scope === "whole" || scope === "limited" || scope === "unknown";
  const limited = isLimitedDraft(suggestion);
  return (
    <article className="space-y-4" data-testid="commit-message-suggestion">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1"><span className={`text-[11px] font-medium ${limited || scope === "unknown" ? "text-amber-200" : "text-cyan-200"}`}>{agent ? "Agent suggestion" : limited ? "Partial reusable suggestion" : "Reusable suggestion"}</span>
        {scoped && <span className={`text-xs font-medium ${scope === "whole" ? "text-cyan-200" : "text-amber-200"}`}>{scope === "whole" ? "Whole-selection draft" : scope === "limited" ? "Limited draft" : "Draft scope unavailable"}</span>}
      </div>
      <h3 className="break-words text-base font-semibold leading-relaxed text-slate-100">{suggestion.title}</h3>
      {suggestion.body && <p className="whitespace-pre-wrap break-words text-sm leading-relaxed text-slate-300">{suggestion.body}</p>}
      {scope === "whole" && <p className="text-xs text-slate-300">Summarizes the whole staged selection.</p>}
      {scoped && !limited && suggestion.messageScopeExplanation && <p className="text-xs text-slate-400">{suggestion.messageScopeExplanation}</p>}
      {coverage && <div className="space-y-1 text-xs text-slate-400">
        {coverage.coveredPaths.length + coverage.uncoveredPaths.length > 0 && <p>{scoped ? "Detailed evidence inspected:" : "Fully describes"} {count(coverage.coveredPaths.length)} of {count(coverage.coveredPaths.length + coverage.uncoveredPaths.length)} staged files.</p>}
      </div>}
      {limited && <div className="space-y-2 rounded-lg border border-amber-800/70 bg-amber-950/20 p-3 text-xs text-amber-100" data-testid="commit-message-partial-coverage">
        {scope === "limited" ? <><h4 className="font-semibold">Why this draft is limited</h4><p>{suggestion.messageScopeExplanation || "Some themes remain unresolved. Review this draft before using it."}</p></> : <p>{agent ? "Some changes are not described by this result." : "Other changes are included as related updates."} Review them before using this draft.</p>}
      </div>}
      {scope === "unknown" && <p className="text-xs text-amber-200">The draft scope could not be interpreted. Review the message and its evidence before using it.</p>}
      {scoped && suggestion.unknowns.length > 0 && <section className="space-y-2"><h4 className="text-xs font-medium text-amber-200">Uncertainties to review</h4><PreviewList items={suggestion.unknowns} label="Synthesis uncertainties" pageSize={3}>{unknown => <span className="text-xs text-amber-200">{unknown}</span>}</PreviewList></section>}
    </article>
  );
}

type Evidence = { patches: number; limited: number; groups: [string, number][]; reasons: [string, number][] };

function EvidencePanel({ s, candidate, evidence }: { s: Suggestions; candidate?: StagedMessageSuggestion; evidence: Evidence }) {
  const [open, setOpen] = useState(false);
  const collection = s.session.collection;
  const scoped = candidate && !getDraftScope(candidate).startsWith("legacy_");
  return <details open={open} className="min-w-0 border-t border-slate-700 pt-2 text-xs text-slate-400 lg:border-l lg:border-t-0 lg:pl-5">
    <summary className="flex min-h-11 cursor-pointer flex-wrap items-center justify-between gap-2 text-slate-300" onClick={event => { event.preventDefault(); setOpen(current => !current); }}><span className="inline-flex items-center gap-2"><FileSearch className="h-4 w-4" aria-hidden="true" />Review evidence</span><span className="text-[11px] text-slate-400">Initial patch details: {count(evidence.patches)} · {collection?.contexts.length ?? 0} {(collection?.contexts.length ?? 0) === 1 ? "source" : "sources"}</span></summary>
    {open && <div className="space-y-5 py-3">
      <div className="space-y-2">{s.job && s.job.evidenceFilesTotal > 0 && <p className="font-medium text-cyan-200">{s.job.continuationOfJobId ? "Additional detailed files read" : "Detailed files read"}: {count(s.job.evidenceFilesRead)}</p>}<p>Initial patch details: {count(evidence.patches)} {evidence.patches === 1 ? "file" : "files"}. Initial evidence limits: {count(evidence.limited)} {evidence.limited === 1 ? "file" : "files"}.</p><p>All staged paths and content identities remain bound to this suggestion. Evidence limits do not change which files will be committed.</p><p>These counts describe the initial evidence only. Further investigation can read more staged evidence; draft scope is shown separately from detailed inspection.</p></div>
      <section><h4 className="mb-2 font-medium text-slate-300">Staged file groups ({count(evidence.groups.length)})</h4><PreviewList key={s.snapshot?.digest} items={evidence.groups} label="Staged file groups" pageSize={8}>{([group, files]) => <span className="break-all font-mono">{group} · {count(files)} files</span>}</PreviewList></section>
      {candidate?.coverage?.uncoveredPaths.length ? <section><h4 className={`mb-2 font-medium ${scoped ? "text-slate-300" : "text-amber-200"}`}>{scoped ? "Files without detailed inspection" : "Changes not fully described"}</h4><PreviewList key={candidate.id} items={candidate.coverage.uncoveredPaths} label={scoped ? "Files without detailed inspection" : "Uncovered staged files"}>{path => <span className="break-all font-mono text-[11px]">{path}</span>}</PreviewList></section> : null}
      {candidate && !scoped && <PreviewList key={`unknowns-${candidate.id}`} items={candidate.unknowns} label="Suggestion unknowns">{unknown => <span className="text-amber-200">{unknown}</span>}</PreviewList>}
      {collection && <section><h4 className="mb-2 font-medium text-slate-300">Related context ({collection.contexts.length})</h4>{collection.contexts.length === 0 && <p className="mb-3">No related work context is available. Staged-diff evidence is separate and remains available where captured.</p>}<p className="mb-3">{s.session.selectedContextIDs.length} selected; up to 20. Relationships explain intent, not authorship.</p>
        <PreviewList key={collection.id} items={collection.contexts} label="Related sources" pageSize={10}>{context => <div className="space-y-2 border-b border-slate-700 pb-3"><label className="flex items-start gap-3">
          <input type="checkbox" checked={s.session.selectedContextIDs.includes(context.id)} disabled={Boolean(s.action) || Boolean(s.session.jobID || s.session.requestKey) || (!s.session.selectedContextIDs.includes(context.id) && s.session.selectedContextIDs.length >= 20) || ["private", "unavailable", "inaccessible"].includes(context.status)} onChange={event => s.update({ selectedContextIDs: event.target.checked ? [...s.session.selectedContextIDs, context.id] : s.session.selectedContextIDs.filter(id => id !== context.id) })} className="mt-1 h-4 w-4" />
          <span className="min-w-0"><span className="block font-medium text-slate-200">{context.title || context.kind}</span><span className="block">{context.detail || context.status}</span></span>
        </label><div className="ml-7 space-y-2"><PreviewList items={context.matchedPaths} label="Matched staged files">{path => <span className="break-all font-mono text-[11px]">{path}</span>}</PreviewList>{context.body && <p className="whitespace-pre-wrap break-words text-slate-300">{context.body.slice(0, 1200)}{context.body.length > 1200 ? "… (excerpt)" : ""}</p>}<PreviewList items={context.unknowns} label="Source unknowns">{unknown => <span className="text-amber-200">{unknown}</span>}</PreviewList></div></div>}</PreviewList>
      </section>}
      {collection?.omissions.length ? <section><h4 className="mb-2 font-medium text-amber-200">Initial evidence limits ({count(collection.omissions.length)})</h4><PreviewList key={`reasons-${collection.id}`} items={evidence.reasons} label="Limit reasons">{([reason, total]) => <span>{reason} · {count(total)} entries</span>}</PreviewList><details className="mt-3"><summary className="flex min-h-11 cursor-pointer items-center gap-2 py-3"><FileSearch className="h-4 w-4" aria-hidden="true" />Inspect limit entries</summary><PreviewList key={collection.id} items={collection.omissions} label="Evidence limit entries">{omission => <span><span className="break-all font-mono">{omission.path}</span>{omission.path ? ": " : ""}{omission.reason}</span>}</PreviewList></details></section> : null}
    </div>}
  </details>;
}

function OptionalContext({ s, defaultOpen }: { s: Suggestions; defaultOpen: boolean }) {
  const [open, setOpen] = useState(defaultOpen);
  return <details open={open} className="text-xs text-slate-400"><summary className="flex min-h-11 cursor-pointer items-center gap-2 py-3 text-slate-300" onClick={event => { event.preventDefault(); setOpen(current => !current); }}><MessageSquarePlus className="h-4 w-4" aria-hidden="true" />Add context <span className="text-slate-400">· optional</span></summary>{open && <label className="block">Why were these changes needed?<textarea value={s.session.note} onChange={event => s.update({ note: event.target.value })} rows={2} maxLength={4000} disabled={Boolean(s.session.requestKey)} className="mt-2 w-full rounded-lg border border-slate-600 bg-slate-800 p-3 gct-commit-input text-slate-100 outline-none focus:border-cyan-400" placeholder="Add a sentence of context for the drafting agent" /></label>}</details>;
}

export function MessageSuggestionDialog({ suggestions: s, message, disabled, onApply }: {
  suggestions: Suggestions; message: string; disabled: boolean; onApply: (message: string) => void;
}) {
  const currentMessage = useRef(message);
  currentMessage.current = message;
  const disabledRef = useRef(disabled);
  disabledRef.current = disabled;
  const applying = useRef(false);
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; };
  }, []);
  const [selectedID, setSelectedID] = useState("");
  const [comparison, setComparison] = useState<StagedMessageSuggestion>();
  const elapsed = useElapsed(s.job?.createdAt);
  const collection = s.session.collection;
  const evidence = useMemo(() => {
    const groups = new Map<string, number>();
    let patches = 0;
    let limited = 0;
    for (const file of s.snapshot?.files ?? []) {
      if (file.patch) patches++;
      if (file.omissionReason) limited++;
      const segments = file.path.split("/", 3);
      const directory = segments[0] ?? "(repository root)";
      const group = segments.length === 1 ? "(repository root)" : ["scenarios", "packages"].includes(directory) ? segments.slice(0, 2).join("/") : directory;
      groups.set(group, (groups.get(group) ?? 0) + 1);
    }
    const reasons = new Map<string, number>();
    for (const omission of collection?.omissions ?? []) reasons.set(omission.reason, (reasons.get(omission.reason) ?? 0) + 1);
    return {
      patches, limited,
      groups: [...groups].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])),
      reasons: [...reasons].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])),
    };
  }, [s.snapshot, collection?.omissions]);
  const active = isMessageJobActive(s.job?.status);
  const agent = s.job?.status === "succeeded" && Boolean(s.job.result);
  const resultScope = s.job?.result ? getDraftScope(s.job.result) : undefined;
  const keepPrevious = !agent || isLimitedDraft(s.job?.result) || resultScope === "unknown";
  const previous = keepPrevious ? s.previousSuggestion : undefined;
  const candidates = agent && s.job?.result ? [s.job.result] : previous ? [previous] : collection?.suggestions ?? [];
  const candidate = candidates.find(item => item.id === selectedID) ?? candidates[0];
  const collecting = s.session.phase === "collecting";
  const terminalFailure = s.job?.status === "failed" || s.job?.status === "cancelled";
  const unknownStatus = Boolean(s.job && !["queued", "running", "cancelling", "succeeded", "failed", "cancelled"].includes(s.job.status));
  const invalidResult = s.job?.status === "succeeded" && !s.job.result;
  const acceptSuggestion = async (suggestion: StagedMessageSuggestion, replace = false) => {
    if (disabledRef.current || active || applying.current || unknownStatus || invalidResult || ["starting", "recovering", "unknown", "not_found"].includes(s.admissionState) || s.reconnecting) return;
    const messageAtClick = currentMessage.current;
    applying.current = true;
    try {
      if (!(await s.verify()) || !s.canAdoptVerified() || disabledRef.current || !mounted.current) return;
      if (currentMessage.current !== messageAtClick || (!replace && currentMessage.current !== s.session.initialMessage && currentMessage.current.trim())) {
        setComparison(suggestion);
        return;
      }
      const existing = parseMessage(currentMessage.current);
      const rendered = serializeMessage({ title: suggestion.title, description: suggestion.body, trailers: mergeSuggestionTrailers(existing.trailers, suggestion.trailers) });
      onApply(rendered);
      s.update({ applied: true, initialMessage: rendered });
      setComparison(undefined);
      s.setOpen(false);
    } finally { applying.current = false; }
  };
  const close = () => { setComparison(undefined); s.setOpen(false); };
  const refresh = () => { setComparison(undefined); void s.collect(currentMessage.current); };

  const starting = s.admissionState === "starting";
  const recovering = s.admissionState === "recovering" || s.reconnecting;
  const unknownAdmission = s.admissionState === "unknown";
  const missingAdmission = s.admissionState === "not_found";
  const failedWithoutHandle = s.job?.status === "failed" && !s.job.executionId;
  const cancelling = s.action === "cancel" || s.job?.status === "cancelling" || Boolean(s.job?.cancelRequested);
  const confirmingLaunch = active && !s.job?.executionId;
  const detail = s.job?.error || s.session.launchError?.detail || s.error;
  const activeError = active && Boolean(s.error || s.job?.error);
  const signInInstruction = "Sign in through a trusted browser session before starting a drafting agent";
  const signInRequired = detail?.endsWith(signInInstruction);
  const authorizationError = /unauthenticated|unauthorized|401/i.test(detail ?? "");
  const humanError = s.session.launchError?.message || (signInRequired ? `${signInInstruction}.` : authorizationError ? "Drafting could not verify your sign-in. Your message is unchanged." : "Your message is unchanged. You can review the request and try again.");
  const reportedFailure = detail?.trim();
  const failureReason = reportedFailure
    ? signInRequired || authorizationError ? humanError : reportedFailure
    : "No failure explanation is available for this saved request.";
  const previousContext = s.job?.status === "failed"
    ? "The investigation failed. This earlier draft is still available."
    : s.job?.status === "cancelled" ? "The investigation was cancelled. This earlier draft is still available."
    : agent ? "This earlier draft is still available. The new draft may leave unresolved themes."
    : "This earlier draft is preserved while this request is unresolved.";
  const busy = Boolean(s.action) || disabled;
  const unavailable = terminalFailure || unknownStatus || invalidResult || s.session.phase === "cancelled";
  let primary: { label: string; run: () => void; disabled: boolean; icon: LucideIcon } = { label: "Back to message", run: close, disabled: false, icon: ArrowLeft };
  if (collecting) primary = { label: "Cancel", run: s.cancelCollection, disabled: false, icon: X };
  else if (starting) primary = { label: "Starting agent…", run: close, disabled: true, icon: Loader2 };
  else if (recovering) primary = { label: "Checking launch status…", run: close, disabled: true, icon: Loader2 };
  else if (unknownAdmission || activeError) primary = { label: "Reconnect to job", run: () => void s.reconnect(), disabled: busy, icon: RefreshCw };
  else if (missingAdmission) primary = s.canRetryAdmission ? { label: "Retry same request", run: () => void s.start(), disabled: busy, icon: RefreshCw } : { label: "Reconnect to job", run: () => void s.reconnect(), disabled: busy, icon: RefreshCw };
  else if (active) primary = { label: "Hide progress", run: close, disabled: false, icon: Eye };
  else if (s.staleReason) primary = { label: "Refresh evidence", run: refresh, disabled: busy, icon: RefreshCw };
  else if (comparison) primary = { label: "Use replacement", run: () => void acceptSuggestion(comparison, true), disabled: busy, icon: Check };
  else if (previous && terminalFailure) primary = { label: "Use previous draft", run: () => void acceptSuggestion(previous), disabled: busy, icon: Check };
  else if (unavailable || (s.error && !collection)) primary = { label: "Check evidence again", run: refresh, disabled: busy, icon: RefreshCw };
  else if (candidate) primary = { label: getDraftScope(candidate) === "limited" ? "Use limited suggestion" : getDraftScope(candidate) === "legacy_partial" ? "Use partial suggestion" : "Use suggestion", run: () => void acceptSuggestion(candidate), disabled: busy, icon: Check };
  else if (collection?.snapshot?.files.length) primary = { label: "Start drafting agent", run: () => void s.start(), disabled: busy, icon: Sparkles };
  const showInvestigation = s.canInvestigate && !comparison && !s.staleReason;
  const PrimaryIcon = primary.icon;
  const canPrepare = !collecting && !starting && !recovering && !unknownAdmission && !missingAdmission && !active && !unavailable && !comparison && !agent;
  return <ResponsiveDialog open={s.open} onClose={close}
    title={<span className="flex items-center gap-2 text-slate-100"><Sparkles className="h-4 w-4 text-cyan-300" aria-hidden="true" />Commit suggestions</span>}
    ariaLabel="Commit message suggestions" closeLabel="Hide suggestions" size="lg" contentPadding="comfortable" avoidKeyboard
    testId="commit-message-suggestions" panelClassName="!border-slate-700 !bg-slate-900 !text-slate-200"
    footer={<div className="flex w-full flex-col gap-2"><span className="text-[11px] text-slate-400">Fills the editor. You commit separately.</span><div data-testid="commit-message-actions" className="flex flex-col gap-2 sm:flex-row sm:justify-end"><Button type="button" data-testid="commit-message-primary" className={`${primaryClass} sm:order-last`} disabled={primary.disabled} onClick={primary.run}><PrimaryIcon className={`h-4 w-4 ${primary.icon === Loader2 ? "animate-spin" : ""}`} aria-hidden="true" />{primary.label}</Button>{showInvestigation && <button type="button" data-testid="commit-message-investigate" className={actionClass} disabled={busy} onClick={() => void s.investigateRemaining()}><Search className="h-4 w-4" aria-hidden="true" />{candidate && getDraftScope(candidate) === "limited" ? "Investigate unresolved themes" : "Investigate remaining changes"}</button>}</div></div>}
  >
    <div className="space-y-4 text-sm leading-relaxed">
      <p className="text-xs text-slate-400">{s.snapshot ? `Staged selection: ${count(s.snapshot.files.length)} ${s.snapshot.files.length === 1 ? "file" : "files"}` : "Staged selection"}</p>
      <div className="grid min-w-0 gap-5 lg:grid-cols-[minmax(0,1fr)_16rem]">
        <main className="min-w-0 space-y-4" data-testid="commit-message-stage">
          {starting || recovering ? <div role="status" className="space-y-3 py-2" data-testid="commit-drafting-progress"><h3 className="flex items-center gap-2 text-base font-semibold"><Loader2 className="h-4 w-4 animate-spin text-cyan-300" aria-hidden="true" />{starting ? "Starting drafting agent…" : "Checking launch status…"}</h3><p className="text-slate-400">{starting ? "Confirming your drafting request." : "Reconnecting to the existing request without starting another agent."} Your message stays unchanged.</p></div>
            : unknownAdmission || missingAdmission ? <div role="alert" className="space-y-3"><h3 className="text-base font-semibold text-amber-200">{missingAdmission ? "No job found yet" : "Could not confirm launch"}</h3><p className="text-slate-400">{missingAdmission ? "No saved job was found at the last check. Reconnect or retry the same request; its original identity is preserved." : "Your request may have been received. Reconnect before trying another launch."}</p><p className="text-slate-400">{humanError}</p></div>
            : active ? <div role="status" className="space-y-3 py-2" data-testid="commit-drafting-progress"><h3 className="text-base font-semibold">{s.action === "cancel" ? "Requesting cancellation…" : cancelling ? confirmingLaunch ? "Cancelling drafting request…" : "Stopping agent…" : confirmingLaunch ? "Confirming agent launch" : activeError ? `Drafting job · last known state: ${s.job?.status}` : s.job?.status === "cancelling" ? "Stopping agent…" : s.job?.status === "queued" ? "Drafting agent queued" : "Drafting agent is running"}</h3><div className="flex items-baseline gap-2"><span className="font-mono text-3xl tabular-nums">{elapsed}</span><span className="text-xs text-slate-400">elapsed</span></div><p className="flex items-center gap-2 text-slate-400"><Loader2 className="h-4 w-4 animate-spin text-cyan-300" aria-hidden="true" />{confirmingLaunch ? "Your request is saved. Confirming whether the agent has started." : activeError ? `Last reported phase: ${s.job?.phase || "unknown"}` : s.job?.phase || "Choosing useful staged evidence"}</p>{s.job && s.job.evidenceFilesTotal > 0 && <p className="text-xs text-slate-400">{s.job.continuationOfJobId ? "Additional detailed files read" : "Detailed files read"}: {count(s.job.evidenceFilesRead)}</p>}{activeError && <p role="alert" className="text-sm text-amber-200">{signInRequired || authorizationError ? humanError : "Status is temporarily unavailable."} Reconnect to follow this job.</p>}<button type="button" className={actionClass} disabled={busy || s.job?.status === "cancelling" || s.job?.cancelRequested} onClick={() => void s.cancelAgent()}><X className="h-4 w-4" aria-hidden="true" />{confirmingLaunch ? "Cancel request" : "Cancel agent"}</button></div>
            : collecting ? <div role="status" className="space-y-3 py-2"><h3 className="flex items-center gap-2 text-base font-semibold"><Loader2 className="h-4 w-4 animate-spin text-cyan-300" aria-hidden="true" />Gathering evidence</h3><p className="text-slate-400">Checking saved messages and related activity for the staged selection.</p></div>
            : unavailable && !comparison ? <div role={s.job?.status === "cancelled" || s.session.phase === "cancelled" ? "status" : "alert"} className="space-y-3"><h3 className="text-base font-semibold text-amber-200">{unknownStatus ? "Execution status unavailable" : failedWithoutHandle ? "Drafting request failed" : s.job?.status === "failed" ? "Drafting failed" : invalidResult ? "No completed suggestion" : "Cancelled"}</h3>{s.job?.status === "failed" ? <><p className="break-words whitespace-pre-wrap text-slate-200">{failureReason}</p>{failedWithoutHandle && <p className="text-slate-400">We could not confirm whether the agent started.</p>}<p className="text-slate-400">Your message is unchanged. {previous ? s.canInvestigate ? "Use the previous draft, or investigate again." : "You can use the previous draft." : "Check the evidence before trying again."}</p></> : <p className="text-slate-400">{unknownStatus ? `An unrecognized status was returned: ${s.job?.status}.` : invalidResult ? "The request completed without a valid commit message." : "Your original title and description are preserved."}</p>}</div>
            : comparison ? <div className="space-y-4"><h3 className="text-base font-semibold">Keep your edits in view</h3><p className="text-xs text-slate-400">You edited the message while this draft was being prepared.</p><div className="border-l-2 border-slate-600 pl-3"><p className="mb-2 text-xs text-slate-400">Your current message</p><pre className="whitespace-pre-wrap break-words font-sans text-sm">{message}</pre></div><SuggestionPreview suggestion={comparison} agent={agent || Boolean(previous)} /><button type="button" className={actionClass} onClick={() => setComparison(undefined)}><ArrowLeft className="h-4 w-4" aria-hidden="true" />Keep mine</button></div>
            : candidate ? <>{candidates.length > 1 && <label className="block text-xs text-slate-400">Reusable message<select className="mt-1 w-full rounded-lg border border-slate-600 bg-slate-800 p-2 gct-commit-input text-slate-100" value={candidate.id} onChange={event => setSelectedID(event.target.value)}>{candidates.map(item => <option key={item.id} value={item.id}>{item.title}</option>)}</select></label>}<SuggestionPreview suggestion={candidate} agent={agent} /></>
            : collection ? <div className="space-y-2 py-2"><h3 className="text-base font-semibold">{collection.status === "empty" ? "Nothing staged" : "No reusable message found"}</h3><p className="text-slate-400">{collection.status === "empty" ? "Stage changes before asking for a message." : "No saved message matched these staged changes. An agent can draft for the whole selection using staged-diff evidence; add context if it would help."}</p></div> : s.error ? <div role="alert"><h3 className="text-base font-semibold text-amber-200">Could not gather suggestions</h3><p className="mt-2 text-slate-400">{humanError}</p></div> : null}
          {detail && <details className="text-xs text-slate-400"><summary className="flex min-h-11 cursor-pointer items-center gap-2 py-3"><Info className="h-4 w-4" aria-hidden="true" />Technical details</summary><p className="break-words whitespace-pre-wrap">{detail}</p></details>}
          {previous && !comparison && (active || unavailable || unknownAdmission || missingAdmission || starting || recovering || agent && candidate && (isLimitedDraft(candidate) || getDraftScope(candidate) === "unknown")) && <section className="space-y-3 border-t border-slate-700 pt-4" data-testid="commit-message-previous-draft"><h3 className="flex items-center gap-2 text-sm font-semibold text-slate-200"><FileSearch className="h-4 w-4" aria-hidden="true" />Previous draft kept</h3><p className="text-xs text-slate-400">{previousContext}</p><SuggestionPreview suggestion={previous} agent />{agent && <button type="button" className={actionClass} disabled={busy} onClick={() => void acceptSuggestion(previous)}><Check className="h-4 w-4" aria-hidden="true" />Use previous draft</button>}</section>}
          {s.error && candidate && !unavailable && !active && !unknownAdmission && !missingAdmission && !starting && !recovering && <p role="alert" className="text-xs text-amber-200">Could not verify this request. Your message is unchanged; try again when the service is available.</p>}
          {s.staleReason && <div role="status" className="border-l-2 border-amber-500 pl-3 text-xs text-amber-200" data-testid="commit-message-stale"><strong>Review staged selection.</strong> {s.staleReason} Your message and this result are preserved.</div>}
          {canPrepare && collection?.snapshot?.files.length ? <OptionalContext key={collection.id} s={s} defaultOpen={!candidate} /> : null}
          {canPrepare && candidate && !s.staleReason && <button type="button" disabled={busy} className="inline-flex min-h-11 items-center gap-2 text-xs font-medium text-cyan-200 hover:underline disabled:opacity-50" onClick={() => void s.start()}><Search className="h-4 w-4" aria-hidden="true" />Ask agent to complete</button>}
          {agent && !comparison && <div className="flex flex-wrap gap-4"><button type="button" className="inline-flex min-h-11 items-center gap-2 text-xs text-slate-400 hover:text-slate-200" onClick={s.discard}><Trash2 className="h-4 w-4" aria-hidden="true" />Discard</button><button type="button" className="inline-flex min-h-11 items-center gap-2 text-xs text-slate-400 hover:text-slate-200" onClick={refresh}><RefreshCw className="h-4 w-4" aria-hidden="true" />Check evidence again</button></div>}
        </main>
        {!collecting && !comparison && s.snapshot && <EvidencePanel key={collection?.id ?? s.snapshot.digest} s={s} candidate={candidate} evidence={evidence} />}
      </div>
    </div>
  </ResponsiveDialog>;
}
