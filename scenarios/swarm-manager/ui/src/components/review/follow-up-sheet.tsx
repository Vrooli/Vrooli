/**
 * FollowUpSheet — Drawer-based follow-up panel that shows evidence
 * context inline when addressing review issues.
 *
 * Replaces the previous FollowUpDialog (centered Dialog) with a
 * responsive Drawer (420px right-side on desktop, bottom-sheet on mobile).
 */

import { useState, useEffect, useCallback } from "react";
import { Loader2, MessageSquare, Wrench, PenLine, Clock, Zap, DollarSign, FileCode, AlertTriangle } from "lucide-react";
import { Drawer } from "../ui/drawer";
import { Button } from "../ui/button";
import { EvidenceContextSummary } from "./evidence-context-summary";
import { buildFinalizationContext, cn, hasActionableFinalizationIssues } from "../../lib";
import { selectors } from "../../consts/selectors";
import { agentManagerService, executionService } from "../../services";
import type { AgentRunState } from "../../types";
import type { ExecutionRecord } from "../../types";
import type { FollowUpRequest } from "../../services/execution-service";
import type { ReviewRound } from "../../services/review-service";

type FollowUpType = FollowUpRequest["followUpType"];
type RunMode = FollowUpRequest["runMode"];

export interface FollowUpSheetProps {
  isOpen: boolean;
  onClose: () => void;
  execution: ExecutionRecord;
  reviewRounds: ReviewRound[];
  onSuccess?: (newExecution: ExecutionRecord) => void;
}

function buildDefaultContext(execution: ExecutionRecord, type: FollowUpType): string {
  if (type !== "fixup") return "";
  return buildFinalizationContext(execution.finalization);
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

/** Estimated context window size for common models. */
const CONTEXT_WINDOW_TOKENS = 200_000;

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${Math.round(seconds)}s`;
  const mins = Math.floor(seconds / 60);
  const secs = Math.round(seconds % 60);
  return secs > 0 ? `${mins}m ${secs}s` : `${mins}m`;
}

function formatTokens(tokens: number): string {
  if (tokens >= 1_000_000) return `${(tokens / 1_000_000).toFixed(1)}M`;
  if (tokens >= 1_000) return `${(tokens / 1_000).toFixed(0)}k`;
  return String(tokens);
}

function RunHealthIndicator({ runState }: { runState: AgentRunState | null }) {
  if (!runState) return null;

  const { contextTokens, turnsUsed, costEstimate, changedFiles, durationSeconds } = runState;
  const hasAnyMetric = contextTokens || turnsUsed || costEstimate || changedFiles || durationSeconds;
  if (!hasAnyMetric) return null;

  const contextPercent = contextTokens ? Math.min((contextTokens / CONTEXT_WINDOW_TOKENS) * 100, 100) : 0;
  const isContextHigh = contextPercent > 70;

  return (
    <div
      className="rounded-lg border border-white/10 bg-slate-800/30 p-3 space-y-2.5"
      data-testid={selectors.followUp.runHealth}
    >
      <p className="text-xs font-medium text-slate-400">Previous Run</p>

      <div className="grid grid-cols-2 gap-x-4 gap-y-1.5">
        {turnsUsed != null && (
          <div className="flex items-center gap-1.5 text-xs text-slate-400">
            <MessageSquare className="h-3 w-3 shrink-0 text-slate-500" />
            <span><span className="text-slate-300 font-medium">{turnsUsed}</span> turns</span>
          </div>
        )}
        {durationSeconds != null && (
          <div className="flex items-center gap-1.5 text-xs text-slate-400">
            <Clock className="h-3 w-3 shrink-0 text-slate-500" />
            <span className="text-slate-300 font-medium">{formatDuration(durationSeconds)}</span>
          </div>
        )}
        {changedFiles != null && (
          <div className="flex items-center gap-1.5 text-xs text-slate-400">
            <FileCode className="h-3 w-3 shrink-0 text-slate-500" />
            <span><span className="text-slate-300 font-medium">{changedFiles}</span> files changed</span>
          </div>
        )}
        {costEstimate != null && (
          <div className="flex items-center gap-1.5 text-xs text-slate-400">
            <DollarSign className="h-3 w-3 shrink-0 text-slate-500" />
            <span className="text-slate-300 font-medium">${costEstimate.toFixed(2)}</span>
          </div>
        )}
      </div>

      {/* Context window usage bar */}
      {contextTokens != null && (
        <div className="space-y-1">
          <div className="flex items-center justify-between text-xs">
            <div className="flex items-center gap-1.5 text-slate-400">
              <Zap className="h-3 w-3 shrink-0 text-slate-500" />
              <span>Context: <span className="text-slate-300 font-medium">{formatTokens(contextTokens)}</span> tokens</span>
            </div>
            <span className={cn("font-medium", isContextHigh ? "text-amber-400" : "text-slate-500")}>
              {contextPercent.toFixed(0)}%
            </span>
          </div>
          <div className="h-1.5 w-full rounded-full bg-slate-700">
            <div
              className={cn(
                "h-full rounded-full transition-all",
                contextPercent > 85 ? "bg-red-500" : isContextHigh ? "bg-amber-500" : "bg-cyan-500",
              )}
              style={{ width: `${contextPercent}%` }}
            />
          </div>
        </div>
      )}

      {/* Warning hint */}
      {isContextHigh && (
        <div className="flex items-start gap-1.5 text-[11px] text-amber-400/80">
          <AlertTriangle className="mt-0.5 h-3 w-3 shrink-0" />
          <span>Context window is filling up — a new run may be more reliable.</span>
        </div>
      )}
    </div>
  );
}

export function FollowUpSheet({ isOpen, onClose, execution, reviewRounds, onSuccess }: FollowUpSheetProps) {
  const hasReviewIssues = hasActionableFinalizationIssues(execution);
  const canContinue = Boolean(execution.runId);

  const [followUpType, setFollowUpType] = useState<FollowUpType>(hasReviewIssues ? "fixup" : "followup");
  const [runMode, setRunMode] = useState<RunMode>(canContinue ? "continue" : "new");
  const [context, setContext] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedEvidenceIds, setSelectedEvidenceIds] = useState<Set<string>>(new Set());
  const [runState, setRunState] = useState<AgentRunState | null>(null);

  // Fetch run metrics when the sheet opens and a runId exists.
  useEffect(() => {
    if (!isOpen || !execution.runId) {
      setRunState(null);
      return;
    }
    let cancelled = false;
    agentManagerService.getRunState(execution.runId).then(
      (state) => { if (!cancelled) setRunState(state); },
      () => { if (!cancelled) setRunState(null); },
    );
    return () => { cancelled = true; };
  }, [isOpen, execution.runId]);

  useEffect(() => {
    if (isOpen) {
      const defaultType: FollowUpType = hasReviewIssues ? "fixup" : "followup";
      setFollowUpType(defaultType);
      setRunMode(canContinue ? "continue" : "new");
      setContext(buildDefaultContext(execution, defaultType));
      setError(null);
      setIsSubmitting(false);
      // Pre-select all evidence from latest round for fixup; empty otherwise
      if (defaultType === "fixup" && reviewRounds.length > 0) {
        const latest = reviewRounds[reviewRounds.length - 1];
        setSelectedEvidenceIds(new Set<string>(latest?.evidence.map((e) => e.id) ?? []));
      } else {
        setSelectedEvidenceIds(new Set<string>());
      }
    }
  }, [isOpen, execution, hasReviewIssues, canContinue, reviewRounds]);

  const handleTypeChange = useCallback((type: FollowUpType) => {
    setFollowUpType(type);
    setContext(buildDefaultContext(execution, type));
  }, [execution]);

  const handleSubmit = async () => {
    setIsSubmitting(true);
    setError(null);
    try {
      // Build context with optional evidence references
      let finalContext = context.trim();
      if (selectedEvidenceIds.size > 0) {
        const latestRound = reviewRounds[reviewRounds.length - 1];
        const selectedItems = latestRound?.evidence.filter((e) => selectedEvidenceIds.has(e.id)) ?? [];
        if (selectedItems.length > 0) {
          const evidenceBlock = selectedItems
            .map((e) => `- [${e.type}] ${e.title}: ${e.description}`)
            .join("\n");
          finalContext = finalContext
            ? `${finalContext}\n\n--- Referenced Evidence ---\n${evidenceBlock}`
            : `--- Referenced Evidence ---\n${evidenceBlock}`;
        }
      }
      const newExecution = await executionService.followUp(execution.executionId, {
        followUpType,
        context: finalContext || undefined,
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
    <Drawer
      isOpen={isOpen}
      onClose={onClose}
      title={`Follow Up: ${execution.backlogKind}/${execution.backlogName}`}
      testId={selectors.review.followUpSheet}
      footer={
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
      }
    >
      <div className="space-y-5 px-4 py-4">
        {/* Evidence context — always shown when rounds exist */}
        {reviewRounds.length > 0 && (
          <EvidenceContextSummary
            rounds={reviewRounds}
            selectable
            selectedIds={selectedEvidenceIds}
            onToggle={(id) => {
              setSelectedEvidenceIds((prev) => {
                const next = new Set<string>(prev);
                if (next.has(id)) next.delete(id);
                else next.add(id);
                return next;
              });
            }}
          />
        )}

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

        {/* Run health — informs Continue vs New Run decision */}
        {canContinue && <RunHealthIndicator runState={runState} />}

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
      </div>
    </Drawer>
  );
}
