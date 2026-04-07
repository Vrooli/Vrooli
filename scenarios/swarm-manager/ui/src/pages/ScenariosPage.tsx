/**
 * Scenarios Page
 *
 * Displays the scenario catalog with search, filtering, and status information.
 * [REQ:REQ-P0-006] Prioritized scenario catalog with search and filter
 *
 * Responsibility: Presentation only - rendering scenarios and handling user interactions.
 * Data fetching is delegated to the scenarios store, which uses the scenarios service seam.
 * Domain logic (types, status icons/colors) is imported from the types module.
 *
 * Error Handling:
 * - Clearly distinguishes between "no scenarios" (empty state) and "error loading" (error state)
 * - Shows user-friendly error messages based on error type
 * - Provides retry functionality for recoverable errors
 *
 * Experience Architecture (Phase 29):
 * - Status summary shows running/error counts for quick health assessment
 * - Quick status filter buttons allow one-click filtering (iteration 4)
 * - Helps ecosystem reviewers quickly understand ecosystem state
 */

import { useState, useMemo, useEffect, type MouseEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Filter, Package, X } from "lucide-react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ErrorState } from "../components/ui/error-state";
import { InlineLoadingIndicator, PageLoadingState } from "../components/ui/loading-states";
import { ResponsiveList } from "../components/ui/responsive-list";
import { SearchBar } from "../components/ui/search-bar";
import { Select } from "../components/ui/select";
import { capitalize } from "../lib";
import { scenariosService } from "../services";
import { selectors } from "../consts/selectors";
import type { ScenarioStatus } from "../types";
import { useScenariosStore } from "../stores";
import { ScenarioCard, type ScenarioAction } from "./ScenarioCard";
import { ScenarioStatusSummary } from "./ScenarioStatusSummary";

/** Available status values for filtering */
const STATUS_OPTIONS: ScenarioStatus[] = ["running", "stopped", "error", "unknown"];

export function ScenariosPage() {
  // Search and filter state
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState<ScenarioStatus | "">("");
  const [showFilters, setShowFilters] = useState(false);
  const navigate = useNavigate();
  const scenarios = useScenariosStore((state) => state.scenarios);
  const status = useScenariosStore((state) => state.status);
  const error = useScenariosStore((state) => state.error);
  const isRefreshing = useScenariosStore((state) => state.isRefreshing);
  const fetchScenarios = useScenariosStore((state) => state.fetchScenarios);
  const upsertScenario = useScenariosStore((state) => state.upsertScenario);
  const hasLoaded = status !== "idle";

  useEffect(() => {
    void fetchScenarios();
  }, [fetchScenarios]);

  // Client-side filtering and searching
  // API supports server-side filtering, but client-side provides instant feedback
  const filteredScenarios = useMemo(() => {
    if (!scenarios) return [];

    let result = scenarios;

    // Apply search filter (name, displayName, or description)
    if (searchTerm.trim()) {
      const search = searchTerm.toLowerCase();
      result = result.filter(
        (s) =>
          s.name.toLowerCase().includes(search) ||
          s.displayName.toLowerCase().includes(search) ||
          s.description.toLowerCase().includes(search)
      );
    }

    // Apply status filter
    if (statusFilter) {
      result = result.filter((s) => s.status === statusFilter);
    }

    // Already sorted by priority from API, but ensure consistent ordering
    return result.sort((a, b) => {
      if (a.priority !== b.priority) {
        return a.priority - b.priority;
      }
      return a.name.localeCompare(b.name);
    });
  }, [scenarios, searchTerm, statusFilter]);

  const actionMutation = useMutation({
    mutationFn: ({ name, action }: { name: string; action: ScenarioAction }) => {
      if (action === "start") {
        return scenariosService.start(name);
      }
      if (action === "stop") {
        return scenariosService.stop(name);
      }
      return scenariosService.restart(name);
    },
    onSuccess: (updatedScenario) => {
      upsertScenario(updatedScenario);
    },
  });

  const actionError = actionMutation.isError
    ? actionMutation.error instanceof Error ? actionMutation.error.message
      : `Failed to ${actionMutation.variables?.action ?? "run action"}. Please try again.`
    : null;

  const isActionPending = (scenarioName: string, action: ScenarioAction) =>
    actionMutation.isPending &&
    actionMutation.variables?.name === scenarioName &&
    actionMutation.variables?.action === action;

  const handleAction = (
    event: MouseEvent<HTMLButtonElement>,
    scenarioName: string,
    action: ScenarioAction
  ) => {
    event.preventDefault();
    event.stopPropagation();
    actionMutation.mutate({ name: scenarioName, action });
  };

  // Count active filters for badge
  const activeFilterCount = (statusFilter ? 1 : 0);

  // Compute status summary for quick health overview (Phase 29)
  // Shows running/error counts to help users quickly assess ecosystem health
  const statusSummary = useMemo(() => {
    if (!scenarios) return { running: 0, stopped: 0, error: 0 };
    return {
      running: scenarios.filter((s) => s.status === "running").length,
      stopped: scenarios.filter((s) => s.status === "stopped").length,
      error: scenarios.filter((s) => s.status === "error").length,
    };
  }, [scenarios]);

  // Clear all filters
  const clearFilters = () => {
    setSearchTerm("");
    setStatusFilter("");
    setShowFilters(false);
  };

  return (
    <div className="space-y-6" data-testid={selectors.scenarios.page}>
      {/* Header actions */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2">
          <SearchBar
            placeholder="Search scenarios..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            data-testid={selectors.scenarios.search}
          />
          <div className="relative">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowFilters(!showFilters)}
              data-testid={selectors.scenarios.filter}
              className={activeFilterCount > 0 ? "border-cyan-500/50" : ""}
              aria-label="Filter scenarios"
            >
              <Filter className="h-4 w-4" />
              {activeFilterCount > 0 && (
                <span className="ml-1 rounded-full bg-cyan-500 px-1.5 text-xs text-white">
                  {activeFilterCount}
                </span>
              )}
            </Button>

            {/* Filter dropdown */}
            {showFilters && (
              <div
                className="absolute right-0 top-full z-10 mt-2 w-56 rounded-lg border border-white/10 bg-slate-800 p-3 shadow-xl"
                data-testid={selectors.scenarios.filterDropdown}
              >
                <div className="mb-2 flex items-center justify-between">
                  <span className="text-sm font-medium text-slate-200">Filters</span>
                  {activeFilterCount > 0 && (
                    <button
                      onClick={clearFilters}
                      className="text-xs text-slate-400 hover:text-slate-200"
                    >
                      Clear all
                    </button>
                  )}
                </div>

                {/* Status filter */}
                <div className="space-y-1">
                  <label htmlFor="scenarios-status-filter" className="text-xs text-slate-400">
                    Status
                  </label>
                  <Select
                    id="scenarios-status-filter"
                    value={statusFilter}
                    onChange={(e) => setStatusFilter(e.target.value as ScenarioStatus | "")}
                    variant="filter"
                    withChevron
                    data-testid={selectors.scenarios.statusFilter}
                  >
                    <option value="">All statuses</option>
                    {STATUS_OPTIONS.map((status) => (
                      <option key={status} value={status}>
                        {capitalize(status)}
                      </option>
                    ))}
                  </Select>
                </div>
              </div>
            )}
          </div>
        </div>
        <div className="flex items-center gap-3">
          {/* Active filter chips */}
          {searchTerm && (
            <span className="flex items-center gap-1 rounded-full bg-slate-700 px-2 py-0.5 text-xs text-slate-300">
              Search: "{searchTerm.slice(0, 15)}{searchTerm.length > 15 ? '...' : ''}"
              <button onClick={() => setSearchTerm("")} className="ml-1 hover:text-white" aria-label="Clear search">
                <X className="h-3 w-3" />
              </button>
            </span>
          )}
          {statusFilter && (
            <span className="flex items-center gap-1 rounded-full bg-slate-700 px-2 py-0.5 text-xs text-slate-300">
              Status: {statusFilter}
              <button
                onClick={() => setStatusFilter("")}
                className="ml-1 hover:text-white"
                aria-label="Clear status filter"
              >
                <X className="h-3 w-3" />
              </button>
            </span>
          )}
          <span className="text-sm text-slate-400" data-testid={selectors.scenarios.count}>
            {filteredScenarios.length} scenario{filteredScenarios.length !== 1 ? 's' : ''}
            {scenarios && filteredScenarios.length !== scenarios.length && (
              <span className="text-slate-500"> of {scenarios.length}</span>
            )}
          </span>
          {isRefreshing && scenarios.length > 0 && (
            <InlineLoadingIndicator
              label="Refreshing scenarios..."
              testId="scenarios-refreshing-indicator"
            />
          )}
          {/* Quick status filters - one-click filtering for common jobs (Phase 29 iter 4) */}
          {scenarios && scenarios.length > 0 && (
            <ScenarioStatusSummary
              summary={statusSummary}
              activeFilter={statusFilter}
              onFilterToggle={setStatusFilter}
            />
          )}
        </div>
      </div>

      {/* Scenarios list */}
      <div className="space-y-4">
        {actionError && (
          <Card padding="sm">
            <div className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
              {actionError}
            </div>
          </Card>
        )}
        {(status === "loading" || !hasLoaded) && scenarios.length === 0 && (
          <PageLoadingState
            label="Loading scenarios..."
            variant="list"
            testId="scenarios-loading-state"
          />
        )}

        {/* Error state - clearly different from empty state */}
        {error && scenarios.length === 0 && hasLoaded && (
          <ErrorState
            error={error}
            title="Unable to load scenarios"
            onRetry={() => fetchScenarios({ force: true })}
          />
        )}

        {/* Empty state - only shown when we successfully loaded but have no data */}
        {scenarios.length === 0 && hasLoaded && (
          <Card padding="lg" centered data-testid={selectors.scenarios.empty}>
            <Package className="mx-auto h-12 w-12 text-slate-600" />
            <h3 className="mt-4 text-lg font-medium text-slate-300">No scenarios found</h3>
            <p className="mt-2 text-sm text-slate-400">
              Your ecosystem doesn't have any scenarios yet
            </p>
          </Card>
        )}

        {/* No results after filtering */}
        {scenarios && scenarios.length > 0 && filteredScenarios.length === 0 && (
          <Card padding="lg" centered data-testid={selectors.scenarios.noResults}>
            <Package className="mx-auto h-12 w-12 text-slate-600" />
            <h3 className="mt-4 text-lg font-medium text-slate-300">No matching scenarios</h3>
            <p className="mt-2 text-sm text-slate-400">
              Try adjusting your search or filter criteria
            </p>
            <Button
              variant="outline"
              size="sm"
              className="mt-4"
              onClick={clearFilters}
            >
              Clear filters
            </Button>
          </Card>
        )}

        {filteredScenarios.length > 0 && (
          <ResponsiveList
            data-testid={selectors.scenarios.list}
            columns="md:grid-cols-2 xl:grid-cols-3"
          >
            {filteredScenarios.map((scenario) => (
              <ScenarioCard
                key={scenario.name}
                name={scenario.name}
                displayName={scenario.displayName}
                description={scenario.description}
                status={scenario.status}
                priority={scenario.priority}
                isGreenfield={scenario.isGreenfield}
                tags={scenario.tags}
                completenessScore={scenario.completenessScore}
                lastReviewClassification={scenario.lastReviewClassification}
                lastReviewAt={scenario.lastReviewAt}
                isAnyActionPending={actionMutation.isPending}
                isActionPending={(action) => isActionPending(scenario.name, action)}
                onAction={(event, action) => handleAction(event, scenario.name, action)}
                onNavigate={() => navigate(`/scenarios/${scenario.name}`)}
              />
            ))}
          </ResponsiveList>
        )}
      </div>
    </div>
  );
}
