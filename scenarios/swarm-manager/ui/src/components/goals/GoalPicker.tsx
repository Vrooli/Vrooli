/**
 * GoalPicker — the board's goal scope selector. Picking a goal scopes the
 * board (and its ETA) to that goal's transitive closure via the server `goal`
 * param; "All work" clears the scope. The dropdown also surfaces per-goal
 * progress and priority, with inline ▲/▼ priority controls (priority feeds the
 * goal-directed drain comparator).
 */

import { useRef, useState } from "react";
import { ChevronUp, ChevronDown, Plus } from "lucide-react";
import { ENTITY_TYPE_ICONS } from "../../types/constants";
import { cn } from "../../lib/utils";
import type { GoalWithScope } from "../../types/goal";
import { useGoals, useGoalMutations } from "../../surfaces/plan/hooks/useGoals";
import { CreateGoalDialog } from "./CreateGoalDialog";
import { Popover } from "../ui/popover";
import { useAttachToSessionAction } from "../session/context/useAttachToSessionAction";
import { goalOption } from "../session/context/session-context-refs";

const MAX_PRIORITY = 10;
const MIN_PRIORITY = 0;

/** Active goals, highest priority first, then title. */
function sortGoals(goals: GoalWithScope[]): GoalWithScope[] {
  return goals
    .filter((g) => g.goal.status === "active")
    .slice()
    .sort((a, b) => {
      if (b.goal.priority !== a.goal.priority) return b.goal.priority - a.goal.priority;
      return a.goal.title.localeCompare(b.goal.title);
    });
}

function progressLabel(g: GoalWithScope): string {
  return `${Math.round(g.scope.progressPct)}% · ${g.scope.completedCount}/${g.scope.total}`;
}

export function GoalPicker({
  goal,
  onSelect,
}: {
  goal: string;
  onSelect: (goal: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [showCreateGoal, setShowCreateGoal] = useState(false);
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const { data: goals = [] } = useGoals();
  const { setPriority } = useGoalMutations();

  const active = sortGoals(goals);
  const selected = goals.find((g) => g.goal.name === goal);
  const label = selected ? selected.goal.title : "All work";
  const attach = useAttachToSessionAction(selected ? goalOption(selected) : null);

  const changePriority = (name: string, current: number, delta: number) => {
    const next = Math.max(MIN_PRIORITY, Math.min(MAX_PRIORITY, current + delta));
    if (next !== current) setPriority.mutate({ name, priority: next });
  };

  return (
    <div className="flex items-center gap-1">
      <button
        ref={triggerRef}
        type="button"
        onMouseDown={(event) => event.stopPropagation()}
        onClick={() => setOpen((prev) => !prev)}
        className={cn(
          "flex items-center gap-1 rounded px-2 py-1 text-xs transition-colors",
          goal ? "text-cyan-400" : "text-slate-500 hover:text-slate-300",
        )}
        title={goal ? `Board scoped to goal: ${label}` : "Scope the board to a goal"}
        data-testid="plan-goal-picker"
      >
        <ENTITY_TYPE_ICONS.goal className="h-3.5 w-3.5" aria-hidden />
        <span className="max-w-[10rem] truncate">{label}</span>
        {selected && (
          <span className="text-slate-500" data-testid="plan-goal-picker-progress">
            {Math.round(selected.scope.progressPct)}%
          </span>
        )}
      </button>
      {selected && (
        <>
          {attach.button}
          {attach.sheet}
        </>
      )}

      <Popover
        isOpen={open}
        onClose={() => setOpen(false)}
        triggerRef={triggerRef}
        className="w-72 rounded-lg border-slate-700/80 bg-slate-900/95 p-1 shadow-xl backdrop-blur"
        testId="plan-goal-picker-menu"
      >
        <div role="listbox">
          <button
            type="button"
            onClick={() => {
              onSelect("");
              setOpen(false);
            }}
            className={cn(
              "flex w-full items-center justify-between rounded px-2 py-1.5 text-left text-xs transition-colors hover:bg-slate-800",
              !goal ? "text-cyan-400" : "text-slate-300",
            )}
            role="option"
            aria-selected={!goal}
            data-testid="plan-goal-option-all"
          >
            <span>All work</span>
          </button>

          {active.length === 0 && (
            <div className="space-y-2 px-2 py-2" data-testid="plan-goal-picker-empty">
              <p className="text-xs leading-snug text-slate-500">
                No goals yet. Create one to scope the board around a target set.
              </p>
              <button
                type="button"
                onClick={() => setShowCreateGoal(true)}
                className="inline-flex items-center gap-1.5 rounded border border-slate-700 px-2 py-1 text-xs font-medium text-slate-200 transition-colors hover:border-slate-500 hover:bg-slate-800"
                data-testid="plan-goal-picker-create"
              >
                <Plus className="h-3.5 w-3.5" aria-hidden />
                Create goal
              </button>
            </div>
          )}

          {active.map((g) => (
            <div
              key={g.goal.name}
              className={cn(
                "flex items-center gap-1 rounded px-2 py-1.5 text-xs transition-colors hover:bg-slate-800",
                g.goal.name === goal ? "text-cyan-400" : "text-slate-300",
              )}
              data-testid={`plan-goal-option-${g.goal.name}`}
            >
              <button
                type="button"
                onClick={() => {
                  onSelect(g.goal.name);
                  setOpen(false);
                }}
                className="flex min-w-0 flex-1 flex-col items-start text-left"
                role="option"
                aria-selected={g.goal.name === goal}
              >
                <span className="w-full truncate font-medium">{g.goal.title}</span>
                <span className="text-slate-500">
                  {progressLabel(g)}
                  {g.eta ? ` · ETA ${g.eta.p50Label}–${g.eta.p80Label}` : ""}
                </span>
              </button>
              <span className="flex flex-col items-center">
                <span className="tabular-nums text-slate-500" title="Goal priority" data-testid={`plan-goal-priority-${g.goal.name}`}>
                  P{g.goal.priority}
                </span>
              </span>
              <span className="flex flex-col">
                <button
                  type="button"
                  onClick={() => changePriority(g.goal.name, g.goal.priority, 1)}
                  disabled={g.goal.priority >= MAX_PRIORITY}
                  className="text-slate-500 transition-colors hover:text-slate-200 disabled:opacity-30"
                  title="Raise priority"
                  data-testid={`plan-goal-priority-up-${g.goal.name}`}
                >
                  <ChevronUp className="h-3.5 w-3.5" aria-hidden />
                </button>
                <button
                  type="button"
                  onClick={() => changePriority(g.goal.name, g.goal.priority, -1)}
                  disabled={g.goal.priority <= MIN_PRIORITY}
                  className="text-slate-500 transition-colors hover:text-slate-200 disabled:opacity-30"
                  title="Lower priority"
                  data-testid={`plan-goal-priority-down-${g.goal.name}`}
                >
                  <ChevronDown className="h-3.5 w-3.5" aria-hidden />
                </button>
              </span>
            </div>
          ))}
        </div>
      </Popover>
      <CreateGoalDialog
        isOpen={showCreateGoal}
        onClose={() => setShowCreateGoal(false)}
        onCreated={(created) => {
          onSelect(created.goal.name);
          setOpen(false);
        }}
      />
    </div>
  );
}
