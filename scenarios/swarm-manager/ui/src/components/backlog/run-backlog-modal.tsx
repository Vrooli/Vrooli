/**
 * RunBacklogModal — lightweight queue preflight and confirmation dialog.
 *
 * New flow:
 * 1. When modal opens, auto-submit a queue call (mode=yolo, confirm=true).
 * 2. If no issues + single item: fire onSuccess, close. User never sees the modal.
 * 3. If blocking reasons: show blocker dialog, user can override and re-submit.
 * 4. For bulk mode: show progress as items are queued sequentially.
 */

import { useEffect, useMemo, useRef, useState } from "react";
import { Loader2, AlertTriangle } from "lucide-react";
import { Dialog } from "../ui/dialog";
import { Button } from "../ui/button";
import { backlogService } from "../../services";
import type { QueueResponse } from "../../services";
import type { BacklogKind } from "../../types";
import { selectors } from "../../consts/selectors";
import { isApiError } from "../../lib/api-client";
import { StalePlanPanel } from "./stale-plan-panel";
import { extractMissingPaths, type MissingPath } from "./stale-plan-utils";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface RunBacklogTarget {
  kind: BacklogKind;
  name: string;
  title?: string;
}

export interface RunBacklogModalProps {
  isOpen: boolean;
  onClose: () => void;
  /** Single-item mode */
  target?: RunBacklogTarget;
  /** Bulk mode (overrides target) */
  targets?: RunBacklogTarget[];
  /** Called after a successful (non-dry-run) queue */
  onSuccess?: (result: QueueResponse) => void;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function RunBacklogModal({
  isOpen,
  onClose,
  target,
  targets,
  onSuccess,
}: RunBacklogModalProps) {
  // State
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [blockedResult, setBlockedResult] = useState<QueueResponse | null>(null);
  const [forceConfirmed, setForceConfirmed] = useState(false);
  // Track whether the auto-run preflight has completed (to avoid showing dialog flash)
  const [preflightDone, setPreflightDone] = useState(false);
  // plan_stale state — surfaces the StalePlanPanel for the offending item.
  const [stalePlanFor, setStalePlanFor] = useState<{
    kind: BacklogKind;
    name: string;
    missingPaths: MissingPath[];
  } | null>(null);

  const isBulk = Boolean(targets && targets.length > 0);
  const effectiveTargets = useMemo(
    () => (isBulk && targets ? targets : target ? [target] : []),
    [isBulk, target, targets],
  );

  // Ref to track the open session and prevent stale callbacks
  const sessionRef = useRef(0);

  // -------------------------------------------------------------------------
  // Auto-run on open
  // -------------------------------------------------------------------------

  useEffect(() => {
    if (!isOpen) return;

    // Reset state
    setError(null);
    setBlockedResult(null);
    setIsSubmitting(false);
    setForceConfirmed(false);
    setPreflightDone(false);

    const session = ++sessionRef.current;

    // For bulk mode, always show the dialog for progress tracking
    if (isBulk) {
      setPreflightDone(true);
      return;
    }

    // Single item: auto-submit
    if (effectiveTargets.length === 1) {
      const item = effectiveTargets[0];
      if (!item) {
        setPreflightDone(true);
        return;
      }

      setIsSubmitting(true);
      backlogService
        .queue(item.kind, item.name, {
          mode: "yolo",
          startedBy: "swarm-manager-ui",
          confirm: true,
        })
        .then((result) => {
          if (session !== sessionRef.current) return;

          if (result.dryRun && result.blockingReasons.length > 0) {
            // Show dialog with blockers
            setBlockedResult(result);
            setPreflightDone(true);
            setIsSubmitting(false);
            return;
          }

          // Success - close without ever showing the modal
          onSuccess?.(result);
          onClose();
        })
        .catch((err) => {
          if (session !== sessionRef.current) return;
          if (isApiError(err) && err.code === "plan_stale") {
            setStalePlanFor({
              kind: item.kind,
              name: item.name,
              missingPaths: extractMissingPaths(err.details),
            });
            setPreflightDone(true);
            setIsSubmitting(false);
            return;
          }
          setError(
            err instanceof Error ? err.message : "Failed to queue backlog item.",
          );
          setPreflightDone(true);
          setIsSubmitting(false);
        });
      return;
    }

    // No targets
    setPreflightDone(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen]);

  // -------------------------------------------------------------------------
  // Submit (manual click from dialog)
  // -------------------------------------------------------------------------

  const handleSubmit = async () => {
    if (effectiveTargets.length === 0 || isSubmitting) return;

    setIsSubmitting(true);
    setError(null);
    setBlockedResult(null);

    const options = {
      mode: "yolo" as const,
      startedBy: "swarm-manager-ui",
      confirm: true,
      ...(forceConfirmed ? { force: true } : {}),
    };

    try {
      if (isBulk) {
        let queuedCount = 0;
        const failures: string[] = [];
        let lastResult: QueueResponse | undefined;

        for (const item of effectiveTargets) {
          try {
            const result = await backlogService.queue(item.kind, item.name, options);
            if (result.dryRun && result.blockingReasons.length > 0) {
              failures.push(`${item.kind}/${item.name}`);
              continue;
            }
            lastResult = result;
            queuedCount += 1;
          } catch {
            failures.push(`${item.kind}/${item.name}`);
          }
        }

        if (failures.length > 0) {
          const preview = failures.slice(0, 3).join(", ");
          const suffix = failures.length > 3 ? ", ..." : "";
          setError(
            `Queued ${queuedCount}/${effectiveTargets.length}. Failed: ${preview}${suffix}`,
          );
          if (queuedCount > 0 && lastResult) {
            onSuccess?.(lastResult);
          }
        } else if (lastResult) {
          onSuccess?.(lastResult);
          onClose();
        }
      } else {
        const item = effectiveTargets[0];
        if (!item) return;
        const result = await backlogService.queue(item.kind, item.name, options);

        if (result.dryRun && result.blockingReasons.length > 0) {
          setBlockedResult(result);
          return;
        }

        onSuccess?.(result);
        onClose();
      }
    } catch (err) {
      if (isApiError(err) && err.code === "plan_stale") {
        // Single-item path; for bulk we currently surface as a generic error.
        const item = effectiveTargets[0];
        if (!isBulk && item) {
          setStalePlanFor({
            kind: item.kind,
            name: item.name,
            missingPaths: extractMissingPaths(err.details),
          });
          return;
        }
      }
      setError(
        err instanceof Error ? err.message : "Failed to queue backlog item.",
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  // -------------------------------------------------------------------------
  // Derived
  // -------------------------------------------------------------------------

  const title = isBulk
    ? `Run ${effectiveTargets.length} item${effectiveTargets.length === 1 ? "" : "s"}`
    : target?.title
      ? `Run "${target.title}"`
      : "Run backlog item";

  // Don't render the dialog content until preflight resolves (prevents flash)
  const showDialog = preflightDone;

  // -------------------------------------------------------------------------
  // Render
  // -------------------------------------------------------------------------

  return (
    <Dialog
      isOpen={isOpen && showDialog}
      onClose={onClose}
      title={title}
      maxWidth="max-w-md"
      isLoading={isSubmitting}
      testId={selectors.runBacklog.dialog}
    >
      <div className="space-y-5">
        {/* Stale plan panel — opens a fresh Plan Workshop review for plan_stale errors */}
        {stalePlanFor && (
          <StalePlanPanel
            kind={stalePlanFor.kind}
            name={stalePlanFor.name}
            missingPaths={stalePlanFor.missingPaths}
            onReWorkshopped={() => {
              setStalePlanFor(null);
              onClose();
            }}
            onCancel={() => {
              setStalePlanFor(null);
              onClose();
            }}
          />
        )}

        {/* Blocking reasons — with option to force-run */}
        {blockedResult && blockedResult.blockingReasons.length > 0 && (
          <div
            className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-200"
            data-testid={selectors.runBacklog.blockingReasons}
          >
            <p className="font-medium">{blockedResult.message}</p>
            <ul className="mt-2 list-disc pl-5 space-y-1">
              {blockedResult.blockingReasons.map((reason, i) => (
                <li key={i}>
                  {reason.message}
                  {reason.forceable && (
                    <span className="ml-1.5 text-xs text-amber-400">(overridable)</span>
                  )}
                </li>
              ))}
            </ul>
            {blockedResult.pendingDecisions > 0 && (
              <p className="mt-2 text-xs text-amber-300">
                {blockedResult.pendingDecisions} pending decision
                {blockedResult.pendingDecisions === 1 ? "" : "s"}
              </p>
            )}
            {!forceConfirmed && (
              <Button
                variant="outline"
                size="sm"
                className="mt-3 border-amber-500/40 text-amber-200 hover:bg-amber-500/20"
                onClick={() => setForceConfirmed(true)}
              >
                <AlertTriangle className="mr-1.5 h-3.5 w-3.5" />
                Override and run anyway
              </Button>
            )}
            {forceConfirmed && (
              <p className="mt-3 text-xs font-medium text-amber-100">
                Blockers will be overridden. Click the run button to confirm.
              </p>
            )}
          </div>
        )}

        {/* Error */}
        {error && (
          <div
            className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300"
            data-testid={selectors.runBacklog.error}
          >
            {error}
          </div>
        )}

        {/* Footer */}
        <div className="flex justify-end gap-3 pt-2">
          <Button
            variant="outline"
            onClick={onClose}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={isSubmitting || effectiveTargets.length === 0}
            data-testid={selectors.runBacklog.submitButton}
          >
            {isSubmitting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Running...
              </>
            ) : "Run"}
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
