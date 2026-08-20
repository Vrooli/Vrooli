import { useQuery } from "@tanstack/react-query";

import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { EmptyState } from "../../components/ui/empty-state";
import { StatusToken, TrustTriple } from "../../components/ui/instrument-status";
import { ExperienceSurface, type ExperienceSurfaceState } from "../../components/experience/ExperienceSurface";
import { fetchCondition, fetchTrust } from "../../api/reliability";

export function ConditionPage() {
  const condition = useQuery({ queryKey: ["reliability", "condition"], queryFn: fetchCondition });
  const trust = useQuery({ queryKey: ["reliability", "trust"], queryFn: fetchTrust });
  const triple = trust.data?.trust;
  const distribution = Object.fromEntries((triple?.distribution ?? []).map((item) => [trustName(item.verdict), item.count]));
  const state: ExperienceSurfaceState = condition.isLoading || trust.isLoading
    ? "loading"
    : condition.error || trust.error
      ? "error"
      : condition.data?.sources.some((source) => !source.available) ? "partial" : "ready";
  const statusMessage = state === "loading" ? "Reading trusted condition." : state === "error" ? "The condition source is unavailable." : undefined;
  return (
    <section data-testid="page-condition" aria-labelledby="condition-heading" className="flex flex-col gap-4">
      <div><p className="text-xs uppercase tracking-wide text-app-muted-foreground">Condition and trust</p><h2 id="condition-heading" className="text-2xl font-semibold">Can the reading be believed?</h2></div>
      <ExperienceSurface surfaceId="trust-distribution" state={state} data-testid="condition-trust" statusMessage={statusMessage}>
        <Card><CardHeader><CardTitle>Trust triple</CardTitle></CardHeader><CardContent><TrustTriple value={{ distribution, checked: triple?.checkedDenominator ?? 0, total: triple?.total ?? 0 }} /></CardContent></Card>
      </ExperienceSurface>
      <ExperienceSurface surfaceId="readings" state={state} data-testid="condition-readings" statusMessage={statusMessage}>
        <Card><CardHeader><CardTitle>Readings</CardTitle></CardHeader><CardContent>
          {state === "loading" ? <p role="status">Reading trusted condition…</p> : state === "error" ? <EmptyState title="Condition unavailable" description="The live condition source could not be read. No healthy reading is inferred." /> : condition.data?.readings.length ? <div className="flex flex-col gap-3">{condition.data.readings.map((reading) => <div key={reading.id} className="flex flex-wrap items-center justify-between gap-3 border-b border-app-border/60 pb-3"><div><p className="font-medium">{reading.cellRef}</p><p className="text-xs text-app-muted-foreground">{reading.value} {reading.unit} · {reading.source}</p></div><div className="flex gap-2"><StatusToken verdict={trustName(reading.trustVerdict)} /><StatusToken verdict={bandName(reading.bandVerdict)} /></div></div>)}</div> : <EmptyState title="Nothing to report" description="No condition readings are currently in the selected space." />}
        </CardContent></Card>
      </ExperienceSurface>
      <ExperienceSurface surfaceId="source-availability" state={state === "ready" ? "ready" : state} data-testid="condition-availability" statusMessage={statusMessage}>
        <Card><CardHeader><CardTitle>Source availability</CardTitle></CardHeader><CardContent>
          {condition.data?.sources.length ? <ul className="space-y-2 text-sm">{condition.data.sources.map((source) => <li key={source.source}><strong>{source.source}</strong>: {source.available ? "readable" : `unavailable — ${source.reason || "reason not supplied"}`}</li>)}</ul> : <EmptyState title="No source report" description="The condition response did not include source availability." />}
        </CardContent></Card>
      </ExperienceSurface>
      <ExperienceSurface surfaceId="history" state="empty" data-testid="condition-history">
        <Card><CardHeader><CardTitle>History</CardTitle></CardHeader><CardContent><EmptyState title="Select a cell to inspect history" description="Historical readings are re-banded against the current deadband when a cell is selected." /></CardContent></Card>
      </ExperienceSurface>
    </section>
  );
}

function trustName(value: number): "VALID" | "GHOST" | "SATURATED" | "SHELVED" | "UNIT_MISMATCH" | "UNAVAILABLE" | "UNTRUSTED" { return ["UNTRUSTED", "VALID", "GHOST", "SATURATED", "SHELVED", "UNIT_MISMATCH", "UNAVAILABLE", "UNTRUSTED"][value] as never; }
function bandName(value: number): "IN_BAND" | "OUT_OF_BAND" | "PENDING_SUSTAIN" | "NEEDS_BASELINE" | "NOT_EVALUATED" { return ["NOT_EVALUATED", "IN_BAND", "OUT_OF_BAND", "PENDING_SUSTAIN", "NEEDS_BASELINE", "NOT_EVALUATED"][value] as never; }
