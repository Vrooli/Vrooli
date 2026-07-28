import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Book, List, Monitor, Zap, Folder, Shield } from "lucide-react";
import { useEffect, useMemo, useRef } from "react";
import { useIsMobile } from "./hooks/useMediaQuery";
import { GeneratorPage } from "./pages";
import { ScenarioInventory } from "./components/scenario-inventory";
import { DocsPanel } from "./components/docs/DocsPanel";
import { SigningPage } from "./components/signing";
import { SpawnAgentButton } from "./components/state/SpawnAgentButton";
import {
  ErrorBoundary,
  SectionErrorBoundary,
} from "./components/ui/ErrorBoundary";
import { LiveDesktopDrawer } from "./components/livedesktop";
import { CapturesDrawer } from "./components/captures";
import type { ScenarioDesktopStatus } from "./components/scenario-inventory/types";
import { usePipelineStore } from "./store";
import { useFormStore } from "./store/formStore";
import { useUrlState, type ViewMode } from "./hooks/useUrlState";
import { useServerSync } from "./hooks/useServerSync";
import {
  loadGeneratorAppState,
  saveGeneratorAppState,
} from "./lib/draftStorage";
import { cn } from "./lib/utils";
import { RecordsManager } from "./components/scenario-inventory/RecordsManager";
import { selectors } from "./consts/selectors";
import { StatusBadge } from "./components/ui/status-badge";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

function AppContent() {
  const storedState = useMemo(() => loadGeneratorAppState(), []);

  // URL state is the single source of truth for view, scenario, and doc
  const urlState = useUrlState({
    defaultView: storedState ? (storedState.viewMode as ViewMode) : "inventory",
    defaultScenario: storedState?.selectedScenarioName ?? "",
    defaultDoc: storedState?.docPath ?? null,
  });

  const {
    viewMode,
    setViewMode,
    scenarioName: selectedScenarioName,
    setScenarioName: setSelectedScenarioName,
    docPath,
    setDocPath,
  } = urlState;

  // selectionSource is transient metadata — always changes alongside scenarioName
  const selectionSourceRef = useRef<"inventory" | "manual" | null>(
    urlState.initialParams.scenario
      ? "manual"
      : (storedState?.selectionSource ?? null),
  );

  // selectedTemplate lives in formStore (it's form state)
  const selectedTemplate = useFormStore((s) => s.selectedTemplate);
  const setSelectedTemplate = useFormStore((s) => s.setSelectedTemplate);

  // Initialize template from localStorage on mount
  useEffect(() => {
    if (
      storedState?.selectedTemplate &&
      storedState.selectedTemplate !== "basic"
    ) {
      setSelectedTemplate(storedState.selectedTemplate);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Intentionally run once on mount

  // Pipeline store - single source of truth for build status
  const storePipelineId = usePipelineStore((s) => s.pipelineId);
  const storeRunStatus = usePipelineStore((s) => s.runStatus);
  const storeGenerateResult = usePipelineStore((s) => s.generateResult);
  const setPipelineScenario = usePipelineStore((s) => s.setScenario);

  // Map store run status to UI-friendly status for bottom bar
  const uiBuildStatus = useMemo(() => {
    if (!storePipelineId) return null;
    let status: "building" | "ready" | "failed";
    switch (storeRunStatus) {
      case "running":
      case "starting":
        status = "building";
        break;
      case "completed":
        status = "ready";
        break;
      case "failed":
      case "cancelled":
        status = "failed";
        break;
      default:
        return null; // idle state — no status to show
    }
    return {
      status,
      output_path: storeGenerateResult?.desktopPath,
      pipeline_id: storePipelineId,
    };
  }, [storePipelineId, storeRunStatus, storeGenerateResult]);

  // Server sync — persists build + smoke test status to server
  useServerSync({ scenarioName: selectedScenarioName, viewMode });

  // Sync scenario with pipeline store
  useEffect(() => {
    if (selectedScenarioName) {
      setPipelineScenario(selectedScenarioName);
    }
  }, [selectedScenarioName, setPipelineScenario]);

  // Persist UI preferences to localStorage
  useEffect(() => {
    saveGeneratorAppState({
      viewMode,
      selectedScenarioName,
      selectedTemplate,
      selectionSource: selectionSourceRef.current,
      currentBuildId: storePipelineId,
      docPath,
    });
  }, [
    viewMode,
    selectedScenarioName,
    selectedTemplate,
    storePipelineId,
    docPath,
  ]);

  const handleInventorySelect = (scenario: ScenarioDesktopStatus) => {
    selectionSourceRef.current = "inventory";
    setSelectedScenarioName(scenario.name);
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
      selectionSourceRef.current = "inventory";
      setSelectedScenarioName(scenario);
    }
    setViewMode("generator");
  };

  const isMobile = useIsMobile();

  /** Tab definitions — single source of truth for the view-mode selector. */
  const tabs: { mode: ViewMode; icon: typeof List; label: string }[] = useMemo(
    () => [
      { mode: "inventory", icon: List, label: "Inventory" },
      { mode: "generator", icon: Zap, label: "Generate" },
      { mode: "records", icon: Folder, label: "Apps" },
      { mode: "signing", icon: Shield, label: "Signing" },
      { mode: "docs", icon: Book, label: "Docs" },
    ],
    [],
  );

  return (
    <div
      data-testid={selectors.app.root}
      className="min-h-full w-full overflow-x-clip bg-[var(--color-background)] text-slate-50 scroll-smooth"
    >
      <div className="mx-auto max-w-7xl p-2 md:p-6">
        {/* Header + tabs — merged into a compact strip on mobile */}
        <div className="mb-3 md:mb-8 text-center">
          <div className="mb-1.5 md:mb-3 flex items-center justify-center gap-2 md:gap-3">
            <Monitor className="hidden md:block h-10 w-10 text-blue-400" />
            <h1 className="text-xl md:text-4xl font-bold">
              Scenario to Desktop
            </h1>
          </div>
          <p className="hidden md:block text-lg text-slate-300">
            Transform Vrooli scenarios into professional desktop applications
          </p>
        </div>

        {/* View Mode Selector — compact icon-only pills on mobile */}
        <div className="mb-3 md:mb-6 flex justify-center">
          <div
            className="flex items-center gap-0.5 md:gap-1 rounded-full border border-slate-800 bg-slate-900/60 p-0.5 md:p-1 shadow-lg shadow-blue-950/40 overflow-x-auto scrollbar-hide max-w-full"
            role="tablist"
          >
            {tabs.map(({ mode, icon: Icon, label }) => (
              <button
                key={mode}
                type="button"
                role="tab"
                aria-selected={viewMode === mode}
                data-testid={
                  mode === "generator"
                    ? selectors.app.generateTab
                    : mode === "signing"
                      ? selectors.app.signingTab
                      : undefined
                }
                className={cn(
                  "flex items-center gap-1.5 md:gap-2 rounded-full px-2.5 md:px-4 py-1.5 md:py-2 text-sm font-semibold transition whitespace-nowrap shrink-0",
                  viewMode === mode
                    ? "bg-gradient-to-r from-blue-600 to-blue-500 text-white shadow"
                    : "text-slate-300 hover:text-white",
                )}
                onClick={() => {
                  setViewMode(mode);
                }}
              >
                <Icon className="h-4 w-4" />
                {(!isMobile || viewMode === mode) && <span>{label}</span>}
              </button>
            ))}
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
                setDocPath(path || null);
              }}
            />
          </SectionErrorBoundary>
        ) : viewMode === "signing" ? (
          <SectionErrorBoundary name="Code Signing">
            <SigningPage
              initialScenario={selectedScenarioName}
              onScenarioChange={(name) => {
                selectionSourceRef.current = "manual";
                setSelectedScenarioName(name);
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
              onEditSigning={(scenarioName) => {
                openSigningTab(scenarioName);
              }}
              onRebuildWithSigning={(scenarioName) => {
                openGeneratorForScenario(scenarioName);
              }}
            />
          </SectionErrorBoundary>
        ) : (
          <SectionErrorBoundary name="Desktop App Generator">
            <GeneratorPage
              scenarioName={selectedScenarioName}
              onScenarioNameChange={(name) => {
                selectionSourceRef.current = "manual";
                setSelectedScenarioName(name);
              }}
              selectedTemplate={selectedTemplate}
              onTemplateChange={setSelectedTemplate}
              selectionSource={selectionSourceRef.current}
              onOpenSigningTab={openSigningTab}
            />
          </SectionErrorBoundary>
        )}

        {/* Bottom spacer for fixed action bar clearance */}
        <div className="pb-24" />
      </div>

      {/* Fixed Bottom Action Bar - shows when there's an active build */}
      {viewMode === "generator" && storePipelineId && uiBuildStatus && (
        <div className="fixed bottom-0 left-0 right-0 z-40 border-t border-slate-700/80 bg-slate-900/95 backdrop-blur-md shadow-lg shadow-slate-950/50">
          <div className="mx-auto max-w-7xl px-3 md:px-6 py-3">
            <div className="flex items-center justify-between gap-4">
              <div className="flex items-center gap-3 min-w-0">
                <StatusBadge
                  aria-live="polite"
                  tone={
                    uiBuildStatus.status === "failed"
                      ? "danger"
                      : uiBuildStatus.status === "ready"
                        ? "success"
                        : "info"
                  }
                >
                  {selectedScenarioName || "Build"}: {uiBuildStatus.status}
                </StatusBadge>
                <span className="text-xs text-slate-400 hidden sm:inline">
                  {uiBuildStatus.status === "failed"
                    ? "Build failed - spawn an agent to investigate"
                    : uiBuildStatus.status === "ready"
                      ? "Build ready - spawn an agent to verify or improve"
                      : "Build in progress..."}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <SpawnAgentButton pipelineId={storePipelineId} />
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
        <LiveDesktopDrawer />
        <CapturesDrawer />
      </QueryClientProvider>
    </ErrorBoundary>
  );
}
