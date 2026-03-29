/**
 * Follow-Up Dialog - Triggers a follow-up execution from a completed/failed execution.
 *
 * Allows the user to choose:
 * - Follow-up type: fixup (from review), general continuation, or custom
 * - Run mode: continue existing agent session or spawn a fresh run
 * - Additional context/instructions
 */
import { useState, useEffect, useCallback } from "react";
import { Loader2, MessageSquare, Wrench, PenLine } from "lucide-react";
import { Dialog } from "../ui/dialog";
import { Button } from "../ui/button";
import { cn } from "../../lib";
import { selectors } from "../../consts/selectors";
import { executionService } from "../../services";
import type { ExecutionRecord, ReviewResult } from "../../types";
import type { FollowUpRequest } from "../../services/execution-service";

type FollowUpType = FollowUpRequest["followUpType"];
type RunMode = FollowUpRequest["runMode"];

interface FollowUpDialogProps {
  isOpen: boolean;
  onClose: () => void;
  execution: ExecutionRecord;
  onSuccess?: (newExecution: ExecutionRecord) => void;
}

function buildDefaultContext(execution: ExecutionRecord, type: FollowUpType): string {
  if (type !== "fixup" || !execution.reviewResult) return "";
  const rr = execution.reviewResult;
  let ctx = rr.summary ?? "";
  for (const dim of rr.dimensions ?? []) {
    if (dim.status !== "green" && dim.status !== "skipped") {
      ctx += `\n- ${dim.name} (${dim.status})${dim.details ? `: ${dim.details}` : ""}`;
    }
  }
  return ctx;
}

function ReviewSummaryPanel({ result }: { result: ReviewResult }) {
  const nonGreen = (result.dimensions ?? []).filter((d) => d.status !== "green" && d.status !== "skipped");
  if (nonGreen.length === 0) return null;

  return (
    <div className="rounded-md border border-slate-700 bg-slate-800/50 p-3 space-y-1.5" data-testid={selectors.followUp.reviewSummary}>
      <p className="text-xs font-medium text-slate-300">Review Findings</p>
      {nonGreen.map((dim) => (
        <div key={dim.name} className="flex items-center gap-2 text-xs">
          <span className={cn(
            "h-1.5 w-1.5 shrink-0 rounded-full",
            dim.status === "red" ? "bg-red-500" : "bg-amber-500",
          )} />
          <span className="text-slate-300">{dim.name}</span>
          {dim.details && <span className="text-slate-500">— {dim.details}</span>}
        </div>
      ))}
    </div>
  );
}

const TYPE_OPTIONS: { type: FollowUpType; label: string; description: string; icon: typeof Wrench; requiresReview: boolean }[] = [
  {
    type: "fixup",
    label: "Fix Review Issues",
    description: "Address issues found by the review",
    icon: Wrench,
    requiresReview: true,
  },
  {
    type: "followup",
    label: "General Follow-up",
    description: "Continue working on this item",
    icon: MessageSquare,
    requiresReview: false,
  },
  {
    type: "custom",
    label: "Custom",
    description: "Free-form instructions",
    icon: PenLine,
    requiresReview: false,
  },
];

export function FollowUpDialog({ isOpen, onClose, execution, onSuccess }: FollowUpDialogProps) {
  const hasReviewIssues = execution.reviewResult?.classification === "needs_work" || execution.reviewResult?.classification === "not_assessable";
  const canContinue = Boolean(execution.runId);

  const [followUpType, setFollowUpType] = useState<FollowUpType>(hasReviewIssues ? "fixup" : "followup");
  const [runMode, setRunMode] = useState<RunMode>(canContinue ? "continue" : "new");
  const [context, setContext] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Reset state when dialog opens
  useEffect(() => {
    if (isOpen) {
      const defaultType: FollowUpType = hasReviewIssues ? "fixup" : "followup";
      setFollowUpType(defaultType);
      setRunMode(canContinue ? "continue" : "new");
      setContext(buildDefaultContext(execution, defaultType));
      setError(null);
      setIsSubmitting(false);
    }
  }, [isOpen, execution, hasReviewIssues, canContinue]);

  // Update context when type changes
  const handleTypeChange = useCallback((type: FollowUpType) => {
    setFollowUpType(type);
    setContext(buildDefaultContext(execution, type));
  }, [execution]);

  const handleSubmit = async () => {
    setIsSubmitting(true);
    setError(null);
    try {
      const newExecution = await executionService.followUp(execution.executionId, {
        followUpType,
        context: context.trim() || undefined,
        runMode,
      });
      onSuccess?.(newExecution);
      onClose();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to create follow-up";
      if (msg.includes("session expired") || msg.includes("409")) {
        setError("Agent session has expired. Try again with \"New Run\" mode.");
      } else {
        setError(msg);
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const availableTypes = TYPE_OPTIONS.filter((opt) => !opt.requiresReview || hasReviewIssues);

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title={`Follow Up: ${execution.backlogKind}/${execution.backlogName}`}
      isLoading={isSubmitting}
      testId={selectors.followUp.dialog}
      maxWidth="max-w-md"
    >
      <div className="space-y-5">
        {/* Follow-up type selection */}
        <div className="space-y-2">
          <label className="text-sm font-medium text-slate-300">Type</label>
          <div className="grid gap-2">
            {availableTypes.map((opt) => {
              const Icon = opt.icon;
              const isSelected = followUpType === opt.type;
              return (
                <button
                  key={opt.type}
                  type="button"
                  className={cn(
                    "flex items-start gap-3 rounded-lg border p-3 text-left transition-colors",
                    isSelected
                      ? "border-cyan-500 bg-cyan-500/10"
                      : "border-white/10 bg-slate-800/50 hover:border-white/20",
                  )}
                  onClick={() => handleTypeChange(opt.type)}
                  data-testid={selectors.followUp[`type${opt.type.charAt(0).toUpperCase() + opt.type.slice(1)}` as keyof typeof selectors.followUp]}
                >
                  <Icon className={cn("mt-0.5 h-4 w-4 shrink-0", isSelected ? "text-cyan-400" : "text-slate-500")} />
                  <div>
                    <p className={cn("text-sm font-medium", isSelected ? "text-cyan-300" : "text-slate-300")}>
                      {opt.label}
                    </p>
                    <p className="text-xs text-slate-500">{opt.description}</p>
                  </div>
                </button>
              );
            })}
          </div>
        </div>

        {/* Run mode toggle */}
        <div className="space-y-2">
          <label className="text-sm font-medium text-slate-300">Run Mode</label>
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              className={cn(
                "rounded-lg border py-2 text-sm font-medium transition-colors",
                runMode === "continue"
                  ? "border-cyan-500 bg-slate-900 text-cyan-400"
                  : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20",
                !canContinue && "opacity-50 cursor-not-allowed",
              )}
              onClick={() => canContinue && setRunMode("continue")}
              disabled={!canContinue}
              title={!canContinue ? "No previous run to continue" : undefined}
              data-testid={selectors.followUp.runModeContinue}
            >
              Continue Run
            </button>
            <button
              type="button"
              className={cn(
                "rounded-lg border py-2 text-sm font-medium transition-colors",
                runMode === "new"
                  ? "border-cyan-500 bg-slate-900 text-cyan-400"
                  : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20",
              )}
              onClick={() => setRunMode("new")}
              data-testid={selectors.followUp.runModeNew}
            >
              New Run
            </button>
          </div>
          <p className="text-xs text-slate-500">
            {runMode === "continue"
              ? "Replies to the existing agent session, preserving conversation context."
              : "Starts a fresh agent with the follow-up instructions."}
          </p>
        </div>

        {/* Review summary (when review result exists and type is fixup) */}
        {followUpType === "fixup" && execution.reviewResult && (
          <ReviewSummaryPanel result={execution.reviewResult} />
        )}

        {/* Context textarea */}
        <div className="space-y-2">
          <label htmlFor="follow-up-context" className="text-sm font-medium text-slate-300">
            {followUpType === "custom" ? "Instructions" : "Additional Context"}
          </label>
          <textarea
            id="follow-up-context"
            className="w-full rounded-lg border border-white/10 bg-slate-800/50 px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            rows={4}
            value={context}
            onChange={(e) => setContext(e.target.value)}
            placeholder={followUpType === "custom" ? "Enter your instructions..." : "Add any additional context (optional)..."}
            data-testid={selectors.followUp.contextInput}
          />
        </div>

        {/* Error */}
        {error && (
          <p className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300" data-testid={selectors.followUp.error}>
            {error}
          </p>
        )}

        {/* Submit */}
        <div className="flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={onClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button
            size="sm"
            onClick={handleSubmit}
            disabled={isSubmitting || (followUpType === "custom" && !context.trim())}
            data-testid={selectors.followUp.submitButton}
          >
            {isSubmitting ? <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" /> : null}
            {runMode === "continue" ? "Continue" : "Start Follow-up"}
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
