/**
 * RunBacklogModal — lightweight auto-run + warning dialog.
 *
 * New flow:
 * 1. When modal opens, auto-submit a queue call (mode=yolo, confirm=true).
 * 2. If no issues + single item: fire onSuccess, close. User never sees the modal.
 * 3. If readiness warnings: show warning dialog, user clicks "Execute Anyway".
 * 4. If blocking reasons: show blocker dialog, user can override and re-submit.
 * 5. For bulk mode: show progress as items are queued sequentially.
 */

import { useEffect, useMemo, useRef, useState } from "react";
import { Loader2, AlertTriangle } from "lucide-react";
import { Dialog } from "../ui/dialog";
import { Button } from "../ui/button";
import { backlogService } from "../../services";
import type { QueueResponse } from "../../services";
import type { BacklogKind } from "../../types";
import { selectors } from "../../consts/selectors";
import type { ReadinessIndicatorData } from "../../lib/maturity";
import { READINESS_DIMENSIONS, DIMENSION_LABELS } from "../../lib/maturity";

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
  /** Readiness data for single-item mode */
  readinessData?: ReadinessIndicatorData | null;
  /** Readiness data map for bulk mode (keyed by "kind/name") */
  readinessDataMap?: Map<string, ReadinessIndicatorData>;
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
  readinessData,
  readinessDataMap,
}: RunBacklogModalProps) {
  // State
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [blockedResult, setBlockedResult] = useState<QueueResponse | null>(null);
  const [showReadinessWarning, setShowReadinessWarning] = useState(false);
  const [forceConfirmed, setForceConfirmed] = useState(false);
  // Track whether the auto-run preflight has completed (to avoid showing dialog flash)
  const [preflightDone, setPreflightDone] = useState(false);

  const isBulk = Boolean(targets && targets.length > 0);
  const effectiveTargets = useMemo(
    () => (isBulk && targets ? targets : target ? [target] : []),
    [isBulk, target, targets],
  );

  // Ref to track the open session and prevent stale callbacks
  const sessionRef = useRef(0);

  // -------------------------------------------------------------------------
  // Readiness warnings
  // -------------------------------------------------------------------------

  /** Collect readiness warnings for items that have a plan but are not fully ready. */
  const readinessWarnings = useMemo(() => {
    const warnings: Array<{
      key: string;
      title?: string;
      noWorkshopRounds: boolean;
      weakDimensions: string[];
    }> = [];

    for (const t of effectiveTargets) {
      const key = `${t.kind}/${t.name}`;
      const data = isBulk
        ? readinessDataMap?.get(key)
        : readinessData ?? undefined;
      if (!data) continue;
      // Only warn when a plan exists but readiness is incomplete
      if (!data.hasPlan) continue;
      if (data.ready) continue;

      const noWorkshopRounds = data.roundsCompleted === 0;
      const weakDimensions = READINESS_DIMENSIONS
        .filter((dim) => data.effectiveScores[dim] < 3)
        .map((dim) => `${DIMENSION_LABELS[dim]} (${data.effectiveScores[dim]}/3)`);

      if (noWorkshopRounds || weakDimensions.length > 0) {
        warnings.push({ key, title: t.title, noWorkshopRounds, weakDimensions });
      }
    }
    return warnings;
  }, [effectiveTargets, isBulk, readinessData, readinessDataMap]);

  const needsReadinessConfirmation = readinessWarnings.length > 0;

  // -------------------------------------------------------------------------
  // Auto-run on open
  // -------------------------------------------------------------------------

  useEffect(() => {
    if (!isOpen) return;

    // Reset state
    setError(null);
    setBlockedResult(null);
    setIsSubmitting(false);
    setShowReadinessWarning(false);
    setForceConfirmed(false);
    setPreflightDone(false);

    const session = ++sessionRef.current;

    // If readiness warnings exist, show dialog immediately
    if (needsReadinessConfirmation) {
      setShowReadinessWarning(true);
      setPreflightDone(true);
      return;
    }

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

    // Gate: show readiness warning if needed and not yet acknowledged
    if (needsReadinessConfirmation && !showReadinessWarning) {
      setShowReadinessWarning(true);
      return;
    }

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
        {/* Readiness warning */}
        {showReadinessWarning && readinessWarnings.length > 0 && (
          <div
            className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-200"
            data-testid={selectors.runBacklog.readinessWarning}
          >
            <div className="mb-2 flex items-center gap-2 font-medium">
              <AlertTriangle className="h-4 w-4 text-amber-400" />
              Low readiness — proceed with caution
            </div>
            {readinessWarnings.map((w) => (
              <div key={w.key} className="mt-1.5">
                {isBulk && (
                  <p className="font-medium text-amber-100">
                    {w.title ?? w.key}
                  </p>
                )}
                {w.noWorkshopRounds && (
                  <p>No workshop rounds completed — plan was created manually</p>
                )}
                {w.weakDimensions.length > 0 && (
                  <ul className="mt-1 list-disc pl-5 space-y-0.5">
                    {w.weakDimensions.map((d) => (
                      <li key={d}>{d}</li>
                    ))}
                  </ul>
                )}
              </div>
            ))}
          </div>
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
                <li key={i}>{reason}</li>
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
            onClick={() => {
              if (showReadinessWarning) {
                setShowReadinessWarning(false);
              } else {
                onClose();
              }
            }}
            disabled={isSubmitting}
          >
            {showReadinessWarning ? "Back" : "Cancel"}
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
            ) : showReadinessWarning ? (
              "Execute Anyway"
            ) : (
              "Run"
            )}
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
