/**
 * FilterBar renders per-tab filter and sort controls.
 *
 * Active tab: status pills, owner/projectRoot text, createdAt /
 * lastUsedAt / fileCount sort.
 *
 * History tab: status pills (approved/rejected/deleted), owner /
 * projectRoot / runId text, snapshot date range, snapshotAt /
 * totalBlobBytes sort.
 */

import { useState } from "react";
import { ChevronDown, Filter as FilterIcon } from "lucide-react";

import type { Status } from "../../lib/api";
import {
  ACTIVE_TAB_STATUSES,
  HISTORY_TAB_STATUSES,
  type ActiveSortField,
  type HistorySortField,
  type SidebarTab,
  type TabFilters,
  type TabSorts,
} from "./types";
import type { SidebarAction } from "./useSidebarState";

const ACTIVE_SORTS: Array<{ value: ActiveSortField; label: string }> = [
  { value: "createdAt", label: "Created" },
  { value: "lastUsedAt", label: "Last used" },
  { value: "sizeBytes", label: "Size" },
  { value: "fileCount", label: "File count" },
];

const HISTORY_SORTS: Array<{ value: HistorySortField; label: string }> = [
  { value: "snapshotAt", label: "Snapshot" },
  { value: "totalBlobBytes", label: "Archive size" },
  { value: "fileCount", label: "File count" },
];

const STATUS_LABELS: Partial<Record<Status, string>> = {
  creating: "Creating",
  active: "Active",
  stopped: "Stopped",
  checkpointed: "Checkpointed",
  error: "Error",
  approved: "Approved",
  rejected: "Rejected",
  deleted: "Deleted",
};

interface FilterBarProps {
  activeTab: SidebarTab;
  filters: TabFilters;
  sorts: TabSorts;
  dispatch: React.Dispatch<SidebarAction>;
}

function StatusPills({
  options,
  selected,
  onToggle,
}: {
  options: readonly Status[];
  selected: Status[];
  onToggle: (status: Status) => void;
}) {
  return (
    <div className="flex flex-wrap gap-1" data-testid="sidebar-filter-statuses">
      {options.map((status) => {
        const active = selected.includes(status);
        return (
          <button
            key={status}
            type="button"
            onClick={() => onToggle(status)}
            className={`px-2 py-0.5 rounded text-[10px] font-medium border transition-colors ${
              active
                ? "bg-emerald-500/15 border-emerald-500/40 text-emerald-200"
                : "bg-slate-800/50 border-slate-700 text-slate-400 hover:text-slate-200"
            }`}
            data-testid={`sidebar-filter-status-${status}`}
            aria-pressed={active}
          >
            {STATUS_LABELS[status] ?? status}
          </button>
        );
      })}
    </div>
  );
}

export function FilterBar({ activeTab, filters, sorts, dispatch }: FilterBarProps) {
  const [expanded, setExpanded] = useState(false);

  if (activeTab === "active") {
    const f = filters.active;
    const s = sorts.active;
    const hasFilters =
      f.statuses.length > 0 || f.owner !== "" || f.projectRoot !== "";
    return (
      <div className="border-b border-slate-800 bg-slate-900/30" data-testid="sidebar-filter-bar">
        <button
          type="button"
          className="flex items-center justify-between w-full px-3 py-1.5 text-xs text-slate-400 hover:text-slate-200"
          onClick={() => setExpanded((v) => !v)}
          data-testid="sidebar-filter-toggle"
        >
          <span className="flex items-center gap-1.5">
            <FilterIcon className="h-3 w-3" />
            Filter & sort
            {hasFilters && (
              <span className="ml-1 px-1.5 py-0.5 rounded-full text-[9px] bg-emerald-500/20 text-emerald-300">
                on
              </span>
            )}
          </span>
          <ChevronDown
            className={`h-3 w-3 transition-transform ${expanded ? "rotate-180" : ""}`}
          />
        </button>
        {expanded && (
          <div className="px-3 pb-3 pt-1 space-y-2">
            <StatusPills
              options={ACTIVE_TAB_STATUSES}
              selected={f.statuses}
              onToggle={(status) => {
                const next = f.statuses.includes(status)
                  ? f.statuses.filter((s) => s !== status)
                  : [...f.statuses, status];
                dispatch({ type: "SET_ACTIVE_FILTERS", filters: { statuses: next } });
              }}
            />
            <input
              type="text"
              value={f.owner}
              onChange={(e) =>
                dispatch({ type: "SET_ACTIVE_FILTERS", filters: { owner: e.target.value } })
              }
              placeholder="Owner..."
              className="w-full px-2 py-1 text-xs rounded bg-slate-800/50 border border-slate-700 text-slate-200 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-emerald-500/50"
              data-testid="sidebar-filter-owner"
            />
            <input
              type="text"
              value={f.projectRoot}
              onChange={(e) =>
                dispatch({ type: "SET_ACTIVE_FILTERS", filters: { projectRoot: e.target.value } })
              }
              placeholder="Project root..."
              className="w-full px-2 py-1 text-xs rounded bg-slate-800/50 border border-slate-700 text-slate-200 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-emerald-500/50"
              data-testid="sidebar-filter-projectroot"
            />
            <div className="flex items-center gap-2">
              <label className="text-[10px] text-slate-500">Sort:</label>
              <select
                value={s.field}
                onChange={(e) =>
                  dispatch({
                    type: "SET_ACTIVE_SORT",
                    sort: { field: e.target.value as ActiveSortField },
                  })
                }
                className="flex-1 px-1.5 py-0.5 text-[11px] rounded bg-slate-800/50 border border-slate-700 text-slate-200"
                data-testid="sidebar-sort-field"
              >
                {ACTIVE_SORTS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
              <button
                type="button"
                onClick={() =>
                  dispatch({
                    type: "SET_ACTIVE_SORT",
                    sort: { direction: s.direction === "asc" ? "desc" : "asc" },
                  })
                }
                className="px-1.5 py-0.5 text-[11px] rounded bg-slate-800/50 border border-slate-700 text-slate-200"
                data-testid="sidebar-sort-direction"
              >
                {s.direction === "asc" ? "↑" : "↓"}
              </button>
            </div>
            {hasFilters && (
              <button
                type="button"
                onClick={() => dispatch({ type: "CLEAR_FILTERS", tab: "active" })}
                className="text-[10px] text-slate-500 hover:text-slate-300"
                data-testid="sidebar-filter-clear"
              >
                Clear filters
              </button>
            )}
          </div>
        )}
      </div>
    );
  }

  // History tab
  const f = filters.history;
  const s = sorts.history;
  const hasFilters =
    f.statuses.length > 0 ||
    f.owner !== "" ||
    f.projectRoot !== "" ||
    f.search !== "" ||
    f.agentManagerRunId !== "" ||
    f.snapshotAtFrom !== "" ||
    f.snapshotAtTo !== "";

  return (
    <div className="border-b border-slate-800 bg-slate-900/30" data-testid="sidebar-filter-bar">
      <button
        type="button"
        className="flex items-center justify-between w-full px-3 py-1.5 text-xs text-slate-400 hover:text-slate-200"
        onClick={() => setExpanded((v) => !v)}
        data-testid="sidebar-filter-toggle"
      >
        <span className="flex items-center gap-1.5">
          <FilterIcon className="h-3 w-3" />
          Filter & sort
          {hasFilters && (
            <span className="ml-1 px-1.5 py-0.5 rounded-full text-[9px] bg-emerald-500/20 text-emerald-300">
              on
            </span>
          )}
        </span>
        <ChevronDown className={`h-3 w-3 transition-transform ${expanded ? "rotate-180" : ""}`} />
      </button>
      {expanded && (
        <div className="px-3 pb-3 pt-1 space-y-2">
          <StatusPills
            options={HISTORY_TAB_STATUSES}
            selected={f.statuses}
            onToggle={(status) => {
              const next = f.statuses.includes(status)
                ? f.statuses.filter((s) => s !== status)
                : [...f.statuses, status];
              dispatch({ type: "SET_HISTORY_FILTERS", filters: { statuses: next } });
            }}
          />
          <input
            type="text"
            value={f.owner}
            onChange={(e) =>
              dispatch({ type: "SET_HISTORY_FILTERS", filters: { owner: e.target.value } })
            }
            placeholder="Owner..."
            className="w-full px-2 py-1 text-xs rounded bg-slate-800/50 border border-slate-700 text-slate-200 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-emerald-500/50"
            data-testid="sidebar-filter-owner"
          />
          <input
            type="text"
            value={f.projectRoot}
            onChange={(e) =>
              dispatch({ type: "SET_HISTORY_FILTERS", filters: { projectRoot: e.target.value } })
            }
            placeholder="Project root..."
            className="w-full px-2 py-1 text-xs rounded bg-slate-800/50 border border-slate-700 text-slate-200 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-emerald-500/50"
            data-testid="sidebar-filter-projectroot"
          />
          <input
            type="text"
            value={f.agentManagerRunId}
            onChange={(e) =>
              dispatch({
                type: "SET_HISTORY_FILTERS",
                filters: { agentManagerRunId: e.target.value },
              })
            }
            placeholder="Agent-manager run ID..."
            className="w-full px-2 py-1 text-xs rounded bg-slate-800/50 border border-slate-700 text-slate-200 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-emerald-500/50"
            data-testid="sidebar-filter-runid"
          />
          <div className="grid grid-cols-2 gap-2">
            <label className="flex flex-col gap-0.5">
              <span className="text-[10px] text-slate-500">From</span>
              <input
                type="date"
                value={f.snapshotAtFrom}
                onChange={(e) =>
                  dispatch({
                    type: "SET_HISTORY_FILTERS",
                    filters: { snapshotAtFrom: e.target.value },
                  })
                }
                className="px-1.5 py-1 text-xs rounded bg-slate-800/50 border border-slate-700 text-slate-200"
                data-testid="sidebar-filter-from"
              />
            </label>
            <label className="flex flex-col gap-0.5">
              <span className="text-[10px] text-slate-500">To</span>
              <input
                type="date"
                value={f.snapshotAtTo}
                onChange={(e) =>
                  dispatch({
                    type: "SET_HISTORY_FILTERS",
                    filters: { snapshotAtTo: e.target.value },
                  })
                }
                className="px-1.5 py-1 text-xs rounded bg-slate-800/50 border border-slate-700 text-slate-200"
                data-testid="sidebar-filter-to"
              />
            </label>
          </div>
          <div className="flex items-center gap-2">
            <label className="text-[10px] text-slate-500">Sort:</label>
            <select
              value={s.field}
              onChange={(e) =>
                dispatch({
                  type: "SET_HISTORY_SORT",
                  sort: { field: e.target.value as HistorySortField },
                })
              }
              className="flex-1 px-1.5 py-0.5 text-[11px] rounded bg-slate-800/50 border border-slate-700 text-slate-200"
              data-testid="sidebar-sort-field"
            >
              {HISTORY_SORTS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
            <button
              type="button"
              onClick={() =>
                dispatch({
                  type: "SET_HISTORY_SORT",
                  sort: { direction: s.direction === "asc" ? "desc" : "asc" },
                })
              }
              className="px-1.5 py-0.5 text-[11px] rounded bg-slate-800/50 border border-slate-700 text-slate-200"
              data-testid="sidebar-sort-direction"
            >
              {s.direction === "asc" ? "↑" : "↓"}
            </button>
          </div>
          {hasFilters && (
            <button
              type="button"
              onClick={() => dispatch({ type: "CLEAR_FILTERS", tab: "history" })}
              className="text-[10px] text-slate-500 hover:text-slate-300"
              data-testid="sidebar-filter-clear"
            >
              Clear filters
            </button>
          )}
        </div>
      )}
    </div>
  );
}
