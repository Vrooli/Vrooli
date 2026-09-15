import "@testing-library/jest-dom";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";

import { BundleInventory } from "./components/deployments/tabs/BundleInventory";
import { DriftTab } from "./components/deployments/tabs/DriftTab";
import { FilesTab } from "./components/deployments/tabs/FilesTab";
import { HistoryTab } from "./components/deployments/tabs/HistoryTab";
import { InvestigationsTab } from "./components/deployments/tabs/InvestigationsTab";
import { LiveStateTab } from "./components/deployments/tabs/LiveStateTab";
import { ProcessCards } from "./components/deployments/tabs/ProcessCards";
import { TerminalTab } from "./components/deployments/tabs/TerminalTab";
import { VPSManagement } from "./components/deployments/tabs/VPSManagement";
import { SystemResources } from "./components/deployments/tabs/SystemResources";
import { renderWithProviders } from "./test-utils/renderWithProviders";

const mocks = vi.hoisted(() => ({
  useFiles: vi.fn(),
  useFileContent: vi.fn(),
  useLiveState: vi.fn(),
  useDrift: vi.fn(),
  useHistory: vi.fn(),
  useLogs: vi.fn(),
  useRestartProcess: vi.fn(),
  useKillProcess: vi.fn(),
  useProcessControl: vi.fn(),
  useVPSAction: vi.fn(),
  useInvestigations: vi.fn(),
  listBundles: vi.fn(),
  getBundleStats: vi.fn(),
  cleanupBundles: vi.fn(),
}));

vi.mock("./hooks/useLiveState", async () => {
  const actual = await vi.importActual<typeof import("./hooks/useLiveState")>("./hooks/useLiveState");
  return {
    ...actual,
    useFiles: mocks.useFiles,
    useFileContent: mocks.useFileContent,
    useLiveState: mocks.useLiveState,
    useDrift: mocks.useDrift,
    useHistory: mocks.useHistory,
    useLogs: mocks.useLogs,
    useRestartProcess: mocks.useRestartProcess,
    useKillProcess: mocks.useKillProcess,
    useProcessControl: mocks.useProcessControl,
    useVPSAction: mocks.useVPSAction,
  };
});

vi.mock("./hooks/useInvestigation", async () => {
  const actual = await vi.importActual<typeof import("./hooks/useInvestigation")>("./hooks/useInvestigation");
  return { ...actual, useInvestigations: mocks.useInvestigations };
});

vi.mock("./lib/api", async () => {
  const actual = await vi.importActual<typeof import("./lib/api")>("./lib/api");
  return {
    ...actual,
    listBundles: mocks.listBundles,
    getBundleStats: mocks.getBundleStats,
    cleanupBundles: mocks.cleanupBundles,
  };
});

function idleMutation() {
  return { mutate: vi.fn(), mutateAsync: vi.fn().mockResolvedValue({ ok: true, message: "done" }), isPending: false };
}

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
  mocks.useRestartProcess.mockReturnValue(idleMutation());
  mocks.useKillProcess.mockReturnValue(idleMutation());
  mocks.useProcessControl.mockReturnValue(idleMutation());
  mocks.useVPSAction.mockReturnValue(idleMutation());
});

describe("deployment operator surfaces", () => {
  it("covers file explorer navigation, selection, resizing, and bundle inventory", async () => {
    const refetch = vi.fn();
    mocks.useFiles.mockReturnValue({
      data: {
        path: "/srv/vrooli",
        entries: [
          { name: "app", type: "directory", size_bytes: 0, modified: "", permissions: "" },
          { name: "config.json", type: "file", size_bytes: 32, modified: "", permissions: "" },
        ],
      },
      isLoading: false,
      error: null,
      refetch,
      isFetching: false,
    });
    mocks.useFileContent.mockReturnValue({
      data: { path: "/srv/vrooli/config.json", content: "{}", sizeBytes: 2, truncated: false },
      isLoading: false,
      error: null,
    });
    mocks.listBundles.mockResolvedValue({
      bundles: [
        { scenario_id: "demo", path: "/tmp/old.tgz", sha256: "old-hash", size_bytes: 1024, created_at: "2026-01-01T00:00:00Z" },
        { scenario_id: "demo", path: "/tmp/new.tgz", sha256: "new-hash", size_bytes: 2048, created_at: "2026-02-01T00:00:00Z" },
      ],
    });
    mocks.getBundleStats.mockResolvedValue({ stats: { total_count: 2, total_size_bytes: 3072 } });
    mocks.cleanupBundles.mockResolvedValue({ local_deleted: ["old"], local_freed_bytes: 1024 });

    const { container } = renderWithProviders(<FilesTab deploymentId="deployment-1" />);
    expect(await screen.findByText("Path: /srv/vrooli")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /config.json/ }));
    expect(await screen.findByText("config.json")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("separator", { name: "Resize panels" }));
    fireEvent.mouseMove(window, { clientX: 600 });
    fireEvent.mouseUp(window);
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    expect(refetch).toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Bundles" }));
    expect(await screen.findByText("Bundle Inventory")).toBeInTheDocument();
    expect(await screen.findByText("new-hash")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Cleanup Old/ }));
    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
    await waitFor(() => expect(screen.getByText(/Cleaned up 1 bundles/)).toBeInTheDocument());
    expect(container).toHaveTextContent("Bundle Inventory");
  });

  it("covers history filters and the investigations tree including orphan fixes", () => {
    mocks.useHistory.mockReturnValue({
      data: [{ type: "deploy_completed", timestamp: new Date().toISOString(), message: "ready", success: true }],
      isLoading: false,
      error: null,
      refetch: vi.fn(),
      isFetching: false,
    });
    mocks.useLogs.mockReturnValue({
      data: { logs: [{ timestamp: new Date().toISOString(), source: "api", level: "INFO", message: "healthy" }], total: 1, sources: ["api", "worker"] },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
      isFetching: false,
    });
    const parent = { id: "inv-1", status: "completed", created_at: "2026-01-01T00:00:00Z", deployment_run_id: "old" };
    const fix = { id: "fix-1", status: "failed", created_at: "2026-01-02T00:00:00Z", source_investigation_id: "inv-1", error_message: "could not apply" };
    const orphan = { id: "orphan", status: "cancelled", created_at: "2026-01-03T00:00:00Z", source_investigation_id: "missing" };
    mocks.useInvestigations.mockReturnValue({ data: [parent, fix, orphan], isLoading: false, error: null, refetch: vi.fn(), isFetching: false });

    renderWithProviders(<HistoryTab deploymentId="deployment-1" />);
    expect(screen.getByText("ready")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Logs" }));
    expect(screen.getByPlaceholderText("Search logs...")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Search logs..."), { target: { value: "health" } });
    const filters = screen.getAllByRole("combobox");
    const [levelFilter, severityFilter, statusFilter] = filters;
    if (!levelFilter || !severityFilter || !statusFilter) throw new Error("expected log filters");
    fireEvent.change(levelFilter, { target: { value: "worker" } });
    fireEvent.change(severityFilter, { target: { value: "ERROR" } });
    fireEvent.change(statusFilter, { target: { value: "500" } });

    const onViewReport = vi.fn();
    renderWithProviders(
      <InvestigationsTab deploymentId="deployment-1" deploymentRunId="current" lastDeployedAt="2026-01-01T12:00:00Z" onViewReport={onViewReport} />,
    );
    expect(screen.getByText("2 investigations")).toBeInTheDocument();
    const investigationButton = screen.getAllByRole("button").find(
      (button) => button.textContent?.includes("Investigation") && !button.textContent?.includes("Fix Application"),
    );
    if (!investigationButton) throw new Error("expected investigation button");
    fireEvent.click(investigationButton);
    expect(onViewReport).toHaveBeenCalledWith(parent);
    fireEvent.click(screen.getByRole("button", { name: /Fix Application/ }));
    expect(onViewReport).toHaveBeenCalledWith(fix);
  });

  it("covers investigation loading, errors, empty state, statuses, and keyboard activation", () => {
    mocks.useInvestigations.mockReturnValue({ data: undefined, isLoading: true, error: null, refetch: vi.fn(), isFetching: false });
    renderWithProviders(<InvestigationsTab deploymentId="deployment-1" onViewReport={vi.fn()} />);
    expect(document.querySelector("svg.animate-spin")).toBeInTheDocument();
    mocks.useInvestigations.mockReturnValue({ data: undefined, isLoading: false, error: new Error("investigations unavailable"), refetch: vi.fn(), isFetching: false });
    renderWithProviders(<InvestigationsTab deploymentId="deployment-1" onViewReport={vi.fn()} />);
    expect(screen.getByText(/Failed to load investigations/)).toBeInTheDocument();
    mocks.useInvestigations.mockReturnValue({ data: [], isLoading: false, error: null, refetch: vi.fn(), isFetching: false });
    renderWithProviders(<InvestigationsTab deploymentId="deployment-1" onViewReport={vi.fn()} />);
    expect(screen.getByText("No investigations yet")).toBeInTheDocument();
    const onViewReport = vi.fn();
    mocks.useInvestigations.mockReturnValue({
      data: [
        { id: "complete", status: "completed", created_at: "2026-01-01T00:00:00Z" },
        { id: "failed", status: "failed", created_at: "2026-01-01T00:00:00Z" },
        { id: "cancelled", status: "cancelled", created_at: "2026-01-01T00:00:00Z" },
        { id: "running", status: "running", created_at: "2026-01-01T00:00:00Z" },
        { id: "pending", status: "pending", created_at: "2026-01-01T00:00:00Z" },
      ], isLoading: false, error: null, refetch: vi.fn(), isFetching: false,
    });
    renderWithProviders(<InvestigationsTab deploymentId="deployment-1" onViewReport={onViewReport} />);
    const complete = screen.getAllByRole("button", { name: /Investigation/ })[0];
    if (!complete) throw new Error("expected completed investigation button");
    fireEvent.keyDown(complete, { key: "Enter" });
    fireEvent.keyDown(complete, { key: " " });
    expect(onViewReport).toHaveBeenCalledTimes(2);
  });

  it("covers drift actions, live-state navigation, and process cards", () => {
    const restart = idleMutation();
    const kill = idleMutation();
    mocks.useRestartProcess.mockReturnValue(restart);
    mocks.useKillProcess.mockReturnValue(kill);
    mocks.useDrift.mockReturnValue({
      data: {
        summary: { passed: 1, warnings: 1, drifts: 1 },
        checks: [
          { category: "scenarios", name: "demo", status: "drift", expected: "running", actual: "stopped PID 123", message: "restart needed", actions: ["restart_scenario", "kill_pid"] },
          { category: "resources", name: "postgres", status: "warning", expected: "healthy", actual: "degraded", actions: ["restart_resource"] },
          { category: "ports", name: "8080", status: "pass", expected: "open", actual: "open", actions: ["stop"] },
          { category: "edge", name: "tls", status: "pass", expected: "valid", actual: "valid" },
        ],
      },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
      isFetching: false,
    });
    renderWithProviders(<DriftTab deploymentId="deployment-1" />);
    expect(screen.getByText("1 passed")).toBeInTheDocument();
    const restartButton = screen.getAllByRole("button", { name: "Restart" })[0];
    if (!restartButton) throw new Error("expected restart button");
    fireEvent.click(restartButton);
    fireEvent.click(screen.getByRole("button", { name: "Kill" }));
    expect(restart.mutate).toHaveBeenCalled();
    expect(kill.mutate).toHaveBeenCalledWith({ pid: 123, signal: "TERM" });

    mocks.useLiveState.mockReturnValue({
      data: {
        ok: true,
        timestamp: new Date().toISOString(),
        sync_duration_ms: 12,
        processes: { scenarios: [], resources: [], unexpected: [] },
        expected: [],
        ports: [],
        system: {
          cpu: { cores: 4, model: "test-cpu", usage_percent: 10, load_average: [0.1, 0.2, 0.3] },
          memory: { total_mb: 4096, used_mb: 1024, free_mb: 3072, usage_percent: 25 },
          disk: { total_gb: 100, used_gb: 20, free_gb: 80, usage_percent: 20 },
          swap: { total_mb: 1024, used_mb: 128, usage_percent: 12 },
          ssh: { connected: true, latency_ms: 5, verification_state: "authorized", auth_mode: "explicit_key" },
          uptime_seconds: 8,
        },
        caddy: { status: "running", version: "2", sites: [] },
      },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
      isFetching: false,
    });
    renderWithProviders(<LiveStateTab deploymentId="deployment-1" deploymentName="Demo" />);
    expect(screen.getByText(/Last synced/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Network" }));
    expect(screen.getByText("No listening ports detected")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "System" }));
    expect(screen.getByText(/CPU/)).toBeInTheDocument();

    const processMutation = idleMutation();
    mocks.useProcessControl.mockReturnValue(processMutation);
    renderWithProviders(
      <ProcessCards
        deploymentId="deployment-1"
        processes={{
          scenarios: [{ id: "demo", status: "running", pid: 1, uptime_seconds: 3600, resources: { cpu_percent: 1, memory_mb: 20, memory_percent: 2 }, ports: [{ port: 3000, responding: true }] }, { id: "stopped", status: "stopped", pid: 2, uptime_seconds: 0, resources: { cpu_percent: 0, memory_mb: 0, memory_percent: 0 }, ports: [] }],
          resources: [{ id: "postgres", status: "running", pid: 3, port: 5432, uptime_seconds: 60, resources: { cpu_percent: 1, memory_mb: 30, memory_percent: 3 } }],
          unexpected: [{ pid: 99, command: "rogue" }],
        } as never}
        expected={[{ type: "scenario", id: "missing", state: "stopped" }] as never}
      />,
    );
    screen.getAllByRole("button", { name: "Stop" }).forEach((button) => fireEvent.click(button));
    screen.getAllByRole("button", { name: "Start" }).forEach((button) => fireEvent.click(button));
    screen.getAllByRole("button", { name: /Kill/ }).forEach((button) =>
      fireEvent.click(button),
    );
    expect(processMutation.mutate).toHaveBeenCalled();
  });

  it("covers VPS confirmation flows and terminal websocket interaction", async () => {
    const vpsMutation = idleMutation();
    mocks.useVPSAction.mockReturnValue(vpsMutation);
    renderWithProviders(<VPSManagement deploymentId="deployment-1" deploymentName="Demo" />);
    fireEvent.click(screen.getByRole("button", { name: /Restart VPS Reboot/ }));
    fireEvent.change(screen.getByPlaceholderText("REBOOT"), { target: { value: "REBOOT" } });
    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
    await waitFor(() => expect(vpsMutation.mutateAsync).toHaveBeenCalled());
    fireEvent.click(screen.getByRole("button", { name: /Docker Reset Level 3/ }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

    class FakeSocket {
      static OPEN = 1;
      readyState = FakeSocket.OPEN;
      send = vi.fn();
      close = vi.fn();
      onopen?: () => void;
      onmessage?: (event: MessageEvent) => void;
      onerror?: () => void;
      onclose?: () => void;
      constructor() { setTimeout(() => this.onopen?.(), 0); }
    }
    vi.stubGlobal("WebSocket", FakeSocket);
    renderWithProviders(<TerminalTab deploymentId="deployment-1" />);
    fireEvent.click(screen.getByRole("button", { name: "Connect" }));
    await waitFor(() => expect(screen.getByText("Connected")).toBeInTheDocument());
    const socket = (FakeSocket as unknown as { last?: FakeSocket }).last;
    void socket;
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "ls" } });
    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });
    fireEvent.click(screen.getByRole("button", { name: "Disconnect" }));
  });

  it("covers empty process groups, missing setup variants, and pending controls", () => {
    const pendingMutation = { mutate: vi.fn(), mutateAsync: vi.fn(), isPending: true };
    mocks.useProcessControl.mockReturnValue(pendingMutation);
    mocks.useKillProcess.mockReturnValue(pendingMutation);
    renderWithProviders(
      <ProcessCards
        deploymentId="deployment-1"
        processes={{ scenarios: [], resources: [], unexpected: [{ pid: 9, command: "rogue", user: "root", port: 9999 }] } as never}
        expected={[
          { type: "scenario", id: "needs-scenario", state: "needs_setup" },
          { type: "resource", id: "needs-resource", state: "needs_setup" },
          { type: "resource", id: "stopped-resource", state: "stopped" },
        ] as never}
      />,
    );
    expect(screen.getByText("No scenarios running")).toBeInTheDocument();
    expect(screen.getByText("No resources running")).toBeInTheDocument();
    expect(screen.getAllByText("needs setup")).toHaveLength(2);
    expect(screen.getByText("stopped")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Setup" })).toBeDisabled();
    expect(screen.getAllByRole("button", { name: "Start" }).every((button) => (button as HTMLButtonElement).disabled)).toBe(true);
    expect(screen.getByRole("button", { name: "Kill" })).toBeDisabled();
  });

  it("covers operator error and empty states plus resource authorization variants", () => {
    mocks.useFiles.mockReturnValue({ data: undefined, isLoading: true, error: null, refetch: vi.fn(), isFetching: false });
    renderWithProviders(<FilesTab deploymentId="deployment-1" />);
    expect(document.querySelector("svg.animate-spin")).toBeTruthy();
    mocks.useFiles.mockReturnValue({ data: undefined, isLoading: false, error: new Error("files unavailable"), refetch: vi.fn(), isFetching: false });
    renderWithProviders(<FilesTab deploymentId="deployment-1" />);
    expect(screen.getByText(/Failed to load files/)).toBeInTheDocument();

    mocks.useLiveState.mockReturnValue({ data: undefined, isLoading: true, error: null, refetch: vi.fn(), isFetching: false });
    renderWithProviders(<LiveStateTab deploymentId="deployment-1" />);
    expect(document.querySelector("svg.animate-spin")).toBeTruthy();
    mocks.useLiveState.mockReturnValue({ data: undefined, isLoading: false, error: new Error("offline"), refetch: vi.fn(), isFetching: false });
    renderWithProviders(<LiveStateTab deploymentId="deployment-1" />);
    expect(screen.getByText(/Failed to load live state/)).toBeInTheDocument();
    mocks.useLiveState.mockReturnValue({ data: { ok: false, error: "VPS rejected the request" }, isLoading: false, error: null, refetch: vi.fn(), isFetching: false });
    renderWithProviders(<LiveStateTab deploymentId="deployment-1" />);
    expect(screen.getByText("VPS rejected the request")).toBeInTheDocument();

    const base = {
      cpu: { cores: 2, model: "", usage_percent: 90, load_average: [] },
      memory: { total_mb: 1000, used_mb: 900, free_mb: 100, usage_percent: 90 },
      disk: { total_gb: 10, used_gb: 9, free_gb: 1, usage_percent: 90 },
      swap: { total_mb: 0, used_mb: 0, usage_percent: 0 }, uptime_seconds: 65,
      ssh: { connected: false, latency_ms: 0, verification_state: "unknown", auth_mode: "agent", key_path: "/tmp/id" },
    };
    renderWithProviders(<SystemResources system={base as any} />);
    expect(screen.getByText("Not configured")).toBeInTheDocument();
    expect(screen.getAllByText("Disconnected").length).toBeGreaterThan(0);
    const unauthorized = { ...base, ssh: { ...base.ssh, connected: true, verification_state: "unauthorized", error: "bad key" }, swap: { total_mb: 100, used_mb: 60, usage_percent: 60 } };
    renderWithProviders(<SystemResources system={unauthorized as any} />);
    expect(screen.getByText("Unauthorized")).toBeInTheDocument();
  });
});
