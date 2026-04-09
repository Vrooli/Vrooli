/**
 * ValidationBadge — Small inline indicator for plan validation status.
 *
 * Shows a colored pill indicating whether the plan passed, has warnings, or failed validation.
 * Follows the CircuitBrokenBadge pattern.
 */

import { CheckCircle, AlertTriangle, XCircle } from "lucide-react";

interface PlanValidationResult {
  sections_present: string[];
  sections_missing: string[];
  warnings: string[];
  passed: boolean;
  validated_at: string;
}

interface ValidationBadgeProps {
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

export function ValidationBadge({ validationJson }: ValidationBadgeProps) {
  if (!validationJson) return null;

  const result = parsePlanValidationResult(validationJson);
  if (!result) {
    return null;
  }

  if (result.passed && result.warnings?.length > 0) {
    return (
      <span
        className="inline-flex items-center gap-1 rounded-full bg-yellow-500/15 px-1.5 py-0.5 text-[10px] font-medium text-yellow-400"
        title={`Plan validation passed with ${result.warnings.length} warning(s): ${result.warnings.join("; ")}`}
      >
        <AlertTriangle className="h-3 w-3" />
        Warnings
      </span>
    );
  }

  if (result.passed) {
    return (
      <span
        className="inline-flex items-center gap-1 rounded-full bg-green-500/15 px-1.5 py-0.5 text-[10px] font-medium text-green-400"
        title="Plan validation passed — all 13 mandatory sections present"
      >
        <CheckCircle className="h-3 w-3" />
        Valid plan
      </span>
    );
  }

  const missingCount = result.sections_missing?.length ?? 0;
  const warningCount = result.warnings?.length ?? 0;
  const details: string[] = [];
  if (missingCount > 0) details.push(`${missingCount} missing section(s): ${result.sections_missing.join(", ")}`);
  if (warningCount > 0) details.push(`${warningCount} warning(s): ${result.warnings.join("; ")}`);

  return (
    <span
      className="inline-flex items-center gap-1 rounded-full bg-red-500/15 px-1.5 py-0.5 text-[10px] font-medium text-red-400"
      title={`Plan validation failed — ${details.join(". ")}`}
    >
      <XCircle className="h-3 w-3" />
      Invalid plan
    </span>
  );
}
