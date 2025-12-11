import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { Book, List, Monitor, Zap, Folder, Info, Shield } from "lucide-react";
import { useEffect, useMemo, useRef, useState, type RefObject } from "react";
import { BuildStatus } from "./components/BuildStatus";
import { GeneratorForm } from "./components/GeneratorForm";
import { ScenarioInventory } from "./components/scenario-inventory";
import { DownloadButtons } from "./components/scenario-inventory/DownloadButtons";
import { TelemetryUploadCard } from "./components/scenario-inventory/TelemetryUploadCard";
import { DocsPanel } from "./components/docs/DocsPanel";
import { SigningPage } from "./components/signing";
import type { ScenarioDesktopStatus, ScenariosResponse } from "./components/scenario-inventory/types";
import { StatsPanel } from "./components/StatsPanel";
import { TemplateGrid } from "./components/TemplateGrid";
import { Card, CardContent, CardHeader, CardTitle } from "./components/ui/card";
import { fetchScenarioDesktopStatus } from "./lib/api";
import { cn } from "./lib/utils";
import { RecordsManager } from "./components/RecordsManager";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1
    }
  }
});

type ViewMode = "generator" | "inventory" | "docs" | "records" | "signing";

function parseSearchParams(): { view?: ViewMode; scenario?: string; doc?: string } {
  if (typeof window === "undefined") return {};
  const params = new URLSearchParams(window.location.search);
  const view = params.get("view") as ViewMode | null;
  const scenario = params.get("scenario") || undefined;
  const doc = params.get("doc") || undefined;
  return { view: view || undefined, scenario, doc };
}

function AppContent() {
  const [selectedTemplate, setSelectedTemplate] = useState("basic");
  const [currentBuildId, setCurrentBuildId] = useState<string | null>(null);
  const [selectedScenarioName, setSelectedScenarioName] = useState("");
  const [selectionSource, setSelectionSource] = useState<"inventory" | "manual" | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>("inventory");
  const [docPath, setDocPath] = useState<string | null>(null);
  const [activeStep, setActiveStep] = useState(2);
  const [userPinnedStep, setUserPinnedStep] = useState(false);
  const overviewRef = useRef<HTMLDivElement>(null);
  const configureRef = useRef<HTMLDivElement>(null);
  const buildRef = useRef<HTMLDivElement>(null);
  const deliverRef = useRef<HTMLDivElement>(null);

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

  const { data: scenariosData } = useQuery<ScenariosResponse>({
    queryKey: ["scenarios-desktop-status"],
    queryFn: fetchScenarioDesktopStatus,
    refetchInterval: 30000
  });

  const selectedScenario: ScenarioDesktopStatus | null = useMemo(
    () => scenariosData?.scenarios.find((s) => s.name === selectedScenarioName) || null,
    [scenariosData, selectedScenarioName]
  );

  const recommendedStep = useMemo(() => {
    if (viewMode !== "generator") return 2;
    if (selectedScenario?.build_artifacts?.length) return 4;
    if (currentBuildId) return 3;
    if (selectedScenarioName) return 2;
    return 1;
  }, [viewMode, selectedScenario?.build_artifacts?.length, currentBuildId, selectedScenarioName]);

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

  const scrollTargets: Record<number, RefObject<HTMLDivElement>> = useMemo(
    () => ({
      1: overviewRef,
      2: configureRef,
      3: buildRef,
      4: deliverRef
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
    { id: 4, title: "Deliver", description: "Download installers and share telemetry" }
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
              <div className="grid gap-6 lg:grid-cols-2" ref={configureRef}>
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
                        setCurrentBuildId(buildId);
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
                    />
                  </CardContent>
                </Card>

                <Card className="border-slate-800/80 bg-slate-900/70">
                  <CardHeader>
                    <CardTitle>Available Templates</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="mb-4 flex items-start gap-3 rounded-lg border border-slate-800/80 bg-slate-950/50 p-3 text-sm text-slate-200">
                      <Info className="mt-0.5 h-5 w-5 text-blue-300" />
                      <div className="space-y-1">
                        <p className="font-semibold text-slate-100">All templates share the same Electron base.</p>
                        <p className="text-slate-300">
                          Pick the wrapper that matches today&apos;s needs; you can switch templates later from this form
                          or from the Generated Apps tab. Your scenario logic stays the same.
                        </p>
                      </div>
                    </div>
                    <TemplateGrid
                      selectedTemplate={selectedTemplate}
                      onSelect={setSelectedTemplate}
                    />
                  </CardContent>
                </Card>
              </div>

              {/* Step 3 */}
              <Card className="border-slate-800/80 bg-slate-900/70" ref={buildRef}>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2 text-sm uppercase tracking-wide text-slate-300">
                    Step 3 · Build installers
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  {currentBuildId ? (
                    <BuildStatus buildId={currentBuildId} />
                  ) : (
                    <p className="text-sm text-slate-300">
                      Generate from step 2 to kick off a build. Progress will appear here automatically.
                    </p>
                  )}
                </CardContent>
              </Card>

              {/* Step 4 */}
              <Card className="border-slate-800/80 bg-slate-900/70" ref={deliverRef}>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2 text-sm uppercase tracking-wide text-slate-300">
                    Step 4 · Download & Telemetry
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  {!selectedScenarioName && (
                    <p className="text-sm text-slate-300">
                      Select a scenario in step 2 to unlock downloads and telemetry uploads.
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
                          <TelemetryUploadCard
                            scenarioName={selectedScenario.name}
                            appDisplayName={selectedScenario.display_name || selectedScenario.name}
                          />
                        </div>
                      ) : (
                        <p className="text-sm text-slate-300">
                          Build at least one installer to unlock downloads and telemetry uploads.
                        </p>
                      )}
                    </>
                  )}
                </CardContent>
              </Card>
            </div>
          </>
        )}

        {/* Footer */}
        <div className="mt-12 text-center text-sm text-slate-400">
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
    </div>
  );
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
