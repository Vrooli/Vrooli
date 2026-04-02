/**
 * CommandPostOverlay — Full-screen overlay for the Command Post.
 *
 * Manages internal view state (summary vs decision-stream).
 * Escape key closes the overlay.
 */

import { useState, useEffect, useCallback, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { X } from "lucide-react";
import type { DetailSelection } from "../../stores/detail-selection-store";
import { useSnoozeStore, useSnoozedKeys } from "../../stores/snooze-store";
import { aggregateCrossItemQuestions } from "../../lib/command-post-utils";

import { backlogService } from "../../services";
import { SummaryView } from "./SummaryView";
import { DecisionStreamView } from "./DecisionStreamView";

interface CommandPostOverlayProps {
  isOpen: boolean;
  onClose: () => void;
  onNavigateToDetail: (selection: DetailSelection) => void;
  onSwitchLens: (lens: string) => void;
}

type ViewState = "summary" | "decision-stream";

export function CommandPostOverlay({
  isOpen,
  onClose,
  onNavigateToDetail,
  onSwitchLens,
}: CommandPostOverlayProps) {
  const [view, setView] = useState<ViewState>("summary");
  const snoozedKeys = useSnoozedKeys();
  const snooze = useSnoozeStore((s) => s.snooze);

  const summaryQuery = useQuery({
    queryKey: ["backlog-summary"],
    queryFn: () => backlogService.getBacklogSummary(),
    staleTime: 60_000,
    enabled: isOpen,
  });

  const questions = useMemo(() => {
    const pqi = summaryQuery.data?.pending_questions?.items ?? [];
    return aggregateCrossItemQuestions(pqi, snoozedKeys);
  }, [summaryQuery.data?.pending_questions, snoozedKeys]);

  // Reset to summary when reopened
  useEffect(() => {
    if (isOpen) setView("summary");
  }, [isOpen]);

  // Escape key closes
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        if (view === "decision-stream") {
          setView("summary");
        } else {
          onClose();
        }
      }
    },
    [onClose, view],
  );

  useEffect(() => {
    if (!isOpen) return;
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, handleKeyDown]);

  if (!isOpen) return null;

  return (
    <div
      className="absolute inset-0 z-40 overflow-y-auto bg-slate-950"
      data-testid="command-post-overlay"
    >
      {/* Header */}
      <div className="sticky top-0 z-10 flex items-center justify-between border-b border-slate-800 bg-slate-950/95 px-6 py-4 backdrop-blur-sm">
        <h2 className="text-lg font-semibold text-slate-100">Command Post</h2>
        <button
          type="button"
          onClick={onClose}
          className="rounded-lg p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label="Close Command Post"
          data-testid="command-post-close"
        >
          <X className="h-5 w-5" />
        </button>
      </div>

      {/* Content */}
      <div className="mx-auto max-w-4xl px-6 py-6">
        {view === "summary" ? (
          <SummaryView
            onEnterDecisionStream={() => setView("decision-stream")}
            onNavigateToDetail={onNavigateToDetail}
            onSwitchLens={(lens) => {
              onSwitchLens(lens);
              onClose();
            }}
            onClose={onClose}
          />
        ) : (
          <DecisionStreamView
            questions={questions}
            onComplete={() => setView("summary")}
            onBack={() => setView("summary")}
            onSnoozeItem={(key) => snooze(key, Date.now() + 3_600_000)}
          />
        )}
      </div>
    </div>
  );
}
