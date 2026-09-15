/**
 * OperationalTargetsPanel
 *
 * Displays archive targets grouped by criticality and requirements
 * grouped by module, with selection, review mode, and CRUD actions.
 *
 * Selection and review state come from backlog-detail-ui-store.
 */

import { useState, useMemo } from "react";
import { ChevronDown, ChevronRight, Target, FileCheck, Plus, Pencil, Trash2, ArrowUp, ArrowDown, ClipboardCheck } from "lucide-react";
import { ReviewCard } from "./review-card";
import { ActionsMenu, type ActionsMenuItem } from "./actions-menu";
import { getReviewStatus, ReviewSummary, ReviewProgressBar, StatusIcon, ReviewStatusIndicator, countAllRequirements } from "./review-indicators";
import { cn } from "../../lib";
import { useBacklogDetailUIStore } from "../../stores";
import type { ArchiveTarget, ArchiveRequirement, ArchiveRequirementGroup, ReviewAction } from "../../types";

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface OperationalTargetsPanelProps {
  targets: ArchiveTarget[];
  requirements: ArchiveRequirementGroup[];
  editable?: boolean;
  onCreateRequirement?: (groupId: string) => void;
  onEditRequirement?: (groupId: string, requirement: ArchiveRequirement) => void;
  onDeleteRequirement?: (groupId: string, requirementId: string) => void;
  onReorderRequirement?: (groupId: string, requirementId: string, direction: "up" | "down") => void;
  onCreateModule?: () => void;
  onEditModule?: (groupId: string) => void;
  onDeleteModule?: (groupId: string) => void;
  onCreateTarget?: () => void;
  onEditTarget?: (target: ArchiveTarget) => void;
  onDeleteTarget?: (targetId: string) => void;
  onReviewAction?: (id: string, type: "target" | "requirement", action: ReviewAction) => void;
  reviewSaving?: boolean;
  reviewError?: string | null;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const CRITICALITY_ORDER = ["P0", "P1", "P2"] as const;
const CRITICALITY_COLORS: Record<string, string> = {
  P0: "text-red-400 border-red-500/30 bg-red-500/10",
  P1: "text-orange-400 border-orange-500/30 bg-orange-500/10",
  P2: "text-green-400 border-green-500/30 bg-green-500/10",
};
const CRITICALITY_LABELS: Record<string, string> = {
  P0: "P0 \u2013 Must ship",
  P1: "P1 \u2013 Should have",
  P2: "P2 \u2013 Future",
};

// ---------------------------------------------------------------------------
// RequirementGroupNode (recursive)
// ---------------------------------------------------------------------------

function RequirementGroupNode({
  group, depth = 0, selectedIds, onToggle, editable, reviewMode,
  onReviewAction, reviewSaving, reviewError,
  onCreateRequirement, onEditRequirement, onDeleteRequirement, onReorderRequirement,
  onEditModule, onDeleteModule,
}: {
  group: ArchiveRequirementGroup;
  depth?: number;
  selectedIds?: Set<string>;
  onToggle?: (id: string) => void;
  editable?: boolean;
  reviewMode?: boolean;
  onReviewAction?: (id: string, type: "target" | "requirement", action: ReviewAction) => void;
  reviewSaving?: boolean;
  reviewError?: string | null;
  onCreateRequirement?: (groupId: string) => void;
  onEditRequirement?: (groupId: string, requirement: ArchiveRequirement) => void;
  onDeleteRequirement?: (groupId: string, requirementId: string) => void;
  onReorderRequirement?: (groupId: string, requirementId: string, direction: "up" | "down") => void;
  onEditModule?: (groupId: string) => void;
  onDeleteModule?: (groupId: string) => void;
}) {
  const [expanded, setExpanded] = useState(depth === 0);
  const hasContent = group.requirements.length > 0 || group.children.length > 0;
  if (!hasContent && !editable) return null;

  const moduleActions: ActionsMenuItem[] = [];
  if (onCreateRequirement) moduleActions.push({ label: "Add Requirement", icon: <Plus className="h-3.5 w-3.5" />, onClick: () => onCreateRequirement(group.id) });
  if (onEditModule) moduleActions.push({ label: "Edit Module", icon: <Pencil className="h-3.5 w-3.5" />, onClick: () => onEditModule(group.id) });
  if (onDeleteModule) moduleActions.push({ label: "Delete Module", icon: <Trash2 className="h-3.5 w-3.5" />, onClick: () => onDeleteModule(group.id), destructive: true });

  return (
    <div className={depth > 0 ? "ml-4 border-l border-slate-700/50 pl-3" : ""}>
      <div className="flex items-center gap-1">
        <button type="button" onClick={() => setExpanded(!expanded)} className="flex flex-1 items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm font-medium text-slate-200 hover:bg-slate-800/50">
          {expanded ? <ChevronDown className="h-3.5 w-3.5 text-slate-500" /> : <ChevronRight className="h-3.5 w-3.5 text-slate-500" />}
          <span>{group.name}</span>
          <span className="text-xs text-slate-500">({group.requirements.length})</span>
        </button>
        {editable && !reviewMode && <ActionsMenu items={moduleActions} />}
      </div>
      {expanded && (
        <div className="mt-1 space-y-1">
          {group.requirements.map((req, idx) => {
            if (reviewMode && onReviewAction) {
              return (
                <ReviewCard
                  key={req.id}
                  item={req}
                  itemType="requirement"
                  currentStatus={getReviewStatus(req)}
                  comment={req.review_comment}
                  onApprove={() => onReviewAction(req.id, "requirement", { review_status: "approved" })}
                  onFlag={(c) => onReviewAction(req.id, "requirement", { review_status: "flagged", review_comment: c })}
                  onUnreview={() => onReviewAction(req.id, "requirement", { review_status: "unreviewed" })}
                  onEdit={() => onEditRequirement?.(group.id, req)}
                  onRemove={() => onDeleteRequirement?.(group.id, req.id)}
                  saving={reviewSaving}
                  error={reviewError}
                />
              );
            }

            const reqActions: ActionsMenuItem[] = [];
            if (onEditRequirement) reqActions.push({ label: "Edit", icon: <Pencil className="h-3.5 w-3.5" />, onClick: () => onEditRequirement(group.id, req) });
            if (onDeleteRequirement) reqActions.push({ label: "Delete", icon: <Trash2 className="h-3.5 w-3.5" />, onClick: () => onDeleteRequirement(group.id, req.id), destructive: true });
            if (onReorderRequirement && idx > 0) reqActions.push({ label: "Move Up", icon: <ArrowUp className="h-3.5 w-3.5" />, onClick: () => onReorderRequirement(group.id, req.id, "up") });
            if (onReorderRequirement && idx < group.requirements.length - 1) reqActions.push({ label: "Move Down", icon: <ArrowDown className="h-3.5 w-3.5" />, onClick: () => onReorderRequirement(group.id, req.id, "down") });

            const effectiveStatus = getReviewStatus(req);

            if (onToggle) {
              return (
                <button key={req.id} type="button" onClick={() => onToggle(req.id)} className={cn("group/req flex w-full items-start gap-2 rounded-md px-2 py-1.5 text-left text-sm hover:bg-slate-800/30 select-none", effectiveStatus === "flagged" && "border-l-2 border-l-amber-500/60")}>
                  <input type="checkbox" checked={selectedIds?.has(req.id) ?? false} readOnly tabIndex={-1} className="mt-0.5 h-4 w-4 accent-cyan-500 pointer-events-none" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-xs text-slate-500">{req.id}</span>
                      {req.category && <span className="rounded bg-slate-800 px-1.5 py-0.5 text-[10px] text-slate-400">{req.category}</span>}
                      <ReviewStatusIndicator reviewStatus={effectiveStatus} />
                    </div>
                    <p className="text-slate-300">{req.title}</p>
                    {req.description && <p className="mt-0.5 text-xs text-slate-500">{req.description}</p>}
                  </div>
                </button>
              );
            }

            return (
              <div key={req.id} className={cn("group/req flex items-start gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-slate-800/30", effectiveStatus === "flagged" && "border-l-2 border-l-amber-500/60")}>
                <StatusIcon status={req.status} />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-xs text-slate-500">{req.id}</span>
                    {req.category && <span className="rounded bg-slate-800 px-1.5 py-0.5 text-[10px] text-slate-400">{req.category}</span>}
                    <ReviewStatusIndicator reviewStatus={effectiveStatus} />
                  </div>
                  <p className="text-slate-300">{req.title}</p>
                  {req.description && <p className="mt-0.5 text-xs text-slate-500">{req.description}</p>}
                </div>
                {editable && <div className="opacity-0 group-hover/req:opacity-100"><ActionsMenu items={reqActions} /></div>}
              </div>
            );
          })}
          {group.children.map((child) => (
            <RequirementGroupNode
              key={child.id} group={child} depth={depth + 1} selectedIds={selectedIds} onToggle={onToggle}
              editable={editable} reviewMode={reviewMode} onReviewAction={onReviewAction} reviewSaving={reviewSaving} reviewError={reviewError}
              onCreateRequirement={onCreateRequirement} onEditRequirement={onEditRequirement} onDeleteRequirement={onDeleteRequirement}
              onReorderRequirement={onReorderRequirement} onEditModule={onEditModule} onDeleteModule={onDeleteModule}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function OperationalTargetsPanel({
  targets, requirements, editable = false,
  onCreateRequirement, onEditRequirement, onDeleteRequirement, onReorderRequirement,
  onCreateModule, onEditModule, onDeleteModule,
  onCreateTarget, onEditTarget, onDeleteTarget,
  onReviewAction, reviewSaving, reviewError,
}: OperationalTargetsPanelProps) {
  const selectedTargetIds = useBacklogDetailUIStore((s) => s.selectedTargetIds);
  const selectedRequirementIds = useBacklogDetailUIStore((s) => s.selectedRequirementIds);
  const reviewMode = useBacklogDetailUIStore((s) => s.reviewMode);
  const onToggleReviewMode = useBacklogDetailUIStore((s) => s.toggleReviewMode);
  const onTargetToggle = useBacklogDetailUIStore((s) => reviewMode ? undefined : s.toggleTargetId);
  const onRequirementToggle = useBacklogDetailUIStore((s) => reviewMode ? undefined : s.toggleRequirementId);

  const [targetsExpanded, setTargetsExpanded] = useState(true);
  const [requirementsExpanded, setRequirementsExpanded] = useState(true);

  const groupedTargets = useMemo(() => {
    const groups: Partial<Record<(typeof CRITICALITY_ORDER)[number] | "Other", ArchiveTarget[]>> =
      {};
    for (const level of CRITICALITY_ORDER) {
      groups[level] = targets.filter((t) => t.criticality === level);
    }
    const knownIds = new Set<string>(CRITICALITY_ORDER);
    const other = targets.filter((t) => !knownIds.has(t.criticality));
    if (other.length > 0) groups["Other"] = other;
    return groups;
  }, [targets]);

  const allRequirements = useMemo(
    () => countAllRequirements(requirements) as ArchiveRequirement[],
    [requirements],
  );

  const targetReviewStats = useMemo(() => {
    let reviewed = 0, flagged = 0;
    for (const t of targets) { const s = getReviewStatus(t); if (s !== "unreviewed") reviewed++; if (s === "flagged") flagged++; }
    return { reviewed, flagged, total: targets.length };
  }, [targets]);

  const reqReviewStats = useMemo(() => {
    let reviewed = 0, flagged = 0;
    for (const r of allRequirements) { const s = r.review_status ?? "unreviewed"; if (s !== "unreviewed") reviewed++; if (s === "flagged") flagged++; }
    return { reviewed, flagged, total: allRequirements.length };
  }, [allRequirements]);

  const hasTargets = targets.length > 0;
  const hasRequirements = requirements.length > 0;
  if (!hasTargets && !hasRequirements && !editable) return null;

  const reviewToggle = (
    <button
      type="button"
      onClick={onToggleReviewMode}
      className={cn(
        "ml-2 flex items-center gap-1 rounded px-2 py-1 text-xs transition-colors",
        reviewMode ? "bg-cyan-500/10 text-cyan-400 hover:bg-cyan-500/20" : "text-slate-400 hover:bg-slate-800/50 hover:text-slate-200",
      )}
      title={reviewMode ? "Exit review mode" : "Enter review mode"}
    >
      <ClipboardCheck className="h-3.5 w-3.5" />
      <span>Review</span>
    </button>
  );

  return (
    <div className="space-y-4">
      {reviewError && (
        <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">Review failed: {reviewError}</div>
      )}

      {/* Targets section */}
      {(hasTargets || editable) && (
        <div className="space-y-4 mt-4 border-t border-slate-800 pt-4">
            <div className="flex items-center border-b border-slate-800 pb-2">
              <button type="button" onClick={() => setTargetsExpanded(!targetsExpanded)} className="flex flex-1 items-center gap-2 text-left">
                {targetsExpanded ? <ChevronDown className="h-4 w-4 text-slate-400" /> : <ChevronRight className="h-4 w-4 text-slate-400" />}
                <Target className="h-4 w-4 text-slate-400" />
                <h2 className="text-base font-semibold text-slate-100">Operational Targets</h2>
                <span className="ml-auto text-xs text-slate-500">{targets.length} targets</span>
              </button>
              {reviewToggle}
              {editable && onCreateTarget && !reviewMode && (
                <button type="button" onClick={onCreateTarget} className="ml-2 flex items-center gap-1 rounded px-2 py-1 text-xs text-slate-400 hover:bg-slate-800/50 hover:text-slate-200" title="Add target">
                  <Plus className="h-3.5 w-3.5" /><span>Add Target</span>
                </button>
              )}
            </div>
            {targetReviewStats.reviewed > 0 && !reviewMode && <ReviewSummary {...targetReviewStats} />}
            {targetsExpanded && (
              <div className="space-y-4">
                {reviewMode && <ReviewProgressBar reviewed={targetReviewStats.reviewed} total={targetReviewStats.total} />}
                {CRITICALITY_ORDER.map((level) => {
                  const group = groupedTargets[level];
                  if (!group || group.length === 0) return null;
                  return (
                    <div key={level}>
                      <div className={`mb-2 inline-block rounded-full border px-2.5 py-0.5 text-xs font-medium ${CRITICALITY_COLORS[level] ?? "text-slate-400 border-slate-600 bg-slate-800"}`}>
                        {CRITICALITY_LABELS[level] ?? level}
                      </div>
                      <div className="space-y-1">
                        {group.map((target) => {
                          if (reviewMode && onReviewAction) {
                            return (
                              <ReviewCard key={target.id} item={target} itemType="target" currentStatus={getReviewStatus(target)} comment={target.review_comment}
                                onApprove={() => onReviewAction(target.id, "target", { review_status: "approved" })}
                                onFlag={(c) => onReviewAction(target.id, "target", { review_status: "flagged", review_comment: c })}
                                onUnreview={() => onReviewAction(target.id, "target", { review_status: "unreviewed" })}
                                onEdit={() => onEditTarget?.(target)} onRemove={() => onDeleteTarget?.(target.id)}
                                saving={reviewSaving} error={reviewError}
                              />
                            );
                          }

                          const targetActions: ActionsMenuItem[] = [];
                          if (onEditTarget) targetActions.push({ label: "Edit Target", icon: <Pencil className="h-3.5 w-3.5" />, onClick: () => onEditTarget(target) });
                          if (onDeleteTarget) targetActions.push({ label: "Delete Target", icon: <Trash2 className="h-3.5 w-3.5" />, onClick: () => onDeleteTarget(target.id), destructive: true });

                          const effectiveStatus = getReviewStatus(target);

                          if (onTargetToggle) {
                            return (
                              <button key={target.id} type="button" onClick={() => onTargetToggle(target.id)} className={cn("group/target flex w-full items-start gap-2 rounded-md px-2 py-1.5 text-left text-sm hover:bg-slate-800/30 select-none", effectiveStatus === "flagged" && "border-l-2 border-l-amber-500/60")}>
                                <input type="checkbox" checked={selectedTargetIds?.has(target.id) ?? false} readOnly tabIndex={-1} className="mt-0.5 h-4 w-4 accent-cyan-500 pointer-events-none" />
                                <div className="min-w-0 flex-1">
                                  <div className="flex items-center gap-2">
                                    <span className="font-mono text-xs text-slate-500">{target.id}</span>
                                    <ReviewStatusIndicator reviewStatus={effectiveStatus} />
                                  </div>
                                  <p className="text-slate-300">{target.title}</p>
                                  {target.notes && <p className="mt-0.5 text-xs text-slate-500">{target.notes}</p>}
                                  {target.linked_requirement_ids.length > 0 && (
                                    <div className="mt-1 flex flex-wrap gap-1">
                                      {target.linked_requirement_ids.map((reqId) => <span key={reqId} className="rounded bg-cyan-500/10 px-1.5 py-0.5 text-[10px] text-cyan-400">{reqId}</span>)}
                                    </div>
                                  )}
                                </div>
                              </button>
                            );
                          }

                          return (
                            <div key={target.id} className={cn("group/target flex items-start gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-slate-800/30", effectiveStatus === "flagged" && "border-l-2 border-l-amber-500/60")}>
                              <StatusIcon status={target.status} />
                              <div className="min-w-0 flex-1">
                                <div className="flex items-center gap-2">
                                  <span className="font-mono text-xs text-slate-500">{target.id}</span>
                                  <ReviewStatusIndicator reviewStatus={effectiveStatus} />
                                </div>
                                <p className="text-slate-300">{target.title}</p>
                                {target.notes && <p className="mt-0.5 text-xs text-slate-500">{target.notes}</p>}
                                {target.linked_requirement_ids.length > 0 && (
                                  <div className="mt-1 flex flex-wrap gap-1">
                                    {target.linked_requirement_ids.map((reqId) => <span key={reqId} className="rounded bg-cyan-500/10 px-1.5 py-0.5 text-[10px] text-cyan-400">{reqId}</span>)}
                                  </div>
                                )}
                              </div>
                              {editable && targetActions.length > 0 && <div className="opacity-0 group-hover/target:opacity-100"><ActionsMenu items={targetActions} /></div>}
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
        </div>
      )}

      {/* Requirements section */}
      {(hasRequirements || editable) && (
        <div className="space-y-3 mt-4 border-t border-slate-800 pt-4">
            <div className="flex items-center border-b border-slate-800 pb-2">
              <button type="button" onClick={() => setRequirementsExpanded(!requirementsExpanded)} className="flex flex-1 items-center gap-2 text-left">
                {requirementsExpanded ? <ChevronDown className="h-4 w-4 text-slate-400" /> : <ChevronRight className="h-4 w-4 text-slate-400" />}
                <FileCheck className="h-4 w-4 text-slate-400" />
                <h2 className="text-base font-semibold text-slate-100">Requirements</h2>
              </button>
              {editable && onCreateModule && !reviewMode && (
                <button type="button" onClick={onCreateModule} className="flex items-center gap-1 rounded px-2 py-1 text-xs text-slate-400 hover:bg-slate-800/50 hover:text-slate-200" title="Add module">
                  <Plus className="h-3.5 w-3.5" /><span>Add Module</span>
                </button>
              )}
            </div>
            {reqReviewStats.reviewed > 0 && !reviewMode && <ReviewSummary {...reqReviewStats} />}
            {requirementsExpanded && (
              <div className="space-y-1">
                {reviewMode && <ReviewProgressBar reviewed={reqReviewStats.reviewed} total={reqReviewStats.total} />}
                {requirements.map((group) => (
                  <RequirementGroupNode
                    key={group.id} group={group} selectedIds={selectedRequirementIds}
                    onToggle={reviewMode ? undefined : onRequirementToggle}
                    editable={editable} reviewMode={reviewMode} onReviewAction={onReviewAction}
                    reviewSaving={reviewSaving} reviewError={reviewError}
                    onCreateRequirement={onCreateRequirement} onEditRequirement={onEditRequirement}
                    onDeleteRequirement={onDeleteRequirement} onReorderRequirement={onReorderRequirement}
                    onEditModule={onEditModule} onDeleteModule={onDeleteModule}
                  />
                ))}
              </div>
            )}
        </div>
      )}
    </div>
  );
}
