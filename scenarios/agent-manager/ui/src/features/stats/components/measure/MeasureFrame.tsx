import { Info, MapPin } from "lucide-react";
import type { ReactNode } from "react";
import { InsufficientDataCard } from "../../../../components/stats/InsufficientDataCard";
import type { MeasureDefinitionView, MeasureResponse } from "../../api/statsClient";

interface MeasureFrameProps {
  label: string;
  result?: MeasureResponse;
  definition?: MeasureDefinitionView;
  loading?: boolean;
  error?: string;
  children: ReactNode;
  testId?: string;
}

export function MeasureFrame({ label, result, definition, loading, error, children, testId }: MeasureFrameProps) {
  if (loading) {
    return <div className="animate-pulse rounded-lg border border-border/60 bg-card/40 p-3" data-testid={testId}><div className="h-3 w-24 rounded bg-muted/40" /><div className="mt-2 h-7 w-32 rounded bg-muted/40" /></div>;
  }
  if (error) {
    return <div className="rounded-lg border border-red-500/30 bg-red-500/5 p-3 text-sm text-red-500" role="alert" data-testid={testId}>{label}: {error}</div>;
  }
  if (!result || !result.validity || result.validity.state === "unavailable") {
    return <InsufficientDataCard label={label} reason={result?.validity.reason || "the measure did not return usable evidence"} testId={testId} />;
  }
  if (result.validity.state === "unreliable") {
    return <InsufficientDataCard label={label} reason={result.validity.reason} have={result.validity.sampleSize} testId={testId} />;
  }

  return (
    <div data-testid={testId}>
      {children}
      <div className="mt-2 flex flex-wrap gap-2 text-xs text-muted-foreground">
        {definition && (
          <details>
            <summary className="inline-flex cursor-pointer list-none items-center gap-1 rounded px-1 py-0.5 hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" aria-label={`Definition for ${label}`}>
              <Info className="h-3.5 w-3.5" /> Definition
            </summary>
            <div className="mt-2 max-w-md rounded border border-border/60 bg-card p-3 shadow-sm">
              <p><span className="font-medium text-foreground">Counts:</span> {definition.counts}</p>
              <p><span className="font-medium text-foreground">Numerator:</span> {definition.numerator}</p>
              <p><span className="font-medium text-foreground">Denominator:</span> {definition.denominator}</p>
              <p><span className="font-medium text-foreground">Source:</span> {definition.sourceTable}</p>
              {definition.limitation && <p><span className="font-medium text-foreground">Limitation:</span> {definition.limitation}</p>}
            </div>
          </details>
        )}
        {result.provenance && (
          <details>
            <summary className="inline-flex cursor-pointer list-none items-center gap-1 rounded px-1 py-0.5 hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" aria-label={`Provenance for ${label}`}>
              <MapPin className="h-3.5 w-3.5" /> Evidence
            </summary>
            <div className="mt-2 max-w-md rounded border border-border/60 bg-card p-3 shadow-sm">
              <p><span className="font-medium text-foreground">Source:</span> {result.provenance.sourceTable}</p>
              <p><span className="font-medium text-foreground">Window:</span> {result.provenance.windowStart} – {result.provenance.windowEnd}</p>
              <p><span className="font-medium text-foreground">Rows:</span> {result.provenance.rowCount}</p>
              {result.provenance.appliedFilters.length > 0 && <p><span className="font-medium text-foreground">Filters:</span> {result.provenance.appliedFilters.map((filter) => `${filter.field}=${filter.value}`).join(", ")}</p>}
            </div>
          </details>
        )}
      </div>
    </div>
  );
}
