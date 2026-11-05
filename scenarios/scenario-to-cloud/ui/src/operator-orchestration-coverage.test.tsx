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

const executeResult = vi.hoisted(() => ({ operation_id: "op-2", plan_digest: "sha256:plan", state: "admitted", timestamp: "2026-09-09T00:00:00Z" }));
const mutationError = vi.hoisted(() => ({ current: null as Error | null }));
const mutateAsync = vi.hoisted(() => vi.fn());
const mutate = vi.hoisted(() => vi.fn());
const mutation = vi.hoisted(() => () => ({ isPending: false, variables: undefined, error: mutationError.current, mutate, mutateAsync }));
const consoleProps = vi.hoisted(() => ({ last: null as any }));

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
vi.mock("./components/deployments/console/DeploymentConsole", () => ({
  DeploymentConsole: (props: any) => {
    consoleProps.last = props;
    return <div data-testid="console-stub">Console for {props.deployment.id}{props.attachRequest ? ` attached ${props.attachRequest.operation_id}` : ""}</div>;
  },
}));
vi.mock("./hooks/useLiveState", () => ({
  useLiveState: () => ({ data: { system: { ssh: { verification_state: "authorized" } } } }),
  useHealthObservation: () => ({ data: null }),
}));
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
vi.mock("./components/deployments/tabs", () => ({
  LiveStateTab: () => <div>Live state tab</div>, FilesTab: () => <div>Files tab</div>,
  DriftTab: () => <div>Drift tab</div>, SecretsTab: () => <div>Secrets tab</div>,
  HistoryTab: () => <div>History tab</div>, InvestigationsTab: () => <div>Investigations tab</div>,
  TerminalTab: () => <div>Terminal tab</div>,
  HealthObservationBadge: () => null,
}));
import { DeploymentDetails } from "./components/deployments/DeploymentDetails";
import { DeploymentsPage } from "./components/deployments/DeploymentsPage";

const record = {
  id: "dep-1", name: "Production Demo", scenario_id: "demo", status: "deployed",
  created_at: "2026-08-13T00:00:00Z", updated_at: "2026-08-14T00:00:00Z",
  last_deployed_at: "2026-08-14T00:00:00Z", last_inspected_at: "2026-08-14T00:00:00Z",
  environment: "production", target: { machine_id: "m-1", node_id: "n-1", enrollment_generation: 3, transport: "bridge", locator: { host: "vps.example" } }, fence: 4,
  bundle_path: "/tmp/demo.tar.gz", bundle_sha256: "abc", bundle_size_bytes: 1024,
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
    mutationError.current = null;
    mutateAsync.mockReset();
    mutate.mockReset();
    mutateAsync.mockResolvedValue(executeResult);
    consoleProps.last = null;
    sessionStorage.clear();
  });

  // [REQ:STC-P0-038] The identity header answers "which environment and
  // machine" and the advanced surfaces stay behind one disclosure.
  it("renders the identity header, keeps advanced surfaces behind a disclosure, and navigates tabs", async () => {
    const onBack = vi.fn();
    renderWithProviders(<DeploymentDetails deploymentId="dep-1" onBack={onBack} />);
    expect(screen.getByRole("heading", { name: "Production Demo" })).toBeInTheDocument();
    expect(screen.getByTestId("console-identity-environment")).toHaveTextContent("production");
    expect(screen.getByTestId("console-identity-target")).toHaveTextContent("machine:m-1");
    expect(screen.getByTestId("console-identity-transport")).toHaveTextContent("bridge");
    expect(screen.getByTestId("console-identity-id")).toHaveTextContent("dep-1");
    expect(screen.getByTestId("console-identity-fence")).toHaveTextContent("4");
    expect(screen.getByTestId("console-stub")).toHaveTextContent("Console for dep-1");
    expect(screen.queryByRole("tab", { name: "Terminal" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Inspect" }));
    expect(mutate).toHaveBeenCalledWith("dep-1");
    fireEvent.click(screen.getByRole("button", { name: "Back to deployments" }));
    expect(onBack).toHaveBeenCalled();

    const toggle = screen.getByTestId("console-advanced-toggle");
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    for (const name of ["Live State", "Files", "Drift", "Secrets", "History", "Investigations", "Terminal", "Overview"]) {
      fireEvent.click(screen.getByRole("tab", { name }));
    }
    fireEvent.click(screen.getByRole("button", { name: "Deployment Manifest" }));
    fireEvent.click(screen.getByRole("button", { name: "Setup Result" }));
    fireEvent.click(screen.getByRole("button", { name: "Deploy Result" }));
    fireEvent.click(screen.getByRole("button", { name: "Logs" }));
    expect(screen.getByText("api healthy")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("console-stop"));
    fireEvent.click(screen.getByTestId("console-start"));
    expect(mutate).toHaveBeenCalledTimes(3);
  });

  it("opens the disclosure for a deep-linked advanced tab and shows refused actions", () => {
    state.tab = "terminal";
    mutationError.current = new Error("forbidden_scope: needs scenario-to-cloud:destructive");
    renderWithProviders(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    expect(screen.getByTestId("console-advanced-toggle")).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("Terminal tab")).toBeInTheDocument();
    expect(screen.getByText(/needs scenario-to-cloud:destructive/)).toBeInTheDocument();
  });

  // [REQ:STC-P0-039] The legacy pipeline is previewed, confirmed by typing
  // the short id, and its durable operation is handed to the console.
  it("runs the legacy pipeline through the destructive preview and attaches the operation", async () => {
    renderWithProviders(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    fireEvent.click(screen.getByTestId("console-advanced-toggle"));
    fireEvent.click(screen.getByRole("switch", { name: "Build a new bundle" }));
    fireEvent.click(screen.getByTestId("console-run-pipeline"));
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveTextContent("machine:m-1");
    expect(dialog).toHaveTextContent("build new bundle");
    expect(screen.getByTestId("console-destructive-confirm")).toBeDisabled();
    fireEvent.change(screen.getByTestId("console-destructive-confirm-input"), { target: { value: "dep-1".slice(0, 8) } });
    fireEvent.click(screen.getByTestId("console-destructive-confirm"));
    await waitFor(() => expect(mutateAsync).toHaveBeenCalledWith({ id: "dep-1", options: { forceBundleBuild: true, runPreflight: false } }));
    await waitFor(() => expect(screen.getByTestId("console-stub")).toHaveTextContent("attached op-2"));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
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
    expect(screen.getByRole("status", { name: "Loading deployment" })).toBeInTheDocument();
    state.loading = false;
    state.error = new Error("request failed");
    rerender(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    expect(screen.getByText("request failed")).toBeInTheDocument();
    state.error = null;
    state.record = {
      ...record,
      status: "pending",
      manifest: {},
      environment: undefined,
      target: undefined,
      fence: undefined,
      last_deployed_at: null,
      last_inspected_at: null,
      setup_result: null,
      deploy_result: null,
      last_inspect_result: null,
      error_message: null,
      bundle_path: null,
    };
    rerender(<DeploymentDetails deploymentId="dep-1" onBack={vi.fn()} />);
    expect(screen.getByTestId("console-identity-target")).toHaveTextContent("unbound");
    fireEvent.click(screen.getByTestId("console-advanced-toggle"));
    expect(screen.queryByText("Dependencies")).not.toBeInTheDocument();
    state.investigation = {
      id: "inv-1", created_at: "2020-01-01T00:00:00Z",
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
});
