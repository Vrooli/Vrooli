/**
 * FilterBar - Collapsible per-tab filter and sort controls.
 */

import type { Dispatch } from "react";
import { ArrowDown, ArrowUp, X } from "lucide-react";
import { cn } from "../../../../lib/utils";
import { CollapsibleSection } from "../../../../components/ui/collapsible-section";
import type { SidebarAction } from "./useSidebarState";
import type { BacklogFilters, CaptureFilters, ExecutionFilters, SessionFilters, SidebarTab, SortConfig, SortDirection, SortField } from "./types";
import { DEFAULT_SORT } from "./types";

interface FilterBarProps {
  activeTab: SidebarTab;
  backlogFilters: BacklogFilters;
  captureFilters: CaptureFilters;
  executionFilters: ExecutionFilters;
  sessionFilters: SessionFilters;
  sort: SortConfig;
  dispatch: Dispatch<SidebarAction>;
}

// ============================================================================
// Chip Component
// ============================================================================

function Chip({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "rounded-full px-2 py-0.5 text-[11px] font-medium transition-colors",
        active
          ? "bg-cyan-500/20 text-cyan-300"
          : "bg-slate-800/60 text-slate-400 hover:bg-slate-700/60 hover:text-slate-300",
      )}
    >
      {label}
    </button>
  );
}

// ============================================================================
// Sort Controls
// ============================================================================

const SORT_OPTIONS: { field: SortField; label: string }[] = [
  { field: "priority", label: "Priority" },
  { field: "recency", label: "Recent" },
  { field: "status", label: "Status" },
  { field: "alphabetical", label: "A-Z" },
];

function SortControls({ sort, tab, dispatch }: { sort: SortConfig; tab: SidebarTab; dispatch: Dispatch<SidebarAction> }) {
  return (
    <div className="flex flex-wrap items-center gap-1">
      {SORT_OPTIONS.map((opt) => (
        <button
          key={opt.field}
          type="button"
          onClick={() => {
            if (sort.field === opt.field) {
              const nextDir: SortDirection = sort.direction === "asc" ? "desc" : "asc";
              dispatch({ type: "SET_SORT", tab, sort: { direction: nextDir } });
            } else {
              dispatch({ type: "SET_SORT", tab, sort: { field: opt.field, direction: DEFAULT_SORT[tab].direction } });
            }
          }}
          className={cn(
            "flex items-center gap-0.5 rounded-full px-2 py-0.5 text-[11px] font-medium transition-colors",
            sort.field === opt.field
              ? "bg-cyan-500/20 text-cyan-300"
              : "bg-slate-800/60 text-slate-400 hover:bg-slate-700/60",
          )}
        >
          {opt.label}
          {sort.field === opt.field && (
            sort.direction === "asc" ? <ArrowUp className="h-2.5 w-2.5" /> : <ArrowDown className="h-2.5 w-2.5" />
          )}
        </button>
      ))}
    </div>
  );
}

// ============================================================================
// Per-Tab Filters
// ============================================================================

const BACKLOG_STATUSES = ["backlog", "researching", "ready", "queued", "in_progress", "completed", "failed"] as const;
const BACKLOG_KINDS = ["idea", "research", "fix", "execute", "chore"] as const;
const CAPTURE_STATUSES = ["classifying", "classified", "failed"] as const;
const EXECUTION_STATUSES = ["pending", "starting", "running", "needs_review", "validating", "needs_fixup", "completed", "failed", "canceled"] as const;
const EXECUTION_MODES = ["manual", "yolo"] as const;
const SESSION_STATUSES = ["starting", "running", "waiting_for_user", "proposal_ready", "applying", "complete", "failed", "canceled"] as const;
const SESSION_KINDS = [
  { value: "meta_orchestration", label: "Plan work" },
  { value: "swarm_operations", label: "Operations" },
	{ value: "workflow_authoring", label: "Workflow authoring" },
] as const;

function toggleInArray<T>(arr: T[], value: T): T[] {
  return arr.includes(value) ? arr.filter((v) => v !== value) : [...arr, value];
}

function BacklogFilterChips({ filters, dispatch }: { filters: BacklogFilters; dispatch: Dispatch<SidebarAction> }) {
  return (
    <>
      <div>
        <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-slate-500">Status</p>
        <div className="flex flex-wrap gap-1">
          {BACKLOG_STATUSES.map((s) => (
            <Chip
              key={s}
              label={s.replace(/_/g, " ")}
              active={filters.statuses.includes(s)}
              onClick={() => dispatch({ type: "SET_BACKLOG_FILTERS", filters: { statuses: toggleInArray(filters.statuses, s) } })}
            />
          ))}
        </div>
      </div>
      <div>
        <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-slate-500">Kind</p>
        <div className="flex flex-wrap gap-1">
          {BACKLOG_KINDS.map((k) => (
            <Chip
              key={k}
              label={k}
              active={filters.kinds.includes(k)}
              onClick={() => dispatch({ type: "SET_BACKLOG_FILTERS", filters: { kinds: toggleInArray(filters.kinds, k) } })}
            />
          ))}
        </div>
      </div>
      <div>
        <label className="flex items-center gap-1.5 cursor-pointer">
          <input
            type="checkbox"
            checked={filters.showArchived}
            onChange={() => dispatch({ type: "SET_BACKLOG_FILTERS", filters: { showArchived: !filters.showArchived } })}
            className="h-3 w-3 rounded border-slate-600 bg-slate-800 text-cyan-500 focus:ring-cyan-500/30"
          />
          <span className="text-[11px] text-slate-400">Show Archived</span>
        </label>
      </div>
    </>
  );
}

function CaptureFilterChips({ filters, dispatch }: { filters: CaptureFilters; dispatch: Dispatch<SidebarAction> }) {
  return (
    <div>
      <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-slate-500">Status</p>
      <div className="flex flex-wrap gap-1">
        {CAPTURE_STATUSES.map((s) => (
          <Chip
            key={s}
            label={s}
            active={filters.statuses.includes(s)}
            onClick={() => dispatch({ type: "SET_CAPTURE_FILTERS", filters: { statuses: toggleInArray(filters.statuses, s) } })}
          />
        ))}
      </div>
    </div>
  );
}

function ExecutionFilterChips({ filters, dispatch }: { filters: ExecutionFilters; dispatch: Dispatch<SidebarAction> }) {
  return (
    <>
      <div>
        <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-slate-500">Status</p>
        <div className="flex flex-wrap gap-1">
          {EXECUTION_STATUSES.map((s) => (
            <Chip
              key={s}
              label={s.replace(/_/g, " ")}
              active={filters.statuses.includes(s)}
              onClick={() => dispatch({ type: "SET_EXECUTION_FILTERS", filters: { statuses: toggleInArray(filters.statuses, s) } })}
            />
          ))}
        </div>
      </div>
      <div>
        <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-slate-500">Mode</p>
        <div className="flex flex-wrap gap-1">
          {EXECUTION_MODES.map((m) => (
            <Chip
              key={m}
              label={m}
              active={filters.modes.includes(m)}
              onClick={() => dispatch({ type: "SET_EXECUTION_FILTERS", filters: { modes: toggleInArray(filters.modes, m) } })}
            />
          ))}
        </div>
      </div>
    </>
  );
}

function SessionFilterChips({ filters, dispatch }: { filters: SessionFilters; dispatch: Dispatch<SidebarAction> }) {
  return (
    <>
      <div>
        <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-slate-500">Status</p>
        <div className="flex flex-wrap gap-1">
          {SESSION_STATUSES.map((s) => (
            <Chip
              key={s}
              label={s.replace(/_/g, " ")}
              active={filters.statuses.includes(s)}
              onClick={() => dispatch({ type: "SET_SESSION_FILTERS", filters: { statuses: toggleInArray(filters.statuses, s) } })}
            />
          ))}
        </div>
      </div>
      <div>
        <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-slate-500">Kind</p>
        <div className="flex flex-wrap gap-1">
          {SESSION_KINDS.map((kind) => (
            <Chip
              key={kind.value}
              label={kind.label}
              active={filters.kinds.includes(kind.value)}
              onClick={() => dispatch({ type: "SET_SESSION_FILTERS", filters: { kinds: toggleInArray(filters.kinds, kind.value) } })}
            />
          ))}
        </div>
      </div>
      <div className="space-y-1">
        <label className="flex cursor-pointer items-center gap-1.5">
          <input
            type="checkbox"
            checked={filters.activeOnly}
            onChange={() => dispatch({ type: "SET_SESSION_FILTERS", filters: { activeOnly: !filters.activeOnly } })}
            className="h-3 w-3 rounded border-slate-600 bg-slate-800 text-cyan-500 focus:ring-cyan-500/30"
          />
          <span className="text-[11px] text-slate-400">Active only</span>
        </label>
        <label className="flex cursor-pointer items-center gap-1.5">
          <input
            type="checkbox"
            checked={filters.hasProposals}
            onChange={() => dispatch({ type: "SET_SESSION_FILTERS", filters: { hasProposals: !filters.hasProposals } })}
            className="h-3 w-3 rounded border-slate-600 bg-slate-800 text-cyan-500 focus:ring-cyan-500/30"
          />
          <span className="text-[11px] text-slate-400">Has proposals</span>
        </label>
        <label className="flex cursor-pointer items-center gap-1.5">
          <input
            type="checkbox"
            checked={filters.hasAppliedArtifacts}
            onChange={() => dispatch({ type: "SET_SESSION_FILTERS", filters: { hasAppliedArtifacts: !filters.hasAppliedArtifacts } })}
            className="h-3 w-3 rounded border-slate-600 bg-slate-800 text-cyan-500 focus:ring-cyan-500/30"
          />
          <span className="text-[11px] text-slate-400">Has applied artifacts</span>
        </label>
      </div>
    </>
  );
}

// ============================================================================
// FilterBar
// ============================================================================

function hasActiveFiltersForTab(tab: SidebarTab, backlog: BacklogFilters, captures: CaptureFilters, executions: ExecutionFilters, sessions: SessionFilters, sort: SortConfig): boolean {
  const defaultSort = DEFAULT_SORT[tab];
  const sortChanged = sort.field !== defaultSort.field || sort.direction !== defaultSort.direction;

  switch (tab) {
    case "backlog": return backlog.statuses.length > 0 || backlog.kinds.length > 0 || backlog.priorityMin !== null || backlog.priorityMax !== null || backlog.showArchived || sortChanged;
    case "captures": return captures.statuses.length > 0 || sortChanged;
    case "goals": return sortChanged;
    case "executions": return executions.statuses.length > 0 || executions.modes.length > 0 || sortChanged;
    case "sessions": return sessions.statuses.length > 0 || sessions.kinds.length > 0 || sessions.activeOnly || sessions.hasProposals || sessions.hasAppliedArtifacts || sortChanged;
  }
}

export function FilterBar({ activeTab, backlogFilters, captureFilters, executionFilters, sessionFilters, sort, dispatch }: FilterBarProps) {
  const hasActive = hasActiveFiltersForTab(activeTab, backlogFilters, captureFilters, executionFilters, sessionFilters, sort);

  return (
    <CollapsibleSection
      storageKey="sidebar-filters"
      className="border-b border-slate-200/20"
      headerClassName="px-3 py-1.5"
      contentClassName="space-y-2.5 px-3 pb-2.5"
      toggleTestId="filter-bar-toggle"
      contentTestId="filter-bar-content"
      label={
        <>
          Filters & Sort
          {hasActive && <span className="ml-1 h-1.5 w-1.5 rounded-full bg-cyan-400" />}
        </>
      }
      headerRight={
        hasActive ? (
          <button
            type="button"
            onClick={() => dispatch({ type: "CLEAR_FILTERS", tab: activeTab })}
            className="flex items-center gap-0.5 text-[11px] text-slate-500 hover:text-slate-300"
          >
            <X className="h-3 w-3" />
            Clear
          </button>
        ) : undefined
      }
    >
      <div>
        <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-slate-500">Sort by</p>
        <SortControls sort={sort} tab={activeTab} dispatch={dispatch} />
      </div>
      {activeTab === "backlog" && <BacklogFilterChips filters={backlogFilters} dispatch={dispatch} />}
      {activeTab === "captures" && <CaptureFilterChips filters={captureFilters} dispatch={dispatch} />}
      {activeTab === "executions" && <ExecutionFilterChips filters={executionFilters} dispatch={dispatch} />}
      {activeTab === "sessions" && <SessionFilterChips filters={sessionFilters} dispatch={dispatch} />}
    </CollapsibleSection>
  );
}
