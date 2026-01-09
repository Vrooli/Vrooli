import { useState } from "react";
import { AlertCircle, AlertTriangle, CheckCircle2, ChevronDown, ChevronRight, Code } from "lucide-react";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";
import type { ValidationIssue } from "../../lib/api";

interface ValidationSummaryProps {
  issues: ValidationIssue[] | null;
  error: string | null;
  isValidating: boolean;
  onViewInJson?: () => void;
  className?: string;
}

/**
 * Compact validation status display for Form mode.
 * Shows error/warning counts with expandable issue list.
 */
export function ValidationSummary({
  issues,
  error,
  isValidating,
  onViewInJson,
  className,
}: ValidationSummaryProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  // Calculate counts
  const errorCount = issues?.filter((i) => i.severity === "error").length ?? 0;
  const warnCount = issues?.filter((i) => i.severity === "warn").length ?? 0;
  const hasIssues = errorCount > 0 || warnCount > 0;
  const isValid = issues !== null && errorCount === 0;

  // Don't render if no validation has occurred yet and not currently validating
  if (issues === null && !isValidating && !error) {
    return null;
  }

  return (
    <div
      className={cn(
        "rounded-lg border p-3",
        error && "border-red-500/30 bg-red-500/5",
        !error && errorCount > 0 && "border-red-500/30 bg-red-500/5",
        !error && errorCount === 0 && warnCount > 0 && "border-amber-500/30 bg-amber-500/5",
        !error && isValid && warnCount === 0 && "border-green-500/30 bg-green-500/5",
        isValidating && "border-slate-500/30 bg-slate-500/5",
        className
      )}
    >
      <div className="flex items-center justify-between gap-3">
        {/* Status indicator */}
        <div className="flex items-center gap-2 min-w-0">
          {isValidating ? (
            <>
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-slate-500 border-t-transparent flex-shrink-0" />
              <span className="text-sm text-slate-400">Validating...</span>
            </>
          ) : error ? (
            <>
              <AlertCircle className="h-4 w-4 text-red-400 flex-shrink-0" />
              <span className="text-sm text-red-400 truncate">{error}</span>
            </>
          ) : errorCount > 0 ? (
            <>
              <AlertCircle className="h-4 w-4 text-red-400 flex-shrink-0" />
              <span className="text-sm text-red-400">
                {errorCount} error{errorCount !== 1 ? "s" : ""}
                {warnCount > 0 && `, ${warnCount} warning${warnCount !== 1 ? "s" : ""}`}
              </span>
            </>
          ) : warnCount > 0 ? (
            <>
              <AlertTriangle className="h-4 w-4 text-amber-400 flex-shrink-0" />
              <span className="text-sm text-amber-400">
                {warnCount} warning{warnCount !== 1 ? "s" : ""}
              </span>
            </>
          ) : (
            <>
              <CheckCircle2 className="h-4 w-4 text-green-400 flex-shrink-0" />
              <span className="text-sm text-green-400">Manifest is valid</span>
            </>
          )}
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2 flex-shrink-0">
          {hasIssues && (
            <button
              onClick={() => setIsExpanded(!isExpanded)}
              className="flex items-center gap-1 text-xs text-slate-400 hover:text-slate-200 transition-colors"
            >
              {isExpanded ? (
                <ChevronDown className="h-3.5 w-3.5" />
              ) : (
                <ChevronRight className="h-3.5 w-3.5" />
              )}
              {isExpanded ? "Hide" : "Show"} details
            </button>
          )}
          {onViewInJson && (hasIssues || error) && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onViewInJson}
              className="text-xs h-7 px-2"
            >
              <Code className="h-3 w-3 mr-1" />
              View in JSON
            </Button>
          )}
        </div>
      </div>

      {/* Expanded issue list */}
      {isExpanded && issues && issues.length > 0 && (
        <div className="mt-3 space-y-2 border-t border-slate-700/50 pt-3">
          {issues.map((issue, index) => (
            <div
              key={`${issue.severity}-${issue.path}-${index}`}
              className="flex items-start gap-2 text-sm"
            >
              {issue.severity === "error" ? (
                <AlertCircle className="h-3.5 w-3.5 text-red-400 flex-shrink-0 mt-0.5" />
              ) : (
                <AlertTriangle className="h-3.5 w-3.5 text-amber-400 flex-shrink-0 mt-0.5" />
              )}
              <div className="min-w-0">
                <code className="text-xs text-slate-500 font-mono">{issue.path}</code>
                <p className="text-slate-300">{issue.message}</p>
                {issue.hint && (
                  <p className="text-xs text-slate-500 mt-0.5">{issue.hint}</p>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
