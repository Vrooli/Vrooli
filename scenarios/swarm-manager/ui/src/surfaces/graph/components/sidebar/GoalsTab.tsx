import { memo } from "react";
import type { MouseEvent } from "react";
import { ChevronDown, ChevronUp, Loader2, Plus } from "lucide-react";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { Button } from "../../../../components/ui/button";
import { cn } from "../../../../lib/utils";
import type { GoalWithScope } from "../../../../types/goal";
import { useGoals, useGoalMutations } from "../../../plan/hooks/useGoals";
import { matchesSearch } from "./useSidebarSearch";
import { SidebarEmptyState } from "./SidebarEmptyState";
import type { SortConfig } from "./types";
import { GoalProgressCard } from "../../../../components/goals/GoalProgressCard";

const MAX_PRIORITY = 10;
const MIN_PRIORITY = 0;

interface GoalsTabProps {
  searchQuery: string;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
  onCreateGoal?: () => void;
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

function GoalsTabImpl({
  searchQuery,
  sort,
  onItemClick,
  onClearSearch,
  onCreateGoal,
}: GoalsTabProps) {
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

  const stopCardNavigation = (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
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
      <SidebarEmptyState
        icon={SIDEBAR_TAB_ICONS.goals}
        title="No goals yet."
        hint="Create a goal to track progress, blockers, and ETA across a target set."
        query={searchQuery}
        onClearSearch={onClearSearch}
        action={
          onCreateGoal ? (
            <Button
              type="button"
              size="sm"
              className="mt-1"
              onClick={onCreateGoal}
              data-testid="goals-tab-create-goal"
            >
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              Create goal
            </Button>
          ) : undefined
        }
      />
    );
  }

  return (
    <div className="space-y-1.5" data-testid="goals-tab">
      {sorted.map((goal) => {
        return (
          <GoalProgressCard
            key={goal.goal.name}
            title={goal.goal.title}
            subtitle={`${Math.round(goal.scope.progressPct)}% · ${goal.scope.completedCount}/${goal.scope.total}${goal.eta ? ` · ETA ${goal.eta.p50Label}-${goal.eta.p80Label}` : ""}`}
            priority={goal.goal.priority}
            completed={goal.scope.completedCount}
            total={goal.scope.total}
            targets={goal.scope.targets.length}
            ready={goal.scope.ready.length}
            blocked={goal.scope.blockedCount}
            data-testid={`goal-row-${goal.goal.name}`}
            onOpen={() => onItemClick(`goal/${goal.goal.name}`)}
            controls={(
              <>
              <button
                type="button"
                onClick={(event) => {
                  stopCardNavigation(event);
                  changePriority(goal.goal.name, goal.goal.priority, 1);
                }}
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
                onClick={(event) => {
                  stopCardNavigation(event);
                  changePriority(goal.goal.name, goal.goal.priority, -1);
                }}
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
              </>
            )}
          />
        );
      })}
    </div>
  );
}

export const GoalsTab = memo(GoalsTabImpl);
