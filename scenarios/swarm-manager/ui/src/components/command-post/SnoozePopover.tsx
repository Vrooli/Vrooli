/**
 * SnoozePopover — Popover with snooze preset buttons.
 *
 * Positions relative to its trigger child. Shows 1h, 4h, and Tomorrow presets.
 */

import { useState, useRef, useEffect, useCallback } from "react";
import { Clock } from "lucide-react";
import { SNOOZE_PRESETS, getPresetExpiry } from "../../lib/snooze-utils";

interface SnoozePopoverProps {
  itemKey: string;
  onSnooze: (key: string, expiresAt: number) => void;
  children: React.ReactNode;
}

export function SnoozePopover({ itemKey, onSnooze, children }: SnoozePopoverProps) {
  const [open, setOpen] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);

  const handleClickOutside = useCallback((e: MouseEvent) => {
    if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
      setOpen(false);
    }
  }, []);

  useEffect(() => {
    if (open) {
      document.addEventListener("mousedown", handleClickOutside);
      return () => document.removeEventListener("mousedown", handleClickOutside);
    }
  }, [open, handleClickOutside]);

  return (
    <div className="relative" ref={popoverRef}>
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          setOpen((prev) => !prev);
        }}
        className="rounded p-1 text-slate-400 transition-colors hover:bg-slate-700/50 hover:text-slate-200"
        aria-label="Snooze"
        data-testid="snooze-trigger"
      >
        {children}
      </button>

      {open && (
        <div
          className="absolute right-0 top-full z-50 mt-1 w-40 rounded-lg border border-slate-700 bg-slate-800 p-1 shadow-lg"
          data-testid="snooze-popover"
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
        </div>
      )}
    </div>
  );
}
