import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { Book, List, Monitor, Zap, Folder, Shield, Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { GeneratorPage } from "./pages";
import { ScenarioInventory } from "./components/scenario-inventory";
import { DocsPanel } from "./components/docs/DocsPanel";
import { SigningPage } from "./components/signing";
import { SpawnAgentButton } from "./components/state/SpawnAgentButton";
import { ErrorBoundary, SectionErrorBoundary } from "./components/ui/ErrorBoundary";
import type { ScenarioDesktopStatus } from "./components/scenario-inventory/types";
import { getPipelineStatus } from "./lib/api";
import type { FormState, SmokeTestStageDetails } from "./lib/api";
import { usePipelineStore } from "./store";
import { mapPipelineToUiStatus, type UiBuildStatus } from "./domain/build";
import { useUrlState, type ViewMode } from "./hooks/useUrlState";
import { loadGeneratorAppState, saveGeneratorAppState } from "./lib/draftStorage";
import { cn } from "./lib/utils";
import { RecordsManager } from "./components/scenario-inventory/RecordsManager";
import { useScenarioState } from "./hooks";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1
    }
  }
});

function AppContent() {
  const storedState = useMemo(() => loadGeneratorAppState(), []);

  // URL state hook handles view, scenario, and doc synchronization with URL
  const urlState = useUrlState({
    defaultView: (storedState?.viewMode as ViewMode) ?? "inventory",
  });

  // Use URL state, with stored state as fallback for scenario
  const [selectedScenarioName, setSelectedScenarioNameState] = useState(
    urlState.initialParams.scenario ?? storedState?.selectedScenarioName ?? ""
  );
  const [selectionSource, setSelectionSource] = useState<"inventory" | "manual" | null>(
    urlState.initialParams.scenario ? "manual" : storedState?.selectionSource ?? null
  );
  const [docPath, setDocPathState] = useState<string | null>(
    urlState.initialParams.doc ?? storedState?.docPath ?? null
  );
  const [viewMode, setViewModeState] = useState<ViewMode>(
    urlState.initialParams.view ?? (storedState?.viewMode as ViewMode) ?? "inventory"
  );

  const [selectedTemplate, setSelectedTemplate] = useState(storedState?.selectedTemplate || "basic");
  // Wrapper build state - will be initialized from server-side state
  const [wrapperBuildId, setWrapperBuildId] = useState<string | null>(storedState?.currentBuildId ?? null);
  const [wrapperBuildStatus, setWrapperBuildStatus] = useState<UiBuildStatus | null>(null);
  const [wrapperBuildInitialized, setWrapperBuildInitialized] = useState(false);

  // Fetch build status from server - only poll when we have an active build in "building" state
  // For completed builds (ready/failed), we rely on persisted server-side state
  const shouldPollBuildStatus = Boolean(wrapperBuildId) && (
    !wrapperBuildInitialized || // Still loading from server
    wrapperBuildStatus?.status === "building" // Actively building
  );

  const { data: fetchedBuildStatus } = useQuery({
    queryKey: ["build-status-global", wrapperBuildId],
    queryFn: async () => {
      if (!wrapperBuildId) return null;
      // Use verbose to get generate stage details for output_path
      const pipeline = await getPipelineStatus(wrapperBuildId, { verbose: true });
      return mapPipelineToUiStatus(pipeline);
    },
    enabled: shouldPollBuildStatus,
    refetchInterval: (query) => {
      const data = query.state.data as UiBuildStatus | null;
      // Stop polling when build is complete or failed
      return data?.status === "ready" || data?.status === "failed" ? false : 2000;
    },
    // Don't throw on error - build ID may be stale
    retry: 1,
  });

  // Use server-persisted status as primary, fetched as fallback for active builds
  const effectiveBuildStatus = wrapperBuildStatus || fetchedBuildStatus;

  const [smokeTestInitialized, setSmokeTestInitialized] = useState(false);

  // Wrapped setters that also update URL via the hook
  const setViewMode = useCallback((view: ViewMode) => {
    setViewModeState(view);
    urlState.setViewMode(view);
  }, [urlState]);

  const setSelectedScenarioName = useCallback((name: string) => {
    setSelectedScenarioNameState(name);
    urlState.setScenarioName(name);
  }, [urlState]);

  const setDocPath = useCallback((path: string | null) => {
    setDocPathState(path);
    urlState.setDocPath(path);
  }, [urlState]);

  // Pipeline store for smoke test - only what we need for server state syncing
  const {
    setScenario: setPipelineScenario,
    smokeTestResult,
    clearError: clearSmokeTestError,
  } = usePipelineStore();

  // Server-side state persistence for smoke test and wrapper build results
  const {
    hasInitiallyLoaded: serverStateLoaded,
    updateFormState: updateServerFormState,
    saveStageResult,
  } = useScenarioState({
    scenarioName: selectedScenarioName,
    enabled: Boolean(selectedScenarioName),
    // Only check for staleness on the generator view where form editing is active
    checkStaleness: viewMode === "generator",
    onStateLoaded: (state) => {
      if (!state.form_state) return;
      const fs = state.form_state;

      // Initialize wrapper build state from server on load
      if (!wrapperBuildInitialized) {
        if (fs.wrapper_build_id) {
          setWrapperBuildId(fs.wrapper_build_id);
        }
        // Restore the build status from server - this is the key fix!
        // Creates a minimal UiBuildStatus object from persisted state
        if (fs.wrapper_build_status) {
          setWrapperBuildStatus({
            status: fs.wrapper_build_status,
            output_path: fs.wrapper_output_path ?? undefined,
            pipeline_id: fs.wrapper_build_id || "",
            scenario_name: selectedScenarioName,
          });
        }
        setWrapperBuildInitialized(true);
      }

      // Mark smoke test state as initialized from server
      if (!smokeTestInitialized) {
        setSmokeTestInitialized(true);
      }
    },
  });

  // Sync scenario with pipeline store
  useEffect(() => {
    if (selectedScenarioName) {
      setPipelineScenario(selectedScenarioName);
    }
  }, [selectedScenarioName, setPipelineScenario]);

  // Compute smoke test status from pipeline store (for server persistence)
  const smokeTestStatus: SmokeTestStageDetails | null = smokeTestResult;

  // Reset state when scenario changes
  useEffect(() => {
    setSmokeTestInitialized(false);
    clearSmokeTestError();
    // Also reset wrapper build initialization so it reloads from server for new scenario
    setWrapperBuildInitialized(false);
    setWrapperBuildStatus(null);
    setWrapperBuildId(null);
  }, [selectedScenarioName, clearSmokeTestError]);

  // Persist wrapper build status changes to server
  const prevWrapperBuildStatusRef = useRef<string | null>(null);
  useEffect(() => {
    if (!selectedScenarioName || !serverStateLoaded) return;
    if (!wrapperBuildId) return;
    // Use effective status which combines local state and fetched data
    const currentStatus = effectiveBuildStatus;
    if (!currentStatus) return;

    // Only persist when status actually changes to avoid redundant saves
    const statusKey = `${wrapperBuildId}:${currentStatus.status}`;
    if (statusKey === prevWrapperBuildStatusRef.current) return;
    prevWrapperBuildStatusRef.current = statusKey;

    // Persist wrapper build state to server
    updateServerFormState({
      wrapper_build_id: wrapperBuildId,
      wrapper_build_status: currentStatus.status as "building" | "ready" | "failed",
      wrapper_output_path: currentStatus.output_path ?? null,
    });

    // Update local state to match (in case it came from fetchedBuildStatus)
    if (!wrapperBuildStatus || wrapperBuildStatus.status !== currentStatus.status) {
      setWrapperBuildStatus(currentStatus);
    }
  }, [
    selectedScenarioName,
    serverStateLoaded,
    wrapperBuildId,
    effectiveBuildStatus,
    wrapperBuildStatus,
    updateServerFormState,
  ]);

  // Persist smoke test status changes to server
  const prevSmokeTestStatusRef = useRef<string | null>(null);
  useEffect(() => {
    if (!selectedScenarioName || !serverStateLoaded) return;
    if (!smokeTestStatus) return;

    // Use the smoke test's own ID for status key (not pipeline ID)
    const testId = smokeTestStatus.smoke_test_id;
    if (!testId) return;

    // Only persist when status actually changes to avoid redundant saves
    const statusKey = `${testId}:${smokeTestStatus.status}`;
    if (statusKey === prevSmokeTestStatusRef.current) return;
    prevSmokeTestStatusRef.current = statusKey;

    // Build form state update for smoke test
    const smokeTestFormState: Partial<FormState> = {
      smoke_test_id: testId,
      smoke_test_platform: smokeTestStatus.platform as "win" | "mac" | "linux" | null,
      smoke_test_status: smokeTestStatus.status as "running" | "passed" | "failed" | null,
      smoke_test_started_at: smokeTestStatus.started_at,
      smoke_test_completed_at: smokeTestStatus.completed_at ?? null,
      smoke_test_logs: smokeTestStatus.logs ?? null,
      smoke_test_error: smokeTestStatus.error ?? null,
      smoke_test_telemetry_uploaded: smokeTestStatus.telemetry_uploaded ?? false,
    };

    // If test completed (passed or failed), save as stage result
    if (smokeTestStatus.status === "passed" || smokeTestStatus.status === "failed") {
      void saveStageResult("smoke_test", smokeTestStatus, smokeTestFormState);
    } else {
      // Just update form state for running status
      updateServerFormState(smokeTestFormState);
    }
  }, [
    selectedScenarioName,
    serverStateLoaded,
    smokeTestStatus,
    saveStageResult,
    updateServerFormState,
  ]);

  useEffect(() => {
    saveGeneratorAppState({
      viewMode,
      selectedScenarioName,
      selectedTemplate,
      selectionSource,
      currentBuildId: wrapperBuildId,
      docPath
    });
  }, [
    viewMode,
    selectedScenarioName,
    selectedTemplate,
    selectionSource,
    wrapperBuildId,
    docPath
  ]);

  const handleInventorySelect = (scenario: ScenarioDesktopStatus) => {
    setSelectedScenarioName(scenario.name);
    setSelectionSource("inventory");
    setViewMode("generator");
  };

  const openSigningTab = (scenario?: string) => {
    if (scenario) {
      setSelectedScenarioName(scenario);
    }
    setViewMode("signing");
  };

  const openGeneratorForScenario = (scenario?: string) => {
    if (scenario) {
      setSelectedScenarioName(scenario);
      setSelectionSource("inventory");
    }
    setViewMode("generator");
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-blue-950 to-slate-950 text-slate-50 scroll-smooth">
      <div className="mx-auto max-w-7xl p-6">
        {/* Header */}
        <div className="mb-8 text-center">
          <div className="mb-3 flex items-center justify-center gap-3">
            <Monitor className="h-10 w-10 text-blue-400" />
            <h1 className="text-4xl font-bold">Scenario to Desktop</h1>
          </div>
          <p className="text-lg text-slate-300">
            Transform Vrooli scenarios into professional desktop applications
          </p>
        </div>

        {/* View Mode Selector */}
        <div className="mb-6 flex justify-center">
          <div className="inline-flex items-center gap-1 rounded-full border border-slate-800 bg-slate-900/60 p-1 shadow-lg shadow-blue-950/40">
            <button
              type="button"
              className={cn(
                "flex items-center gap-2 rounded-full px-4 py-2 text-sm font-semibold transition",
                viewMode === "inventory"
                  ? "bg-gradient-to-r from-blue-600 to-blue-500 text-white shadow"
                  : "text-slate-300 hover:text-white"
              )}
              onClick={() => setViewMode("inventory")}
            >
              <List className="h-4 w-4" />
              Scenario Inventory
            </button>
            <button
              type="button"
              className={cn(
                "flex items-center gap-2 rounded-full px-4 py-2 text-sm font-semibold transition",
                viewMode === "generator"
                  ? "bg-gradient-to-r from-blue-600 to-blue-500 text-white shadow"
                  : "text-slate-300 hover:text-white"
              )}
              onClick={() => setViewMode("generator")}
            >
              <Zap className="h-4 w-4" />
              Generate Desktop App
            </button>
            <button
              type="button"
              className={cn(
                "flex items-center gap-2 rounded-full px-4 py-2 text-sm font-semibold transition",
                viewMode === "records"
                  ? "bg-gradient-to-r from-blue-600 to-blue-500 text-white shadow"
                  : "text-slate-300 hover:text-white"
              )}
              onClick={() => setViewMode("records")}
            >
              <Folder className="h-4 w-4" />
              Generated Apps
            </button>
            <button
              type="button"
              className={cn(
                "flex items-center gap-2 rounded-full px-4 py-2 text-sm font-semibold transition",
                viewMode === "signing"
                  ? "bg-gradient-to-r from-blue-600 to-blue-500 text-white shadow"
                  : "text-slate-300 hover:text-white"
              )}
              onClick={() => setViewMode("signing")}
            >
              <Shield className="h-4 w-4" />
              Signing
            </button>
            <button
              type="button"
              className={cn(
                "flex items-center gap-2 rounded-full px-4 py-2 text-sm font-semibold transition",
                viewMode === "docs"
                  ? "bg-gradient-to-r from-blue-600 to-blue-500 text-white shadow"
                  : "text-slate-300 hover:text-white"
              )}
              onClick={() => setViewMode("docs")}
            >
              <Book className="h-4 w-4" />
              Docs
            </button>
          </div>
        </div>

        {/* Conditional Content - Each section wrapped with Error Boundary for graceful degradation */}
        {viewMode === "inventory" ? (
          <SectionErrorBoundary name="Scenario Inventory">
            <ScenarioInventory onScenarioLaunch={handleInventorySelect} />
          </SectionErrorBoundary>
        ) : viewMode === "docs" ? (
          <SectionErrorBoundary name="Documentation">
            <DocsPanel
              initialPath={docPath}
              onPathChange={(path) => {
                if (viewMode === "docs") {
                  setDocPath(path || null);
                }
              }}
            />
          </SectionErrorBoundary>
        ) : viewMode === "signing" ? (
          <SectionErrorBoundary name="Code Signing">
            <SigningPage
              initialScenario={selectedScenarioName}
              onScenarioChange={(name) => {
                setSelectedScenarioName(name);
                setSelectionSource("manual");
              }}
            />
          </SectionErrorBoundary>
        ) : viewMode === "records" ? (
          <SectionErrorBoundary name="Generated Apps">
            <RecordsManager
              onSwitchTemplate={(scenarioName, templateType) => {
                openGeneratorForScenario(scenarioName);
                setSelectedTemplate(templateType || "basic");
              }}
              onEditSigning={(scenarioName) => openSigningTab(scenarioName)}
              onRebuildWithSigning={(scenarioName) => openGeneratorForScenario(scenarioName)}
            />
          </SectionErrorBoundary>
        ) : (
          <SectionErrorBoundary name="Desktop App Generator">
            <GeneratorPage
              scenarioName={selectedScenarioName}
              onScenarioNameChange={(name) => {
                setSelectedScenarioName(name);
                setSelectionSource("manual");
              }}
              selectedTemplate={selectedTemplate}
              onTemplateChange={setSelectedTemplate}
              selectionSource={selectionSource}
              onOpenSigningTab={openSigningTab}
              buildId={wrapperBuildId}
              onBuildStart={(buildId) => {
                setWrapperBuildId(buildId);
              }}
            />
          </SectionErrorBoundary>
        )}

        {/* Footer */}
        <div className="mt-12 pb-24 text-center text-sm text-slate-400">
          <p>
            Built with ❤️ by the{" "}
            <a
              href="https://vrooli.com"
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-400 hover:underline"
            >
              Vrooli Platform
            </a>
            {" | "}
            <a
              href="https://github.com/vrooli/vrooli"
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-400 hover:underline"
            >
              View on GitHub
            </a>
          </p>
        </div>
      </div>

      {/* Fixed Bottom Action Bar - shows when there's an active build */}
      {viewMode === "generator" && wrapperBuildId && (
        <div className="fixed bottom-0 left-0 right-0 z-40 border-t border-slate-700/80 bg-slate-900/95 backdrop-blur-md shadow-lg shadow-slate-950/50">
          <div className="mx-auto max-w-7xl px-6 py-3">
            <div className="flex items-center justify-between gap-4">
              <div className="flex items-center gap-3 min-w-0">
                <div className="flex items-center gap-2">
                  {!effectiveBuildStatus ? (
                    <Loader2 className="h-4 w-4 animate-spin text-slate-400" />
                  ) : effectiveBuildStatus.status === "failed" ? (
                    <div className="h-2.5 w-2.5 rounded-full bg-red-500 animate-pulse" />
                  ) : effectiveBuildStatus.status === "ready" ? (
                    <div className="h-2.5 w-2.5 rounded-full bg-emerald-500" />
                  ) : (
                    <div className="h-2.5 w-2.5 rounded-full bg-blue-500 animate-pulse" />
                  )}
                  <span className="text-sm font-medium text-slate-200 truncate">
                    {selectedScenarioName || "Build"}
                  </span>
                </div>
                <span className="text-xs text-slate-400 hidden sm:inline">
                  {!effectiveBuildStatus ? (
                    "Loading build status..."
                  ) : effectiveBuildStatus.status === "failed" ? (
                    "Build failed - spawn an agent to investigate"
                  ) : effectiveBuildStatus.status === "ready" ? (
                    "Build ready - spawn an agent to verify or improve"
                  ) : effectiveBuildStatus.status === "building" ? (
                    "Build in progress..."
                  ) : (
                    "Spawn an agent to analyze this build"
                  )}
                </span>
              </div>
              <div className="flex items-center gap-2">
                {!effectiveBuildStatus && (
                  <span className="text-xs text-amber-400 hidden md:inline">
                    Waiting for status
                  </span>
                )}
                <SpawnAgentButton
                  pipelineId={wrapperBuildId}
                  disabled={!effectiveBuildStatus}
                />
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function App() {
  return (
    <ErrorBoundary sectionName="Application" showHomeButton>
      <QueryClientProvider client={queryClient}>
        <AppContent />
      </QueryClientProvider>
    </ErrorBoundary>
  );
}
