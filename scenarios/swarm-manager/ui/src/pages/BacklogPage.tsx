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
 * Experience Architecture:
 * - Sort control lets users reorder by priority, recency, status, or title
 * - Summary stats show total items and count ready for processing
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { useUrlState } from "../hooks/use-url-state";
import { Plus, Filter, Lightbulb, ArrowUpDown, CheckSquare, Terminal, X, Search, Wrench, Play, LayoutGrid, MessageSquareText } from "lucide-react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ErrorState } from "../components/ui/error-state";
import { FloatingActionButton } from "../components/ui/floating-action-button";
import { InlineLoadingIndicator, PageLoadingState } from "../components/ui/loading-states";
import { ResponsiveList, ResponsiveListItem } from "../components/ui/responsive-list";
import { SearchBar } from "../components/ui/search-bar";
import { Select } from "../components/ui/select";
import { StatusLegend } from "../components/ui/status-legend";
import { BACKLOG_STATUS_LEGEND_ITEMS } from "../components/ui/status-legend.constants";
import { WelcomeHint } from "../components/ui/welcome-hint";
import { Tabs, TabsList, TabsTrigger } from "../components/ui/tabs";
import { getItemActions, hasBlockingDeps, isBacklogQueueable } from "../lib";
import { dependencyAwareSort } from "../lib/dependency-sort";
import { buildReadinessData } from "../lib/maturity";
import type { ReadinessIndicatorData } from "../lib/maturity";
import { backlogService } from "../services";
import { selectors } from "../consts/selectors";
import {
  BACKLOG_KIND_LABELS,
  formatBacklogStatus,
  type BacklogKind,
  type BacklogStatus,
} from "../types";
import { BacklogFormDialog } from "../components/backlog/backlog-form-dialog";
import { CaptureCard } from "../components/backlog/capture-card";
import { FeedbackHubModal } from "../components/backlog/feedback-hub-modal";
import { QuickCaptureInput } from "../components/backlog/quick-capture-input";
import { RunBacklogModal } from "../components/backlog/run-backlog-modal";
import type { RunBacklogTarget } from "../components/backlog/run-backlog-modal";
import type { StepperCompletionResult } from "../components/backlog/inline-question-stepper";
import { buildFeed, countActionableItems, getAttentionReasons, type AttentionReason, type FeedbackItem, type FeedItem, type MaturityItem } from "../lib/feed";
import { BacklogCard } from "../components/backlog/backlog-card";
import { useAgentActivitiesStore, useBacklogStore, useCaptureStore } from "../stores";
import type { BacklogFormValues, BacklogItem, PendingQuestion } from "../types";

type SortField = "priority" | "updated" | "status" | "title";

const SORT_OPTIONS: Array<{ field: SortField; label: string }> = [
  { field: "priority", label: "Priority" },
  { field: "updated", label: "Recently Updated" },
  { field: "status", label: "Status" },
  { field: "title", label: "Title" },
];

const STATUS_SORT_ORDER: Record<BacklogStatus, number> = {
  backlog: 0,
  researching: 1,
  ready: 2,
  queued: 3,
  in_progress: 4,
  failed: 5,
  completed: 6,
  archived: 7,
};

const STATUS_OPTIONS: BacklogStatus[] = [
  "backlog",
  "researching",
  "ready",
  "queued",
  "in_progress",
  "failed",
  "completed",
  "archived",
];

type TabKind = BacklogKind | "all";

const VALID_KINDS = new Set<string>(["all", "idea", "fix", "execute", "chore", "research"]);
const isTabKind = (v: string): v is TabKind => VALID_KINDS.has(v);

const VALID_SORT_FIELDS = new Set<string>(["priority", "updated", "status", "title"]);
const isSortField = (v: string): v is SortField => VALID_SORT_FIELDS.has(v);

const VALID_STATUSES = new Set<string>(["", "backlog", "researching", "ready", "queued", "in_progress", "failed", "completed", "archived"]);
const isBacklogStatusOrEmpty = (v: string): v is BacklogStatus | "" => VALID_STATUSES.has(v);

const FINISHED_STATUSES: BacklogStatus[] = ["archived"];

const SEARCH_DEBOUNCE_MS = 300;
const ACTIVE_BACKLOG_REFRESH_MS = 6000;

const BACKLOG_KIND_TABS: Array<{
  kind: TabKind;
  label: string;
  icon: typeof Lightbulb;
  emptyTitle: string;
  emptyDescription: string;
  ctaLabel: string;
}> = [
  {
    kind: "all",
    label: "All",
    icon: LayoutGrid,
    emptyTitle: "Capture your first thought",
    emptyDescription: "Type what's on your mind above — the system will classify and organize it for you.",
    ctaLabel: "New Item",
  },
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
  {
    kind: "chore",
    label: "Chore",
    icon: Wrench,
    emptyTitle: "No chores yet",
    emptyDescription: "Track maintenance, cleanup, dependency updates, and infrastructure work.",
    ctaLabel: "New Chore",
  },
];

export function BacklogPage() {
  // URL-synced state (persists across navigation)
  const [activeKind, setActiveKind] = useUrlState<TabKind>("kind", "all", { validate: isTabKind });
  const [statusFilter, setStatusFilter] = useUrlState<BacklogStatus | "">("status", "", { validate: isBacklogStatusOrEmpty });
  const [sortField, setSortField] = useUrlState<SortField>("sort", "priority", { validate: isSortField });
  const [showFinishedParam, setShowFinishedParam] = useUrlState<"" | "1">("finished", "");
  const showFinished = showFinishedParam === "1";
  const setShowFinished = useCallback((v: boolean) => setShowFinishedParam(v ? "1" : ""), [setShowFinishedParam]);

  // Search: local state for instant UI + debounced URL sync
  const [searchParams, setSearchParams] = useSearchParams();
  const [searchTerm, setSearchTerm] = useState(() => searchParams.get("q") ?? "");
  const searchTimerRef = useRef<ReturnType<typeof setTimeout>>();
  useEffect(() => {
    clearTimeout(searchTimerRef.current);
    searchTimerRef.current = setTimeout(() => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        if (searchTerm) {
          next.set("q", searchTerm);
        } else {
          next.delete("q");
        }
        return next;
      }, { replace: true });
    }, SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(searchTimerRef.current);
  }, [searchTerm, setSearchParams]);

  // Sync URL → local state when user navigates back (browser back button)
  const urlQ = searchParams.get("q") ?? "";
  useEffect(() => {
    setSearchTerm((prev) => (prev !== urlQ ? urlQ : prev));
  }, [urlQ]);

  const location = useLocation();
  const navigate = useNavigate();

  // Ephemeral UI state (not persisted)
  const [showSort, setShowSort] = useState(false);
  const [showFilters, setShowFilters] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [createPrefill, setCreatePrefill] = useState<BacklogFormValues | undefined>();
  const [showFeedbackHub, setShowFeedbackHub] = useState(false);
  const [feedbackHubInitialTab, setFeedbackHubInitialTab] = useState<"review" | "export" | "import">("review");
  const [feedbackHubSelectedNames, setFeedbackHubSelectedNames] = useState<string[] | undefined>();
  const [batchMode, setBatchMode] = useState(false);
  const [runModalTarget, setRunModalTarget] = useState<RunBacklogTarget | null>(null);
  const [runModalTargets, setRunModalTargets] = useState<RunBacklogTarget[] | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<string[]>([]);
  const [completedSteppers, setCompletedSteppers] = useState<Set<string>>(() => new Set());
  /** Items whose stepper just completed — kept visible with a transition message before feed refresh. */
  const [transitionItems, setTransitionItems] = useState<Map<string, StepperCompletionResult>>(() => new Map());
  const [searchFocused, setSearchFocused] = useState(false);
  const queryClient = useQueryClient();
  const agentActivities = useAgentActivitiesStore((state) => state.activities);
  const refreshActivities = useAgentActivitiesStore((state) => state.refreshActivities);
  const activeRunKeys = useMemo(() => {
    const keys = new Set<string>();
    for (const activity of agentActivities) {
      if (activity.ownerType === "backlog" && activity.ownerKind && activity.ownerName) {
        keys.add(`${activity.ownerKind}/${activity.ownerName}`);
      }
    }
    return keys;
  }, [agentActivities]);
  /** Map from backlog key → human-readable running label (e.g. "Running workshop…"). */
  const activeRunLabels = useMemo(() => {
    const labels = new Map<string, string>();
    for (const activity of agentActivities) {
      if (activity.ownerType === "backlog" && activity.ownerKind && activity.ownerName) {
        const key = `${activity.ownerKind}/${activity.ownerName}`;
        switch (activity.purpose) {
          case "workshop": labels.set(key, "Running workshop…"); break;
          case "finalize": labels.set(key, "Running finalize…"); break;
          case "research": labels.set(key, "Running research…"); break;
          case "initialize": labels.set(key, "Initializing workshop…"); break;
          case "process": labels.set(key, "Processing…"); break;
          default: labels.set(key, "Agent running…"); break;
        }
      }
    }
    return labels;
  }, [agentActivities]);
  const items = useBacklogStore((state) => state.items);
  const status = useBacklogStore((state) => state.status);
  const error = useBacklogStore((state) => state.error);
  const isRefreshing = useBacklogStore((state) => state.isRefreshing);
  const fetchBacklog = useBacklogStore((state) => state.fetchBacklog);
  const hasLoaded = status !== "idle";

  const captures = useCaptureStore((s) => s.captures);
  const fetchCaptures = useCaptureStore((s) => s.fetchCaptures);

  useEffect(() => {
    void fetchBacklog();
    void fetchCaptures();
  }, [fetchBacklog, fetchCaptures]);

  useEffect(() => {
    void refreshActivities(true);
    const interval = window.setInterval(() => void refreshActivities(true), 5000);
    return () => window.clearInterval(interval);
  }, [refreshActivities]);

  // Polling: refresh captures every 3s when any are classifying (max 60s then stop)
  useEffect(() => {
    const classifyingCaptures = captures.filter((c) => c.status === "classifying");
    if (classifyingCaptures.length === 0) return;

    // Stop polling if all classifying captures are older than 60s (let backend auto-fail at 2 min)
    const allStale = classifyingCaptures.every((c) => {
      const age = Date.now() - new Date(c.created).getTime();
      return age > 60_000;
    });
    if (allStale) return;

    const interval = setInterval(() => void fetchCaptures({ force: true }), 3000);
    return () => clearInterval(interval);
  }, [captures, fetchCaptures]);

  // Single combined query replaces 3 separate summary RPCs.
  const summaryQuery = useQuery({
    queryKey: ["backlog-summary"],
    queryFn: () => backlogService.getBacklogSummary(),
    staleTime: 60_000,
    refetchInterval: activeRunKeys.size > 0 ? ACTIVE_BACKLOG_REFRESH_MS : false,
  });
  const feedbackSummary = summaryQuery.data?.feedback;

  const readinessMap = useMemo(() => {
    const map = new Map<string, ReadinessIndicatorData>();
    if (!summaryQuery.data?.maturity?.items) return map;
    for (const item of summaryQuery.data.maturity.items) {
      map.set(`${item.kind}/${item.name}`, buildReadinessData(item));
    }
    return map;
  }, [summaryQuery.data?.maturity]);

  const pendingQuestionsMap = useMemo(() => {
    const map = new Map<string, PendingQuestion[]>();
    if (!summaryQuery.data?.pending_questions?.items) return map;
    for (const pqi of summaryQuery.data.pending_questions.items) {
      map.set(`${pqi.kind}/${pqi.name}`, pqi.questions);
    }
    return map;
  }, [summaryQuery.data?.pending_questions]);

  const feedbackItems = useMemo<FeedbackItem[]>(
    () => (feedbackSummary?.items ?? []).map((item) => ({
      kind: item.kind,
      name: item.name,
      pendingDecisions: item.pending_decisions ?? 0,
    })),
    [feedbackSummary],
  );

  const maturityItems = useMemo<MaturityItem[]>(
    () => (summaryQuery.data?.maturity?.items ?? []).map((item) => ({
      kind: item.kind,
      name: item.name,
      ready: item.ready ?? false,
      pendingItems: item.pending_items ?? 0,
    })),
    [summaryQuery.data?.maturity],
  );

  // Build unified feed for the "All" tab
  const feedItems = useMemo(() => {
    if (activeKind !== "all") return [];
    return buildFeed(captures, items, feedbackItems, maturityItems, { showFinished });
  }, [activeKind, captures, items, feedbackItems, maturityItems, showFinished]);

  // Attention reasons for all items (used when not in feed mode)
  const attentionReasonsMap = useMemo(() => {
    const fbMap = new Map(feedbackItems.map((f) => [`${f.kind}/${f.name}`, f]));
    const matMap = new Map(maturityItems.map((m) => [`${m.kind}/${m.name}`, m]));
    const map = new Map<string, AttentionReason[]>();
    for (const item of items) {
      const reasons = getAttentionReasons(item, fbMap, matMap);
      if (reasons.length > 0) map.set(`${item.kind}/${item.name}`, reasons);
    }
    return map;
  }, [items, feedbackItems, maturityItems]);

  const actionableCount = useMemo(() => countActionableItems(feedItems), [feedItems]);

  const createMutation = useMutation({
    mutationFn: backlogService.create,
    onSuccess: () => {
      void fetchBacklog({ force: true });
      setShowCreate(false);
    },
  });

  const archiveMutation = useMutation({
    mutationFn: ({ kind, name: itemName, item }: { kind: BacklogKind; name: string; item: BacklogItem }) =>
      backlogService.update(kind, itemName, { ...item, status: "archived" }),
    onSuccess: () => {
      void fetchBacklog({ force: true });
    },
  });

  /** Trigger a workshop/finalize round directly from a card. */
  const [workshopError, setWorkshopError] = useState<string | null>(null);
  const workshopMutation = useMutation({
    mutationFn: ({
      kind,
      name: itemName,
      mode,
      prompt,
    }: {
      kind: BacklogKind;
      name: string;
      mode: "workshop" | "finalize";
      prompt: string;
    }) =>
      backlogService.research(kind, itemName, {
        mode,
        prompt,
      }),
    onSuccess: (result, _variables) => {
      setWorkshopError(null);
      if (result.runId) {
        void refreshActivities(true);
      }
      void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
    },
    onError: (error) => {
      const msg = error instanceof Error ? error.message : "Failed to start workshop round";
      setWorkshopError(msg);
      // Auto-clear error after 8 seconds
      setTimeout(() => setWorkshopError(null), 8000);
    },
  });
  const handleStepperCompleted = useCallback((itemKey: string, item: BacklogItem, result: StepperCompletionResult) => {
    setCompletedSteppers((prev) => {
      const next = new Set(prev);
      next.add(itemKey);
      return next;
    });
    if (result.autoAdvance?.triggered && result.autoAdvance.runId) {
      void refreshActivities(true);
    }
    setTransitionItems((prev) => {
      const next = new Map(prev);
      next.set(itemKey, result);
      return next;
    });
    setTimeout(() => {
      setTransitionItems((prev) => {
        const next = new Map(prev);
        next.delete(itemKey);
        return next;
      });
      void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
    }, 4000);
  }, [queryClient, refreshActivities]);

  const openFeedbackHub = (tab: "review" | "export" | "import", names?: string[]) => {
    setFeedbackHubInitialTab(tab);
    setFeedbackHubSelectedNames(names);
    setShowFeedbackHub(true);
  };

  const createError = createMutation.isError
    ? createMutation.error instanceof Error ? createMutation.error.message : "Failed to create backlog item. Please try again."
    : null;

  const kindItems = useMemo(
    () => activeKind === "all" ? items : items.filter((item) => item.kind === activeKind),
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
    } else if (!showFinished) {
      result = result.filter((item) => !FINISHED_STATUSES.includes(item.status));
    }
    return result;
  }, [kindItems, searchTerm, statusFilter, showFinished]);

  const sortedItems = useMemo(() => {
    const compareFn = (a: BacklogItem, b: BacklogItem): number => {
      switch (sortField) {
        case "priority":
          return a.priority !== b.priority
            ? a.priority - b.priority
            : new Date(b.updated).getTime() - new Date(a.updated).getTime();
        case "updated":
          return new Date(b.updated).getTime() - new Date(a.updated).getTime();
        case "status":
          return STATUS_SORT_ORDER[a.status] !== STATUS_SORT_ORDER[b.status]
            ? STATUS_SORT_ORDER[a.status] - STATUS_SORT_ORDER[b.status]
            : a.priority - b.priority;
        case "title":
          return a.title.localeCompare(b.title);
      }
    };
    // Dependency-aware sort: items with incomplete deps sort below their deps.
    // `items` (full unfiltered list) is passed so depth computation sees deps
    // that may be excluded by the current filter.
    return dependencyAwareSort(filteredItems, compareFn, items);
  }, [filteredItems, sortField, items]);

  const stats = useMemo(() => {
    const finishedCount = kindItems.filter((item) => FINISHED_STATUSES.includes(item.status)).length;
    return {
      total: kindItems.length,
      ready: kindItems.filter((item) => item.status === "ready").length,
      finishedCount,
    };
  }, [kindItems]);

  const activeTab = BACKLOG_KIND_TABS.find((tab) => tab.kind === activeKind) ?? BACKLOG_KIND_TABS[0];
  const queueableFilteredItems = useMemo(
    () => filteredItems.filter((item) => isBacklogQueueable(item) && !hasBlockingDeps(item, items)),
    [filteredItems, items]
  );
  const queueableFilteredKeySet = useMemo(
    () => new Set(queueableFilteredItems.map((item) => `${item.kind}/${item.name}`)),
    [queueableFilteredItems]
  );
  const selectedQueueableItems = useMemo(
    () => queueableFilteredItems.filter((item) => selectedKeys.includes(`${item.kind}/${item.name}`)),
    [queueableFilteredItems, selectedKeys]
  );
  const allQueueableSelected =
    queueableFilteredItems.length > 0 && selectedQueueableItems.length === queueableFilteredItems.length;
  const hasAnySelectedQueueable = selectedQueueableItems.length > 0;

  useEffect(() => {
    setSelectedKeys((prev) => {
      const next = prev.filter((key) => queueableFilteredKeySet.has(key));
      return next.length === prev.length ? prev : next;
    });
  }, [queueableFilteredKeySet]);

  const toggleItemSelection = (item: { kind: BacklogKind; name: string }) => {
    const key = `${item.kind}/${item.name}`;
    setSelectedKeys((prev) => (prev.includes(key) ? prev.filter((existing) => existing !== key) : [...prev, key]));
  };

  const toggleSelectAllQueueable = () => {
    if (allQueueableSelected) {
      setSelectedKeys([]);
      return;
    }
    setSelectedKeys(queueableFilteredItems.map((item) => `${item.kind}/${item.name}`));
  };

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
            <Tabs value={activeKind} onValueChange={(value) => setActiveKind(value as TabKind)} className="w-full">
              <div className="-mx-6 md:mx-0">
                <TabsList
                  className="w-full flex-nowrap justify-start gap-1 overflow-x-auto no-scrollbar rounded-none bg-transparent p-0 md:w-auto md:flex-wrap md:overflow-visible md:rounded-md md:bg-slate-800/50 md:p-1"
                  data-testid={selectors.backlog.kindTabs}
                >
                  {BACKLOG_KIND_TABS.map((tab) => {
                    const Icon = tab.icon;
                    const count = tab.kind === "all" ? actionableCount : 0;
                    return (
                      <TabsTrigger key={tab.kind} value={tab.kind} className="gap-2">
                        <Icon className="h-4 w-4" />
                        {tab.label}
                        {count > 0 && (
                          <span className="ml-1 rounded-full bg-cyan-500/20 px-1.5 py-0.5 text-[10px] font-medium text-cyan-400">
                            {count}
                          </span>
                        )}
                      </TabsTrigger>
                    );
                  })}
                </TabsList>
              </div>
            </Tabs>
            <div className="hidden md:flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => openFeedbackHub("review")}
                data-testid={selectors.backlog.feedbackHub.cta}
                className={feedbackSummary && feedbackSummary.total_items_affected > 0 ? "border-cyan-500/50 text-cyan-300" : ""}
              >
                <MessageSquareText className="mr-2 h-4 w-4" />
                Feedback & Export
                {feedbackSummary && feedbackSummary.total_items_affected > 0 && (
                  <span className="ml-2 rounded-full bg-cyan-500/20 px-2 py-0.5 text-xs text-cyan-300">
                    {feedbackSummary.total_items_affected}
                  </span>
                )}
              </Button>
              <Button
                data-testid={selectors.backlog.createButton}
                onClick={() => setShowCreate(true)}
              >
                <Plus className="mr-2 h-4 w-4" />
                {activeTab.ctaLabel}
              </Button>
            </div>
          </div>

          {/* Quick capture input — only on "All" tab, hidden when searching */}
          {activeKind === "all" && !searchTerm && !searchFocused && (
            <QuickCaptureInput
              onOpenForm={(draftText) => {
                if (draftText) {
                  setCreatePrefill({ description: draftText } as BacklogFormValues);
                }
                setShowCreate(true);
              }}
            />
          )}

          <div className="flex items-center gap-2">
              <SearchBar
                placeholder={activeKind === "all" ? `Search ${stats.total} item${stats.total !== 1 ? "s" : ""}...` : `Search ${BACKLOG_KIND_LABELS[activeKind].toLowerCase()}...`}
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                onFocus={() => setSearchFocused(true)}
                onBlur={() => setSearchFocused(false)}
                data-testid={selectors.backlog.search}
                widthClass="min-w-0 flex-1"
                className="!text-base"
              />
              <div className="relative">
                <Button
                  variant="outline"
                  size="sm"
                  aria-label="Sort backlog"
                  data-testid={selectors.backlog.sortButton}
                  onClick={() => { setShowSort(!showSort); setShowFilters(false); }}
                  className={sortField !== "priority" ? "border-cyan-500/50" : ""}
                >
                  <ArrowUpDown className="h-4 w-4" />
                </Button>
                {showSort && (
                  <div className="absolute left-0 top-full z-10 mt-2 w-48 rounded-lg border border-white/10 bg-slate-800 p-2 shadow-xl">
                    <div className="mb-1 text-sm font-medium text-slate-200 px-2 py-1">Sort by</div>
                    {SORT_OPTIONS.map((opt) => (
                      <button
                        key={opt.field}
                        onClick={() => { setSortField(opt.field); setShowSort(false); }}
                        className={`w-full rounded px-2 py-1.5 text-left text-sm ${
                          sortField === opt.field ? "bg-cyan-500/15 text-cyan-300" : "text-slate-300 hover:bg-slate-700/50"
                        }`}
                      >
                        {opt.label}
                      </button>
                    ))}
                  </div>
                )}
              </div>
              <div className="relative">
                <Button
                  variant="outline"
                  size="sm"
                  aria-label="Filter backlog"
                  data-testid={selectors.backlog.filter}
                  onClick={() => { setShowFilters(!showFilters); setShowSort(false); }}
                  className={statusFilter || (!statusFilter && showFinished) ? "border-cyan-500/50" : ""}
                >
                  <Filter className="h-4 w-4" />
                </Button>
                {showFilters && (
                  <div className="absolute left-0 top-full z-10 mt-2 w-56 rounded-lg border border-white/10 bg-slate-800 p-3 shadow-xl">
                    <div className="mb-2 flex items-center justify-between">
                      <span className="text-sm font-medium text-slate-200">Filters</span>
                      {(statusFilter || showFinished) && (
                        <button
                          onClick={() => { setStatusFilter(""); setShowFinished(false); }}
                          className="text-xs text-slate-400 hover:text-slate-200"
                        >
                          Clear all
                        </button>
                      )}
                    </div>
                    <div className="space-y-3">
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
                      {!statusFilter && stats.finishedCount > 0 && (
                        <label className="flex items-center gap-2 text-sm text-slate-300 cursor-pointer">
                          <input
                            type="checkbox"
                            checked={showFinished}
                            onChange={() => setShowFinished(!showFinished)}
                            className="rounded border-slate-600 bg-slate-700 text-cyan-500 focus:ring-cyan-500/30"
                            data-testid={selectors.backlog.showFinishedToggle}
                          />
                          Show {stats.finishedCount} archived
                        </label>
                      )}
                    </div>
                  </div>
                )}
              </div>
              <Button
                variant="outline"
                size="sm"
                aria-label="Toggle batch mode"
                data-testid={selectors.backlog.batchToggle}
                onClick={() => {
                  setBatchMode((prev) => {
                    if (prev) setSelectedKeys([]);
                    return !prev;
                  });
                  setShowSort(false);
                  setShowFilters(false);
                }}
                className={batchMode ? "border-cyan-500/50 text-cyan-300" : ""}
              >
                <CheckSquare className="h-4 w-4" />
              </Button>

              {kindItems.length > 0 && (
                <div className="hidden sm:flex items-center gap-2 ml-auto text-sm text-slate-400" data-testid={selectors.backlog.summaryStats}>
                  {stats.ready > 0 && (
                    <span className="text-cyan-400 whitespace-nowrap" data-testid={selectors.backlog.readyCount}>
                      {stats.ready} ready
                    </span>
                  )}
                  <StatusLegend
                    items={BACKLOG_STATUS_LEGEND_ITEMS}
                    title="Status Guide"
                    compact
                    data-testid={selectors.backlog.statusLegend}
                  />
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
      {isRefreshing && items.length > 0 && (
        <InlineLoadingIndicator
          label="Refreshing backlog..."
          testId="backlog-refreshing-indicator"
        />
      )}

      <div className="space-y-4">
        {(status === "loading" || !hasLoaded) && items.length === 0 && (
          <PageLoadingState
            label="Loading backlog..."
            variant="list"
            testId="backlog-loading-state"
          />
        )}

        {error && items.length === 0 && hasLoaded && (
          <ErrorState
            error={error}
            title="Unable to load backlog"
            onRetry={() => fetchBacklog({ force: true })}
          />
        )}

        {!error && kindItems.length === 0 && hasLoaded && !(activeKind === "all" && captures.length > 0) && (
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

        {/* Workshop mutation error banner */}
        {workshopError && (
          <div className="mb-3 flex items-center gap-2 rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-300">
            <X className="h-4 w-4 shrink-0 cursor-pointer hover:text-red-200" onClick={() => setWorkshopError(null)} />
            {workshopError}
          </div>
        )}

        {/* Backlog items + captures (unified rendering for all tabs) */}
        {(() => {
          const isUnfilteredAll = activeKind === "all" && !searchTerm && !statusFilter;
          const displayItems: FeedItem[] = isUnfilteredAll
            ? feedItems
            : sortedItems.map((item) => ({ type: "backlog" as const, item }));
          if (displayItems.length === 0) return null;
          return (
            <div className="space-y-3">
              {batchMode && (
                <Card className="border border-slate-700/70 bg-slate-900/45 p-3">
                  <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                    <div className="flex flex-wrap items-center gap-3 text-sm text-slate-300">
                      <label className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          aria-label="Select all queueable items"
                          checked={allQueueableSelected}
                          onChange={toggleSelectAllQueueable}
                          disabled={queueableFilteredItems.length === 0}
                        />
                        <span>Select all queueable</span>
                      </label>
                      <span className="text-slate-400">
                        {selectedQueueableItems.length} selected
                      </span>
                      <span className="text-xs text-slate-500">
                        {queueableFilteredItems.length} queueable in current view
                      </span>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          const names = selectedQueueableItems.map((item) => `${item.kind}/${item.name}`);
                          openFeedbackHub("export", names);
                        }}
                        disabled={!hasAnySelectedQueueable}
                      >
                        <MessageSquareText className="mr-1 h-3 w-3" />
                        Export Selected
                      </Button>
                      <Button
                        size="sm"
                        onClick={() =>
                          setRunModalTargets(
                            selectedQueueableItems.map((item) => ({
                              kind: item.kind,
                              name: item.name,
                              title: item.title,
                            })),
                          )
                        }
                        disabled={!hasAnySelectedQueueable}
                      >
                        <Play className="mr-1 h-3 w-3" />
                        Run Selected
                      </Button>
                    </div>
                  </div>
                </Card>
              )}
              <ResponsiveList data-testid={selectors.backlog.grid}>
                {displayItems.map((entry) => {
                  if (entry.type === "capture") {
                    return (
                      <ResponsiveListItem
                        key={`capture-${entry.capture.id}`}
                        className="group relative block overflow-hidden"
                      >
                        <CaptureCard
                          capture={entry.capture}
                          onEditItem={(prefill: BacklogFormValues) => {
                            setCreatePrefill(prefill);
                            setShowCreate(true);
                          }}
                        />
                      </ResponsiveListItem>
                    );
                  }
                  const item = entry.item;
                  const itemKey = `${item.kind}/${item.name}`;
                  const readiness = readinessMap.get(itemKey);
                  const reasons = entry.type === "attention"
                    ? entry.reasons
                    : (attentionReasonsMap.get(itemKey) ?? []);
                  return (
                    <ResponsiveListItem
                      as={Link}
                      key={`${item.kind}-${item.name}`}
                      to={`/backlog/${item.kind}/${item.name}`}
                      state={{ backlogSearch: location.search }}
                      interactive
                      className="group relative block overflow-hidden"
                      data-testid={selectors.backlog.cardByName({ kind: item.kind, name: item.name })}
                    >
                      <BacklogCard
                        item={item}
                        allItems={items}
                        readinessData={readiness}
                        itemActions={getItemActions({
                          item,
                          allItems: items,
                          readinessReady: readiness ? readiness.ready : null,
                          pendingSynthesis: readiness?.pendingSynthesis ?? false,
                          agentRunning: activeRunKeys.has(itemKey),
                          hasPendingDecisions: (pendingQuestionsMap.get(itemKey)?.length ?? 0) > 0,
                          hasExecutionHistory: item.status === "completed" || item.status === "failed",
                        })}
                        attentionReasons={reasons}
                        pendingQuestions={pendingQuestionsMap.get(itemKey)}
                        isStepperCompleted={completedSteppers.has(itemKey)}
                        transitionResult={transitionItems.get(itemKey)}
                        onStepperCompleted={(result) => handleStepperCompleted(itemKey, item, result)}
                        batchMode={batchMode}
                        isSelected={selectedKeys.includes(itemKey)}
                        onToggleSelection={() => toggleItemSelection(item)}
                        onRun={() => setRunModalTarget({ kind: item.kind, name: item.name, title: item.title })}
                        onArchive={() => archiveMutation.mutate({ kind: item.kind as BacklogKind, name: item.name, item })}
                        onFollowUp={() => navigate(`/backlog/${item.kind}/${item.name}?action=followup`)}
                        onFinalize={() => workshopMutation.mutate({
                          kind: item.kind as BacklogKind,
                          name: item.name,
                          mode: "finalize",
                          prompt: "Finalize the latest workshop answers into the primary deliverable for this backlog item.",
                        })}
                        onWorkshop={() => workshopMutation.mutate({
                          kind: item.kind as BacklogKind,
                          name: item.name,
                          mode: "workshop",
                          prompt: "Run the next workshop round for this backlog item.",
                        })}
                        runningLabel={activeRunLabels.get(itemKey)}
                        archivePending={archiveMutation.isPending}
                        finalizePending={
                          workshopMutation.isPending &&
                          workshopMutation.variables?.kind === item.kind &&
                          workshopMutation.variables?.name === item.name &&
                          workshopMutation.variables?.mode === "finalize"
                        }
                        workshopPending={
                          workshopMutation.isPending &&
                          workshopMutation.variables?.kind === item.kind &&
                          workshopMutation.variables?.name === item.name &&
                          workshopMutation.variables?.mode === "workshop"
                        }
                        workshopLabel={(readiness?.roundsCompleted ?? 0) > 0 ? "Next Round" : "Workshop"}
                      />
                    </ResponsiveListItem>
                  );
                })}
              </ResponsiveList>
            </div>
          );
        })()}
      </div>

      <FloatingActionButton
        icon={<Plus className="h-5 w-5" />}
        label={activeTab.ctaLabel}
        onClick={() => setShowCreate(true)}
        disabled={createMutation.isPending}
        className={activeKind === "all" ? "hidden" : "md:hidden"}
      />

      <BacklogFormDialog
        isOpen={showCreate}
        mode="create"
        defaultKind={createPrefill?.kind ?? (activeKind === "all" ? "idea" : activeKind)}
        initialValues={createPrefill}
        isSubmitting={createMutation.isPending}
        submitError={createError}
        onClose={() => {
          setShowCreate(false);
          setCreatePrefill(undefined);
          createMutation.reset();
        }}
        onSubmit={(values) => createMutation.mutate(values)}
      />

      <RunBacklogModal
        isOpen={runModalTarget !== null || runModalTargets !== null}
        onClose={() => {
          setRunModalTarget(null);
          setRunModalTargets(null);
        }}
        target={runModalTarget ?? undefined}
        targets={runModalTargets ?? undefined}
        readinessData={runModalTarget ? readinessMap.get(`${runModalTarget.kind}/${runModalTarget.name}`) ?? null : null}
        readinessDataMap={readinessMap}
        onSuccess={() => {
          void fetchBacklog({ force: true });
          setSelectedKeys([]);
          setRunModalTarget(null);
          setRunModalTargets(null);
        }}
      />

      <FeedbackHubModal
        isOpen={showFeedbackHub}
        onClose={() => setShowFeedbackHub(false)}
        feedbackSummary={feedbackSummary}
        activeKind={activeKind}
        statusFilter={statusFilter}
        selectedNames={feedbackHubSelectedNames}
        onDataChanged={() => {
          void fetchBacklog({ force: true });
          void summaryQuery.refetch();
        }}
        initialTab={feedbackHubInitialTab}
      />
    </div>
  );
}
