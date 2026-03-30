/**
 * FilterBar - Collapsible per-tab filter and sort controls.
 */

import { useState } from "react";
import { ArrowDown, ArrowUp, ChevronDown, ChevronUp, X } from "lucide-react";
import { cn } from "../../../../lib/utils";
import type { SidebarAction } from "./useSidebarState";
import type { BacklogFilters, CaptureFilters, ExecutionFilters, InitiativeFilters, SidebarTab, SortConfig, SortDirection, SortField } from "./types";
import { DEFAULT_SORT } from "./types";

interface FilterBarProps {
  activeTab: SidebarTab;
  backlogFilters: BacklogFilters;
  captureFilters: CaptureFilters;
  initiativeFilters: InitiativeFilters;
  executionFilters: ExecutionFilters;
  sort: SortConfig;
  dispatch: React.Dispatch<SidebarAction>;
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

function SortControls({ sort, tab, dispatch }: { sort: SortConfig; tab: SidebarTab; dispatch: React.Dispatch<SidebarAction> }) {
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

const BACKLOG_STATUSES = ["backlog", "researching", "ready", "queued", "in_progress", "completed", "failed", "archived"] as const;
const BACKLOG_KINDS = ["idea", "research", "fix", "execute", "chore"] as const;
const CAPTURE_STATUSES = ["classifying", "classified", "failed"] as const;
const INITIATIVE_STATUSES = ["active", "completed", "archived"] as const;
const EXECUTION_STATUSES = ["pending", "starting", "running", "needs_review", "validating", "needs_fixup", "completed", "failed", "canceled"] as const;
const EXECUTION_MODES = ["manual", "yolo"] as const;

function toggleInArray<T>(arr: T[], value: T): T[] {
  return arr.includes(value) ? arr.filter((v) => v !== value) : [...arr, value];
}

function BacklogFilterChips({ filters, dispatch }: { filters: BacklogFilters; dispatch: React.Dispatch<SidebarAction> }) {
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
    </>
  );
}

function CaptureFilterChips({ filters, dispatch }: { filters: CaptureFilters; dispatch: React.Dispatch<SidebarAction> }) {
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

function InitiativeFilterChips({ filters, dispatch }: { filters: InitiativeFilters; dispatch: React.Dispatch<SidebarAction> }) {
  return (
    <div>
      <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-slate-500">Status</p>
      <div className="flex flex-wrap gap-1">
        {INITIATIVE_STATUSES.map((s) => (
          <Chip
            key={s}
            label={s}
            active={filters.statuses.includes(s)}
            onClick={() => dispatch({ type: "SET_INITIATIVE_FILTERS", filters: { statuses: toggleInArray(filters.statuses, s) } })}
          />
        ))}
      </div>
    </div>
  );
}

function ExecutionFilterChips({ filters, dispatch }: { filters: ExecutionFilters; dispatch: React.Dispatch<SidebarAction> }) {
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

// ============================================================================
// FilterBar
// ============================================================================

function hasActiveFiltersForTab(tab: SidebarTab, backlog: BacklogFilters, captures: CaptureFilters, initiatives: InitiativeFilters, executions: ExecutionFilters, sort: SortConfig): boolean {
  const defaultSort = DEFAULT_SORT[tab];
  const sortChanged = sort.field !== defaultSort.field || sort.direction !== defaultSort.direction;

  switch (tab) {
    case "activity": return sortChanged;
    case "backlog": return backlog.statuses.length > 0 || backlog.kinds.length > 0 || backlog.priorityMin !== null || backlog.priorityMax !== null || sortChanged;
    case "captures": return captures.statuses.length > 0 || sortChanged;
    case "initiatives": return initiatives.statuses.length > 0 || sortChanged;
    case "executions": return executions.statuses.length > 0 || executions.modes.length > 0 || sortChanged;
  }
}

export function FilterBar({ activeTab, backlogFilters, captureFilters, initiativeFilters, executionFilters, sort, dispatch }: FilterBarProps) {
  const [expanded, setExpanded] = useState(false);

  if (activeTab === "activity") return null;

  const hasActive = hasActiveFiltersForTab(activeTab, backlogFilters, captureFilters, initiativeFilters, executionFilters, sort);

  return (
    <div className="border-b border-slate-200/20">
      <div className="flex items-center justify-between px-3 py-1.5">
        <button
          type="button"
          onClick={() => setExpanded(!expanded)}
          className="flex items-center gap-1 text-[11px] font-medium text-slate-400 hover:text-slate-200"
          data-testid="filter-bar-toggle"
        >
          {expanded ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
          Filters & Sort
          {hasActive && <span className="ml-1 h-1.5 w-1.5 rounded-full bg-cyan-400" />}
        </button>
        {hasActive && (
          <button
            type="button"
            onClick={() => dispatch({ type: "CLEAR_FILTERS", tab: activeTab })}
            className="flex items-center gap-0.5 text-[11px] text-slate-500 hover:text-slate-300"
          >
            <X className="h-3 w-3" />
            Clear
          </button>
        )}
      </div>
      {expanded && (
        <div className="space-y-2.5 px-3 pb-2.5" data-testid="filter-bar-content">
          <div>
            <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-slate-500">Sort by</p>
            <SortControls sort={sort} tab={activeTab} dispatch={dispatch} />
          </div>
          {activeTab === "backlog" && <BacklogFilterChips filters={backlogFilters} dispatch={dispatch} />}
          {activeTab === "captures" && <CaptureFilterChips filters={captureFilters} dispatch={dispatch} />}
          {activeTab === "initiatives" && <InitiativeFilterChips filters={initiativeFilters} dispatch={dispatch} />}
          {activeTab === "executions" && <ExecutionFilterChips filters={executionFilters} dispatch={dispatch} />}
        </div>
      )}
    </div>
  );
}
