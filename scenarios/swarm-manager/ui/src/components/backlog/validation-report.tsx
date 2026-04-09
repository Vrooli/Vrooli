/**
 * ValidationReport — Collapsible panel showing plan validation details.
 *
 * Renders sections present (green), missing (red), and warnings (yellow).
 * Used on the backlog item detail page info tab.
 */

import { useState } from "react";
import { CheckCircle, XCircle, AlertTriangle, ChevronDown, ChevronRight, ShieldCheck } from "lucide-react";

interface PlanValidationResult {
  sections_present: string[];
  sections_missing: string[];
  warnings: string[];
  passed: boolean;
  validated_at: string;
}

interface ValidationReportProps {
  validationJson: string | undefined;
}

function parsePlanValidationResult(validationJson: string): PlanValidationResult | null {
  try {
    const parsed: unknown = JSON.parse(validationJson);
    if (
      typeof parsed === "object"
      && parsed !== null
      && "passed" in parsed
      && Array.isArray((parsed as Record<string, unknown>).sections_present)
      && Array.isArray((parsed as Record<string, unknown>).sections_missing)
      && Array.isArray((parsed as Record<string, unknown>).warnings)
      && typeof (parsed as Record<string, unknown>).passed === "boolean"
      && typeof (parsed as Record<string, unknown>).validated_at === "string"
    ) {
      return parsed as PlanValidationResult;
    }
  } catch {
    return null;
  }
  return null;
}

export function ValidationReport({ validationJson }: ValidationReportProps) {
  const [expanded, setExpanded] = useState(false);

  if (!validationJson) return null;

  const result = parsePlanValidationResult(validationJson);
  if (!result) {
    return null;
  }

  const hasSections = (result.sections_present?.length ?? 0) > 0 || (result.sections_missing?.length ?? 0) > 0;
  if (!hasSections && (result.warnings?.length ?? 0) === 0) return null;

  const statusColor = result.passed ? "text-green-400" : "text-red-400";
  const statusLabel = result.passed ? "Validation Passed" : "Validation Failed";

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/50 px-4 py-3">
      <button
        type="button"
        className="flex w-full items-center gap-2 text-left"
        onClick={() => setExpanded(!expanded)}
      >
        <ShieldCheck className={`h-4 w-4 ${statusColor}`} />
        <span className={`text-sm font-medium ${statusColor}`}>{statusLabel}</span>
        <span className="ml-auto text-slate-500">
          {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        </span>
      </button>

      {expanded && (
        <div className="mt-3 space-y-2 text-sm">
          {result.sections_present?.length > 0 && (
            <div>
              <span className="text-xs font-medium uppercase tracking-wider text-slate-500">Present</span>
              <ul className="mt-1 space-y-0.5">
                {result.sections_present.map((s) => (
                  <li key={s} className="flex items-center gap-1.5 text-green-400/80">
                    <CheckCircle className="h-3 w-3 shrink-0" />
                    {s}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {result.sections_missing?.length > 0 && (
            <div>
              <span className="text-xs font-medium uppercase tracking-wider text-slate-500">Missing</span>
              <ul className="mt-1 space-y-0.5">
                {result.sections_missing.map((s) => (
                  <li key={s} className="flex items-center gap-1.5 text-red-400">
                    <XCircle className="h-3 w-3 shrink-0" />
                    {s}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {result.warnings?.length > 0 && (
            <div>
              <span className="text-xs font-medium uppercase tracking-wider text-slate-500">Warnings</span>
              <ul className="mt-1 space-y-0.5">
                {result.warnings.map((w) => (
                  <li key={w} className="flex items-center gap-1.5 text-yellow-400">
                    <AlertTriangle className="h-3 w-3 shrink-0" />
                    {w}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {result.validated_at && (
            <p className="text-xs text-slate-600">
              Validated: {new Date(result.validated_at).toLocaleString()}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
