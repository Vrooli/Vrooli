/**
 * CreateGoalDialog — create a goal from workspace surfaces.
 *
 * Unlike SetAsGoalDialog, this starts from an optional target set rather than
 * a single promoted item. It reuses the same drawer + entity-card + chip-tray
 * primitives as the session-context picker so target selection looks and
 * behaves like the rest of swarm-manager (full-page BottomSheet, CompactTabBar
 * tabs, the shared InitiativeSummaryCard / BacklogCard in pick-mode, and the
 * ContextChipTray for the current selection).
 */

import { useEffect, useMemo, useRef, useState } from "react";
import { Layers, ListTodo, Loader2, Target } from "lucide-react";
import { BottomSheet } from "../ui/bottom-sheet";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { CompactTabBar } from "../ui/compact-tab-bar";
import { useBacklogStore } from "../../stores";
import { useInitiativeStore } from "../../stores/initiative-store";
import { useGoalMutations } from "../../surfaces/plan/hooks/useGoals";
import { BacklogCard } from "../backlog/backlog-card";
import { InitiativeSummaryCard } from "../initiative/initiative-summary-card";
import { ContextChipTray, type ComposerContextChip } from "../composer/ContextChipTray";
import type { CardSelection } from "../session/context/selectable";
import type { BacklogItem, InitiativeWithRollup } from "../../types";
import type { GoalWithScope } from "../../types/goal";

export interface CreateGoalDialogProps {
  isOpen: boolean;
  onClose: () => void;
  initialTargets?: string[];
  onCreated?: (goal: GoalWithScope) => void;
}

const EMPTY_TARGETS: string[] = [];

type TargetTab = "initiative" | "item";

const TAB_LABELS: Record<TargetTab, string> = {
  initiative: "Initiatives",
  item: "Items",
};

/** A selectable goal target, carrying its entity and the ref persisted on the goal. */
interface InitiativeTarget {
  ref: string;
  entity: InitiativeWithRollup;
  title: string;
  subtitle: string;
}
interface BacklogTarget {
  ref: string;
  entity: BacklogItem;
  title: string;
  subtitle: string;
}

function initiativeRef(entry: InitiativeWithRollup): string {
  return `initiative/${entry.initiative.name}`;
}
function backlogRef(item: BacklogItem): string {
  return `${item.kind}/${item.name}`;
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
  const [activeTab, setActiveTab] = useState<TargetTab>("initiative");
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
      setActiveTab("initiative");
    }
    if (backlogItems.length === 0) {
      void fetchBacklog();
    }
    if (initiatives.length === 0) {
      void fetchInitiatives();
    }
  }, [backlogItems.length, fetchBacklog, fetchInitiatives, initialTargets, initiatives.length, isOpen]);

  const initiativeTargets = useMemo<InitiativeTarget[]>(() => (
    initiatives
      .filter((entry) => !(entry.initiative as { archivedAt?: string }).archivedAt)
      .map((entry) => ({
        ref: initiativeRef(entry),
        entity: entry,
        title: entry.initiative.title || entry.initiative.name,
        subtitle: `initiative/${entry.initiative.name}`,
      }))
      .sort((a, b) => a.title.localeCompare(b.title))
  ), [initiatives]);

  const backlogTargets = useMemo<BacklogTarget[]>(() => (
    backlogItems
      .filter((item) => !item.archivedAt)
      .map((item) => ({
        ref: backlogRef(item),
        entity: item,
        title: item.title || item.name,
        subtitle: `${item.kind}/${item.name}`,
      }))
      .sort((a, b) => a.title.localeCompare(b.title))
  ), [backlogItems]);

  // Ref → display metadata, used to render selection chips even for targets
  // whose entity is not in the currently-loaded lists (e.g. initialTargets).
  const metaByRef = useMemo(() => {
    const map = new Map<string, { type: TargetTab; title: string; subtitle: string }>();
    for (const t of initiativeTargets) map.set(t.ref, { type: "initiative", title: t.title, subtitle: t.subtitle });
    for (const t of backlogTargets) map.set(t.ref, { type: "item", title: t.title, subtitle: t.subtitle });
    return map;
  }, [initiativeTargets, backlogTargets]);

  const filteredInitiatives = useMemo(() => {
    const q = targetQuery.trim().toLowerCase();
    if (!q) return initiativeTargets.slice(0, 80);
    return initiativeTargets
      .filter((t) => t.title.toLowerCase().includes(q) || t.subtitle.toLowerCase().includes(q))
      .slice(0, 80);
  }, [initiativeTargets, targetQuery]);
  const filteredBacklog = useMemo(() => {
    const q = targetQuery.trim().toLowerCase();
    if (!q) return backlogTargets.slice(0, 80);
    return backlogTargets
      .filter((t) => t.title.toLowerCase().includes(q) || t.subtitle.toLowerCase().includes(q))
      .slice(0, 80);
  }, [backlogTargets, targetQuery]);

  const selectedSet = useMemo(() => new Set(selectedTargets), [selectedTargets]);

  const selectedChips = useMemo<ComposerContextChip[]>(() => (
    selectedTargets.map((ref) => {
      const meta = metaByRef.get(ref);
      return {
        type: meta?.type === "item" ? "backlog_item" : "initiative",
        ref,
        title: meta?.title ?? ref,
        subtitle: meta?.subtitle ?? ref,
      };
    })
  ), [selectedTargets, metaByRef]);

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

  const selectionFor = (ref: string): CardSelection => ({
    selectionMode: true,
    selected: selectedSet.has(ref),
    onToggleSelect: () => toggleTarget(ref),
  });

  const resetAndClose = () => {
    if (busy) return;
    setTitle("");
    setDescription("");
    setTargetQuery("");
    setActiveTab("initiative");
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

  const activeList = activeTab === "initiative" ? filteredInitiatives : filteredBacklog;

  return (
    <BottomSheet
      isOpen={isOpen}
      onClose={resetAndClose}
      title="Create goal"
      description="Group initiatives and items under a single target outcome."
      className="!max-w-2xl border-slate-700/80 bg-slate-900"
      contentClassName="px-0 py-0"
      data-testid="create-goal-dialog"
      footer={
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-xs text-slate-400" data-testid="create-goal-selected-count">
            {create.isError ? (
              <span className="text-red-300" data-testid="create-goal-error">
                {create.error instanceof Error ? create.error.message : "Failed to create goal."}
              </span>
            ) : selectedTargets.length > 0 ? (
              `${selectedTargets.length} ${selectedTargets.length === 1 ? "target" : "targets"} selected.`
            ) : (
              "Optional — goals are most useful with at least one target."
            )}
          </p>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" size="sm" onClick={resetAndClose} disabled={busy}>
              Cancel
            </Button>
            <Button type="button" size="sm" onClick={handleSubmit} disabled={busy} data-testid="create-goal-submit">
              {busy && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Create goal
            </Button>
          </div>
        </div>
      }
    >
      <div className="flex min-h-0 flex-col">
        {/* Goal details */}
        <div className="space-y-3 border-b border-white/10 px-3 py-3 sm:px-4">
          <label className="grid gap-1.5 text-sm">
            <span className="font-medium text-slate-200">Title</span>
            <Input
              type="text"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              placeholder="Ship workspace goals"
              variant={titleError ? "error" : "default"}
              size="sm"
              data-testid="create-goal-title"
            />
            {titleError && <span className="text-xs text-red-300">Title is required.</span>}
          </label>

          <label className="grid gap-1.5 text-sm">
            <span className="font-medium text-slate-200">Description</span>
            <textarea
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              rows={2}
              placeholder="Optional context for the target outcome."
              className="w-full resize-none rounded-lg border border-white/10 bg-slate-800/50 px-3 py-2 text-base text-slate-50 placeholder:text-slate-400 transition-colors focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500 md:text-sm"
              data-testid="create-goal-description"
            />
          </label>
        </div>

        {/* Targets */}
        <div className="space-y-2.5 border-b border-white/10 px-3 py-2.5 sm:px-4">
          <div className="flex items-center justify-between gap-3">
            <p className="text-sm font-medium text-slate-200">Targets</p>
            <span className="rounded-full border border-slate-700/80 px-2 py-0.5 text-xs text-slate-400">
              {selectedTargets.length} selected
            </span>
          </div>

          <ContextChipTray
            items={selectedChips}
            onRemove={(_type, ref) => toggleTarget(ref)}
            className="max-h-16"
            testId="create-goal-selected-tray"
          />

          <Input
            type="search"
            value={targetQuery}
            onChange={(event) => setTargetQuery(event.target.value)}
            placeholder="Find targets"
            size="sm"
            leftIcon={<Target className="h-4 w-4" />}
            data-testid="create-goal-target-search"
          />
        </div>

        <CompactTabBar
          items={[
            { value: "initiative" as const, label: TAB_LABELS.initiative, icon: Layers, count: initiativeTargets.length },
            { value: "item" as const, label: TAB_LABELS.item, icon: ListTodo, count: backlogTargets.length },
          ]}
          activeValue={activeTab}
          onValueChange={setActiveTab}
          aria-label="Target types"
          className="border-b border-white/10 px-1"
          tabTestIdPrefix="create-goal-tab"
        />

        <div className="max-h-[46vh] overflow-y-auto px-2.5 py-2.5 sm:px-3" data-testid="create-goal-targets">
          {activeList.length === 0 ? (
            <div className="flex flex-col items-center gap-2 rounded-md border border-dashed border-slate-700 bg-slate-950/40 px-4 py-10 text-center text-sm text-slate-500">
              <Target className="h-6 w-6" />
              <span>
                {targetQuery.trim()
                  ? "No targets match this search."
                  : `No ${TAB_LABELS[activeTab].toLowerCase()} available.`}
              </span>
            </div>
          ) : (
            <div className="space-y-1.5">
              {activeTab === "initiative"
                ? filteredInitiatives.map((target) => (
                    <InitiativeSummaryCard
                      key={target.ref}
                      item={target.entity}
                      selection={selectionFor(target.ref)}
                    />
                  ))
                : filteredBacklog.map((target) => (
                    <BacklogCard
                      key={target.ref}
                      item={target.entity}
                      selection={selectionFor(target.ref)}
                    />
                  ))}
            </div>
          )}
        </div>
      </div>
    </BottomSheet>
  );
}
