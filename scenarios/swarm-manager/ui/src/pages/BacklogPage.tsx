/**
 * Backlog Page
 *
 * Displays backlog items with search, filtering, and CRUD operations.
 *
 * Responsibility: Presentation only - rendering backlog items and handling user interactions.
 * Data fetching is delegated to the backlog store, which uses the backlog service seam.
 * Domain logic (types, status formatting) is imported from the types module.
 *
 * Error Handling:
 * - Clearly distinguishes between "no items" (empty state) and "error loading" (error state)
 * - Shows user-friendly error messages based on error type
 * - Provides retry functionality for recoverable errors
 *
 * Experience Architecture (Phase 29):
 * - "Continue Working" section surfaces recently updated non-completed items
 * - Summary stats show total items and count ready for processing
 * - Reduces cognitive load for returning users by highlighting active work
 */

import { useMemo, useState, useEffect } from "react";
import { useMutation } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Plus, Filter, Lightbulb, ArrowRight, Clock, Terminal, X, Search, Wrench, Play } from "lucide-react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ErrorState } from "../components/ui/error-state";
import { FloatingActionButton } from "../components/ui/floating-action-button";
import { ResponsiveList, ResponsiveListItem } from "../components/ui/responsive-list";
import { SearchBar } from "../components/ui/search-bar";
import { Select } from "../components/ui/select";
import { StatusLegend } from "../components/ui/status-legend";
import { BACKLOG_STATUS_LEGEND_ITEMS } from "../components/ui/status-legend.constants";
import { TagList } from "../components/ui/tag-list";
import { WelcomeHint } from "../components/ui/welcome-hint";
import { Tabs, TabsList, TabsTrigger } from "../components/ui/tabs";
import { formatRelativeTime } from "../lib";
import { backlogService } from "../services";
import { selectors } from "../consts/selectors";
import {
  BACKLOG_KIND_LABELS,
  BACKLOG_STATUS_COLORS,
  formatBacklogStatus,
  type BacklogKind,
  type BacklogStatus,
} from "../types";
import { displayLimitsConfig } from "../config";
import { BacklogFormDialog } from "../components/backlog/backlog-form-dialog";
import { useBacklogStore } from "../stores";

/** Statuses that indicate an item is "completed" and shouldn't appear in Continue Working */
const COMPLETED_STATUSES: BacklogStatus[] = ["completed", "archived"];

const STATUS_OPTIONS: BacklogStatus[] = [
  "backlog",
  "researching",
  "ready",
  "queued",
  "in_progress",
  "completed",
  "archived",
];

const BACKLOG_KIND_TABS: Array<{
  kind: BacklogKind;
  label: string;
  icon: typeof Lightbulb;
  emptyTitle: string;
  emptyDescription: string;
  ctaLabel: string;
}> = [
  {
    kind: "research",
    label: "Research",
    icon: Search,
    emptyTitle: "No research yet",
    emptyDescription: "Capture research questions, investigations, and discoveries before moving to fixes or execution.",
    ctaLabel: "New Research",
  },
  {
    kind: "idea",
    label: "Idea",
    icon: Lightbulb,
    emptyTitle: "No ideas yet",
    emptyDescription: "Ideas are the starting point for new scenarios. Track what you want to build next.",
    ctaLabel: "New Idea",
  },
  {
    kind: "fix",
    label: "Fix",
    icon: Wrench,
    emptyTitle: "No fixes yet",
    emptyDescription: "Capture fixes the swarm should apply to existing scenarios and tooling.",
    ctaLabel: "New Fix",
  },
  {
    kind: "execute",
    label: "Execute",
    icon: Play,
    emptyTitle: "No execution tasks yet",
    emptyDescription: "Track tasks that should be executed by the swarm with focused instructions.",
    ctaLabel: "New Execution",
  },
];

export function BacklogPage() {
  const [activeKind, setActiveKind] = useState<BacklogKind>("idea");
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState<BacklogStatus | "">("");
  const [showFilters, setShowFilters] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const items = useBacklogStore((state) => state.items);
  const status = useBacklogStore((state) => state.status);
  const error = useBacklogStore((state) => state.error);
  const fetchBacklog = useBacklogStore((state) => state.fetchBacklog);
  const hasLoaded = status !== "idle";

  useEffect(() => {
    void fetchBacklog();
  }, [fetchBacklog]);

  const createMutation = useMutation({
    mutationFn: backlogService.create,
    onSuccess: () => {
      void fetchBacklog({ force: true });
      setShowCreate(false);
    },
  });

  const createError = createMutation.isError ? "Failed to create backlog item. Please try again." : null;

  const kindItems = useMemo(
    () => items.filter((item) => item.kind === activeKind),
    [items, activeKind]
  );

  const filteredItems = useMemo(() => {
    let result = kindItems;
    if (searchTerm.trim()) {
      const term = searchTerm.toLowerCase();
      result = result.filter(
        (item) =>
          item.title.toLowerCase().includes(term) ||
          item.description.toLowerCase().includes(term) ||
          item.name.toLowerCase().includes(term)
      );
    }
    if (statusFilter) {
      result = result.filter((item) => item.status === statusFilter);
    }
    return result;
  }, [kindItems, searchTerm, statusFilter]);

  const continueWorkingItems = useMemo(() => {
    if (!filteredItems || filteredItems.length === 0) return [];
    return filteredItems
      .filter((item) => !COMPLETED_STATUSES.includes(item.status))
      .sort((a, b) => new Date(b.updated).getTime() - new Date(a.updated).getTime())
      .slice(0, 3);
  }, [filteredItems]);

  const stats = useMemo(() => {
    return {
      total: kindItems.length,
      ready: kindItems.filter((item) => item.status === "ready").length,
    };
  }, [kindItems]);

  const activeTab = BACKLOG_KIND_TABS.find((tab) => tab.kind === activeKind) ?? BACKLOG_KIND_TABS[0];
  if (!activeTab) {
    return (
      <div className="space-y-6" data-testid={selectors.backlog.page}>
        <Card className="border border-dashed border-slate-700/60 bg-slate-900/40 p-6 text-slate-200">
          No backlog categories configured.
        </Card>
      </div>
    );
  }
  const EmptyIcon = activeTab.icon;

  return (
    <div className="space-y-6" data-testid={selectors.backlog.page}>
      <div className="flex flex-col gap-4">
        <div className="order-2 md:order-1">
          <WelcomeHint data-testid={selectors.backlog.welcomeHint} />
        </div>

        <div className="order-1 md:order-2 flex flex-col gap-4">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <Tabs value={activeKind} onValueChange={(value) => setActiveKind(value as BacklogKind)} className="w-full">
              <div className="-mx-6 md:mx-0">
                <TabsList
                  className="w-full flex-nowrap justify-start gap-1 overflow-x-auto no-scrollbar rounded-none bg-transparent p-0 md:w-auto md:flex-wrap md:overflow-visible md:rounded-md md:bg-slate-800/50 md:p-1"
                  data-testid={selectors.backlog.kindTabs}
                >
                  {BACKLOG_KIND_TABS.map((tab) => {
                    const Icon = tab.icon;
                    return (
                      <TabsTrigger key={tab.kind} value={tab.kind} className="gap-2">
                        <Icon className="h-4 w-4" />
                        {tab.label}
                      </TabsTrigger>
                    );
                  })}
                </TabsList>
              </div>
            </Tabs>
            <Button
              data-testid={selectors.backlog.createButton}
              onClick={() => setShowCreate(true)}
              className="hidden md:inline-flex"
            >
              <Plus className="mr-2 h-4 w-4" />
              {activeTab.ctaLabel}
            </Button>
          </div>

          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex flex-wrap items-center gap-2">
              <SearchBar
                placeholder={`Search ${BACKLOG_KIND_LABELS[activeKind].toLowerCase()} backlog...`}
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                data-testid={selectors.backlog.search}
                widthClass="w-full sm:w-80"
              />
              <div className="relative">
                <Button
                  variant="outline"
                  size="sm"
                  aria-label="Filter backlog"
                  data-testid={selectors.backlog.filter}
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
                      <label htmlFor="backlog-status-filter" className="text-xs text-slate-400">
                        Status
                      </label>
                      <Select
                        id="backlog-status-filter"
                        value={statusFilter}
                        onChange={(e) => setStatusFilter(e.target.value as BacklogStatus | "")}
                        variant="filter"
                        withChevron
                      >
                        <option value="">All statuses</option>
                        {STATUS_OPTIONS.map((option) => (
                          <option key={option} value={option}>
                            {formatBacklogStatus(option)}
                          </option>
                        ))}
                      </Select>
                    </div>
                  </div>
                )}
              </div>
              <StatusLegend
                items={BACKLOG_STATUS_LEGEND_ITEMS}
                title="Status Guide"
                compact
                data-testid={selectors.backlog.statusLegend}
              />
            </div>

            {kindItems.length > 0 && (
              <div className="flex items-center gap-3 text-sm text-slate-400" data-testid={selectors.backlog.summaryStats}>
                <span>{stats.total} item{stats.total !== 1 ? "s" : ""}</span>
                {stats.ready > 0 && (
                  <span className="text-cyan-400" data-testid={selectors.backlog.readyCount}>
                    {stats.ready} ready to queue
                  </span>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {searchTerm && (
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <span>
            Showing results for <span className="text-slate-200">"{searchTerm}"</span>
          </span>
          <button
            onClick={() => setSearchTerm("")}
            className="text-slate-400 hover:text-slate-200"
            aria-label="Clear search"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      <div className="space-y-4">
        {(status === "loading" || !hasLoaded) && items.length === 0 && (
          <Card padding="lg" centered>
            <p className="text-slate-400">Loading backlog...</p>
          </Card>
        )}

        {error && items.length === 0 && hasLoaded && (
          <ErrorState
            error={error}
            title="Unable to load backlog"
            onRetry={() => fetchBacklog({ force: true })}
          />
        )}

        {!error && kindItems.length === 0 && hasLoaded && (
          <Card padding="lg" centered data-testid={selectors.backlog.empty}>
            <EmptyIcon className="mx-auto h-12 w-12 text-slate-600" />
            <h3 className="mt-4 text-lg font-medium text-slate-300">{activeTab.emptyTitle}</h3>
            <p className="mt-2 text-sm text-slate-400 max-w-md">
              {activeTab.emptyDescription}
            </p>
            <div className="mt-6 space-y-3">
              <Button
                className="w-full sm:w-auto"
                data-testid={selectors.backlog.createFirstButton}
                onClick={() => setShowCreate(true)}
              >
                <Plus className="mr-2 h-4 w-4" />
                {activeTab.ctaLabel}
              </Button>
              <div className="flex items-center gap-2 text-xs text-slate-400" data-testid={selectors.backlog.cliHint}>
                <Terminal className="h-3.5 w-3.5" />
                <span>
                  Or use CLI: <code className="rounded bg-slate-800 px-1.5 py-0.5 font-mono text-slate-400">swarm-manager backlog create</code>
                </span>
              </div>
            </div>
          </Card>
        )}

        {continueWorkingItems.length > 0 && (
          <div className="space-y-3" data-testid={selectors.backlog.continueSection}>
            <div className="flex items-center gap-2 text-sm font-medium text-slate-300">
              <Clock className="h-4 w-4 text-cyan-400" />
              <span>Continue Working</span>
            </div>
            <ResponsiveList data-testid={selectors.backlog.continueList}>
              {continueWorkingItems.map((item) => (
                <ResponsiveListItem
                  as={Link}
                  key={`continue-${item.kind}-${item.name}`}
                  to={`/backlog/${item.kind}/${item.name}`}
                  interactive
                  className="group flex items-center gap-3 md:border-cyan-500/30 md:bg-cyan-500/5 md:hover:border-cyan-500/50 md:hover:bg-cyan-500/10"
                >
                  <span
                    className={`inline-block h-2 w-2 rounded-full ${BACKLOG_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
                  />
                  <div className="flex-1 min-w-0">
                    <h4 className="truncate font-medium text-slate-100">{item.title}</h4>
                    <p className="text-xs text-slate-400">
                      {formatBacklogStatus(item.status)} · {formatRelativeTime(item.updated)}
                    </p>
                  </div>
                  <ArrowRight className="h-4 w-4 text-cyan-400 opacity-0 transition group-hover:opacity-100" />
                </ResponsiveListItem>
              ))}
            </ResponsiveList>
          </div>
        )}

        {kindItems.length > 0 && filteredItems.length === 0 && (
          <Card padding="lg" centered data-testid={selectors.backlog.noResults}>
            <Search className="mx-auto h-12 w-12 text-slate-600" />
            <h3 className="mt-4 text-lg font-medium text-slate-300">No matching items</h3>
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

        {filteredItems.length > 0 && (
          <div className="space-y-3">
            {continueWorkingItems.length > 0 && (
              <div className="text-sm font-medium text-slate-400">All {activeTab.label} Items</div>
            )}
            <ResponsiveList data-testid={selectors.backlog.grid}>
              {filteredItems.map((item) => (
                <ResponsiveListItem
                  as={Link}
                  key={`${item.kind}-${item.name}`}
                  to={`/backlog/${item.kind}/${item.name}`}
                  interactive
                  className="group block"
                  data-testid={selectors.backlog.cardByName({ kind: item.kind, name: item.name })}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-2">
                      <span
                        className={`inline-block h-2 w-2 rounded-full ${BACKLOG_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
                      />
                      <span className="text-xs uppercase tracking-wider text-slate-400">
                        {formatBacklogStatus(item.status)}
                      </span>
                    </div>
                    <span className="rounded-full bg-slate-700 px-2 py-0.5 text-xs text-slate-300">
                      P{item.priority}
                    </span>
                  </div>
                  <h3 className="mt-3 font-medium text-slate-100">{item.title}</h3>
                  <p className="mt-1 line-clamp-2 text-sm text-slate-400">{item.description}</p>
                  <TagList
                    tags={item.tags}
                    maxTags={displayLimitsConfig.backlogCardMaxTags}
                    className="mt-3"
                  />
                  <div className="mt-4 flex items-center justify-between text-xs text-slate-400">
                    <span title={new Date(item.updated).toLocaleString()}>{formatRelativeTime(item.updated)}</span>
                    <ArrowRight className="h-4 w-4 opacity-0 transition group-hover:opacity-100" />
                  </div>
                </ResponsiveListItem>
              ))}
            </ResponsiveList>
          </div>
        )}
      </div>

      <FloatingActionButton
        icon={<Plus className="h-5 w-5" />}
        label={activeTab.ctaLabel}
        onClick={() => setShowCreate(true)}
        disabled={createMutation.isPending}
        className="md:hidden"
      />

      <BacklogFormDialog
        isOpen={showCreate}
        mode="create"
        defaultKind={activeKind}
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
