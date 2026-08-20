import { useQuery } from "@tanstack/react-query";

import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { EmptyState } from "../../components/ui/empty-state";
import { ExperienceSurface, type ExperienceSurfaceState } from "../../components/experience/ExperienceSurface";
import { fetchFocus } from "../../api/reliability";

export function FocusPage() {
  const focus = useQuery({ queryKey: ["reliability", "focus"], queryFn: fetchFocus });
  const data = focus.data;
  const state: ExperienceSurfaceState = focus.isLoading ? "loading" : focus.error ? "error" : data?.allSourcesUnavailable ? "partial" : data?.noFindings ? "empty" : "ready";
  const statusMessage = state === "loading" ? "Ranking next findings." : state === "error" ? "The focus sources are unavailable." : undefined;
  return (
    <section data-testid="page-focus" aria-labelledby="focus-heading" className="flex flex-col gap-4">
      <div><p className="text-xs uppercase tracking-wide text-app-muted-foreground">Ranked error surface</p><h2 id="focus-heading" className="text-2xl font-semibold">What should happen next?</h2><p className="text-sm text-app-muted-foreground">Cascade order is visible beside every finding; this page offers no actuation.</p></div>
      <ExperienceSurface surfaceId="ranking-rationale" state="static" data-testid="focus-rationale"><Card><CardHeader><CardTitle>Ranking rationale</CardTitle></CardHeader><CardContent><p className="text-sm">Findings are ordered by the reliability cascade, with instrument integrity ahead of plant condition. This surface is read-only.</p></CardContent></Card></ExperienceSurface>
      <ExperienceSurface surfaceId="ranked-surface" state={state} data-testid="focus-surface" statusMessage={statusMessage}><Card><CardHeader><CardTitle>Next findings</CardTitle></CardHeader><CardContent>{state === "loading" ? <p role="status">Ranking next findings…</p> : state === "error" ? <EmptyState title="Focus unavailable" description="The instrument could not read its finding sources." /> : !data || data.allSourcesUnavailable ? <EmptyState title="Nothing could be read" description={data?.sources.map((source) => `${source.id}: ${source.reason || "unavailable"}`).join(" · ") || "The focus response was unavailable."} /> : data.noFindings ? <EmptyState title="Nothing to report" description="All readable sources currently have no ranked findings." /> : <div className="flex flex-col gap-3">{data.findings.map((finding) => <article key={finding.id} className="border-b border-app-border/60 pb-3"><div className="flex flex-wrap items-center gap-2"><span className="font-mono text-xs">#{finding.rationale?.rank ?? "—"}</span><h3 className="font-medium">{finding.title}</h3><span className="text-xs text-app-muted-foreground">{finding.source}</span></div><p className="mt-1 text-sm text-app-muted-foreground">{finding.message}</p><p className="mt-1 text-xs">{finding.rationale?.cascadeStage}: {finding.rationale?.explanation}</p></article>)}</div>}</CardContent></Card></ExperienceSurface>
      <ExperienceSurface surfaceId="source-health" state={state === "ready" ? "ready" : state} data-testid="focus-sources" statusMessage={statusMessage}><Card><CardHeader><CardTitle>Sources</CardTitle></CardHeader><CardContent>{data?.sources.length ? <ul className="space-y-2 text-sm">{data.sources.map((source) => <li key={source.id}><strong>{source.label}</strong>: {source.available ? `${source.findingCount} finding(s)` : `unavailable — ${source.reason}`}</li>)}</ul> : <p className="text-sm">No source availability response yet.</p>}</CardContent></Card></ExperienceSurface>
      <ExperienceSurface surfaceId="efficacy" state="empty" data-testid="focus-efficacy"><Card><CardHeader><CardTitle>Actuation efficacy</CardTitle></CardHeader><CardContent><EmptyState title="No completed work to grade" description="A finding's named sensor is re-read only after downstream work records a completion." /></CardContent></Card></ExperienceSurface>
    </section>
  );
}
