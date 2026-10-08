import { type KeyboardEvent, type ReactNode } from "react";
import { cn } from "../../lib/utils";

/**
 * The compact goal summary used wherever a goal is surfaced as work context.
 * Keeping this treatment shared makes goals recognisable whether they appear
 * in the sidebar or in another entity's detail view.
 *
 * `GoalProgressSummary` is the content alone, for CollectionList rows (the
 * row owns chrome and opening). `GoalProgressCard` adds standalone card chrome
 * and an optional open interaction for surfaces that are not a list.
 */
export interface GoalProgressSummaryProps {
  title: string;
  subtitle?: string;
  priority: number;
  completed: number;
  total: number;
  inProgress?: number;
  failed?: number;
  pending?: number;
  targets?: number;
  ready?: number;
  blocked?: number;
  controls?: ReactNode;
}

export interface GoalProgressCardProps extends GoalProgressSummaryProps {
  onOpen?: () => void;
  className?: string;
  "data-testid"?: string;
}

export function GoalProgressSummary({
  title,
  subtitle,
  priority,
  completed,
  total,
  inProgress = 0,
  failed = 0,
  pending = 0,
  targets,
  ready,
  blocked,
  controls,
}: GoalProgressSummaryProps) {
  const pct = total > 0 ? Math.max(0, Math.min(100, Math.round((completed / total) * 100))) : 0;
  const summary = `${pct}% · ${completed}/${total}`;

  return (
    <>
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="truncate text-sm font-medium text-slate-100">{title}</div>
          <div className="mt-0.5 truncate text-xs text-slate-500">{subtitle || summary}</div>
        </div>
        <span className="shrink-0 rounded bg-slate-800 px-1.5 py-0.5 text-[11px] text-slate-400">
          P{priority}
        </span>
      </div>
      <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-slate-800">
        <div className="h-full rounded-full bg-cyan-500 transition-[width]" style={{ width: `${pct}%` }} aria-hidden />
      </div>
      <div className="mt-1.5 flex flex-wrap gap-x-2 gap-y-0.5 text-[11px] text-slate-500">
        {targets !== undefined && <span>{targets} targets</span>}
        {ready !== undefined && <span>{ready} ready</span>}
        {inProgress > 0 && <span className="text-purple-300">{inProgress} active</span>}
        {failed > 0 && <span className="text-red-300">{failed} failed</span>}
        {pending > 0 && <span>{pending} pending</span>}
        {blocked !== undefined && blocked > 0 && <span className="text-red-300">{blocked} blocked</span>}
      </div>
      {controls && <div className="mt-2 flex items-center justify-end gap-1">{controls}</div>}
    </>
  );
}

export function GoalProgressCard({
  onOpen,
  className,
  "data-testid": testId,
  ...summary
}: GoalProgressCardProps) {
  const openFromKeyboard = (event: KeyboardEvent<HTMLDivElement>) => {
    if (onOpen && (event.key === "Enter" || event.key === " ")) {
      event.preventDefault();
      onOpen();
    }
  };

  return (
    <div
      className={cn(
        "min-w-0 max-w-full overflow-hidden rounded-lg border border-slate-800/80 bg-slate-900/60 p-2.5 text-left transition-colors hover:border-slate-700 hover:bg-slate-900",
        className,
      )}
      data-testid={testId}
      onClick={onOpen}
      onKeyDown={openFromKeyboard}
      role={onOpen ? "button" : undefined}
      tabIndex={onOpen ? 0 : undefined}
    >
      <GoalProgressSummary {...summary} />
    </div>
  );
}
