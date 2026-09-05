import { useEffect, useMemo, useState } from "react";
import { Binoculars, RefreshCw, ShieldCheck } from "lucide-react";
import type { CohortWatch } from "@vrooli/proto-types/agent-manager/v1/domain/watch_pb";
import { WatchDisposition, WatchStatus } from "@vrooli/proto-types/agent-manager/v1/domain/watch_pb";
import { useCohortWatches, type CohortWatchActionView, type CohortWatchInspection } from "../hooks/useApi";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";

const watchStatuses: Record<number, string> = {
  [WatchStatus.ACTIVE]: "active",
  [WatchStatus.TERMINAL]: "terminal",
  [WatchStatus.CANCELED]: "canceled",
  [WatchStatus.FAILED]: "failed",
};
const dispositions: Record<number, string> = {
  [WatchDisposition.QUIET]: "quiet",
  [WatchDisposition.SIGNAL]: "signal",
  [WatchDisposition.TERMINAL]: "terminal",
  [WatchDisposition.CURSOR_RESET]: "cursor reset",
  [WatchDisposition.UNAVAILABLE]: "unavailable",
};
const actionStates: Record<number, string> = {
  1: "requested",
  2: "accepted",
  3: "applied",
  4: "rejected",
  5: "superseded",
};

function shortId(value?: string): string {
  return value ? value.slice(0, 8) : "—";
}

function actionLabel(action: CohortWatchActionView): string {
  return `${action.kind} · ${actionStates[action.state] ?? (action.status || "unknown")}`;
}

export function WatchesPage() {
  const watches = useCohortWatches();
  const inspect = watches.inspect;
  const [selectedId, setSelectedId] = useState("");
  const [detail, setDetail] = useState<CohortWatchInspection | null>(null);
  const [detailError, setDetailError] = useState<string | null>(null);

  const selected = useMemo(
    () => watches.data?.find((watch) => watch.watchId === selectedId) ?? watches.data?.[0],
    [selectedId, watches.data],
  );

  useEffect(() => {
    if (!selected) {
      setDetail(null);
      return;
    }
    setSelectedId(selected.watchId);
    setDetailError(null);
    void inspect(selected.watchId).then(setDetail).catch((error: unknown) => {
      setDetailError(error instanceof Error ? error.message : "Failed to inspect cohort watch");
    });
  }, [inspect, selected]);

  const renderedWatch: CohortWatch | undefined = detail?.inspection.watch ?? selected;

  return (
    <section className="h-full overflow-auto p-4 sm:p-6" aria-labelledby="watch-console-title">
      <div className="mx-auto max-w-7xl space-y-4">
        <header className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 id="watch-console-title" className="flex items-center gap-2 text-xl font-semibold"><Binoculars className="h-5 w-5 text-primary" /> Cohort watches</h1>
            <p className="mt-1 text-sm text-muted-foreground">Durable supervision decisions, bounded evidence, cursor health, and action delivery.</p>
          </div>
          <Button variant="outline" size="sm" onClick={() => void watches.refetch()} disabled={watches.loading}><RefreshCw className="mr-2 h-4 w-4" /> Refresh</Button>
        </header>

        <div className="flex items-center gap-2 rounded-md border border-border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
          <ShieldCheck className="h-4 w-4 shrink-0 text-success" /> Inspection is read-only and exposes bounded event metadata, never prompts or result payloads.
        </div>
        {watches.error ? <div role="alert" className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">{watches.error}</div> : null}
        {watches.loading && !watches.data?.length ? <div className="rounded-md border border-border p-8 text-center text-sm text-muted-foreground">Loading cohort watches…</div> : null}
        {!watches.loading && !watches.data?.length ? <div className="rounded-md border border-dashed border-border p-8 text-center text-sm text-muted-foreground">No cohort watches yet.</div> : null}

        {watches.data?.length ? <div className="grid min-h-[32rem] gap-4 lg:grid-cols-[minmax(18rem,0.8fr)_minmax(0,1.7fr)]">
          <div className="overflow-hidden rounded-md border border-border bg-card" aria-label="Cohort watch history">
            {watches.data.map((watch) => <Button key={watch.watchId} type="button" onClick={() => setSelectedId(watch.watchId)} aria-pressed={selected?.watchId === watch.watchId} variant="ghost" className={`h-auto block w-full rounded-none border-b border-border px-3 py-3 text-left whitespace-normal last:border-b-0 ${selected?.watchId === watch.watchId ? "bg-accent" : "hover:bg-muted/50"}`}>
              <div className="flex items-center justify-between gap-2"><span className="font-mono">{shortId(watch.watchId)}</span><Badge variant="outline">{watchStatuses[watch.status] ?? "unknown"}</Badge></div>
              <div className="mt-1 text-xs text-muted-foreground">family {shortId(watch.spec?.familyExecutionId)} · {watch.spec?.subjects.length ?? 0} subjects · r{watch.revision.toString()}</div>
            </Button>)}
          </div>

          {renderedWatch ? <article className="space-y-4 rounded-md border border-border bg-card p-4" aria-label={`Watch ${renderedWatch.watchId} details`}>
            <div><h2 className="font-semibold">Watch {shortId(renderedWatch.watchId)}</h2><p className="break-all font-mono text-xs text-muted-foreground">{renderedWatch.watchId}</p></div>
            {detailError ? <div role="alert" className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">{detailError}</div> : null}
            <dl className="grid gap-2 text-sm sm:grid-cols-3">
              <div className="rounded bg-muted/40 p-2"><dt className="text-xs text-muted-foreground">Status / revision</dt><dd>{watchStatuses[renderedWatch.status] ?? "unknown"} / {renderedWatch.revision.toString()}</dd></div>
              <div className="rounded bg-muted/40 p-2"><dt className="text-xs text-muted-foreground">Family / parent</dt><dd className="font-mono">{shortId(renderedWatch.spec?.familyExecutionId)} / {shortId(renderedWatch.spec?.parentRunId)}</dd></div>
              <div className="rounded bg-muted/40 p-2"><dt className="text-xs text-muted-foreground">Subjects / policy</dt><dd>{renderedWatch.spec?.subjects.length ?? 0} / {renderedWatch.spec?.policyVersion || "—"}</dd></div>
            </dl>
            {detail?.inspection.cursorResetRequired ? <div role="alert" className="rounded-md border border-warning/40 bg-warning/10 p-3 text-sm">Cursor reset required: {detail.inspection.resetReason || "retention changed"}</div> : null}
            <section aria-labelledby="decision-title"><h3 id="decision-title" className="mb-2 text-sm font-semibold">Last decision</h3>{renderedWatch.lastDecision ? <div className="rounded bg-muted/30 p-3 text-sm"><p>{dispositions[renderedWatch.lastDecision.disposition] ?? "unknown"} · {renderedWatch.lastDecision.classification || "unclassified"} · confidence {renderedWatch.lastDecision.confidence.toFixed(2)}</p><p className="mt-1 text-xs text-muted-foreground">{renderedWatch.lastDecision.evidenceIds.length} bounded evidence IDs · recommended action {renderedWatch.lastDecision.recommendedAction}</p></div> : <p className="text-sm text-muted-foreground">No decision recorded.</p>}</section>
            <section aria-labelledby="events-title"><h3 id="events-title" className="mb-2 text-sm font-semibold">Pending event metadata ({detail?.inspection.events.length ?? 0})</h3>{detail?.inspection.events.length ? <ol className="max-h-48 space-y-1 overflow-auto text-xs">{detail.inspection.events.map((event) => <li key={event.eventId} className="grid grid-cols-[5rem_1fr_auto] gap-2 rounded bg-muted/30 p-2"><span className="font-mono">{shortId(event.runId)}</span><span>{event.eventType}</span><span>#{event.sequence.toString()}</span></li>)}</ol> : <p className="text-sm text-muted-foreground">No unread event metadata.</p>}</section>
            <section aria-labelledby="actions-title"><h3 id="actions-title" className="mb-2 text-sm font-semibold">Action transitions ({detail?.actions.length ?? 0})</h3>{detail?.actions.length ? <ol className="max-h-48 space-y-1 overflow-auto text-xs">{detail.actions.map((action) => <li key={`${action.actionId}-${action.state}`} className="rounded bg-muted/30 p-2"><span>{actionLabel(action)}</span><span className="ml-2 font-mono text-muted-foreground">{shortId(action.targetRunId)}</span>{action.rejectionReason ? <span className="block text-destructive">{action.rejectionReason}</span> : null}</li>)}</ol> : <p className="text-sm text-muted-foreground">No actions recorded.</p>}</section>
          </article> : null}
        </div> : null}
      </div>
    </section>
  );
}
