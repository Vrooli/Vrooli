import "@testing-library/jest-dom";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithProviders } from "./test-utils/renderWithProviders";

const state = vi.hoisted(() => ({
  selectedId: null as string | null,
  tab: "overview" as string,
  modal: null as string | null,
  deployments: [] as unknown[],
  record: null as any,
  progress: null as any,
  loading: false,
  error: null as Error | null,
  investigation: null as any,
}));

const mutation = vi.hoisted(() => () => ({ isPending: false, variables: undefined, error: null, mutate: vi.fn(), mutateAsync: vi.fn().mockResolvedValue({ run_id: "run-2" }) }));

vi.mock("./hooks/useDeployments", () => ({
  useDeployments: () => ({ data: state.loading ? undefined : state.deployments, isLoading: state.loading, error: state.error, refetch: vi.fn() }),
  useDeployment: () => ({ data: state.record, isLoading: state.loading, error: state.error, refetch: vi.fn() }),
  useInspectDeployment: mutation,
  useStopDeployment: mutation,
  useStartDeployment: mutation,
  useExecuteDeployment: mutation,
  useDeleteDeployment: mutation,
  getStatusInfo: (status: string) => ({
    label: status === "deployed" ? "Deployed" : status === "failed" ? "Failed" : status,
    color: status === "deployed" ? "emerald" : status === "failed" ? "red" : "blue",
    icon: status === "deployed" ? "check-circle" : status === "failed" ? "x-circle" : "loader",
  }),
}));

vi.mock("./hooks/useDeploymentListProgress", () => ({ useDeploymentListProgress: () => ({ progressMap: {}, isPolling: false }) }));
vi.mock("./hooks/useDeploymentProgress", () => ({
  useDeploymentProgress: () => ({ progress: state.progress, isConnected: true, connectionError: null, reset: vi.fn() }),
}));
vi.mock("./hooks/useLiveState", () => ({ useLiveState: () => ({ data: { system: { ssh: { verification_state: "authorized" } } } }) }));
vi.mock("./hooks/useInvestigation", () => ({
  useDeploymentInvestigation: () => ({
    activeInvestigation: state.investigation,
    isRunning: false,
    isStopping: false,
    isApplyingFixes: false,
    stop: vi.fn(),
    viewReport: vi.fn(),
    applyFixes: vi.fn(),
  }),
}));
vi.mock("./hooks/useDeploymentUrl", async () => {
  const React = await vi.importActual<typeof import("react")>("react");
  return {
    useDeploymentUrl: () => {
      const [, refresh] = React.useState(0);
      const update = (fn: () => void) => { fn(); refresh((value) => value + 1); };
      return {
        state: { deploymentId: state.selectedId, tab: state.tab, subtab: "processes", modal: state.modal, modalParams: {} },
        setTab: (tab: string) => update(() => { state.tab = tab; }),
        selectDeployment: (id: string | null) => update(() => { state.selectedId = id; }),
        openModal: (modal: string) => update(() => { state.modal = modal; }),
        closeModal: () => update(() => { state.modal = null; }),
      };
    },
  };
});
vi.mock("./components/wizard/SpawnAgentButton", () => ({ SpawnAgentButton: () => <button>Spawn agent</button> }));
vi.mock("./components/wizard/InvestigationProgress", () => ({ InvestigationProgress: () => <div>Investigation progress</div> }));
vi.mock("./components/wizard/InvestigationReport", () => ({ InvestigationReport: () => <div>Investigation report</div> }));
vi.mock("./components/wizard/StepBuild", () => ({ BuildStatusPanel: () => <div>Build status</div> }));
vi.mock("./components/wizard/DeploymentProgress", () => ({
  DeploymentProgressView: () => <div>Deployment progress</div>,
  getStepStatusFromProgress: () => "pending",
}));
vi.mock("./components/wizard/StepPreflight", () => ({
  buildChecksToDisplay: () => [],
  buildReadOnlyChecks: () => [],
  usePreflightActions: () => ({
    actionLoading: null, actionError: null, diskUsage: null, showDiskModal: false,
    setShowDiskModal: vi.fn(), cleanupLoading: false, showPortModal: false,
    setShowPortModal: vi.fn(), portSelections: {}, portBindings: [], handleAction: vi.fn(),
    handlePortStop: vi.fn(), togglePortService: vi.fn(), togglePortPID: vi.fn(), handleCleanup: vi.fn(),
  }),
  DiskUsageModal: () => <div>Disk modal</div>,
  PortStopModal: () => <div>Port modal</div>,
  PreflightChecksPanel: () => <div>Preflight checks</div>,
}));
vi.mock("./components/deployments/tabs", () => ({
  LiveStateTab: () => <div>Live state tab</div>, FilesTab: () => <div>Files tab</div>,
  DriftTab: () => <div>Drift tab</div>, SecretsTab: () => <div>Secrets tab</div>,
  HistoryTab: () => <div>History tab</div>, InvestigationsTab: () => <div>Investigations tab</div>,
  TerminalTab: () => <div>Terminal tab</div>,
}));
import { DeploymentDetails } from "./components/deployments/DeploymentDetails";
import { DeploymentsPage } from "./components/deployments/DeploymentsPage";

const record = {
  id: "dep-1", name: "Production Demo", scenario_id: "demo", status: "deployed",
  created_at: "2026-08-13T00:00:00Z", updated_at: "2026-08-14T00:00:00Z",
  last_deployed_at: "2026-08-14T00:00:00Z", last_inspected_at: "2026-08-14T00:00:00Z",
  run_id: "run-1", bundle_path: "/tmp/demo.tar.gz", bundle_sha256: "abc", bundle_size_bytes: 1024,
  manifest: { scenario: { id: "demo" }, edge: { domain: "demo.example" }, target: { vps: { host: "vps.example" } }, dependencies: { resources: ["postgres"], scenarios: ["helper"] } },
  setup_result: { ok: true }, deploy_result: { ok: true }, error_message: "old warning", error_step: "deploy",
  last_inspect_result: { scenario_logs: "api healthy" },
};

describe("operator orchestration surfaces", () => {
  beforeEach(() => {
    state.selectedId = null; state.tab = "overview"; state.modal = null;
    state.record = record;
    state.progress = null;
    state.loading = false; state.error = null; state.investigation = null;
    state.deployments = [{ ...record, progress_step: "deploy", progress_percent: 80 }];
    sessionStorage.clear();
  });

  it("renders deployment details and exercises every navigation and overview disclosure", async () => {
    const onBack = vi.fn();
    renderWithProviders(<DeploymentDetails deploymentId="dep-1" onBack={onBack} />);
    expect(screen.getByRole("heading", { name: "Production Demo" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Inspect" }));
    fireEvent.click(screen.getByRole("button", { name: "Stop" }));
    fireEvent.click(screen.getByRole("button", { name: "Back to Deployments" }));
    expect(onBack).toHaveBeenCalled();
    for (const name of ["Live State", "Files", "Drift", "Secrets", "History", "Investigations", "Terminal", "Overview"]) {
      fireEvent.click(screen.getByRole("button", { name }));
    }
    fireEvent.click(screen.getByRole("button", { name: "Deployment Manifest" }));
    fireEvent.click(screen.getByRole("button", { name: "Setup Result" }));
    fireEvent.click(screen.getByRole("button", { name: "Deploy Result" }));
    fireEvent.click(screen.getByRole("button", { name: "Logs" }));
    fireEvent.click(screen.getByRole("button", { name: "Re-deploy" }));
    expect(screen.getByText("Re-deploy configuration")).toBeInTheDocument();
    const redeploySwitches = screen.getAllByRole("switch");
    const redeployPreflightSwitch = redeploySwitches[1];
    if (!redeployPreflightSwitch) throw new Error("expected redeploy preflight switch");
    fireEvent.click(redeployPreflightSwitch);
    const redeployButtons = screen.getAllByRole("button", { name: "Re-deploy" });
    const redeployButton = redeployButtons[redeployButtons.length - 1];
    if (!redeployButton) throw new Error("expected redeploy action button");
    fireEvent.click(redeployButton);
    await waitFor(() => expect(state.modal).toBe("redeploy"));
    fireEvent.click(screen.getByRole("button", { name: "Close redeploy dialog" }));
  });

  it("covers detail loading and not-found states", () => {
    state.record = null;
    const { rerender } = renderWithProviders(<DeploymentDetails deploymentId="missing" onBack={vi.fn()} />);
    expect(screen.getByText("Deployment not found")).toBeInTheDocument();
    rerender(<DeploymentsPage onBack={vi.fn()} />);
    expect(screen.getByText("Production Demo")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Inspect/ }));
    fireEvent.click(screen.getByRole("button", { name: /Delete/ }));
  });

  it("covers loading, request errors, sparse records, and investigation report state", () => {
    state.loading = true;
    const { rerender } = renderWithProviders(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    expect(document.querySelector("svg.animate-spin")).toBeInTheDocument();
    state.loading = false;
    state.error = new Error("request failed");
    rerender(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    expect(screen.getByText("request failed")).toBeInTheDocument();
    state.error = null;
    state.record = {
      ...record,
      status: "pending",
      manifest: {},
      last_deployed_at: null,
      last_inspected_at: null,
      setup_result: null,
      deploy_result: null,
      last_inspect_result: null,
      error_message: null,
      dependencies: undefined,
      bundle_path: null,
    };
    rerender(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    expect(screen.queryByText("Dependencies")).not.toBeInTheDocument();
    state.investigation = {
      id: "inv-1", deployment_run_id: "old-run", created_at: "2020-01-01T00:00:00Z",
      status: "completed", findings: [], summary: "stale",
    };
    state.modal = "investigation-report";
    rerender(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    expect(screen.getByText("Investigation report")).toBeInTheDocument();
  });

  it("covers deployment list empty, error, selection, and deletion choices", async () => {
    state.deployments = [];
    const back = vi.fn();
    const { rerender } = renderWithProviders(<DeploymentsPage onBack={back} />);
    expect(screen.getByText("No deployments yet")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Go to Dashboard" }));
    expect(back).toHaveBeenCalled();
    state.deployments = [{ ...record, status: "failed" }];
    rerender(<DeploymentsPage onBack={back} />);
    fireEvent.click(screen.getByRole("button", { name: /Delete/ }));
    fireEvent.click(screen.getByLabelText(/stop the scenario/i));
    fireEvent.click(screen.getByLabelText(/delete associated/i));
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    await waitFor(() => expect(screen.queryByText("Delete Deployment")).not.toBeInTheDocument());
  });

  it("covers failed/stopped actions and redeploy step-state branches", () => {
    state.record = { ...record, status: "failed", bundle_path: null };
    state.progress = {
      currentStep: "bundle_build", isComplete: false, error: null,
      steps: [{ id: "bundle_build", status: "running" }, { id: "deploy", status: "pending" }],
    };
    const { rerender } = renderWithProviders(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(screen.getByText("Re-deploy configuration")).toBeInTheDocument();
    const switches = screen.getAllByRole("switch");
    const preflightSwitch = switches[1];
    if (!preflightSwitch) throw new Error("expected preflight switch");
    fireEvent.click(preflightSwitch);
    fireEvent.click(screen.getByRole("button", { name: /Build/ }));
    fireEvent.click(screen.getByRole("button", { name: "Close redeploy dialog" }));
    state.record = { ...record, status: "stopped" };
    rerender(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: "Start" }));
    state.record = { ...record, status: "deploying" };
    rerender(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: "View Progress" }));
    expect(screen.getAllByText("Deployment").length).toBeGreaterThan(0);
  });

  it("covers deployment list loading, errors, refresh, live progress, and card links", () => {
    state.loading = true;
    const { rerender } = renderWithProviders(<DeploymentsPage onBack={vi.fn()} />);
    expect(document.querySelector("svg.animate-spin")).toBeInTheDocument();
    state.loading = false;
    state.error = new Error("deployment list unavailable");
    rerender(<DeploymentsPage onBack={vi.fn()} />);
    expect(screen.getByText(/deployment list unavailable/)).toBeInTheDocument();
    state.error = null;
    state.deployments = [
      { ...record, status: "deployed", domain: "live.example", progress_step: undefined, progress_percent: undefined },
      { ...record, id: "pending", status: "pending", domain: null, progress_step: undefined, progress_percent: undefined },
    ];
    rerender(<DeploymentsPage onBack={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    expect(screen.getByTitle("Open in browser")).toHaveAttribute("href", "https://live.example");
    expect(screen.getByText("Starting...")).toBeInTheDocument();
  });

  it("covers redeploy preflight step states and bundle fallback branches", () => {
    state.record = { ...record, status: "deployed", bundle_path: null, bundle_sha256: null, bundle_size_bytes: null };
    state.modal = "redeploy";
    state.progress = {
      currentStep: "preflight", isComplete: false, error: null,
      steps: [{ id: "bundle_build", status: "completed" }, { id: "preflight", status: "running" }, { id: "deploy", status: "pending" }],
    };
    const { rerender } = renderWithProviders(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    const switches = screen.getAllByRole("switch");
    const preflightSwitch = switches.find((element) => element.parentElement?.parentElement?.textContent?.includes("Run preflight checks"));
    if (!preflightSwitch) throw new Error("expected preflight switch");
    fireEvent.click(preflightSwitch);
    expect(screen.getByText(/Run preflight checks, then/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Preflight/ }));
    expect(screen.getByText("Preflight running")).toBeInTheDocument();
    state.progress = { ...state.progress, error: "Preflight failed", isComplete: true };
    rerender(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    expect(screen.getByText("Preflight failed")).toBeInTheDocument();
    state.progress = { ...state.progress, currentStep: "deploy", error: undefined, isComplete: true };
    rerender(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    expect(screen.getByText("Preflight complete")).toBeInTheDocument();
  });
});
