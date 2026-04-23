/**
 * Dependency Chip List
 *
 * Renders a labeled list of dependency rows on the Backlog Details page.
 * Each row shows:
 *
 *   [title button]  [status chip]  [activity chip | attention chips]
 *
 * Clicking the title navigates to that backlog item. The status chip carries
 * the lifecycle color (the same palette used by graph nodes and the Initiative
 * Details row) and is itself clickable for inline status changes when
 * `onStatusChange` is provided.
 *
 * The third slot is conditional and resolves in priority order:
 *   1. If an agent is actively running on the dependency, show a pulsing
 *      purpose-specific chip ("Workshopping" / "Reviewing" / …).
 *   2. Otherwise, if the item has pending-input reasons (unanswered decisions,
 *      plan ready, review ready), show those as small badges.
 *   3. Otherwise, render nothing.
 */

import { memo, useState, useCallback, useRef } from "react";
import { Check } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { ResolvedDependency } from "../../lib/backlog-queue-utils";
import {
  getAgentActivityLabel,
  getAgentActivityTone,
  type AgentActivityTone,
} from "../../lib/agent-activity-utils";
import { getStatusColorClasses } from "../../surfaces/graph/lib/status-colors";
import {
  BACKLOG_STATUS_COLORS,
  USER_SETTABLE_STATUSES,
  formatBacklogStatus,
  type BacklogStatus,
} from "../../types";
import { Popover } from "../ui/popover";
import { StatusChip, type StatusChipColors } from "../ui/status-chip";
import { PendingDecisionBadge } from "./pending-decision-badge";
import { useDetailSelectionStore } from "../../stores/detail-selection-store";

interface DependencyChipListProps {
  label: string;
  items: ResolvedDependency[];
  icon: LucideIcon;
  /** When provided, the status chip on each row becomes a click target that
   *  opens a popover for inline status changes. */
  onStatusChange?: (dep: ResolvedDependency, newStatus: BacklogStatus) => void;
}

// Chip color palettes for the activity slot.
const ACTIVITY_COLORS: Record<AgentActivityTone, StatusChipColors> = {
  busy: {
    background: "bg-amber-500/15",
    border: "border-amber-400/40",
    text: "text-amber-300",
    dot: "bg-amber-400",
  },
  "needs-review": {
    background: "bg-cyan-500/15",
    border: "border-cyan-400/40",
    text: "text-cyan-300",
    dot: "bg-cyan-400",
  },
};

function statusChipColors(status: BacklogStatus): StatusChipColors {
  const base = getStatusColorClasses(status);
  return {
    background: base.background,
    border: base.border,
    text: base.text,
    dot: BACKLOG_STATUS_COLORS[status] ?? "bg-slate-500",
  };
}

function StatusChipWithPopover({
  dep,
  onStatusChange,
}: {
  dep: ResolvedDependency;
  onStatusChange: (dep: ResolvedDependency, newStatus: BacklogStatus) => void;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const [popoverPos, setPopoverPos] = useState({ x: 0, y: 0 });
  const triggerRef = useRef<HTMLElement | null>(null);

  const handleOpen = useCallback((e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    triggerRef.current = e.currentTarget;
    const rect = e.currentTarget.getBoundingClientRect();
    setPopoverPos({ x: rect.left, y: rect.bottom + 4 });
    setIsOpen(true);
  }, []);

  const handleSelect = useCallback(
    (status: BacklogStatus, e: React.MouseEvent) => {
      e.stopPropagation();
      e.preventDefault();
      setIsOpen(false);
      if (status !== dep.status) {
        onStatusChange(dep, status);
      }
    },
    [dep, onStatusChange],
  );

  return (
    <>
      <StatusChip
        label={formatBacklogStatus(dep.status)}
        colors={statusChipColors(dep.status)}
        leadingDot
        onClick={handleOpen}
        title={`Change status (${formatBacklogStatus(dep.status)})`}
        data-testid={`dep-status-dot-${dep.kind}-${dep.name}`}
      />

      <Popover
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        x={popoverPos.x}
        y={popoverPos.y}
        className="min-w-[160px] py-1"
        testId={`dep-status-popover-${dep.kind}-${dep.name}`}
      >
        {USER_SETTABLE_STATUSES.map((s) => (
          <button
            key={s}
            type="button"
            onClick={(e) => handleSelect(s, e)}
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs uppercase tracking-wider text-slate-300 transition-colors hover:bg-slate-800"
            data-testid={`dep-status-option-${s}`}
          >
            <span
              className={`inline-block h-2 w-2 rounded-full ${BACKLOG_STATUS_COLORS[s] ?? "bg-slate-500"}`}
            />
            <span className="flex-1">{formatBacklogStatus(s)}</span>
            {s === dep.status ? (
              <Check className="h-3 w-3 text-slate-400" />
            ) : null}
          </button>
        ))}
      </Popover>
    </>
  );
}

/**
 * The activity chip is suppressed when it would be redundant with the
 * lifecycle status chip — specifically when the item is `in_progress` and
 * the active agent is actually executing it. In every other case, it adds
 * information the status chip doesn't convey (what the agent is doing,
 * whether it's waiting for the user).
 */
function shouldShowActivity(dep: ResolvedDependency): boolean {
  if (!dep.activity) return false;
  if (dep.status === "in_progress" && dep.activity.purpose === "process") return false;
  return true;
}

function DependencyRow({
  dep,
  onStatusChange,
}: {
  dep: ResolvedDependency;
  onStatusChange?: (dep: ResolvedDependency, newStatus: BacklogStatus) => void;
}) {
  const selectBacklog = useDetailSelectionStore((s) => s.selectBacklog);

  const showActivity = shouldShowActivity(dep);
  const attentionReasons = dep.attentionReasons ?? [];
  const showAttention = !showActivity && attentionReasons.length > 0;

  return (
    <div
      className="group flex flex-wrap items-center gap-2 rounded-md px-1.5 py-1 transition-colors hover:bg-slate-800/60"
      data-testid={`dep-row-${dep.kind}-${dep.name}`}
    >
      <button
        type="button"
        onClick={() => selectBacklog(dep.kind, dep.name)}
        className="min-w-0 flex-1 truncate text-left text-sm text-slate-200 transition-colors hover:text-cyan-300"
        title={dep.title}
      >
        {dep.title}
      </button>
      <div className="flex shrink-0 flex-wrap items-center gap-1">
        {onStatusChange ? (
          <StatusChipWithPopover dep={dep} onStatusChange={onStatusChange} />
        ) : (
          <StatusChip
            label={formatBacklogStatus(dep.status)}
            colors={statusChipColors(dep.status)}
            leadingDot
            title={formatBacklogStatus(dep.status)}
            data-testid={`dep-status-chip-${dep.kind}-${dep.name}`}
          />
        )}

        {showActivity && dep.activity ? (
          <StatusChip
            label={getAgentActivityLabel(dep.activity.purpose)}
            colors={ACTIVITY_COLORS[getAgentActivityTone(dep.activity.status)]}
            leadingDot
            pulse={getAgentActivityTone(dep.activity.status) === "busy"}
            title={
              dep.activity.status === "needs_review"
                ? "Awaiting review — your turn"
                : "An agent is running on this item"
            }
            data-testid={`dep-activity-chip-${dep.kind}-${dep.name}`}
          />
        ) : null}

        {showAttention ? (
          <PendingDecisionBadge reasons={attentionReasons} />
        ) : null}
      </div>
    </div>
  );
}

export const DependencyChipList = memo(function DependencyChipList({
  label,
  items,
  icon: Icon,
  onStatusChange,
}: DependencyChipListProps) {
  if (items.length === 0) return null;

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
        <Icon className="h-3.5 w-3.5" />
        {label}
      </div>
      <div className="flex flex-col gap-0.5">
        {items.map((dep) => (
          <DependencyRow
            key={`${dep.kind}/${dep.name}`}
            dep={dep}
            onStatusChange={onStatusChange}
          />
        ))}
      </div>
    </div>
  );
});
