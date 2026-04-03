import { Input } from "../ui/input";
import { Select } from "../ui/select";
import {
  EXECUTION_MODES,
  EXECUTION_STATUSES,
  formatExecutionMode,
  formatExecutionStatus,
  type ExecutionMode,
  type ExecutionStatus,
} from "../../types";

export interface ExecutionFilterValues {
  statusFilter: ExecutionStatus | "";
  modeFilter: ExecutionMode | "";
  operationFilter: "generator" | "improver" | "";
  startedByFilter: string;
  backlogFilter: string;
  fromFilter: string;
  toFilter: string;
}

export interface ExecutionFiltersProps extends ExecutionFilterValues {
  activeFilterCount: number;
  onStatusFilterChange: (value: ExecutionStatus | "") => void;
  onModeFilterChange: (value: ExecutionMode | "") => void;
  onOperationFilterChange: (value: "generator" | "improver" | "") => void;
  onStartedByFilterChange: (value: string) => void;
  onBacklogFilterChange: (value: string) => void;
  onFromFilterChange: (value: string) => void;
  onToFilterChange: (value: string) => void;
  onClearAll: () => void;
}

export function ExecutionFilters({
  statusFilter,
  modeFilter,
  operationFilter,
  startedByFilter,
  backlogFilter,
  fromFilter,
  toFilter,
  activeFilterCount,
  onStatusFilterChange,
  onModeFilterChange,
  onOperationFilterChange,
  onStartedByFilterChange,
  onBacklogFilterChange,
  onFromFilterChange,
  onToFilterChange,
  onClearAll,
}: ExecutionFiltersProps) {
  return (
    <div className="absolute left-0 top-full z-10 mt-2 w-80 space-y-3 rounded-lg border border-white/10 bg-slate-800 p-3 shadow-xl">
      <div className="mb-2 flex items-center justify-between">
        <span className="text-sm font-medium text-slate-200">Filters</span>
        {activeFilterCount > 0 ? (
          <button onClick={onClearAll} className="text-xs text-slate-400 hover:text-slate-200">
            Clear all
          </button>
        ) : null}
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div className="space-y-1">
          <label htmlFor="execution-status-filter" className="text-xs text-slate-400">Status</label>
          <Select
            id="execution-status-filter"
            value={statusFilter}
            onChange={(event) => onStatusFilterChange(event.target.value as ExecutionStatus | "")}
            variant="filter"
            withChevron
          >
            <option value="">All statuses</option>
            {EXECUTION_STATUSES.map((option) => (
              <option key={option} value={option}>{formatExecutionStatus(option)}</option>
            ))}
          </Select>
        </div>

        <div className="space-y-1">
          <label htmlFor="execution-mode-filter" className="text-xs text-slate-400">Mode</label>
          <Select
            id="execution-mode-filter"
            value={modeFilter}
            onChange={(event) => onModeFilterChange(event.target.value as ExecutionMode | "")}
            variant="filter"
            withChevron
          >
            <option value="">All modes</option>
            {EXECUTION_MODES.map((option) => (
              <option key={option} value={option}>{formatExecutionMode(option)}</option>
            ))}
          </Select>
        </div>
      </div>

      <div className="space-y-1">
        <label htmlFor="execution-operation-filter" className="text-xs text-slate-400">Operation</label>
        <Select
          id="execution-operation-filter"
          value={operationFilter}
          onChange={(event) => onOperationFilterChange(event.target.value as "generator" | "improver" | "")}
          variant="filter"
          withChevron
        >
          <option value="">All operations</option>
          <option value="generator">Generator</option>
          <option value="improver">Improver</option>
        </Select>
      </div>

      <div className="space-y-1">
        <label htmlFor="execution-started-by-filter" className="text-xs text-slate-400">Started by</label>
        <Input
          id="execution-started-by-filter"
          size="sm"
          placeholder="team source"
          value={startedByFilter}
          onChange={(event) => onStartedByFilterChange(event.target.value)}
        />
      </div>

      <div className="space-y-1">
        <label htmlFor="execution-backlog-filter" className="text-xs text-slate-400">Scenario/backlog</label>
        <Input
          id="execution-backlog-filter"
          size="sm"
          placeholder="kind/name"
          value={backlogFilter}
          onChange={(event) => onBacklogFilterChange(event.target.value)}
        />
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div className="space-y-1">
          <label htmlFor="execution-created-from" className="text-xs text-slate-400">Created from</label>
          <Input
            id="execution-created-from"
            type="datetime-local"
            size="sm"
            value={fromFilter}
            onChange={(event) => onFromFilterChange(event.target.value)}
          />
        </div>
        <div className="space-y-1">
          <label htmlFor="execution-created-to" className="text-xs text-slate-400">Created to</label>
          <Input
            id="execution-created-to"
            type="datetime-local"
            size="sm"
            value={toFilter}
            onChange={(event) => onToFilterChange(event.target.value)}
          />
        </div>
      </div>
    </div>
  );
}
