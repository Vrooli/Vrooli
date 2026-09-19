import { useEffect, useState } from "react";
import { API_BASE } from "../api/client";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";

type Take = { id: string; blob_ref?: string; provenance?: { seed?: number; applied_rung?: string } };
export function AuditionPage() {
  const [takes, setTakes] = useState<Take[]>([]);
  useEffect(() => { void fetch(`${API_BASE}/v1/takes`).then(r => r.ok ? r.json() : { takes: [] }).then((body: { takes?: Take[] } | Take[]) => setTakes(Array.isArray(body) ? body : body.takes ?? [])); }, []);
  return <section className="flex flex-col gap-4" aria-labelledby="audition-heading"><h2 id="audition-heading" className="text-2xl font-semibold">Audition</h2><p className="text-app-muted-foreground">Compare peer takes without ranking them for the operator.</p><div className="grid gap-3">{takes.map(take => <Card key={take.id} data-testid="take-row"><CardHeader><CardTitle className="font-mono text-base">{take.id}</CardTitle></CardHeader><CardContent><audio data-testid="take-play" controls aria-label={`Play ${take.id}`} src={`${API_BASE}/v1/takes/${take.id}/audio`} /><div data-testid="take-waveform" aria-label="Waveform" className="h-8" /><div data-testid="take-provenance" className="text-sm text-app-muted-foreground">Seed {take.provenance?.seed ?? "—"}</div><span data-testid="take-rung-badge" role="status">rung {take.provenance?.applied_rung ?? "—"}</span><button className="touch-target" data-testid="take-keep" type="button">Keep</button><button className="touch-target" data-testid="take-discard" type="button">Discard</button></CardContent></Card>)}<p data-testid="batch-partial-notice" role="status">Completed takes remain usable if a batch ends early.</p></div></section>;
}
