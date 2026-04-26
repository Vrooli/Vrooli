/**
 * StalePlanPanel — surfaced when a queue/spawn attempt fails with a
 * `plan_stale` error. The plan was authored against an older repo state
 * and one or more `acceptance_allow` globs reference paths that no longer
 * exist on disk and are not declared in `creates`. The fix is not to
 * patch the globs in isolation (the plan body itself reasons about the
 * stale paths) but to re-author the plan against the current repo —
 * which is what the "Re-workshop" button kicks off.
 */

import { useState } from "react";
import { Loader2, AlertTriangle } from "lucide-react";
import { Button } from "../ui/button";
import { backlogService } from "../../services";
import type { BacklogKind } from "../../types";
import type { MissingPath } from "./stale-plan-utils";

export interface StalePlanPanelProps {
  kind: BacklogKind;
  name: string;
  missingPaths: MissingPath[];
  /** Called after the re-workshop trigger completes successfully. */
  onReWorkshopped?: () => void;
  /** Called when the user dismisses the panel without re-workshopping. */
  onCancel?: () => void;
}

export function StalePlanPanel({
  kind,
  name,
  missingPaths,
  onReWorkshopped,
  onCancel,
}: StalePlanPanelProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleReWorkshop = async () => {
    setIsSubmitting(true);
    setError(null);
    try {
      await backlogService.reWorkshop(kind, name);
      onReWorkshopped?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to re-workshop item.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div
      className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-200"
      data-testid="stale-plan-panel"
    >
      <div className="mb-2 flex items-center gap-2 font-medium">
        <AlertTriangle className="h-4 w-4 text-amber-400" />
        This plan references paths that no longer exist.
      </div>
      <p className="mb-2 text-amber-200/90">
        The plan was likely written against an earlier repo state. Re-workshopping
        rewrites the plan and acceptance against the current repo.
      </p>
      {missingPaths.length > 0 && (
        <ul className="mb-3 list-disc pl-5 space-y-1">
          {missingPaths.map((mp, i) => (
            <li key={`${mp.glob}-${i}`} data-testid="stale-plan-missing-path">
              <code className="font-mono text-amber-100">{mp.glob}</code>
              {mp.resolved ? (
                <>
                  {" "}
                  <span className="text-amber-300">→</span>{" "}
                  <code className="font-mono text-amber-100">{mp.resolved}</code>
                </>
              ) : null}
              {mp.reason ? (
                <span className="ml-1 text-xs text-amber-300/80">({mp.reason})</span>
              ) : null}
            </li>
          ))}
        </ul>
      )}
      {error && (
        <div className="mb-2 rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">
          {error}
        </div>
      )}
      <div className="flex justify-end gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={onCancel}
          disabled={isSubmitting}
        >
          Cancel
        </Button>
        <Button
          size="sm"
          onClick={handleReWorkshop}
          disabled={isSubmitting}
          data-testid="stale-plan-reworkshop-button"
        >
          {isSubmitting ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Re-workshopping...
            </>
          ) : (
            "Re-workshop"
          )}
        </Button>
      </div>
    </div>
  );
}
