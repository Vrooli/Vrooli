import { useState, useEffect, useCallback, useMemo } from "react";
import { ClipboardCheck, Loader2, ChevronDown, Camera } from "lucide-react";
import { Card, CardHeader, CardTitle } from "./ui/card";
import { useTriggerVisualCapture, useCapabilities } from "../lib/hooks";
import { getCapturePresets } from "../lib/api";
import type { CapturePreset, RepoFileStats, AgentContextItem } from "../lib/api";
import { AggregateMetricsContent } from "./ChangeMetricsModal";
import { AgentTab } from "./AgentTab";
import { AIProvenanceTab } from "./AIProvenanceTab";
import type { ScenarioReviewState, DeepPartial } from "../hooks/useScenarioReviewState";
import { ScenarioPickerModal } from "./ScenarioPickerModal";
import { OverviewTab } from "./ScenarioReviewPanelOverview";
import { ScreenshotsTab } from "./ScenarioReviewPanelScreenshots";
import { WorkflowsTab } from "./ScenarioReviewPanelWorkflows";
import { TestsTab } from "./ScenarioReviewPanelTests";
import { BaselinesTab } from "../features/baselines/BaselinesTab";
import type { ReviewTab } from "../hooks/useUrlState";

type Tab = ReviewTab;

interface ScenarioReviewPanelProps {
  scenarioSlug: string;
  repoId?: string | null;
  fileStats?: RepoFileStats;
  onChangeScenario: (slug: string) => void;
  activeTab?: Tab;
  onActiveTabChange?: (tab: Tab) => void;
  agentRunId?: string | null;
  onAgentRunIdChange?: (id: string | null) => void;
  scenarioState?: ScenarioReviewState;
  onScenarioStateChange?: (patch: DeepPartial<ScenarioReviewState>) => void;
  isMobile?: boolean;
}

export function ScenarioReviewPanel({ scenarioSlug, repoId, fileStats, onChangeScenario, activeTab: controlledTab, onActiveTabChange, agentRunId, onAgentRunIdChange, isMobile }: ScenarioReviewPanelProps) {
  const [isPickerOpen, setIsPickerOpen] = useState(false);
  const [internalTab, setInternalTab] = useState<Tab>("overview");
  const activeTab = controlledTab ?? internalTab;
  const setActiveTab = onActiveTabChange ?? setInternalTab;
  const [evidenceTarget, setEvidenceTarget] = useState<{ runId: string; phase: string } | null>(null);
  const openExactTestPhase = useCallback((runId: string, phase: string) => {
    setEvidenceTarget({ runId, phase });
    setActiveTab("tests");
  }, [setActiveTab]);

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
  }, [setActiveTab]);
  const removeAgentContext = useCallback((id: string) => {
    setAgentContext((prev) => prev.filter((c) => c.id !== id));
  }, []);
  const clearAgentContext = useCallback(() => setAgentContext([]), []);

  const [presetConfig, setPresetConfigState] = useState<CapturePreset[]>(() => getCapturePresets(scenarioSlug));
  useEffect(() => {
    setPresetConfigState(getCapturePresets(scenarioSlug));
  }, [scenarioSlug]);
  const isCapturing = triggerCapture.isPending;
  const handleCapture = useCallback(() => triggerCapture.mutate({ scenarioSlug, presets: presetConfig }), [triggerCapture, scenarioSlug, presetConfig]);

  const tabLabels: Record<Tab, string> = {
    overview: "Overview",
    baselines: "Baselines",
    metrics: "Metrics",
    screenshots: "Screenshots",
    workflows: "Workflows",
    tests: "Tests",
    "ai-provenance": "AI Changes",
    agent: "Agent",
  };

  const workspaceSandboxAvailable = capabilities.data?.capabilities?.some(
    c => c.id === "workspace-sandbox" && c.status === "available"
  ) ?? false;

  const visibleTabs = (Object.keys(tabLabels) as Tab[]).filter(
    tab => {
      if (tab === "metrics") return Boolean(scenarioFileStats);
      if (tab === "ai-provenance") return workspaceSandboxAvailable;
      if (tab === "agent") return agentManagerAvailable;
      return true;
    }
  );

  useEffect(() => {
    if (visibleTabs.length > 0 && !visibleTabs.includes(activeTab)) {
      setActiveTab("overview");
    }
  }, [visibleTabs, activeTab, setActiveTab]);

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
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          basAvailable={basAvailable}
          testGenieAvailable={testGenieAvailable}
          tidinessAvailable={tidinessAvailable}
          isCapturing={isCapturing}
          onCapture={handleCapture}
          fileStats={scenarioFileStats}
          agentManagerAvailable={agentManagerAvailable}
          onAttachToAgent={addAgentContext}
          onOpenBaselines={() => setActiveTab("baselines")}
        />
      ) : activeTab === "baselines" ? (
        <BaselinesTab
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          onOpenTab={(tab) => setActiveTab(tab)}
          agentManagerAvailable={agentManagerAvailable}
          onAttachToAgent={addAgentContext}
        />
      ) : activeTab === "metrics" ? (
        scenarioFileStats ? <AggregateMetricsContent fileStats={scenarioFileStats} /> : null
      ) : activeTab === "screenshots" ? (
        <ScreenshotsTab
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          testGenieAvailable={testGenieAvailable}
          agentManagerAvailable={agentManagerAvailable}
          onAttachToAgent={addAgentContext}
          onOpenBaselines={() => setActiveTab("baselines")}
          onOpenTests={openExactTestPhase}
        />
      ) : activeTab === "workflows" ? (
        <WorkflowsTab
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          testGenieAvailable={testGenieAvailable}
          agentManagerAvailable={agentManagerAvailable}
          onAttachToAgent={addAgentContext}
          onOpenBaselines={() => setActiveTab("baselines")}
          onOpenTests={openExactTestPhase}
        />
      ) : activeTab === "tests" ? (
        <TestsTab
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          testGenieAvailable={testGenieAvailable}
          agentManagerAvailable={agentManagerAvailable}
          onAttachToAgent={addAgentContext}
          onOpenBaselines={() => setActiveTab("baselines")}
          target={evidenceTarget}
        />
      ) : activeTab === "ai-provenance" ? (
        <AIProvenanceTab repoId={repoId} />
      ) : (
        <AgentTab
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          agentManagerAvailable={agentManagerAvailable}
          workspaceSandboxAvailable={workspaceSandboxAvailable}
          contextItems={agentContext}
          onAddContext={addAgentContext}
          onRemoveContext={removeAgentContext}
          onClearContext={clearAgentContext}
          testGenieAvailable={testGenieAvailable}
          tidinessAvailable={tidinessAvailable}
          auditorAvailable={auditorAvailable}
          visualCaptureAvailable={basAvailable}
          fileStats={scenarioFileStats}
          activeRunId={agentRunId}
          onActiveRunIdChange={onAgentRunIdChange}
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
        </CardTitle>
        <div className="flex items-center gap-1 shrink-0">
          {basAvailable && activeTab !== "agent" && (
            <button
              type="button"
              className={`${isMobile ? "h-11 w-11 touch-target" : "h-7 px-2"} inline-flex items-center justify-center gap-1 rounded text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800/60 transition-colors`}
              onClick={handleCapture}
              disabled={isCapturing}
              title="Capture current screenshots"
            >
              {isCapturing ? (
                <Loader2 className={`${isMobile ? "h-4 w-4" : "h-3.5 w-3.5"} animate-spin`} />
              ) : (
                <Camera className={isMobile ? "h-4 w-4" : "h-3.5 w-3.5"} />
              )}
            </button>
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
