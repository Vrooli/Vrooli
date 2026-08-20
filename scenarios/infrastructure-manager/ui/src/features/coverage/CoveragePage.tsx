import { useQuery } from "@tanstack/react-query";

import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { EmptyState } from "../../components/ui/empty-state";
import { RatioConfidence } from "../../components/ui/instrument-status";
import { ExperienceSurface, type ExperienceSurfaceState } from "../../components/experience/ExperienceSurface";
import { fetchCells, fetchCoverage } from "../../api/reliability";
import { Projection } from "@vrooli/proto-types/infrastructure-manager/v1/coverage/coverage_pb";

export function CoveragePage() {
  const coverage = useQuery({ queryKey: ["reliability", "coverage"], queryFn: fetchCoverage });
  const cells = useQuery({ queryKey: ["reliability", "cells"], queryFn: fetchCells });

  const projections = coverage.data?.projections ?? [];
  const missing = (cells.data?.cells ?? []).filter((cell) => cell.status === 3);
  const state: ExperienceSurfaceState = coverage.isLoading || cells.isLoading ? "loading" : coverage.error || cells.error ? "error" : "ready";
  const statusMessage = state === "loading" ? "Reading coverage spaces." : state === "error" ? "The coverage source is unavailable." : undefined;
  return (
    <section data-testid="page-coverage" aria-labelledby="coverage-heading" className="flex flex-col gap-4">
      <div>
        <p className="text-xs uppercase tracking-wide text-app-muted-foreground">Instrument coverage</p>
        <h2 id="coverage-heading" className="text-2xl font-semibold">What is actually instrumented?</h2>
        <p className="text-sm text-app-muted-foreground">Ratios stay adjacent to the confidence of the owner-authored denominator.</p>
      </div>
      <ExperienceSurface surfaceId="cell-grid" state={state} data-testid="coverage-grid" statusMessage={statusMessage}>
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {state === "loading" ? <p role="status">Reading coverage spaces…</p> : state === "error" ? <EmptyState title="Coverage unavailable" description="The coverage source could not be read. The instrument has named the failure instead of fabricating a ratio." /> : projections.map((projection) => (
            <Card key={projection.projection}>
              <CardHeader><CardTitle>{projectionName(projection.projection)}</CardTitle></CardHeader>
              <CardContent><RatioConfidence value={{ ratio: projection.available ? (projection.ratio?.value ?? null) : null, confidence: confidenceName(projection.confidence?.level), rationale: projection.confidence?.rationale || projection.unavailableReason || "No denominator rationale supplied." }} /><p className="mt-3 text-xs text-app-muted-foreground">NOW {projection.nowCount} · IN-REACH {projection.inReachCount} · MISSING {projection.missingCount}</p></CardContent>
            </Card>
          ))}
        </div>
      </ExperienceSurface>
      <ExperienceSurface surfaceId="confidence" state={state} data-testid="coverage-confidence" statusMessage={statusMessage}>
        <Card><CardHeader><CardTitle>Denominator confidence</CardTitle></CardHeader><CardContent><p className="text-sm text-app-muted-foreground">Every ratio above is accompanied by the owner-authored confidence and rationale of its denominator.</p></CardContent></Card>
      </ExperienceSurface>
      <ExperienceSurface surfaceId="open-loop" state={state} data-testid="coverage-open-loop" statusMessage={statusMessage}>
        <Card><CardHeader><CardTitle>Open loop ({missing.length})</CardTitle></CardHeader><CardContent>{state === "loading" ? <p role="status">Reading dated gaps…</p> : state === "error" ? <p className="text-sm">Open-loop cells are unavailable because the coverage source failed.</p> : missing.length === 0 ? <p className="text-sm">No dated coverage gaps.</p> : <div className="overflow-x-auto"><table className="w-full text-left text-sm"><thead><tr className="border-b border-app-border"><th className="py-2 pr-3">Cell</th><th className="py-2 pr-3">Question</th><th className="py-2 pr-3">Opened</th><th className="py-2">Age</th></tr></thead><tbody>{missing.map((cell) => <tr key={`${cell.projection}-${cell.id}`} className="border-b border-app-border/60"><td className="py-2 pr-3 font-medium">{projectionName(cell.projection)}/{cell.id}</td><td className="py-2 pr-3">{cell.question}</td><td className="py-2 pr-3">{cell.gapOpenedOn || "—"}</td><td className="py-2">{cell.gapOpenDays}d</td></tr>)}</tbody></table></div>}</CardContent></Card>
      </ExperienceSurface>
      <ExperienceSurface surfaceId="integrity" state={state} data-testid="coverage-integrity" statusMessage={statusMessage}>
        <Card><CardHeader><CardTitle>Integrity findings</CardTitle></CardHeader><CardContent>{coverage.data?.integrityFindings.length ? <ul className="space-y-2 text-sm">{coverage.data.integrityFindings.map((finding) => <li key={`${finding.code}-${finding.location}`}><strong>{finding.code}</strong>: {finding.message}</li>)}</ul> : <p className="text-sm">No setpoint-integrity findings were returned.</p>}</CardContent></Card>
      </ExperienceSurface>
    </section>
  );
}

function projectionName(value: number): string { return (Projection[value] ?? "UNKNOWN").replace("PROJECTION_", "").toLowerCase().replace(/_/g, "-"); }
function confidenceName(value: number | undefined): "AUTHORITATIVE" | "PARTIAL" | "SKETCH" { return value === 1 ? "AUTHORITATIVE" : value === 2 ? "PARTIAL" : "SKETCH"; }
