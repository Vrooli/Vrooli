/**
 * SetAsGoalDialog — promote a backlog item to a goal. The target
 * can be added to any existing goal (as a new end-state target) or seed a brand
 * new goal. Shared by the graph node inspector and the plan card menu.
 */

import { useState } from "react";
import { Check, Plus } from "lucide-react";
import { ENTITY_TYPE_ICONS } from "../../types/constants";
import { Dialog } from "../ui/dialog";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";
import { useGoals, useGoalMutations } from "../../surfaces/plan/hooks/useGoals";

export interface SetAsGoalDialogProps {
  isOpen: boolean;
  onClose: () => void;
  /** Target ref: "<kind>/<name>" for a backlog item. */
  targetRef: string;
  /** Human title used to seed a new goal and label the dialog. */
  targetTitle: string;
}

export function SetAsGoalDialog({ isOpen, onClose, targetRef, targetTitle }: SetAsGoalDialogProps) {
  const { data: goals = [] } = useGoals();
  const { create, addTargets } = useGoalMutations();
  const [newTitle, setNewTitle] = useState(targetTitle);

  const active = goals.filter((g) => g.goal.status === "active");
  const busy = create.isPending || addTargets.isPending;

  const handleCreate = () => {
    const title = newTitle.trim() || targetTitle;
    create.mutate(
      { title, targets: [targetRef] },
      { onSuccess: () => onClose() },
    );
  };

  const handleAdd = (name: string) => {
    addTargets.mutate({ name, targets: [targetRef] }, { onSuccess: () => onClose() });
  };

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title="Set as goal"
      maxWidth="max-w-md"
      isLoading={busy}
      testId="set-as-goal-dialog"
    >
      <div className="space-y-4">
        <p className="text-sm text-slate-400">
          Track work toward{" "}
          <span className="font-medium text-slate-200">{targetTitle}</span> — add it as a target
          of an existing goal, or create a new one.
        </p>

        {active.length > 0 && (
          <div className="space-y-1" data-testid="set-as-goal-existing">
            <p className="text-xs font-medium uppercase tracking-wider text-slate-500">Add to goal</p>
            {active.map((g) => {
              const already = g.goal.targets.includes(targetRef);
              return (
                <button
                  key={g.goal.name}
                  type="button"
                  disabled={already || busy}
                  onClick={() => handleAdd(g.goal.name)}
                  className={cn(
                    "flex w-full items-center justify-between rounded border border-slate-800 px-3 py-2 text-left text-sm transition-colors",
                    already ? "text-slate-500" : "text-slate-200 hover:border-slate-700 hover:bg-slate-800/60",
                  )}
                  data-testid={`set-as-goal-add-${g.goal.name}`}
                >
                  <span className="flex items-center gap-2">
                    <ENTITY_TYPE_ICONS.goal className="h-3.5 w-3.5 text-cyan-400" aria-hidden />
                    <span className="truncate">{g.goal.title}</span>
                  </span>
                  {already ? (
                    <span className="flex items-center gap-1 text-xs text-emerald-400">
                      <Check className="h-3.5 w-3.5" aria-hidden /> in goal
                    </span>
                  ) : (
                    <Plus className="h-3.5 w-3.5 text-slate-500" aria-hidden />
                  )}
                </button>
              );
            })}
          </div>
        )}

        <div className="space-y-2" data-testid="set-as-goal-new">
          <p className="text-xs font-medium uppercase tracking-wider text-slate-500">New goal</p>
          <input
            type="text"
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            placeholder="Goal title"
            className="w-full rounded border border-slate-800 bg-slate-900 px-3 py-2 text-sm text-slate-200 placeholder:text-slate-600 focus:border-cyan-600 focus:outline-none"
            data-testid="set-as-goal-new-title"
          />
        </div>

        {(create.errorDescription ?? addTargets.errorDescription) && (
          <p className="text-xs text-rose-400" data-testid="set-as-goal-error">
            {(create.errorDescription ?? addTargets.errorDescription)?.message}
          </p>
        )}

        <div className="flex justify-end gap-2 pt-1">
          <Button variant="outline" size="sm" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button size="sm" onClick={handleCreate} disabled={busy} data-testid="set-as-goal-create">
            Create goal
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
