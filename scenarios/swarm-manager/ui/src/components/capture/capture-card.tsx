/**
 * Capture Card
 *
 * Displays a raw capture with classification triage.
 *
 * States:
 * 1. Classifying: spinner + "Classifying..." below title
 * 2. Classified: title + compact suggested items with accept/edit/dismiss
 * 3. Classified (no-op): title + "Nothing actionable detected" + dismiss
 * 4. Failed: title + error with retry
 */

import { useState } from "react";
import { Loader2, RefreshCw, X } from "lucide-react";
import { captureService } from "../../services/capture-service";
import { useCaptureStore } from "../../stores/capture-store";
import { formatRelativeTime } from "../../lib";
import { selectors } from "../../consts/selectors";
import { CaptureTriage } from "./capture-triage";
import type { Capture, CaptureFailureReason } from "../../types";
import { NoteIndicator } from "../ui/note-indicator";
import type { BacklogFormValues } from "../../types";

/** User-facing failure messages keyed by categorized failure reason. */
const FAILURE_MESSAGES: Record<CaptureFailureReason, { label: string; hint: string }> = {
  dependency_unavailable: {
    label: "Service unavailable",
    hint: "Agent-manager or prompt-manager isn't running yet. Retry once services are healthy.",
  },
  classification_timeout: {
    label: "Classification timed out",
    hint: "The classification agent didn't finish in time. Retry to try again.",
  },
  prompt_missing: {
    label: "Classification skill not found",
    hint: "The prompt-manager may still be starting up, or the skill is misconfigured.",
  },
  agent_error: {
    label: "Agent failed to start",
    hint: "Check agent-manager logs for details.",
  },
  internal_error: {
    label: "Unexpected error",
    hint: "Something went wrong on the server. Retry or check logs.",
  },
};

interface CaptureCardProps {
  capture: Capture;
  onEditItem?: (prefill: BacklogFormValues) => void;
  onClick?: () => void;
  className?: string;
}

export function CaptureCard({ capture, onEditItem, onClick, className }: CaptureCardProps) {
  const [isRetrying, setIsRetrying] = useState(false);
  const removeCapture = useCaptureStore((s) => s.removeCapture);
  const updateCapture = useCaptureStore((s) => s.updateCapture);

  const items = capture.classification?.items ?? [];

  const handleDismissCapture = async (e?: React.MouseEvent) => {
    e?.stopPropagation();
    await captureService.remove(capture.id);
    removeCapture(capture.id);
  };

  const handleRetry = async (e?: React.MouseEvent) => {
    e?.stopPropagation();
    setIsRetrying(true);
    try {
      await captureService.classify(capture.id);
      updateCapture(capture.id, { status: "classifying", classification: null });
    } catch {
      // Keep failed state.
    } finally {
      setIsRetrying(false);
    }
  };

  const statusDotColor =
    capture.status === "classifying" ? "bg-cyan-400" :
    capture.status === "failed" ? "bg-red-400" :
    capture.status === "classified" && items.length > 0 ? "bg-emerald-400" :
    "bg-slate-500";

  return (
    <div
      className={`${className ?? ""}${onClick ? " cursor-pointer transition-colors hover:bg-slate-800/50" : ""}`}
      data-testid={selectors.captures.card}
      onClick={onClick}
      role={onClick ? "button" : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={onClick ? (e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); onClick(); } } : undefined}
    >
      {/* Header: status dot + capture badge (left), timestamp + dismiss (right) */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-2">
          <span className={`inline-block h-2 w-2 rounded-full ${statusDotColor}`} />
          <span className="rounded-full bg-violet-500/20 px-2 py-0.5 text-[11px] font-medium text-violet-300">
            Capture
          </span>
          <NoteIndicator note={capture.note} />
        </div>
        <div className="flex items-center gap-1.5">
          <span className="text-[11px] text-slate-600">{formatRelativeTime(capture.created)}</span>
          <button
            onClick={(e) => { e.stopPropagation(); handleDismissCapture(); }}
            className="shrink-0 rounded p-0.5 text-slate-600 transition-colors hover:bg-slate-700 hover:text-slate-400"
            title="Dismiss capture"
            data-testid={selectors.captures.dismissButton}
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>
      {/* Title: original capture text */}
      <h3 className="mt-3 font-medium text-slate-100">{capture.text}</h3>

      {/* Classifying */}
      {capture.status === "classifying" && (
        <div className="mt-1 flex items-center gap-1.5 text-xs text-cyan-400">
          <Loader2 className="h-3 w-3 animate-spin" />
          Classifying...
        </div>
      )}

      {/* Failed — show categorized message with recovery hint */}
      {capture.status === "failed" && (() => {
        const info = capture.failureReason
          ? FAILURE_MESSAGES[capture.failureReason] ?? FAILURE_MESSAGES.internal_error
          : { label: "Classification failed", hint: "Retry to try again." };
        return (
          <div className="mt-1.5 space-y-0.5">
            <div className="flex items-center gap-2">
              <span className="text-xs font-medium text-red-400">{info.label}</span>
              <button
                onClick={(e) => handleRetry(e)}
                disabled={isRetrying}
                className="inline-flex items-center gap-1 text-xs text-slate-400 hover:text-slate-200 disabled:opacity-50"
                data-testid={selectors.captures.retryButton}
              >
                <RefreshCw className={`h-3 w-3 ${isRetrying ? "animate-spin" : ""}`} />
                Retry
              </button>
            </div>
            <p className="text-[11px] text-slate-500">{info.hint}</p>
          </div>
        );
      })()}

      {/* No-op: nothing actionable */}
      {capture.status === "classified" && items.length === 0 && (
        <div className="mt-1 flex items-center justify-between">
          <span className="text-xs text-slate-500 italic">Nothing actionable detected</span>
          <button
            onClick={(e) => { e.stopPropagation(); handleDismissCapture(); }}
            className="text-xs text-slate-500 hover:text-slate-300"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* Suggestions via shared CaptureTriage */}
      {capture.status === "classified" && items.length > 0 && (
        <div className="mt-1.5" onClick={(e) => e.stopPropagation()}>
          <CaptureTriage
            capture={capture}
            onEditItem={onEditItem}
          />
        </div>
      )}
    </div>
  );
}
