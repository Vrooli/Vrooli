import { Link } from "react-router-dom";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { useRecurringFindings } from "../hooks/useApi";

export function FindingsPage() {
  const findings = useRecurringFindings();
  return (
    <section className="h-full overflow-auto p-6 space-y-4" aria-label="Investigation findings">
      <div className="flex items-center justify-between gap-3">
        <div><h1 className="text-2xl font-semibold">Recurring findings</h1><p className="text-sm text-muted-foreground">Recommendations ordered by recurrence across completed investigations.</p></div>
        <Button variant="outline" onClick={() => findings.refetch()} disabled={findings.loading}>Refresh</Button>
      </div>
      {findings.error && <p role="alert" className="text-sm text-destructive">{findings.error}</p>}
      {!findings.loading && findings.data?.length === 0 && <Card><CardContent className="p-6 text-sm text-muted-foreground">No persisted investigation findings yet.</CardContent></Card>}
      {findings.data?.map((finding) => (
        <Card key={finding.id}>
          <CardHeader className="pb-3"><CardTitle className="text-base">{finding.recommendation}</CardTitle><CardDescription>{finding.category} · {finding.severity} · occurrences: {finding.occurrences} · decision: {finding.decision || "unreviewed"}</CardDescription></CardHeader>
          <CardContent className="space-y-2 text-sm"><p>{finding.evidence || "No evidence summary recorded."}</p>{finding.targetPath && <p className="font-mono text-xs">{finding.targetPath}</p>}<Link className="text-primary underline" to={`/runs/${finding.runId}`}>Open source run</Link></CardContent>
        </Card>
      ))}
    </section>
  );
}
