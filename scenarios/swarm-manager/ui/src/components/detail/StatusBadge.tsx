/**
 * StatusBadge
 *
 * Inline status chip for detail pages. Uses the unified status-colors
 * mapping so status colors are consistent between graph nodes and detail views.
 *
 * When `onStatusChange` is provided, the badge becomes a clickable button
 * that opens a popover with user-settable status options.
 */

import { useState, useCallback, useRef } from "react";
import { Check, Loader2 } from "lucide-react";
import { cn } from "../../lib/utils";
import { getStatusColorClasses } from "../../surfaces/graph/lib/status-colors";
import { Popover } from "../ui/popover";
import {
  BACKLOG_STATUS_COLORS,
  USER_SETTABLE_STATUSES,
  formatBacklogStatus,
} from "../../types";
import type { BacklogStatus } from "../../types";

interface StatusBadgeProps {
  status: string;
  size?: "sm" | "md";
  className?: string;
  /** When provided, the badge becomes clickable and opens a status-change popover. */
  onStatusChange?: (newStatus: BacklogStatus) => void;
  /** Show a spinner when a status change is in flight. */
  statusChangePending?: boolean;
}

export function StatusBadge({ status, size = "sm", className, onStatusChange, statusChangePending }: StatusBadgeProps) {
  const colors = getStatusColorClasses(status);
  const [isOpen, setIsOpen] = useState(false);
  const [popoverPos, setPopoverPos] = useState({ x: 0, y: 0 });
  const buttonRef = useRef<HTMLButtonElement>(null);

  const handleOpen = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      e.preventDefault();
      if (statusChangePending) return;
      const rect = buttonRef.current?.getBoundingClientRect();
      if (rect) {
        setPopoverPos({ x: rect.left, y: rect.bottom + 4 });
      }
      setIsOpen(true);
    },
    [statusChangePending],
  );

  const handleSelect = useCallback(
    (newStatus: BacklogStatus, e: React.MouseEvent) => {
      e.stopPropagation();
      e.preventDefault();
      setIsOpen(false);
      if (newStatus !== status) {
        onStatusChange?.(newStatus);
      }
    },
    [status, onStatusChange],
  );

  const badgeClasses = cn(
    "inline-flex items-center rounded-full border font-medium",
    colors.background,
    colors.border,
    colors.text,
    size === "sm" && "px-2.5 py-0.5 text-xs",
    size === "md" && "px-3 py-1 text-sm",
    onStatusChange && "cursor-pointer transition-colors hover:brightness-125",
    className,
  );

  const label = status.replace(/_/g, " ");
  const isInReview = status === "in_review";
  // Leading pulsing dot when the review agent is running; signals "busy —
  // check back when it lands in review_pending".
  const pulseDot = isInReview ? (
    <span className="relative mr-1.5 inline-flex h-1.5 w-1.5 shrink-0" aria-hidden>
      <span className={cn("absolute inline-flex h-full w-full animate-ping rounded-full opacity-75", colors.text)} />
      <span className={cn("relative inline-flex h-1.5 w-1.5 rounded-full", colors.text)} />
    </span>
  ) : null;

  if (!onStatusChange) {
    return (
      <span className={badgeClasses} data-testid="status-badge">
        {pulseDot}
        {label}
      </span>
    );
  }

  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        onClick={handleOpen}
        className={badgeClasses}
        data-testid="status-badge"
      >
        {statusChangePending ? (
          <Loader2 className="mr-1 h-3 w-3 animate-spin" />
        ) : pulseDot}
        {label}
      </button>

      <Popover
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        x={popoverPos.x}
        y={popoverPos.y}
        className="min-w-[160px] py-1"
        testId="status-badge-popover"
      >
        {USER_SETTABLE_STATUSES.map((s) => (
          <button
            key={s}
            type="button"
            onClick={(e) => handleSelect(s, e)}
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs uppercase tracking-wider text-slate-300 transition-colors hover:bg-slate-800"
            data-testid={`status-badge-option-${s}`}
          >
            <span
              className={`inline-block h-2 w-2 rounded-full ${BACKLOG_STATUS_COLORS[s] ?? "bg-slate-500"}`}
            />
            <span className="flex-1">{formatBacklogStatus(s)}</span>
            {s === status ? (
              <Check className="h-3 w-3 text-slate-400" />
            ) : null}
          </button>
        ))}
      </Popover>
    </>
  );
}
