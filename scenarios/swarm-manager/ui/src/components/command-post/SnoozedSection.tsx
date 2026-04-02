/**
 * SnoozedSection — Collapsible section showing snoozed items with unsnooze buttons.
 *
 * Displays "Snoozed (N)" header, expands to show item list with relative expiry times.
 */

import { useState } from "react";
import { ChevronRight, Clock, X } from "lucide-react";
import { cn } from "../../lib/utils";
import type { SnoozeEntry } from "../../lib/snooze-utils";

interface SnoozedSectionProps {
  entries: SnoozeEntry[];
  onUnsnooze: (key: string) => void;
}

function formatRelativeExpiry(expiresAt: number): string {
  const remaining = expiresAt - Date.now();
  if (remaining <= 0) return "expiring...";
  const hours = Math.floor(remaining / 3_600_000);
  const minutes = Math.floor((remaining % 3_600_000) / 60_000);
  if (hours > 0) return `unsnoozes in ${hours}h ${minutes}m`;
  return `unsnoozes in ${minutes}m`;
}

function formatKeyTitle(key: string): string {
  // key format: "backlog:kind/name" or "execution:id" or "capture:id"
  const colonIdx = key.indexOf(":");
  if (colonIdx === -1) return key;
  const value = key.slice(colonIdx + 1);
  const slashIdx = value.indexOf("/");
  return slashIdx !== -1 ? value.slice(slashIdx + 1) : value;
}

export function SnoozedSection({ entries, onUnsnooze }: SnoozedSectionProps) {
  const [expanded, setExpanded] = useState(false);

  if (entries.length === 0) return null;

  return (
    <div data-testid="snoozed-section">
      <button
        type="button"
        onClick={() => setExpanded((prev) => !prev)}
        className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-slate-400 transition-colors hover:bg-slate-800/50 hover:text-slate-300"
        data-testid="snoozed-section-toggle"
      >
        <ChevronRight
          className={cn("h-4 w-4 transition-transform", expanded && "rotate-90")}
        />
        <Clock className="h-3.5 w-3.5" />
        <span>Snoozed ({entries.length})</span>
      </button>

      {expanded && (
        <div className="mt-1 space-y-1 pl-4">
          {entries.map((entry) => (
            <div
              key={entry.key}
              className="flex items-center gap-2 rounded-md px-3 py-1.5 text-sm"
            >
              <span className="min-w-0 flex-1 truncate text-slate-300">
                {formatKeyTitle(entry.key)}
              </span>
              <span className="shrink-0 text-xs text-slate-500">
                {formatRelativeExpiry(entry.expiresAt)}
              </span>
              <button
                type="button"
                onClick={() => onUnsnooze(entry.key)}
                className="shrink-0 rounded p-0.5 text-slate-500 transition-colors hover:bg-slate-700/50 hover:text-slate-300"
                aria-label={`Unsnooze ${formatKeyTitle(entry.key)}`}
                data-testid={`unsnooze-${entry.key}`}
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
