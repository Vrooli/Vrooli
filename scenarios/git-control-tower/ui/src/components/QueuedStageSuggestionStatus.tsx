import { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertCircle, Check, Clock3, LoaderCircle, RotateCw, Sparkles, RefreshCw, Search } from "lucide-react";
import { StagedMessageMode } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
import type { StagedMessageJob, StagedMessageSuggestion } from "../features/commit-message/api";
import { workflowExecutionLink } from "../features/commit-message/workflowExecutionLink";
import { getMessageJob, getMessageProviderReadiness, waitMessageJob } from "../features/commit-message/api";
import { formatMicroUSD } from "../features/commit-message/charge";
import type { QueuedStageSelection } from "../features/commit-message/stageQueue";

type DisplayState = "saved" | "awaiting_unstage" | "queued" | "dispatching" | "running" | "awaiting_input" | "awaiting_provider" | "cancelling" | "failed" | "cancelled" | "stale" | "succeeded" | "unavailable";

interface Props {
  repositoryId: string;
  selection: QueuedStageSelection;
  /** True only while the parent has staged this queued selection. */
  isStaged: boolean;
  /** Passes the captured suggestion to the owner; this card never edits a message. */
  onUse: (suggestion: StagedMessageSuggestion) => void;
  /** Lets the queue owner retry a failed launch without transferring state ownership. */
  onRetry?: (selection: QueuedStageSelection) => void;
  /** Lets the queue owner refresh stale captured paths; this card does not stage files. */
  onRefresh?: (selection: QueuedStageSelection) => void;
  onStartSuggestion?: (selection: QueuedStageSelection, mode: StagedMessageMode) => void;
  onFinishUnstage?: (selection: QueuedStageSelection) => void;
}

const activeStatuses = new Set(["queued", "dispatching", "running", "awaiting_input", "cancelling"]);
const pause = (ms: number, signal: AbortSignal) => new Promise<void>(resolve => {
  if (signal.aborted) return resolve();
  const timer = window.setTimeout(done, ms);
  function done() { window.clearTimeout(timer); signal.removeEventListener("abort", done); resolve(); }
  signal.addEventListener("abort", done, { once: true });
});

function statusOf(job: StagedMessageJob): DisplayState {
  if (!job.applicable && !activeStatuses.has(job.status)) return "stale";
  if (job.status === "succeeded") return "succeeded";
  if (job.status === "stale") return "stale";
  if (job.status === "cancelled") return "cancelled";
  if (job.status === "failed") return "failed";
  if (job.status === "awaiting_provider") return "awaiting_provider";
  if (activeStatuses.has(job.status)) return job.status as DisplayState;
  return "unavailable";
}

function StateMark({ state }: { state: DisplayState }) {
  const icon = state === "running" ? <LoaderCircle className="h-4 w-4 animate-spin" aria-hidden="true" />
    : state === "succeeded" ? <Check className="h-4 w-4" aria-hidden="true" />
    : state === "queued" || state === "saved" ? <Clock3 className="h-4 w-4" aria-hidden="true" />
    : <AlertCircle className="h-4 w-4" aria-hidden="true" />;
  const label = ({ unavailable: "Status unavailable", awaiting_unstage: "Unstage to finish", awaiting_input: "Awaiting input", awaiting_provider: "Waiting for provider admission", dispatching: "Dispatching", cancelling: "Cancelling" } as Partial<Record<DisplayState, string>>)[state] || state.charAt(0).toUpperCase() + state.slice(1);
  const color = state === "succeeded" ? "text-emerald-200" : state === "stale" || state === "failed" || state === "awaiting_provider" || state === "unavailable" ? "text-amber-200" : "text-cyan-200";
  return <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${color}`} role="status" data-testid="queued-suggestion-state">{icon}{label}</span>;
}

/** Reattaches to one durable worktree suggestion and observes its owner with Wait. */
export function QueuedStageSuggestionStatus({ repositoryId, selection, isStaged, onUse, onRetry, onRefresh, onStartSuggestion, onFinishUnstage }: Props) {
  const queryClient = useQueryClient();
  const handle = selection.requestKey || selection.jobId || "";
  const queryKey = ["commit-message", "queued-suggestion-job", repositoryId, handle] as const;
  const jobQuery = useQuery({
    queryKey,
    enabled: Boolean(handle),
    queryFn: async ({ signal }) => {
      let observed = await getMessageJob(repositoryId, selection.jobId ?? "", selection.requestKey ?? "");
      if (signal.aborted) throw new DOMException("Aborted", "AbortError");
      queryClient.setQueryData(queryKey, observed);
      let unchanged = 0;
      let fingerprint = "";
      while (activeStatuses.has(observed.status)) {
        const began = Date.now();
        observed = await waitMessageJob(repositoryId, selection.jobId ?? "", selection.requestKey ?? "", signal);
        if (signal.aborted) throw new DOMException("Aborted", "AbortError");
        queryClient.setQueryData(queryKey, observed);
        if (!activeStatuses.has(observed.status)) break;
        const nextFingerprint = `${observed.status}\0${observed.phase}`;
        unchanged = nextFingerprint === fingerprint ? unchanged + 1 : 0;
        fingerprint = nextFingerprint;
        await pause(Math.max(0, Math.min(8000, 1000 * 2 ** Math.min(unchanged, 3)) - (Date.now() - began)), signal);
      }
      return observed;
    },
    retry: false,
    staleTime: 0,
    refetchOnMount: true,
    refetchOnWindowFocus: false,
  });
  const job = jobQuery.data as StagedMessageJob | undefined;
  const providerNotReadyFailure = jobMaintenance(selection.suggestionError) || ["provider_admission_maintenance", "provider_unavailable"].includes(job?.errorCode ?? "");
  const readinessQuery = useQuery({
    queryKey: ["commit-message", "provider-readiness"],
    queryFn: ({ signal }) => getMessageProviderReadiness(signal),
    enabled: providerNotReadyFailure,
    retry: false,
    staleTime: 0,
    refetchOnWindowFocus: false,
    refetchInterval: providerNotReadyFailure ? 5000 : false,
  });
  const readinessPaused = providerNotReadyFailure
    && !(readinessQuery.data?.known && readinessQuery.data.state === "open");
  const error = jobQuery.error instanceof Error ? jobQuery.error.message : jobQuery.error ? "Could not reconnect to this suggestion." : undefined;

  const state: DisplayState = selection.state === "awaiting_unstage" ? "awaiting_unstage"
    : job ? statusOf(job)
    : selection.state === "stale" ? "stale"
    : error || selection.suggestionError || selection.suggestionState === "failed" ? "failed"
    : selection.suggestionState === "running" ? "running"
    : selection.suggestionState === "succeeded" ? "unavailable"
    : selection.suggestionMode ? "queued" : "saved";
  const suggestion = job?.status === "succeeded" ? job.result : undefined;
  const canUse = Boolean(isStaged && job?.status === "succeeded" && job.applicable && suggestion);
  const [clockNow, setClockNow] = useState(() => Date.now());
  useEffect(() => {
    if (!job || !activeStatuses.has(job.status)) return;
    const timer = window.setInterval(() => setClockNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [job?.createdAt, job?.status]);
  const elapsedEnd = job && activeStatuses.has(job.status) ? clockNow : job ? Date.parse(job.updatedAt) : undefined;
  const elapsedSeconds = job && elapsedEnd !== undefined ? Math.max(0, Math.floor((elapsedEnd - Date.parse(job.createdAt)) / 1000)) : undefined;
  return <article className="space-y-3 rounded-xl border border-slate-700 bg-slate-900/70 p-4" data-testid="queued-suggestion-card">
    <header className="flex flex-wrap items-center justify-between gap-2">
      <div className="flex items-center gap-2"><Sparkles className="h-4 w-4 text-cyan-300" aria-hidden="true" /><h3 className="text-sm font-semibold text-slate-100">{selection.suggestionMode ? `Queued ${selection.suggestionMode === "deep" ? "deep" : "quick"} suggestion` : "Queued selection"}</h3></div>
      <StateMark state={state} />
    </header>
    <p className="text-xs text-slate-400">{selection.paths.length} {selection.paths.length === 1 ? "file" : "files"} captured · queued {new Date(selection.queuedAt).toLocaleString()}</p>
    {state === "saved" && <><p className="text-sm text-slate-300">The files are saved in the queue. Choose a suggestion when you are ready.</p>{onStartSuggestion && <div className="flex flex-wrap gap-2"><button type="button" onClick={() => onStartSuggestion(selection, StagedMessageMode.QUICK)} className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-slate-600 px-3 text-xs text-cyan-200"><Sparkles className="h-3.5 w-3.5" aria-hidden="true" />Start quick suggestion</button><button type="button" onClick={() => onStartSuggestion(selection, StagedMessageMode.DEEP)} className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-slate-600 px-3 text-xs text-cyan-200"><Search className="h-3.5 w-3.5" aria-hidden="true" />Start deep investigation</button></div>}</>}
    {state === "awaiting_unstage" && <><p className="text-sm text-amber-100">This saved selection is still staged with the current commit. Unstage it to finish queueing.</p>{onFinishUnstage && <button type="button" onClick={() => onFinishUnstage(selection)} className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-amber-700/70 px-3 text-xs text-amber-100"><RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />Finish queueing</button>}</>}
    {state === "queued" && <p className="text-sm text-slate-300">The selected suggestion job is waiting to start.</p>}
    {state === "dispatching" && <p className="text-sm text-slate-300">The saved job is being sent to Agent Manager.</p>}
    {state === "running" && job?.errorCode === "evidence_unavailable" ? <p className="text-sm text-slate-300">GCT is still reconciling the completed result. The saved owner identity is retained while its evidence is re-read.</p> : state === "running" && <p className="text-sm text-slate-300">The drafting job is active. This card reconnects to its saved owner and waits for an update.</p>}
    {job?.error && activeStatuses.has(job.status) && <div className="space-y-2 rounded-lg border border-amber-800/70 bg-amber-950/30 p-3" role="alert"><p className="text-sm text-amber-100">{job.error}</p><button type="button" onClick={() => { void queryClient.invalidateQueries({ queryKey }); }} className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-slate-600 px-3 text-xs text-cyan-200"><RotateCw className="h-3.5 w-3.5" aria-hidden="true" />Reconnect to saved job</button></div>}
    {state === "awaiting_input" && <p className="text-sm text-slate-300">The agent is waiting for input before it can continue.</p>}
    {state === "awaiting_provider" && <p className="text-sm text-amber-100">{job?.error || "Workspace Sandbox is temporarily closed for lifecycle maintenance."} The captured selection is preserved. Retry starts a new job only after provider readiness is known open.</p>}
    {state === "succeeded" && !isStaged && <p className="text-sm text-slate-300">Suggestion ready. It becomes available to use after the current commit succeeds and this selection is staged.</p>}
    {state === "stale" && <p className="text-sm text-amber-100">The captured files have changed, so this suggestion cannot be used. Remove this selection, then stage and queue the current files as a new selection.</p>}
    {state === "failed" && <p className="text-sm text-amber-100">{job?.error || selection.suggestionError || error || "The drafting job did not finish successfully."}</p>}
    {state === "cancelled" && <p className="text-sm text-slate-300">The drafting job was cancelled. The captured selection remains saved.</p>}
    {job && <p className="text-xs text-slate-400">Last stage: {job.phase || job.status}{elapsedSeconds !== undefined && Number.isFinite(elapsedSeconds) ? ` · elapsed ${elapsedSeconds}s` : ""}</p>}
    {job?.errorCode && <p className="text-xs text-slate-400">Failure category: {job.errorCode}</p>}
    {job?.executionStartedKnown ? <p className="text-xs text-slate-400">{job.executionStarted ? "Agent started" : "Agent did not start"}{job.tokensKnown ? ` · ${job.tokensUsed} tokens used` : ""}</p>
      : job?.tokensKnown ? <p className="text-xs text-slate-400">Agent start status unknown · {String(job.tokensUsed)} tokens recorded</p>
      : job && ["failed", "cancelled", "stale", "awaiting_provider"].includes(state) ? <p className="text-xs text-slate-400">Agent start and token usage are unknown.</p> : null}
    {job && ["failed", "cancelled", "stale", "awaiting_provider"].includes(state) && <p className="text-xs text-slate-400">{job.chargeKnown ? `Marginal charge: ${formatMicroUSD(job.chargeMicroUsd)} USD.` : "Marginal charge is unknown; token counts are not a charge estimate."}</p>}
    {job?.executionId && <a className="text-xs text-cyan-200 underline" href={workflowExecutionLink(job.executionId, window.location.hostname, window.location.protocol)} target="_blank" rel="noreferrer">Open workflow execution</a>}
    {(state === "failed" || state === "awaiting_provider") && readinessPaused && <p className="text-xs text-slate-300">Retry is paused until Workspace Sandbox readiness is known open. The saved selection is preserved.</p>}
    {(state === "awaiting_provider" || state === "failed") && job && job.retryable && onRefresh && <button type="button" disabled={readinessPaused} onClick={() => onRefresh(selection)} className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-amber-700/70 px-3 text-xs text-amber-100 disabled:cursor-not-allowed disabled:opacity-50"><RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />Start a new job with the same selection</button>}
    {state === "failed" && job && !job.retryable && onStartSuggestion && <div className="space-y-2"><p className="text-xs text-slate-300">This attempt is marked non-retryable with the saved mode. Choose a different mode or queue a changed selection to start a new job.</p><div className="flex flex-wrap gap-2">{(!selection.suggestionMode || selection.suggestionMode !== "quick") && <button type="button" onClick={() => onStartSuggestion(selection, StagedMessageMode.QUICK)} className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-slate-600 px-3 text-xs text-cyan-200"><Sparkles className="h-3.5 w-3.5" aria-hidden="true" />Start a new quick draft</button>}{(!selection.suggestionMode || selection.suggestionMode !== "deep") && <button type="button" onClick={() => onStartSuggestion(selection, StagedMessageMode.DEEP)} className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-slate-600 px-3 text-xs text-cyan-200"><Search className="h-3.5 w-3.5" aria-hidden="true" />Start a new deep investigation</button>}</div></div>}
    {state === "failed" && !job && onRetry && <button type="button" disabled={readinessPaused} onClick={() => onRetry(selection)} className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-slate-600 px-3 text-xs text-cyan-200 disabled:cursor-not-allowed disabled:opacity-50"><RotateCw className="h-3.5 w-3.5" aria-hidden="true" />Retry queued suggestion</button>}
    {state === "unavailable" && <div className="flex flex-wrap items-center justify-between gap-2"><p className="text-sm text-amber-100">The owner returned an unrecognized status. Reconnect to check its saved state.</p><button type="button" onClick={() => { void queryClient.invalidateQueries({ queryKey }); }} className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-slate-600 px-3 text-xs text-cyan-200"><RotateCw className="h-3.5 w-3.5" aria-hidden="true" />Reconnect</button></div>}
    {error && state !== "failed" && <p className="text-xs text-amber-200">Reconnect issue: {error}</p>}
    {suggestion && <section className="space-y-2 rounded-lg border border-slate-700 bg-slate-950/60 p-3" aria-label="Captured suggestion">
      <h4 className="break-words text-sm font-semibold text-slate-100">{suggestion.title}</h4>
      {suggestion.body && <p className="whitespace-pre-wrap break-words text-sm text-slate-300">{suggestion.body}</p>}
      {job?.applicabilityReason && !job.applicable && <p className="text-xs text-amber-200">{job.applicabilityReason}</p>}
    </section>}
    {canUse && suggestion && <div className="space-y-2"><p className="text-xs text-slate-400">This suggestion will be passed to the parent for review. Your commit message stays unchanged until you choose to apply it.</p><button type="button" onClick={() => onUse(suggestion)} className="inline-flex min-h-10 items-center gap-2 rounded-lg border border-cyan-700 bg-cyan-950/40 px-3 text-sm font-medium text-cyan-100 hover:bg-cyan-950/70"><Check className="h-4 w-4" aria-hidden="true" />Use suggestion</button></div>}
    {error && <button type="button" onClick={() => { void queryClient.invalidateQueries({ queryKey }); }} className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-slate-600 px-3 text-xs text-cyan-200"><RotateCw className="h-3.5 w-3.5" aria-hidden="true" />Reconnect</button>}
  </article>;
}

function jobMaintenance(message?: string) {
  return Boolean(message?.toLowerCase().includes("workspace sandbox maintenance"));
}

