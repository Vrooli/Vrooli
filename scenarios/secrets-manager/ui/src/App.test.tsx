import { act, cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "./test-utils";
import App from "./App";

const mocks = vi.hoisted(() => ({
  secretsData: vi.fn(),
  vulnerabilities: vi.fn(),
  resourcePanel: vi.fn(),
  journeys: vi.fn(),
  scenarios: vi.fn(),
  campaigns: vi.fn(),
  tabRouting: vi.fn(),
  journeyOptions: vi.fn()
}));

vi.mock("./hooks/useSecretsData", () => ({ useSecretsData: mocks.secretsData }));
vi.mock("./hooks/useVulnerabilities", () => ({ useVulnerabilities: mocks.vulnerabilities }));
vi.mock("./hooks/useResourcePanel", () => ({ useResourcePanel: mocks.resourcePanel }));
vi.mock("./hooks/useJourneys", () => ({ useJourneys: mocks.journeys }));
vi.mock("./hooks/useScenarios", () => ({ useScenarios: mocks.scenarios }));
vi.mock("./hooks/useCampaigns", () => ({ useCampaigns: mocks.campaigns }));
vi.mock("./hooks/useTabRouting", () => ({ useTabRouting: mocks.tabRouting }));

vi.mock("./sections/Header", () => ({ Header: ({ onRefresh }: { onRefresh: () => void }) => <button onClick={onRefresh}>Refresh all</button> }));
vi.mock("./sections/OrientationHub", () => ({ OrientationHub: ({ onJourneySelect }: { onJourneySelect: (id: "prep-deployment") => void }) => <button onClick={() => onJourneySelect("prep-deployment")}>Orientation hub</button> }));
vi.mock("./sections/TierReadiness", () => ({ TierReadiness: () => <div>Tier readiness</div> }));
vi.mock("./features/manifest-editor", () => ({ ManifestWorkspace: ({ onScenarioChange, onToggleCollapse }: { onScenarioChange: (scenario: string) => void; onToggleCollapse: () => void }) => <div>Manifest workspace<button onClick={() => onScenarioChange("api-gateway")}>Change workspace scenario</button><button onClick={onToggleCollapse}>Toggle workspace</button></div> }));
vi.mock("./sections/ComplianceOverview", () => ({ ComplianceOverview: () => <div>Compliance overview</div> }));
vi.mock("./sections/SecurityTables", () => ({ SecurityTables: () => <div>Security tables</div> }));
vi.mock("./features/resource-panel/ResourcePanel", () => ({ ResourcePanel: ({ onClose, onSwitchResource }: { onClose: () => void; onSwitchResource: (resource: string) => void }) => <div>Resource panel<button onClick={onClose}>Close resource</button><button onClick={() => onSwitchResource("redis")}>Switch resource</button></div> }));
vi.mock("./sections/SnapshotPanel", () => ({ SnapshotPanel: () => <div>Snapshot panel</div> }));
vi.mock("./sections/ResourceTable", () => ({ ResourceTable: () => <div>Resource table</div> }));
vi.mock("./sections/CampaignsPanel", () => ({ CampaignsPanel: ({ onSelectScenario, onToggleCollapse }: { onSelectScenario: (scenario: string) => void; onToggleCollapse: () => void }) => <div>Campaigns panel<button onClick={() => onSelectScenario("api-gateway")}>Select campaign</button><button onClick={onToggleCollapse}>Toggle campaigns</button></div> }));
vi.mock("./components/ui/TabNav", () => ({ TabNav: ({ tabs, onChange }: { tabs: Array<{ id: string; label: string }>; onChange: (id: string) => void }) => <div>{tabs.map((tab) => <button key={tab.id} onClick={() => onChange(tab.id)}>{tab.label}</button>)}</div> }));
vi.mock("./components/ui/TabTip", () => ({ TabTip: ({ title, onAction }: { title: string; onAction?: () => void }) => <div><span>{title}</span>{onAction && <button onClick={onAction}>Tip action</button>}</div> }));
vi.mock("./components/ui/TutorialOverlay", () => ({ TutorialOverlay: ({ onClose, onNext, onBack, onSelectTutorial }: { onClose: () => void; onNext: () => void; onBack?: () => void; onSelectTutorial: (journey: string) => void }) => <div>Tutorial overlay<button onClick={onClose}>Close tutorial</button><button onClick={onNext}>Next tutorial</button>{onBack && <button onClick={onBack}>Back tutorial</button>}<button onClick={() => onSelectTutorial("fix-vulnerabilities")}>Switch tutorial</button></div> }));

function setDefaults(options: { activeTab?: string; resourceTab?: string; initialLoading?: boolean; activeJourney?: "prep-deployment" | null; activeResource?: string | null; emptyCounts?: boolean; scenarioNames?: string[]; journeyStep?: number; hasResourceInsight?: boolean } = {}) {
  const setActiveTab = vi.fn();
  const setResourceTab = vi.fn();
  mocks.tabRouting.mockReturnValue({
    activeTab: options.activeTab ?? "dashboard",
    resourceTab: options.resourceTab ?? "tier",
    setActiveTab,
    setResourceTab
  });
  mocks.secretsData.mockReturnValue({
    healthQuery: { data: options.initialLoading ? undefined : {}, isLoading: false },
    vaultQuery: { data: options.initialLoading ? undefined : { missing_secrets: options.emptyCounts ? [] : ["TOKEN"], resource_statuses: options.emptyCounts ? [] : [{ resource_name: "vault", secrets_missing: 1 }] }, isLoading: false },
    complianceQuery: { data: options.initialLoading ? undefined : { vulnerability_summary: { critical: options.emptyCounts ? 0 : 1, high: 0, medium: 0, low: 0 } }, isLoading: false },
    orientationQuery: { data: { hero_stats: { missing_secrets: options.emptyCounts ? 0 : 1 }, journeys: [{ id: "prep-deployment", title: "Prep deployment", description: "Prepare a bundle" }, { id: "fix-vulnerabilities", title: "Fix vulnerabilities", description: "Fix findings" }], tier_readiness: options.emptyCounts ? [] : [{ ready_percent: 50, strategized: 1, total: 2 }], resource_insights: options.hasResourceInsight ? [{ resource_name: "vault" }] : [], updated_at: "now" }, isLoading: false },
    isRefreshing: false,
    isInitialLoading: options.initialLoading ?? false,
    refreshAll: vi.fn()
  });
  mocks.vulnerabilities.mockReturnValue({ vulnerabilityQuery: { data: { vulnerabilities: [] }, isLoading: false, refetch: vi.fn() }, componentType: "all", componentFilter: "all", severityFilter: "all", componentOptions: [], setComponentType: vi.fn(), setComponentFilter: vi.fn(), setSeverityFilter: vi.fn() });
  mocks.resourcePanel.mockReturnValue({ activeResource: options.activeResource ?? null, selectedSecretKey: undefined, strategyTier: "tier-2-desktop", strategyHandling: "", strategyPrompt: "", strategyDescription: "", overrideReason: "", isOverrideMode: false, currentOverride: undefined, resourceDetailQuery: { data: undefined, isLoading: false, isFetching: false }, openResourcePanel: vi.fn(), closeResourcePanel: vi.fn(), setSelectedSecretKey: vi.fn(), setStrategyTier: vi.fn(), setStrategyHandling: vi.fn(), setStrategyPrompt: vi.fn(), setStrategyDescription: vi.fn(), setOverrideReason: vi.fn(), setIsOverrideMode: vi.fn(), handleSecretUpdate: vi.fn(), handleStrategyApply: vi.fn(), handleDeleteOverride: vi.fn(), handleVulnerabilityStatus: vi.fn() });
  mocks.journeys.mockImplementation((args) => {
    mocks.journeyOptions(args);
    return { activeJourney: options.activeJourney ?? null, journeyStep: options.journeyStep ?? 1, journeySteps: [{ content: "intro" }, { content: "configure" }, { content: "export" }], handleJourneySelect: vi.fn(), handleJourneyExit: vi.fn(), handleJourneyNext: vi.fn(), handleJourneyBack: vi.fn(), setJourneyStep: vi.fn(), journeyNextDisabled: false };
  });
  const scenarioNames = options.scenarioNames ?? ["secrets-manager"];
  mocks.scenarios.mockReturnValue({ search: "", setSearch: vi.fn(), query: { data: { scenarios: scenarioNames.map((name) => ({ name })) }, isLoading: false }, scenarios: scenarioNames.map((name) => ({ name })), filtered: [] });
  mocks.campaigns.mockReturnValue({ search: "", setSearch: vi.fn(), query: { isLoading: false }, readinessQuery: { isLoading: false }, filtered: [] });
  return { setActiveTab, setResourceTab };
}

// [REQ:SEC-UX-002] Guided operator journeys
// [REQ:SEC-UI-001] Operator dashboard
describe("App", () => {
  afterEach(cleanup);

  it("shows initial loading and dashboard guidance, including journey navigation", () => {
    const { setActiveTab } = setDefaults({ initialLoading: true, activeJourney: "prep-deployment" });
    renderWithProviders(<App />);
    expect(screen.getByText("Loading Security Dashboard")).toBeInTheDocument();
    expect(screen.getByText("Snapshot panel")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Tip action"));
    expect(setActiveTab).toHaveBeenCalledWith("resources");
    fireEvent.click(screen.getByText("Orientation hub"));
    expect(setActiveTab).toHaveBeenCalledWith("dashboard");
  });

  it("renders resources, deployment, and compliance tab content", () => {
    setDefaults({ activeTab: "resources", resourceTab: "tier" });
    const { rerender } = renderWithProviders(<App />);
    expect(screen.getByText("Tier readiness")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Per Resource"));

    setDefaults({ activeTab: "resources", resourceTab: "resource" });
    rerender(<App />);
    expect(screen.getByText("Resource table")).toBeInTheDocument();

    setDefaults({ activeTab: "deployment" });
    rerender(<App />);
    expect(screen.getByText("Campaigns panel")).toBeInTheDocument();
    expect(screen.getByText("Manifest workspace")).toBeInTheDocument();

    setDefaults({ activeTab: "compliance" });
    rerender(<App />);
    expect(screen.getByText("Compliance overview")).toBeInTheDocument();
    expect(screen.getByText("Security tables")).toBeInTheDocument();
  });

  it("handles empty readiness, campaign selection, a resource workbench, and tutorial controls", () => {
    setDefaults({ activeTab: "deployment", activeJourney: "prep-deployment", activeResource: "vault", emptyCounts: true, scenarioNames: ["api-gateway"], journeyStep: 2, hasResourceInsight: true });
    renderWithProviders(<App />);
    expect(screen.queryByText("Deployment prep needs strategies")).not.toBeInTheDocument();
    expect(screen.getByText("Resource panel")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Select campaign"));
    fireEvent.click(screen.getByText("Toggle campaigns"));
    fireEvent.click(screen.getByText("Change workspace scenario"));
    fireEvent.click(screen.getByText("Toggle workspace"));
    fireEvent.click(screen.getByText("Close resource"));
    fireEvent.click(screen.getByText("Switch resource"));

    const tutorialAnchor = document.createElement("div");
    tutorialAnchor.id = "anchor-campaigns";
    document.body.append(tutorialAnchor);
    const journeyCalls = mocks.journeyOptions.mock.calls;
    const journeyOptions = journeyCalls[journeyCalls.length - 1]?.[0] as { onStartTutorial: (journey: "prep-deployment", step?: number) => void };
    act(() => journeyOptions.onStartTutorial("prep-deployment", 1));
    expect(screen.getByText("Tutorial overlay")).toBeInTheDocument();
    expect(tutorialAnchor).toHaveAttribute("tabindex", "-1");
    fireEvent.click(screen.getByText("Next tutorial"));
    fireEvent.click(screen.getByText("Back tutorial"));
    fireEvent.click(screen.getByText("Switch tutorial"));
    fireEvent.click(screen.getByText("Close tutorial"));
    tutorialAnchor.remove();
  });
});
