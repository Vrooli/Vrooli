import { useEffect, useMemo, useState } from "react";
import { AlertTriangle, Check, CheckCircle2, Gauge, Info, Layers3, Loader2, Play, Sparkles } from "lucide-react";
import { Drawer } from "../ui/drawer";
import { Button } from "../ui/button";
import { Select } from "../ui/select";
import { Slider } from "@vrooli/react-component-library/Slider/1.2.4";
import { backlogService } from "../../services";
import type { QueueResponse } from "../../services";
import type { BacklogKind } from "../../types";
import { defaultApiClient, isApiError } from "../../lib/api-client";
import { API_ENDPOINTS } from "../../lib/api-endpoints";
import { StalePlanPanel } from "./stale-plan-panel";
import { extractMissingPaths, type MissingPath } from "./stale-plan-utils";
import { ExecutionLimitsSummary } from "./execution-limits-summary";

export interface RunSheetTarget {
  kind: BacklogKind;
  name: string;
  title?: string;
}

export interface ExecutionStrategy {
  id: string;
  workflow_key: string;
  display_name: string;
  description: string;
  when_to_use: string;
  cost_band: string;
  cost_estimate: number;
}

export interface ExecutionOption {
  runner_type: string;
  available: boolean;
  message: string;
  native_objective: boolean;
  default_model: string;
  models: Array<{ id: string; canonical_model: string; is_default: boolean }>;
  effort_levels: string[];
}

export interface RunSheetProps {
  isOpen: boolean;
  onClose: () => void;
  target?: RunSheetTarget;
  targets?: RunSheetTarget[];
  onSuccess?: (result: QueueResponse) => void;
}

// RunSheet is deliberately non-mutating on open. It first displays the
// current preflight and declared strategy, then the operator explicitly queues
// the selected work from the sticky footer.
export function RunSheet({ isOpen, onClose, target, targets, onSuccess }: RunSheetProps) {
  const effectiveTargets = useMemo(() => targets?.length ? targets : target ? [target] : [], [target, targets]);
  const isBulk = effectiveTargets.length > 1;
  const [preflight, setPreflight] = useState<QueueResponse | null>(null);
  const [previews, setPreviews] = useState<QueueResponse[]>([]);
  const [strategies, setStrategies] = useState<ExecutionStrategy[]>([]);
  const [executionOptions, setExecutionOptions] = useState<ExecutionOption[]>([]);
  const [strategy, setStrategy] = useState("");
  const [maxSlices, setMaxSlices] = useState(6);
  const [preferredRunner, setPreferredRunner] = useState("");
  const [model, setModel] = useState("");
  const [effort, setEffort] = useState("");
  const [force, setForce] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [stalePlanFor, setStalePlanFor] = useState<{ kind: BacklogKind; name: string; missingPaths: MissingPath[] } | null>(null);

  useEffect(() => {
    if (!isOpen) return;
    let active = true;
    setError(null); setForce(false); setStalePlanFor(null); setPreflight(null); setPreviews([]); setStrategies([]); setExecutionOptions([]); setStrategy(""); setMaxSlices(6); setPreferredRunner(""); setModel(""); setEffort(""); setLoading(true);
    const strategyEndpoint = !isBulk && effectiveTargets[0]
      ? `${API_ENDPOINTS.executionStrategies}?backlog_kind=${encodeURIComponent(effectiveTargets[0].kind)}&backlog_name=${encodeURIComponent(effectiveTargets[0].name)}`
      : API_ENDPOINTS.executionStrategies;
    void Promise.all([
      defaultApiClient.get<{ items: ExecutionStrategy[] }>(strategyEndpoint),
      defaultApiClient.get<{ options?: ExecutionOption[] }>(API_ENDPOINTS.executionOptions + "?role=code.smart"),
      Promise.all(effectiveTargets.map((item) => backlogService.queue(item.kind, item.name, { mode: "yolo", confirm: false }))),
    ]).then(([strategyResponse, optionResponse, itemPreviews]) => {
      if (!active) return;
      const preview = itemPreviews[0] ?? null;
      setPreviews(itemPreviews);
      setStrategies(strategyResponse.items ?? []);
      setExecutionOptions(optionResponse.options ?? []);
      const savedStrategy = preview?.item?.executionStrategy;
      // A saved strategy is part of the reviewed item. Do not replace it with
      // the first catalog entry or a selection retained from another item.
      if (savedStrategy && !strategyResponse.items?.some((entry) => entry.id === savedStrategy)) {
        setError(`The item's execution approach “${savedStrategy}” is unavailable.`);
      } else {
        setStrategy(savedStrategy || strategyResponse.items?.[0]?.id || "");
      }
      setPreflight(preview);
      setMaxSlices(preview?.item?.executionLimits?.maxSlices ?? 6);
      const firstRunner = (optionResponse.options ?? []).find((entry) => entry.available);
      setPreferredRunner(firstRunner?.runner_type ?? "");
      setModel(firstRunner?.default_model ?? firstRunner?.models?.find((entry) => entry.is_default)?.id ?? "");
      setEffort(firstRunner?.effort_levels?.[0] ?? "");
    }).catch((cause) => {
      if (active) setError(cause instanceof Error ? cause.message : "Unable to load run options.");
    }).finally(() => {
      if (active) setLoading(false);
    });
    return () => { active = false; };
  }, [effectiveTargets, isBulk, isOpen]);

  const blockingReasons = previews.flatMap((preview) => preview.blockingReasons);
  const mayForce = blockingReasons.length > 0 && blockingReasons.every((reason) => reason.forceable);
  const blocked = blockingReasons.length > 0 && !force;
  const title = isBulk ? `Run ${effectiveTargets.length} items` : target?.title ? `Run “${target.title}”` : "Run backlog item";
  const itemHref = !isBulk && effectiveTargets[0]
    ? `/backlog/${encodeURIComponent(effectiveTargets[0].kind)}/${encodeURIComponent(effectiveTargets[0].name)}`
    : undefined;
  const selectedStrategy = strategies.find((entry) => entry.id === strategy);
  const acceptedSliceLimit = preflight?.item?.executionLimits?.maxSlices ?? 6;
  const isGoalSession = strategy === "goal-session";
  const availableRunners = executionOptions.filter((entry) => entry.available && (!isGoalSession || entry.native_objective));
  const selectedRunner = availableRunners.find((entry) => entry.runner_type === preferredRunner) ?? availableRunners[0];
  const goalRunnerBlocked = isGoalSession && availableRunners.length === 0;
  const effectiveBlocked = blocked || (goalRunnerBlocked && !force);

  useEffect(() => {
    if (!selectedRunner) return;
    if (selectedRunner.runner_type !== preferredRunner) setPreferredRunner(selectedRunner.runner_type);
    const nextModel = selectedRunner.default_model || selectedRunner.models.find((entry) => entry.is_default)?.id || "";
    if (!selectedRunner.models.some((entry) => entry.id === model) && model !== nextModel) setModel(nextModel);
    if (!selectedRunner.effort_levels.includes(effort)) setEffort(selectedRunner.effort_levels[0] ?? "");
  }, [selectedRunner, preferredRunner, model, effort]);

  const queue = async () => {
    if (submitting || effectiveTargets.length === 0) return;
    setSubmitting(true); setError(null);
    try {
      let last: QueueResponse | undefined;
      for (const item of effectiveTargets) {
        last = await backlogService.queue(item.kind, item.name, { mode: "yolo", startedBy: "swarm-manager-ui", confirm: true, force, ...(!isBulk ? { strategy, maxSlices, ...((preferredRunner || model || effort) ? { preferredRunner, model, effort } : {}) } : {}) });
      }
      if (last) onSuccess?.(last);
      onClose();
    } catch (cause) {
      if (isApiError(cause) && cause.code === "plan_stale" && target) {
        setStalePlanFor({ kind: target.kind, name: target.name, missingPaths: extractMissingPaths(cause.details) });
      } else {
        setError(cause instanceof Error ? cause.message : "Unable to queue this work.");
      }
    } finally { setSubmitting(false); }
  };

  return <Drawer isOpen={isOpen} onClose={onClose} title={title} description="Review the approach and guardrails before starting work." className="md:w-[520px]" testId="run-sheet" footer={<div className="flex items-center justify-between gap-3"><p className="hidden text-xs text-slate-500 sm:block">Starts immediately after you confirm.</p><div className="ml-auto flex justify-end gap-2"><Button variant="outline" onClick={onClose} disabled={submitting}>Cancel</Button><Button onClick={() => void queue()} disabled={loading || submitting || effectiveBlocked || (!isBulk && !strategy) || previews.length !== effectiveTargets.length || effectiveTargets.length === 0}>{submitting ? <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Starting…</> : <><Play className="h-4 w-4 fill-current" />Start run</>}</Button></div></div>}>
    <div className="space-y-5 p-4">
      {loading ? <div className="flex items-center gap-2 text-sm text-slate-400"><Loader2 className="h-4 w-4 animate-spin" />Checking readiness…</div> : null}
      {stalePlanFor ? <StalePlanPanel kind={stalePlanFor.kind} name={stalePlanFor.name} missingPaths={stalePlanFor.missingPaths} onReWorkshopped={onClose} onCancel={() => setStalePlanFor(null)} /> : null}
      {preflight ? <section className={`rounded-xl border p-4 ${blockingReasons.length ? "border-amber-400/30 bg-amber-400/[0.08]" : "border-emerald-400/25 bg-emerald-400/[0.06]"}`} aria-label="Preflight status"><div className="flex items-start gap-3">{blockingReasons.length ? <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-300" aria-hidden="true" /> : <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0 text-emerald-300" aria-hidden="true" />}<div className="min-w-0"><h3 className="text-sm font-semibold text-white">{blockingReasons.length ? "Needs attention before starting" : "Ready to start"}</h3>{blockingReasons.length ? <ul className="mt-2 space-y-1 text-sm leading-5 text-amber-100">{blockingReasons.map((reason) => <li key={reason.message}>{reason.message}{reason.forceable ? " (can be overridden)" : ""}</li>)}</ul> : <p className="mt-1 text-xs leading-5 text-emerald-100/80">The item passed its readiness check. Review the run approach and approved allowance below.</p>}{mayForce ? <label className="mt-3 flex items-start gap-2 text-xs leading-5 text-amber-100"><input type="checkbox" checked={force} onChange={(event) => setForce(event.target.checked)} className="mt-1" />Override eligible blockers for this run.</label> : null}</div></div></section> : null}
      {!loading && !isBulk ? <section className="rounded-xl border border-white/10 bg-slate-950/30 p-4"><div className="flex items-start gap-3"><div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-violet-400/10 text-violet-200"><Sparkles className="h-5 w-5" aria-hidden="true" /></div><div><h3 className="text-sm font-semibold text-white">Choose how the work runs</h3><p className="mt-1 text-xs leading-5 text-slate-400">These approaches use the same reviewed plan, but differ in how they move through the work.</p></div></div><div className="mt-4 space-y-3">{strategies.map((entry) => { const selected = strategy === entry.id; return <label key={entry.id} className={`group relative block cursor-pointer rounded-xl border p-4 transition-colors ${selected ? "border-cyan-300/70 bg-cyan-300/[0.10] shadow-[0_0_0_1px_rgba(103,232,249,0.12)]" : "border-white/10 bg-slate-950/30 hover:border-white/20 hover:bg-white/[0.03]"}`}><input className="sr-only" type="radio" name="execution-strategy" aria-label={entry.display_name} checked={selected} onChange={() => setStrategy(entry.id)} /><div className="flex items-start gap-3"><span className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full border ${selected ? "border-cyan-300 bg-cyan-300 text-slate-950" : "border-slate-500 text-transparent"}`} aria-hidden="true"><Check className="h-3.5 w-3.5" /></span><span className="min-w-0 flex-1"><span className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1"><span className="font-medium text-slate-100">{entry.display_name}</span><span className="rounded-full bg-slate-900/70 px-2 py-0.5 text-xs text-cyan-200">≈ ${entry.cost_estimate.toFixed(2)} est.</span></span><span className="mt-2 block text-xs leading-5 text-slate-300">{entry.description}</span><span className="mt-3 flex items-start gap-1.5 text-xs leading-5 text-slate-400"><Info className="mt-0.5 h-3.5 w-3.5 shrink-0 text-slate-500" aria-hidden="true" /><span><span className="font-medium text-slate-300">Best when:</span> {entry.when_to_use} <span className="text-slate-500">({entry.cost_band})</span></span></span></span></div>{selected ? <span className="absolute right-3 top-[-9px] rounded-full border border-cyan-300/40 bg-slate-900 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-cyan-200">Selected</span> : null}</label>; })}</div>{selectedStrategy ? <p className="mt-3 flex items-start gap-2 text-xs leading-5 text-slate-400"><Gauge className="mt-0.5 h-3.5 w-3.5 shrink-0 text-cyan-300" aria-hidden="true" />This run will use <span className="font-medium text-slate-200">{selectedStrategy.display_name}</span>. You can reduce its slice count below without changing the reviewed allowance.</p> : null}</section> : null}
      {isBulk ? <section className="rounded-xl border border-white/10 bg-slate-950/30 p-4"><div className="flex items-start gap-3"><Layers3 className="mt-0.5 h-5 w-5 shrink-0 text-cyan-300" aria-hidden="true" /><div><h3 className="text-sm font-semibold text-white">Bulk run</h3><p className="mt-1 text-xs leading-5 text-slate-400">Each of the {effectiveTargets.length} items keeps its saved execution approach and limits. Open one item to review or narrow its run.</p></div></div></section> : null}
      {!isBulk ? <section className="rounded-xl border border-white/10 bg-slate-950/30 p-4"><div className="flex items-center justify-between gap-3"><div><h3 className="text-sm font-semibold text-white">Run scope</h3><p className="mt-1 text-xs text-slate-400">Choose how many {isGoalSession ? "native-goal sessions" : "verified slices"} to attempt now.</p></div><span className="rounded-lg bg-cyan-300/10 px-3 py-1.5 text-sm font-semibold text-cyan-200">{maxSlices} {maxSlices === 1 ? (isGoalSession ? "session" : "slice") : (isGoalSession ? "sessions" : "slices")}</span></div><div className="mt-4"><Slider aria-label={isGoalSession ? "Maximum sessions" : "Maximum slices"} min={1} max={acceptedSliceLimit} step={1} value={maxSlices} showValue="none" formatValue={(value) => `${value} ${value === 1 ? (isGoalSession ? "session" : "slice") : (isGoalSession ? "sessions" : "slices")}`} onChange={setMaxSlices} /></div><div className="mt-2 flex justify-between text-[11px] text-slate-500"><span>1 {isGoalSession ? "session" : "slice"}</span><span>Reviewed maximum: {acceptedSliceLimit}</span></div><div className="mt-3 rounded-lg border border-cyan-300/10 bg-cyan-300/[0.04] p-3 text-xs leading-5 text-slate-300"><p><span className="font-semibold text-cyan-200">{isGoalSession ? "What is a slice or session?" : "What is a slice?"}</span> A slice is one bounded pass; a session is one warm native-goal pass. Each verifies progress and leaves a handoff. Reaching this cap ends the run as budget-exhausted; item continuation can start the next run automatically.</p><p className="mt-2 text-slate-400">This setting narrows the reviewed allowance for this run; it does not change the item contract. Starting smaller is a safe way to inspect progress before using the full allowance.</p></div></section> : null}
      {!isBulk ? <section className="rounded-xl border border-white/10 bg-slate-950/30 p-4" aria-label="Runner preferences"><div><h3 className="text-sm font-semibold text-white">Runner preferences</h3><p className="mt-1 text-xs leading-5 text-slate-400">Choose from Agent Manager’s live catalog. The selected values are recorded with the execution.</p></div><div className="mt-3 grid gap-3 sm:grid-cols-3"><label className="text-xs text-slate-400">Runner<Select aria-label="Preferred runner" value={preferredRunner} onChange={(event) => { setPreferredRunner(event.target.value); setModel(""); }} disabled={!availableRunners.length || loading}><option value="">Agent Manager default</option>{availableRunners.map((entry) => <option key={entry.runner_type} value={entry.runner_type}>{entry.runner_type}</option>)}</Select></label><label className="text-xs text-slate-400">Model<Select aria-label="Model" value={model} onChange={(event) => setModel(event.target.value)} disabled={!selectedRunner}><option value="">Runner default</option>{selectedRunner?.models.map((entry) => <option key={entry.id} value={entry.id}>{entry.canonical_model || entry.id}</option>)}</Select></label><label className="text-xs text-slate-400">Effort<Select aria-label="Effort" value={effort} onChange={(event) => setEffort(event.target.value)} disabled={!selectedRunner}><option value="">Runner default</option>{selectedRunner?.effort_levels.map((level) => <option key={level} value={level}>{level}</option>)}</Select></label></div>{selectedRunner?.message && <p className="mt-2 text-xs text-slate-500">{selectedRunner.message}</p>}{goalRunnerBlocked && <p role="alert" className="mt-2 text-xs text-amber-200">Goal sessions require an available native-objective runner. This run is disabled until Agent Manager reports one.</p>}</section> : null}
      {!isBulk && (strategy === "adaptive-improvement" || strategy === "goal-session") ? <section className="rounded-xl border border-amber-300/15 bg-amber-300/[0.04] p-4" aria-label="Unattended operation"><h3 className="text-sm font-semibold text-amber-100">Unattended operation</h3><p className="mt-1 text-xs leading-5 text-slate-300">These reviewed item settings control continuation after this run reaches its cap. The dialog displays them; it does not change them.</p><dl className="mt-3 grid grid-cols-2 gap-3 text-xs"><div><dt className="text-slate-500">Continuation</dt><dd className="mt-1 font-medium text-slate-200">{preflight?.item?.continuation || "manual"}</dd></div><div><dt className="text-slate-500">Scope policy</dt><dd className="mt-1 font-medium text-slate-200">{preflight?.item?.scopePolicy || "fixed"}</dd></div></dl><p className="mt-3 text-xs text-slate-400">A budget-exhausted run records its handoff; “until allowance” may start a fresh child run while the allowance and scope remain valid.</p>{itemHref ? <a className="mt-3 inline-flex text-xs font-medium text-cyan-300 hover:text-cyan-200" href={itemHref} onClick={onClose}>Edit on the item <span aria-hidden="true">→</span></a> : null}</section> : null}
      {preflight?.item?.executionLimits ? <ExecutionLimitsSummary limits={preflight.item.executionLimits} /> : null}
      {error ? <div className="rounded-lg border border-rose-400/30 bg-rose-400/10 p-3 text-sm text-rose-100">{error}</div> : null}
      {blocked && !mayForce ? <div className="flex gap-2 rounded-lg border border-amber-400/30 bg-amber-400/10 p-3 text-sm text-amber-100"><AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />Resolve the non-overridable blockers before running.</div> : null}
    </div>
  </Drawer>;
}
