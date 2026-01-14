import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { Book, List, Monitor, Zap, Folder, Shield, Cloud, CheckCircle2, Loader2, Play, XCircle } from "lucide-react";
import { useEffect, useMemo, useRef, useState, type RefObject } from "react";
import { BuildStatus } from "./components/BuildStatus";
import { GeneratorForm } from "./components/GeneratorForm";
import { ScenarioInventory } from "./components/scenario-inventory";
import { BuildDesktopButton } from "./components/scenario-inventory/BuildDesktopButton";
import { DownloadButtons } from "./components/scenario-inventory/DownloadButtons";
import { TelemetryUploadCard } from "./components/scenario-inventory/TelemetryUploadCard";
import { DocsPanel } from "./components/docs/DocsPanel";
import { SigningPage } from "./components/signing";
import { DistributionPage, DistributionUploadSection } from "./components/distribution";
import { SpawnAgentButton } from "./components/SpawnAgentButton";
import type { ScenarioDesktopStatus, ScenariosResponse } from "./components/scenario-inventory/types";
import { StatsPanel } from "./components/StatsPanel";
import { Card, CardContent, CardHeader, CardTitle } from "./components/ui/card";
import { Button } from "./components/ui/button";
import { cancelSmokeTest, fetchBuildStatus, fetchScenarioDesktopStatus, fetchSmokeTestStatus, startSmokeTest } from "./lib/api";
import type { BuildStatus as BuildStatusType, SmokeTestStatus, FormState } from "./lib/api";
import { loadGeneratorAppState, saveGeneratorAppState } from "./lib/draftStorage";
import { cn } from "./lib/utils";
import { RecordsManager } from "./components/RecordsManager";
import { useScenarioState } from "./hooks";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1
    }
  }
});

type ViewMode = "generator" | "inventory" | "docs" | "records" | "signing" | "distribution";

function parseSearchParams(): { view?: ViewMode; scenario?: string; doc?: string } {
  if (typeof window === "undefined") return {};
  const params = new URLSearchParams(window.location.search);
  const view = params.get("view") as ViewMode | null;
  const scenario = params.get("scenario") || undefined;
  const doc = params.get("doc") || undefined;
  return { view: view || undefined, scenario, doc };
}

function AppContent() {
  const initialParams = useMemo(() => parseSearchParams(), []);
  const storedState = useMemo(() => loadGeneratorAppState(), []);
  const [selectedTemplate, setSelectedTemplate] = useState(storedState?.selectedTemplate || "basic");
  // Wrapper build state - will be initialized from server-side state
  const [wrapperBuildId, setWrapperBuildId] = useState<string | null>(storedState?.currentBuildId ?? null);
  const [wrapperBuildStatus, setWrapperBuildStatus] = useState<BuildStatusType | null>(null);
  const [wrapperBuildInitialized, setWrapperBuildInitialized] = useState(false);
  const [selectedScenarioName, setSelectedScenarioName] = useState(
    initialParams.scenario ?? storedState?.selectedScenarioName ?? ""
  );
  const [selectionSource, setSelectionSource] = useState<"inventory" | "manual" | null>(
    initialParams.scenario ? "manual" : storedState?.selectionSource ?? null
  );

  // Fetch build status from server - only poll when we have an active build in "building" state
  // For completed builds (ready/failed), we rely on persisted server-side state
  const shouldPollBuildStatus = Boolean(wrapperBuildId) && (
    !wrapperBuildInitialized || // Still loading from server
    wrapperBuildStatus?.status === "building" // Actively building
  );

  const { data: fetchedBuildStatus } = useQuery({
    queryKey: ["build-status-global", wrapperBuildId],
    queryFn: () => (wrapperBuildId ? fetchBuildStatus(wrapperBuildId) : null),
    enabled: shouldPollBuildStatus,
    refetchInterval: (query) => {
      const data = query.state.data as BuildStatusType | null;
      // Stop polling when build is complete or failed
      return data?.status === "ready" || data?.status === "failed" ? false : 2000;
    },
    // Don't throw on error - build ID may be stale
    retry: 1,
  });

  // Use server-persisted status as primary, fetched as fallback for active builds
  const effectiveBuildStatus = wrapperBuildStatus || fetchedBuildStatus;

  const [viewMode, setViewMode] = useState<ViewMode>(
    (initialParams.view as ViewMode | undefined) ?? (storedState?.viewMode as ViewMode | undefined) ?? "inventory"
  );
  const [docPath, setDocPath] = useState<string | null>(
    initialParams.doc ?? storedState?.docPath ?? null
  );
  const [activeStep, setActiveStep] = useState(storedState?.activeStep ?? 2);
  const [userPinnedStep, setUserPinnedStep] = useState(storedState?.userPinnedStep ?? false);
  const [generateState, setGenerateState] = useState<{ pending: boolean; error: string | null }>({
    pending: false,
    error: null
  });
  const [smokeTestId, setSmokeTestId] = useState<string | null>(null);
  const [smokeTestError, setSmokeTestError] = useState<string | null>(null);
  const [smokeTestCancelling, setSmokeTestCancelling] = useState(false);
  const [smokeTestInitialized, setSmokeTestInitialized] = useState(false);
  const apiBase = useMemo(() => resolveApiBase({ appendSuffix: true }), []);

  // Server-side state persistence for smoke test and wrapper build results
  const {
    formState: serverFormState,
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
        // Creates a minimal BuildStatus object from persisted state
        if (fs.wrapper_build_status) {
          setWrapperBuildStatus({
            status: fs.wrapper_build_status,
            output_path: fs.wrapper_output_path ?? undefined,
          } as BuildStatusType);
        }
        setWrapperBuildInitialized(true);
      }

      // Initialize smoke test state from server on load
      if (!smokeTestInitialized) {
        if (fs.smoke_test_id) {
          setSmokeTestId(fs.smoke_test_id);
        }
        if (fs.smoke_test_error) {
          setSmokeTestError(fs.smoke_test_error);
        }
        setSmokeTestInitialized(true);
      }
    },
  });
  const overviewRef = useRef<HTMLDivElement>(null);
  const configureRef = useRef<HTMLDivElement>(null);
  const buildRef = useRef<HTMLDivElement>(null);
  const validateRef = useRef<HTMLDivElement>(null);
  const distributeRef = useRef<HTMLDivElement>(null);

  // Sync view/scenario with URL hash for sharable routes
  useEffect(() => {
    if (typeof window === "undefined") return;
    const { view, scenario } = parseSearchParams();
    if (view) setViewMode(view);
    if (scenario) setSelectedScenarioName(scenario);

    const handleHashChange = () => {
      const { view: nextView, scenario: nextScenario } = parseSearchParams();
      if (nextView) setViewMode(nextView);
      if (nextScenario !== undefined) {
        setSelectedScenarioName(nextScenario);
        setSelectionSource("manual");
      }
    };

    window.addEventListener("hashchange", handleHashChange);
    return () => window.removeEventListener("hashchange", handleHashChange);
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const params = new URLSearchParams();
    params.set("view", viewMode);
    if (selectedScenarioName) {
      params.set("scenario", selectedScenarioName);
    }
    const newHash = `#${params.toString()}`;
    if (window.location.hash !== newHash) {
      window.history.replaceState(null, "", newHash);
    }
  }, [viewMode, selectedScenarioName]);

  // Only poll scenario list on views that display or use it
  const shouldPollScenarios = viewMode === "inventory" || viewMode === "generator" || viewMode === "signing";

  const { data: scenariosData } = useQuery<ScenariosResponse>({
    queryKey: ["scenarios-desktop-status"],
    queryFn: fetchScenarioDesktopStatus,
    refetchInterval: shouldPollScenarios ? 30000 : false
  });

  const selectedScenario: ScenarioDesktopStatus | null = useMemo(
    () => scenariosData?.scenarios.find((s) => s.name === selectedScenarioName) || null,
    [scenariosData, selectedScenarioName]
  );
  const detectedPlatform = useMemo(() => detectClientPlatform(), []);
  const detectedPlatformLabel = useMemo(() => formatPlatformLabel(detectedPlatform), [detectedPlatform]);
  const matchingArtifact = useMemo(
    () => selectedScenario?.build_artifacts?.find((artifact) => artifact.platform === detectedPlatform) || null,
    [selectedScenario, detectedPlatform]
  );
  const matchingArtifactLabel = matchingArtifact?.file_name ?? matchingArtifact?.relative_path ?? "";

  const { data: smokeTestStatus, refetch: refetchSmokeStatus } = useQuery<SmokeTestStatus | null>({
    queryKey: ["smoke-test-status", smokeTestId],
    queryFn: async () => {
      if (!smokeTestId) return null;
      return fetchSmokeTestStatus(smokeTestId);
    },
    enabled: !!smokeTestId,
    refetchInterval: smokeTestId ? 2000 : false,
    refetchIntervalInBackground: true,
    refetchOnReconnect: true,
    refetchOnMount: true,
    retry: 1
  });

  const smokeTestRunning = smokeTestStatus?.status === "running";
  const smokeTestPassed = smokeTestStatus?.status === "passed";
  const smokeTestFailed = smokeTestStatus?.status === "failed";
  const recommendedStep = useMemo(() => {
    if (viewMode !== "generator") return 2;
    // If smoke test passed, recommend distribution step
    if (smokeTestPassed) return 5;
    // If build artifacts exist, recommend validation step
    if (selectedScenario?.build_artifacts?.length) return 4;
    // If wrapper build started, recommend build step
    if (wrapperBuildId) return 3;
    // If scenario selected, recommend configure step
    if (selectedScenarioName) return 2;
    return 1;
  }, [viewMode, smokeTestPassed, selectedScenario?.build_artifacts?.length, wrapperBuildId, selectedScenarioName]);
  const smokeTestTimeoutMessage =
    smokeTestStatus?.error && smokeTestStatus.error.includes("timed out")
      ? smokeTestStatus.error
      : null;
  const smokeTestStatusUrl = useMemo(() => {
    if (!smokeTestId) return "";
    return buildApiUrl(`/desktop/smoke-test/status/${smokeTestId}`, { baseUrl: apiBase });
  }, [apiBase, smokeTestId]);
  const generatorFormId = "desktop-generator-form";
  const hasWrapper = Boolean(wrapperBuildId) || Boolean(selectedScenario?.has_desktop);
  const generateLabel = hasWrapper ? "Regenerate Wrapper" : "Generate Wrapper";
  const canGenerate = Boolean(selectedScenarioName);
  const canBuildInstallers = Boolean(
    selectedScenarioName && (effectiveBuildStatus?.status === "ready" || selectedScenario?.has_desktop)
  );

  // Reset state when scenario changes
  useEffect(() => {
    setSmokeTestId(null);
    setSmokeTestError(null);
    setSmokeTestCancelling(false);
    setSmokeTestInitialized(false);
    // Also reset wrapper build initialization so it reloads from server for new scenario
    setWrapperBuildInitialized(false);
    setWrapperBuildStatus(null);
    setWrapperBuildId(null);
  }, [selectedScenarioName]);

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
    if (!smokeTestId) return;
    if (!smokeTestStatus) return;

    // Only persist when status actually changes to avoid redundant saves
    const statusKey = `${smokeTestId}:${smokeTestStatus.status}`;
    if (statusKey === prevSmokeTestStatusRef.current) return;
    prevSmokeTestStatusRef.current = statusKey;

    // Build form state update for smoke test
    const smokeTestFormState: Partial<FormState> = {
      smoke_test_id: smokeTestId,
      smoke_test_platform: smokeTestStatus.platform,
      smoke_test_status: smokeTestStatus.status,
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
    smokeTestId,
    smokeTestStatus,
    saveStageResult,
    updateServerFormState,
  ]);

  // Sync view/scenario/doc from URL on load and back/forward
  useEffect(() => {
    if (typeof window === "undefined") return;
    const syncFromLocation = () => {
      const { view, scenario, doc } = parseSearchParams();
      if (view) setViewMode(view);
      if (scenario !== undefined) setSelectedScenarioName(scenario);
      if (doc !== undefined) setDocPath(doc);
    };
    syncFromLocation();
    window.addEventListener("popstate", syncFromLocation);
    return () => window.removeEventListener("popstate", syncFromLocation);
  }, []);

  // Persist view/scenario/doc to URL for shareable routing
  useEffect(() => {
    if (typeof window === "undefined") return;
    const url = new URL(window.location.href);
    const params = new URLSearchParams(url.search);
    params.set("view", viewMode);
    if (selectedScenarioName) {
      params.set("scenario", selectedScenarioName);
    } else {
      params.delete("scenario");
    }
    if (docPath) {
      params.set("doc", docPath);
    } else {
      params.delete("doc");
    }
    url.search = params.toString();
    const newUrl = url.toString();
    if (window.location.href !== newUrl) {
      window.history.replaceState(null, "", newUrl);
    }
  }, [viewMode, selectedScenarioName, docPath]);

  useEffect(() => {
    saveGeneratorAppState({
      viewMode,
      selectedScenarioName,
      selectedTemplate,
      selectionSource,
      currentBuildId: wrapperBuildId,
      activeStep,
      userPinnedStep,
      docPath
    });
  }, [
    viewMode,
    selectedScenarioName,
    selectedTemplate,
    selectionSource,
    wrapperBuildId,
    activeStep,
    userPinnedStep,
    docPath
  ]);

  useEffect(() => {
    if (viewMode !== "generator") {
      setUserPinnedStep(false);
      setActiveStep(2);
      return;
    }
    // Only update automatically when the user hasn't pinned a step.
    if (!userPinnedStep) {
      setActiveStep(recommendedStep);
    }
  }, [viewMode, recommendedStep, userPinnedStep]);

  const handleInventorySelect = (scenario: ScenarioDesktopStatus) => {
    setSelectedScenarioName(scenario.name);
    setSelectionSource("inventory");
    setViewMode("generator");
    setUserPinnedStep(false);
  };

  const openSigningTab = (scenario?: string) => {
    if (scenario) {
      setSelectedScenarioName(scenario);
    }
    setViewMode("signing");
    setUserPinnedStep(false);
  };

  const openGeneratorForScenario = (scenario?: string) => {
    if (scenario) {
      setSelectedScenarioName(scenario);
      setSelectionSource("inventory");
    }
    setViewMode("generator");
    setUserPinnedStep(false);
    setActiveStep(2);
  };

  const scrollTargets: Record<number, RefObject<HTMLDivElement | null>> = useMemo(
    () => ({
      1: overviewRef,
      2: configureRef,
      3: buildRef,
      4: validateRef,
      5: distributeRef
    }),
    []
  );

  const handleStepSelect = (stepId: number) => {
    setUserPinnedStep(true);
    setActiveStep(stepId);
    // Scroll after a short delay to ensure refs are on the page.
    window.requestAnimationFrame(() => {
      const target = scrollTargets[stepId]?.current;
      if (target) {
        target.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    });
  };

  const steps = [
    { id: 1, title: "Overview", description: "How the desktop build works" },
    { id: 2, title: "Configure", description: "Select scenario, template, and connection" },
    { id: 3, title: "Build", description: "Kick off installers and watch progress" },
    { id: 4, title: "Validate", description: "Download and smoke test installers" },
    { id: 5, title: "Distribute", description: "Upload to cloud storage for distribution" }
  ];

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
                viewMode === "distribution"
                  ? "bg-gradient-to-r from-blue-600 to-blue-500 text-white shadow"
                  : "text-slate-300 hover:text-white"
              )}
              onClick={() => setViewMode("distribution")}
            >
              <Cloud className="h-4 w-4" />
              Distribution
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

        {/* Conditional Content */}
        {viewMode === "inventory" ? (
          <ScenarioInventory onScenarioLaunch={handleInventorySelect} />
        ) : viewMode === "docs" ? (
          <DocsPanel
            initialPath={docPath}
            onPathChange={(path) => {
              if (viewMode === "docs") {
                setDocPath(path || null);
              }
            }}
          />
        ) : viewMode === "signing" ? (
          <SigningPage
            initialScenario={selectedScenarioName}
            onScenarioChange={(name) => {
              setSelectedScenarioName(name);
              setSelectionSource("manual");
            }}
          />
        ) : viewMode === "distribution" ? (
          <DistributionPage />
        ) : viewMode === "records" ? (
          <RecordsManager
            onSwitchTemplate={(scenarioName, templateType) => {
              openGeneratorForScenario(scenarioName);
              setSelectedTemplate(templateType || "basic");
            }}
            onEditSigning={(scenarioName) => openSigningTab(scenarioName)}
            onRebuildWithSigning={(scenarioName) => openGeneratorForScenario(scenarioName)}
          />
        ) : (
          <>
            <Stepper steps={steps} activeStep={activeStep} onStepSelect={handleStepSelect} />

            <div className="space-y-6">
              {/* Step 1 */}
              <div ref={overviewRef}>
                <Card className="border-slate-800/80 bg-slate-900/70">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-sm uppercase tracking-wide text-slate-300">
                      Step 1 · Overview
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="grid gap-6 lg:grid-cols-3">
                    <div className="lg:col-span-2 space-y-2">
                      <p className="text-lg font-semibold text-slate-50">
                        Connect → Build → Deliver with telemetry awareness
                      </p>
                      <p className="text-sm text-slate-300">
                        Start by understanding the journey: we link to your running scenario, package installers across
                        platforms, then give you downloads and optional telemetry upload so deployment-manager can spot
                        missing dependencies.
                      </p>
                      <a
                        href="https://github.com/vrooli/vrooli/blob/main/docs/deployment/tiers/tier-2-desktop.md"
                        target="_blank"
                        rel="noreferrer"
                        className="text-sm text-blue-300 underline hover:text-blue-200"
                      >
                        Read the Desktop tier guide
                      </a>
                    </div>
                    <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-3">
                      <StatsPanel />
                    </div>
                  </CardContent>
                </Card>
              </div>

              {/* Step 2 */}
              <div className="space-y-6" ref={configureRef}>
                <Card className="border-blue-800/70 bg-slate-900/70">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-sm uppercase tracking-wide text-slate-300">
                      Step 2 · Configure
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <GeneratorForm
                      selectedTemplate={selectedTemplate}
                      onTemplateChange={setSelectedTemplate}
                      onBuildStart={(buildId) => {
                        setWrapperBuildId(buildId);
                        setActiveStep(3);
                      }}
                      scenarioName={selectedScenarioName}
                      onScenarioNameChange={(name) => {
                        setSelectedScenarioName(name);
                        setSelectionSource("manual");
                        setActiveStep(name ? 2 : 1);
                      }}
                      selectionSource={selectionSource}
                      onOpenSigningTab={openSigningTab}
                      formId={generatorFormId}
                      showSubmit={false}
                      onGenerateStateChange={setGenerateState}
                    />
                  </CardContent>
                </Card>
              </div>

              {/* Step 3 */}
              <Card className="border-slate-800/80 bg-slate-900/70" ref={buildRef}>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2 text-sm uppercase tracking-wide text-slate-300">
                    Step 3 · Generate wrapper & build installers
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="rounded-lg border border-slate-800 bg-slate-950/50 p-3 text-sm text-slate-300">
                    <p className="font-semibold text-slate-200">Wrapper vs installer</p>
                    <p>
                      <span className="font-semibold text-slate-200">Wrapper</span>: Electron project scaffold stored in{" "}
                      <span className="font-mono text-slate-200">platforms/electron</span> for development and config.
                    </p>
                    <p>
                      <span className="font-semibold text-slate-200">Installer</span>: Packaged distributables built into{" "}
                      <span className="font-mono text-slate-200">dist-electron/</span> (AppImage/EXE/PKG/ZIP).
                    </p>
                  </div>

                  <div className="space-y-4">
                    <div className="text-sm font-semibold text-slate-200">1) Generate wrapper</div>
                    <div className="flex flex-wrap items-center gap-3">
                      <Button
                        type="submit"
                        form={generatorFormId}
                        disabled={!canGenerate || generateState.pending}
                      >
                        {generateState.pending ? "Generating wrapper..." : generateLabel}
                      </Button>
                      {!canGenerate && (
                        <p className="text-xs text-slate-400">Pick a scenario in step 2 to enable builds.</p>
                      )}
                    </div>
                    {generateState.error && (
                      <div className="rounded-lg bg-red-900/20 p-3 text-sm text-red-300">
                        <strong>Error:</strong> {generateState.error}
                      </div>
                    )}
                    {wrapperBuildId ? (
                      <BuildStatus
                        buildId={wrapperBuildId}
                        onStatusChange={(status) => {
                          setWrapperBuildStatus(status);
                        }}
                      />
                    ) : (
                      <p className="text-sm text-slate-300">
                        Generate the wrapper using the settings from step 2. Progress will appear here automatically.
                      </p>
                    )}
                  </div>

                  <div className="space-y-3">
                    <div className="text-sm font-semibold text-slate-200">2) Build installers</div>
                    {selectedScenarioName ? (
                      canBuildInstallers ? (
                        <BuildDesktopButton scenarioName={selectedScenarioName} />
                      ) : (
                        <p className="text-xs text-slate-400">
                          Generate the wrapper first so installer packaging can run.
                        </p>
                      )
                    ) : (
                      <p className="text-xs text-slate-400">Select a scenario to unlock installer builds.</p>
                    )}
                  </div>
                </CardContent>
              </Card>

              {/* Step 4 */}
              <Card className="border-slate-800/80 bg-slate-900/70" ref={validateRef}>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2 text-sm uppercase tracking-wide text-slate-300">
                    Step 4 · Download & Validate
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  {!selectedScenarioName && (
                    <p className="text-sm text-slate-300">
                      Select a scenario in step 2 to unlock downloads and smoke testing.
                    </p>
                  )}
                  {selectedScenarioName && !selectedScenario && (
                    <p className="text-sm text-slate-300">
                      Loading scenario details...
                    </p>
                  )}
                  {selectedScenario && (
                    <>
                      {(selectedScenario.build_artifacts?.length ?? 0) > 0 ? (
                        <div className="space-y-4">
                          <DownloadButtons
                            scenarioName={selectedScenario.name}
                            artifacts={selectedScenario.build_artifacts || []}
                          />

                          <div className="rounded-lg border border-slate-800 bg-slate-950/40 p-4 space-y-4">
                            <div className="flex items-center justify-between">
                              <div className="text-sm font-semibold text-slate-100">Validate build</div>
                              <div className="text-xs text-slate-400">Smoke test + telemetry</div>
                            </div>

                            <div className="grid gap-2 text-xs text-slate-300">
                              <div className="flex items-center justify-between">
                                <span>Detected OS</span>
                                <span className="font-medium text-slate-100">{detectedPlatformLabel}</span>
                              </div>
                              <div className="flex items-center justify-between">
                                <span>Matching build</span>
                                {matchingArtifact ? (
                                  <span className="text-emerald-300">Found</span>
                                ) : (
                                  <span className="text-amber-300">Not found</span>
                                )}
                              </div>
                              {matchingArtifactLabel && (
                                <div className="text-[11px] text-slate-500">
                                  {matchingArtifactLabel}
                                </div>
                              )}
                            </div>

                            <div className="flex flex-wrap items-center gap-3">
                              <Button
                                type="button"
                                size="sm"
                                disabled={!matchingArtifact || smokeTestRunning}
                                onClick={async () => {
                                  if (!selectedScenario) return;
                                  setSmokeTestError(null);
                                  try {
                                    const status = await startSmokeTest({
                                      scenario_name: selectedScenario.name,
                                      platform: detectedPlatform
                                    });
                                    setSmokeTestId(status.smoke_test_id);
                                    // Persist initial smoke test state to server
                                    updateServerFormState({
                                      smoke_test_id: status.smoke_test_id,
                                      smoke_test_platform: status.platform,
                                      smoke_test_status: status.status,
                                      smoke_test_started_at: status.started_at,
                                      smoke_test_completed_at: null,
                                      smoke_test_logs: null,
                                      smoke_test_error: null,
                                      smoke_test_telemetry_uploaded: false,
                                    });
                                  } catch (error) {
                                    setSmokeTestError(error instanceof Error ? error.message : "Failed to start smoke test");
                                  }
                                }}
                              >
                                {smokeTestRunning ? (
                                  <>
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                    Running...
                                  </>
                                ) : (
                                  <>
                                    <Play className="h-4 w-4" />
                                    Run smoke test
                                  </>
                                )}
                              </Button>

                              {smokeTestRunning && smokeTestId && (
                                <Button
                                  type="button"
                                  size="sm"
                                  variant="outline"
                                  disabled={smokeTestCancelling}
                                  onClick={async () => {
                                    setSmokeTestError(null);
                                    setSmokeTestCancelling(true);
                                    try {
                                      await cancelSmokeTest(smokeTestId);
                                    } catch (error) {
                                      setSmokeTestError(
                                        error instanceof Error ? error.message : "Failed to cancel smoke test"
                                      );
                                    } finally {
                                      setSmokeTestCancelling(false);
                                    }
                                  }}
                                >
                                  {smokeTestCancelling ? "Cancelling..." : "Cancel"}
                                </Button>
                              )}

                              {smokeTestId && !smokeTestRunning && (
                                <Button
                                  type="button"
                                  size="sm"
                                  variant="ghost"
                                  onClick={() => {
                                    void refetchSmokeStatus();
                                  }}
                                >
                                  Refresh status
                                </Button>
                              )}

                              {!matchingArtifact && (
                                <span className="text-xs text-slate-400">
                                  Build an installer for this OS to enable smoke testing.
                                </span>
                              )}
                            </div>

                            {smokeTestError && (
                              <div className="rounded bg-red-900/20 px-2 py-1 text-xs text-red-300">
                                {smokeTestError}
                              </div>
                            )}

                            {(smokeTestStatus || smokeTestRunning) && (
                              <div className="rounded border border-slate-800 bg-slate-900/40 p-3 text-xs text-slate-300 space-y-2">
                                <div className="flex items-center gap-2">
                                  {smokeTestPassed && <CheckCircle2 className="h-4 w-4 text-emerald-300" />}
                                  {smokeTestFailed && <XCircle className="h-4 w-4 text-red-300" />}
                                  {smokeTestRunning && <Loader2 className="h-4 w-4 animate-spin text-slate-300" />}
                                  <span>
                                    {smokeTestRunning && "Smoke test running..."}
                                    {smokeTestPassed && "Smoke test passed"}
                                    {smokeTestFailed && "Smoke test failed"}
                                  </span>
                                </div>
                                {smokeTestCancelling && (
                                  <p className="text-amber-300">Cancelling smoke test...</p>
                                )}
                                {smokeTestStatus?.telemetry_uploaded && (
                                  <p className="text-emerald-300">Telemetry auto-uploaded.</p>
                                )}
                                {smokeTestStatus?.telemetry_upload_error && (
                                  <p className="text-amber-300">{smokeTestStatus.telemetry_upload_error}</p>
                                )}
                                {smokeTestTimeoutMessage && (
                                  <p className="text-amber-300">{smokeTestTimeoutMessage}</p>
                                )}
                                {smokeTestId && (
                                  <div className="text-[11px] text-slate-400">
                                    <p>Smoke test ID: <span className="font-mono text-slate-200">{smokeTestId}</span></p>
                                    {smokeTestStatusUrl && (
                                      <p>
                                        Status URL:{" "}
                                        <a
                                          className="font-mono text-slate-200 underline"
                                          href={smokeTestStatusUrl}
                                          target="_blank"
                                          rel="noreferrer"
                                        >
                                          {smokeTestStatusUrl}
                                        </a>
                                      </p>
                                    )}
                                  </div>
                                )}
                                {smokeTestStatus?.error && !smokeTestTimeoutMessage && (
                                  <p className="text-red-300">{smokeTestStatus.error}</p>
                                )}
                                {smokeTestStatus?.logs && smokeTestStatus.logs.length > 0 && (
                                  <details>
                                    <summary className="cursor-pointer text-slate-400">Logs</summary>
                                    <pre className="mt-2 max-h-40 overflow-auto rounded bg-black/40 p-2 text-[11px] text-slate-300">
                                      {smokeTestStatus.logs.join("\n\n")}
                                    </pre>
                                  </details>
                                )}
                              </div>
                            )}

                            <div className="border-t border-slate-800 pt-3">
                              <TelemetryUploadCard
                                scenarioName={selectedScenario.name}
                                appDisplayName={selectedScenario.display_name || selectedScenario.name}
                              />
                            </div>
                          </div>
                        </div>
                      ) : (
                        <div className="space-y-3 text-sm text-slate-300">
                          <p>No installers detected yet for this scenario.</p>
                          <p className="text-xs text-slate-400">
                            Wrappers live in{" "}
                            <span className="font-mono text-slate-200">platforms/electron</span>; installers appear in{" "}
                            <span className="font-mono text-slate-200">dist-electron/</span> after packaging.
                          </p>
                          <div className="space-y-1 text-xs text-slate-400">
                            <p>
                              Expected output:{" "}
                              <span className="font-mono text-slate-200">
                                {selectedScenario.artifacts_expected_path ||
                                  `scenarios/${selectedScenario.name}/platforms/electron/dist-electron`}
                              </span>
                            </p>
                            {selectedScenario.record_output_path && (
                              <p>
                                Last recorded build output:{" "}
                                <span className="font-mono text-slate-200">
                                  {selectedScenario.record_output_path}
                                </span>
                                {selectedScenario.record_location_mode && (
                                  <> ({selectedScenario.record_location_mode})</>
                                )}
                              </p>
                            )}
                            {selectedScenario.record_updated_at && (
                              <p>Last recorded build: {selectedScenario.record_updated_at}</p>
                            )}
                          </div>
                          <div className="flex flex-wrap items-center gap-2">
                            <Button type="button" variant="outline" size="sm" onClick={() => setViewMode("records")}>
                              Open Records
                            </Button>
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() => setActiveStep(3)}
                            >
                              Back to build
                            </Button>
                          </div>
                        </div>
                      )}
                    </>
                  )}
                </CardContent>
              </Card>

              {/* Step 5 */}
              <Card className="border-slate-800/80 bg-slate-900/70" ref={distributeRef}>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2 text-sm uppercase tracking-wide text-slate-300">
                    Step 5 · Distribute
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  {!selectedScenarioName && (
                    <p className="text-sm text-slate-300">
                      Select a scenario in step 2 to unlock distribution uploads.
                    </p>
                  )}
                  {selectedScenarioName && !selectedScenario && (
                    <p className="text-sm text-slate-300">
                      Loading scenario details...
                    </p>
                  )}
                  {selectedScenario && (
                    <>
                      {(selectedScenario.build_artifacts?.length ?? 0) > 0 ? (
                        <div className="space-y-4">
                          <div className="rounded-lg border border-slate-800 bg-slate-950/50 p-3 text-sm text-slate-300">
                            <p className="font-semibold text-slate-200">Upload to cloud storage</p>
                            <p className="text-xs text-slate-400">
                              Configure distribution targets in the Distribution tab, then upload your installers to
                              S3-compatible storage for public download links.
                            </p>
                          </div>

                          <DistributionUploadSection
                            scenarioName={selectedScenario.name}
                            artifacts={
                              (selectedScenario.build_artifacts || []).reduce((acc, artifact) => {
                                const path = artifact.absolute_path || artifact.relative_path;
                                if (path && artifact.platform) {
                                  acc[artifact.platform] = path;
                                }
                                return acc;
                              }, {} as Record<string, string>)
                            }
                          />

                          <div className="flex flex-wrap items-center gap-2 text-xs">
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              onClick={() => setViewMode("distribution")}
                            >
                              Configure Distribution Targets
                            </Button>
                            <span className="text-slate-400">
                              Set up S3/R2 buckets for hosting downloads.
                            </span>
                          </div>
                        </div>
                      ) : (
                        <div className="space-y-3 text-sm text-slate-300">
                          <p>Build installers first to enable distribution uploads.</p>
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={() => setActiveStep(3)}
                          >
                            Back to build
                          </Button>
                        </div>
                      )}
                    </>
                  )}
                </CardContent>
              </Card>
            </div>
          </>
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

function detectClientPlatform(): "win" | "mac" | "linux" {
  if (typeof navigator === "undefined") return "linux";
  const platform = (navigator.platform || navigator.userAgent || "").toLowerCase();
  if (platform.includes("win")) return "win";
  if (platform.includes("mac")) return "mac";
  return "linux";
}

function formatPlatformLabel(platform: "win" | "mac" | "linux"): string {
  switch (platform) {
    case "win":
      return "Windows";
    case "mac":
      return "macOS";
    default:
      return "Linux";
  }
}

interface StepperProps {
  steps: { id: number; title: string; description: string }[];
  activeStep: number;
  onStepSelect: (step: number) => void;
}

function Stepper({ steps, activeStep, onStepSelect }: StepperProps) {
  return (
    <div className="mb-6 overflow-x-auto">
      <div className="flex min-w-full justify-center gap-3 rounded-xl border border-slate-800/80 bg-slate-900/60 p-3">
        {steps.map((step) => (
          <button
            key={step.id}
            type="button"
            onClick={() => onStepSelect(step.id)}
            className={cn(
              "flex min-w-[180px] flex-1 flex-col gap-1 rounded-lg border px-4 py-3 text-left transition",
              activeStep === step.id
                ? "border-blue-600 bg-blue-950/40 text-white shadow"
                : "border-slate-800 bg-slate-950/40 text-slate-300 hover:border-slate-600"
            )}
          >
            <div className="flex items-center gap-2 text-xs uppercase tracking-wide">
              <span
                className={cn(
                  "flex h-6 w-6 items-center justify-center rounded-full text-sm font-bold",
                  activeStep === step.id ? "bg-blue-500 text-slate-950" : "bg-slate-800 text-slate-200"
                )}
              >
                {step.id}
              </span>
              <span>{step.title}</span>
            </div>
            <p className="text-sm text-slate-300">{step.description}</p>
          </button>
        ))}
      </div>
    </div>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AppContent />
    </QueryClientProvider>
  );
}
