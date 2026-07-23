/**
 * BacklogDetailsPanel
 *
 * Displays the metadata section of a backlog item: title, description,
 * tags, milestone link, dependency lists, acceptance globs, and timestamps.
 *
 * Extracted from BacklogDetailsPage to reduce file size and improve modularity.
 */

import { useState, useEffect } from "react";
import { MarkdownRenderer } from "../markdown/MarkdownRenderer";
import {
  ArrowRightLeft,
  ArrowUpRight,
  Edit,
  FolderOpen,
  GitBranch,
  Link2,
  Tags,
  Target,
  X,
} from "lucide-react";
import { TagList } from "../ui/tag-list";
import { EntityLink } from "../ui/entity-link";
import { DetailSection } from "../detail/DetailSection";
import { AttributionChip } from "../detail/AttributionChip";
import { NoteEditor } from "../ui/note-editor";
import { Drawer } from "../ui/drawer";
import { Button } from "../ui/button";
import { BottomSheet } from "../ui/bottom-sheet";
import { GoalPicker } from "../goals/GoalPicker";
import { useGoals, useGoalMutations } from "../../surfaces/plan/hooks/useGoals";
import { DependencyChipList } from "./dependency-chip-list";
import { formatRelativeTime } from "../../lib";
import { selectors } from "../../consts/selectors";
import { BACKLOG_KIND_ICONS } from "../../types";
import type { BacklogItem, BacklogStatus } from "../../types";
import type { DependencyRelations, ResolvedDependency } from "../../lib/backlog-queue-utils";

export interface BacklogDetailsPanelProps {
  item: BacklogItem;
  depRelations: DependencyRelations;
  spawnedItems: BacklogItem[] | undefined;
  isLocked: boolean;
  onEditGlobs: () => void;
  onDepStatusChange: (dep: ResolvedDependency, newStatus: BacklogStatus) => void;
  onSaveNote: (note: string) => Promise<void>;
  onSaveDescription: (description: string) => Promise<unknown>;
  isSavingDescription: boolean;
  descriptionSaveError?: string | null;
}

export function BacklogDetailsPanel({
  item,
  depRelations,
  spawnedItems,
  isLocked,
  onEditGlobs,
  onDepStatusChange,
  onSaveNote,
  onSaveDescription,
  isSavingDescription,
  descriptionSaveError,
}: BacklogDetailsPanelProps) {
  const [descExpanded, setDescExpanded] = useState(false);
  const [descOverflows, setDescOverflows] = useState(false);
  const [allowExpanded, setAllowExpanded] = useState(false);
  const [denyExpanded, setDenyExpanded] = useState(false);
  const [descriptionEditorOpen, setDescriptionEditorOpen] = useState(false);
  const [draftDescription, setDraftDescription] = useState(item.description ?? "");
  const [goalPickerOpen, setGoalPickerOpen] = useState(false);
  const [goalPendingDetach, setGoalPendingDetach] = useState<string | null>(null);
  const { data: goals = [] } = useGoals();
  const { addTargets, removeTargets } = useGoalMutations();
  const targetRef = `${item.kind}/${item.name}`;
  const owningGoals = goals.filter(({ goal }) => goal.targets.includes(targetRef));
  const goalMutationError = addTargets.error ?? removeTargets.error;

  useEffect(() => {
    const desc = item.description ?? "";
    setDescOverflows(desc.length > 120 || desc.includes("\n"));
  }, [item.description]);

  useEffect(() => {
    if (!descriptionEditorOpen) setDraftDescription(item.description ?? "");
  }, [item.description, descriptionEditorOpen]);

  const saveDescription = async () => {
    try {
      await onSaveDescription(draftDescription);
      setDescriptionEditorOpen(false);
    } catch {
      // The mutation exposes a typed error inline while keeping the drawer open.
    }
  };

  const attachToGoal = async (goalName: string) => {
    if (!goalName) return;
    try {
      await addTargets.mutateAsync({ name: goalName, targets: [targetRef] });
      setGoalPickerOpen(false);
    } catch {
      // React Query retains the mutation error for an accessible inline message.
    }
  };

  const detachFromGoal = async () => {
    if (!goalPendingDetach) return;
    try {
      await removeTargets.mutateAsync({ name: goalPendingDetach, targets: [targetRef] });
      setGoalPendingDetach(null);
    } catch {
      // React Query retains the mutation error for an accessible inline message.
    }
  };

  return (
    <>
      <DetailSection
        title="Overview"
        icon={BACKLOG_KIND_ICONS[item.kind]}
        hideDivider
        action={!isLocked ? (
          <button
            type="button"
            onClick={() => setDescriptionEditorOpen(true)}
            className="rounded p-1 text-slate-400 transition-colors hover:bg-white/10 hover:text-white"
            aria-label="Edit description"
            data-testid="edit-backlog-description"
          >
            <Edit className="h-3.5 w-3.5" />
          </button>
        ) : undefined}
      >
      <div className="space-y-3">
        <div className="relative">
          <MarkdownRenderer
            className={`prose-sm-slate text-sm leading-relaxed text-slate-300 ${descExpanded ? "" : "line-clamp-3"}`}
            data-testid={selectors.backlogDetails.description}
            content={item.description || "No description provided"}
          />
          {(descOverflows || descExpanded) && (
            <button
              type="button"
              onClick={() => setDescExpanded(!descExpanded)}
              className="mt-1 text-xs font-medium text-blue-400 hover:text-blue-300"
            >
              {descExpanded ? "Show less" : "Show more\u2026"}
            </button>
          )}
        </div>
        {item.tags.length > 0 && (
          <div className="space-y-2">
            <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
              <Tags className="h-3.5 w-3.5" />
              Tags
            </div>
            <TagList tags={item.tags} maxTags={10} />
          </div>
        )}
        {item.milestone && (
          <div className="space-y-2">
            <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
              <Target className="h-3.5 w-3.5" />
              Milestone
            </div>
			<span className="rounded bg-slate-800 px-2 py-1 text-xs text-slate-200" data-testid={selectors.backlogDetails.milestoneChip}>{item.milestone}</span>
          </div>
        )}
        <div className="space-y-2">
          <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
            <Target className="h-3.5 w-3.5" />
            Goal
          </div>
          {owningGoals.length > 0 ? (
            <div className="flex flex-wrap items-center gap-1.5">
              {owningGoals.map(({ goal }) => (
                <div key={goal.name} className="inline-flex items-center rounded-full bg-sky-500/15 pr-1 text-sky-400">
                  <EntityLink entityType="goal" name={goal.name} label={goal.title} className="bg-transparent hover:bg-sky-500/10" />
                  {!isLocked && (
                    <button
                      type="button"
                      onClick={() => setGoalPendingDetach(goal.name)}
                      className="rounded-full p-1 text-sky-300 hover:bg-sky-500/20 hover:text-white"
                      aria-label={`Detach from ${goal.title}`}
                    >
                      <X className="h-3 w-3" />
                    </button>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <span className="text-xs italic text-slate-500">None</span>
              {!isLocked && (
                <button
                  type="button"
                  onClick={() => setGoalPickerOpen(true)}
                  className="inline-flex items-center gap-1 text-xs text-sky-400 hover:text-sky-300"
                >
                  <Link2 className="h-3.5 w-3.5" /> Attach goal
                </button>
              )}
            </div>
          )}
          {goalMutationError && (
            <p role="alert" className="text-xs text-red-300">
              {goalMutationError instanceof Error
                ? goalMutationError.message
                : "Unable to update goal membership."}
            </p>
          )}
        </div>
        <DependencyChipList
          label="Depends on"
          items={depRelations.parents}
          icon={ArrowUpRight}
          onStatusChange={onDepStatusChange}
        />
        <DependencyChipList
          label="Depended on by"
          items={depRelations.children}
          icon={ArrowRightLeft}
          onStatusChange={onDepStatusChange}
        />
        {item.spawnedFrom && (
          <div className="space-y-2">
            <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
              <GitBranch className="h-3.5 w-3.5" />
              Spawned from
            </div>
            {(() => {
              const sf = item.spawnedFrom ?? "";
              const slashIdx = sf.indexOf("/");
              const spawnKind = slashIdx > 0 ? sf.slice(0, slashIdx) : "";
              const spawnName = slashIdx > 0 ? sf.slice(slashIdx + 1) : sf;
              return (
                <EntityLink
                  entityType="backlog"
                  kind={spawnKind}
                  name={spawnName}
                  label={sf}
                />
              );
            })()}
          </div>
        )}
        {spawnedItems && spawnedItems.length > 0 && (
          <div className="space-y-2">
            <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
              <GitBranch className="h-3.5 w-3.5" />
              Spawned items
            </div>
            <div className="flex flex-wrap gap-1.5">
              {spawnedItems.map((si) => (
                <EntityLink
                  key={`${si.kind}/${si.name}`}
                  entityType="backlog"
                  kind={si.kind}
                  name={si.name}
                  label={si.title}
                />
              ))}
            </div>
          </div>
        )}
        <div className="space-y-2 border-t border-slate-800 pt-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
              <FolderOpen className="h-3.5 w-3.5" />
              Acceptance Globs
            </div>
            {!isLocked && (
              <button
                type="button"
                onClick={onEditGlobs}
                className="rounded p-1 text-slate-400 hover:bg-white/10 hover:text-white transition-colors"
                aria-label="Edit acceptance globs"
                data-testid="edit-acceptance-globs-btn"
              >
                <Edit className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
          {(!item.acceptanceAllow || item.acceptanceAllow.length === 0) &&
           (!item.acceptanceDeny || item.acceptanceDeny.length === 0) ? (
            <button
              type="button"
              onClick={() => !isLocked && onEditGlobs()}
              disabled={isLocked}
              className="text-xs italic text-slate-500 hover:text-blue-400 transition-colors disabled:cursor-not-allowed disabled:hover:text-slate-500"
              data-testid="acceptance-globs-empty-state"
            >
              No patterns set — click to add
            </button>
          ) : (
            <>
              {item.acceptanceAllow && item.acceptanceAllow.length > 0 && (
                <div className="space-y-1.5">
                  <p className="text-[11px] font-medium text-slate-500">Allow</p>
                  <div className="flex flex-wrap gap-1.5">
                    {(allowExpanded ? item.acceptanceAllow : item.acceptanceAllow.slice(0, 3)).map((glob) => (
                      <code
                        key={glob}
                        className="inline-block rounded bg-slate-800 px-2 py-0.5 text-xs text-slate-300 font-mono"
                      >
                        {glob}
                      </code>
                    ))}
                  </div>
                  {item.acceptanceAllow.length > 3 && (
                    <button
                      type="button"
                      onClick={() => setAllowExpanded(!allowExpanded)}
                      className="mt-1 text-xs font-medium text-blue-400 hover:text-blue-300"
                    >
                      {allowExpanded ? "Show less" : `Show more\u2026 (${item.acceptanceAllow.length - 3} more)`}
                    </button>
                  )}
                </div>
              )}
              {item.acceptanceDeny && item.acceptanceDeny.length > 0 && (
                <div className="space-y-1.5">
                  <p className="text-[11px] font-medium text-slate-500">Deny</p>
                  <div className="flex flex-wrap gap-1.5">
                    {(denyExpanded ? item.acceptanceDeny : item.acceptanceDeny.slice(0, 3)).map((glob) => (
                      <code
                        key={glob}
                        className="inline-block rounded bg-slate-800 px-2 py-0.5 text-xs text-slate-300 font-mono"
                      >
                        {glob}
                      </code>
                    ))}
                  </div>
                  {item.acceptanceDeny.length > 3 && (
                    <button
                      type="button"
                      onClick={() => setDenyExpanded(!denyExpanded)}
                      className="mt-1 text-xs font-medium text-blue-400 hover:text-blue-300"
                    >
                      {denyExpanded ? "Show less" : `Show more\u2026 (${item.acceptanceDeny.length - 3} more)`}
                    </button>
                  )}
                </div>
              )}
            </>
          )}
        </div>
        <NoteEditor note={item.note ?? ""} onSave={onSaveNote} />

        <div className="grid grid-cols-2 gap-3 border-t border-slate-800 pt-3">
          <div className="space-y-1">
            <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">Priority</p>
            <p className="text-sm text-slate-300">P{item.priority}</p>
          </div>
          {item.createdBy && (
            <div className="space-y-1">
              <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">Created by</p>
              <AttributionChip attribution={item.createdBy} />
            </div>
          )}
          <div className="space-y-1">
            <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">Created</p>
            <p className="text-sm text-slate-300" title={new Date(item.created).toLocaleString()}>
              {formatRelativeTime(item.created)}
            </p>
          </div>
          <div className="space-y-1">
            <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">Updated</p>
            <p className="text-sm text-slate-300" title={new Date(item.updated).toLocaleString()}>
              {formatRelativeTime(item.updated)}
            </p>
			{item.stale && <p className="text-xs font-medium text-amber-300">Stale — refresh or triage before execution.</p>}
          </div>
        </div>
      </div>
      </DetailSection>
      <Drawer
        isOpen={descriptionEditorOpen}
        onClose={() => setDescriptionEditorOpen(false)}
        title="Edit description"
        description="Update this backlog item's markdown description."
        footer={(
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setDescriptionEditorOpen(false)} disabled={isSavingDescription}>Cancel</Button>
            <Button onClick={() => void saveDescription()} disabled={isSavingDescription}>
              {isSavingDescription ? "Saving..." : "Save description"}
            </Button>
          </div>
        )}
      >
        <div className="space-y-3 p-4">
          <label htmlFor="backlog-description-editor" className="text-sm font-medium text-slate-200">Description</label>
          <textarea
            id="backlog-description-editor"
            value={draftDescription}
            onChange={(event) => setDraftDescription(event.target.value)}
            rows={12}
            disabled={isSavingDescription}
            className="w-full resize-y rounded-lg border border-white/10 bg-slate-800/50 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
          />
          {descriptionSaveError && <p role="alert" className="text-sm text-red-300">{descriptionSaveError}</p>}
        </div>
      </Drawer>
      <Drawer
        isOpen={goalPickerOpen}
        onClose={() => setGoalPickerOpen(false)}
        title="Attach to goal"
        description="Choose the goal that should explicitly own this backlog item."
      >
        <div className="p-4">
          <GoalPicker goal="" onSelect={(goalName) => void attachToGoal(goalName)} />
        </div>
      </Drawer>
      <BottomSheet
        isOpen={Boolean(goalPendingDetach)}
        onClose={() => setGoalPendingDetach(null)}
        title="Detach from goal"
        description="This removes the explicit target membership; dependencies remain unchanged."
        footer={(
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setGoalPendingDetach(null)} disabled={removeTargets.isPending}>Cancel</Button>
            <Button variant="destructive" onClick={() => void detachFromGoal()} disabled={removeTargets.isPending}>
              {removeTargets.isPending ? "Detaching..." : "Detach"}
            </Button>
          </div>
        )}
      >
        <p className="text-sm text-slate-300">The item will no longer count as a direct target of this goal.</p>
      </BottomSheet>
    </>
  );
}
