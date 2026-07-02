/**
 * BoardCard — shared presentation-only card primitive for board surfaces
 * (Plan lens columns; informed by ActivityRow and the command-post feed
 * cards). Owns layout and interaction affordances only: status dot, title,
 * meta line, badge slot, and an action slot. All behavior arrives via
 * props — no store access, no navigation.
 */

import type { KeyboardEvent, ReactNode } from "react";
import { cn } from "../../lib/utils";

export type BoardCardTone =
  | "neutral"
  | "active"
  | "attention"
  | "positive"
  | "negative"
  | "muted";

const TONE_DOT: Record<BoardCardTone, string> = {
  neutral: "bg-slate-500",
  active: "bg-cyan-400",
  attention: "bg-amber-400",
  positive: "bg-emerald-400",
  negative: "bg-rose-400",
  muted: "bg-slate-600",
};

export interface BoardCardProps {
  title: string;
  /** Small status/kind line under the title. */
  subtitle?: ReactNode;
  /** Tone of the leading status dot. */
  tone?: BoardCardTone;
  /** Pulse the status dot (in-flight work). */
  pulse?: boolean;
  /** Right-aligned badges (wave chip, counts). */
  badges?: ReactNode;
  /** Trailing action slot (buttons / menus); clicks do not bubble. */
  action?: ReactNode;
  onClick?: () => void;
  dimmed?: boolean;
  testId?: string;
}

export function BoardCard({
  title,
  subtitle,
  tone = "neutral",
  pulse = false,
  badges,
  action,
  onClick,
  dimmed = false,
  testId,
}: BoardCardProps) {
  const interactive = Boolean(onClick);

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!onClick) return;
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onClick();
    }
  };

  return (
    <div
      role={interactive ? "button" : undefined}
      tabIndex={interactive ? 0 : undefined}
      onClick={onClick}
      onKeyDown={handleKeyDown}
      className={cn(
        "flex items-start gap-2 rounded-lg border border-slate-700/40 bg-slate-900/40 px-3 py-2",
        interactive && "cursor-pointer transition-colors hover:bg-slate-800/60",
        dimmed && "opacity-50",
      )}
      data-testid={testId}
    >
      <span
        className={cn(
          "mt-1.5 h-2 w-2 shrink-0 rounded-full",
          TONE_DOT[tone],
          pulse && "animate-pulse",
        )}
        aria-hidden
      />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-slate-200">{title}</p>
        {subtitle ? (
          <div className="mt-0.5 truncate text-xs text-slate-500">{subtitle}</div>
        ) : null}
      </div>
      {badges ? <div className="flex shrink-0 items-center gap-1">{badges}</div> : null}
      {action ? (
        <div
          className="flex shrink-0 items-center gap-1"
          onClick={(event) => event.stopPropagation()}
        >
          {action}
        </div>
      ) : null}
    </div>
  );
}
