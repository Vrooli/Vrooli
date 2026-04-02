/**
 * CommandPostButton — HUD button with notification badge.
 *
 * Matches the HUD button styling used by Stats, Settings, and Help buttons
 * in GraphWorkspace. Shows a cyan pill badge when there are actionable items.
 */

import { Inbox } from "lucide-react";
import { cn } from "../../lib/utils";

export interface CommandPostButtonProps {
  count: number;
  onClick: () => void;
  className?: string;
}

export function CommandPostButton({ count, onClick, className }: CommandPostButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex items-center gap-1.5 rounded-lg border border-slate-700/60 bg-slate-900/80 p-2 text-slate-400 transition-colors hover:bg-slate-800/80 hover:text-slate-200",
        className,
      )}
      aria-label="Command Post"
      data-testid="command-post-button"
    >
      <Inbox className="h-4 w-4" />
      {count > 0 && (
        <span className="rounded-full bg-cyan-500/20 px-1.5 py-0.5 text-xs text-cyan-200">
          {count}
        </span>
      )}
    </button>
  );
}
