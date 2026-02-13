import { useEffect, useMemo, useState } from "react";
import { Activity, Filter, RefreshCw, X } from "lucide-react";
import { Card } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { ErrorState } from "../components/ui/error-state";
import { Input } from "../components/ui/input";
import { ResponsiveList, ResponsiveListItem } from "../components/ui/responsive-list";
import { SearchBar } from "../components/ui/search-bar";
import { Select } from "../components/ui/select";
import { Tabs, TabsList, TabsTrigger } from "../components/ui/tabs";
import { ExecutionCard } from "../components/execution/execution-card";
import { selectors } from "../consts/selectors";
import {
  canCancelExecution,
  canRetryExecution,
  canStartExecution,
  EXECUTION_TAB_CONFIG,
  isExecutionActive,
  isExecutionInTab,
  matchesExecutionFilters,
  type ExecutionTabId,
} from "../lib";
import { executionService } from "../services";
import { useExecutionStore } from "../stores";
import {
  EXECUTION_MODES,
  EXECUTION_STATUSES,
  formatExecutionMode,
  formatExecutionStatus,
  type ExecutionMode,
  type ExecutionRecord,
  type ExecutionStatus,
} from "../types";

const AUTO_REFRESH_MS = 6000;

export function ExecutionPage() {
  const { items, status, error, isRefreshing, fetchExecutions, upsertExecution } = useExecutionStore();

  const [activeTab, setActiveTab] = useState<ExecutionTabId>("all");
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState<ExecutionStatus | "">("");
  const [modeFilter, setModeFilter] = useState<ExecutionMode | "">("");
  const [operationFilter, setOperationFilter] = useState<"generator" | "improver" | "">("");
  const [startedByFilter, setStartedByFilter] = useState("");
  const [backlogFilter, setBacklogFilter] = useState("");
  const [fromFilter, setFromFilter] = useState("");
  const [toFilter, setToFilter] = useState("");
  const [showFilters, setShowFilters] = useState(false);

  const [busyId, setBusyId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchExecutions();
    const interval = window.setInterval(() => {
      void fetchExecutions({ force: true });
    }, AUTO_REFRESH_MS);
    return () => window.clearInterval(interval);
  }, [fetchExecutions]);

  const activeTabConfig = EXECUTION_TAB_CONFIG.find((tab) => tab.id === activeTab) ?? EXECUTION_TAB_CONFIG[0];
  const hasLoaded = status !== "idle";

  const tabItems = useMemo(
    () => items.filter((item) => isExecutionInTab(item, activeTab)),
    [items, activeTab]
  );

  const filteredItems = useMemo(() => {
    return tabItems.filter((item) =>
      matchesExecutionFilters(item, {
        searchTerm,
        statusFilter,
        modeFilter,
        operationFilter,
        startedByFilter,
        backlogFilter,
        fromFilter,
        toFilter,
      })
    );
  }, [tabItems, searchTerm, statusFilter, modeFilter, operationFilter, startedByFilter, backlogFilter, fromFilter, toFilter]);

  const activeRuns = useMemo(() => {
    return filteredItems.filter(isExecutionActive).slice(0, 3);
  }, [filteredItems]);

  const stats = useMemo(() => {
    const running = tabItems.filter((item) => item.status === "running").length;
    const failed = tabItems.filter((item) => item.status === "failed" || item.status === "canceled").length;
    return {
      total: tabItems.length,
      running,
      failed,
    };
  }, [tabItems]);

  const activeFilterCount = [
    searchTerm,
    statusFilter,
    modeFilter,
    operationFilter,
    startedByFilter,
    backlogFilter,
    fromFilter,
    toFilter,
  ].filter((value) => value.trim().length > 0).length;

  const clearFilters = () => {
    setSearchTerm("");
    setStatusFilter("");
    setModeFilter("");
    setOperationFilter("");
    setStartedByFilter("");
    setBacklogFilter("");
    setFromFilter("");
    setToFilter("");
    setShowFilters(false);
  };

  const runAction = async (executionId: string, action: "start" | "cancel" | "retry") => {
    setBusyId(executionId);
    setActionError(null);
    try {
      let updated: ExecutionRecord;
      if (action === "start") {
        updated = await executionService.start(executionId);
      } else if (action === "cancel") {
        updated = await executionService.cancel(executionId);
      } else {
        updated = await executionService.retry(executionId);
      }
      upsertExecution(updated);
    } catch (actionErr) {
      const message = actionErr instanceof Error ? actionErr.message : `Failed to ${action} execution.`;
      setActionError(message);
    } finally {
      setBusyId(null);
    }
  };

  if (!activeTabConfig) {
    return null;
  }

  return (
    <div className="space-y-6" data-testid={selectors.execution.page}>
      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as ExecutionTabId)} className="w-full">
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
            onClick={() => void fetchExecutions({ force: true })}
            disabled={status === "loading" || isRefreshing}
          >
            <RefreshCw className={`mr-2 h-4 w-4 ${status === "loading" || isRefreshing ? "animate-spin" : ""}`} />
            Refresh
          </Button>
        </div>

        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-wrap items-center gap-2">
            <SearchBar
              placeholder="Search executions..."
              value={searchTerm}
              onChange={(event) => setSearchTerm(event.target.value)}
              data-testid={selectors.execution.search}
              widthClass="w-full sm:w-80"
            />

            <div className="relative">
              <Button
                variant="outline"
                size="sm"
                aria-label="Filter executions"
                data-testid={selectors.execution.filter}
                onClick={() => setShowFilters(!showFilters)}
                className={activeFilterCount > 0 ? "border-cyan-500/50" : ""}
              >
                <Filter className="h-4 w-4" />
                {activeFilterCount > 0 ? (
                  <span className="ml-1 rounded-full bg-cyan-500 px-1.5 text-xs text-white">{activeFilterCount}</span>
                ) : null}
              </Button>

              {showFilters ? (
                <div className="absolute left-0 top-full z-10 mt-2 w-80 space-y-3 rounded-lg border border-white/10 bg-slate-800 p-3 shadow-xl">
                  <div className="mb-2 flex items-center justify-between">
                    <span className="text-sm font-medium text-slate-200">Filters</span>
                    {activeFilterCount > 0 ? (
                      <button onClick={clearFilters} className="text-xs text-slate-400 hover:text-slate-200">
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
                        onChange={(event) => setStatusFilter(event.target.value as ExecutionStatus | "")}
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
                        onChange={(event) => setModeFilter(event.target.value as ExecutionMode | "")}
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
                      onChange={(event) => setOperationFilter(event.target.value as "generator" | "improver" | "")}
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
                      onChange={(event) => setStartedByFilter(event.target.value)}
                    />
                  </div>

                  <div className="space-y-1">
                    <label htmlFor="execution-backlog-filter" className="text-xs text-slate-400">Scenario/backlog</label>
                    <Input
                      id="execution-backlog-filter"
                      size="sm"
                      placeholder="kind/name"
                      value={backlogFilter}
                      onChange={(event) => setBacklogFilter(event.target.value)}
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
                        onChange={(event) => setFromFilter(event.target.value)}
                      />
                    </div>
                    <div className="space-y-1">
                      <label htmlFor="execution-created-to" className="text-xs text-slate-400">Created to</label>
                      <Input
                        id="execution-created-to"
                        type="datetime-local"
                        size="sm"
                        value={toFilter}
                        onChange={(event) => setToFilter(event.target.value)}
                      />
                    </div>
                  </div>
                </div>
              ) : null}
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3 text-sm text-slate-400">
            <span>{stats.total} run{stats.total !== 1 ? "s" : ""}</span>
            {stats.running > 0 ? <span className="text-cyan-400">{stats.running} running</span> : null}
            {stats.failed > 0 ? <span className="text-amber-400">{stats.failed} failed/canceled</span> : null}
          </div>
        </div>
      </div>

      {searchTerm ? (
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <span>
            Showing results for <span className="text-slate-200">"{searchTerm}"</span>
          </span>
          <button onClick={() => setSearchTerm("")} className="text-slate-400 hover:text-slate-200" aria-label="Clear search">
            <X className="h-4 w-4" />
          </button>
        </div>
      ) : null}

      {actionError ? (
        <Card className="border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-sm text-amber-200">
          {actionError}
        </Card>
      ) : null}

      <div className="space-y-4">
        {(status === "loading" || !hasLoaded) && items.length === 0 ? (
          <Card padding="lg" centered>
            <p className="text-slate-400">Loading execution runs...</p>
          </Card>
        ) : null}

        {error && items.length === 0 && hasLoaded ? (
          <ErrorState
            error={error}
            title="Unable to load execution runs"
            onRetry={() => fetchExecutions({ force: true })}
          />
        ) : null}

        {!error && tabItems.length === 0 && hasLoaded ? (
          <Card padding="lg" centered data-testid={selectors.execution.empty}>
            <Activity className="mx-auto h-12 w-12 text-slate-600" />
            <h3 className="mt-4 text-lg font-medium text-slate-300">{activeTabConfig.emptyTitle}</h3>
            <p className="mt-2 max-w-md text-sm text-slate-400">{activeTabConfig.emptyDescription}</p>
          </Card>
        ) : null}

        {tabItems.length > 0 && filteredItems.length === 0 ? (
          <Card padding="lg" centered data-testid={selectors.execution.noResults}>
            <Activity className="mx-auto h-12 w-12 text-slate-600" />
            <h3 className="mt-4 text-lg font-medium text-slate-300">No matching runs</h3>
            <p className="mt-2 text-sm text-slate-400">Try adjusting your search or filter criteria.</p>
            <Button variant="outline" size="sm" className="mt-4" onClick={clearFilters}>
              Clear filters
            </Button>
          </Card>
        ) : null}

        {activeRuns.length > 0 ? (
          <div className="space-y-3" data-testid={selectors.execution.activeSection}>
            <div className="flex items-center gap-2 text-sm font-medium text-slate-300">
              <Activity className="h-4 w-4 text-cyan-400" />
              <span>Active Runs</span>
            </div>
            <ResponsiveList data-testid={selectors.execution.activeList} columns="md:grid-cols-1 lg:grid-cols-2">
              {activeRuns.map((item) => (
                <ResponsiveListItem
                  key={`active-${item.executionId}`}
                  className="md:border-cyan-500/30 md:bg-cyan-500/5 md:hover:border-cyan-500/50 md:hover:bg-cyan-500/10"
                >
                  <ExecutionCard
                    item={item}
                    isBusy={busyId === item.executionId}
                    canStart={canStartExecution(item.status)}
                    canCancel={canCancelExecution(item.status)}
                    canRetry={canRetryExecution(item.status)}
                    onStart={(executionId) => void runAction(executionId, "start")}
                    onCancel={(executionId) => void runAction(executionId, "cancel")}
                    onRetry={(executionId) => void runAction(executionId, "retry")}
                  />
                </ResponsiveListItem>
              ))}
            </ResponsiveList>
          </div>
        ) : null}

        {filteredItems.length > 0 ? (
          <div className="space-y-3">
            {activeRuns.length > 0 ? (
              <div className="text-sm font-medium text-slate-400">All {activeTabConfig.label}</div>
            ) : null}
            <ResponsiveList data-testid={selectors.execution.grid} columns="md:grid-cols-1 lg:grid-cols-2 xl:grid-cols-3">
              {filteredItems.map((item) => (
                <ResponsiveListItem key={item.executionId} interactive>
                  <ExecutionCard
                    item={item}
                    isBusy={busyId === item.executionId}
                    canStart={canStartExecution(item.status)}
                    canCancel={canCancelExecution(item.status)}
                    canRetry={canRetryExecution(item.status)}
                    onStart={(executionId) => void runAction(executionId, "start")}
                    onCancel={(executionId) => void runAction(executionId, "cancel")}
                    onRetry={(executionId) => void runAction(executionId, "retry")}
                  />
                </ResponsiveListItem>
              ))}
            </ResponsiveList>
          </div>
        ) : null}
      </div>
    </div>
  );
}
