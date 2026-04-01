/**
 * StatusChipPopover — clickable status indicator that opens a popover
 * for inline status changes. Used in BacklogCard.
 */

import { useState, useCallback, useRef } from "react";
import { Check, Loader2 } from "lucide-react";
import { Popover } from "../ui/popover";
import {
  BACKLOG_STATUS_COLORS,
  USER_SETTABLE_STATUSES,
  formatBacklogStatus,
} from "../../types";
import type { BacklogStatus } from "../../types";

interface StatusChipPopoverProps {
  currentStatus: BacklogStatus;
  onStatusChange: (newStatus: BacklogStatus) => void;
  pending?: boolean;
}

export function StatusChipPopover({
  currentStatus,
  onStatusChange,
  pending,
}: StatusChipPopoverProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [popoverPos, setPopoverPos] = useState({ x: 0, y: 0 });
  const buttonRef = useRef<HTMLButtonElement>(null);

  const handleOpen = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      e.preventDefault();
      if (pending) return;
      const rect = buttonRef.current?.getBoundingClientRect();
      if (rect) {
        setPopoverPos({ x: rect.left, y: rect.bottom + 4 });
      }
      setIsOpen(true);
    },
    [pending],
  );

  const handleSelect = useCallback(
    (status: BacklogStatus, e: React.MouseEvent) => {
      e.stopPropagation();
      e.preventDefault();
      setIsOpen(false);
      if (status !== currentStatus) {
        onStatusChange(status);
      }
    },
    [currentStatus, onStatusChange],
  );

  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        onClick={handleOpen}
        className="inline-flex items-center gap-2 rounded-md px-1.5 py-0.5 -mx-1.5 -my-0.5 transition-colors hover:bg-slate-800 cursor-pointer"
        data-testid="status-chip-trigger"
      >
        <span
          className={`inline-block h-2 w-2 rounded-full ${BACKLOG_STATUS_COLORS[currentStatus] ?? "bg-slate-500"}`}
        />
        {pending ? (
          <Loader2 className="h-3 w-3 animate-spin text-slate-400" />
        ) : null}
        <span className="text-xs uppercase tracking-wider text-slate-400">
          {formatBacklogStatus(currentStatus)}
        </span>
      </button>

      <Popover
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        x={popoverPos.x}
        y={popoverPos.y}
        className="min-w-[160px] py-1"
        testId="status-chip-popover"
      >
        {USER_SETTABLE_STATUSES.map((status) => (
          <button
            key={status}
            type="button"
            onClick={(e) => handleSelect(status, e)}
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs uppercase tracking-wider text-slate-300 transition-colors hover:bg-slate-800"
            data-testid={`status-option-${status}`}
          >
            <span
              className={`inline-block h-2 w-2 rounded-full ${BACKLOG_STATUS_COLORS[status] ?? "bg-slate-500"}`}
            />
            <span className="flex-1">{formatBacklogStatus(status)}</span>
            {status === currentStatus ? (
              <Check className="h-3 w-3 text-slate-400" />
            ) : null}
          </button>
        ))}
      </Popover>
    </>
  );
}
