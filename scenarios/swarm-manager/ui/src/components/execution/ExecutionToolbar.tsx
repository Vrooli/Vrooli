import type { ChangeEvent } from "react";
import { Filter, RefreshCw } from "lucide-react";
import { Button } from "../ui/button";
import { SearchBar } from "../ui/search-bar";
import { Tabs, TabsList, TabsTrigger } from "../ui/tabs";
import { ExecutionFilters } from "./ExecutionFilters";
import { selectors } from "../../consts/selectors";
import { EXECUTION_TAB_CONFIG, isExecutionInTab, type ExecutionTabId } from "../../lib";
import type { ExecutionMode, ExecutionRecord, ExecutionStatus } from "../../types";

export interface ExecutionToolbarProps {
  items: ExecutionRecord[];
  activeTab: ExecutionTabId;
  onActiveTabChange: (tab: ExecutionTabId) => void;
  status: string;
  isRefreshing: boolean;
  onRefresh: () => void;
  searchTerm: string;
  onSearchChange: (event: ChangeEvent<HTMLInputElement>) => void;
  showFilters: boolean;
  onToggleFilters: () => void;
  activeFilterCount: number;
  // Filter values
  statusFilter: ExecutionStatus | "";
  modeFilter: ExecutionMode | "";
  operationFilter: "generator" | "improver" | "";
  startedByFilter: string;
  backlogFilter: string;
  fromFilter: string;
  toFilter: string;
  onStatusFilterChange: (value: ExecutionStatus | "") => void;
  onModeFilterChange: (value: ExecutionMode | "") => void;
  onOperationFilterChange: (value: "generator" | "improver" | "") => void;
  onStartedByFilterChange: (value: string) => void;
  onBacklogFilterChange: (value: string) => void;
  onFromFilterChange: (value: string) => void;
  onToFilterChange: (value: string) => void;
  onClearFilters: () => void;
  // Stats
  stats: { total: number; running: number; validating: number; review: number; failed: number };
  gctAvailable: boolean | null;
}

export function ExecutionToolbar({
  items,
  activeTab,
  onActiveTabChange,
  status,
  isRefreshing,
  onRefresh,
  searchTerm,
  onSearchChange,
  showFilters,
  onToggleFilters,
  activeFilterCount,
  statusFilter,
  modeFilter,
  operationFilter,
  startedByFilter,
  backlogFilter,
  fromFilter,
  toFilter,
  onStatusFilterChange,
  onModeFilterChange,
  onOperationFilterChange,
  onStartedByFilterChange,
  onBacklogFilterChange,
  onFromFilterChange,
  onToFilterChange,
  onClearFilters,
  stats,
  gctAvailable,
}: ExecutionToolbarProps) {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <Tabs value={activeTab} onValueChange={(value) => onActiveTabChange(value as ExecutionTabId)} className="w-full">
          <div className="-mx-6 md:mx-0">
            <TabsList
              className="w-full flex-nowrap justify-start gap-1 overflow-x-auto no-scrollbar rounded-none bg-transparent p-0 md:w-auto md:flex-wrap md:overflow-visible md:rounded-md md:bg-slate-800/50 md:p-1"
              data-testid={selectors.execution.tabs}
            >
              {EXECUTION_TAB_CONFIG.map((tab) => (
                <TabsTrigger key={tab.id} value={tab.id} className="gap-2">
                  {tab.label}
                  <span className="rounded-full bg-slate-600 px-2 py-0.5 text-xs text-slate-200">
                    {items.filter((item) => isExecutionInTab(item, tab.id)).length}
                  </span>
                </TabsTrigger>
              ))}
            </TabsList>
          </div>
        </Tabs>

        <Button
          variant="outline"
          size="sm"
          onClick={onRefresh}
          disabled={status === "loading" || isRefreshing}
          aria-label="Refresh"
          className="shrink-0"
        >
          <RefreshCw className={`h-4 w-4 lg:mr-2 ${status === "loading" || isRefreshing ? "animate-spin" : ""}`} />
          <span className="hidden lg:inline">Refresh</span>
        </Button>
      </div>

      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap items-center gap-2">
          <SearchBar
            placeholder="Search executions..."
            value={searchTerm}
            onChange={onSearchChange}
            data-testid={selectors.execution.search}
            widthClass="w-full sm:w-80"
          />

          <div className="relative">
            <Button
              variant="outline"
              size="sm"
              aria-label="Filter executions"
              data-testid={selectors.execution.filter}
              onClick={onToggleFilters}
              className={activeFilterCount > 0 ? "border-cyan-500/50" : ""}
            >
              <Filter className="h-4 w-4" />
              {activeFilterCount > 0 ? (
                <span className="ml-1 rounded-full bg-cyan-500 px-1.5 text-xs text-white">{activeFilterCount}</span>
              ) : null}
            </Button>

            {showFilters ? (
              <ExecutionFilters
                statusFilter={statusFilter}
                modeFilter={modeFilter}
                operationFilter={operationFilter}
                startedByFilter={startedByFilter}
                backlogFilter={backlogFilter}
                fromFilter={fromFilter}
                toFilter={toFilter}
                activeFilterCount={activeFilterCount}
                onStatusFilterChange={onStatusFilterChange}
                onModeFilterChange={onModeFilterChange}
                onOperationFilterChange={onOperationFilterChange}
                onStartedByFilterChange={onStartedByFilterChange}
                onBacklogFilterChange={onBacklogFilterChange}
                onFromFilterChange={onFromFilterChange}
                onToFilterChange={onToFilterChange}
                onClearAll={onClearFilters}
              />
            ) : null}
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-3 text-sm text-slate-400">
          <span>{stats.total} run{stats.total !== 1 ? "s" : ""}</span>
          {stats.running > 0 ? <span className="text-cyan-400">{stats.running} running</span> : null}
          {stats.validating > 0 ? <span className="text-indigo-400">{stats.validating} validating</span> : null}
          {stats.review > 0 ? <span className="text-yellow-400">{stats.review} needs review</span> : null}
          {stats.failed > 0 ? <span className="text-amber-400">{stats.failed} failed/canceled</span> : null}
          <span className={gctAvailable ? "text-emerald-400" : "text-slate-500"} data-testid="gct-status-indicator">
            GCT: {gctAvailable === null ? "..." : gctAvailable ? "Connected" : "Offline"}
          </span>
        </div>
      </div>
    </div>
  );
}
