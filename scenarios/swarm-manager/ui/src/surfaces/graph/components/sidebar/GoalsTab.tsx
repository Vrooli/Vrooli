import { memo, useState } from "react";
import { ChevronDown, ChevronUp, Loader2, Plus, Target } from "lucide-react";
import { Button } from "../../../../components/ui/button";
import { CreateGoalDialog } from "../../../../components/goals/CreateGoalDialog";
import { cn } from "../../../../lib/utils";
import type { GoalWithScope } from "../../../../types/goal";
import { useGoals, useGoalMutations } from "../../../plan/hooks/useGoals";
import { matchesSearch } from "./useSidebarSearch";
import { SidebarEmptyState } from "./SidebarEmptyState";
import type { SortConfig } from "./types";

const MAX_PRIORITY = 10;
const MIN_PRIORITY = 0;

interface GoalsTabProps {
  searchQuery: string;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
}

function sortGoals(goals: GoalWithScope[], sort: SortConfig): GoalWithScope[] {
  const dir = sort.direction === "asc" ? 1 : -1;
  return [...goals].sort((a, b) => {
    switch (sort.field) {
      case "priority":
        if (a.goal.priority !== b.goal.priority) {
          return (a.goal.priority - b.goal.priority) * dir;
        }
        return a.goal.title.localeCompare(b.goal.title);
      case "status":
        return a.goal.status.localeCompare(b.goal.status) * dir;
      case "alphabetical":
        return a.goal.title.localeCompare(b.goal.title) * dir;
      case "recency":
        return (new Date(b.goal.updated).getTime() - new Date(a.goal.updated).getTime()) * dir;
    }
  });
}

function LoadingSkeleton() {
  return (
    <div className="space-y-1.5">
      {[1, 2, 3].map((i) => (
        <div key={i} className="animate-pulse rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5">
          <div className="h-4 w-3/4 rounded bg-slate-800" />
          <div className="mt-2 h-2 rounded bg-slate-800" />
          <div className="mt-2 h-3 w-1/2 rounded bg-slate-800" />
        </div>
      ))}
    </div>
  );
}

function progressLabel(goal: GoalWithScope): string {
  return `${Math.round(goal.scope.progressPct)}% · ${goal.scope.completedCount}/${goal.scope.total}`;
}

function GoalsTabImpl({
  searchQuery,
  sort,
  onItemClick,
  onClearSearch,
}: GoalsTabProps) {
  const [showCreateGoal, setShowCreateGoal] = useState(false);
  const { data: goals = [], isLoading, error } = useGoals();
  const { setPriority } = useGoalMutations();

  const filtered = goals
    .filter((goal) => goal.goal.status === "active")
    .filter((goal) => (
      searchQuery
        ? matchesSearch(searchQuery, goal.goal.title, goal.goal.name, goal.goal.description ?? "")
        : true
    ));
  const sorted = sortGoals(filtered, sort);

  const changePriority = (name: string, current: number, delta: number) => {
    const next = Math.max(MIN_PRIORITY, Math.min(MAX_PRIORITY, current + delta));
    if (next !== current) {
      setPriority.mutate({ name, priority: next });
    }
  };

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300" data-testid="goals-tab-error">
        Failed to load goals: {error.message}
      </div>
    );
  }

  if (sorted.length === 0) {
    return (
      <>
        <SidebarEmptyState
          icon={Target}
          title="No goals yet."
          hint="Create a goal to track progress, blockers, and ETA across a target set."
          query={searchQuery}
          onClearSearch={onClearSearch}
          action={
            <Button
              type="button"
              size="sm"
              className="mt-1"
              onClick={() => setShowCreateGoal(true)}
              data-testid="goals-tab-create-goal"
            >
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              Create goal
            </Button>
          }
        />
        <CreateGoalDialog
          isOpen={showCreateGoal}
          onClose={() => setShowCreateGoal(false)}
        />
      </>
    );
  }

  return (
    <div className="space-y-1.5" data-testid="goals-tab">
      {sorted.map((goal) => {
        const pct = Math.max(0, Math.min(100, Math.round(goal.scope.progressPct)));
        return (
          <div
            key={goal.goal.name}
            className="rounded-lg border border-slate-800/80 bg-slate-900/60 p-2.5 text-left transition-colors hover:border-slate-700 hover:bg-slate-900"
            data-testid={`goal-row-${goal.goal.name}`}
          >
            <button
              type="button"
              className="w-full text-left"
              onClick={() => onItemClick(`goal/${goal.goal.name}`)}
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <div className="truncate text-sm font-medium text-slate-100">{goal.goal.title}</div>
                  <div className="mt-0.5 text-xs text-slate-500">
                    {progressLabel(goal)}
                    {goal.eta ? ` · ETA ${goal.eta.p50Label}-${goal.eta.p80Label}` : ""}
                  </div>
                </div>
                <span className="shrink-0 rounded bg-slate-800 px-1.5 py-0.5 text-[11px] text-slate-400">
                  P{goal.goal.priority}
                </span>
              </div>
              <div className="mt-2 h-1.5 rounded-full bg-slate-800">
                <div
                  className="h-1.5 rounded-full bg-cyan-500"
                  style={{ width: `${pct}%` }}
                  aria-hidden
                />
              </div>
              <div className="mt-1.5 flex flex-wrap gap-2 text-[11px] text-slate-500">
                <span>{goal.scope.targets.length} targets</span>
                <span>{goal.scope.ready.length} ready</span>
                {goal.scope.blockedCount > 0 && <span className="text-red-300">{goal.scope.blockedCount} blocked</span>}
              </div>
            </button>
            <div className="mt-2 flex items-center justify-end gap-1">
              <button
                type="button"
                onClick={() => changePriority(goal.goal.name, goal.goal.priority, 1)}
                disabled={goal.goal.priority >= MAX_PRIORITY || setPriority.isPending}
                className={cn(
                  "rounded border border-slate-700/60 p-1 text-slate-400 transition-colors hover:border-slate-500 hover:text-slate-200",
                  "disabled:cursor-not-allowed disabled:opacity-40",
                )}
                aria-label={`Raise ${goal.goal.title} priority`}
                data-testid={`goal-priority-up-${goal.goal.name}`}
              >
                {setPriority.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <ChevronUp className="h-3.5 w-3.5" />}
              </button>
              <button
                type="button"
                onClick={() => changePriority(goal.goal.name, goal.goal.priority, -1)}
                disabled={goal.goal.priority <= MIN_PRIORITY || setPriority.isPending}
                className={cn(
                  "rounded border border-slate-700/60 p-1 text-slate-400 transition-colors hover:border-slate-500 hover:text-slate-200",
                  "disabled:cursor-not-allowed disabled:opacity-40",
                )}
                aria-label={`Lower ${goal.goal.title} priority`}
                data-testid={`goal-priority-down-${goal.goal.name}`}
              >
                <ChevronDown className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
        );
      })}
    </div>
  );
}

export const GoalsTab = memo(GoalsTabImpl);
