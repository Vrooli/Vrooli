/**
 * CreateGoalDialog — create a goal from workspace surfaces.
 *
 * Unlike SetAsGoalDialog, this starts from an optional target set rather than
 * a single promoted item.
 */

import { useEffect, useMemo, useRef, useState } from "react";
import { Loader2, Search, Target } from "lucide-react";
import { Dialog } from "../ui/dialog";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";
import { useBacklogStore } from "../../stores";
import { useInitiativeStore } from "../../stores/initiative-store";
import { useGoalMutations } from "../../surfaces/plan/hooks/useGoals";
import type { GoalWithScope } from "../../types/goal";

export interface CreateGoalDialogProps {
  isOpen: boolean;
  onClose: () => void;
  initialTargets?: string[];
  onCreated?: (goal: GoalWithScope) => void;
}

const EMPTY_TARGETS: string[] = [];

interface TargetOption {
  ref: string;
  title: string;
  subtitle: string;
}

function buildTargetOptions(
  backlogItems: ReturnType<typeof useBacklogStore.getState>["items"],
  initiatives: ReturnType<typeof useInitiativeStore.getState>["items"],
): TargetOption[] {
  const itemOptions = backlogItems
    .filter((item) => !item.archivedAt)
    .map((item) => ({
      ref: `${item.kind}/${item.name}`,
      title: item.title || item.name,
      subtitle: `${item.kind}/${item.name}`,
    }));
  const initiativeOptions = initiatives
    .filter((entry) => !entry.initiative.archivedAt)
    .map((entry) => ({
      ref: `initiative/${entry.initiative.name}`,
      title: entry.initiative.title || entry.initiative.name,
      subtitle: `initiative/${entry.initiative.name}`,
    }));
  return [...initiativeOptions, ...itemOptions].sort((a, b) => a.title.localeCompare(b.title));
}

export function CreateGoalDialog({
  isOpen,
  onClose,
  initialTargets = EMPTY_TARGETS,
  onCreated,
}: CreateGoalDialogProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [targetQuery, setTargetQuery] = useState("");
  const [selectedTargets, setSelectedTargets] = useState<string[]>(initialTargets);
  const [attemptedSubmit, setAttemptedSubmit] = useState(false);
  const wasOpenRef = useRef(isOpen);

  const backlogItems = useBacklogStore((s) => s.items);
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const initiatives = useInitiativeStore((s) => s.items);
  const fetchInitiatives = useInitiativeStore((s) => s.fetchInitiatives);
  const { create } = useGoalMutations();

  useEffect(() => {
    const wasOpen = wasOpenRef.current;
    wasOpenRef.current = isOpen;
    if (!isOpen) return;
    if (!wasOpen) {
      setSelectedTargets(initialTargets);
      setAttemptedSubmit(false);
      setTargetQuery("");
    }
    if (backlogItems.length === 0) {
      void fetchBacklog();
    }
    if (initiatives.length === 0) {
      void fetchInitiatives();
    }
  }, [backlogItems.length, fetchBacklog, fetchInitiatives, initialTargets, initiatives.length, isOpen]);

  const options = useMemo(() => buildTargetOptions(backlogItems, initiatives), [backlogItems, initiatives]);
  const filteredOptions = useMemo(() => {
    const q = targetQuery.trim().toLowerCase();
    if (!q) return options.slice(0, 40);
    return options
      .filter((option) => (
        option.title.toLowerCase().includes(q) ||
        option.subtitle.toLowerCase().includes(q)
      ))
      .slice(0, 40);
  }, [options, targetQuery]);

  const trimmedTitle = title.trim();
  const busy = create.isPending;
  const titleError = attemptedSubmit && trimmedTitle.length === 0;

  const toggleTarget = (ref: string) => {
    setSelectedTargets((current) => (
      current.includes(ref)
        ? current.filter((item) => item !== ref)
        : [...current, ref]
    ));
  };

  const resetAndClose = () => {
    if (busy) return;
    setTitle("");
    setDescription("");
    setTargetQuery("");
    setSelectedTargets(initialTargets);
    setAttemptedSubmit(false);
    onClose();
  };

  const handleSubmit = () => {
    setAttemptedSubmit(true);
    if (!trimmedTitle) return;
    create.mutate(
      {
        title: trimmedTitle,
        description: description.trim() || undefined,
        targets: selectedTargets,
      },
      {
        onSuccess: (goal) => {
          onCreated?.(goal);
          resetAndClose();
        },
      },
    );
  };

  return (
    <Dialog
      isOpen={isOpen}
      onClose={resetAndClose}
      title="Create goal"
      maxWidth="max-w-2xl"
      isLoading={busy}
      testId="create-goal-dialog"
    >
      <div className="space-y-5">
        <div className="grid gap-3">
          <label className="grid gap-1.5 text-sm">
            <span className="font-medium text-slate-200">Title</span>
            <input
              type="text"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              placeholder="Ship workspace goals"
              className={cn(
                "w-full rounded border bg-slate-950 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-600 focus:outline-none",
                titleError ? "border-red-500 focus:border-red-400" : "border-slate-800 focus:border-cyan-600",
              )}
              data-testid="create-goal-title"
            />
            {titleError && <span className="text-xs text-red-300">Title is required.</span>}
          </label>

          <label className="grid gap-1.5 text-sm">
            <span className="font-medium text-slate-200">Description</span>
            <textarea
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              rows={3}
              placeholder="Optional context for the target outcome."
              className="w-full resize-none rounded border border-slate-800 bg-slate-950 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-600 focus:border-cyan-600 focus:outline-none"
              data-testid="create-goal-description"
            />
          </label>
        </div>

        <div className="space-y-2">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-sm font-medium text-slate-200">Targets</p>
              <p className="text-xs text-slate-500">Optional, but goals are most useful with at least one item or initiative.</p>
            </div>
            <span className="rounded-full border border-slate-700/80 px-2 py-0.5 text-xs text-slate-400">
              {selectedTargets.length} selected
            </span>
          </div>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
            <input
              type="search"
              value={targetQuery}
              onChange={(event) => setTargetQuery(event.target.value)}
              placeholder="Find targets"
              className="w-full rounded border border-slate-800 bg-slate-950 py-2 pl-9 pr-3 text-sm text-slate-100 placeholder:text-slate-600 focus:border-cyan-600 focus:outline-none"
              data-testid="create-goal-target-search"
            />
          </div>
          <div className="max-h-64 overflow-y-auto rounded-lg border border-slate-800 bg-slate-950/70" data-testid="create-goal-targets">
            {filteredOptions.length === 0 ? (
              <div className="flex flex-col items-center gap-2 px-4 py-8 text-center text-sm text-slate-500">
                <Target className="h-6 w-6" />
                <span>No targets match this search.</span>
              </div>
            ) : (
              filteredOptions.map((option) => {
                const checked = selectedTargets.includes(option.ref);
                return (
                  <label
                    key={option.ref}
                    className="flex cursor-pointer items-start gap-3 border-b border-slate-900 px-3 py-2.5 last:border-b-0 hover:bg-slate-900"
                  >
                    <input
                      type="checkbox"
                      checked={checked}
                      onChange={() => toggleTarget(option.ref)}
                      className="mt-0.5 h-4 w-4 rounded border-slate-600 bg-slate-800 text-cyan-500 focus:ring-cyan-500"
                      data-testid={`create-goal-target-${option.ref}`}
                    />
                    <span className="min-w-0">
                      <span className="block truncate text-sm font-medium text-slate-200">{option.title}</span>
                      <span className="block truncate text-xs text-slate-500">{option.subtitle}</span>
                    </span>
                  </label>
                );
              })
            )}
          </div>
        </div>

        {create.isError && (
          <p className="text-sm text-red-300" data-testid="create-goal-error">
            {create.error instanceof Error ? create.error.message : "Failed to create goal."}
          </p>
        )}

        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" size="sm" onClick={resetAndClose} disabled={busy}>
            Cancel
          </Button>
          <Button type="button" size="sm" onClick={handleSubmit} disabled={busy} data-testid="create-goal-submit">
            {busy && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Create goal
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
