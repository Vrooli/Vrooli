/**
 * Scenarios Page
 *
 * Displays the scenario catalog with search, filtering, and status information.
 * [REQ:REQ-P0-006] Prioritized scenario catalog with search and filter
 *
 * Responsibility: Presentation only - rendering scenarios and handling user interactions.
 * Data fetching is delegated to the scenarios service (a seam for testability).
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

import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Filter, Package, ArrowRight, Circle, X, ChevronDown } from "lucide-react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ErrorState } from "../components/ui/error-state";
import { SearchBar } from "../components/ui/search-bar";
import { TagList } from "../components/ui/tag-list";
import { capitalize, defaultQueryOptions } from "../lib";
import { scenariosService } from "../services";
import { selectors } from "../consts/selectors";
import { SCENARIO_STATUS_ICONS, SCENARIO_STATUS_COLORS, type ScenarioStatus } from "../types";
import { displayLimitsConfig } from "../config";

/** Available status values for filtering */
const STATUS_OPTIONS: ScenarioStatus[] = ["running", "stopped", "error", "unknown"];

export function ScenariosPage() {
  // Search and filter state
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState<ScenarioStatus | "">("");
  const [showFilters, setShowFilters] = useState(false);

  const { data: scenarios, isLoading, error, refetch } = useQuery({
    queryKey: ["scenarios"],
    queryFn: () => scenariosService.list(),
    ...defaultQueryOptions,
  });

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
                  <label className="text-xs text-slate-400">Status</label>
                  <div className="relative">
                    <select
                      value={statusFilter}
                      onChange={(e) => setStatusFilter(e.target.value as ScenarioStatus | "")}
                      className="w-full appearance-none rounded-md border border-white/10 bg-slate-700 px-3 py-1.5 text-sm text-slate-200 focus:border-cyan-500 focus:outline-none"
                      data-testid={selectors.scenarios.statusFilter}
                    >
                      <option value="">All statuses</option>
                      {STATUS_OPTIONS.map((status) => (
                        <option key={status} value={status}>
                          {capitalize(status)}
                        </option>
                      ))}
                    </select>
                    <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                  </div>
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
              <button onClick={() => setSearchTerm("")} className="ml-1 hover:text-white">
                <X className="h-3 w-3" />
              </button>
            </span>
          )}
          {statusFilter && (
            <span className="flex items-center gap-1 rounded-full bg-slate-700 px-2 py-0.5 text-xs text-slate-300">
              Status: {statusFilter}
              <button onClick={() => setStatusFilter("")} className="ml-1 hover:text-white">
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
          {/* Quick status filters - one-click filtering for common jobs (Phase 29 iter 4) */}
          {scenarios && scenarios.length > 0 && (
            <div className="flex items-center gap-1" data-testid={selectors.scenarios.statusSummary}>
              {statusSummary.running > 0 && (
                <button
                  onClick={() => setStatusFilter(statusFilter === "running" ? "" : "running")}
                  className={`flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium transition-colors ${
                    statusFilter === "running"
                      ? "bg-green-500/20 text-green-300 ring-1 ring-green-500/50"
                      : "text-green-400 hover:bg-green-500/10"
                  }`}
                  data-testid={selectors.scenarios.runningCount}
                  title="Click to filter by running status"
                >
                  {statusSummary.running} running
                </button>
              )}
              {statusSummary.stopped > 0 && (
                <button
                  onClick={() => setStatusFilter(statusFilter === "stopped" ? "" : "stopped")}
                  className={`flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium transition-colors ${
                    statusFilter === "stopped"
                      ? "bg-slate-500/30 text-slate-200 ring-1 ring-slate-400/50"
                      : "text-slate-400 hover:bg-slate-500/10"
                  }`}
                  data-testid={selectors.scenarios.stoppedCount}
                  title="Click to filter by stopped status"
                >
                  {statusSummary.stopped} stopped
                </button>
              )}
              {statusSummary.error > 0 && (
                <button
                  onClick={() => setStatusFilter(statusFilter === "error" ? "" : "error")}
                  className={`flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium transition-colors ${
                    statusFilter === "error"
                      ? "bg-red-500/20 text-red-300 ring-1 ring-red-500/50"
                      : "text-red-400 hover:bg-red-500/10"
                  }`}
                  data-testid={selectors.scenarios.errorCount}
                  title="Click to filter by error status"
                >
                  {statusSummary.error} error{statusSummary.error !== 1 ? 's' : ''}
                </button>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Scenarios list */}
      <div className="space-y-4">
        {isLoading && (
          <Card padding="lg" centered>
            <p className="text-slate-400">Loading scenarios...</p>
          </Card>
        )}

        {/* Error state - clearly different from empty state */}
        {error && (
          <ErrorState
            error={error}
            title="Unable to load scenarios"
            onRetry={() => refetch()}
          />
        )}

        {/* Empty state - only shown when we successfully loaded but have no data */}
        {scenarios && scenarios.length === 0 && (
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
          <div className="space-y-3" data-testid={selectors.scenarios.list}>
            {filteredScenarios.map((scenario) => {
              const StatusIcon = SCENARIO_STATUS_ICONS[scenario.status] || Circle;
              return (
                <Link
                  key={scenario.name}
                  to={`/scenarios/${scenario.name}`}
                  className="group block cursor-pointer rounded-xl border border-white/10 bg-slate-800/30 p-4 transition hover:border-cyan-500/50 hover:bg-slate-800/50"
                  data-testid={selectors.scenarios.cardByName({ name: scenario.name })}
                >
                  <div className="flex items-start gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-3">
                        <StatusIcon
                          className={`h-4 w-4 ${SCENARIO_STATUS_COLORS[scenario.status]}`}
                        />
                        <h3 className="font-medium text-slate-100">{scenario.displayName}</h3>
                        {scenario.isGreenfield && (
                          <span className="rounded-full bg-cyan-500/20 px-2 py-0.5 text-xs text-cyan-400">
                            Greenfield
                          </span>
                        )}
                      </div>
                      <p className="mt-1 text-sm text-slate-400">{scenario.description}</p>
                      <TagList
                        tags={scenario.tags}
                        maxTags={displayLimitsConfig.scenarioCardMaxTags}
                        className="mt-2"
                      />
                    </div>
                    <div className="flex flex-col items-end gap-2">
                      <span className="rounded-full bg-slate-700 px-2 py-0.5 text-xs text-slate-300">
                        P{scenario.priority}
                      </span>
                      {scenario.completenessScore !== undefined && (
                        <div className="flex items-center gap-1">
                          <div className="h-1.5 w-16 overflow-hidden rounded-full bg-slate-700">
                            <div
                              className="h-full bg-gradient-to-r from-cyan-500 to-purple-500"
                              style={{ width: `${scenario.completenessScore}%` }}
                            />
                          </div>
                          <span className="text-xs text-slate-400">{scenario.completenessScore}%</span>
                        </div>
                      )}
                      <ArrowRight className="h-4 w-4 text-slate-500 opacity-0 transition group-hover:opacity-100" />
                    </div>
                  </div>
                </Link>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
