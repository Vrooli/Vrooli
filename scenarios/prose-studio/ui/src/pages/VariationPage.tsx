import { useState } from "react";
import type { ReactNode } from "react";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { EmptyState } from "../components/ui/empty-state";
import { selectors } from "../consts/selectors";

export function VariationPage() {
  return <SurfaceFrame eyebrow="Generation" title="Variation Board" description="Review a spread of measured candidates. The board never orders prose by a taste score." testId={selectors.pages.variation} headingId="variation-heading"><VariationBoard /></SurfaceFrame>;
}

export function VariationBoard({
  candidates = [],
  diversity,
  loading = false,
  error,
  onReroll,
}: { candidates?: readonly string[]; diversity?: number; loading?: boolean; error?: string; onReroll?: () => void }) {
  const [rerolled, setRerolled] = useState(false);
  const hasCandidates = candidates.length > 0;
  const reroll = () => { setRerolled(true); onReroll?.(); };
  return <div className="flex flex-col gap-4">
    <Card><CardHeader><CardTitle>Candidate spread</CardTitle><CardDescription>Set diversity is reported once for the set; individual cards show measurements, never a verdict.</CardDescription></CardHeader><CardContent>
      <div className="flex items-center justify-between gap-4 rounded-control bg-app-surface-muted p-4"><div><p className="text-sm text-app-muted-foreground">Diversity basis</p><p className="font-medium">Lexical 1–3 gram similarity; deterministic</p></div><p className="text-2xl font-semibold" aria-label={diversity === undefined ? "Set diversity is not available yet" : `Set diversity ${diversity}`}>{diversity === undefined ? "—" : diversity.toFixed(2)}</p></div>
    </CardContent></Card>
    {loading && <p role="status" className="rounded-control border border-app-border p-4">Generating a measured candidate set…</p>}
    {error && <p role="alert" className="rounded-control border border-app-danger p-4">{error}</p>}
    {!loading && !error && candidates.length === 0 && <EmptyState title="No candidate set yet" description="Choose a profile and query to create the first measured round. Candidates remain visible when they fail a hard constraint." action={<Button disabled title="Generation has not started">Generate candidates</Button>} />}
    {candidates.map((candidate, index) => <Card key={`${candidate}-${index}`} tabIndex={0} aria-label={`Candidate ${index + 1}`}><CardContent><p>{candidate}</p><p className="mt-3 text-xs text-app-muted-foreground">Measurement vector is available after generation.</p></CardContent></Card>)}
    <div className="flex flex-wrap gap-3"><Button type="button" disabled={!hasCandidates || rerolled} onClick={reroll} title={!hasCandidates ? "Generate a round before rerolling" : undefined}>None of these — reroll</Button>{!hasCandidates && <span className="self-center text-sm text-app-muted-foreground">Disabled until a candidate set exists.</span>}{rerolled && <span role="status" className="self-center text-sm text-app-muted-foreground">Reroll requested; the next set will be measured independently.</span>}</div>
  </div>;
}

function SurfaceFrame({ eyebrow, title, description, testId, headingId, children }: { eyebrow: string; title: string; description: string; testId: string; headingId: string; children: ReactNode }) {
  return <section data-testid={testId} className="flex flex-col gap-6" aria-labelledby={headingId}><div className="flex flex-col gap-2"><p className="text-xs font-semibold uppercase tracking-[0.18em] text-app-primary">{eyebrow}</p><h2 id={headingId} className="text-2xl font-semibold">{title}</h2><p className="max-w-2xl text-app-muted-foreground">{description}</p></div>{children}</section>;
}

export { SurfaceFrame };
