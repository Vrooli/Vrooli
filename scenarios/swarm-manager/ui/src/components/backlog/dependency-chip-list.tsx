/**
 * Dependency Chip List
 *
 * Renders a labeled list of clickable, status-colored chips for parent
 * (upstream) or children (downstream) dependencies. Each chip navigates
 * to the dependency's backlog details page.
 *
 * When `onStatusChange` is provided, each chip also renders a small
 * clickable status dot that opens a popover for inline status changes.
 */

import { memo, useState, useCallback, useRef } from "react";
import { Check } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { ResolvedDependency } from "../../lib/backlog-queue-utils";
import {
  BACKLOG_STATUS_CHIP_COLORS,
  BACKLOG_STATUS_COLORS,
  USER_SETTABLE_STATUSES,
  formatBacklogStatus,
  type BacklogStatus,
} from "../../types";
import { Popover } from "../ui/popover";
import { useDetailSelectionStore } from "../../stores/detail-selection-store";

interface DependencyChipListProps {
  label: string;
  items: ResolvedDependency[];
  icon: LucideIcon;
  /** When provided, each chip gets a clickable status dot for inline status changes. */
  onStatusChange?: (dep: ResolvedDependency, newStatus: BacklogStatus) => void;
}

function StatusDot({
  dep,
  onStatusChange,
}: {
  dep: ResolvedDependency;
  onStatusChange: (dep: ResolvedDependency, newStatus: BacklogStatus) => void;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const [popoverPos, setPopoverPos] = useState({ x: 0, y: 0 });
  const dotRef = useRef<HTMLButtonElement>(null);

  const handleOpen = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      e.preventDefault();
      const rect = dotRef.current?.getBoundingClientRect();
      if (rect) {
        setPopoverPos({ x: rect.left, y: rect.bottom + 4 });
      }
      setIsOpen(true);
    },
    [],
  );

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
      <button
        ref={dotRef}
        type="button"
        onClick={handleOpen}
        className="mr-1.5 inline-block h-2 w-2 shrink-0 rounded-full transition-transform hover:scale-150 cursor-pointer"
        style={{ backgroundColor: "currentColor" }}
        title={`Change status (${formatBacklogStatus(dep.status)})`}
        data-testid={`dep-status-dot-${dep.kind}-${dep.name}`}
      >
        <span className={`block h-full w-full rounded-full ${BACKLOG_STATUS_COLORS[dep.status] ?? "bg-slate-500"}`} />
      </button>

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

export const DependencyChipList = memo(function DependencyChipList({
  label,
  items,
  icon: Icon,
  onStatusChange,
}: DependencyChipListProps) {
  const selectBacklog = useDetailSelectionStore((s) => s.selectBacklog);

  if (items.length === 0) return null;

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
        <Icon className="h-3.5 w-3.5" />
        {label}
      </div>
      <div className="flex flex-wrap gap-1.5">
        {items.map((dep) => (
          <button
            key={`${dep.kind}/${dep.name}`}
            type="button"
            onClick={() => selectBacklog(dep.kind, dep.name)}
            title={formatBacklogStatus(dep.status)}
            className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium transition-colors hover:brightness-125 ${BACKLOG_STATUS_CHIP_COLORS[dep.status]}`}
          >
            {onStatusChange ? (
              <StatusDot dep={dep} onStatusChange={onStatusChange} />
            ) : null}
            {dep.title}
          </button>
        ))}
      </div>
    </div>
  );
});
