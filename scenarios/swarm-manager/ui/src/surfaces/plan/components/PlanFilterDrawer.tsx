/**
 * PlanFilterDrawer — the board's single shared filter surface (replacing
 * the Operations Center filter bar and the Command Post's per-section
 * toggles). Follows the SettingsDrawer FloatingPanel pattern; values are
 * URL-persisted by usePlanUrlState so filtered boards stay shareable.
 */

import { FloatingPanel } from "../../../components/ui/floating-panel";
import { Button } from "../../../components/ui/button";
import { laneLabel } from "../../../components/operations/utils";
import {
  OPERATIONS_LANES,
  type OperationsFilters,
  type OperationsViewMode,
} from "../../../types/operations";

const STATUS_OPTIONS = [
  "pending",
  "starting",
  "running",
  "needs_review",
  "complete",
  "failed",
  "cancelled",
] as const;

const OWNER_TYPE_OPTIONS = [
  "milestone",
  "backlog",
  "scenario",
  "capture",
  "session",
] as const;

export interface PlanFilterDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  filters: OperationsFilters;
  viewMode: OperationsViewMode;
  showSnoozed: boolean;
  hasActiveFilters: boolean;
  onFiltersChange: (filters: Partial<OperationsFilters>) => void;
  onViewModeChange: (mode: OperationsViewMode) => void;
  onShowSnoozedChange: (show: boolean) => void;
  onReset: () => void;
}

function FilterField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1 text-xs text-slate-400">
      <span className="font-medium uppercase tracking-wider">{label}</span>
      {children}
    </label>
  );
}

const selectClass =
  "rounded-md border border-slate-700 bg-slate-900 px-2 py-1.5 text-sm text-slate-200 focus:border-cyan-500 focus:outline-none";

export function PlanFilterDrawer({
  isOpen,
  onClose,
  filters,
  viewMode,
  showSnoozed,
  hasActiveFilters,
  onFiltersChange,
  onViewModeChange,
  onShowSnoozedChange,
  onReset,
}: PlanFilterDrawerProps) {
  return (
    <FloatingPanel
      isOpen={isOpen}
      onClose={onClose}
      title="Board Filters"
      className="max-w-md"
      testId="plan-filter-drawer"
    >
      <div className="flex flex-col gap-4">
        <FilterField label="Search">
          <input
            type="search"
            value={filters.q ?? ""}
            onChange={(e) => onFiltersChange({ q: e.target.value })}
            placeholder="Title, name, or run id…"
            className={selectClass}
            data-testid="plan-filter-search"
          />
        </FilterField>

        <div className="grid grid-cols-2 gap-3">
          <FilterField label="Status">
            <select
              value={filters.statuses?.[0] ?? ""}
              onChange={(e) => onFiltersChange({ statuses: e.target.value ? [e.target.value] : [] })}
              className={selectClass}
              data-testid="plan-filter-status"
            >
              <option value="">All</option>
              {STATUS_OPTIONS.map((status) => (
                <option key={status} value={status}>
                  {status.replaceAll("_", " ")}
                </option>
              ))}
            </select>
          </FilterField>

          <FilterField label="Lane">
            <select
              value={filters.lanes?.[0] ?? ""}
              onChange={(e) => onFiltersChange({ lanes: e.target.value ? [e.target.value] : [] })}
              className={selectClass}
              data-testid="plan-filter-lane"
            >
              <option value="">All</option>
              {OPERATIONS_LANES.map((lane) => (
                <option key={lane} value={lane}>
                  {laneLabel(lane)}
                </option>
              ))}
            </select>
          </FilterField>

          <FilterField label="Owner type">
            <select
              value={filters.ownerTypes?.[0] ?? ""}
              onChange={(e) => onFiltersChange({ ownerTypes: e.target.value ? [e.target.value] : [] })}
              className={selectClass}
              data-testid="plan-filter-owner-type"
            >
              <option value="">All</option>
              {OWNER_TYPE_OPTIONS.map((type) => (
                <option key={type} value={type}>
                  {type}
                </option>
              ))}
            </select>
          </FilterField>

          <FilterField label="Group Now by">
            <select
              value={viewMode}
              onChange={(e) => onViewModeChange(e.target.value as OperationsViewMode)}
              className={selectClass}
              data-testid="plan-filter-group-by"
            >
              <option value="by-milestone">Milestone</option>
              <option value="by-phase">Phase</option>
            </select>
          </FilterField>
        </div>

        <label className="flex items-center gap-2 text-sm text-slate-300">
          <input
            type="checkbox"
            checked={showSnoozed}
            onChange={(e) => onShowSnoozedChange(e.target.checked)}
            className="h-4 w-4 rounded border-slate-600 bg-slate-900 accent-cyan-500"
            data-testid="plan-filter-show-snoozed"
          />
          Show snoozed cards (dimmed)
        </label>

        {hasActiveFilters && (
          <Button variant="outline" size="sm" onClick={onReset} data-testid="plan-filter-reset">
            Reset filters
          </Button>
        )}
      </div>
    </FloatingPanel>
  );
}
