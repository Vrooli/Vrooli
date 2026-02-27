import { useState } from "react";
import { ChevronDown, ChevronRight, Target, FileCheck, CheckCircle2, Circle } from "lucide-react";
import { Card } from "../ui/card";
import type { ArchiveTarget, ArchiveRequirement, ArchiveRequirementGroup } from "../../types";

interface OperationalTargetsPanelProps {
  targets: ArchiveTarget[];
  requirements: ArchiveRequirementGroup[];
  selectedTargetIds?: Set<string>;
  selectedRequirementIds?: Set<string>;
  onTargetToggle?: (id: string) => void;
  onRequirementToggle?: (id: string) => void;
}

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

function StatusIcon({ status }: { status: string }) {
  if (status === "complete") {
    return <CheckCircle2 className="h-4 w-4 text-green-400" />;
  }
  return <Circle className="h-4 w-4 text-slate-500" />;
}

function RequirementGroupNode({
  group,
  depth = 0,
  selectedIds,
  onToggle,
}: {
  group: ArchiveRequirementGroup;
  depth?: number;
  selectedIds?: Set<string>;
  onToggle?: (id: string) => void;
}) {
  const [expanded, setExpanded] = useState(depth === 0);
  const hasContent = group.requirements.length > 0 || group.children.length > 0;

  if (!hasContent) return null;

  return (
    <div className={depth > 0 ? "ml-4 border-l border-slate-700/50 pl-3" : ""}>
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm font-medium text-slate-200 hover:bg-slate-800/50"
      >
        {expanded ? (
          <ChevronDown className="h-3.5 w-3.5 text-slate-500" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5 text-slate-500" />
        )}
        <span>{group.name}</span>
        <span className="text-xs text-slate-500">({group.requirements.length})</span>
      </button>
      {expanded && (
        <div className="mt-1 space-y-1">
          {group.requirements.map((req) => {
            const Wrapper = onToggle ? "label" : "div";
            return (
              <Wrapper
                key={req.id}
                className={`flex items-start gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-slate-800/30${onToggle ? " cursor-pointer" : ""}`}
              >
                {onToggle ? (
                  <input
                    type="checkbox"
                    checked={selectedIds?.has(req.id) ?? false}
                    onChange={() => onToggle(req.id)}
                    className="mt-0.5 h-4 w-4 accent-cyan-500"
                  />
                ) : (
                  <StatusIcon status={req.status} />
                )}
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-xs text-slate-500">{req.id}</span>
                    {req.category && (
                      <span className="rounded bg-slate-800 px-1.5 py-0.5 text-[10px] text-slate-400">{req.category}</span>
                    )}
                  </div>
                  <p className="text-slate-300">{req.title}</p>
                  {req.description && (
                    <p className="mt-0.5 text-xs text-slate-500">{req.description}</p>
                  )}
                </div>
              </Wrapper>
            );
          })}
          {group.children.map((child) => (
            <RequirementGroupNode key={child.id} group={child} depth={depth + 1} selectedIds={selectedIds} onToggle={onToggle} />
          ))}
        </div>
      )}
    </div>
  );
}

export function OperationalTargetsPanel({
  targets,
  requirements,
  selectedTargetIds,
  selectedRequirementIds,
  onTargetToggle,
  onRequirementToggle,
}: OperationalTargetsPanelProps) {
  const [targetsExpanded, setTargetsExpanded] = useState(true);
  const [requirementsExpanded, setRequirementsExpanded] = useState(true);

  const groupedTargets = CRITICALITY_ORDER.reduce<Record<string, ArchiveTarget[]>>((acc, level) => {
    acc[level] = targets.filter((t) => t.criticality === level);
    return acc;
  }, {} as Record<string, ArchiveTarget[]>);

  // Also include any targets with unknown criticality
  const knownIds = new Set<string>(CRITICALITY_ORDER);
  const otherTargets = targets.filter((t) => !knownIds.has(t.criticality));
  if (otherTargets.length > 0) {
    groupedTargets["Other"] = otherTargets;
  }

  const hasTargets = targets.length > 0;
  const hasRequirements = requirements.length > 0;

  if (!hasTargets && !hasRequirements) return null;

  return (
    <div className="space-y-4">
      {hasTargets && (
        <Card padding="sm" className="rounded-lg border-slate-700/60 bg-slate-900/45">
          <div className="space-y-4">
            <button
              type="button"
              onClick={() => setTargetsExpanded(!targetsExpanded)}
              className="flex w-full items-center gap-2 border-b border-slate-800 pb-2 text-left"
            >
              {targetsExpanded ? (
                <ChevronDown className="h-4 w-4 text-slate-400" />
              ) : (
                <ChevronRight className="h-4 w-4 text-slate-400" />
              )}
              <Target className="h-4 w-4 text-slate-400" />
              <h2 className="text-base font-semibold text-slate-100">Operational Targets</h2>
              <span className="ml-auto text-xs text-slate-500">{targets.length} targets</span>
            </button>
            {targetsExpanded && (
              <div className="space-y-4">
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
                          const TargetWrapper = onTargetToggle ? "label" : "div";
                          return (
                            <TargetWrapper
                              key={target.id}
                              className={`flex items-start gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-slate-800/30${onTargetToggle ? " cursor-pointer" : ""}`}
                            >
                              {onTargetToggle ? (
                                <input
                                  type="checkbox"
                                  checked={selectedTargetIds?.has(target.id) ?? false}
                                  onChange={() => onTargetToggle(target.id)}
                                  className="mt-0.5 h-4 w-4 accent-cyan-500"
                                />
                              ) : (
                                <StatusIcon status={target.status} />
                              )}
                              <div className="min-w-0 flex-1">
                                <div className="flex items-center gap-2">
                                  <span className="font-mono text-xs text-slate-500">{target.id}</span>
                                </div>
                                <p className="text-slate-300">{target.title}</p>
                                {target.notes && (
                                  <p className="mt-0.5 text-xs text-slate-500">{target.notes}</p>
                                )}
                                {target.linked_requirement_ids.length > 0 && (
                                  <div className="mt-1 flex flex-wrap gap-1">
                                    {target.linked_requirement_ids.map((reqId) => (
                                      <span
                                        key={reqId}
                                        className="rounded bg-cyan-500/10 px-1.5 py-0.5 text-[10px] text-cyan-400"
                                      >
                                        {reqId}
                                      </span>
                                    ))}
                                  </div>
                                )}
                              </div>
                            </TargetWrapper>
                          );
                        })}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </Card>
      )}

      {hasRequirements && (
        <Card padding="sm" className="rounded-lg border-slate-700/60 bg-slate-900/45">
          <div className="space-y-3">
            <button
              type="button"
              onClick={() => setRequirementsExpanded(!requirementsExpanded)}
              className="flex w-full items-center gap-2 border-b border-slate-800 pb-2 text-left"
            >
              {requirementsExpanded ? (
                <ChevronDown className="h-4 w-4 text-slate-400" />
              ) : (
                <ChevronRight className="h-4 w-4 text-slate-400" />
              )}
              <FileCheck className="h-4 w-4 text-slate-400" />
              <h2 className="text-base font-semibold text-slate-100">Requirements</h2>
            </button>
            {requirementsExpanded && (
              <div className="space-y-1">
                {requirements.map((group) => (
                  <RequirementGroupNode
                    key={group.id}
                    group={group}
                    selectedIds={selectedRequirementIds}
                    onToggle={onRequirementToggle}
                  />
                ))}
              </div>
            )}
          </div>
        </Card>
      )}
    </div>
  );
}
