/**
 * Ideas Page
 *
 * Displays the idea backlog with search, filtering, and CRUD operations.
 *
 * Responsibility: Presentation only - rendering ideas and handling user interactions.
 * Data fetching is delegated to the ideas service (a seam for testability).
 * Domain logic (types, status formatting) is imported from the types module.
 *
 * Error Handling:
 * - Clearly distinguishes between "no ideas" (empty state) and "error loading" (error state)
 * - Shows user-friendly error messages based on error type
 * - Provides retry functionality for recoverable errors
 *
 * Experience Architecture (Phase 29):
 * - "Continue Working" section surfaces recently updated non-completed ideas
 * - Summary stats show total ideas and count ready for processing
 * - Reduces cognitive load for returning users by highlighting active work
 */

import { useMemo, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Plus, Filter, Lightbulb, ArrowRight, Clock, Terminal, X, ChevronDown } from "lucide-react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ErrorState } from "../components/ui/error-state";
import { SearchBar } from "../components/ui/search-bar";
import { StatusLegend } from "../components/ui/status-legend";
import { IDEA_STATUS_LEGEND_ITEMS } from "../components/ui/status-legend.constants";
import { TagList } from "../components/ui/tag-list";
import { WelcomeHint } from "../components/ui/welcome-hint";
import { defaultQueryOptions, formatRelativeTime } from "../lib";
import { ideasService } from "../services";
import { selectors } from "../consts/selectors";
import { IDEA_STATUS_COLORS, formatIdeaStatus, type IdeaStatus } from "../types";
import { displayLimitsConfig } from "../config";
import { IdeaFormDialog } from "../components/ideas/idea-form-dialog";

/** Statuses that indicate an idea is "completed" and shouldn't appear in Continue Working */
const COMPLETED_STATUSES = ["completed", "archived"] as const;
const STATUS_OPTIONS: IdeaStatus[] = [
  "backlog",
  "researching",
  "ready",
  "queued",
  "in_progress",
  "completed",
  "archived",
];

export function IdeasPage() {
  const queryClient = useQueryClient();
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState<IdeaStatus | "">("");
  const [showFilters, setShowFilters] = useState(false);
  const [showCreate, setShowCreate] = useState(false);

  const { data: ideas, isLoading, error, refetch } = useQuery({
    queryKey: ["ideas"],
    queryFn: () => ideasService.list(),
    ...defaultQueryOptions,
  });

  const createMutation = useMutation({
    mutationFn: ideasService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ideas"] });
      setShowCreate(false);
    },
  });

  const createError = createMutation.isError ? "Failed to create idea. Please try again." : null;

  // Compute "Continue Working" items - recently updated non-completed ideas
  // Sorted by most recently updated first, limited to 3 items
  const filteredIdeas = useMemo(() => {
    if (!ideas) return [];
    let result = ideas;
    if (searchTerm.trim()) {
      const term = searchTerm.toLowerCase();
      result = result.filter(
        (idea) =>
          idea.title.toLowerCase().includes(term) ||
          idea.description.toLowerCase().includes(term) ||
          idea.name.toLowerCase().includes(term)
      );
    }
    if (statusFilter) {
      result = result.filter((idea) => idea.status === statusFilter);
    }
    return result;
  }, [ideas, searchTerm, statusFilter]);

  const continueWorkingItems = useMemo(() => {
    if (!filteredIdeas || filteredIdeas.length === 0) return [];
    return filteredIdeas
      .filter((idea) => !COMPLETED_STATUSES.includes(idea.status as typeof COMPLETED_STATUSES[number]))
      .sort((a, b) => new Date(b.updated).getTime() - new Date(a.updated).getTime())
      .slice(0, 3);
  }, [filteredIdeas]);

  // Compute summary stats
  const stats = useMemo(() => {
    if (!ideas) return { total: 0, ready: 0 };
    return {
      total: ideas.length,
      ready: ideas.filter((idea) => idea.status === "ready").length,
    };
  }, [ideas]);

  return (
    <div className="space-y-6" data-testid={selectors.ideas.page}>
      {/* Welcome hint for first-time users (Phase 29 Iteration 5) */}
      <WelcomeHint data-testid={selectors.ideas.welcomeHint} />

      {/* Header actions */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2">
          <SearchBar
            placeholder="Search ideas..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            data-testid={selectors.ideas.search}
          />
          <div className="relative">
            <Button
              variant="outline"
              size="sm"
              aria-label="Filter ideas"
              data-testid={selectors.ideas.filter}
              onClick={() => setShowFilters(!showFilters)}
              className={statusFilter ? "border-cyan-500/50" : ""}
            >
              <Filter className="h-4 w-4" />
            </Button>
            {showFilters && (
              <div className="absolute left-0 top-full z-10 mt-2 w-56 rounded-lg border border-white/10 bg-slate-800 p-3 shadow-xl">
                <div className="mb-2 flex items-center justify-between">
                  <span className="text-sm font-medium text-slate-200">Filters</span>
                  {statusFilter && (
                    <button
                      onClick={() => setStatusFilter("")}
                      className="text-xs text-slate-400 hover:text-slate-200"
                    >
                      Clear
                    </button>
                  )}
                </div>
                <div className="space-y-1">
                  <label className="text-xs text-slate-400">Status</label>
                  <div className="relative">
                    <select
                      value={statusFilter}
                      onChange={(e) => setStatusFilter(e.target.value as IdeaStatus | "")}
                      className="w-full appearance-none rounded-md border border-white/10 bg-slate-700 px-3 py-1.5 text-sm text-slate-200 focus:border-cyan-500 focus:outline-none"
                    >
                      <option value="">All statuses</option>
                      {STATUS_OPTIONS.map((status) => (
                        <option key={status} value={status}>
                          {formatIdeaStatus(status)}
                        </option>
                      ))}
                    </select>
                    <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                  </div>
                </div>
              </div>
            )}
          </div>
          {/* Status legend helps new users understand visual coding (Phase 29 Iteration 5) */}
          <StatusLegend
            items={IDEA_STATUS_LEGEND_ITEMS}
            title="Status Guide"
            data-testid={selectors.ideas.statusLegend}
          />
        </div>
        <div className="flex items-center gap-4">
          {/* Summary stats - helps returning users understand ecosystem state */}
          {ideas && ideas.length > 0 && (
            <div className="flex items-center gap-3 text-sm text-slate-400" data-testid={selectors.ideas.summaryStats}>
              <span>{stats.total} idea{stats.total !== 1 ? 's' : ''}</span>
              {stats.ready > 0 && (
                <span className="text-cyan-400" data-testid={selectors.ideas.readyCount}>
                  {stats.ready} ready to queue
                </span>
              )}
            </div>
          )}
          <Button
            data-testid={selectors.ideas.createButton}
            onClick={() => setShowCreate(true)}
          >
            <Plus className="mr-2 h-4 w-4" />
            New Idea
          </Button>
        </div>
      </div>

      {searchTerm && (
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <span>
            Showing results for <span className="text-slate-200">"{searchTerm}"</span>
          </span>
          <button onClick={() => setSearchTerm("")} className="text-slate-400 hover:text-slate-200">
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      {/* Ideas list */}
      <div className="space-y-4">
        {isLoading && (
          <Card padding="lg" centered>
            <p className="text-slate-400">Loading ideas...</p>
          </Card>
        )}

        {/* Error state - clearly different from empty state */}
        {error && (
          <ErrorState
            error={error}
            title="Unable to load ideas"
            onRetry={() => refetch()}
          />
        )}

        {/* Empty state - only shown when we successfully loaded but have no data */}
        {/* Experience Architecture (Phase 29): Provides actionable guidance including CLI usage */}
        {ideas && ideas.length === 0 && (
          <Card padding="lg" centered data-testid={selectors.ideas.empty}>
            <Lightbulb className="mx-auto h-12 w-12 text-slate-600" />
            <h3 className="mt-4 text-lg font-medium text-slate-300">No ideas yet</h3>
            <p className="mt-2 text-sm text-slate-400 max-w-md">
              Ideas are the starting point for new scenarios. Add ideas to your backlog to track what you want to build.
            </p>
            <div className="mt-6 space-y-3">
              <Button
                className="w-full sm:w-auto"
                data-testid={selectors.ideas.createFirstButton}
                onClick={() => setShowCreate(true)}
              >
                <Plus className="mr-2 h-4 w-4" />
                Create First Idea
              </Button>
              {/* CLI hint - helps users who may not know about alternative methods */}
              <div className="flex items-center gap-2 text-xs text-slate-500" data-testid={selectors.ideas.cliHint}>
                <Terminal className="h-3.5 w-3.5" />
                <span>Or use CLI: <code className="rounded bg-slate-800 px-1.5 py-0.5 font-mono text-slate-400">swarm-manager idea create</code></span>
              </div>
            </div>
          </Card>
        )}

        {/* Continue Working section - surfaces recently updated non-completed ideas */}
        {continueWorkingItems.length > 0 && (
          <div className="space-y-3" data-testid={selectors.ideas.continueSection}>
            <div className="flex items-center gap-2 text-sm font-medium text-slate-300">
              <Clock className="h-4 w-4 text-cyan-400" />
              <span>Continue Working</span>
            </div>
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3" data-testid={selectors.ideas.continueList}>
              {continueWorkingItems.map((idea) => (
                <Link
                  key={`continue-${idea.name}`}
                  to={`/ideas/${idea.name}`}
                  className="group flex items-center gap-3 rounded-lg border border-cyan-500/30 bg-cyan-500/5 p-3 transition hover:border-cyan-500/50 hover:bg-cyan-500/10"
                >
                  <span
                    className={`inline-block h-2 w-2 rounded-full ${IDEA_STATUS_COLORS[idea.status] ?? "bg-slate-500"}`}
                  />
                  <div className="flex-1 min-w-0">
                    <h4 className="truncate font-medium text-slate-100">{idea.title}</h4>
                    <p className="text-xs text-slate-400">
                      {formatIdeaStatus(idea.status)} · {formatRelativeTime(idea.updated)}
                    </p>
                  </div>
                  <ArrowRight className="h-4 w-4 text-cyan-400 opacity-0 transition group-hover:opacity-100" />
                </Link>
              ))}
            </div>
          </div>
        )}

        {/* All Ideas grid */}
        {ideas && ideas.length > 0 && filteredIdeas.length === 0 && (
          <Card padding="lg" centered data-testid={selectors.ideas.noResults}>
            <Lightbulb className="mx-auto h-12 w-12 text-slate-600" />
            <h3 className="mt-4 text-lg font-medium text-slate-300">No matching ideas</h3>
            <p className="mt-2 text-sm text-slate-400">
              Try adjusting your search or filter criteria.
            </p>
            <Button
              variant="outline"
              size="sm"
              className="mt-4"
              onClick={() => {
                setSearchTerm("");
                setStatusFilter("");
              }}
            >
              Clear filters
            </Button>
          </Card>
        )}

        {filteredIdeas.length > 0 && (
          <div className="space-y-3">
            {continueWorkingItems.length > 0 && (
              <div className="text-sm font-medium text-slate-400">All Ideas</div>
            )}
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3" data-testid={selectors.ideas.grid}>
              {filteredIdeas.map((idea) => (
                <Link
                  key={idea.name}
                  to={`/ideas/${idea.name}`}
                  className="group rounded-xl border border-white/10 bg-slate-800/30 p-4 transition hover:border-cyan-500/50 hover:bg-slate-800/50"
                  data-testid={selectors.ideas.cardByName({ name: idea.name })}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-2">
                      <span
                        className={`inline-block h-2 w-2 rounded-full ${IDEA_STATUS_COLORS[idea.status] ?? "bg-slate-500"}`}
                      />
                      <span className="text-xs uppercase tracking-wider text-slate-500">
                        {formatIdeaStatus(idea.status)}
                      </span>
                    </div>
                    <span className="rounded-full bg-slate-700 px-2 py-0.5 text-xs text-slate-300">
                      P{idea.priority}
                    </span>
                  </div>
                  <h3 className="mt-3 font-medium text-slate-100">{idea.title}</h3>
                  <p className="mt-1 line-clamp-2 text-sm text-slate-400">{idea.description}</p>
                  <TagList
                    tags={idea.tags}
                    maxTags={displayLimitsConfig.ideaCardMaxTags}
                    className="mt-3"
                  />
                  <div className="mt-4 flex items-center justify-between text-xs text-slate-500">
                    <span title={new Date(idea.updated).toLocaleString()}>{formatRelativeTime(idea.updated)}</span>
                    <ArrowRight className="h-4 w-4 opacity-0 transition group-hover:opacity-100" />
                  </div>
                </Link>
              ))}
            </div>
          </div>
        )}
      </div>

      <IdeaFormDialog
        isOpen={showCreate}
        mode="create"
        isSubmitting={createMutation.isPending}
        submitError={createError}
        onClose={() => {
          setShowCreate(false);
          createMutation.reset();
        }}
        onSubmit={(values) => createMutation.mutate(values)}
      />
    </div>
  );
}
