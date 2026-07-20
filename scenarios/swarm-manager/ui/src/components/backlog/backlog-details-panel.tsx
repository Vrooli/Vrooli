/**
 * BacklogDetailsPanel
 *
 * Displays the metadata section of a backlog item: title, description,
 * tags, initiative link, dependency lists, acceptance globs, and timestamps.
 *
 * Extracted from BacklogDetailsPage to reduce file size and improve modularity.
 */

import { useState, useEffect } from "react";
import { renderMarkdown } from "../../lib/render-markdown";
import {
  ArrowRightLeft,
  ArrowUpRight,
  Edit,
  FolderOpen,
  GitBranch,
  Tags,
  Target,
} from "lucide-react";
import { TagList } from "../ui/tag-list";
import { EntityLink } from "../ui/entity-link";
import { DetailSection } from "../detail/DetailSection";
import { AttributionChip } from "../detail/AttributionChip";
import { NoteEditor } from "../ui/note-editor";
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
}

export function BacklogDetailsPanel({
  item,
  depRelations,
  spawnedItems,
  isLocked,
  onEditGlobs,
  onDepStatusChange,
  onSaveNote,
}: BacklogDetailsPanelProps) {
  const [descExpanded, setDescExpanded] = useState(false);
  const [descOverflows, setDescOverflows] = useState(false);
  const [allowExpanded, setAllowExpanded] = useState(false);
  const [denyExpanded, setDenyExpanded] = useState(false);

  useEffect(() => {
    const desc = item.description ?? "";
    setDescOverflows(desc.length > 120 || desc.includes("\n"));
  }, [item.description]);

  return (
    <DetailSection title="Overview" icon={BACKLOG_KIND_ICONS[item.kind]} hideDivider>
      <div className="space-y-3">
        <div className="relative">
          <div
            className={`prose-sm-slate text-sm leading-relaxed text-slate-300 ${descExpanded ? "" : "line-clamp-3"}`}
            data-testid={selectors.backlogDetails.description}
            dangerouslySetInnerHTML={{ __html: item.description ? renderMarkdown(item.description) : "No description provided" }}
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
        {item.initiative && (
          <div className="space-y-2">
            <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
              <Target className="h-3.5 w-3.5" />
              Initiative
            </div>
            <EntityLink
              entityType="initiative"
              name={item.initiative}
              label={item.initiative}
              data-testid={selectors.backlogDetails.initiativeChip}
            />
          </div>
        )}
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
  );
}
