import { useState, useEffect, useCallback, useMemo } from "react";
import { createPortal } from "react-dom";
import { ClipboardCheck, RefreshCw, Loader2, Play, CheckCircle2, XCircle, AlertTriangle, ChevronDown, ChevronRight, ChevronLeft, Plus, Minus, X, Anchor, Camera, ExternalLink, AlertCircle, Settings, Copy, Check } from "lucide-react";
import { Button } from "./ui/button";
import { Card, CardHeader, CardTitle } from "./ui/card";
import { useVisualCaptures, useTriggerVisualCapture, useCapabilities, useTestExecutions, useTriggerTestExecution, useTidinessScore, useTidinessIssues, useTidinessStaleness, useTriggerTidinessScan, useWorkflowCaptures, useTriggerWorkflowCapture, useScenarios, useStartAuditorCheck, useAuditorJobStatus } from "../lib/hooks";
import { buildCaptureScreenshotUrl, buildCaptureVideoUrl, buildWorkflowVideoUrl, presetSuffix, presetLabel, presetKey, getCapturePresets, setCapturePresets as saveCapturePresets, SIZE_PRESETS, DEFAULT_PRESETS } from "../lib/api";
import type { CapturePreset, CaptureTheme, SnapshotSetMeta, SnapshotStalenessInfo, TestExecutionResult, TestPhaseResult, RepoFileStats, TidinessIssue, TidinessLightScanResult, TidinessStalenessInfo, AgentContextItem, ExecutionMode, WorkflowCaptureResult, WorkflowExecutionResult, AuditorViolation, AuditorJobStatus } from "../lib/api";
import { AggregateMetricsContent } from "./ChangeMetricsModal";
import { aggregateFileStats, formatNetLines } from "../lib/metrics";
import { AgentTab, AttachToAgentButton, type SentMessage } from "./AgentTab";
import { testFailureContextItems, codeQualityContextItems, changeSummaryContextItem, scenarioQualityContextItem, ruleViolationContextItems, rulesSummaryContextItem } from "../lib/agentContext";
import { Popover } from "./ui/popover";
import { ScenarioPickerModal } from "./ScenarioPickerModal";

type Tab = "overview" | "metrics" | "screenshots" | "workflows" | "tests" | "code-quality" | "rules" | "agent";

interface ScenarioReviewPanelProps {
  scenarioSlug: string;
  repoId?: string | null;
  fileStats?: RepoFileStats;
  onChangeScenario: (slug: string) => void;
  activeTab?: Tab;
  onActiveTabChange?: (tab: Tab) => void;
  agentRunId?: string | null;
  onAgentRunIdChange?: (id: string | null) => void;
  isMobile?: boolean;
}

export function ScenarioReviewPanel({ scenarioSlug, repoId, fileStats, onChangeScenario, activeTab: controlledTab, onActiveTabChange, agentRunId, onAgentRunIdChange, isMobile }: ScenarioReviewPanelProps) {
  const [isPickerOpen, setIsPickerOpen] = useState(false);
  const [internalTab, setInternalTab] = useState<Tab>("overview");
  const [agentSentMessages, setAgentSentMessages] = useState<SentMessage[]>([]);
  const activeTab = controlledTab ?? internalTab;
  const setActiveTab = onActiveTabChange ?? setInternalTab;
  const capturesQuery = useVisualCaptures(scenarioSlug, true, repoId);

  // Filter fileStats to only include files belonging to this scenario
  const scenarioFileStats = useMemo(() => {
    if (!fileStats) return undefined;
    const prefix = `scenarios/${scenarioSlug}/`;
    const filterCategory = (cat?: Record<string, import("../lib/api").DiffStats>) => {
      if (!cat) return undefined;
      const filtered: Record<string, import("../lib/api").DiffStats> = {};
      for (const [path, stats] of Object.entries(cat)) {
        if (path.startsWith(prefix)) filtered[path] = stats;
      }
      return Object.keys(filtered).length > 0 ? filtered : undefined;
    };
    const result: RepoFileStats = {
      staged: filterCategory(fileStats.staged),
      unstaged: filterCategory(fileStats.unstaged),
      untracked: filterCategory(fileStats.untracked),
    };
    if (!result.staged && !result.unstaged && !result.untracked) return undefined;
    return result;
  }, [fileStats, scenarioSlug]);
  const triggerCapture = useTriggerVisualCapture(repoId);
  const capabilities = useCapabilities();

  const basAvailable = capabilities.data?.capabilities?.some(
    c => c.id === "browser-automation-studio" && c.status === "available"
  ) ?? false;
  const testGenieAvailable = capabilities.data?.capabilities?.some(
    c => c.id === "test-genie" && c.status === "available"
  ) ?? false;
  const tidinessAvailable = capabilities.data?.capabilities?.some(
    c => c.id === "tidiness-manager" && c.status === "available"
  ) ?? false;
  const agentManagerAvailable = capabilities.data?.capabilities?.some(
    c => c.id === "agent-manager" && c.status === "available"
  ) ?? false;
  const auditorAvailable = capabilities.data?.capabilities?.some(
    c => c.id === "scenario-auditor" && c.status === "available"
  ) ?? false;

  const [agentContext, setAgentContext] = useState<AgentContextItem[]>([]);
  const addAgentContext = useCallback((item: AgentContextItem) => {
    setAgentContext((prev) => {
      if (prev.some((c) => c.id === item.id)) return prev;
      return [...prev, item];
    });
    setActiveTab("agent");
  }, []);
  const removeAgentContext = useCallback((id: string) => {
    setAgentContext((prev) => prev.filter((c) => c.id !== id));
  }, []);
  const clearAgentContext = useCallback(() => setAgentContext([]), []);

  const snapshots = capturesQuery.data?.snapshots ?? [];
  const completeSnapshots = snapshots.filter(s => s.status === "complete");
  // Role-based before/after: baseline is "Before", capture is "After"
  // Legacy snapshots without a role field are treated as captures
  const effectiveRole = (s: SnapshotSetMeta) => s.role || "capture";
  const baseline = completeSnapshots.find(s => effectiveRole(s) === "baseline");
  const capture = completeSnapshots.find(s => effectiveRole(s) === "capture");
  const captureStaleness = capturesQuery.data?.staleness;

  const [presetConfig, setPresetConfigState] = useState<CapturePreset[]>(() => getCapturePresets(scenarioSlug));
  useEffect(() => {
    setPresetConfigState(getCapturePresets(scenarioSlug));
  }, [scenarioSlug]);
  const handlePresetConfigChange = useCallback((presets: CapturePreset[]) => {
    setPresetConfigState(presets);
    saveCapturePresets(scenarioSlug, presets);
  }, [scenarioSlug]);

  const isCapturing = triggerCapture.isPending;
  const handleBaseline = useCallback(() => triggerCapture.mutate({ scenarioSlug, mode: "baseline", presets: presetConfig }), [triggerCapture, scenarioSlug, presetConfig]);
  const handleCapture = useCallback(() => triggerCapture.mutate({ scenarioSlug, mode: "capture", presets: presetConfig }), [triggerCapture, scenarioSlug, presetConfig]);

  // Workflow captures (mirrors screenshot capture pattern)
  const workflowCapturesQuery = useWorkflowCaptures(scenarioSlug, true, repoId);
  const triggerWorkflow = useTriggerWorkflowCapture(repoId);
  // Show all workflow captures (both complete and failed) — unlike screenshots,
  // failed workflow captures still have useful per-workflow error details to display.
  const workflowCaptures = workflowCapturesQuery.data?.captures ?? [];
  const workflowBaseline = workflowCaptures.find(c => c.role === "baseline");
  const workflowCapture = workflowCaptures.find(c => c.role === "capture");
  const workflowStaleness = workflowCapturesQuery.data?.staleness;
  const isRunningWorkflows = triggerWorkflow.isPending;
  const handleWorkflowBaseline = useCallback((executionModes: ExecutionMode[]) => {
    triggerWorkflow.mutate({ scenarioSlug, mode: "baseline", executionModes });
  }, [triggerWorkflow, scenarioSlug]);
  const handleWorkflowCapture = useCallback((executionModes: ExecutionMode[]) => {
    triggerWorkflow.mutate({ scenarioSlug, mode: "capture", executionModes });
  }, [triggerWorkflow, scenarioSlug]);

  const tabLabels: Record<Tab, string> = {
    overview: "Overview",
    metrics: "Metrics",
    screenshots: "Screenshots",
    workflows: "Workflows",
    tests: "Tests",
    "code-quality": "Code Quality",
    rules: "Rules",
    agent: "Agent",
  };

  const visibleTabs = (Object.keys(tabLabels) as Tab[]).filter(
    tab => {
      if (tab === "metrics") return Boolean(scenarioFileStats);
      if (tab === "code-quality") return tidinessAvailable;
      if (tab === "rules") return auditorAvailable;
      if (tab === "agent") return agentManagerAvailable;
      return true;
    }
  );

  const captureBanner = isCapturing && (
    <div className="flex items-center gap-2 px-4 py-2 bg-blue-950/50 border-b border-blue-900/50 text-blue-300 text-xs">
      <Loader2 className="h-3.5 w-3.5 animate-spin" />
      Capturing screenshots...
    </div>
  );

  const tabNav = (
    <div className="flex border-b border-slate-800 px-4 overflow-x-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]">
      {visibleTabs.map((tab) => (
        <button
          key={tab}
          type="button"
          onClick={() => setActiveTab(tab)}
          className={`px-4 ${isMobile ? "py-3 text-sm" : "py-2 text-xs"} font-medium border-b-2 transition-colors whitespace-nowrap ${
            activeTab === tab
              ? "text-blue-400 border-blue-400"
              : "text-slate-400 border-transparent hover:text-slate-200"
          }`}
        >
          {tabLabels[tab]}
        </button>
      ))}
    </div>
  );

  const tabContent = (
    <div className={`flex-1 ${activeTab === "agent" ? "flex flex-col overflow-hidden" : "overflow-y-auto px-4 py-4"}`}>
      {activeTab === "overview" ? (
        <OverviewTab
          baseline={baseline}
          capture={capture}
          captureStaleness={captureStaleness}
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          basAvailable={basAvailable}
          testGenieAvailable={testGenieAvailable}
          tidinessAvailable={tidinessAvailable}
          isCapturing={isCapturing}
          onBaseline={handleBaseline}
          onCapture={handleCapture}
          fileStats={scenarioFileStats}
          agentManagerAvailable={agentManagerAvailable}
          onAttachToAgent={addAgentContext}
        />
      ) : activeTab === "metrics" ? (
        scenarioFileStats ? <AggregateMetricsContent fileStats={scenarioFileStats} /> : null
      ) : activeTab === "screenshots" ? (
        capturesQuery.isLoading ? (
          <div className="space-y-4">
            <div className="h-48 animate-pulse bg-slate-800 rounded" />
            <div className="h-48 animate-pulse bg-slate-800 rounded" />
          </div>
        ) : capturesQuery.error ? (
          <MutationErrorBanner error={capturesQuery.error} />
        ) : (
          <ScreenshotsTab
            baseline={baseline}
            capture={capture}
            captureStaleness={captureStaleness}
            scenarioSlug={scenarioSlug}
            isMobile={isMobile ?? false}
            basAvailable={basAvailable}
            isCapturing={isCapturing}
            onBaseline={handleBaseline}
            onCapture={handleCapture}
            presetConfig={presetConfig}
            onPresetConfigChange={handlePresetConfigChange}
            mutationError={triggerCapture.error}
            onDismissError={() => triggerCapture.reset()}
          />
        )
      ) : activeTab === "workflows" ? (
        <WorkflowsTab
          baseline={workflowBaseline}
          capture={workflowCapture}
          captureStaleness={workflowStaleness}
          scenarioSlug={scenarioSlug}
          basAvailable={basAvailable}
          isRunning={isRunningWorkflows}
          onBaseline={handleWorkflowBaseline}
          onCapture={handleWorkflowCapture}
          mutationError={triggerWorkflow.error}
          onDismissError={() => triggerWorkflow.reset()}
        />
      ) : activeTab === "tests" ? (
        <TestsTab
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          testGenieAvailable={testGenieAvailable}
          agentManagerAvailable={agentManagerAvailable}
          onAttachToAgent={addAgentContext}
        />
      ) : activeTab === "code-quality" ? (
        <CodeQualityTab
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          tidinessAvailable={tidinessAvailable}
          fileStats={scenarioFileStats}
          agentManagerAvailable={agentManagerAvailable}
          onAttachToAgent={addAgentContext}
        />
      ) : activeTab === "rules" ? (
        <RulesTab
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          auditorAvailable={auditorAvailable}
          agentManagerAvailable={agentManagerAvailable}
          onAttachToAgent={addAgentContext}
        />
      ) : (
        <AgentTab
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          agentManagerAvailable={agentManagerAvailable}
          contextItems={agentContext}
          onAddContext={addAgentContext}
          onRemoveContext={removeAgentContext}
          onClearContext={clearAgentContext}
          testGenieAvailable={testGenieAvailable}
          tidinessAvailable={tidinessAvailable}
          auditorAvailable={auditorAvailable}
          fileStats={scenarioFileStats}
          activeRunId={agentRunId}
          onActiveRunIdChange={onAgentRunIdChange}
          sentMessages={agentSentMessages}
          onSentMessagesChange={setAgentSentMessages}
        />
      )}
    </div>
  );

  if (!scenarioSlug) {
    return (
      <Card className="h-full flex flex-col">
        <div className="flex-1 flex flex-col items-center justify-center text-slate-500 gap-3 p-8">
          <ClipboardCheck className="h-10 w-10 text-slate-600" />
          <p className="text-sm text-center">No scenario selected. Choose a scenario to review.</p>
          <button
            type="button"
            onClick={() => setIsPickerOpen(true)}
            className="px-3 py-1.5 rounded-lg border border-slate-700 text-xs text-slate-300 hover:bg-slate-800/60 transition-colors"
          >
            Choose scenario
          </button>
          <ScenarioPickerModal
            isOpen={isPickerOpen}
            onClose={() => setIsPickerOpen(false)}
            currentScenario={scenarioSlug}
            onSelect={(slug) => {
              setIsPickerOpen(false);
              onChangeScenario(slug);
            }}
          />
        </div>
      </Card>
    );
  }

  return (
    <Card className="h-full flex flex-col">
      <CardHeader className={`flex-row items-center justify-between space-y-0 ${isMobile ? "py-4 px-4" : "py-3"}`}>
        <CardTitle className="flex items-center gap-2 min-w-0">
          <ClipboardCheck className="h-4 w-4 text-slate-400 shrink-0" />
          <button
            type="button"
            onClick={() => setIsPickerOpen(true)}
            className={`font-semibold text-slate-100 ${isMobile ? "text-base" : "text-sm"} hover:text-blue-400 cursor-pointer flex items-center gap-1 transition-colors truncate`}
          >
            {scenarioSlug}
            <ChevronDown className="h-3 w-3 text-slate-500 shrink-0" />
          </button>
          {(capture || baseline) && (
            <span className="text-[11px] text-slate-500 hidden sm:inline shrink-0">
              {capture ? `Captured ${new Date(capture.createdAt).toLocaleString()}` : `Baseline ${new Date(baseline!.createdAt).toLocaleString()}`}
              {captureStaleness?.isStale && <span className="ml-1 text-amber-500">(stale)</span>}
            </span>
          )}
        </CardTitle>
        <div className="flex items-center gap-1 shrink-0">
          {basAvailable && activeTab !== "agent" && (
            <>
              <button
                type="button"
                className={`${isMobile ? "h-11 w-11 touch-target" : "h-7 px-2"} inline-flex items-center justify-center gap-1 rounded text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800/60 transition-colors`}
                onClick={handleBaseline}
                disabled={isCapturing}
                title="Set baseline (reset Before)"
              >
                {isCapturing ? (
                  <Loader2 className={`${isMobile ? "h-4 w-4" : "h-3.5 w-3.5"} animate-spin`} />
                ) : (
                  <Anchor className={isMobile ? "h-4 w-4" : "h-3.5 w-3.5"} />
                )}
              </button>
              <button
                type="button"
                className={`${isMobile ? "h-11 w-11 touch-target" : "h-7 px-2"} inline-flex items-center justify-center gap-1 rounded text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800/60 transition-colors`}
                onClick={handleCapture}
                disabled={isCapturing}
                title="Capture current state (After)"
              >
                {isCapturing ? (
                  <Loader2 className={`${isMobile ? "h-4 w-4" : "h-3.5 w-3.5"} animate-spin`} />
                ) : (
                  <Camera className={isMobile ? "h-4 w-4" : "h-3.5 w-3.5"} />
                )}
              </button>
            </>
          )}
        </div>
      </CardHeader>
      {captureBanner}
      {tabNav}
      {tabContent}
      <ScenarioPickerModal
        isOpen={isPickerOpen}
        onClose={() => setIsPickerOpen(false)}
        currentScenario={scenarioSlug}
        onSelect={(slug) => {
          setIsPickerOpen(false);
          onChangeScenario(slug);
        }}
      />
    </Card>
  );
}

// ============================================================================
// Media Lightbox with navigation
// ============================================================================

interface LightboxItem {
  label: string;
  sublabel?: string;
  type: "image" | "video";
  url: string;
}

function MediaLightbox({
  items,
  initialIndex,
  isOpen,
  onClose,
}: {
  items: LightboxItem[];
  initialIndex: number;
  isOpen: boolean;
  onClose: () => void;
}) {
  const [index, setIndex] = useState(initialIndex);

  // Reset index when opening with a new initialIndex
  useEffect(() => {
    if (isOpen) setIndex(initialIndex);
  }, [isOpen, initialIndex]);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === "Escape") onClose();
    if (e.key === "ArrowLeft") setIndex(i => Math.max(0, i - 1));
    if (e.key === "ArrowRight") setIndex(i => Math.min(items.length - 1, i + 1));
  }, [onClose, items.length]);

  useEffect(() => {
    if (!isOpen) return;
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, handleKeyDown]);

  if (!isOpen || items.length === 0) return null;

  const clampedIndex = Math.max(0, Math.min(index, items.length - 1));
  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion -- clampedIndex is always in bounds after the length check above
  const current = items[clampedIndex]!;
  const hasPrev = clampedIndex > 0;
  const hasNext = clampedIndex < items.length - 1;

  return createPortal(
    <div
      className="fixed inset-0 z-[60] flex flex-col bg-black/95"
      onClick={onClose}
    >
      {/* Top info bar */}
      <div
        className="flex items-center justify-between px-4 py-3 bg-black/80 border-b border-slate-800/50"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="min-w-0">
          <p className="text-sm font-medium text-slate-200 truncate">{current.label}</p>
          {current.sublabel && (
            <p className="text-[11px] text-slate-500 truncate">{current.sublabel}</p>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0 ml-4">
          {items.length > 1 && (
            <span className="text-xs text-slate-500">{clampedIndex + 1} / {items.length}</span>
          )}
          <button
            type="button"
            className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Media area */}
      <div
        className="flex-1 flex items-center justify-center p-4 min-h-0"
        onClick={(e) => e.stopPropagation()}
      >
        {current.type === "image" ? (
          <img
            key={current.url}
            src={current.url}
            alt={current.label}
            className="max-w-full max-h-full object-contain rounded-lg"
          />
        ) : (
          <video
            key={current.url}
            controls
            autoPlay
            src={current.url}
            className="max-w-full max-h-full rounded-lg"
          />
        )}
      </div>

      {/* Bottom nav bar */}
      {items.length > 1 && (
        <div
          className="flex items-center justify-center gap-4 px-4 py-3 bg-black/80 border-t border-slate-800/50"
          onClick={(e) => e.stopPropagation()}
        >
          <button
            type="button"
            disabled={!hasPrev}
            onClick={() => setIndex(i => i - 1)}
            className="h-10 w-10 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800 disabled:opacity-30 disabled:cursor-not-allowed"
            aria-label="Previous"
          >
            <ChevronLeft className="h-5 w-5" />
          </button>
          <div className="flex gap-1.5">
            {items.map((_, i) => (
              <button
                key={i}
                type="button"
                onClick={() => setIndex(i)}
                className={`h-2 rounded-full transition-all ${
                  i === clampedIndex ? "w-6 bg-blue-400" : "w-2 bg-slate-600 hover:bg-slate-500"
                }`}
                aria-label={`Go to item ${i + 1}`}
              />
            ))}
          </div>
          <button
            type="button"
            disabled={!hasNext}
            onClick={() => setIndex(i => i + 1)}
            className="h-10 w-10 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800 disabled:opacity-30 disabled:cursor-not-allowed"
            aria-label="Next"
          >
            <ChevronRight className="h-5 w-5" />
          </button>
        </div>
      )}
    </div>,
    document.body,
  );
}

// ============================================================================
// Shared UI Helpers
// ============================================================================

function MutationErrorBanner({ error, onDismiss }: { error: Error | null; onDismiss?: () => void }) {
  const [copied, setCopied] = useState(false);
  if (!error) return null;
  const handleCopy = () => {
    void navigator.clipboard.writeText(error.message).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };
  return (
    <div className="flex items-start gap-2 p-3 rounded-lg bg-red-950/30 border border-red-900/40">
      <AlertTriangle className="h-3.5 w-3.5 text-red-400 mt-0.5 shrink-0" />
      <p className="flex-1 text-xs text-red-300 max-h-32 overflow-y-auto break-words">{error.message}</p>
      <button type="button" onClick={handleCopy} className="text-red-400 hover:text-red-300 shrink-0" aria-label="Copy error" title="Copy error">
        {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
      </button>
      {onDismiss && (
        <button type="button" onClick={onDismiss} className="text-red-400 hover:text-red-300 shrink-0" aria-label="Dismiss">
          <X className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  );
}

function ServiceUnavailableBanner({ name, message }: { name: string; message?: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-slate-500">
      <AlertCircle className="h-8 w-8 mb-3 opacity-50" />
      <p className="text-sm">{name} is not available</p>
      {message && (
        <p className="text-xs mt-2 text-slate-600 text-center max-w-sm">{message}</p>
      )}
    </div>
  );
}

// ============================================================================
// Overview Tab
// ============================================================================

function OverviewTab({
  baseline,
  capture,
  captureStaleness,
  scenarioSlug,
  repoId,
  basAvailable,
  testGenieAvailable,
  tidinessAvailable,
  isCapturing,
  onBaseline,
  onCapture,
  fileStats,
  agentManagerAvailable,
  onAttachToAgent,
}: {
  baseline?: SnapshotSetMeta;
  capture?: SnapshotSetMeta;
  captureStaleness?: SnapshotStalenessInfo;
  scenarioSlug: string;
  repoId?: string | null;
  basAvailable: boolean;
  testGenieAvailable: boolean;
  tidinessAvailable: boolean;
  isCapturing: boolean;
  onBaseline: () => void;
  onCapture: () => void;
  fileStats?: RepoFileStats;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
}) {
  const testExecutions = useTestExecutions(scenarioSlug, testGenieAvailable, repoId);
  const latestTest = testExecutions.data?.items?.[0] as TestExecutionResult | undefined;
  const tidinessScore = useTidinessScore(scenarioSlug, tidinessAvailable, repoId);
  const tidinessStaleness = useTidinessStaleness(scenarioSlug, tidinessAvailable, repoId);
  const scenarios = useScenarios();
  const scenarioInfo = scenarios.data?.find(s => s.name === scenarioSlug);
  const [proxyUrl, setProxyUrl] = useState(`/embedded/${encodeURIComponent(scenarioSlug)}/`);

  useEffect(() => {
    let cancelled = false;
    fetch(`/embedded/${encodeURIComponent(scenarioSlug)}/external-url`)
      .then(res => res.ok ? res.json() : null)
      .then(data => {
        if (!cancelled && data?.url) {
          setProxyUrl(data.url as string);
        }
      })
      .catch(() => { /* keep fallback */ });
    return () => { cancelled = true; };
  }, [scenarioSlug]);

  // Readiness logic — either a baseline or capture counts as "has screenshots"
  const latestSnapshot = capture ?? baseline;
  const hasScreenshots = latestSnapshot && latestSnapshot.screenshotCount > 0;
  const hasTests = Boolean(latestTest);
  const testsPass = latestTest?.success ?? false;
  const qualityScore = tidinessScore.data?.score ?? null;
  const hasBeenScanned = Boolean(tidinessStaleness.data?.last_scan_at) ||
    (tidinessStaleness.data ? !tidinessStaleness.data.stale_reason?.includes("no scans") : false);
  const qualityOk = hasBeenScanned && qualityScore !== null && qualityScore >= 60;

  let readiness: "green" | "yellow" | "red" = "red";
  if (hasScreenshots && hasTests && testsPass && qualityOk) {
    readiness = "green";
  } else if (hasScreenshots || hasTests || qualityOk) {
    readiness = "yellow";
  }

  const readinessColors = {
    green: "bg-emerald-500",
    yellow: "bg-amber-500",
    red: "bg-red-500",
  };
  const readinessLabels = {
    green: "Ready",
    yellow: "Incomplete",
    red: "No data",
  };

  return (
    <div className="space-y-4">
      {/* Scenario Info Card */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-xs font-medium text-slate-400">Scenario</h3>
          <div className="flex items-center gap-2">
            <div className={`h-2 w-2 rounded-full ${scenarioInfo?.status === "running" ? "bg-emerald-500" : "bg-slate-500"}`} />
            <span className={`text-[11px] ${scenarioInfo?.status === "running" ? "text-emerald-400" : "text-slate-500"}`}>
              {scenarioInfo?.status ?? "unknown"}
            </span>
          </div>
        </div>
        <p className="text-sm font-medium text-slate-200 mb-2">
          {scenarioInfo?.display_name || scenarioSlug}
        </p>
        <div className="flex items-center gap-2">
          <span className="text-[11px] text-slate-500 truncate flex-1 font-mono">{proxyUrl}</span>
          <a
            href={proxyUrl}
            target="_blank"
            rel="noopener"
            className={`inline-flex items-center gap-1 text-[11px] px-2 py-1 rounded border transition-colors shrink-0 ${
              scenarioInfo?.status === "running"
                ? "text-blue-400 border-blue-800 hover:bg-blue-950/50"
                : "text-slate-600 border-slate-700 pointer-events-none opacity-50"
            }`}
            aria-label="Open scenario in new tab"
          >
            <ExternalLink className="h-3 w-3" />
            Open
          </a>
        </div>
        {scenarioInfo?.health_status && scenarioInfo.health_status !== "healthy" && (
          <div className="flex items-center gap-2 mt-2 text-[11px] text-amber-400">
            <AlertTriangle className="h-3 w-3" />
            {scenarioInfo.health_status}
          </div>
        )}
      </div>

      {/* Readiness indicator */}
      <div className="flex items-center gap-2 mb-4">
        <div className={`h-3 w-3 rounded-full ${readinessColors[readiness]}`} />
        <span className="text-sm font-medium text-slate-200">{readinessLabels[readiness]}</span>
      </div>

      {/* Change Summary Card */}
      {fileStats && (() => {
        const agg = aggregateFileStats(fileStats);
        if (!agg || agg.totalFiles === 0) return null;
        return (
          <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-xs font-medium text-slate-400">Change Summary</h3>
              {agentManagerAvailable && onAttachToAgent && (
                <AttachToAgentButton onClick={() => onAttachToAgent(changeSummaryContextItem(fileStats))} />
              )}
            </div>
            <div className="flex items-center gap-4">
              <span className="flex items-center gap-1 text-emerald-500 text-sm font-medium">
                <Plus className="h-3.5 w-3.5" />
                {agg.totalAdditions}
              </span>
              <span className="flex items-center gap-1 text-red-500 text-sm font-medium">
                <Minus className="h-3.5 w-3.5" />
                {agg.totalDeletions}
              </span>
              <span className="text-sm font-medium text-blue-400">
                net {formatNetLines(agg.totalNetLines)}
              </span>
            </div>
            <div className="mt-2 text-xs text-slate-400">
              {agg.totalFiles} file{agg.totalFiles !== 1 ? "s" : ""} changed
            </div>
          </div>
        );
      })()}

      {/* Visual Status Card */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-xs font-medium text-slate-400">Visual Status</h3>
          {captureStaleness?.isStale && (
            <span className="text-[11px] text-amber-400 flex items-center gap-1">
              <AlertTriangle className="h-3 w-3" />
              Stale
            </span>
          )}
        </div>
        {!basAvailable ? (
          <p className="text-xs text-slate-500">Start browser-automation-studio to enable visual captures</p>
        ) : !latestSnapshot ? (
          <div className="space-y-2">
            <p className="text-xs text-slate-500">No captures yet</p>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={onBaseline} disabled={isCapturing} className="h-7 text-xs gap-1">
                {isCapturing ? <Loader2 className="h-3 w-3 animate-spin" /> : <Anchor className="h-3 w-3" />}
                Set Baseline
              </Button>
            </div>
          </div>
        ) : (
          <div className="space-y-2">
            {baseline && (
              <div className="flex justify-between text-xs">
                <span className="text-slate-400">Baseline</span>
                <span className="text-slate-200">{new Date(baseline.createdAt).toLocaleString()}</span>
              </div>
            )}
            {capture && (
              <div className="flex justify-between text-xs">
                <span className="text-slate-400">Capture{captureStaleness?.isStale ? " (stale)" : ""}</span>
                <span className="text-slate-200">{new Date(capture.createdAt).toLocaleString()}</span>
              </div>
            )}
            <div className="flex justify-between text-xs">
              <span className="text-slate-400">Screenshots</span>
              <span className="text-slate-200">{latestSnapshot.screenshotCount}</span>
            </div>
            {latestSnapshot.pageDiscoveryMethod === "fallback" && (
              <div className="flex items-start gap-2 mt-2 p-2 rounded bg-amber-950/30 border border-amber-900/40">
                <AlertTriangle className="h-3.5 w-3.5 text-amber-400 mt-0.5 shrink-0" />
                <p className="text-[11px] text-amber-300">
                  Pages discovered via fallback (root only). Add <code className="bg-slate-800 px-1 rounded">.vrooli/lighthouse.json</code> to capture all pages.
                </p>
              </div>
            )}
            {!capture && baseline && (
              <Button variant="outline" size="sm" onClick={onCapture} disabled={isCapturing} className="h-7 text-xs gap-1 mt-2">
                {isCapturing ? <Loader2 className="h-3 w-3 animate-spin" /> : <Camera className="h-3 w-3" />}
                Capture Changes
              </Button>
            )}
          </div>
        )}
      </div>

      {/* Test Status Card */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-xs font-medium text-slate-400">Test Status</h3>
          {agentManagerAvailable && onAttachToAgent && latestTest && !latestTest.success && (
            <AttachToAgentButton onClick={() => {
              const failedPhases = latestTest.phases.filter(p => p.status === "failed");
              for (const item of testFailureContextItems(failedPhases)) {
                onAttachToAgent(item);
              }
            }} />
          )}
        </div>
        {!testGenieAvailable ? (
          <p className="text-xs text-slate-500">Start test-genie to enable automated tests</p>
        ) : testExecutions.isLoading ? (
          <div className="h-12 animate-pulse bg-slate-800 rounded" />
        ) : !latestTest ? (
          <p className="text-xs text-slate-500">No tests run yet</p>
        ) : (
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              {latestTest.success ? (
                <CheckCircle2 className="h-4 w-4 text-emerald-400" />
              ) : (
                <XCircle className="h-4 w-4 text-red-400" />
              )}
              <span className={`text-xs font-medium ${latestTest.success ? "text-emerald-300" : "text-red-300"}`}>
                {latestTest.success ? "Passed" : "Failed"}
              </span>
            </div>
            <div className="flex justify-between text-xs">
              <span className="text-slate-400">Phases</span>
              <span className="text-slate-200">
                {latestTest.phaseSummary.passed}/{latestTest.phaseSummary.total} passed
              </span>
            </div>
            <div className="flex justify-between text-xs">
              <span className="text-slate-400">Duration</span>
              <span className="text-slate-200">{formatDuration(latestTest.phaseSummary.durationSeconds)}</span>
            </div>
            {latestTest.completedAt && (
              <div className="flex justify-between text-xs">
                <span className="text-slate-400">Completed</span>
                <span className="text-slate-200">{new Date(latestTest.completedAt).toLocaleString()}</span>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Code Quality Card */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-xs font-medium text-slate-400">Code Quality</h3>
          {agentManagerAvailable && onAttachToAgent && tidinessScore.data && hasBeenScanned && tidinessScore.data.violations > 0 && (
            <AttachToAgentButton onClick={() => onAttachToAgent(scenarioQualityContextItem(tidinessScore.data as { score: number; violations: number }))} />
          )}
        </div>
        {!tidinessAvailable ? (
          <p className="text-xs text-slate-500">Start tidiness-manager to view code quality data</p>
        ) : tidinessScore.isLoading ? (
          <div className="h-12 animate-pulse bg-slate-800 rounded" />
        ) : tidinessScore.error ? (
          <p className="text-xs text-slate-500">No quality data available</p>
        ) : tidinessScore.data ? (
          !hasBeenScanned ? (
            <div className="space-y-2">
              <p className="text-xs text-slate-500">Not yet scanned</p>
              <p className="text-[11px] text-slate-600">Open the Code Quality tab to run a scan</p>
            </div>
          ) : (
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <div className={`h-4 w-4 rounded-full flex items-center justify-center ${
                  tidinessScore.data.score >= 70 ? "bg-emerald-500" :
                  tidinessScore.data.score >= 40 ? "bg-amber-500" : "bg-red-500"
                }`}>
                  <span className="text-[8px] font-bold text-white">
                    {Math.round(tidinessScore.data.score)}
                  </span>
                </div>
                <span className={`text-xs font-medium ${
                  tidinessScore.data.score >= 70 ? "text-emerald-300" :
                  tidinessScore.data.score >= 40 ? "text-amber-300" : "text-red-300"
                }`}>
                  {tidinessScore.data.score >= 70 ? "Good" :
                   tidinessScore.data.score >= 40 ? "Fair" : "Poor"}
                </span>
              </div>
              <div className="flex justify-between text-xs">
                <span className="text-slate-400">Score</span>
                <span className="text-slate-200">{Math.round(tidinessScore.data.score)}/100</span>
              </div>
              <div className="flex justify-between text-xs">
                <span className="text-slate-400">Violations</span>
                <span className="text-slate-200">{tidinessScore.data.violations}</span>
              </div>
              {tidinessStaleness.data?.last_scan_at && (
                <div className="flex justify-between text-xs">
                  <span className="text-slate-400">Last scan</span>
                  <span className="text-slate-200">{formatRelativeTime(tidinessStaleness.data.last_scan_at)}</span>
                </div>
              )}
              {tidinessStaleness.data?.is_stale && (
                <div className="flex items-start gap-2 mt-2 p-2 rounded bg-amber-950/30 border border-amber-900/40">
                  <AlertTriangle className="h-3.5 w-3.5 text-amber-400 mt-0.5 shrink-0" />
                  <p className="text-[11px] text-amber-300">
                    {formatStalenessMessage(tidinessStaleness.data)}
                  </p>
                </div>
              )}
            </div>
          )
        ) : (
          <p className="text-xs text-slate-500">No quality data available</p>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// Screenshots Tab
// ============================================================================

function ScreenshotsTab({
  baseline,
  capture,
  captureStaleness,
  scenarioSlug,
  isMobile,
  basAvailable,
  isCapturing,
  onBaseline,
  onCapture,
  presetConfig,
  onPresetConfigChange,
  mutationError,
  onDismissError,
}: {
  baseline?: SnapshotSetMeta;
  capture?: SnapshotSetMeta;
  captureStaleness?: SnapshotStalenessInfo;
  scenarioSlug: string;
  isMobile: boolean;
  basAvailable: boolean;
  isCapturing: boolean;
  onBaseline: () => void;
  onCapture: () => void;
  presetConfig: CapturePreset[];
  onPresetConfigChange: (presets: CapturePreset[]) => void;
  mutationError?: Error | null;
  onDismissError?: () => void;
}) {
  const [selectedPage, setSelectedPage] = useState(0);
  const [lightboxIndex, setLightboxIndex] = useState(-1);
  const [mobileDropdownOpen, setMobileDropdownOpen] = useState(false);
  const [configOpen, setConfigOpen] = useState(false);

  // Determine which presets were captured (from snapshot metadata)
  const primarySnapshot = capture ?? baseline;
  const capturedPresets: CapturePreset[] = primarySnapshot?.presets ?? [];

  // Active preset for filtering — default to first captured preset, or first config preset
  const [activePreset, setActivePreset] = useState<CapturePreset>(
    () => capturedPresets[0] ?? presetConfig[0] ?? { name: "Desktop Light", width: 1440, height: 900, theme: "light" as CaptureTheme }
  );

  // No snapshots at all
  if (!baseline && !capture) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <MutationErrorBanner error={mutationError ?? null} onDismiss={onDismissError} />
        <ClipboardCheck className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm">No captures yet</p>
        <p className="text-xs mt-1 mb-3 text-slate-600">Set a baseline to start comparing visual changes</p>
        {basAvailable ? (
          <Button variant="outline" size="sm" onClick={onBaseline} disabled={isCapturing} className="h-7 text-xs gap-1">
            {isCapturing ? <Loader2 className="h-3 w-3 animate-spin" /> : <Anchor className="h-3 w-3" />}
            Set Baseline
          </Button>
        ) : (
          <p className="text-xs">Start browser-automation-studio to enable visual captures</p>
        )}
      </div>
    );
  }

  const primaryPages = primarySnapshot!.pages ?? [];
  const currentPage = primaryPages[selectedPage] ?? "/";

  // Build filename for current preset
  const screenshotFilename = (pagePath: string) =>
    sanitizePagePath(pagePath) + presetSuffix(activePreset) + ".png";

  // Build lightbox items for the active preset: baseline first, then capture
  const lightboxItems: LightboxItem[] = [];
  const baselinePages = baseline?.pages ?? [];
  if (baseline) {
    for (const page of baselinePages) {
      lightboxItems.push({
        label: `Baseline: ${page === "/" ? "/ (Home)" : page}`,
        sublabel: `${new Date(baseline.createdAt).toLocaleString()} (${presetLabel(activePreset)})`,
        type: "image",
        url: buildCaptureScreenshotUrl(baseline.id, scenarioSlug, screenshotFilename(page)),
      });
    }
  }
  if (capture) {
    const capturePages = capture.pages ?? [];
    for (const page of capturePages) {
      lightboxItems.push({
        label: baseline ? `Capture: ${page === "/" ? "/ (Home)" : page}` : page === "/" ? "/ (Home)" : page,
        sublabel: `${new Date(capture.createdAt).toLocaleString()} (${presetLabel(activePreset)})`,
        type: "image",
        url: buildCaptureScreenshotUrl(capture.id, scenarioSlug, screenshotFilename(page)),
      });
    }
  }

  const baselineIndex = (pageIdx: number) => pageIdx;
  const captureIndex = (pageIdx: number) => baselinePages.length + pageIdx;

  // Capture time estimate
  const numPages = primaryPages.length || 1;
  const numPresets = presetConfig.length || 1;
  const estimatedSeconds = numPages * numPresets * 3;

  return (
    <div className="space-y-4">
      <MutationErrorBanner error={mutationError ?? null} onDismiss={onDismissError} />
      {/* Action buttons */}
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={onBaseline} disabled={isCapturing} className="h-7 text-xs gap-1">
          {isCapturing ? <Loader2 className="h-3 w-3 animate-spin" /> : <Anchor className="h-3 w-3" />}
          {baseline ? "Reset Baseline" : "Set Baseline"}
        </Button>
        <Button variant="outline" size="sm" onClick={onCapture} disabled={isCapturing} className="h-7 text-xs gap-1">
          {isCapturing ? <Loader2 className="h-3 w-3 animate-spin" /> : <Camera className="h-3 w-3" />}
          {capture ? "Re-capture" : "Capture"}
        </Button>
      </div>

      {/* Capture time estimate */}
      {(numPresets > 1 || numPages > 1) && (
        <p className="text-[10px] text-slate-600">
          ~{estimatedSeconds}s for {numPresets} preset{numPresets > 1 ? "s" : ""} &times; {numPages} page{numPages > 1 ? "s" : ""}
        </p>
      )}

      {/* Preset filter — desktop: chips + gear, mobile: dropdown */}
      {capturedPresets.length > 0 && (
        isMobile ? (
          <div className="relative">
            <button
              type="button"
              onClick={() => setMobileDropdownOpen(v => !v)}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-slate-800 text-xs text-slate-300 hover:text-white transition-colors"
            >
              {presetLabel(activePreset)}
              <ChevronDown className="h-3 w-3" />
            </button>
            {mobileDropdownOpen && (
              <div className="absolute left-0 top-full mt-1 z-50 min-w-[180px] rounded-lg border border-slate-700 bg-slate-900 shadow-xl py-1">
                {capturedPresets.map(p => (
                  <button
                    key={presetKey(p)}
                    type="button"
                    onClick={() => { setActivePreset(p); setMobileDropdownOpen(false); }}
                    className={`block w-full text-left px-3 py-1.5 text-xs transition-colors ${
                      presetKey(p) === presetKey(activePreset)
                        ? "bg-blue-600 text-white"
                        : "text-slate-400 hover:text-slate-200 hover:bg-slate-800"
                    }`}
                  >
                    {presetLabel(p)}
                  </button>
                ))}
                <div className="border-t border-slate-700 mt-1 pt-1">
                  <button
                    type="button"
                    onClick={() => { setMobileDropdownOpen(false); setConfigOpen(true); }}
                    className="block w-full text-left px-3 py-1.5 text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800"
                  >
                    <Settings className="h-3 w-3 inline mr-1.5" />
                    Configure presets...
                  </button>
                </div>
              </div>
            )}
          </div>
        ) : (
          <div className="flex items-center gap-1.5">
            {capturedPresets.map(p => (
              <button
                key={presetKey(p)}
                type="button"
                onClick={() => setActivePreset(p)}
                className={`px-2.5 py-1 rounded-full text-xs whitespace-nowrap transition-colors ${
                  presetKey(p) === presetKey(activePreset)
                    ? "bg-blue-600 text-white"
                    : "bg-slate-800 text-slate-400 hover:text-slate-200"
                }`}
              >
                {presetLabel(p)}
              </button>
            ))}
            <Popover
              trigger={<Settings className="h-3.5 w-3.5 text-slate-500 hover:text-slate-300 transition-colors" />}
              align="end"
            >
              <PresetConfigPanel
                config={presetConfig}
                onChange={onPresetConfigChange}
              />
            </Popover>
          </div>
        )
      )}

      {/* Preset config modal for mobile (triggered from dropdown) */}
      {configOpen && (
        <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/60" onClick={() => setConfigOpen(false)}>
          <div className="bg-slate-900 border border-slate-700 rounded-t-xl sm:rounded-xl w-full sm:max-w-sm p-4" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-3">
              <span className="text-sm font-medium text-slate-200">Configure Presets</span>
              <button type="button" onClick={() => setConfigOpen(false)} className="text-slate-500 hover:text-slate-300"><X className="h-4 w-4" /></button>
            </div>
            <PresetConfigPanel config={presetConfig} onChange={onPresetConfigChange} />
          </div>
        </div>
      )}

      {/* Staleness warning */}
      {captureStaleness?.isStale && capture && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-950/30 border border-amber-900/40">
          <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 shrink-0" />
          <p className="text-xs text-amber-300">
            Files have changed since this capture. Re-capture to see the latest state.
          </p>
        </div>
      )}

      {/* Fallback warning */}
      {primarySnapshot!.pageDiscoveryMethod === "fallback" && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-950/30 border border-amber-900/40">
          <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 shrink-0" />
          <p className="text-xs text-amber-300">
            Pages discovered via fallback (root only). Add <code className="bg-slate-800 px-1 rounded">.vrooli/lighthouse.json</code> to capture all pages.
          </p>
        </div>
      )}

      {/* Page selector */}
      {primaryPages.length > 1 && (
        <div className="flex gap-1.5 overflow-x-auto pb-2">
          {primaryPages.map((page, i) => (
            <button
              key={page}
              type="button"
              onClick={() => setSelectedPage(i)}
              className={`px-2.5 py-1 rounded-full text-xs whitespace-nowrap transition-colors ${
                i === selectedPage
                  ? "bg-blue-600 text-white"
                  : "bg-slate-800 text-slate-400 hover:text-slate-200"
              }`}
            >
              {page === "/" ? "/ (Home)" : page}
            </button>
          ))}
        </div>
      )}

      {/* Current page URL label */}
      <div className="text-xs text-slate-500">
        Page: <code className="bg-slate-800 px-1.5 py-0.5 rounded text-slate-300">{currentPage}</code>
        {currentPage === "/" && " (Home)"}
      </div>

      {/* Status message when only baseline exists */}
      {baseline && !capture && (
        <div className="text-xs text-slate-500 bg-slate-900/50 rounded px-3 py-2">
          Baseline set. Capture to compare changes against it.
        </div>
      )}

      {/* Side-by-side or stacked screenshots */}
      <div className={`gap-4 ${isMobile ? "space-y-4" : baseline && capture ? "grid grid-cols-2" : ""}`}>
        {baseline && capture && (
          <div>
            <div className="flex items-center gap-2 mb-2">
              <span className="text-xs font-medium text-slate-400">Baseline</span>
              <span className="text-[10px] text-slate-600">
                {new Date(baseline.createdAt).toLocaleString()}
              </span>
            </div>
            <ScreenshotImage
              captureId={baseline.id}
              scenarioSlug={scenarioSlug}
              pagePath={currentPage}
              preset={activePreset}
              onClick={() => setLightboxIndex(baselineIndex(selectedPage))}
            />
          </div>
        )}
        <div>
          <div className="flex items-center gap-2 mb-2">
            <span className="text-xs font-medium text-slate-400">
              {capture ? (baseline ? "Capture" : "Current") : "Baseline"}
            </span>
            {captureStaleness?.isStale && capture && (
              <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-900/50 text-amber-300">
                Stale
              </span>
            )}
            {baseline && capture && (
              <span className="text-[10px] text-slate-600">
                {new Date(capture.createdAt).toLocaleString()}
              </span>
            )}
          </div>
          <ScreenshotImage
            captureId={primarySnapshot!.id}
            scenarioSlug={scenarioSlug}
            pagePath={currentPage}
            preset={activePreset}
            onClick={() => setLightboxIndex(capture ? captureIndex(selectedPage) : baselineIndex(selectedPage))}
          />
        </div>
      </div>

      <MediaLightbox
        items={lightboxItems}
        initialIndex={lightboxIndex}
        isOpen={lightboxIndex >= 0}
        onClose={() => setLightboxIndex(-1)}
      />
    </div>
  );
}

function PresetConfigPanel({ config, onChange }: { config: CapturePreset[]; onChange: (presets: CapturePreset[]) => void }) {
  const [addName, setAddName] = useState("");
  const [addSize, setAddSize] = useState("Desktop");
  const [addCustomW, setAddCustomW] = useState("");
  const [addCustomH, setAddCustomH] = useState("");
  const [addTheme, setAddTheme] = useState<CaptureTheme>("light");

  const sizeEntries = Object.entries(SIZE_PRESETS);
  const isCustomSize = addSize === "Custom";

  // Auto-populate name from size + theme
  const autoName = useMemo(() => {
    const sizeName = isCustomSize ? `${addCustomW || "?"}x${addCustomH || "?"}` : addSize;
    return `${sizeName} ${addTheme === "light" ? "Light" : "Dark"}`;
  }, [addSize, addTheme, addCustomW, addCustomH, isCustomSize]);

  const effectiveName = addName || autoName;

  const addPreset = () => {
    let w: number, h: number;
    if (isCustomSize) {
      w = parseInt(addCustomW, 10);
      h = parseInt(addCustomH, 10);
      if (!w || !h || w <= 0 || h <= 0 || w > 7680 || h > 4320) return;
    } else {
      const preset = SIZE_PRESETS[addSize];
      if (!preset) return;
      w = preset.width;
      h = preset.height;
    }
    const newPreset: CapturePreset = { name: effectiveName, width: w, height: h, theme: addTheme };
    // Duplicate check by key
    if (config.some(c => presetKey(c) === presetKey(newPreset))) return;
    onChange([...config, newPreset]);
    setAddName("");
    setAddCustomW("");
    setAddCustomH("");
  };

  const removePreset = (p: CapturePreset) => {
    if (config.length <= 1) return;
    onChange(config.filter(c => presetKey(c) !== presetKey(p)));
  };

  const isDuplicate = (() => {
    let w: number, h: number;
    if (isCustomSize) {
      w = parseInt(addCustomW, 10);
      h = parseInt(addCustomH, 10);
      if (!w || !h) return false;
    } else {
      const preset = SIZE_PRESETS[addSize];
      if (!preset) return false;
      w = preset.width;
      h = preset.height;
    }
    return config.some(c => presetKey(c) === `${w}x${h}_${addTheme}`);
  })();

  return (
    <div className="p-3 space-y-3 min-w-[260px]">
      <p className="text-xs font-medium text-slate-400">Capture presets</p>

      {/* Current preset list */}
      {config.map(p => (
        <div key={presetKey(p)} className="flex items-center gap-2 text-xs text-slate-300">
          <span className="flex-1 truncate">{p.name}</span>
          <span className="text-slate-500">{p.width}&times;{p.height}</span>
          <span className={`px-1.5 py-0.5 rounded text-[10px] ${p.theme === "dark" ? "bg-slate-700 text-slate-300" : "bg-slate-800 text-slate-400"}`}>
            {p.theme === "dark" ? "Dark" : "Light"}
          </span>
          <button
            type="button"
            onClick={() => removePreset(p)}
            disabled={config.length <= 1}
            className="text-slate-600 hover:text-red-400 disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      ))}

      {/* Add form */}
      <div className="pt-2 border-t border-slate-800 space-y-2">
        <p className="text-[10px] font-medium text-slate-500 uppercase tracking-wider">Add preset</p>
        <input
          type="text"
          placeholder={autoName}
          value={addName}
          onChange={e => setAddName(e.target.value)}
          className="w-full px-2 py-1 rounded bg-slate-800 border border-slate-700 text-xs text-slate-300 placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500"
        />
        <div className="flex items-center gap-1.5">
          <select
            value={addSize}
            onChange={e => setAddSize(e.target.value)}
            className="flex-1 px-2 py-1 rounded bg-slate-800 border border-slate-700 text-xs text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            {sizeEntries.map(([name, s]) => (
              <option key={name} value={name}>{name} ({s.width}&times;{s.height})</option>
            ))}
            <option value="Custom">Custom</option>
          </select>
          <select
            value={addTheme}
            onChange={e => setAddTheme(e.target.value as CaptureTheme)}
            className="px-2 py-1 rounded bg-slate-800 border border-slate-700 text-xs text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
        </div>
        {isCustomSize && (
          <div className="flex items-center gap-1.5">
            <input
              type="number"
              placeholder="W"
              value={addCustomW}
              onChange={e => setAddCustomW(e.target.value)}
              className="w-20 px-1.5 py-1 rounded bg-slate-800 border border-slate-700 text-xs text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            <span className="text-xs text-slate-600">&times;</span>
            <input
              type="number"
              placeholder="H"
              value={addCustomH}
              onChange={e => setAddCustomH(e.target.value)}
              className="w-20 px-1.5 py-1 rounded bg-slate-800 border border-slate-700 text-xs text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
        )}
        <button
          type="button"
          onClick={addPreset}
          disabled={isDuplicate || (isCustomSize && (!addCustomW || !addCustomH))}
          className="w-full px-2 py-1.5 rounded bg-blue-600 text-white text-xs hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isDuplicate ? "Already exists" : "Add"}
        </button>
      </div>
    </div>
  );
}

function ScreenshotImage({
  captureId,
  scenarioSlug,
  pagePath,
  preset,
  onClick,
}: {
  captureId: string;
  scenarioSlug: string;
  pagePath: string;
  preset: CapturePreset;
  onClick: () => void;
}) {
  const filename = sanitizePagePath(pagePath) + presetSuffix(preset) + ".png";
  const url = buildCaptureScreenshotUrl(captureId, scenarioSlug, filename);

  return (
    <div
      className="rounded-lg border border-slate-800 overflow-hidden bg-slate-900 cursor-pointer hover:ring-2 hover:ring-blue-500/50 transition-shadow"
      onClick={onClick}
    >
      <img
        src={url}
        alt={`Screenshot of ${pagePath}`}
        className="max-w-full object-contain"
        loading="lazy"
      />
    </div>
  );
}

// ============================================================================
// Workflows Tab
// ============================================================================

const EXECUTION_MODE_COLORS: Record<ExecutionMode, string> = {
  observer: "bg-green-900/50 text-green-300 border-green-700/50",
  mutating: "bg-yellow-900/50 text-yellow-300 border-yellow-700/50",
  destructive: "bg-red-900/50 text-red-300 border-red-700/50",
};

const STATUS_ICONS: Record<string, typeof CheckCircle2> = {
  passed: CheckCircle2,
  failed: XCircle,
  skipped: Minus,
  error: AlertTriangle,
};

function WorkflowsTab({
  baseline,
  capture,
  captureStaleness,
  scenarioSlug,
  basAvailable,
  isRunning,
  onBaseline,
  onCapture,
  mutationError,
  onDismissError,
}: {
  baseline?: WorkflowCaptureResult;
  capture?: WorkflowCaptureResult;
  captureStaleness?: import("../lib/api").SnapshotStalenessInfo;
  scenarioSlug: string;
  basAvailable: boolean;
  isRunning: boolean;
  onBaseline: (executionModes: ExecutionMode[]) => void;
  onCapture: (executionModes: ExecutionMode[]) => void;
  mutationError?: Error | null;
  onDismissError?: () => void;
}) {
  const [selectedModes, setSelectedModes] = useState<Set<ExecutionMode>>(new Set(["observer"]));
  const [lightboxIndex, setLightboxIndex] = useState(-1);
  // Which role's results to show in the table ("capture" by default, toggle to "baseline")
  const [viewRole, setViewRole] = useState<"baseline" | "capture">("capture");
  // Which rows are expanded to show error details
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set());

  const toggleMode = useCallback((mode: ExecutionMode) => {
    setSelectedModes(prev => {
      const next = new Set(prev);
      if (next.has(mode)) next.delete(mode);
      else next.add(mode);
      return next;
    });
  }, []);

  const toggleExpanded = useCallback((idx: number) => {
    setExpandedRows(prev => {
      const next = new Set(prev);
      if (next.has(idx)) next.delete(idx);
      else next.add(idx);
      return next;
    });
  }, []);

  const modesArray = Array.from(selectedModes);

  // Which result to display: user-selected role, falling back to whatever exists
  const viewedResult = viewRole === "capture" ? (capture ?? baseline) : (baseline ?? capture);

  // Build lightbox items from viewed result videos
  const lightboxItems: LightboxItem[] = [];
  if (viewedResult) {
    for (const wfResult of viewedResult.workflowResults) {
      if (wfResult.videoCount > 0 && wfResult.executionId) {
        for (let i = 0; i < wfResult.videoCount; i++) {
          const filename = `${sanitizePagePath(wfResult.workflowName)}_${i}.webm`;
          lightboxItems.push({
            label: wfResult.workflowName,
            sublabel: `${wfResult.executionMode} - ${wfResult.status}`,
            type: "video",
            url: buildWorkflowVideoUrl(viewedResult.id, scenarioSlug, filename),
          });
        }
      }
    }
  }

  // Summary counts for a given result
  const summarize = (result: WorkflowCaptureResult) => ({
    passed: result.workflowResults.filter(r => r.status === "passed").length,
    failed: result.workflowResults.filter(r => r.status === "failed" || r.status === "error").length,
    skipped: result.workflowResults.filter(r => r.status === "skipped").length,
  });

  // No captures at all — empty state
  if (!baseline && !capture) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <MutationErrorBanner error={mutationError ?? null} onDismiss={onDismissError} />
        <Play className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm">No workflow captures yet</p>
        <p className="text-xs mt-1 mb-3 text-slate-600">Set a baseline to start comparing workflow results</p>
        {basAvailable ? (
          <>
            <ExecutionModeSelector selectedModes={selectedModes} onToggle={toggleMode} />
            <Button
              variant="outline"
              size="sm"
              onClick={() => onBaseline(modesArray)}
              disabled={isRunning || selectedModes.size === 0}
              className="h-7 text-xs gap-1 mt-2"
            >
              {isRunning ? <Loader2 className="h-3 w-3 animate-spin" /> : <Anchor className="h-3 w-3" />}
              Set Baseline
            </Button>
          </>
        ) : (
          <p className="text-xs">Start browser-automation-studio to enable workflow captures</p>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <MutationErrorBanner error={mutationError ?? null} onDismiss={onDismissError} />
      {/* Action buttons + execution mode selector */}
      <div className="flex items-center gap-2 flex-wrap">
        <Button
          variant="outline"
          size="sm"
          onClick={() => onBaseline(modesArray)}
          disabled={isRunning || selectedModes.size === 0}
          className="h-7 text-xs gap-1"
        >
          {isRunning ? <Loader2 className="h-3 w-3 animate-spin" /> : <Anchor className="h-3 w-3" />}
          {baseline ? "Reset Baseline" : "Set Baseline"}
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onCapture(modesArray)}
          disabled={isRunning || selectedModes.size === 0}
          className="h-7 text-xs gap-1"
        >
          {isRunning ? <Loader2 className="h-3 w-3 animate-spin" /> : <Camera className="h-3 w-3" />}
          {capture ? "Re-capture" : "Capture"}
        </Button>
        <div className="ml-auto">
          <ExecutionModeSelector selectedModes={selectedModes} onToggle={toggleMode} />
        </div>
      </div>

      {/* Staleness warning */}
      {captureStaleness?.isStale && capture && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-950/30 border border-amber-900/40">
          <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 shrink-0" />
          <p className="text-xs text-amber-300">
            Files have changed since this capture. Re-capture to see the latest workflow results.
          </p>
        </div>
      )}

      {/* Status message when only baseline exists */}
      {baseline && !capture && (
        <div className="text-xs text-slate-500 bg-slate-900/50 rounded px-3 py-2">
          Baseline set. Capture to compare workflow results against it.
        </div>
      )}

      {/* Summary bars — clickable to toggle which role's detail is shown */}
      {baseline && capture ? (
        <div className="space-y-1">
          {[capture, baseline].map((result) => {
            const role = result === capture ? "capture" : "baseline";
            const s = summarize(result);
            const isViewed = viewedResult === result;
            return (
              <button
                key={role}
                type="button"
                onClick={() => setViewRole(role)}
                className={`w-full flex items-center gap-4 px-3 py-2 rounded text-xs transition-colors text-left ${
                  isViewed ? "bg-slate-800/50 ring-1 ring-slate-600" : "bg-slate-800/30 hover:bg-slate-800/40"
                }`}
              >
                <span className="text-slate-300 font-medium capitalize">{role}</span>
                {role === "capture" && captureStaleness?.isStale && (
                  <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-900/50 text-amber-300">
                    Stale
                  </span>
                )}
                {result.status === "failed" && (
                  <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-red-900/50 text-red-300">
                    Failed
                  </span>
                )}
                <span className="text-green-400">{s.passed} passed</span>
                <span className="text-red-400">{s.failed} failed</span>
                <span className="text-slate-400">{s.skipped} skipped</span>
                <span className="text-slate-500 ml-auto">
                  {new Date(result.createdAt).toLocaleString()}
                </span>
                {isViewed && <ChevronRight className="h-3 w-3 text-blue-400" />}
              </button>
            );
          })}
        </div>
      ) : viewedResult && (() => {
        const s = summarize(viewedResult);
        return (
          <div className="flex items-center gap-4 px-3 py-2 bg-slate-800/50 rounded text-xs">
            <span className="text-slate-300 font-medium capitalize">{viewedResult.role}</span>
            {viewedResult.status === "failed" && (
              <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-red-900/50 text-red-300">
                Failed
              </span>
            )}
            <span className="text-green-400">{s.passed} passed</span>
            <span className="text-red-400">{s.failed} failed</span>
            <span className="text-slate-400">{s.skipped} skipped</span>
            <span className="text-slate-500 ml-auto">
              {new Date(viewedResult.createdAt).toLocaleString()}
            </span>
          </div>
        );
      })()}

      {/* Overall capture error */}
      {viewedResult?.error && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-red-950/30 border border-red-900/40">
          <XCircle className="h-4 w-4 text-red-400 mt-0.5 shrink-0" />
          <div className="min-w-0">
            <p className="text-xs font-medium text-red-300 mb-1">Capture error</p>
            <pre className="text-[11px] text-red-200/80 whitespace-pre-wrap break-words font-mono">{viewedResult.error}</pre>
          </div>
        </div>
      )}

      {/* Results table — shows whichever role is selected */}
      {viewedResult && viewedResult.workflowResults.length > 0 && (
        <div className="border border-slate-800 rounded-lg overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-slate-800/50">
                <th className="w-5 px-1 py-2"></th>
                <th className="text-left px-3 py-2 text-slate-400 font-medium">Workflow</th>
                <th className="text-left px-3 py-2 text-slate-400 font-medium">Mode</th>
                <th className="text-center px-3 py-2 text-slate-400 font-medium">Status</th>
                <th className="text-right px-3 py-2 text-slate-400 font-medium">Duration</th>
                <th className="text-center px-3 py-2 text-slate-400 font-medium">Video</th>
              </tr>
            </thead>
            <tbody>
              {viewedResult.workflowResults.map((wfr, idx) => {
                const StatusIcon = STATUS_ICONS[wfr.status] ?? AlertTriangle;
                const statusColor = wfr.status === "passed" ? "text-green-400"
                  : wfr.status === "failed" || wfr.status === "error" ? "text-red-400"
                  : "text-slate-500";
                const hasError = !!wfr.error;
                const isExpanded = expandedRows.has(idx);

                let videoLightboxIdx = -1;
                if (wfr.videoCount > 0) {
                  videoLightboxIdx = lightboxItems.findIndex(
                    item => item.label === wfr.workflowName
                  );
                }

                return (
                  <tr key={idx} className={`border-t border-slate-800/50 ${hasError ? "cursor-pointer" : ""} hover:bg-slate-800/30`} onClick={hasError ? () => toggleExpanded(idx) : undefined}>
                    <td className="px-1 py-2 text-center">
                      {hasError && (isExpanded ? <ChevronDown className="h-3 w-3 text-slate-500 inline" /> : <ChevronRight className="h-3 w-3 text-slate-500 inline" />)}
                    </td>
                    <td className="px-3 py-2 text-slate-200 max-w-[200px] truncate" title={wfr.workflowName}>
                      {wfr.workflowName}
                    </td>
                    <td className="px-3 py-2">
                      <span className={`px-1.5 py-0.5 rounded border text-[10px] ${EXECUTION_MODE_COLORS[wfr.executionMode as ExecutionMode] ?? "text-slate-400"}`}>
                        {wfr.executionMode}
                      </span>
                    </td>
                    <td className="px-3 py-2 text-center">
                      <StatusIcon className={`h-3.5 w-3.5 inline ${statusColor}`} />
                    </td>
                    <td className="px-3 py-2 text-right text-slate-400">
                      {wfr.durationMs > 0 ? formatDuration(Math.round(wfr.durationMs / 1000)) : "-"}
                    </td>
                    <td className="px-3 py-2 text-center">
                      {wfr.videoCount > 0 && videoLightboxIdx >= 0 ? (
                        <button
                          type="button"
                          onClick={(e) => { e.stopPropagation(); setLightboxIndex(videoLightboxIdx); }}
                          className="text-blue-400 hover:text-blue-300 text-[10px] underline"
                        >
                          Watch
                        </button>
                      ) : (
                        <span className="text-slate-600">-</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          {/* Expanded error details rendered outside table for proper layout */}
          {viewedResult.workflowResults.map((wfr, idx) =>
            expandedRows.has(idx) && wfr.error ? (
              <div key={`err-${idx}`} className="px-4 py-2 bg-red-950/20 border-t border-red-900/30">
                <p className="text-[10px] text-slate-400 mb-1">{wfr.workflowName} — error details</p>
                <pre className="text-[11px] text-red-200/80 whitespace-pre-wrap break-words font-mono max-h-48 overflow-y-auto">{wfr.error}</pre>
              </div>
            ) : null
          )}
        </div>
      )}

      <MediaLightbox
        items={lightboxItems}
        initialIndex={lightboxIndex}
        isOpen={lightboxIndex >= 0}
        onClose={() => setLightboxIndex(-1)}
      />
    </div>
  );
}

function ExecutionModeSelector({ selectedModes, onToggle }: { selectedModes: Set<ExecutionMode>; onToggle: (mode: ExecutionMode) => void }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-xs text-slate-400">Modes:</span>
      {(["observer", "mutating", "destructive"] as ExecutionMode[]).map(mode => (
        <label key={mode} className="flex items-center gap-1 text-xs cursor-pointer">
          <input
            type="checkbox"
            checked={selectedModes.has(mode)}
            onChange={() => onToggle(mode)}
            className="rounded border-slate-600"
          />
          <span className={`px-1.5 py-0.5 rounded border text-[10px] ${EXECUTION_MODE_COLORS[mode]}`}>
            {mode}
          </span>
        </label>
      ))}
    </div>
  );
}

// ============================================================================
// Tests Tab
// ============================================================================

function TestsTab({
  scenarioSlug,
  repoId,
  testGenieAvailable,
  agentManagerAvailable,
  onAttachToAgent,
}: {
  scenarioSlug: string;
  repoId?: string | null;
  testGenieAvailable: boolean;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
}) {
  const testExecutions = useTestExecutions(scenarioSlug, testGenieAvailable, repoId);
  const triggerTest = useTriggerTestExecution(repoId);
  const [expandedPhase, setExpandedPhase] = useState<string | null>(null);

  if (!testGenieAvailable) {
    return <ServiceUnavailableBanner name="Test Genie" message="Start the test-genie scenario to run automated tests" />;
  }

  const isRunning = triggerTest.isPending;
  const executions = testExecutions.data?.items ?? [];
  const latest = executions[0] as TestExecutionResult | undefined;
  const history = executions.slice(1, 6);

  return (
    <div className="space-y-4">
      <MutationErrorBanner error={triggerTest.error} onDismiss={() => triggerTest.reset()} />
      {/* Run Tests button */}
      <div className="flex items-center justify-between">
        <h3 className="text-xs font-medium text-slate-400">Test Execution</h3>
        <Button
          variant="outline"
          size="sm"
          onClick={() => triggerTest.mutate({ scenarioName: scenarioSlug })}
          disabled={isRunning}
          className="h-7 text-xs gap-1"
        >
          {isRunning ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <Play className="h-3 w-3" />
          )}
          Run Tests
        </Button>
      </div>

      {/* In-progress banner */}
      {isRunning && (
        <div className="flex items-center gap-2 px-3 py-2 bg-blue-950/50 border border-blue-900/50 rounded-lg text-blue-300 text-xs">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Running tests...
        </div>
      )}

      {/* Loading state */}
      {testExecutions.isLoading ? (
        <div className="space-y-3">
          <div className="h-24 animate-pulse bg-slate-800 rounded" />
          <div className="h-16 animate-pulse bg-slate-800 rounded" />
        </div>
      ) : !latest ? (
        <div className="flex flex-col items-center justify-center py-8 text-slate-500">
          <p className="text-sm">No test results yet</p>
          <p className="text-xs mt-1">Run tests to see results here</p>
        </div>
      ) : (
        <>
          {/* Latest execution card */}
          <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                {latest.success ? (
                  <CheckCircle2 className="h-4 w-4 text-emerald-400" />
                ) : (
                  <XCircle className="h-4 w-4 text-red-400" />
                )}
                <span className={`text-sm font-medium ${latest.success ? "text-emerald-300" : "text-red-300"}`}>
                  {latest.success ? "All Passed" : "Failed"}
                </span>
              </div>
              <span className="text-[11px] text-slate-500">
                {latest.completedAt ? new Date(latest.completedAt).toLocaleString() : latest.startedAt}
              </span>
            </div>
            <div className="flex gap-4 text-xs text-slate-400">
              <span>{latest.phaseSummary.total} total</span>
              <span className="text-emerald-400">{latest.phaseSummary.passed} passed</span>
              {latest.phaseSummary.failed > 0 && (
                <span className="text-red-400">{latest.phaseSummary.failed} failed</span>
              )}
              <span>{formatDuration(latest.phaseSummary.durationSeconds)}</span>
            </div>
            {latest.preset && (
              <div className="text-[11px] text-slate-500">Preset: {latest.preset}</div>
            )}
            {latest.warnings && latest.warnings.length > 0 && (
              <div className="space-y-1">
                {latest.warnings.map((w, i) => (
                  <div key={i} className="flex items-start gap-1.5 text-[11px] text-amber-400">
                    <AlertTriangle className="h-3 w-3 mt-0.5 shrink-0" />
                    {w}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Phase list */}
          <div className="space-y-1">
            <h4 className="text-xs font-medium text-slate-400 mb-2">Phases</h4>
            {latest.phases.map((phase) => (
              <div key={phase.name} className="flex items-start gap-1">
                <div className="flex-1 min-w-0">
                  <PhaseRow
                    phase={phase}
                    expanded={expandedPhase === phase.name}
                    onToggle={() => setExpandedPhase(expandedPhase === phase.name ? null : phase.name)}
                  />
                </div>
                {agentManagerAvailable && phase.status === "failed" && onAttachToAgent && (
                  <div className="mt-2 shrink-0">
                    <AttachToAgentButton onClick={() => { const items = testFailureContextItems([phase]); if (items[0]) onAttachToAgent(items[0]); }} />
                  </div>
                )}
              </div>
            ))}
          </div>

          {/* History */}
          {history.length > 0 && (
            <div>
              <h4 className="text-xs font-medium text-slate-400 mb-2">Recent History</h4>
              <div className="space-y-1">
                {history.map((exec) => (
                  <div
                    key={exec.executionId}
                    className="flex items-center justify-between px-3 py-2 rounded bg-slate-900/50 border border-slate-800/50"
                  >
                    <div className="flex items-center gap-2">
                      {exec.success ? (
                        <div className="h-2 w-2 rounded-full bg-emerald-500" />
                      ) : (
                        <div className="h-2 w-2 rounded-full bg-red-500" />
                      )}
                      <span className="text-[11px] text-slate-400">
                        {exec.completedAt ? new Date(exec.completedAt).toLocaleString() : exec.startedAt}
                      </span>
                    </div>
                    <div className="flex gap-3 text-[11px] text-slate-500">
                      {exec.preset && <span>{exec.preset}</span>}
                      <span>{formatDuration(exec.phaseSummary.durationSeconds)}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ============================================================================
// Phase Row
// ============================================================================

function PhaseRow({
  phase,
  expanded,
  onToggle,
}: {
  phase: TestPhaseResult;
  expanded: boolean;
  onToggle: () => void;
}) {
  const hasDetails = Boolean(phase.error || phase.remediation || (phase.observations && phase.observations.length > 0));

  return (
    <div className="rounded border border-slate-800/50 bg-slate-900/30">
      <button
        type="button"
        onClick={hasDetails ? onToggle : undefined}
        className={`w-full flex items-center justify-between px-3 py-2 text-xs ${hasDetails ? "cursor-pointer hover:bg-slate-800/30" : "cursor-default"}`}
      >
        <div className="flex items-center gap-2">
          {hasDetails ? (
            expanded ? <ChevronDown className="h-3 w-3 text-slate-500" /> : <ChevronRight className="h-3 w-3 text-slate-500" />
          ) : (
            <div className="w-3" />
          )}
          <div className={`h-2 w-2 rounded-full ${phase.status === "passed" ? "bg-emerald-500" : "bg-red-500"}`} />
          <span className="text-slate-200">{phase.name}</span>
        </div>
        <span className="text-slate-500">{formatDuration(phase.durationSeconds)}</span>
      </button>

      {expanded && hasDetails && (
        <div className="px-3 pb-3 pt-1 border-t border-slate-800/30 space-y-2">
          {phase.error && (
            <div className="text-[11px] text-red-400 bg-red-950/30 rounded px-2 py-1.5">
              {phase.error}
            </div>
          )}
          {phase.classification && (
            <div className="text-[11px] text-slate-400">
              <span className="text-slate-500">Classification:</span> {phase.classification}
            </div>
          )}
          {phase.remediation && (
            <div className="text-[11px] text-amber-300 bg-amber-950/20 rounded px-2 py-1.5">
              {phase.remediation}
            </div>
          )}
          {phase.observations && phase.observations.length > 0 && (
            <div className="space-y-1">
              {phase.observations.map((obs, i) => (
                <div key={i} className="text-[11px] text-slate-400 flex gap-1.5">
                  {obs.icon && <span>{obs.icon}</span>}
                  {obs.prefix && (
                    <span className={`font-medium ${
                      obs.prefix === "ERROR" ? "text-red-400" :
                      obs.prefix === "WARNING" ? "text-amber-400" :
                      obs.prefix === "SUCCESS" ? "text-emerald-400" : "text-slate-400"
                    }`}>
                      {obs.prefix}:
                    </span>
                  )}
                  <span>{obs.text}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Code Quality Tab
// ============================================================================

function CodeQualityTab({
  scenarioSlug,
  repoId,
  tidinessAvailable,
  fileStats,
  agentManagerAvailable,
  onAttachToAgent,
}: {
  scenarioSlug: string;
  repoId?: string | null;
  tidinessAvailable: boolean;
  fileStats?: RepoFileStats;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
}) {
  const [view, setView] = useState<"changed" | "scenario">("changed");
  const tidinessScore = useTidinessScore(scenarioSlug, tidinessAvailable, repoId);
  const tidinessIssues = useTidinessIssues(scenarioSlug, undefined, tidinessAvailable, repoId);
  const tidinessStaleness = useTidinessStaleness(scenarioSlug, tidinessAvailable, repoId);
  const triggerScan = useTriggerTidinessScan(repoId);

  // Get changed file paths (scenario-relative)
  const changedFiles = useMemo(() => {
    if (!fileStats) return [];
    const prefix = `scenarios/${scenarioSlug}/`;
    return Object.keys(fileStats).map(p =>
      p.startsWith(prefix) ? p.slice(prefix.length) : p
    );
  }, [fileStats, scenarioSlug]);

  // Filter issues to changed files
  const changedFileIssues = useMemo(() => {
    if (!tidinessIssues.data || changedFiles.length === 0) return [];
    return tidinessIssues.data.filter(issue =>
      changedFiles.includes(issue.file_path)
    );
  }, [tidinessIssues.data, changedFiles]);

  // Group issues by file
  const issuesByFile = useMemo(() => {
    const map = new Map<string, TidinessIssue[]>();
    for (const issue of changedFileIssues) {
      const existing = map.get(issue.file_path) ?? [];
      existing.push(issue);
      map.set(issue.file_path, existing);
    }
    return map;
  }, [changedFileIssues]);

  if (!tidinessAvailable) {
    return <ServiceUnavailableBanner name="Tidiness Manager" message="Start the tidiness-manager scenario to view code quality data" />;
  }

  // Use staleness as source of truth for "has been scanned" — more reliable than score.last_scan
  const stalenessLoaded = !tidinessStaleness.isLoading;
  const neverScanned = stalenessLoaded && (
    !tidinessStaleness.data?.last_scan_at &&
    tidinessStaleness.data?.stale_reason?.includes("no scans")
  );
  const isScanning = triggerScan.isPending;
  const scanResult = triggerScan.data;

  // Never-scanned state: show a prominent CTA
  if (neverScanned) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <p className="text-sm font-medium text-slate-400">No scan data yet</p>
        <p className="text-xs mt-2 text-slate-600 text-center max-w-sm">
          Run a scan to analyze code quality for this scenario
        </p>
        <Button
          variant="outline"
          size="sm"
          onClick={() => triggerScan.mutate({
            scenarioName: scenarioSlug,
            incremental: false,
          })}
          disabled={isScanning}
          className="mt-4 gap-1.5"
        >
          {isScanning ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Play className="h-3.5 w-3.5" />
          )}
          {isScanning ? "Scanning..." : "Run Scan"}
        </Button>
        {scanResult && (
          <ScanResultSummary result={scanResult} />
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <MutationErrorBanner error={triggerScan.error} onDismiss={() => triggerScan.reset()} />
      {/* Staleness banner */}
      {tidinessStaleness.data?.is_stale && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-950/30 border border-amber-900/40">
          <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 shrink-0" />
          <p className="text-xs text-amber-300">
            {formatStalenessMessage(tidinessStaleness.data)}
          </p>
        </div>
      )}

      {/* Last scan info */}
      {tidinessStaleness.data?.last_scan_at && !tidinessStaleness.data?.is_stale && (
        <p className="text-[11px] text-slate-500">
          Last scanned {formatRelativeTime(tidinessStaleness.data.last_scan_at)}
          {tidinessScore.data?.metrics?.total_files ? ` · ${tidinessScore.data.metrics.total_files} files` : ""}
        </p>
      )}

      {/* View toggle + scan button */}
      <div className="flex items-center justify-between">
        <div className="flex gap-1">
          <button
            type="button"
            onClick={() => setView("changed")}
            className={`px-2.5 py-1 rounded-full text-xs transition-colors ${
              view === "changed"
                ? "bg-blue-600 text-white"
                : "bg-slate-800 text-slate-400 hover:text-slate-200"
            }`}
          >
            Changed Files
          </button>
          <button
            type="button"
            onClick={() => setView("scenario")}
            className={`px-2.5 py-1 rounded-full text-xs transition-colors ${
              view === "scenario"
                ? "bg-blue-600 text-white"
                : "bg-slate-800 text-slate-400 hover:text-slate-200"
            }`}
          >
            Full Scenario
          </button>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => triggerScan.mutate({
            scenarioName: scenarioSlug,
            incremental: true,
          })}
          disabled={isScanning}
          className="h-7 text-xs gap-1"
        >
          {isScanning ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <RefreshCw className="h-3 w-3" />
          )}
          Scan
        </Button>
      </div>

      {/* In-progress banner */}
      {isScanning && (
        <div className="flex items-center gap-2 px-3 py-2 bg-blue-950/50 border border-blue-900/50 rounded-lg text-blue-300 text-xs">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Scanning...
        </div>
      )}

      {/* Scan result summary */}
      {scanResult && !isScanning && (
        <ScanResultSummary result={scanResult} />
      )}

      {view === "changed" ? (
        <ChangedFilesView
          issues={changedFileIssues}
          issuesByFile={issuesByFile}
          changedFiles={changedFiles}
          scoreData={tidinessScore.data}
          isLoading={tidinessIssues.isLoading}
          agentManagerAvailable={agentManagerAvailable}
          onAttachToAgent={onAttachToAgent}
        />
      ) : (
        <ScenarioWideView
          scoreData={tidinessScore.data}
          isLoading={tidinessScore.isLoading}
          agentManagerAvailable={agentManagerAvailable}
          onAttachToAgent={onAttachToAgent}
        />
      )}
    </div>
  );
}

// ============================================================================
// Rules Tab
// ============================================================================

function RulesTab({
  scenarioSlug,
  repoId,
  auditorAvailable,
  agentManagerAvailable,
  onAttachToAgent,
}: {
  scenarioSlug: string;
  repoId?: string | null;
  auditorAvailable: boolean;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
}) {
  const startCheck = useStartAuditorCheck(repoId);
  const [jobId, setJobId] = useState<string | null>(null);
  const jobStatus = useAuditorJobStatus(jobId, repoId);
  const [expandedViolation, setExpandedViolation] = useState<string | null>(null);

  if (!auditorAvailable) {
    return <ServiceUnavailableBanner name="Scenario Auditor" message="Start the scenario-auditor scenario to view standards compliance and rule violations" />;
  }

  const handleRunCheck = () => {
    startCheck.mutate({ scenarioName: scenarioSlug }, {
      onSuccess: (data) => setJobId(data.job_id),
    });
  };

  const status = jobStatus.data?.status;
  const isRunning = status === "running" || status === "pending" || startCheck.isPending;
  const isCompleted = status === "completed";
  const isFailed = status === "failed";
  const result = jobStatus.data?.result;
  const violations = result?.violations ?? [];
  const summary = result?.summary;

  // No job started yet
  if (!jobId && !startCheck.data) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <p className="text-sm font-medium text-slate-400">No standards check yet</p>
        <p className="text-xs mt-2 text-slate-600 text-center max-w-sm">
          Run a standards check to analyze rule compliance for this scenario
        </p>
        <Button
          variant="outline"
          size="sm"
          onClick={handleRunCheck}
          disabled={startCheck.isPending}
          className="mt-4 gap-1.5"
        >
          {startCheck.isPending ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Play className="h-3.5 w-3.5" />
          )}
          {startCheck.isPending ? "Starting..." : "Run Check"}
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <MutationErrorBanner error={startCheck.error} onDismiss={() => startCheck.reset()} />

      {/* Progress / summary bar */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 text-xs">
          {isRunning && (
            <span className="flex items-center gap-1 text-blue-400">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              {jobStatus.data?.message || "Running..."}
            </span>
          )}
          {isCompleted && (
            <>
              {violations.length === 0 ? (
                <span className="flex items-center gap-1 text-emerald-400">
                  <CheckCircle2 className="h-3.5 w-3.5" /> No violations
                </span>
              ) : (
                <span className="flex items-center gap-1 text-red-400">
                  <XCircle className="h-3.5 w-3.5" /> {violations.length} violation{violations.length !== 1 ? "s" : ""}
                </span>
              )}
              {result?.files_scanned != null && (
                <span className="text-slate-500">{result.files_scanned} files scanned</span>
              )}
            </>
          )}
          {isFailed && (
            <span className="flex items-center gap-1 text-red-400">
              <AlertTriangle className="h-3.5 w-3.5" /> {jobStatus.data?.error || "Check failed"}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          {agentManagerAvailable && onAttachToAgent && violations.length > 0 && (
            <AttachToAgentButton onClick={() => onAttachToAgent(rulesSummaryContextItem(violations, summary))} />
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={handleRunCheck}
            disabled={isRunning}
            className="h-7 text-xs gap-1"
          >
            {isRunning ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <RefreshCw className="h-3 w-3" />
            )}
            {isRunning ? "Running..." : "Re-run"}
          </Button>
        </div>
      </div>

      {/* Progress bar while running */}
      {isRunning && jobStatus.data && jobStatus.data.total_files > 0 && (
        <div className="space-y-1">
          <div className="h-1.5 w-full bg-slate-800 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 rounded-full transition-all duration-300"
              style={{ width: `${Math.round((jobStatus.data.processed_files / jobStatus.data.total_files) * 100)}%` }}
            />
          </div>
          <p className="text-[11px] text-slate-500">
            {jobStatus.data.processed_files}/{jobStatus.data.total_files} files
            {jobStatus.data.current_file && ` · ${jobStatus.data.current_file}`}
          </p>
        </div>
      )}

      {/* Violations list */}
      {isCompleted && violations.length > 0 && (
        <div className="space-y-1">
          {violations.map((v, i) => {
            const key = v.id || `${v.type}-${i}`;
            const isExpanded = expandedViolation === key;
            return (
              <div key={key} className={`rounded border ${
                v.severity === "high" ? "border-red-900/30 bg-red-950/20" :
                v.severity === "medium" ? "border-amber-900/30 bg-amber-950/20" :
                "border-slate-800/30 bg-slate-900/20"
              }`}>
                <div className="flex items-center">
                  <button
                    type="button"
                    onClick={() => setExpandedViolation(isExpanded ? null : key)}
                    className="flex-1 flex items-center gap-2 px-3 py-2 text-xs cursor-pointer hover:bg-slate-800/20"
                  >
                    {isExpanded ? (
                      <ChevronDown className="h-3 w-3 text-slate-500" />
                    ) : (
                      <ChevronRight className="h-3 w-3 text-slate-500" />
                    )}
                    <div className={`h-1.5 w-1.5 rounded-full shrink-0 ${
                      v.severity === "high" ? "bg-red-500" :
                      v.severity === "medium" ? "bg-amber-500" :
                      v.severity === "low" ? "bg-yellow-500" : "bg-blue-500"
                    }`} />
                    <span className="text-slate-200 font-medium truncate">{v.title}</span>
                    <span className="text-slate-600 shrink-0">{v.type}</span>
                    {v.source && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-slate-800 text-slate-400 shrink-0">{v.source}</span>
                    )}
                  </button>
                  {agentManagerAvailable && onAttachToAgent && (
                    <div className="pr-2">
                      <AttachToAgentButton onClick={() => {
                        const items = ruleViolationContextItems([v]);
                        if (items[0]) onAttachToAgent(items[0]);
                      }} />
                    </div>
                  )}
                </div>
                {isExpanded && (
                  <div className="px-3 pb-3 pt-1 border-t border-slate-800/30 space-y-2 text-[11px]">
                    {v.description && <p className="text-slate-300">{v.description}</p>}
                    {v.file_path && (
                      <div className="text-slate-400">
                        <span className="text-slate-500">File:</span>{" "}
                        <code className="text-slate-300">{v.file_path}{v.line_number ? `:${v.line_number}` : ""}</code>
                      </div>
                    )}
                    {v.code_snippet && (
                      <pre className="p-2 rounded bg-slate-900 border border-slate-800 text-slate-300 overflow-x-auto text-[10px]">{v.code_snippet}</pre>
                    )}
                    {v.recommendation && (
                      <div className="text-slate-400">
                        <span className="text-slate-500">Recommendation:</span> {v.recommendation}
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Completed with no violations */}
      {isCompleted && violations.length === 0 && (
        <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-emerald-400" />
            <span className="text-xs text-emerald-300 font-medium">All rules passed — no violations found</span>
          </div>
        </div>
      )}
    </div>
  );
}

function ChangedFilesView({
  issues,
  issuesByFile,
  changedFiles,
  scoreData,
  isLoading,
  agentManagerAvailable,
  onAttachToAgent,
}: {
  issues: TidinessIssue[];
  issuesByFile: Map<string, TidinessIssue[]>;
  changedFiles: string[];
  scoreData?: { score: number; violations: number } | null;
  isLoading: boolean;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
}) {
  const [expandedFile, setExpandedFile] = useState<string | null>(null);

  if (isLoading) {
    return (
      <div className="space-y-3">
        <div className="h-16 animate-pulse bg-slate-800 rounded" />
        <div className="h-16 animate-pulse bg-slate-800 rounded" />
      </div>
    );
  }

  if (changedFiles.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-slate-500">
        <p className="text-sm">No changed files to analyze</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {/* Scenario-wide score badge for context */}
      {scoreData && (
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <span>Scenario score:</span>
          <span className={`font-medium ${
            scoreData.score >= 70 ? "text-emerald-400" :
            scoreData.score >= 40 ? "text-amber-400" : "text-red-400"
          }`}>
            {Math.round(scoreData.score)}/100
          </span>
        </div>
      )}

      {issues.length === 0 ? (
        <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-emerald-400" />
            <span className="text-xs text-emerald-300 font-medium">No issues in changed files</span>
          </div>
        </div>
      ) : (
        <div className="space-y-1">
          {Array.from(issuesByFile.entries()).map(([filePath, fileIssues]) => (
            <div key={filePath} className="rounded border border-slate-800/50 bg-slate-900/30">
              <div className="flex items-center">
                <button
                  type="button"
                  onClick={() => setExpandedFile(expandedFile === filePath ? null : filePath)}
                  className="flex-1 flex items-center gap-2 px-3 py-2 text-xs cursor-pointer hover:bg-slate-800/30"
                >
                  {expandedFile === filePath ? (
                    <ChevronDown className="h-3 w-3 text-slate-500" />
                  ) : (
                    <ChevronRight className="h-3 w-3 text-slate-500" />
                  )}
                  <code className="text-slate-200">{filePath}</code>
                  <span className="text-slate-500">({fileIssues.length} issue{fileIssues.length !== 1 ? "s" : ""})</span>
                </button>
                {agentManagerAvailable && onAttachToAgent && (
                  <div className="pr-2">
                    <AttachToAgentButton onClick={() => {
                      for (const item of codeQualityContextItems(fileIssues)) {
                        onAttachToAgent(item);
                      }
                    }} />
                  </div>
                )}
              </div>
              {expandedFile === filePath && (
                <div className="px-3 pb-3 pt-1 border-t border-slate-800/30 space-y-1.5">
                  {fileIssues.map(issue => (
                    <div key={issue.id} className="flex items-start gap-2 text-[11px]">
                      <div className={`h-1.5 w-1.5 rounded-full mt-1.5 shrink-0 ${
                        issue.severity === "critical" || issue.severity === "high" ? "bg-red-500" :
                        issue.severity === "medium" ? "bg-amber-500" : "bg-blue-500"
                      }`} />
                      <div className="min-w-0 flex-1">
                        <span className="text-slate-500">{issue.category}:</span>{" "}
                        <span className="text-slate-300">{issue.title}</span>
                        {issue.line_number != null && (
                          <span className="text-slate-600 ml-1">L:{issue.line_number}</span>
                        )}
                      </div>
                      {agentManagerAvailable && onAttachToAgent && (
                        <AttachToAgentButton onClick={() => { const items = codeQualityContextItems([issue]); if (items[0]) onAttachToAgent(items[0]); }} />
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ScenarioWideView({
  scoreData,
  isLoading,
  agentManagerAvailable,
  onAttachToAgent,
}: {
  scoreData?: {
    score: number;
    violations: number;
    breakdown?: {
      lint_issues: number;
      type_issues: number;
      long_files: number;
      complex_functions: number;
      tech_debt_markers: number;
      duplication_issues: number;
    };
    metrics?: {
      total_files: number;
      total_lines: number;
      avg_file_length: number;
      max_complexity: number;
      avg_complexity: number;
      duplication_pct: number;
    };
  } | null;
  isLoading: boolean;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
}) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        <div className="h-24 animate-pulse bg-slate-800 rounded" />
        <div className="h-32 animate-pulse bg-slate-800 rounded" />
      </div>
    );
  }

  if (!scoreData) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-slate-500">
        <p className="text-sm">No quality data available</p>
        <p className="text-xs mt-1">Run a scan to generate quality metrics</p>
      </div>
    );
  }

  const scoreColor = scoreData.score >= 70 ? "bg-emerald-500" :
    scoreData.score >= 40 ? "bg-amber-500" : "bg-red-500";
  const scoreTextColor = scoreData.score >= 70 ? "text-emerald-300" :
    scoreData.score >= 40 ? "text-amber-300" : "text-red-300";

  return (
    <div className="space-y-4">
      {/* Score bar */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4 space-y-3">
        <div className="flex items-center justify-between">
          <span className={`text-2xl font-bold ${scoreTextColor}`}>
            {Math.round(scoreData.score)}
          </span>
          <span className="text-xs text-slate-500">/100</span>
        </div>
        <div className="w-full h-2 bg-slate-800 rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all ${scoreColor}`}
            style={{ width: `${Math.min(100, Math.max(0, scoreData.score))}%` }}
          />
        </div>
        <div className="flex justify-between text-xs">
          <span className="text-slate-400">Violations</span>
          <div className="flex items-center gap-2">
            <span className="text-slate-200">{scoreData.violations}</span>
            {agentManagerAvailable && onAttachToAgent && scoreData.violations > 0 && (
              <AttachToAgentButton onClick={() => onAttachToAgent(scenarioQualityContextItem(scoreData))} />
            )}
          </div>
        </div>
      </div>

      {/* Breakdown */}
      {scoreData.breakdown && (
        <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
          <h4 className="text-xs font-medium text-slate-400 mb-3">Breakdown</h4>
          <div className="space-y-2">
            {([
              ["Lint issues", scoreData.breakdown.lint_issues],
              ["Type issues", scoreData.breakdown.type_issues],
              ["Long files", scoreData.breakdown.long_files],
              ["Complex functions", scoreData.breakdown.complex_functions],
              ["Tech debt markers", scoreData.breakdown.tech_debt_markers],
              ["Duplication", scoreData.breakdown.duplication_issues],
            ] as const).map(([label, value]) => (
              <div key={label} className="flex justify-between text-xs">
                <span className="text-slate-400">{label}</span>
                <span className={value > 0 ? "text-slate-200" : "text-slate-600"}>{value}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Metrics */}
      {scoreData.metrics && (
        <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
          <h4 className="text-xs font-medium text-slate-400 mb-3">Metrics</h4>
          <div className="space-y-2">
            {([
              ["Total files", String(scoreData.metrics.total_files)],
              ["Total lines", scoreData.metrics.total_lines.toLocaleString()],
              ["Avg file length", String(Math.round(scoreData.metrics.avg_file_length))],
              ["Max complexity", String(scoreData.metrics.max_complexity)],
              ["Avg complexity", scoreData.metrics.avg_complexity.toFixed(1)],
              ["Duplication", `${scoreData.metrics.duplication_pct.toFixed(1)}%`],
            ] as const).map(([label, value]) => (
              <div key={label} className="flex justify-between text-xs">
                <span className="text-slate-400">{label}</span>
                <span className="text-slate-200">{value}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Helpers
// ============================================================================

function sanitizePagePath(pagePath: string): string {
  if (pagePath === "/" || pagePath === "") return "_root_";
  let s = pagePath.startsWith("/") ? pagePath.slice(1) : pagePath;
  s = s.endsWith("/") ? s.slice(0, -1) : s;
  s = s.replace(/\//g, "_");
  return "_" + s + "_";
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return secs > 0 ? `${mins}m ${secs}s` : `${mins}m`;
}

function formatRelativeTime(isoString: string): string {
  const date = new Date(isoString);
  const now = Date.now();
  const diffMs = now - date.getTime();
  if (diffMs < 0) return "just now";
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return "just now";
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return `${diffDay}d ago`;
}

function formatStalenessMessage(staleness: TidinessStalenessInfo): string {
  const parts: string[] = [];
  if (staleness.stale_reason) {
    parts.push(staleness.stale_reason);
  } else {
    parts.push("Quality data may be stale");
  }
  if (staleness.last_scan_at) {
    parts.push(`Last scan: ${formatRelativeTime(staleness.last_scan_at)}`);
  }
  if (staleness.modified_files && staleness.modified_files > 0 && !staleness.stale_reason?.includes("file")) {
    parts.push(`${staleness.modified_files} file${staleness.modified_files !== 1 ? "s" : ""} changed`);
  }
  return parts.join(" · ");
}

function ScanResultSummary({ result }: { result: TidinessLightScanResult }) {
  const durationSec = (result.duration_ms / 1000).toFixed(1);
  const totalIssues = result.lint_issues + result.type_issues;
  return (
    <div className="flex items-center gap-2 px-3 py-2 bg-slate-800/50 border border-slate-700/50 rounded-lg text-xs text-slate-300 mt-2">
      <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400 shrink-0" />
      <span>
        Scanned {result.total_files} files ({result.total_lines.toLocaleString()} lines) in {durationSec}s
        {" — "}
        {totalIssues === 0 ? (
          <span className="text-emerald-400">no issues found</span>
        ) : (
          <span className="text-amber-400">
            {result.lint_issues} lint, {result.type_issues} type, {result.long_files_count} long file{result.long_files_count !== 1 ? "s" : ""}
          </span>
        )}
      </span>
    </div>
  );
}
