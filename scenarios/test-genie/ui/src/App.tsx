import { useId, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowRight, ArrowUpRight } from "lucide-react";
import { selectors } from "./consts/selectors";
import { Button } from "./components/ui/button";
import {
  fetchExecutionHistory,
  fetchHealth,
  fetchSuiteRequests,
  queueSuiteRequest,
  triggerSuiteExecution,
  type ApiHealthResponse,
  type SuiteExecutionResult,
  type SuiteRequest
} from "./lib/api";
import { cn } from "./lib/utils";

const REQUESTED_TYPE_OPTIONS = ["unit", "integration", "performance", "vault", "regression"] as const;
const PRIORITY_OPTIONS = ["low", "normal", "high", "urgent"] as const;
const EXECUTION_PRESETS = ["quick", "smoke", "comprehensive"] as const;
const DEFAULT_REQUEST_TYPES = ["unit", "integration"];

const relativeFormatter = new Intl.RelativeTimeFormat("en", { numeric: "auto" });

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
  const [activeQuickRunId, setActiveQuickRunId] = useState<string | null>(null);

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
    requests.forEach((req) => register(collected, req.scenarioName, req.updatedAt ?? req.createdAt));
    executions.forEach((execution) => register(collected, execution.scenarioName, execution.completedAt ?? execution.startedAt));
    register(collected, queueForm.scenarioName);
    register(collected, executionForm.scenarioName);

    return Array.from(collected.values())
      .sort((a, b) => b.timestamp - a.timestamp)
      .map((entry) => entry.label);
  }, [executionForm.scenarioName, executions, queueForm.scenarioName, requests]);

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

  const actionableRequest = useMemo(() => {
    const actionableStatuses = new Set(["queued", "delegated"]);
    const candidates = requests.filter((req) => actionableStatuses.has(req.status.toLowerCase()));
    if (candidates.length === 0) {
      return null;
    }
    return candidates.reduce<SuiteRequest | null>((best, candidate) => {
      if (!best) {
        return candidate;
      }
      const priorityDiff = priorityWeight(candidate.priority) - priorityWeight(best.priority);
      if (priorityDiff > 0) {
        return candidate;
      }
      if (priorityDiff < 0) {
        return best;
      }
      const candidateTimestamp = parseTimestamp(candidate.updatedAt ?? candidate.createdAt);
      const bestTimestamp = parseTimestamp(best.updatedAt ?? best.createdAt);
      return candidateTimestamp < bestTimestamp ? candidate : best;
    }, null);
  }, [requests]);

  const lastFailedExecution = useMemo(
    () => executions.find((execution) => execution.success === false),
    [executions]
  );

  const handleQueueSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
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

  const handleExecutionSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
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
          <h1 className="mt-3 text-3xl font-semibold">Scenario-native test orchestration</h1>
          <p className="mt-4 text-base text-slate-300">
            Queue AI-generated suites, run orchestration inside the Go API, and spot backlog risks
            without leaving the dashboard. Every control on this screen maps to the native API so we
            can retire the bash runner and keep flows cross-platform.
          </p>
        </header>

        <section
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
                      onClick={() => prefillFromRequest(actionableRequest)}
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
                      onClick={() => prefillFromExecution(lastFailedExecution)}
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

        <section
          className="rounded-2xl border border-white/10 bg-white/[0.03] p-6"
          data-testid={selectors.dashboard.scenarioFocus}
        >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Scenario Focus</p>
              <h2 className="mt-2 text-2xl font-semibold">Work on a single scenario faster</h2>
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
          <p className="mt-4 text-xs text-slate-400">
            {focusActive
              ? `Filtering surface data to ${focusScenario}. Clear focus to see everything.`
              : "Focus is optional—leave blank to view all queued requests and executions."}
          </p>
        </section>

        <div className="grid gap-6 lg:grid-cols-2">
          <section
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
            {heroExecution && <ExecutionSummaryCard execution={heroExecution} />}
          </section>
        </div>

        <div className="grid gap-6 lg:grid-cols-2">
          <section
            className="rounded-2xl border border-white/10 bg-white/[0.03] p-6"
            data-testid={selectors.dashboard.suiteRequestForm}
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.25em] text-slate-400">AI Delegation</p>
                <h2 className="mt-2 text-2xl font-semibold">Queue a suite request</h2>
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
            className="rounded-2xl border border-white/10 bg-white/[0.03] p-6"
            data-testid={selectors.dashboard.executionForm}
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Go Runner</p>
                <h2 className="mt-2 text-2xl font-semibold">Run a suite inside the API</h2>
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

        <div className="grid gap-6 lg:grid-cols-2">
          <section
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
                    const canRunInline = ["queued", "delegated"].includes(request.status.toLowerCase());
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
                            onClick={() => prefillFromRequest(request)}
                          >
                            Prefill
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
                </article>
              ))}
            </div>
          </section>
        </div>
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
      : "bg-red-500/20 text-red-200";
  return (
    <span className={cn("rounded-full px-3 py-1 text-xs uppercase tracking-wide", style)}>
      {status}
    </span>
  );
}

function ExecutionSummaryCard({ execution }: { execution: SuiteExecutionResult }) {
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
