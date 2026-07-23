import { useEffect, useMemo, useState } from "react";
import { AlertTriangle, Loader2 } from "lucide-react";
import { Drawer } from "../ui/drawer";
import { Button } from "../ui/button";
import { backlogService } from "../../services";
import type { QueueResponse } from "../../services";
import type { BacklogKind } from "../../types";
import { defaultApiClient, isApiError } from "../../lib/api-client";
import { API_ENDPOINTS } from "../../lib/api-endpoints";
import { StalePlanPanel } from "./stale-plan-panel";
import { extractMissingPaths, type MissingPath } from "./stale-plan-utils";

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
  const [strategies, setStrategies] = useState<ExecutionStrategy[]>([]);
  const [strategy, setStrategy] = useState("");
  const [maxSlices, setMaxSlices] = useState(6);
  const [force, setForce] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [stalePlanFor, setStalePlanFor] = useState<{ kind: BacklogKind; name: string; missingPaths: MissingPath[] } | null>(null);

  useEffect(() => {
    if (!isOpen) return;
    setError(null); setForce(false); setStalePlanFor(null); setPreflight(null); setLoading(true);
    const first = effectiveTargets[0];
    void Promise.all([
      defaultApiClient.get<{ items: ExecutionStrategy[] }>(API_ENDPOINTS.executionStrategies),
      first ? backlogService.queue(first.kind, first.name, { mode: "yolo", confirm: false }) : Promise.resolve(null),
    ]).then(([strategyResponse, preview]) => {
      setStrategies(strategyResponse.items ?? []);
      setStrategy((current) => current || strategyResponse.items?.[0]?.id || "");
      setPreflight(preview);
    }).catch((cause) => setError(cause instanceof Error ? cause.message : "Unable to load run options.")).finally(() => setLoading(false));
  }, [effectiveTargets, isOpen]);

  const mayForce = Boolean(preflight?.blockingReasons.length) && preflight?.blockingReasons.every((reason) => reason.forceable);
  const blocked = Boolean(preflight?.blockingReasons.length) && !force;
  const title = isBulk ? `Run ${effectiveTargets.length} items` : target?.title ? `Run “${target.title}”` : "Run backlog item";

  const queue = async () => {
    if (submitting || effectiveTargets.length === 0) return;
    setSubmitting(true); setError(null);
    try {
      let last: QueueResponse | undefined;
      for (const item of effectiveTargets) {
        last = await backlogService.queue(item.kind, item.name, { mode: "yolo", startedBy: "swarm-manager-ui", confirm: true, force, strategy, maxSlices });
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

  return <Drawer isOpen={isOpen} onClose={onClose} title={title} description="Review readiness, execution approach, and scope before work is queued." testId="run-sheet" footer={<div className="flex justify-end gap-2"><Button variant="outline" onClick={onClose} disabled={submitting}>Cancel</Button><Button onClick={() => void queue()} disabled={loading || submitting || blocked || !strategy || effectiveTargets.length === 0}>{submitting ? <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Queueing…</> : "Run"}</Button></div>}>
    <div className="space-y-5 p-4">
      {loading ? <div className="flex items-center gap-2 text-sm text-slate-400"><Loader2 className="h-4 w-4 animate-spin" />Checking readiness…</div> : null}
      {stalePlanFor ? <StalePlanPanel kind={stalePlanFor.kind} name={stalePlanFor.name} missingPaths={stalePlanFor.missingPaths} onReWorkshopped={onClose} onCancel={() => setStalePlanFor(null)} /> : null}
      {preflight ? <section className="rounded-lg border border-white/10 bg-slate-950/40 p-3"><h3 className="text-sm font-semibold text-white">Preflight</h3>{preflight.blockingReasons.length ? <ul className="mt-2 space-y-1 text-sm text-amber-100">{preflight.blockingReasons.map((reason) => <li key={reason.message}>• {reason.message}{reason.forceable ? " (overridable)" : ""}</li>)}</ul> : <p className="mt-1 text-sm text-emerald-200">Ready to queue.</p>}{mayForce ? <label className="mt-3 flex items-start gap-2 text-sm text-amber-100"><input type="checkbox" checked={force} onChange={(event) => setForce(event.target.checked)} className="mt-1" />Override eligible preflight blockers for this run.</label> : null}</section> : null}
      <section><h3 className="text-sm font-semibold text-white">Execution approach</h3><div className="mt-2 space-y-2">{strategies.map((entry) => <label key={entry.id} className={`block cursor-pointer rounded-lg border p-3 ${strategy === entry.id ? "border-cyan-400/50 bg-cyan-400/10" : "border-white/10 bg-slate-950/30"}`}><input className="sr-only" type="radio" checked={strategy === entry.id} onChange={() => setStrategy(entry.id)} /><span className="flex items-center justify-between gap-3"><span className="font-medium text-slate-100">{entry.display_name}</span><span className="text-xs text-cyan-200">≈ ${entry.cost_estimate.toFixed(2)}</span></span><span className="mt-1 block text-xs leading-5 text-slate-300">{entry.description}</span><span className="mt-1 block text-xs text-slate-500">{entry.when_to_use} {entry.cost_band}</span></label>)}</div></section>
      <label className="block text-sm font-medium text-slate-100">Maximum slices <span className="ml-1 font-normal text-slate-400">({maxSlices})</span><input aria-label="Maximum slices" className="mt-3 block w-full accent-cyan-400" type="range" min="1" max="6" value={maxSlices} onChange={(event) => setMaxSlices(Number(event.target.value))} /></label>
      {error ? <div className="rounded-lg border border-rose-400/30 bg-rose-400/10 p-3 text-sm text-rose-100">{error}</div> : null}
      {blocked && !mayForce ? <div className="flex gap-2 rounded-lg border border-amber-400/30 bg-amber-400/10 p-3 text-sm text-amber-100"><AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />Resolve the non-overridable blockers before running.</div> : null}
    </div>
  </Drawer>;
}
