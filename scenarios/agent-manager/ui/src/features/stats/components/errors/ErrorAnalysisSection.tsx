// Error Analysis Section - displays error patterns with links to affected runs

import { useErrorAnalysis } from "../../hooks/useErrorAnalysis";
import { formatNumber, formatRelativeTime } from "../../utils/formatters";
import { AlertTriangle, ExternalLink } from "lucide-react";
import type { ErrorPattern } from "../../api/types";

interface ErrorItemProps {
  error: ErrorPattern;
}

function ErrorItem({ error }: ErrorItemProps) {
  // Display the error code (which may be an error message or code)
  const errorDisplay = error.errorCode || "Unknown error";
  const truncatedMessage =
    errorDisplay.length > 100
      ? `${errorDisplay.slice(0, 100)}...`
      : errorDisplay;

  return (
    <div className="rounded border border-border/60 bg-card/30 p-3">
      <div className="flex items-start gap-3">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-red-500" />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-xs text-muted-foreground">
              {formatNumber(error.count)} occurrence{error.count !== 1 ? "s" : ""}
            </span>
            <span className="text-xs text-muted-foreground">
              Last: {formatRelativeTime(error.lastSeen)}
            </span>
          </div>
          <p className="text-sm text-foreground font-mono break-all" title={errorDisplay}>
            {truncatedMessage}
          </p>
          {error.sampleRunId && (
            <div className="mt-2">
              <a
                href={`#/runs/${error.sampleRunId}`}
                className="inline-flex items-center gap-1 rounded bg-muted/30 px-2 py-0.5 text-xs font-mono text-muted-foreground hover:bg-muted/50 hover:text-foreground transition-colors"
              >
                View sample run: {error.sampleRunId.slice(0, 8)}
                <ExternalLink className="h-3 w-3" />
              </a>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export function ErrorAnalysisSection() {
  const { data, isLoading, error } = useErrorAnalysis();

  if (isLoading) {
    return (
      <div className="rounded-lg border border-border/60 bg-card/50 p-6">
        <div className="mb-4 h-5 w-36 animate-pulse rounded bg-muted/30" />
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-20 animate-pulse rounded bg-muted/20" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-500/30 bg-red-500/5 p-6">
        <h3 className="text-sm font-semibold">Error Analysis</h3>
        <p className="mt-2 text-sm text-red-500">Failed to load: {error.message}</p>
      </div>
    );
  }

  const errors = data?.errors ?? [];

  return (
    <div className="rounded-lg border border-border/60 bg-card/50 p-6">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
          Error Analysis
        </h3>
        {errors.length > 0 && (
          <span className="text-xs text-muted-foreground">
            {formatNumber(errors.reduce((sum, e) => sum + e.count, 0))} total errors
          </span>
        )}
      </div>

      {errors.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-8 text-center">
          <div className="rounded-full bg-emerald-500/10 p-3 mb-3">
            <svg
              className="h-6 w-6 text-emerald-500"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          </div>
          <p className="text-sm font-medium text-foreground">No errors detected</p>
          <p className="text-xs text-muted-foreground mt-1">
            All runs completed successfully in this time period
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {errors.map((errorPattern, index) => (
            <ErrorItem key={`${errorPattern.errorCode}-${index}`} error={errorPattern} />
          ))}
        </div>
      )}
    </div>
  );
}
