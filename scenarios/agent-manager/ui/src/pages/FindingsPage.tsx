import { Link } from "react-router-dom";
import { Lightbulb, RefreshCw } from "lucide-react";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { DataState } from "../components/feedback/DataState";
import { useRecurringFindings } from "../hooks/useApi";

export function FindingsPage() {
  const findings = useRecurringFindings();
  const rows = findings.data ?? [];
  return (
    <section className="h-full space-y-4 overflow-auto" aria-label="Investigation findings">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-xl font-semibold sm:text-2xl">Recurring findings</h1>
          <p className="text-sm text-muted-foreground">Recommendations ordered by recurrence across completed investigations.</p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => void findings.refetch()}
          pending={findings.loading}
          aria-label="Refresh findings"
          icon={<RefreshCw aria-hidden="true" className="h-4 w-4" />}
        >
          <span className="hidden sm:inline">Refresh</span>
        </Button>
      </div>
      <DataState
        subject="Findings"
        loading={findings.loading}
        error={findings.error}
        isEmpty={rows.length === 0}
        onRetry={findings.refetch}
        empty={
          <EmptyState
            icon={<Lightbulb aria-hidden="true" />}
            title="No findings yet"
            description="Completed investigations add their recurring recommendations here."
          />
        }
      >
        {rows.map((finding) => (
          <Card key={finding.id}>
            <CardHeader>
              <CardTitle as="h2">{finding.recommendation}</CardTitle>
              <CardDescription>{finding.category} · {finding.severity} · occurrences: {finding.occurrences} · decision: {finding.decision || "unreviewed"}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <p className="break-words">{finding.evidence || "No evidence summary recorded."}</p>
              {finding.targetPath && <p className="break-all font-mono text-xs">{finding.targetPath}</p>}
              <Link className="inline-flex min-h-11 items-center text-primary underline sm:min-h-8" to={`/runs/${finding.runId}`}>Open source run</Link>
            </CardContent>
          </Card>
        ))}
      </DataState>
    </section>
  );
}
