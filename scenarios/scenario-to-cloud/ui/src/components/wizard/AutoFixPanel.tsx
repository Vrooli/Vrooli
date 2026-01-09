import { useMemo } from "react";
import { Wand2, Check } from "lucide-react";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";
import type { ValidationIssue } from "../../lib/api";
import type { DeploymentManifest } from "../../types/deployment";

interface AutoFixPanelProps {
  issues: ValidationIssue[] | null;
  normalizedManifest: DeploymentManifest | null;
  currentManifest: DeploymentManifest | null;
  onApplyAll: () => void;
  className?: string;
}

/**
 * Helper to get a value from an object by dot-notation path.
 */
function getValueAtPath(obj: unknown, path: string): unknown {
  const parts = path.split(".");
  let current: unknown = obj;
  for (const part of parts) {
    if (current === null || current === undefined) return undefined;
    if (typeof current !== "object") return undefined;
    current = (current as Record<string, unknown>)[part];
  }
  return current;
}

/**
 * Format a value for display (handles objects, arrays, primitives).
 */
function formatValue(value: unknown): string {
  if (value === undefined) return "undefined";
  if (value === null) return "null";
  if (typeof value === "string") return `"${value}"`;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  if (Array.isArray(value)) return `[${value.length} items]`;
  if (typeof value === "object") return "{...}";
  return String(value);
}

interface FixableIssue extends ValidationIssue {
  currentValue: unknown;
  fixedValue: unknown;
}

/**
 * Panel showing auto-fix actions for validation issues.
 * Compares current manifest with normalized manifest to detect fixable issues.
 */
export function AutoFixPanel({
  issues,
  normalizedManifest,
  currentManifest,
  onApplyAll,
  className,
}: AutoFixPanelProps) {
  // Determine which issues are fixable by comparing current vs normalized values
  const fixableIssues = useMemo(() => {
    if (!issues || !normalizedManifest || !currentManifest) return [];

    const fixable: FixableIssue[] = [];

    for (const issue of issues) {
      const currentValue = getValueAtPath(currentManifest, issue.path);
      const fixedValue = getValueAtPath(normalizedManifest, issue.path);

      // Check if values are different (issue can be fixed)
      const isDifferent = JSON.stringify(currentValue) !== JSON.stringify(fixedValue);

      if (isDifferent) {
        fixable.push({
          ...issue,
          currentValue,
          fixedValue,
        });
      }
    }

    return fixable;
  }, [issues, normalizedManifest, currentManifest]);

  // Don't render if nothing is fixable
  if (fixableIssues.length === 0) {
    return null;
  }

  return (
    <div
      className={cn(
        "rounded-lg border border-blue-500/30 bg-blue-500/5 p-4",
        className
      )}
    >
      <div className="flex items-center justify-between gap-4 mb-3">
        <div className="flex items-center gap-2">
          <Wand2 className="h-4 w-4 text-blue-400" />
          <span className="text-sm font-medium text-blue-300">
            {fixableIssues.length} issue{fixableIssues.length !== 1 ? "s" : ""} can be auto-fixed
          </span>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={onApplyAll}
          className="border-blue-500/50 text-blue-300 hover:bg-blue-500/10"
        >
          <Check className="h-3.5 w-3.5 mr-1.5" />
          Apply All Fixes
        </Button>
      </div>

      {/* List of fixable issues with before/after preview */}
      <div className="space-y-2">
        {fixableIssues.map((issue, index) => (
          <div
            key={`${issue.path}-${index}`}
            className="flex items-start gap-3 text-sm p-2 rounded bg-slate-900/50"
          >
            <div className="flex-1 min-w-0">
              <code className="text-xs text-slate-400 font-mono">{issue.path}</code>
              <div className="mt-1 flex items-center gap-2 text-xs">
                <span className="text-slate-500">
                  {formatValue(issue.currentValue)}
                </span>
                <span className="text-slate-600">→</span>
                <span className="text-blue-300 font-medium">
                  {formatValue(issue.fixedValue)}
                </span>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
