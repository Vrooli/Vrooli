/**
 * RunBacklogModal — unified modal for running backlog items.
 *
 * Replaces the scattered Queue / Start Now / Schedule buttons with a single
 * "Run" entry-point that opens this modal.  Supports single-item and bulk modes.
 *
 * The modal fetches current queue depth on open and hides the "Queue" option
 * when the queue is empty (since it would behave identically to "Run Now").
 */

import { useEffect, useMemo, useState } from "react";
import { Play, Clock, ListOrdered, Loader2, AlertTriangle } from "lucide-react";
import { Dialog } from "../ui/dialog";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { backlogService, executionService } from "../../services";
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

type RunMode = "yolo" | "manual" | "scheduled";

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
// Mode option metadata
// ---------------------------------------------------------------------------

const MODE_OPTIONS: Array<{
  mode: RunMode;
  label: string;
  description: string;
  icon: typeof Play;
}> = [
  {
    mode: "yolo",
    label: "Run Now",
    description: "Starts immediately without waiting for review",
    icon: Play,
  },
  {
    mode: "manual",
    label: "Queue",
    description: "Adds to queue, starts when it reaches the front",
    icon: ListOrdered,
  },
  {
    mode: "scheduled",
    label: "Schedule",
    description: "Starts after a configurable delay",
    icon: Clock,
  },
];

const SUBMIT_LABELS: Record<RunMode, { idle: string; pending: string }> = {
  yolo: { idle: "Run Now", pending: "Running..." },
  manual: { idle: "Queue", pending: "Queueing..." },
  scheduled: { idle: "Schedule", pending: "Scheduling..." },
};

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
  const [selectedMode, setSelectedMode] = useState<RunMode>("yolo");
  const [delaySeconds, setDelaySeconds] = useState(300);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [blockedResult, setBlockedResult] = useState<QueueResponse | null>(null);
  const [showReadinessWarning, setShowReadinessWarning] = useState(false);

  // Queue depth
  const [pendingCount, setPendingCount] = useState(0);
  const [runningCount, setRunningCount] = useState(0);
  const [isLoadingCounts, setIsLoadingCounts] = useState(false);

  const isBulk = Boolean(targets && targets.length > 0);
  const effectiveTargets = useMemo(
    () => (isBulk && targets ? targets : target ? [target] : []),
    [isBulk, target, targets],
  );
  const queueDepth = pendingCount + runningCount;

  // Reset state and fetch queue depth when modal opens
  useEffect(() => {
    if (!isOpen) return;
    setSelectedMode("yolo");
    setDelaySeconds(300);
    setError(null);
    setBlockedResult(null);
    setIsSubmitting(false);
    setShowReadinessWarning(false);

    let cancelled = false;
    setIsLoadingCounts(true);
    Promise.all([
      executionService.list({ status: "pending" }),
      executionService.list({ status: "running" }),
    ])
      .then(([pending, running]) => {
        if (cancelled) return;
        setPendingCount(pending.length);
        setRunningCount(running.length);
      })
      .catch(() => {
        if (cancelled) return;
        setPendingCount(0);
        setRunningCount(0);
      })
      .finally(() => {
        if (!cancelled) setIsLoadingCounts(false);
      });

    return () => {
      cancelled = true;
    };
  }, [isOpen]);

  // If queue is empty and user had selected "manual", reset to "yolo"
  useEffect(() => {
    if (!isLoadingCounts && queueDepth === 0 && selectedMode === "manual") {
      setSelectedMode("yolo");
    }
  }, [isLoadingCounts, queueDepth, selectedMode]);

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
  // Submit
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
      mode: selectedMode,
      ...(selectedMode === "scheduled" ? { delaySeconds } : {}),
      startedBy: "swarm-manager-ui",
      confirm: true,
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

  const visibleModes = MODE_OPTIONS.filter(
    (opt) => opt.mode !== "manual" || queueDepth > 0,
  );

  const submitLabel = SUBMIT_LABELS[selectedMode];
  const delayValue =
    Number.isFinite(delaySeconds) && delaySeconds >= 0 ? delaySeconds : 0;

  // -------------------------------------------------------------------------
  // Render
  // -------------------------------------------------------------------------

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title={title}
      maxWidth="max-w-md"
      isLoading={isSubmitting}
      testId={selectors.runBacklog.dialog}
    >
      <div className="space-y-5">
        {/* Queue depth info */}
        {!isLoadingCounts && queueDepth > 0 && (
          <p
            className="text-sm text-slate-400"
            data-testid={selectors.runBacklog.queueDepth}
          >
            Queue: {pendingCount} pending, {runningCount} running
          </p>
        )}
        {isLoadingCounts && (
          <p className="flex items-center gap-2 text-sm text-slate-500">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            Loading queue status...
          </p>
        )}

        {/* Mode selection */}
        <div className="space-y-2">
          {visibleModes.map((opt) => {
            const Icon = opt.icon;
            const isSelected = selectedMode === opt.mode;
            return (
              <button
                key={opt.mode}
                type="button"
                disabled={isSubmitting}
                onClick={() => setSelectedMode(opt.mode)}
                className={`w-full rounded-lg border p-4 text-left transition ${
                  isSelected
                    ? "border-cyan-500/50 bg-cyan-500/10"
                    : "border-white/10 bg-slate-800/50 hover:bg-slate-800/80"
                } ${isSubmitting ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}`}
                data-testid={
                  opt.mode === "yolo"
                    ? selectors.runBacklog.modeYolo
                    : opt.mode === "manual"
                      ? selectors.runBacklog.modeManual
                      : selectors.runBacklog.modeScheduled
                }
              >
                <div className="flex items-center gap-3">
                  <Icon
                    className={`h-5 w-5 ${isSelected ? "text-cyan-400" : "text-slate-400"}`}
                  />
                  <div>
                    <div
                      className={`font-medium ${isSelected ? "text-cyan-100" : "text-slate-200"}`}
                    >
                      {opt.label}
                    </div>
                    <div className="mt-0.5 text-sm text-slate-400">
                      {opt.description}
                    </div>
                  </div>
                </div>
              </button>
            );
          })}
        </div>

        {/* Schedule delay input */}
        {selectedMode === "scheduled" && (
          <div className="space-y-1.5">
            <label
              htmlFor="run-backlog-delay"
              className="text-sm font-medium text-slate-300"
            >
              Delay (seconds)
            </label>
            <Input
              id="run-backlog-delay"
              type="number"
              min={0}
              step={1}
              value={delayValue}
              onChange={(e) => setDelaySeconds(Number(e.target.value || 0))}
              disabled={isSubmitting}
              className="w-32"
              data-testid={selectors.runBacklog.delayInput}
            />
          </div>
        )}

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

        {/* Blocking reasons */}
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
                {submitLabel.pending}
              </>
            ) : showReadinessWarning ? (
              "Execute Anyway"
            ) : (
              submitLabel.idle
            )}
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
