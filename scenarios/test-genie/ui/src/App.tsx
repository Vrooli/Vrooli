import { useCallback, useEffect, useId, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowRight, ArrowUpRight, Info, Search } from "lucide-react";
import { selectors } from "./consts/selectors";
import { Button } from "./components/ui/button";
import {
  fetchExecutionHistory,
  fetchHealth,
  fetchScenarioSummaries,
  fetchSuiteRequests,
  queueSuiteRequest,
  triggerSuiteExecution,
  type ApiHealthResponse,
  type ScenarioSummary,
  type SuiteExecutionResult,
  type SuiteRequest
} from "./lib/api";
import { cn } from "./lib/utils";

const REQUESTED_TYPE_OPTIONS = ["unit", "integration", "performance", "vault", "regression"] as const;
const PRIORITY_OPTIONS = ["low", "normal", "high", "urgent"] as const;
const EXECUTION_PRESETS = ["quick", "smoke", "comprehensive"] as const;
const DEFAULT_REQUEST_TYPES = ["unit", "integration"];
const SECTION_ANCHORS = {
  scenarioFocus: "scenario-focus-section",
  scenarioDirectory: "scenario-directory-section",
  catalogOverview: "scenario-catalog-overview",
  scenarioCoverage: "scenario-coverage-section",
  queueForm: "queue-form-section",
  executionForm: "execution-form-section",
  suiteRequests: "suite-requests-section",
  executionHistory: "execution-history-section",
  signalHighlights: "signal-highlights-section",
  queueMetrics: "queue-metrics-section",
  latestExecution: "latest-execution-section"
} as const;

const relativeFormatter = new Intl.RelativeTimeFormat("en", { numeric: "auto" });

const DASHBOARD_TABS = [
  { key: "overview", label: "Overview", description: "Intent + persona guides" },
  { key: "operate", label: "Operate", description: "Focus, queue, and run suites" },
  { key: "signals", label: "Signals", description: "Backlog and execution health" },
  { key: "catalog", label: "Catalog", description: "Scenario directory + portfolio status" }
] as const;

const HERO_OUTCOMES = [
  {
    title: "Plan coverage",
    description: "Pick or search for a scenario, then focus everything around that workstream."
  },
  {
    title: "Queue suites",
    description: "Tell Test Genie what coverage you need and let it delegate or fall back instantly."
  },
  {
    title: "Review signals",
    description: "See backlog pressure, failures, and shortcuts without scanning multiple tools."
  }
] as const;

type DashboardTabKey = (typeof DASHBOARD_TABS)[number]["key"];

const PRESET_DETAILS = {
  quick: {
    label: "Quick preset",
    description: "Structure + unit phases replace the bash smoke check so you can sanity check scaffolding without heavy runners.",
    phases: ["structure", "unit"]
  },
  smoke: {
    label: "Smoke preset",
    description: "Structure + integration ensure orchestrator parity with the legacy scripts without leaving the Go API.",
    phases: ["structure", "integration"]
  },
  comprehensive: {
    label: "Comprehensive preset",
    description: "Full Go-native replacement for scripts/scenarios/testing with dependency, unit, integration, business, and performance coverage.",
    phases: ["structure", "dependencies", "unit", "integration", "business", "performance"]
  }
} as const;

const PHASE_LABELS: Record<string, string> = {
  structure: "Structure validation",
  dependencies: "Dependency audit",
  unit: "Unit tests",
  integration: "Integration suite",
  business: "Business validation",
  performance: "Performance checks"
};

const PRESET_ENTRIES = Object.entries(PRESET_DETAILS) as Array<
  [keyof typeof PRESET_DETAILS, (typeof PRESET_DETAILS)[keyof typeof PRESET_DETAILS]]
>;

type QueueFormState = {
  scenarioName: string;
  requestedTypes: string[];
  coverageTarget: number;
  priority: string;
  notes: string;
};

type ExecutionFormState = {
  scenarioName: string;
  preset: string;
  failFast: boolean;
  suiteRequestId: string;
};

const initialQueueForm: QueueFormState = {
  scenarioName: "",
  requestedTypes: DEFAULT_REQUEST_TYPES,
  coverageTarget: 95,
  priority: "normal",
  notes: ""
};

const initialExecutionForm: ExecutionFormState = {
  scenarioName: "",
  preset: "quick",
  failFast: true,
  suiteRequestId: ""
};

export default function App() {
  const queryClient = useQueryClient();
  const scenarioOptionsDatalistId = useId();
  const [queueForm, setQueueForm] = useState<QueueFormState>(initialQueueForm);
  const [executionForm, setExecutionForm] = useState<ExecutionFormState>(initialExecutionForm);
  const [queueFeedback, setQueueFeedback] = useState<string | null>(null);
  const [executionFeedback, setExecutionFeedback] = useState<string | null>(null);
  const [focusScenario, setFocusScenario] = useState("");
  const [activeTab, setActiveTab] = useState<DashboardTabKey>("overview");
  const [activeQuickRunId, setActiveQuickRunId] = useState<string | null>(null);
  const [scenarioSearch, setScenarioSearch] = useState("");
  const [historyScenario, setHistoryScenario] = useState<string | null>(null);
  const scrollToSection = useCallback((sectionId: string) => {
    if (typeof window === "undefined") {
      return;
    }
    const element = document.getElementById(sectionId);
    if (element) {
      element.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }, []);

  const navigateToSection = useCallback(
    (tabKey: DashboardTabKey, sectionId: string) => {
      setActiveTab(tabKey);
      setTimeout(() => {
        scrollToSection(sectionId);
      }, 50);
    },
    [scrollToSection]
  );

  const healthQuery = useQuery<ApiHealthResponse>({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 10000
  });

  const suiteRequestsQuery = useQuery<SuiteRequest[]>({
    queryKey: ["suite-requests"],
    queryFn: fetchSuiteRequests,
    refetchInterval: 10000
  });

  const executionsQuery = useQuery<SuiteExecutionResult[]>({
    queryKey: ["executions"],
    queryFn: () => fetchExecutionHistory({ limit: 6 }),
    refetchInterval: 12000
  });

  const scenarioSummariesQuery = useQuery<ScenarioSummary[]>({
    queryKey: ["scenario-summaries"],
    queryFn: fetchScenarioSummaries,
    refetchInterval: 15000
  });

  const scenarioHistoryQuery = useQuery<SuiteExecutionResult[]>({
    queryKey: ["scenario-history", historyScenario],
    queryFn: () =>
      historyScenario ? fetchExecutionHistory({ scenario: historyScenario, limit: 20 }) : Promise.resolve([]),
    enabled: Boolean(historyScenario)
  });
  const scenarioSummaries = scenarioSummariesQuery.data ?? [];
  const historyExecutions = historyScenario ? scenarioHistoryQuery.data ?? [] : [];

  const applyFocusScenario = (value: string) => {
    const trimmed = value.trim();
    setFocusScenario(trimmed);
    setQueueForm((prev) => ({ ...prev, scenarioName: trimmed }));
    setExecutionForm((prev) => ({ ...prev, scenarioName: trimmed }));
  };

  const clearFocusScenario = () => {
    setFocusScenario("");
    setQueueForm((prev) => ({ ...prev, scenarioName: "" }));
    setExecutionForm((prev) => ({ ...prev, scenarioName: "" }));
  };

  const queueMutation = useMutation({
    mutationFn: queueSuiteRequest,
    onMutate: () => setQueueFeedback(null),
    onSuccess: (result) => {
      setQueueFeedback(`Suite request queued for ${result.scenarioName}`);
      queryClient.invalidateQueries({ queryKey: ["suite-requests"] });
      queryClient.invalidateQueries({ queryKey: ["health"] });
      setQueueForm((prev) => ({
        ...prev,
        scenarioName: "",
        notes: "",
        requestedTypes: prev.requestedTypes.length ? prev.requestedTypes : DEFAULT_REQUEST_TYPES
      }));
    },
    onError: (err: Error) => {
      setQueueFeedback(err.message);
    }
  });

  const executionMutation = useMutation({
    mutationFn: triggerSuiteExecution,
    onMutate: () => setExecutionFeedback(null),
    onSuccess: (result) => {
      const resultLabel = result.success ? "completed" : "failed";
      setExecutionFeedback(`Execution ${resultLabel} in ${(result.phaseSummary.durationSeconds / 60).toFixed(1)} min`);
      queryClient.invalidateQueries({ queryKey: ["executions"] });
      queryClient.invalidateQueries({ queryKey: ["health"] });
    },
    onError: (err: Error) => {
      setExecutionFeedback(err.message);
    }
  });

  const quickRunMutation = useMutation<SuiteExecutionResult, Error, SuiteRequest>({
    mutationFn: (request) =>
      triggerSuiteExecution({
        scenarioName: request.scenarioName,
        suiteRequestId: request.id,
        failFast: true,
        preset: presetFromPriority(request.priority)
      }),
    onMutate: (request) => {
      setExecutionFeedback(null);
      applyFocusScenario(request.scenarioName);
      setActiveQuickRunId(request.id);
    },
    onSuccess: (result) => {
      const status = result.success ? "completed" : "failed";
      setExecutionFeedback(
        `Inline execution ${status} for ${result.scenarioName} (${result.phaseSummary.passed}/${result.phaseSummary.total} phases)`
      );
      queryClient.invalidateQueries({ queryKey: ["executions"] });
      queryClient.invalidateQueries({ queryKey: ["health"] });
      queryClient.invalidateQueries({ queryKey: ["suite-requests"] });
    },
    onError: (err: Error) => {
      setExecutionFeedback(err.message);
    },
    onSettled: () => setActiveQuickRunId(null)
  });

  const queueSnapshot = healthQuery.data?.operations?.queue;
  const lastExecution = healthQuery.data?.operations?.lastExecution;

  const queueMetrics = useMemo(
    () => [
      { label: "Pending", value: queueSnapshot?.pending ?? 0 },
      { label: "Running", value: queueSnapshot?.running ?? 0 },
      { label: "Completed (24h)", value: queueSnapshot?.completed ?? 0 },
      { label: "Failed (24h)", value: queueSnapshot?.failed ?? 0 }
    ],
    [queueSnapshot?.completed, queueSnapshot?.failed, queueSnapshot?.pending, queueSnapshot?.running]
  );

  const requests = suiteRequestsQuery.data ?? [];
  const executions = executionsQuery.data ?? [];
  const normalizedFocus = focusScenario.trim().toLowerCase();
  const focusActive = normalizedFocus.length > 0;

  const scenarioOptions = useMemo(() => {
    type Entry = { label: string; timestamp: number };
    const now = Date.now();
    const register = (collection: Map<string, Entry>, label?: string, timestamp?: string) => {
      const trimmed = label?.trim();
      if (!trimmed) {
        return;
      }
      const normalized = trimmed.toLowerCase();
      const parsed = timestamp ? Date.parse(timestamp) : now;
      const score = Number.isFinite(parsed) ? parsed : now;
      const existing = collection.get(normalized);
      if (!existing || score > existing.timestamp) {
        collection.set(normalized, { label: trimmed, timestamp: score });
      }
    };

    const collected = new Map<string, Entry>();
    scenarioSummaries.forEach((summary) =>
      register(collected, summary.scenarioName, summary.lastExecutionAt ?? summary.lastRequestAt)
    );
    requests.forEach((req) => register(collected, req.scenarioName, req.updatedAt ?? req.createdAt));
    executions.forEach((execution) => register(collected, execution.scenarioName, execution.completedAt ?? execution.startedAt));
    register(collected, queueForm.scenarioName);
    register(collected, executionForm.scenarioName);

    return Array.from(collected.values())
      .sort((a, b) => b.timestamp - a.timestamp)
      .map((entry) => entry.label);
  }, [executionForm.scenarioName, executions, queueForm.scenarioName, requests, scenarioSummaries]);

  const filteredRequests = useMemo(() => {
    if (!focusActive) {
      return requests;
    }
    return requests.filter((req) => req.scenarioName.toLowerCase() === normalizedFocus);
  }, [focusActive, normalizedFocus, requests]);

  const filteredExecutions = useMemo(() => {
    if (!focusActive) {
      return executions;
    }
    return executions.filter((execution) => execution.scenarioName.toLowerCase() === normalizedFocus);
  }, [executions, focusActive, normalizedFocus]);
  const heroExecution = useMemo<SuiteExecutionResult | undefined>(() => {
    if (lastExecution) {
      return {
        executionId: lastExecution.executionId,
        suiteRequestId: undefined,
        scenarioName: lastExecution.scenario,
        startedAt: lastExecution.startedAt,
        completedAt: lastExecution.completedAt,
        success: lastExecution.success,
        preset: lastExecution.preset,
        phases: [],
        phaseSummary: lastExecution.phaseSummary
      };
    }
    return executions.length > 0 ? executions[0] : undefined;
  }, [executions, lastExecution]);

  const actionableRequest = useMemo(() => selectActionableRequest(requests), [requests]);

  const lastFailedExecution = useMemo(
    () => executions.find((execution) => execution.success === false),
    [executions]
  );

  const focusQueueStats = useMemo(() => {
    if (!focusActive) {
      return null;
    }
    const actionableCount = filteredRequests.filter((req) => isActionableRequest(req.status)).length;
    const mostRecentRequest = [...filteredRequests].sort(
      (a, b) =>
        parseTimestamp(b.updatedAt ?? b.createdAt) - parseTimestamp(a.updatedAt ?? a.createdAt)
    )[0];
    const recentExecution = filteredExecutions[0] ?? null;
    const failedExecution =
      filteredExecutions.find((execution) => execution.success === false) ?? null;
    const nextRequest = selectActionableRequest(filteredRequests);
    return {
      actionableCount,
      totalCount: filteredRequests.length,
      mostRecentRequest: mostRecentRequest ?? null,
      recentExecution,
      failedExecution,
      nextRequest
    };
  }, [filteredExecutions, filteredRequests, focusActive]);

  const scenarioDirectoryEntries = useMemo(() => {
    if (scenarioSummaries.length === 0) {
      return [] as Array<ScenarioSummary & { lastActivity: number }>;
    }
    return scenarioSummaries
      .map((summary) => {
        const lastActivity = Math.max(
          timestampOrZero(summary.lastExecutionAt),
          timestampOrZero(summary.lastRequestAt)
        );
        return { ...summary, lastActivity };
      })
      .sort((a, b) => b.lastActivity - a.lastActivity);
  }, [scenarioSummaries]);

  const filteredScenarioDirectory = useMemo(() => {
    const trimmed = scenarioSearch.trim().toLowerCase();
    if (!trimmed) {
      return scenarioDirectoryEntries;
    }
    return scenarioDirectoryEntries.filter((entry) =>
      entry.scenarioName.toLowerCase().includes(trimmed)
    );
  }, [scenarioDirectoryEntries, scenarioSearch]);
  const scenarioStatusRows = scenarioDirectoryEntries;
  const catalogStats = useMemo(() => {
    if (scenarioDirectoryEntries.length === 0) {
      return { tracked: 0, pending: 0, failing: 0, idle: 0 };
    }
    const tracked = scenarioDirectoryEntries.length;
    const pending = scenarioDirectoryEntries.filter((entry) => entry.pendingRequests > 0).length;
    const failing = scenarioDirectoryEntries.filter((entry) => entry.lastExecutionSuccess === false).length;
    const idle = scenarioDirectoryEntries.filter(
      (entry) => entry.pendingRequests === 0 && !entry.lastExecutionAt
    ).length;
    return { tracked, pending, failing, idle };
  }, [scenarioDirectoryEntries]);

  const queuePendingCount = useMemo(
    () => requests.filter((req) => isActionableRequest(req.status)).length,
    [requests]
  );

  const activeTabMeta = useMemo(
    () => DASHBOARD_TABS.find((tab) => tab.key === activeTab) ?? DASHBOARD_TABS[0],
    [activeTab]
  );

  const selectedHistorySummary = useMemo(() => {
    if (!historyScenario) {
      return null;
    }
    return (
      scenarioDirectoryEntries.find(
        (entry) => entry.scenarioName.toLowerCase() === historyScenario.toLowerCase()
      ) ?? null
    );
  }, [historyScenario, scenarioDirectoryEntries]);

  useEffect(() => {
    if (!historyScenario) {
      return;
    }
    const exists = scenarioDirectoryEntries.some(
      (entry) => entry.scenarioName.toLowerCase() === historyScenario.toLowerCase()
    );
    if (!exists) {
      setHistoryScenario(null);
    }
  }, [historyScenario, scenarioDirectoryEntries]);

  const quickNavItems = [
    {
      key: "directory",
      eyebrow: "Shortcut",
      title: "Browse scenario directory",
      description: "Search every tracked scenario and jump straight into queue or rerun actions.",
      statValue:
        scenarioDirectoryEntries.length > 0
          ? `${scenarioDirectoryEntries.length} tracked`
          : "No history yet",
      statLabel: scenarioDirectoryEntries.length > 0 ? "Auto-tracked" : "Populates after first run",
      actionLabel: "Open directory",
      onClick: () => navigateToSection("catalog", SECTION_ANCHORS.scenarioDirectory)
    },
    {
      key: "focus",
      eyebrow: "Step 1",
      title: "Focus a scenario",
      description: "Set the intent once so every section filters to the same scenario.",
      statValue: focusActive ? focusScenario : "Not set",
      statLabel: focusActive ? "Active focus" : "Needed for shortcuts",
      actionLabel: focusActive ? "Adjust focus" : "Set focus",
      onClick: () => {
        if (!focusScenario && scenarioOptions.length > 0) {
          applyFocusScenario(scenarioOptions[0]);
        }
        navigateToSection("operate", SECTION_ANCHORS.scenarioFocus);
      }
    },
    {
      key: "queue",
      eyebrow: "Step 2",
      title: "Queue AI coverage",
      description: "Describe the coverage outcome you need; the UI handles API + delegation wiring.",
      statValue: `${queuePendingCount} pending`,
      statLabel: `${requests.length} total`,
      actionLabel: "Open queue form",
      onClick: () => navigateToSection("operate", SECTION_ANCHORS.queueForm)
    },
    {
      key: "run",
      eyebrow: "Step 3",
      title: "Run a curated preset",
      description: "Trigger scenario-local runners without touching scripts or bash wrappers.",
      statValue: heroExecution
        ? heroExecution.success
          ? "Last run passed"
          : "Last run failed"
        : "No runs yet",
      statLabel: heroExecution ? `Completed ${formatRelative(heroExecution.completedAt)}` : "Record a run",
      actionLabel: "Open runner",
      onClick: () => navigateToSection("operate", SECTION_ANCHORS.executionForm)
    },
    {
      key: "investigate",
      eyebrow: "Signal",
      title: "Investigate backlog/failures",
      description: "Jump to queue or history views from the riskiest request or failure.",
      statValue: lastFailedExecution
        ? lastFailedExecution.scenarioName
        : actionableRequest
        ? actionableRequest.scenarioName
        : "Nothing urgent",
      statLabel: lastFailedExecution
        ? `Failed ${formatRelative(lastFailedExecution.completedAt)}`
        : actionableRequest
        ? `${actionableRequest.priority} priority queued`
        : "Monitor queue",
      actionLabel: lastFailedExecution ? "Go to history" : "Review queue",
      onClick: () =>
        navigateToSection(
          "signals",
          lastFailedExecution ? SECTION_ANCHORS.executionHistory : SECTION_ANCHORS.suiteRequests
        )
    }
  ];

  const signalNavItems = [
    { key: "highlights", label: "Highlights", anchor: SECTION_ANCHORS.signalHighlights },
    { key: "queue", label: "Queue health", anchor: SECTION_ANCHORS.queueMetrics },
    { key: "latest", label: "Latest run", anchor: SECTION_ANCHORS.latestExecution },
    { key: "requests", label: "Queue table", anchor: SECTION_ANCHORS.suiteRequests },
    { key: "history", label: "Execution history", anchor: SECTION_ANCHORS.executionHistory }
  ];

  const handleQueueSubmit = (evt: FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    if (!queueForm.scenarioName.trim()) {
      setQueueFeedback("Scenario name is required");
      return;
    }
    const requestedTypes =
      queueForm.requestedTypes.length > 0 ? queueForm.requestedTypes : DEFAULT_REQUEST_TYPES;
    queueMutation.mutate({
      scenarioName: queueForm.scenarioName.trim(),
      requestedTypes,
      coverageTarget: clampCoverage(queueForm.coverageTarget),
      priority: queueForm.priority,
      notes: queueForm.notes?.trim()
    });
  };

  const handleExecutionSubmit = (evt: FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    if (!executionForm.scenarioName.trim()) {
      setExecutionFeedback("Scenario name is required");
      return;
    }
    executionMutation.mutate({
      scenarioName: executionForm.scenarioName.trim(),
      preset: executionForm.preset,
      failFast: executionForm.failFast,
      suiteRequestId: executionForm.suiteRequestId.trim() || undefined
    });
  };

  const toggleRequestedType = (type: string) => {
    setQueueForm((prev) => {
      const exists = prev.requestedTypes.includes(type);
      const nextTypes = exists
        ? prev.requestedTypes.filter((t) => t !== type)
        : [...prev.requestedTypes, type];
      return { ...prev, requestedTypes: nextTypes };
    });
  };

  const prefillFromRequest = (request: SuiteRequest) => {
    applyFocusScenario(request.scenarioName);
    setQueueForm({
      scenarioName: request.scenarioName,
      requestedTypes: request.requestedTypes,
      coverageTarget: request.coverageTarget,
      priority: request.priority,
      notes: request.notes ?? ""
    });
    setExecutionForm((prev) => ({
      ...prev,
      scenarioName: request.scenarioName,
      suiteRequestId: request.id
    }));
  };

  const prefillFromExecution = (execution: SuiteExecutionResult) => {
    applyFocusScenario(execution.scenarioName);
    setExecutionForm((prev) => ({
      ...prev,
      scenarioName: execution.scenarioName,
      preset: execution.preset ?? prev.preset,
      suiteRequestId: execution.suiteRequestId ?? ""
    }));
  };

  const prefillFromSummary = (summary: ScenarioSummary) => {
    applyFocusScenario(summary.scenarioName);
    setQueueForm((prev) => ({
      ...prev,
      scenarioName: summary.scenarioName,
      requestedTypes:
        summary.lastRequestTypes && summary.lastRequestTypes.length > 0
          ? summary.lastRequestTypes
          : prev.requestedTypes.length > 0
          ? prev.requestedTypes
          : DEFAULT_REQUEST_TYPES,
      coverageTarget: summary.lastRequestCoverageTarget ?? prev.coverageTarget ?? 95,
      priority: summary.lastRequestPriority ?? prev.priority,
      notes: summary.lastRequestNotes ?? prev.notes
    }));
  };

  const prefillRunnerFromSummary = (summary: ScenarioSummary) => {
    applyFocusScenario(summary.scenarioName);
    setExecutionForm((prev) => ({
      ...prev,
      scenarioName: summary.scenarioName,
      preset: summary.lastExecutionPreset || prev.preset,
      suiteRequestId: ""
    }));
  };

  const handleFocusQueueShortcut = () => {
    if (focusQueueStats?.nextRequest) {
      prefillFromRequest(focusQueueStats.nextRequest);
    } else if (focusQueueStats?.mostRecentRequest) {
      prefillFromRequest(focusQueueStats.mostRecentRequest);
    }
    navigateToSection("operate", SECTION_ANCHORS.queueForm);
  };

  const handleFocusRunnerShortcut = () => {
    const executionCandidate = focusQueueStats?.recentExecution ?? focusQueueStats?.failedExecution;
    if (executionCandidate) {
      prefillFromExecution(executionCandidate);
    }
    navigateToSection("operate", SECTION_ANCHORS.executionForm);
  };

  const handleFocusSignalsShortcut = () => {
    if (focusQueueStats?.failedExecution) {
      navigateToSection("signals", SECTION_ANCHORS.executionHistory);
      return;
    }
    if (focusQueueStats?.actionableCount) {
      navigateToSection("signals", SECTION_ANCHORS.suiteRequests);
      return;
    }
    navigateToSection("signals", SECTION_ANCHORS.queueMetrics);
  };

  const handleQuickRun = (request: SuiteRequest) => {
    if (quickRunMutation.isPending) {
      return;
    }
    quickRunMutation.mutate(request);
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      <div className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
        <datalist id={scenarioOptionsDatalistId}>
          {scenarioOptions.map((name) => (
            <option key={name} value={name} />
          ))}
        </datalist>
        <header
          className="rounded-3xl border border-white/10 bg-white/[0.04] p-8 shadow-lg backdrop-blur"
          data-testid={selectors.dashboard.hero}
        >
          <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Test Genie</p>
          <div className="mt-3 flex flex-wrap items-center gap-3">
            <h1 className="text-3xl font-semibold">Coverage control room</h1>
            <InfoTip
              title="Why this view exists"
              description="Keep your intent on the outcomes—plan coverage, run suites, and clear risks—while the UI translates each action into Go-native orchestration calls."
            />
          </div>
          <p className="mt-4 text-base text-slate-300">
            Queue AI-generated suites, run curated presets, and spot backlog risks without memorizing API
            terminology. This dashboard speaks the Go runner&apos;s language for you so you only
            decide what scenario to protect next.
          </p>
          <div className="mt-5 grid gap-4 md:grid-cols-3">
            {HERO_OUTCOMES.map((outcome) => (
              <article key={outcome.title} className="rounded-2xl border border-white/5 bg-black/30 p-4">
                <p className="text-sm font-semibold text-white">{outcome.title}</p>
                <p className="mt-2 text-xs text-slate-300">{outcome.description}</p>
              </article>
            ))}
          </div>
        </header>
        <nav className="rounded-2xl border border-white/10 bg-white/[0.02] p-4" data-testid={selectors.dashboard.experienceNavigator}>
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Workspace layers</p>
              <p className="mt-1 text-sm text-slate-300">{activeTabMeta.description}</p>
            </div>
            <div className="flex flex-wrap gap-2">
              {DASHBOARD_TABS.map((tab) => (
                <button
                  key={tab.key}
                  type="button"
                  onClick={() => setActiveTab(tab.key)}
                  className={cn(
                    "rounded-full border px-4 py-2 text-sm transition",
                    activeTab === tab.key
                      ? "border-cyan-400 bg-cyan-400/20 text-white"
                      : "border-white/20 text-slate-300 hover:border-white/50"
                  )}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>
          <div className="mt-4 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
            <label className="flex-1 text-sm">
              <span className="text-xs uppercase tracking-[0.3em] text-slate-400">Quick focus</span>
              <input
                className="mt-2 w-full rounded-full border border-white/10 bg-black/30 px-4 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                placeholder="Type or pick a scenario to sync every section"
                value={focusScenario}
                onChange={(evt) => applyFocusScenario(evt.target.value)}
                list={scenarioOptionsDatalistId}
              />
              <p className="mt-2 text-xs text-slate-400">
                Focus filters the queue + history tables and pre-fills all forms. Use it before you scroll.
              </p>
            </label>
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" onClick={() => navigateToSection("operate", SECTION_ANCHORS.scenarioFocus)}>
                Open focus tools
              </Button>
              {focusActive ? (
                <Button variant="outline" size="sm" onClick={clearFocusScenario}>
                  Clear focus
                </Button>
              ) : scenarioOptions.length > 0 ? (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => applyFocusScenario(scenarioOptions[0])}
                >
                  Use last scenario
                </Button>
              ) : null}
            </div>
          </div>
        </nav>
        <section
          className="rounded-2xl border border-white/10 bg-gradient-to-br from-emerald-500/5 via-slate-900/60 to-slate-900 p-5 text-sm"
          data-testid={selectors.dashboard.intentSummary}
        >
          {focusActive ? (
            <>
              <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                <div>
                  <p className="text-xs uppercase tracking-[0.3em] text-emerald-200">Intent rail</p>
                  <h2 className="mt-1 text-2xl font-semibold text-white">Focused on {focusScenario}</h2>
                  <p className="mt-2 text-slate-300">
                    Every CTA below is already filtered to this scenario so you can delegate, rerun, or inspect
                    signals without hunting through the tabs.
                  </p>
                </div>
                <div className="grid gap-3 sm:grid-cols-3">
                  <div className="rounded-2xl border border-white/10 bg-black/30 p-4 text-left">
                    <p className="text-xs uppercase tracking-wide text-slate-400">Queue</p>
                    <p className="mt-1 text-lg font-semibold text-white">
                      {focusQueueStats
                        ? `${focusQueueStats.actionableCount}/${focusQueueStats.totalCount} actionable`
                        : "No queue history"}
                    </p>
                    <p className="mt-1 text-xs text-slate-400">
                      {focusQueueStats?.mostRecentRequest
                        ? `Last request ${formatRelative(
                            focusQueueStats.mostRecentRequest.updatedAt ?? focusQueueStats.mostRecentRequest.createdAt
                          )}`
                        : "Queue a suite to start tracking"}
                    </p>
                  </div>
                  <div className="rounded-2xl border border-white/10 bg-black/30 p-4 text-left">
                    <p className="text-xs uppercase tracking-wide text-slate-400">Runner</p>
                    <p className="mt-1 text-lg font-semibold text-white">
                      {focusQueueStats?.recentExecution
                        ? focusQueueStats.recentExecution.success
                          ? "Last run passed"
                          : "Last run failed"
                        : "No runs yet"}
                    </p>
                    <p className="mt-1 text-xs text-slate-400">
                      {focusQueueStats?.recentExecution
                        ? formatRelative(focusQueueStats.recentExecution.completedAt)
                        : "Kick off a preset to record telemetry"}
                    </p>
                  </div>
                  <div className="rounded-2xl border border-white/10 bg-black/30 p-4 text-left">
                    <p className="text-xs uppercase tracking-wide text-slate-400">Signals</p>
                    <p className="mt-1 text-lg font-semibold text-white">
                      {focusQueueStats?.failedExecution
                        ? "Failure in backlog"
                        : focusQueueStats?.actionableCount
                        ? "Suite waiting"
                        : "All clear"}
                    </p>
                    <p className="mt-1 text-xs text-slate-400">
                      {focusQueueStats?.failedExecution
                        ? `Failed ${formatRelative(focusQueueStats.failedExecution.completedAt)}`
                        : focusQueueStats?.actionableCount
                        ? "Run queued suites to keep pace"
                        : "No alerts for this scenario"}
                    </p>
                  </div>
                </div>
              </div>
              <div className="mt-6 grid gap-3 md:grid-cols-3">
                <Button size="sm" className="justify-between" onClick={handleFocusQueueShortcut}>
                  Queue next suite
                  <ArrowRight className="h-4 w-4" />
                </Button>
                <Button size="sm" variant="outline" className="justify-between" onClick={handleFocusRunnerShortcut}>
                  Resume runner
                  <ArrowRight className="h-4 w-4" />
                </Button>
                <Button size="sm" variant="outline" className="justify-between" onClick={handleFocusSignalsShortcut}>
                  Review signals
                  <ArrowRight className="h-4 w-4" />
                </Button>
              </div>
            </>
          ) : (
            <>
              <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                <div>
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Intent rail</p>
                  <h2 className="mt-1 text-2xl font-semibold text-white">Align everything to a scenario</h2>
                  <p className="mt-2 text-slate-300">
                    Set a focus once and queue forms, runner presets, and signal tables will all follow. Pick from
                    recent activity or open the catalog if you are starting fresh.
                  </p>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button variant="outline" size="sm" onClick={() => navigateToSection("catalog", SECTION_ANCHORS.scenarioDirectory)}>
                    Browse catalog
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => navigateToSection("operate", SECTION_ANCHORS.scenarioFocus)}>
                    Open focus tools
                  </Button>
                </div>
              </div>
              <div className="mt-4 flex flex-wrap gap-2">
                {scenarioOptions.length === 0 && (
                  <span className="text-xs text-slate-400">
                    Queue or run a suite to record history. Recent scenarios will populate here automatically.
                  </span>
                )}
                {scenarioOptions.slice(0, 4).map((name) => (
                  <button
                    key={`focus-quick-${name}`}
                    type="button"
                    onClick={() => applyFocusScenario(name)}
                    className="rounded-full border border-white/10 px-4 py-2 text-sm text-slate-200 transition hover:border-white/50"
                  >
                    {name}
                  </button>
                ))}
              </div>
            </>
          )}
        </section>
        {activeTab === "overview" && (
          <>
        <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Guided flows</p>
              <h2 className="mt-2 text-2xl font-semibold">Start from your intent</h2>
              <p className="mt-2 text-sm text-slate-300">
                These shortcuts thread the main jobs—focus a scenario, queue AI suites, run the Go orchestrator,
                and investigate risks—so first-time users do not have to guess where to scroll next.
              </p>
            </div>
            <p className="text-xs text-slate-400">
              Tip: follow the order below once and your next loop becomes muscle memory.
            </p>
          </div>
          <div className="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            {quickNavItems.map((item) => (
              <article
                key={item.key}
                className="flex flex-col rounded-2xl border border-white/5 bg-black/30 p-5"
              >
                <p className="text-[0.65rem] uppercase tracking-[0.4em] text-slate-400">{item.eyebrow}</p>
                <h3 className="mt-2 text-xl font-semibold">{item.title}</h3>
                <p className="mt-2 text-sm text-slate-300">{item.description}</p>
                <div className="mt-4">
                  <p className="text-xs uppercase tracking-wide text-slate-400">{item.statLabel}</p>
                  <p className="mt-1 text-lg font-semibold">{item.statValue}</p>
                </div>
                <div className="mt-4">
                  <Button variant="outline" size="sm" onClick={item.onClick}>
                    {item.actionLabel}
                    <ArrowRight className="ml-2 h-4 w-4" />
                  </Button>
                </div>
              </article>
            ))}
          </div>
        </section>
        <section
          className="rounded-2xl border border-white/10 bg-gradient-to-br from-white/[0.05] to-white/[0.01] p-6"
        >
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Status at a glance</p>
              <h2 className="mt-2 text-2xl font-semibold">Know where to act next</h2>
              <p className="mt-2 text-sm text-slate-300">
                Summaries below pull directly from queue telemetry and the Go runner so you can
                immediately jump to the right CTA—queue a suite, rerun a preset, or review history.
              </p>
            </div>
            <p className="text-xs text-slate-400">
              Data refreshes automatically; click the CTAs to land in the right tab.
            </p>
          </div>
          <div className="mt-6 grid gap-4 md:grid-cols-3">
            <article className="rounded-2xl border border-white/5 bg-black/30 p-5">
              <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Queue health</p>
              <h3 className="mt-2 text-xl font-semibold">
                {queuePendingCount > 0 ? `${queuePendingCount} pending` : "No pending suites"}
              </h3>
              <p className="mt-2 text-sm text-slate-300">
                {actionableRequest
                  ? `${actionableRequest.scenarioName} (${actionableRequest.priority}) is waiting. Prefill forms or review the queue before it blocks progress.`
                  : "The queue is clear; queue new AI coverage or browse the scenario directory to pick your next target."}
              </p>
              <div className="mt-4 flex flex-wrap gap-2">
                <Button
                  size="sm"
                  onClick={() =>
                    actionableRequest
                      ? (prefillFromRequest(actionableRequest), navigateToSection("operate", SECTION_ANCHORS.queueForm))
                      : navigateToSection("catalog", SECTION_ANCHORS.scenarioDirectory)
                  }
                >
                  {actionableRequest ? "Prefill priority suite" : "Open directory"}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => navigateToSection("signals", SECTION_ANCHORS.suiteRequests)}
                >
                  View queue
                </Button>
              </div>
            </article>
            <article className="rounded-2xl border border-white/5 bg-black/30 p-5">
              <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Last execution</p>
              <h3 className="mt-2 text-xl font-semibold">
                {heroExecution
                  ? heroExecution.success
                    ? `${heroExecution.scenarioName} passed`
                    : `${heroExecution.scenarioName} failed`
                  : "No runs recorded"}
              </h3>
              <p className="mt-2 text-sm text-slate-300">
                {heroExecution
                  ? `${heroExecution.phaseSummary.passed}/${heroExecution.phaseSummary.total} phases · ${formatRelative(heroExecution.completedAt)}`
                  : "Kick off a preset to establish baseline telemetry without the legacy bash runner."}
              </p>
              <div className="mt-4 flex flex-wrap gap-2">
                <Button
                  size="sm"
                  onClick={() => {
                    if (heroExecution) {
                      prefillFromExecution(heroExecution);
                    }
                    navigateToSection("operate", SECTION_ANCHORS.executionForm);
                  }}
                >
                  {heroExecution ? "Prefill rerun" : "Open runner"}
                </Button>
                {heroExecution && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      applyFocusScenario(heroExecution.scenarioName);
                      navigateToSection("operate", SECTION_ANCHORS.scenarioFocus);
                    }}
                  >
                    Focus scenario
                  </Button>
                )}
              </div>
            </article>
            <article className="rounded-2xl border border-white/5 bg-black/30 p-5">
              <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Priority signal</p>
              <h3 className="mt-2 text-xl font-semibold">
                {lastFailedExecution
                  ? "Failure needs attention"
                  : actionableRequest
                  ? "Queued work pending"
                  : "All clear"}
              </h3>
              <p className="mt-2 text-sm text-slate-300">
                {lastFailedExecution
                  ? `${lastFailedExecution.scenarioName} failed ${formatRelative(lastFailedExecution.completedAt)}. Resume the preset or inspect history.`
                  : actionableRequest
                  ? `${actionableRequest.scenarioName} is ready to run (${actionableRequest.priority}).`
                  : "No failures or queued suites demand action. Continue generating coverage from the directory."}
              </p>
              <div className="mt-4 flex flex-wrap gap-2">
                <Button
                  size="sm"
                  onClick={() => {
                    if (lastFailedExecution) {
                      prefillFromExecution(lastFailedExecution);
                      navigateToSection("operate", SECTION_ANCHORS.executionForm);
                    } else if (actionableRequest) {
                      prefillFromRequest(actionableRequest);
                      navigateToSection("operate", SECTION_ANCHORS.queueForm);
                    } else if (scenarioDirectoryEntries.length > 0) {
                      applyFocusScenario(scenarioDirectoryEntries[0].scenarioName);
                      navigateToSection("operate", SECTION_ANCHORS.scenarioFocus);
                    } else {
                      navigateToSection("operate", SECTION_ANCHORS.queueForm);
                    }
                  }}
                >
                  {lastFailedExecution
                    ? "Resume failed run"
                    : actionableRequest
                    ? "Prefill queued suite"
                    : "Set focus"}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() =>
                    navigateToSection(
                      "signals",
                      lastFailedExecution ? SECTION_ANCHORS.executionHistory : SECTION_ANCHORS.suiteRequests
                    )
                  }
                >
                  {lastFailedExecution ? "View history" : "Review signals"}
                </Button>
              </div>
            </article>
          </div>
        </section>
          </>
        )}

        {activeTab === "signals" && (
          <section
            id={SECTION_ANCHORS.signalHighlights}
            className="rounded-2xl border border-white/10 bg-white/[0.04] p-6"
            data-testid={selectors.dashboard.flowHighlights}
          >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Flow shortcuts</p>
              <h2 className="mt-2 text-2xl font-semibold">Act on risks without hunting</h2>
              <p className="mt-2 text-sm text-slate-300">
                Move straight from backlog signals to the right action—queue focus, inline runs, or reruns—so
                operators stop bouncing between tables when suites pile up or fail.
              </p>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                healthQuery.refetch();
                executionsQuery.refetch();
                suiteRequestsQuery.refetch();
              }}
            >
              Refresh signals
              <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </div>
          <div className="mt-4 flex flex-wrap gap-2">
            {signalNavItems.map((item) => (
              <button
                key={item.key}
                type="button"
                onClick={() => navigateToSection("signals", item.anchor)}
                className="rounded-full border border-white/20 bg-transparent px-4 py-1.5 text-sm text-slate-300 transition hover:border-white/50 hover:text-white"
              >
                {item.label}
              </button>
            ))}
          </div>
          <div className="mt-6 grid gap-4 md:grid-cols-2">
            <article
              className="rounded-2xl border border-white/5 bg-black/30 p-5"
              data-testid={selectors.dashboard.flowHighlightsBacklog}
            >
              <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Queued attention</p>
              {actionableRequest ? (
                <>
                  <h3 className="mt-2 text-xl font-semibold">{actionableRequest.scenarioName}</h3>
                  <p className="mt-1 text-sm text-slate-300">
                    {actionableRequest.priority} priority · {actionableRequest.status} ·{" "}
                    Queued {formatRelative(actionableRequest.createdAt)}
                  </p>
                  <div className="mt-4 flex flex-wrap gap-3">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        prefillFromRequest(actionableRequest);
                        navigateToSection("operate", SECTION_ANCHORS.queueForm);
                      }}
                      data-testid={selectors.actions.flowPrefillRequest}
                    >
                      Prefill forms
                    </Button>
                    <Button
                      size="sm"
                      onClick={() => handleQuickRun(actionableRequest)}
                      disabled={quickRunMutation.isPending}
                      data-testid={selectors.actions.flowRunInlineRequest}
                    >
                      {quickRunMutation.isPending ? "Running…" : "Run inline"}
                    </Button>
                  </div>
                </>
              ) : (
                <p className="mt-2 text-sm text-slate-400">
                  No queued or delegated requests are waiting. New suites will appear here automatically.
                </p>
              )}
            </article>
            <article
              className="rounded-2xl border border-white/5 bg-black/30 p-5"
              data-testid={selectors.dashboard.flowHighlightsRerun}
            >
              <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Failure follow-up</p>
              {lastFailedExecution ? (
                <>
                  <h3 className="mt-2 text-xl font-semibold">{lastFailedExecution.scenarioName}</h3>
                  <p className="mt-1 text-sm text-slate-300">
                    {lastFailedExecution.preset ? `${lastFailedExecution.preset} preset` : "Custom phases"} ·{" "}
                    {lastFailedExecution.phaseSummary.failed} failed phase(s)
                  </p>
                  <p className="text-sm text-slate-400">
                    Failed {formatRelative(lastFailedExecution.completedAt)}
                  </p>
                  <div className="mt-4 flex flex-wrap gap-3">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        prefillFromExecution(lastFailedExecution);
                        navigateToSection("operate", SECTION_ANCHORS.executionForm);
                      }}
                      data-testid={selectors.actions.flowPrefillRerun}
                    >
                      Prefill rerun
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="border-transparent text-slate-200 hover:border-white/40"
                      onClick={() => applyFocusScenario(lastFailedExecution.scenarioName)}
                    >
                      Focus scenario
                    </Button>
                  </div>
                </>
              ) : (
                <p className="mt-2 text-sm text-slate-400">
                  The last {executions.length > 0 ? executions.length : "few"} executions passed. Failures will surface
                  here with rerun shortcuts.
                </p>
              )}
            </article>
          </div>
          </section>
        )}

        {activeTab === "operate" && (
          <section
            id={SECTION_ANCHORS.scenarioFocus}
            className="rounded-2xl border border-white/10 bg-white/[0.03] p-6"
            data-testid={selectors.dashboard.scenarioFocus}
          >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Scenario Focus</p>
              <div className="mt-2 flex items-center gap-2">
                <h2 className="text-2xl font-semibold">Work on a single scenario faster</h2>
                <InfoTip
                  title="Focus"
                  description="Focus ties forms, tables, and shortcuts to one scenario so you can alternate between delegation and execution without retyping names."
                />
              </div>
              <p className="mt-2 text-sm text-slate-300">
                Set a focus once to prefill queue + runner forms and filter tables to the same scenario so you
                can bounce between delegation and execution without retyping names.
              </p>
            </div>
            {focusActive && (
              <Button variant="outline" size="sm" onClick={clearFocusScenario}>
                Clear focus
              </Button>
            )}
          </div>
          <div className="mt-6 grid gap-6 md:grid-cols-[1.5fr,1fr]">
            <label className="block text-sm">
              <span className="text-slate-300">Active scenario</span>
              <input
                className="mt-2 w-full rounded-xl border border-white/10 bg-black/30 px-4 py-3 text-base text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                placeholder="e.g. ecosystem-manager"
                value={focusScenario}
                onChange={(evt) => applyFocusScenario(evt.target.value)}
                list={scenarioOptionsDatalistId}
              />
            </label>
            <div>
              <p className="text-sm text-slate-300">Recent scenarios</p>
              <div className="mt-3 flex flex-wrap gap-2">
                {scenarioOptions.length === 0 && (
                  <span className="text-xs text-slate-400">Queue or run a suite to record history.</span>
                )}
                {scenarioOptions.slice(0, 5).map((name) => (
                  <button
                    type="button"
                    key={name}
                    onClick={() => applyFocusScenario(name)}
                    className={cn(
                      "rounded-full border px-4 py-2 text-sm transition",
                      focusScenario.toLowerCase() === name.toLowerCase()
                        ? "border-emerald-400 bg-emerald-400/20 text-white"
                        : "border-white/20 text-slate-300 hover:border-white/50"
                    )}
                  >
                    {name}
                  </button>
                ))}
              </div>
            </div>
          </div>
          {focusActive && focusQueueStats && (
            <div className="mt-6 grid gap-4 md:grid-cols-3">
              <article className="rounded-2xl border border-white/5 bg-black/30 p-4">
                <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Queue attention</p>
                <p className="mt-2 text-3xl font-semibold">
                  {focusQueueStats.actionableCount}/{focusQueueStats.totalCount}
                </p>
                <p className="mt-1 text-sm text-slate-300">
                  {focusQueueStats.actionableCount > 0
                    ? "Requests waiting to be executed inside the Go orchestrator."
                    : "No actionable queue entries for this scenario."}
                </p>
                {focusQueueStats.mostRecentRequest && (
                  <p className="mt-2 text-xs text-slate-500">
                    Last requested {formatRelative(focusQueueStats.mostRecentRequest.updatedAt ?? focusQueueStats.mostRecentRequest.createdAt)}
                  </p>
                )}
                <Button
                  size="sm"
                  className="mt-3"
                  variant="outline"
                  onClick={() => {
                    if (focusQueueStats.nextRequest) {
                      prefillFromRequest(focusQueueStats.nextRequest);
                    }
                    scrollToSection(SECTION_ANCHORS.queueForm);
                  }}
                >
                  {focusQueueStats.nextRequest ? "Prefill queue run" : "Open queue"}
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Button>
              </article>
              <article className="rounded-2xl border border-white/5 bg-black/30 p-4">
                <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Recent execution</p>
                <p className="mt-2 text-xl font-semibold">
                  {focusQueueStats.recentExecution
                    ? focusQueueStats.recentExecution.success
                      ? "Last run passed"
                      : "Last run failed"
                    : "No runs recorded"}
                </p>
                <p className="mt-1 text-sm text-slate-300">
                  {focusQueueStats.recentExecution
                    ? `Completed ${formatRelative(focusQueueStats.recentExecution.completedAt)}`
                    : "Trigger the Go runner to record history without bash wrappers."}
                </p>
                {focusQueueStats.failedExecution && focusQueueStats.failedExecution !== focusQueueStats.recentExecution && (
                  <p className="mt-2 text-xs text-amber-300">
                    Previous failure: {formatRelative(focusQueueStats.failedExecution.completedAt)}
                  </p>
                )}
                <Button
                  size="sm"
                  className="mt-3"
                  variant="outline"
                  onClick={() => {
                    if (focusQueueStats.recentExecution) {
                      prefillFromExecution(focusQueueStats.recentExecution);
                    }
                    scrollToSection(SECTION_ANCHORS.executionForm);
                  }}
                >
                  {focusQueueStats.recentExecution ? "Prefill rerun" : "Open runner"}
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Button>
              </article>
              <article className="rounded-2xl border border-white/5 bg-black/30 p-4">
                <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Most recent request</p>
                {focusQueueStats.mostRecentRequest ? (
                  <>
                    <p className="mt-2 text-xl font-semibold">
                      {focusQueueStats.mostRecentRequest.scenarioName}
                    </p>
                    <p className="mt-1 text-sm text-slate-300">
                      {focusQueueStats.mostRecentRequest.priority} priority ·{" "}
                      {focusQueueStats.mostRecentRequest.requestedTypes.join(", ")}
                    </p>
                    {focusQueueStats.mostRecentRequest.notes && (
                      <p className="mt-2 text-xs text-slate-400 truncate">
                        {focusQueueStats.mostRecentRequest.notes}
                      </p>
                    )}
                    <Button
                      size="sm"
                      className="mt-3"
                      variant="outline"
                      onClick={() => {
                        prefillFromRequest(focusQueueStats.mostRecentRequest!);
                        scrollToSection(SECTION_ANCHORS.queueForm);
                      }}
                    >
                      Prefill from request
                      <ArrowRight className="ml-2 h-4 w-4" />
                    </Button>
                  </>
                ) : (
                  <p className="mt-2 text-sm text-slate-300">
                    Queue a request or run an execution to capture intent for this scenario.
                  </p>
                )}
              </article>
            </div>
          )}
          <p className="mt-4 text-xs text-slate-400">
            {focusActive
              ? `Filtering surface data to ${focusScenario}. Clear focus to see everything.`
              : "Set a focus scenario to prefill both forms automatically; leave it blank to browse all queue and execution data."}
          </p>
          </section>
        )}

        {activeTab === "catalog" && (
          <>
            <section
              id={SECTION_ANCHORS.catalogOverview}
              className="rounded-2xl border border-white/10 bg-white/[0.03] p-6"
            >
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Scenario catalog</p>
                  <h2 className="mt-2 text-2xl font-semibold">Portfolio radar</h2>
                  <p className="mt-2 text-sm text-slate-300">
                    Use this tab when you need a bird&apos;s-eye view of every tracked scenario before deciding
                    where to focus, queue suites, or rerun presets.
                  </p>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => navigateToSection("operate", SECTION_ANCHORS.scenarioFocus)}
                  >
                    Go to focus
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => navigateToSection("operate", SECTION_ANCHORS.queueForm)}
                  >
                    Queue coverage
                  </Button>
                </div>
              </div>
              <div className="mt-6 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                <article className="rounded-2xl border border-white/5 bg-black/30 p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Tracked</p>
                  <p className="mt-2 text-3xl font-semibold">
                    {catalogStats.tracked.toString().padStart(2, "0")}
                  </p>
                  <p className="mt-1 text-sm text-slate-300">
                    {catalogStats.tracked > 0 ? "Scenarios with queue or execution history." : "Activity populates automatically after your first run."}
                  </p>
                </article>
                <article className="rounded-2xl border border-white/5 bg-black/30 p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Pending</p>
                  <p className="mt-2 text-3xl font-semibold">
                    {catalogStats.pending.toString().padStart(2, "0")}
                  </p>
                  <p className="mt-1 text-sm text-slate-300">Scenarios waiting for an inline run.</p>
                </article>
                <article className="rounded-2xl border border-white/5 bg-black/30 p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Failing</p>
                  <p className="mt-2 text-3xl font-semibold">
                    {catalogStats.failing.toString().padStart(2, "0")}
                  </p>
                  <p className="mt-1 text-sm text-slate-300">Scenarios whose last run failed inside the Go runner.</p>
                </article>
                <article className="rounded-2xl border border-white/5 bg-black/30 p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Idle</p>
                  <p className="mt-2 text-3xl font-semibold">
                    {catalogStats.idle.toString().padStart(2, "0")}
                  </p>
                  <p className="mt-1 text-sm text-slate-300">Tracked scenarios with no recorded run yet.</p>
                </article>
              </div>
            </section>

            <section
              id={SECTION_ANCHORS.scenarioDirectory}
              className="rounded-2xl border border-white/10 bg-white/[0.02] p-6"
            >
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Scenario Directory</p>
                  <div className="mt-2 flex items-center gap-2">
                    <h2 className="text-2xl font-semibold">Browse everything in scope</h2>
                    <InfoTip
                      title="Scenario directory"
                      description="Search and compare every scenario the API has seen, then jump directly into focus, queue, or rerun actions without leaving this page."
                    />
                  </div>
                  <p className="mt-2 text-sm text-slate-300">
                    Each card combines queue signals and the latest execution so you can decide whether to
                    focus, queue a fresh suite, or rerun a failure in seconds.
                  </p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <div className="relative">
                    <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
                    <input
                      className="w-64 rounded-full border border-white/10 bg-black/40 pl-9 pr-4 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                      placeholder="Search scenario names…"
                      value={scenarioSearch}
                      onChange={(evt) => setScenarioSearch(evt.target.value)}
                    />
                  </div>
                  {scenarioSearch && (
                    <Button variant="outline" size="sm" onClick={() => setScenarioSearch("")}>
                      Clear
                    </Button>
                  )}
                </div>
              </div>
              {scenarioDirectoryEntries.length === 0 && (
                <p className="mt-6 text-sm text-slate-400">
                  Queue a suite or run a preset to populate this directory automatically.
                </p>
              )}
              {scenarioDirectoryEntries.length > 0 && filteredScenarioDirectory.length === 0 && (
                <p className="mt-6 text-sm text-slate-400">
                  No scenarios match “{scenarioSearch}”. Clear the search to see all tracked workstreams.
                </p>
              )}
              {filteredScenarioDirectory.length > 0 && (
                <div className="mt-6 grid gap-4 lg:grid-cols-2">
                {filteredScenarioDirectory.map((entry) => {
                  const status =
                    entry.pendingRequests > 0
                      ? "queued"
                      : entry.lastExecutionSuccess === false
                        ? "failed"
                        : entry.lastExecutionSuccess === true
                        ? "completed"
                        : "idle";
                    const queueSummary =
                      entry.pendingRequests > 0
                        ? `${entry.pendingRequests} request${entry.pendingRequests === 1 ? "" : "s"} waiting for a run`
                        : entry.totalRequests > 0
                        ? "No pending requests"
                        : "Queue a request to start tracking";
                    const executionSummary = entry.lastExecutionAt
                      ? `${entry.lastExecutionSuccess ? "Passed" : "Failed"} ${formatRelative(entry.lastExecutionAt)}`
                      : "No runs recorded";
                    const requestDetail = entry.lastRequestPriority
                      ? `Last request ${entry.lastRequestAt ? formatRelative(entry.lastRequestAt) : "—"} · ${entry.lastRequestPriority} priority`
                      : "";
                  const scenarioStatus = entry.scenarioStatus || "unknown";
                  const tags = entry.scenarioTags?.slice(0, 4) ?? [];
                  return (
                    <article key={entry.scenarioName} className="rounded-2xl border border-white/5 bg-black/30 p-5">
                      <div className="flex items-start justify-between gap-3">
                        <div>
                          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">{queueSummary}</p>
                          <h3 className="mt-1 text-xl font-semibold">{entry.scenarioName}</h3>
                        </div>
                        <div className="flex flex-col items-end gap-2 text-right">
                          <StatusPill status={status} />
                          <div className="text-xs text-slate-400">Scenario: {scenarioStatus}</div>
                        </div>
                      </div>
                      {entry.scenarioDescription && (
                        <p className="mt-3 text-sm text-slate-400">{entry.scenarioDescription}</p>
                      )}
                      <p className="mt-3 text-sm text-slate-300">{executionSummary}</p>
                      {requestDetail && <p className="mt-1 text-xs text-slate-500">{requestDetail}</p>}
                      {tags.length > 0 && (
                        <div className="mt-3 flex flex-wrap gap-2">
                          {tags.map((tag) => (
                            <span key={`${entry.scenarioName}-${tag}`} className="rounded-full border border-white/10 px-3 py-1 text-xs uppercase tracking-wide text-slate-300">
                              {tag}
                            </span>
                          ))}
                        </div>
                      )}
                      <div className="mt-4 flex flex-wrap gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                            onClick={() => {
                              applyFocusScenario(entry.scenarioName);
                              navigateToSection("operate", SECTION_ANCHORS.scenarioFocus);
                            }}
                          >
                            Focus
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => {
                              prefillFromSummary(entry);
                              navigateToSection("operate", SECTION_ANCHORS.queueForm);
                            }}
                          >
                            Prefill queue
                          </Button>
                          <Button
                            size="sm"
                            onClick={() => {
                              prefillRunnerFromSummary(entry);
                              navigateToSection("operate", SECTION_ANCHORS.executionForm);
                            }}
                            disabled={!entry.lastExecutionAt}
                        >
                          Prefill rerun
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          className="border-transparent text-slate-200 hover:border-white/40"
                          onClick={() => {
                            applyFocusScenario(entry.scenarioName);
                            navigateToSection("signals", SECTION_ANCHORS.signalHighlights);
                          }}
                        >
                          View signals
                        </Button>
                      </div>
                    </article>
                  );
                })}
                </div>
              )}
            </section>

            <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6" id={SECTION_ANCHORS.scenarioCoverage}>
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Scenario coverage status</p>
                  <h2 className="mt-2 text-2xl font-semibold">All scenarios at a glance</h2>
                  <p className="mt-2 text-sm text-slate-300">
                    Review every scenario the orchestrator has seen, check pending queues, and jump straight into
                    focus, queue, rerun, or history views.
                  </p>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button variant="outline" size="sm" onClick={() => scrollToSection(SECTION_ANCHORS.scenarioDirectory)}>
                    Open directory
                  </Button>
                </div>
              </div>
              <div className="mt-4 overflow-x-auto rounded-2xl border border-white/5">
                <table className="w-full min-w-[600px] text-left text-sm text-slate-200">
                  <thead className="bg-white/5 text-xs uppercase tracking-wide text-slate-400">
                    <tr>
                      <th className="px-4 py-3">Scenario</th>
                      <th className="px-4 py-3">Status</th>
                      <th className="px-4 py-3">Pending</th>
                      <th className="px-4 py-3">Last run</th>
                      <th className="px-4 py-3">Last failure</th>
                      <th className="px-4 py-3 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {scenarioSummariesQuery.isLoading && scenarioStatusRows.length === 0 && (
                      <tr>
                        <td className="px-4 py-4 text-sm text-slate-400" colSpan={6}>
                          Loading scenario catalog…
                        </td>
                      </tr>
                    )}
                    {!scenarioSummariesQuery.isLoading && scenarioStatusRows.length === 0 && (
                      <tr>
                        <td className="px-4 py-4 text-sm text-slate-400" colSpan={6}>
                          No scenarios have queued or executed suites yet. Queue a request to populate this table.
                        </td>
                      </tr>
                    )}
                    {scenarioStatusRows.map((entry) => (
                      <tr key={`status-${entry.scenarioName}`} className="border-t border-white/5">
                        <td className="px-4 py-3">
                          <div className="font-semibold text-white">{entry.scenarioName}</div>
                          <p className="text-xs text-slate-400">
                            Last activity{" "}
                            {entry.lastExecutionAt
                              ? formatRelative(entry.lastExecutionAt)
                              : entry.lastRequestAt
                              ? formatRelative(entry.lastRequestAt)
                              : "—"}
                          </p>
                        </td>
                      <td className="px-4 py-3">
                        {entry.scenarioStatus ? (
                          <StatusPill status={entry.scenarioStatus} />
                        ) : (
                          <span className="text-xs text-slate-400">Unknown</span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {entry.pendingRequests > 0 ? (
                          <span className="rounded-full bg-amber-400/20 px-3 py-1 text-xs text-amber-100">
                            {entry.pendingRequests} pending
                          </span>
                        ) : (
                            <span className="text-xs text-slate-400">None</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-sm text-slate-300">
                          {entry.lastExecutionAt
                            ? `${entry.lastExecutionSuccess ? "Passed" : "Failed"} ${formatRelative(entry.lastExecutionAt)}`
                            : "No runs"}
                        </td>
                        <td className="px-4 py-3 text-sm text-slate-300">
                          {entry.lastFailureAt ? formatRelative(entry.lastFailureAt) : "—"}
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex flex-wrap justify-end gap-2">
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => {
                                applyFocusScenario(entry.scenarioName);
                                navigateToSection("operate", SECTION_ANCHORS.scenarioFocus);
                              }}
                            >
                              Focus
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => {
                                prefillFromSummary(entry);
                                navigateToSection("operate", SECTION_ANCHORS.queueForm);
                              }}
                            >
                              Queue
                            </Button>
                            <Button
                              size="sm"
                              disabled={!entry.lastExecutionAt}
                              onClick={() => {
                                prefillRunnerFromSummary(entry);
                                navigateToSection("operate", SECTION_ANCHORS.executionForm);
                              }}
                            >
                              Rerun
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => {
                                setHistoryScenario(entry.scenarioName);
                                setActiveTab("signals");
                              }}
                            >
                              History
                            </Button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          </>
        )}

        {activeTab === "signals" && (
          <div className="grid gap-6 lg:grid-cols-2">
            <section
              id={SECTION_ANCHORS.queueMetrics}
              className="rounded-2xl border border-white/10 bg-white/[0.03] p-6"
              data-testid={selectors.dashboard.queueMetrics}
            >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Queue Health</p>
                <h2 className="mt-2 text-2xl font-semibold">Suite backlog</h2>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => healthQuery.refetch()}
                disabled={healthQuery.isFetching}
              >
                Refresh
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </div>
            {healthQuery.isLoading && <p className="mt-4 text-sm text-slate-300">Loading queue state…</p>}
            {healthQuery.error && (
              <p className="mt-4 text-sm text-red-400">
                Unable to reach the API health endpoint. Ensure the scenario lifecycle is running.
              </p>
            )}
            {!healthQuery.isLoading && queueSnapshot && (
              <div className="mt-6 grid gap-4 sm:grid-cols-2">
                {queueMetrics.map((metric) => (
                  <div
                    key={metric.label}
                    className="rounded-xl border border-white/5 bg-black/20 p-4"
                  >
                    <p className="text-xs uppercase tracking-wide text-slate-400">{metric.label}</p>
                    <p className="mt-2 text-3xl font-semibold">{metric.value}</p>
                  </div>
                ))}
              </div>
            )}
            {queueSnapshot && (
              <p className="mt-6 text-sm text-slate-400">
                Oldest queued:{" "}
                {queueSnapshot.oldestQueuedAt
                  ? formatRelative(queueSnapshot.oldestQueuedAt)
                  : "No pending items"}
              </p>
            )}
            </section>

            <section
              id={SECTION_ANCHORS.latestExecution}
              className="rounded-2xl border border-white/10 bg-white/[0.03] p-6"
              data-testid={selectors.dashboard.lastExecution}
            >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Orchestrator</p>
                <h2 className="mt-2 text-2xl font-semibold">Latest execution</h2>
              </div>
              <Button variant="outline" size="sm" onClick={() => executionsQuery.refetch()}>
                Refresh
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </div>
            {!heroExecution && !executionsQuery.isLoading && (
              <p className="mt-6 text-sm text-slate-300">
                No executions have been recorded yet. Use the Run Suite controls to kick off the Go
                runner without touching bash scripts.
              </p>
            )}
              {heroExecution && (
                <ExecutionSummaryCard
                  execution={heroExecution}
                  onPrefill={() => {
                    prefillFromExecution(heroExecution);
                    navigateToSection("operate", SECTION_ANCHORS.executionForm);
                  }}
                  onFocus={() => {
                    applyFocusScenario(heroExecution.scenarioName);
                    navigateToSection("operate", SECTION_ANCHORS.scenarioFocus);
                  }}
                />
              )}
            </section>
          </div>
        )}


        {activeTab === "operate" && (
          <div className="grid gap-6 lg:grid-cols-2">
            <section
              id={SECTION_ANCHORS.queueForm}
              className="rounded-2xl border border-white/10 bg-white/[0.03] p-6"
              data-testid={selectors.dashboard.suiteRequestForm}
            >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.25em] text-slate-400">AI Delegation</p>
                <div className="mt-2 flex items-center gap-2">
                  <h2 className="text-2xl font-semibold">Queue a suite request</h2>
                  <InfoTip
                    title="Queue"
                    description="Tell Test Genie the scenario, coverage target, and desired test types. The API/CLI wiring happens behind the scenes."
                  />
                </div>
                <p className="mt-2 text-sm text-slate-400">
                  Describe the coverage outcome you need; Test Genie will either delegate to AI agents or ship deterministic templates immediately.
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setQueueForm(initialQueueForm)}
                disabled={queueMutation.isPending}
              >
                Reset
              </Button>
            </div>
            <form className="mt-6 space-y-4" onSubmit={handleQueueSubmit}>
              <label className="block text-sm">
                <span className="text-slate-300">Scenario name</span>
                <input
                  className="mt-1 w-full rounded-xl border border-white/10 bg-black/30 px-4 py-3 text-base text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                  value={queueForm.scenarioName}
                  onChange={(evt) => setQueueForm((prev) => ({ ...prev, scenarioName: evt.target.value }))}
                  placeholder="e.g. ecosystem-manager"
                  list={scenarioOptionsDatalistId}
                />
              </label>

              <label className="block text-sm">
                <span className="text-slate-300">Coverage target (%)</span>
                <input
                  type="number"
                  min={1}
                  max={100}
                  className="mt-1 w-full rounded-xl border border-white/10 bg-black/30 px-4 py-3 text-base text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                  value={queueForm.coverageTarget}
                  onChange={(evt) =>
                    setQueueForm((prev) => ({ ...prev, coverageTarget: Number(evt.target.value) }))
                  }
                />
              </label>

              <fieldset className="text-sm">
                <legend className="text-slate-300">Requested test types</legend>
                <div className="mt-3 flex flex-wrap gap-2">
                  {REQUESTED_TYPE_OPTIONS.map((type) => (
                    <button
                      type="button"
                      key={type}
                      onClick={() => toggleRequestedType(type)}
                      className={cn(
                        "rounded-full border px-4 py-2 text-sm transition",
                        queueForm.requestedTypes.includes(type)
                          ? "border-cyan-400 bg-cyan-400/20 text-white"
                          : "border-white/20 text-slate-300 hover:border-white/50"
                      )}
                    >
                      {type}
                    </button>
                  ))}
                </div>
              </fieldset>

              <label className="block text-sm">
                <span className="text-slate-300">Priority</span>
                <div className="mt-3 flex flex-wrap gap-2">
                  {PRIORITY_OPTIONS.map((priority) => (
                    <button
                      type="button"
                      key={priority}
                      onClick={() => setQueueForm((prev) => ({ ...prev, priority }))}
                      className={cn(
                        "rounded-full border px-4 py-2 text-sm capitalize transition",
                        queueForm.priority === priority
                          ? "border-purple-400 bg-purple-400/20 text-white"
                          : "border-white/20 text-slate-300 hover:border-white/50"
                      )}
                    >
                      {priority}
                    </button>
                  ))}
                </div>
              </label>

              <label className="block text-sm">
                <span className="text-slate-300">Notes (optional)</span>
                <textarea
                  className="mt-1 w-full rounded-xl border border-white/10 bg-black/30 px-4 py-3 text-base text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                  rows={4}
                  value={queueForm.notes}
                  onChange={(evt) => setQueueForm((prev) => ({ ...prev, notes: evt.target.value }))}
                  placeholder="Context for App Issue Tracker or deterministic fallback instructions"
                />
              </label>

              <Button
                className="w-full"
                type="submit"
                disabled={queueMutation.isPending}
                data-testid={selectors.actions.submitSuiteRequest}
              >
                {queueMutation.isPending ? "Queueing…" : "Queue suite request"}
                <ArrowUpRight className="ml-2 h-4 w-4" />
              </Button>
              {queueFeedback && (
                <p className={cn("text-sm", queueMutation.isError ? "text-red-400" : "text-emerald-300")}>
                  {queueFeedback}
                </p>
              )}
            </form>
            </section>

            <section
              id={SECTION_ANCHORS.executionForm}
              className="rounded-2xl border border-white/10 bg-white/[0.03] p-6"
              data-testid={selectors.dashboard.executionForm}
            >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Go Runner</p>
                <div className="mt-2 flex items-center gap-2">
                  <h2 className="text-2xl font-semibold">Run a suite inside the API</h2>
                  <InfoTip
                    title="Run"
                    description="Pick a preset to decide how deep the orchestrator goes (structure, integration, performance, etc.) without remembering bash-era scripts."
                  />
                </div>
                <p className="mt-2 text-sm text-slate-400">
                  Presets map to Go-native runners, so whatever you trigger here works the same on macOS, Linux, or Windows.
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setExecutionForm(initialExecutionForm)}
                disabled={executionMutation.isPending}
              >
                Reset
              </Button>
            </div>
            <form className="mt-6 space-y-4" onSubmit={handleExecutionSubmit}>
              <label className="block text-sm">
                <span className="text-slate-300">Scenario name</span>
                <input
                  className="mt-1 w-full rounded-xl border border-white/10 bg-black/30 px-4 py-3 text-base text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                  value={executionForm.scenarioName}
                  onChange={(evt) =>
                    setExecutionForm((prev) => ({ ...prev, scenarioName: evt.target.value }))
                  }
                  placeholder="scenario-under-test"
                  list={scenarioOptionsDatalistId}
                />
              </label>

              <label className="block text-sm">
                <span className="text-slate-300">Preset</span>
                <div className="mt-3 flex flex-wrap gap-2">
                  {EXECUTION_PRESETS.map((preset) => (
                    <button
                      type="button"
                      key={preset}
                      onClick={() => setExecutionForm((prev) => ({ ...prev, preset }))}
                      className={cn(
                        "rounded-full border px-4 py-2 text-sm capitalize transition",
                        executionForm.preset === preset
                          ? "border-emerald-400 bg-emerald-400/20 text-white"
                          : "border-white/20 text-slate-300 hover:border-white/50"
                      )}
                    >
                      {preset}
                    </button>
                  ))}
                </div>
              </label>

              <label className="inline-flex items-center gap-3 text-sm text-slate-300">
                <input
                  type="checkbox"
                  checked={executionForm.failFast}
                  onChange={(evt) =>
                    setExecutionForm((prev) => ({ ...prev, failFast: evt.target.checked }))
                  }
                  className="h-5 w-5 rounded border border-white/20 bg-black/30"
                />
                Fail fast on first failed phase
              </label>

              <label className="block text-sm">
                <span className="text-slate-300">Suite request (optional)</span>
                <input
                  className="mt-1 w-full rounded-xl border border-white/10 bg-black/30 px-4 py-3 text-base text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                  value={executionForm.suiteRequestId}
                  onChange={(evt) =>
                    setExecutionForm((prev) => ({ ...prev, suiteRequestId: evt.target.value }))
                  }
                  placeholder="Link to queue row for status updates"
                />
              </label>
              <div className="rounded-2xl border border-white/5 bg-black/30 p-4">
                <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                  <div>
                    <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Preset coverage map</p>
                    <p className="mt-1 text-sm text-slate-300">
                      Each preset runs Go-native phases that replace scripts/scenarios/testing/. Pick the coverage
                      depth you need and know exactly which runners will execute cross-platform.
                    </p>
                  </div>
                </div>
                <div className="mt-4 grid gap-3 lg:grid-cols-3">
                  {PRESET_ENTRIES.map(([presetKey, detail]) => (
                    <div
                      key={presetKey}
                      className={cn(
                        "rounded-xl border bg-white/5 p-3 text-sm",
                        executionForm.preset === presetKey
                          ? "border-emerald-400/60 bg-emerald-400/10"
                          : "border-white/10 bg-black/30"
                      )}
                    >
                      <p className="font-semibold capitalize">{detail.label}</p>
                      <p className="mt-1 text-xs text-slate-400">{detail.description}</p>
                      <ul className="mt-3 space-y-1 text-xs text-slate-300">
                        {detail.phases.map((phase) => (
                          <li key={`${presetKey}-${phase}`} className="flex items-center gap-2">
                            <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                            <span>{PHASE_LABELS[phase] ?? phase}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  ))}
                </div>
              </div>

              <Button
                className="w-full"
                type="submit"
                disabled={executionMutation.isPending}
                data-testid={selectors.actions.runSuiteExecution}
              >
                {executionMutation.isPending ? "Running…" : "Run suite"}
                <ArrowUpRight className="ml-2 h-4 w-4" />
              </Button>
              {executionFeedback && (
                <p
                  className={cn(
                    "text-sm",
                    executionMutation.isError ? "text-red-400" : executionMutation.isSuccess ? "text-emerald-300" : "text-slate-300"
                  )}
                >
                  {executionFeedback}
                </p>
              )}
            </form>
            </section>
          </div>
        )}

        {activeTab === "signals" && (
          <div className="grid gap-6 lg:grid-cols-2">
            <section
              id={SECTION_ANCHORS.suiteRequests}
              className="rounded-2xl border border-white/10 bg-white/[0.02] p-6"
              data-testid={selectors.dashboard.suiteRequestsTable}
            >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Queue</p>
                <h2 className="mt-2 text-2xl font-semibold">Recent suite requests</h2>
              </div>
              <span className="text-sm text-slate-400">
                {suiteRequestsQuery.isFetching
                  ? "Refreshing…"
                  : focusActive
                  ? `${filteredRequests.length}/${requests.length} match focus`
                  : `${requests.length} loaded`}
              </span>
            </div>
            <div className="mt-4 overflow-hidden rounded-2xl border border-white/5">
              <table className="w-full text-left text-sm text-slate-200">
                <thead className="bg-white/5 text-xs uppercase tracking-wide text-slate-400">
                  <tr>
                    <th className="px-4 py-3">Scenario</th>
                    <th className="px-4 py-3">Types</th>
                    <th className="px-4 py-3">Priority</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredRequests.slice(0, 6).map((request) => {
                    const canRunInline = isActionableRequest(request.status);
                    const isActiveRow = activeQuickRunId === request.id && quickRunMutation.isPending;
                    return (
                    <tr key={request.id} className="border-t border-white/5">
                      <td className="px-4 py-3">
                        <div className="font-medium">{request.scenarioName}</div>
                        <p className="text-xs text-slate-400">{formatRelative(request.createdAt)}</p>
                      </td>
                      <td className="px-4 py-3 text-xs uppercase text-slate-400">
                        {request.requestedTypes.join(", ")}
                      </td>
                      <td className="px-4 py-3">
                        <span className="rounded-full bg-white/10 px-3 py-1 text-xs capitalize">
                          {request.priority}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <StatusPill status={request.status} />
                      </td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex justify-end gap-2">
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            onClick={() => {
                              prefillFromRequest(request);
                              navigateToSection("operate", SECTION_ANCHORS.queueForm);
                            }}
                          >
                            Prefill
                          </Button>
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            className="border-transparent text-slate-200 hover:border-white/40"
                            onClick={() => {
                              applyFocusScenario(request.scenarioName);
                              navigateToSection("operate", SECTION_ANCHORS.scenarioFocus);
                            }}
                            data-testid={selectors.actions.focusScenarioFromQueue}
                          >
                            Focus
                          </Button>
                          <Button
                            type="button"
                            size="sm"
                            disabled={!canRunInline || isActiveRow}
                            onClick={() => handleQuickRun(request)}
                            data-testid={selectors.actions.runSuiteFromQueue}
                          >
                            {isActiveRow ? "Running…" : canRunInline ? "Run inline" : "Locked"}
                          </Button>
                        </div>
                      </td>
                    </tr>
                  )})}
                  {filteredRequests.length === 0 && (
                    <tr>
                      <td className="px-4 py-6 text-center text-slate-400" colSpan={5}>
                        {focusActive
                          ? `No suite requests match “${focusScenario}”. Clear focus to view all requests.`
                          : "No suite requests have been queued yet."}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
            </section>

            <section
              id={SECTION_ANCHORS.executionHistory}
              className="rounded-2xl border border-white/10 bg-white/[0.02] p-6"
              data-testid={selectors.dashboard.executionHistory}
            >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Executions</p>
                <h2 className="mt-2 text-2xl font-semibold">Recent runs</h2>
              </div>
              <span className="text-sm text-slate-400">
                {executionsQuery.isFetching
                  ? "Refreshing…"
                  : focusActive
                  ? `${filteredExecutions.length}/${executions.length} match focus`
                  : `${executions.length} loaded`}
              </span>
            </div>
            <div className="mt-4 space-y-4">
              {filteredExecutions.length === 0 && !executionsQuery.isLoading && (
                <p className="text-sm text-slate-400">
                  {focusActive
                    ? `The Go orchestrator has no executions for “${focusScenario}”. Clear focus to view the full history.`
                    : "The Go orchestrator has not executed any suites yet. Trigger a run and the history will populate automatically."}
                </p>
              )}
              {filteredExecutions.slice(0, 5).map((execution) => (
                <article key={execution.executionId} className="rounded-2xl border border-white/5 p-4">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <p className="text-xs uppercase tracking-wide text-slate-400">
                        {execution.preset ? `${execution.preset} preset` : "Custom phases"}
                      </p>
                      <h3 className="text-xl font-semibold">
                        {execution.scenarioName} · {execution.phaseSummary.passed}/{execution.phaseSummary.total} phases
                      </h3>
                    </div>
                    <StatusPill status={execution.success ? "completed" : "failed"} />
                  </div>
                  <div className="mt-3 flex flex-wrap gap-6 text-sm text-slate-300">
                    <span>Completed {formatRelative(execution.completedAt)}</span>
                    <span>Duration {formatDuration(execution.phaseSummary.durationSeconds)}</span>
                    {execution.phaseSummary.observationCount > 0 && (
                      <span>{execution.phaseSummary.observationCount} observation(s)</span>
                    )}
                  </div>
                  {execution.phases.length > 0 && (
                    <div className="mt-3 grid gap-2 md:grid-cols-2">
                      {execution.phases.slice(0, 4).map((phase) => (
                        <div
                          key={`${execution.executionId}-${phase.name}`}
                          className="rounded-xl border border-white/5 bg-black/20 p-3 text-sm"
                        >
                          <div className="flex items-center justify-between">
                            <span className="font-medium capitalize">{phase.name}</span>
                            <span
                              className={cn(
                                "rounded-full px-2 py-0.5 text-xs uppercase",
                                phase.status === "passed"
                                  ? "bg-emerald-500/20 text-emerald-300"
                                  : "bg-red-500/20 text-red-300"
                              )}
                            >
                              {phase.status}
                            </span>
                          </div>
                          <p className="mt-1 text-xs text-slate-400">
                            {formatDuration(phase.durationSeconds)} ·{" "}
                            {phase.observations?.length ?? 0} observations
                          </p>
                          {phase.error && (
                            <p className="mt-1 text-xs text-red-300">
                              {phase.error.slice(0, 120)}
                              {phase.error.length > 120 ? "…" : ""}
                            </p>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                  <div className="mt-4 flex flex-wrap gap-2">
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        prefillFromExecution(execution);
                        navigateToSection("operate", SECTION_ANCHORS.executionForm);
                      }}
                      data-testid={selectors.actions.prefillExecutionFromHistory}
                    >
                      Prefill rerun
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      className="border-transparent text-slate-200 hover:border-white/40"
                      onClick={() => {
                        applyFocusScenario(execution.scenarioName);
                        navigateToSection("operate", SECTION_ANCHORS.scenarioFocus);
                      }}
                      data-testid={selectors.actions.focusExecutionFromHistory}
                    >
                      Focus scenario
                    </Button>
                  </div>
                </article>
              ))}
            </div>
            </section>
          </div>
        )}

        {activeTab === "signals" && historyScenario && (
          <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <p className="text-xs uppercase tracking-[0.25em] text-slate-400">History</p>
                <h2 className="mt-2 text-2xl font-semibold">Recent runs for {historyScenario}</h2>
                <p className="mt-2 text-sm text-slate-300">
                  {selectedHistorySummary
                    ? `${selectedHistorySummary.pendingRequests} pending / ${selectedHistorySummary.totalExecutions} recorded`
                    : "Loading scenario details…"}
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setHistoryScenario(null)}
                >
                  Close
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => navigateToSection("operate", SECTION_ANCHORS.executionForm)}
                >
                  Go to runner
                </Button>
              </div>
            </div>
            {scenarioHistoryQuery.isLoading && (
              <p className="mt-4 text-sm text-slate-400">Loading execution history…</p>
            )}
            {!scenarioHistoryQuery.isLoading && historyExecutions.length === 0 && (
              <p className="mt-4 text-sm text-slate-400">
                No executions recorded for {historyScenario}. Run a preset to establish a baseline.
              </p>
            )}
            {historyExecutions.length > 0 && (
              <div className="mt-4 space-y-4">
                {historyExecutions.map((execution) => (
                  <article key={`history-${execution.executionId}`} className="rounded-2xl border border-white/5 p-4">
                    <div className="flex flex-wrap items-center justify-between gap-3">
                      <div>
                        <p className="text-xs uppercase tracking-wide text-slate-400">
                          {execution.preset ? `${execution.preset} preset` : "Custom phases"}
                        </p>
                        <h3 className="text-lg font-semibold text-white">
                          {execution.phaseSummary.passed}/{execution.phaseSummary.total} phases ·{" "}
                          {formatRelative(execution.completedAt)}
                        </h3>
                      </div>
                      <StatusPill status={execution.success ? "completed" : "failed"} />
                    </div>
                    <div className="mt-2 flex flex-wrap gap-4 text-xs text-slate-400">
                      <span>Duration {formatDuration(execution.phaseSummary.durationSeconds)}</span>
                      {execution.phaseSummary.observationCount > 0 && (
                        <span>{execution.phaseSummary.observationCount} observation(s)</span>
                      )}
                    </div>
                    <div className="mt-3 flex flex-wrap gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          prefillFromExecution(execution);
                          navigateToSection("operate", SECTION_ANCHORS.executionForm);
                        }}
                      >
                        Prefill rerun
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          applyFocusScenario(execution.scenarioName);
                          setHistoryScenario(execution.scenarioName);
                        }}
                      >
                        Focus scenario
                      </Button>
                    </div>
                  </article>
                ))}
              </div>
            )}
          </section>
        )}
      </div>
    </div>
  );
}

function StatusPill({ status }: { status: string }) {
  const normalized = status.toLowerCase();
  const style =
    normalized === "completed" || normalized === "passed"
      ? "bg-emerald-500/20 text-emerald-300"
      : normalized === "running"
      ? "bg-cyan-500/20 text-cyan-200"
      : normalized === "queued" || normalized === "delegated"
      ? "bg-amber-400/20 text-amber-200"
      : normalized === "idle"
      ? "bg-white/10 text-slate-200"
      : "bg-red-500/20 text-red-200";
  return (
    <span className={cn("rounded-full px-3 py-1 text-xs uppercase tracking-wide", style)}>
      {status}
    </span>
  );
}

function ExecutionSummaryCard({
  execution,
  onPrefill,
  onFocus
}: {
  execution: SuiteExecutionResult;
  onPrefill?: () => void;
  onFocus?: () => void;
}) {
  return (
    <div className="mt-6 rounded-2xl border border-white/5 bg-black/20 p-5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-wide text-slate-400">Scenario</p>
          <h3 className="text-xl font-semibold">{execution.scenarioName}</h3>
        </div>
        <StatusPill status={execution.success ? "completed" : "failed"} />
      </div>
      <p className="mt-2 text-sm text-slate-400">
        {execution.preset ? `${execution.preset} preset · ` : ""}
        Completed {formatRelative(execution.completedAt)}
      </p>
      <div className="mt-4 grid gap-3 sm:grid-cols-2">
        <div className="rounded-xl border border-white/5 p-3 text-sm">
          <p className="text-xs uppercase tracking-wide text-slate-400">Phase summary</p>
          <p className="mt-2 text-base text-white">
            {execution.phaseSummary.passed}/{execution.phaseSummary.total} passed
          </p>
        </div>
        <div className="rounded-xl border border-white/5 p-3 text-sm">
          <p className="text-xs uppercase tracking-wide text-slate-400">Duration</p>
          <p className="mt-2 text-base text-white">
            {formatDuration(execution.phaseSummary.durationSeconds)}
          </p>
        </div>
      </div>
      {(onPrefill || onFocus) && (
        <div className="mt-4 flex flex-wrap gap-2">
          {onPrefill && (
            <Button variant="outline" size="sm" onClick={onPrefill}>
              Prefill rerun
            </Button>
          )}
          {onFocus && (
            <Button
              variant="outline"
              size="sm"
              className="border-transparent text-slate-200 hover:border-white/40"
              onClick={onFocus}
            >
              Focus scenario
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

function InfoTip({ title, description }: { title: string; description: string }) {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <div
      className="relative inline-flex"
      onMouseLeave={() => setIsOpen(false)}
    >
      <button
        type="button"
        className="flex h-7 w-7 items-center justify-center rounded-full border border-white/20 text-slate-300 transition hover:border-cyan-400 hover:text-white"
        onMouseEnter={() => setIsOpen(true)}
        onFocus={() => setIsOpen(true)}
        onBlur={() => setIsOpen(false)}
        onClick={() => setIsOpen((prev) => !prev)}
        aria-expanded={isOpen}
        aria-label={`More about ${title}`}
      >
        <Info className="h-4 w-4" />
      </button>
      <div
        className={cn(
          "absolute right-0 top-9 z-20 w-64 rounded-2xl border border-white/10 bg-slate-900/95 p-4 text-left text-xs text-slate-200 shadow-2xl transition",
          isOpen ? "pointer-events-auto opacity-100" : "pointer-events-none opacity-0"
        )}
        role="status"
      >
        <p className="font-semibold text-white">{title}</p>
        <p className="mt-1 leading-relaxed">{description}</p>
      </div>
    </div>
  );
}

function clampCoverage(value: number) {
  if (Number.isNaN(value)) {
    return 95;
  }
  return Math.min(100, Math.max(1, Math.round(value)));
}

function formatRelative(timestamp?: string) {
  if (!timestamp) {
    return "—";
  }
  const target = new Date(timestamp).getTime();
  const now = Date.now();
  const diffMs = target - now;
  const diffMinutes = Math.round(diffMs / 60000);
  if (Math.abs(diffMinutes) < 60) {
    return relativeFormatter.format(diffMinutes, "minute");
  }
  const diffHours = Math.round(diffMinutes / 60);
  if (Math.abs(diffHours) < 24) {
    return relativeFormatter.format(diffHours, "hour");
  }
  const diffDays = Math.round(diffHours / 24);
  return relativeFormatter.format(diffDays, "day");
}

function formatDuration(seconds?: number) {
  if (!seconds || seconds <= 0) {
    return "0s";
  }
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = seconds / 60;
  if (minutes < 60) {
    return `${minutes.toFixed(1)}m`;
  }
  const hours = minutes / 60;
  return `${hours.toFixed(1)}h`;
}

function parseTimestamp(value?: string) {
  if (!value) {
    return Number.POSITIVE_INFINITY;
  }
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? parsed : Number.POSITIVE_INFINITY;
}

function timestampOrZero(value?: string) {
  if (!value) {
    return 0;
  }
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function priorityWeight(priority: string) {
  const normalized = priority.toLowerCase();
  if (normalized === "urgent") {
    return 3;
  }
  if (normalized === "high") {
    return 2;
  }
  if (normalized === "normal") {
    return 1;
  }
  return 0;
}

function presetFromPriority(priority: string) {
  const normalized = priority.toLowerCase();
  if (normalized === "urgent") {
    return "comprehensive";
  }
  if (normalized === "high") {
    return "smoke";
  }
  return "quick";
}

function isActionableRequest(status: string) {
  const normalized = status.toLowerCase();
  return normalized === "queued" || normalized === "delegated";
}

function selectActionableRequest(requests: SuiteRequest[]) {
  const actionable = requests.filter((req) => isActionableRequest(req.status));
  if (actionable.length === 0) {
    return null;
  }
  let best = actionable[0];
  for (let idx = 1; idx < actionable.length; idx += 1) {
    const candidate = actionable[idx];
    const priorityDiff = priorityWeight(candidate.priority) - priorityWeight(best.priority);
    if (priorityDiff > 0) {
      best = candidate;
      continue;
    }
    if (priorityDiff < 0) {
      continue;
    }
    const candidateTimestamp = parseTimestamp(candidate.updatedAt ?? candidate.createdAt);
    const bestTimestamp = parseTimestamp(best.updatedAt ?? best.createdAt);
    if (candidateTimestamp < bestTimestamp) {
      best = candidate;
    }
  }
  return best;
}
