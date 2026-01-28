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

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Plus, Filter, Lightbulb, ArrowRight, Clock, Terminal } from "lucide-react";
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
import { IDEA_STATUS_COLORS, formatIdeaStatus } from "../types";
import { displayLimitsConfig } from "../config";

/** Statuses that indicate an idea is "completed" and shouldn't appear in Continue Working */
const COMPLETED_STATUSES = ["completed", "archived"] as const;

export function IdeasPage() {
  const { data: ideas, isLoading, error, refetch } = useQuery({
    queryKey: ["ideas"],
    queryFn: () => ideasService.list(),
    ...defaultQueryOptions,
  });

  // Compute "Continue Working" items - recently updated non-completed ideas
  // Sorted by most recently updated first, limited to 3 items
  const continueWorkingItems = useMemo(() => {
    if (!ideas || ideas.length === 0) return [];
    return ideas
      .filter((idea) => !COMPLETED_STATUSES.includes(idea.status as typeof COMPLETED_STATUSES[number]))
      .sort((a, b) => new Date(b.updated).getTime() - new Date(a.updated).getTime())
      .slice(0, 3);
  }, [ideas]);

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
            data-testid={selectors.ideas.search}
          />
          <Button
            variant="outline"
            size="sm"
            aria-label="Filter ideas"
            disabled
            title="Filter functionality coming soon"
            data-testid={selectors.ideas.filter}
          >
            <Filter className="h-4 w-4" />
          </Button>
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
            disabled
            title="Create idea via CLI: swarm-manager idea create"
            data-testid={selectors.ideas.createButton}
          >
            <Plus className="mr-2 h-4 w-4" />
            New Idea
          </Button>
        </div>
      </div>

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
                disabled
                title="Create idea via CLI: swarm-manager idea create"
                data-testid={selectors.ideas.createFirstButton}
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
        {ideas && ideas.length > 0 && (
          <div className="space-y-3">
            {continueWorkingItems.length > 0 && (
              <div className="text-sm font-medium text-slate-400">All Ideas</div>
            )}
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3" data-testid={selectors.ideas.grid}>
              {ideas.map((idea) => (
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
    </div>
  );
}
