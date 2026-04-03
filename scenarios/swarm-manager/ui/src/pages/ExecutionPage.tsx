import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { X } from "lucide-react";
import { Card } from "../components/ui/card";
import { InlineLoadingIndicator } from "../components/ui/loading-states";
import { ExecutionFilters } from "../components/execution/ExecutionFilters";
import { ExecutionListView } from "../components/execution/ExecutionListView";
import { ExecutionToolbar } from "../components/execution/ExecutionToolbar";
import { FollowUpSheet } from "../components/review/follow-up-sheet";
import { selectors } from "../consts/selectors";
import {
  EXECUTION_TAB_CONFIG,
  isExecutionActive,
  isExecutionInTab,
  matchesExecutionFilters,
  type ExecutionTabId,
} from "../lib";
import { executionService, promptService } from "../services";
import { gctService } from "../services/gct-service";
import { useStorePolling } from "../hooks/useStorePolling";
import { useExecutionStore } from "../stores";
import type { ExecutionMode, ExecutionRecord, ExecutionStatus, PromptTrace } from "../types";

const AUTO_REFRESH_MS = 6000;

type AgentManagerRunLookup = { run?: { sandboxId?: string } };

function parseAgentManagerRunLookup(value: unknown): AgentManagerRunLookup | null {
  if (typeof value !== "object" || value === null) return null;
  const candidate = value as { run?: unknown };
  if (typeof candidate.run !== "object" || candidate.run === null) return {};
  const run = candidate.run as { sandboxId?: unknown };
  return { run: typeof run.sandboxId === "string" ? { sandboxId: run.sandboxId } : {} };
}

export function ExecutionPage() {
  const { items, status, error, isRefreshing, fetchExecutions, upsertExecution } = useExecutionStore();
  const navigate = useNavigate();

  // --- Filter state ---
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

  const [searchParams, setSearchParams] = useSearchParams();
  const appliedBacklogParam = useRef(false);

  useEffect(() => {
    if (appliedBacklogParam.current) return;
    const backlogParam = searchParams.get("backlog");
    if (backlogParam) {
      setBacklogFilter(backlogParam);
      setShowFilters(true);
      setActiveTab("all");
      appliedBacklogParam.current = true;
      searchParams.delete("backlog");
      setSearchParams(searchParams, { replace: true });
    }
  }, [searchParams, setSearchParams]);

  // --- Action state ---
  const [busyId, setBusyId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [traceByExecutionId, setTraceByExecutionId] = useState<Record<string, PromptTrace>>({});
  const [traceLoadingId, setTraceLoadingId] = useState<string | null>(null);
  const [agentManagerUiUrl, setAgentManagerUiUrl] = useState<string | null>(null);
  const [followUpTarget, setFollowUpTarget] = useState<ExecutionRecord | null>(null);
  const [workspaceSandboxBaseUrl, setWorkspaceSandboxBaseUrl] = useState<string | null>(null);
  const [gctAvailable, setGctAvailable] = useState<boolean | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetch(`/embedded/${encodeURIComponent("agent-manager")}/external-url`)
      .then((res) => (res.ok ? res.json() : null))
      .then((data: { url?: string } | null) => { if (!cancelled && data?.url) setAgentManagerUiUrl(data.url); })
      .catch(() => {});
    fetch(`/embedded/${encodeURIComponent("workspace-sandbox")}/external-url`)
      .then((res) => (res.ok ? res.json() : null))
      .then((data: { url?: string } | null) => { if (!cancelled && data?.url) setWorkspaceSandboxBaseUrl(data.url); })
      .catch(() => {});
    return () => { cancelled = true; };
  }, []);

  useStorePolling({ enabled: true, intervalMs: AUTO_REFRESH_MS, pollFn: () => void fetchExecutions({ force: true }), immediate: true });
  useStorePolling({ enabled: true, intervalMs: 30_000, pollFn: () => void gctService.getStatus().then((s) => setGctAvailable(s.available)), immediate: true });

  // --- Derived data ---
  const activeTabConfig = EXECUTION_TAB_CONFIG.find((tab) => tab.id === activeTab) ?? EXECUTION_TAB_CONFIG[0];
  const hasLoaded = status !== "idle";

  const tabItems = useMemo(() => items.filter((item) => isExecutionInTab(item, activeTab)), [items, activeTab]);

  const filteredItems = useMemo(() => {
    return tabItems.filter((item) =>
      matchesExecutionFilters(item, { searchTerm, statusFilter, modeFilter, operationFilter, startedByFilter, backlogFilter, fromFilter, toFilter })
    );
  }, [tabItems, searchTerm, statusFilter, modeFilter, operationFilter, startedByFilter, backlogFilter, fromFilter, toFilter]);

  const activeRuns = useMemo(() => filteredItems.filter(isExecutionActive).slice(0, 3), [filteredItems]);

  const stats = useMemo(() => {
    const running = tabItems.filter((i) => i.status === "starting" || i.status === "running").length;
    const validating = tabItems.filter((i) => i.status === "validating").length;
    const review = tabItems.filter((i) => i.status === "needs_review").length;
    const failed = tabItems.filter((i) => i.status === "failed" || i.status === "canceled").length;
    return { total: tabItems.length, running, validating, review, failed };
  }, [tabItems]);

  const activeFilterCount = [searchTerm, statusFilter, modeFilter, operationFilter, startedByFilter, backlogFilter, fromFilter, toFilter]
    .filter((v) => v.trim().length > 0).length;

  const clearFilters = () => {
    setSearchTerm(""); setStatusFilter(""); setModeFilter(""); setOperationFilter("");
    setStartedByFilter(""); setBacklogFilter(""); setFromFilter(""); setToFilter("");
    setShowFilters(false);
  };

  // --- Action handlers ---
  const runAction = async (executionId: string, action: "start" | "cancel" | "retry") => {
    setBusyId(executionId); setActionError(null);
    try {
      let updated: ExecutionRecord;
      if (action === "start") updated = await executionService.start(executionId);
      else if (action === "cancel") updated = await executionService.cancel(executionId);
      else updated = await executionService.retry(executionId);
      upsertExecution(updated);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : `Failed to ${action} execution.`);
    } finally { setBusyId(null); }
  };

  const handleViewTrace = async (executionId: string) => {
    setTraceLoadingId(executionId); setActionError(null);
    try {
      const trace = await promptService.getExecutionPromptTrace(executionId);
      setTraceByExecutionId((prev) => ({ ...prev, [executionId]: trace }));
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed to load prompt trace.");
    } finally { setTraceLoadingId(null); }
  };

  const handleFollowUp = (executionId: string) => {
    const exec = items.find((i) => i.executionId === executionId);
    if (exec) setFollowUpTarget(exec);
  };

  const handleTriggerReview = async (executionId: string) => {
    setBusyId(executionId); setActionError(null);
    try {
      const updated = await executionService.triggerReview(executionId);
      upsertExecution(updated);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed to run post-run checks.");
    } finally { setBusyId(null); }
  };

  const handleOpenReviewSandbox = async (executionId: string) => {
    const exec = items.find((i) => i.executionId === executionId);
    if (!exec?.runId) return;
    const baseUrl = workspaceSandboxBaseUrl ?? `/embedded/workspace-sandbox/`;
    try {
      const runResp = await fetch(`/api/v1/agent-manager/runs/${encodeURIComponent(exec.runId)}`);
      const rawRunData: unknown = runResp.ok ? await runResp.json() : null;
      const runData = parseAgentManagerRunLookup(rawRunData);
      const sandboxId = runData?.run?.sandboxId;
      if (sandboxId) window.open(`${baseUrl.replace(/\/$/, "")}?sandbox=${sandboxId}&review=true`, "_blank");
      else window.open(baseUrl, "_blank");
    } catch { window.open(baseUrl, "_blank"); }
  };

  if (!activeTabConfig) return null;

  return (
    <div className="space-y-6" data-testid={selectors.execution.page}>
      <ExecutionToolbar
        items={items}
        activeTab={activeTab}
        onActiveTabChange={setActiveTab}
        status={status}
        isRefreshing={isRefreshing}
        onRefresh={() => void fetchExecutions({ force: true })}
        searchTerm={searchTerm}
        onSearchChange={(e) => setSearchTerm(e.target.value)}
        showFilters={showFilters}
        onToggleFilters={() => setShowFilters(!showFilters)}
        activeFilterCount={activeFilterCount}
        statusFilter={statusFilter}
        modeFilter={modeFilter}
        operationFilter={operationFilter}
        startedByFilter={startedByFilter}
        backlogFilter={backlogFilter}
        fromFilter={fromFilter}
        toFilter={toFilter}
        onStatusFilterChange={setStatusFilter}
        onModeFilterChange={setModeFilter}
        onOperationFilterChange={setOperationFilter}
        onStartedByFilterChange={setStartedByFilter}
        onBacklogFilterChange={setBacklogFilter}
        onFromFilterChange={setFromFilter}
        onToFilterChange={setToFilter}
        onClearFilters={clearFilters}
        stats={stats}
        gctAvailable={gctAvailable}
      />

      {searchTerm ? (
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <span>Showing results for <span className="text-slate-200">"{searchTerm}"</span></span>
          <button onClick={() => setSearchTerm("")} className="text-slate-400 hover:text-slate-200" aria-label="Clear search">
            <X className="h-4 w-4" />
          </button>
        </div>
      ) : null}

      {actionError ? (
        <Card className="border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-sm text-amber-200">{actionError}</Card>
      ) : null}

      {isRefreshing && items.length > 0 ? (
        <InlineLoadingIndicator label="Refreshing execution runs..." testId="execution-refreshing-indicator" />
      ) : null}

      <ExecutionListView
        status={status}
        hasLoaded={hasLoaded}
        error={error}
        tabItems={tabItems}
        filteredItems={filteredItems}
        activeRuns={activeRuns}
        activeTabConfig={activeTabConfig}
        busyId={busyId}
        traceByExecutionId={traceByExecutionId}
        traceLoadingId={traceLoadingId}
        agentManagerUiUrl={agentManagerUiUrl}
        onStart={(id) => void runAction(id, "start")}
        onCancel={(id) => void runAction(id, "cancel")}
        onRetry={(id) => void runAction(id, "retry")}
        onViewTrace={(id) => void handleViewTrace(id)}
        onViewBacklog={(kind, name) => navigate(`/backlog/${kind}/${name}`)}
        onFollowUp={handleFollowUp}
        onOpenReviewSandbox={(id) => void handleOpenReviewSandbox(id)}
        onTriggerReview={(id) => void handleTriggerReview(id)}
        onFetchRetry={() => fetchExecutions({ force: true })}
        onClearFilters={clearFilters}
      />

      {followUpTarget && (
        <FollowUpSheet
          isOpen={Boolean(followUpTarget)}
          onClose={() => setFollowUpTarget(null)}
          execution={followUpTarget}
          reviewRounds={[]}
          onSuccess={(newExec) => { upsertExecution(newExec); setFollowUpTarget(null); }}
        />
      )}
    </div>
  );
}
