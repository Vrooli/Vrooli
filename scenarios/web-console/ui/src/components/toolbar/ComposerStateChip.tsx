import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { useSessionActivityStore } from "../../stores/useSessionActivityStore";
import { STATE_SLOT_CONFIDENCE_FLOOR } from "../messages/resolveStateSlot";
import type { ActivityState } from "../../api/sessionActivity";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#echo-rows-and-history

const LABEL: Readonly<Record<ActivityState, string>> = {
  idle: strings.composerState.idle,
  working: strings.composerState.working,
  waiting: strings.composerState.waiting,
  unknown: strings.composerState.unknown,
};

const DOT: Readonly<Record<ActivityState, string>> = {
  idle: "bg-emerald-400",
  working: "animate-pulse bg-wc-accent",
  waiting: "bg-amber-400",
  unknown: "border border-wc-text-muted",
};

/**
 * What the active session is doing, next to the composer. Informational only:
 * Send is never disabled by it.
 */
export function ComposerStateChip({ sessionId, className }: { sessionId: string | null; className?: string }) {
  const { t } = useTranslation();
  const activity = useSessionActivityStore((state) => (sessionId ? state.activities[sessionId] : undefined));
  if (!sessionId) return null;
  const state: ActivityState = activity && activity.confidence >= STATE_SLOT_CONFIDENCE_FLOOR ? activity.state : "unknown";
  return (
    <span
      data-testid="composer-state-chip"
      data-state={state}
      title={t(strings.composerState.label)}
      className={cn("inline-flex shrink-0 items-center gap-1.5 text-[11px] leading-4 text-wc-text-secondary", className)}
    >
      <span aria-hidden className={cn("h-1.5 w-1.5 rounded-full", DOT[state])} />
      {t(LABEL[state] as never)}
    </span>
  );
}
