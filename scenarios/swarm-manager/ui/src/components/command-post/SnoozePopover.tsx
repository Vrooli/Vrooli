/**
 * SnoozePopover — Popover with snooze preset buttons.
 *
 * Uses portal-based Popover to escape parent overflow constraints.
 * Shows 1h, 4h, and Tomorrow presets.
 */

import { useState, useRef, useCallback } from "react";
import { Clock } from "lucide-react";
import { SNOOZE_PRESETS, getPresetExpiry } from "../../lib/snooze-utils";
import { Popover } from "../ui/popover";

interface SnoozePopoverProps {
  itemKey: string;
  onSnooze: (key: string, expiresAt: number) => void;
  children: React.ReactNode;
}

export function SnoozePopover({ itemKey, onSnooze, children }: SnoozePopoverProps) {
  const [open, setOpen] = useState(false);
  const [popoverPos, setPopoverPos] = useState({ x: 0, y: 0 });
  const buttonRef = useRef<HTMLButtonElement>(null);

  const handleOpen = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    const rect = buttonRef.current?.getBoundingClientRect();
    if (rect) {
      setPopoverPos({ x: rect.left, y: rect.bottom + 4 });
    }
    setOpen((prev) => !prev);
  }, []);

  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        onClick={handleOpen}
        className="rounded p-1 text-slate-400 transition-colors hover:bg-slate-700/50 hover:text-slate-200"
        aria-label="Snooze"
        data-testid="snooze-trigger"
      >
        {children}
      </button>

      <Popover
        isOpen={open}
        onClose={() => setOpen(false)}
        x={popoverPos.x}
        y={popoverPos.y}
        className="w-40 p-1"
        testId="snooze-popover"
      >
        <p className="px-2 py-1 text-xs font-medium text-slate-400">Snooze for...</p>
        {SNOOZE_PRESETS.map((preset) => (
          <button
            key={preset.label}
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onSnooze(itemKey, getPresetExpiry(preset));
              setOpen(false);
            }}
            className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-sm text-slate-200 hover:bg-slate-700"
            data-testid={`snooze-preset-${preset.label.toLowerCase().replace(/\s/g, "-")}`}
          >
            <Clock className="h-3.5 w-3.5 text-slate-400" />
            {preset.label}
          </button>
        ))}
      </Popover>
    </>
  );
}
