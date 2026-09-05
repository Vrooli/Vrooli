import "@testing-library/jest-dom";
import { act, cleanup, fireEvent, render, renderHook, screen, waitFor } from "@testing-library/react";
import type { ComponentProps } from "react";
import { vi } from "vitest";

// provider-free-exception: preflight panels and action seams are fully controlled by props/mocked API calls.

const api = vi.hoisted(() => ({
  stopPortServices: vi.fn(),
  getDiskUsage: vi.fn(),
  runDiskCleanup: vi.fn(),
  stopScenarioProcesses: vi.fn(),
  openFirewallPorts: vi.fn(),
}));

vi.mock("../../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../../lib/api")>("../../lib/api");
  return { ...actual, ...api };
});

import {
  CHECK_DEFINITIONS,
  DiskUsageModal,
  PortStopModal,
  PreflightChecksPanel,
  StepPreflight,
  buildChecksToDisplay,
  buildReadOnlyChecks,
  usePreflightActions,
} from "./StepPreflight";
import type { DiskUsageResponse, PreflightCheck } from "../../lib/api";

const manifest = {
  scenario: { id: "demo-scenario" },
  target: { vps: { host: "vps.example.test", port: 2222, user: "deploy", key_path: "/tmp/key", workdir: "/srv/vrooli" } },
};

const checks: PreflightCheck[] = [
  { id: "ssh_connect", title: "SSH connectivity", status: "pass", details: "Connected" },
  { id: "ports_80_443", title: "Ports 80/443 availability", status: "fail", details: "Port listeners found", data: {
    port_bindings: JSON.stringify([
      { port: 80, process: "nginx", pid: 11, service: "nginx" },
      { port: 443, process: "proxy", pid: 12 },
    ]),
  } },
  { id: "firewall_inbound", title: "Inbound firewall rules", status: "fail", hint: "Ports are blocked" },
  { id: "dns_edge_apex", title: "Apex domain", status: "fail", hint: "Update DNS\n\n- Cloud: point to VPS" , data: { vps_ips: "203.0.113.10,203.0.113.11" } },
  { id: "stale_processes", title: "Stale process check", status: "warn", hint: "Existing processes" },
];

describe("StepPreflight support and action flows", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.stopPortServices.mockResolvedValue({ ok: true, message: "stopped" });
    api.getDiskUsage.mockResolvedValue({
      ok: true,
      free_space: "20 GB",
      free_bytes: 20,
      total_space: "100 GB",
      total_bytes: 100,
      used_percent: 80,
      largest_dirs: [{ path: "/var/lib/docker", size: "10 GB", bytes: 10 }],
      timestamp: "now",
    });
    api.runDiskCleanup.mockResolvedValue({ ok: true, space_freed: "1 GB" });
    api.stopScenarioProcesses.mockResolvedValue({ ok: true, message: "stopped" });
    api.openFirewallPorts.mockResolvedValue({ ok: true, message: "opened" });
  });

  it("builds complete display lists for pending, running, and read-only states", () => {
    const running = buildChecksToDisplay(checks, true);
    expect(running).toHaveLength(CHECK_DEFINITIONS.length);
    expect(running.find((check) => check.id === "ssh_connect")?.state).toBe("pass");
    expect(running.find((check) => check.id === "cmd_curl")?.state).toBe("running");
    expect(running.find((check) => check.id === "ports_80_443")?.details).toBe("Port listeners found");
    expect(buildChecksToDisplay(null, false).every((check) => check.state === "pending")).toBe(true);
    expect(buildReadOnlyChecks("pass")).toHaveLength(CHECK_DEFINITIONS.length);
  });

  it("renders check actions, multiline DNS instructions, and read-only suppression", () => {
    const onAction = vi.fn();
    render(<PreflightChecksPanel checksToDisplay={buildChecksToDisplay(checks, false)} onAction={onAction} />);
    fireEvent.click(screen.getByRole("button", { name: "Review & Stop" }));
    fireEvent.click(screen.getByRole("button", { name: "Open 80/443" }));
    fireEvent.click(screen.getByRole("button", { name: "Stop Scenario" }));
    fireEvent.click(screen.getByRole("button", { name: "Stop All" }));
    fireEvent.click(screen.getByRole("button", { name: "Show instructions" }));
    expect(screen.getByText("203.0.113.10")).toBeInTheDocument();
    expect(onAction).toHaveBeenCalledWith("ports_80_443", "stop_ports");
    expect(onAction).toHaveBeenCalledWith("firewall_inbound", "open_firewall");
    expect(onAction).toHaveBeenCalledWith("stale_processes", "stop_scenario");
    expect(onAction).toHaveBeenCalledWith("stale_processes", "stop_all");

    const readOnlyAction = vi.fn();
    cleanup();
    render(<PreflightChecksPanel checksToDisplay={buildChecksToDisplay(checks, false)} onAction={readOnlyAction} context="readonly" />);
    expect(screen.queryByRole("button", { name: "Review & Stop" })).not.toBeInTheDocument();
    expect(readOnlyAction).not.toHaveBeenCalled();
  });

  it("supports disk usage modal loading, cleanup actions, and close", async () => {
    const onCleanup = vi.fn();
    const onClose = vi.fn();
    const diskUsage: DiskUsageResponse = {
      ok: true,
      free_space: "20 GB",
      free_bytes: 20,
      total_space: "100 GB",
      total_bytes: 100,
      used_percent: 80,
      largest_dirs: [{ path: "/var/lib/docker", size: "10 GB", bytes: 10 }],
      timestamp: "now",
    };
    const { rerender } = render(<DiskUsageModal usage={null} loading onClose={onClose} onCleanup={onCleanup} cleanupLoading={false} />);
    expect(document.querySelector(".animate-spin")).toBeInTheDocument();
    rerender(<DiskUsageModal usage={diskUsage} loading={false} onClose={onClose} onCleanup={onCleanup} cleanupLoading={false} />);
    expect(screen.getByText("/var/lib/docker")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Clean apt cache" }));
    fireEvent.click(screen.getByRole("button", { name: "Run All" }));
    expect(onCleanup).toHaveBeenNthCalledWith(1, ["apt_clean"]);
    expect(onCleanup).toHaveBeenNthCalledWith(2, ["apt_clean", "journal_vacuum", "docker_prune", "tmp_clean"]);
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("renders port listeners and only confirms selected stop actions", () => {
    const onConfirm = vi.fn();
    const onToggleService = vi.fn();
    const onTogglePID = vi.fn();
    const onClose = vi.fn();
    const { rerender } = render(<PortStopModal
      bindings={[{ port: 80, service: "nginx", pid: 11, process: "nginx" }, { port: 443, pid: 12, process: "proxy" }]}
      selections={{ services: {}, pids: {} }}
      loading={false}
      onToggleService={onToggleService}
      onTogglePID={onTogglePID}
      onConfirm={onConfirm}
      onClose={onClose}
    />);
    expect(screen.getByText("port 80 - nginx - pid 11")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Stop Selected" })).toBeDisabled();
    fireEvent.click(screen.getByLabelText("Stop service nginx"));
    fireEvent.click(screen.getByLabelText("PID 12 - proxy, port 443"));
    expect(onToggleService).toHaveBeenCalledWith("nginx");
    expect(onTogglePID).toHaveBeenCalledWith(12);
    rerender(<PortStopModal bindings={[]} selections={{ services: {}, pids: {} }} loading={false} onToggleService={onToggleService} onTogglePID={onTogglePID} onConfirm={onConfirm} onClose={onClose} />);
    expect(screen.getByText(/No actionable service or PID data/)).toBeInTheDocument();
    rerender(<PortStopModal bindings={[]} selections={{ services: { nginx: true }, pids: {} }} loading={false} onToggleService={onToggleService} onTogglePID={onTogglePID} onConfirm={onConfirm} onClose={onClose} />);
    fireEvent.click(screen.getByRole("button", { name: "Stop Selected" }));
    expect(onConfirm).toHaveBeenCalledOnce();
  });

  it("runs firewall, disk, process, and port-stop actions through the shared SSH config", async () => {
    const onRecheck = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() => usePreflightActions({ manifest, sshKeyPath: null, preflightChecks: checks, onRecheck }));
    await act(async () => { await result.current.handleAction("firewall_inbound", "open_firewall"); });
    expect(api.openFirewallPorts).toHaveBeenCalledWith({ host: "vps.example.test", port: 2222, user: "deploy", key_path: "/tmp/key", ports: [80, 443] });
    await act(async () => { await result.current.handleAction("disk_free", "show_disk"); });
    expect(result.current.showDiskModal).toBe(true);
    await act(async () => { await result.current.handleCleanup(["apt_clean"]); });
    expect(api.runDiskCleanup).toHaveBeenCalled();
    await act(async () => { await result.current.handleAction("stale_processes", "stop_scenario"); });
    expect(api.stopScenarioProcesses).toHaveBeenCalledWith(expect.objectContaining({ scenario_id: "demo-scenario" }));
    await act(async () => { await result.current.handleAction("ports_80_443", "stop_ports"); });
    expect(result.current.showPortModal).toBe(true);
    await act(async () => { await result.current.handlePortStop(); });
    expect(api.stopPortServices).toHaveBeenCalledWith(expect.objectContaining({ services: ["nginx"], pids: [12] }));
    expect(onRecheck).toHaveBeenCalled();
  });

  it("reports missing SSH configuration and action failures", async () => {
    const { result } = renderHook(() => usePreflightActions({
      manifest: { target: { vps: { host: "vps.example.test" } } },
      sshKeyPath: null,
      preflightChecks: [],
    }));
    await act(async () => { await result.current.handleAction("firewall_inbound", "open_firewall"); });
    expect(result.current.actionError).toMatch(/SSH key/);

    const failedResult = { ok: false, message: "firewall denied" };
    api.openFirewallPorts.mockResolvedValueOnce(failedResult);
    const withKey = renderHook(() => usePreflightActions({
      manifest, sshKeyPath: "/tmp/key", preflightChecks: checks,
    }));
    await act(async () => { await withKey.result.current.handleAction("firewall_inbound", "open_firewall"); });
    expect(withKey.result.current.actionError).toBe("firewall denied");
    api.getDiskUsage.mockRejectedValueOnce(new Error("disk unavailable"));
    await act(async () => { await withKey.result.current.handleAction("disk_free", "show_disk"); });
    expect(withKey.result.current.actionError).toBe("Action failed: disk unavailable");
    api.getDiskUsage.mockResolvedValueOnce({ ok: true, free_space: "21 GB", free_bytes: 21, total_space: "100 GB", total_bytes: 100, used_percent: 79, largest_dirs: [], timestamp: "now" });
    api.runDiskCleanup.mockRejectedValueOnce(new Error("cleanup denied"));
    await act(async () => { await withKey.result.current.handleCleanup(["apt_clean"]); });
    expect(withKey.result.current.actionError).toBe("Cleanup failed: cleanup denied");
  });

  it("normalizes malformed port data and all read-only check states", () => {
    const portCheck = checks[1];
    if (!portCheck) throw new Error("expected port check");
    expect(buildChecksToDisplay([{ ...portCheck, data: { port_bindings: "not-json" } }], false)[0]?.details).toBeUndefined();
    for (const state of ["fail", "warn", "pending", "running"] as const) {
      expect(buildReadOnlyChecks(state).every((check) => check.state === state)).toBe(true);
    }
  });

  it("shows the top-level preflight success and failure/override states", () => {
    const runPreflight = vi.fn();
    const setPreflightOverride = vi.fn();
    const base: ComponentProps<typeof StepPreflight>["deployment"] = {
      preflightPassed: true,
      preflightChecks: [checks[0] ?? { id: "ssh_connect", title: "SSH connectivity", status: "pass" }],
      preflightError: null,
      isRunningPreflight: false,
      preflightOverride: false,
      setPreflightOverride,
      runPreflight,
      parsedManifest: { ok: true, value: manifest },
      sshKeyPath: null,
    } as unknown as ComponentProps<typeof StepPreflight>["deployment"];
    render(<StepPreflight deployment={base} />);
    expect(screen.getByText("Preflight Passed")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Run Preflight Checks" }));
    expect(runPreflight).toHaveBeenCalledOnce();
    const failed = { ...base, preflightPassed: false, preflightChecks: checks, preflightError: null };
    render(<StepPreflight deployment={failed} />);
    expect(screen.getByText(/checks failed/)).toBeInTheDocument();
    fireEvent.click(screen.getByText("Continue anyway despite failed checks"));
    expect(setPreflightOverride).toHaveBeenCalledWith(true);
  });

  it("covers disk, DNS, firewall, and process action variants", () => {
    const onAction = vi.fn();
    const actionChecks: PreflightCheck[] = [
      { id: "disk_free", title: "Disk space", status: "warn", details: "Low disk" },
      { id: "dns_edge_www", title: "WWW domain", status: "fail", hint: "Add the www record" },
      { id: "dns_do_origin", title: "Origin domain", status: "fail", hint: "- DNS: point origin\nAdditional context" },
      { id: "firewall_inbound", title: "Firewall", status: "fail", details: "Blocked" },
      { id: "stale_processes", title: "Stale processes", status: "warn", details: "Processes remain" },
      { id: "ports_80_443", title: "Ports", status: "fail", data: { port_bindings: JSON.stringify([{ port: 80, pid: 22 }]) } },
    ];
    render(<PreflightChecksPanel checksToDisplay={buildChecksToDisplay(actionChecks, false)} onAction={onAction} />);
    fireEvent.click(screen.getByRole("button", { name: "Show instructions" }));
    fireEvent.click(screen.getByRole("button", { name: "Open 80/443" }));
    fireEvent.click(screen.getByRole("button", { name: "Details" }));
    fireEvent.click(screen.getByRole("button", { name: "Open 80/443" }));
    fireEvent.click(screen.getByRole("button", { name: "Review & Stop" }));
    fireEvent.click(screen.getByRole("button", { name: "Stop Scenario" }));
    expect(onAction).toHaveBeenCalledWith("disk_free", "show_disk");
    expect(onAction).toHaveBeenCalledWith("firewall_inbound", "open_firewall");
    expect(onAction).toHaveBeenCalledWith("ports_80_443", "stop_ports");
  });

  it("handles empty disk data, loading port controls, and running/error wizard states", () => {
    const onClose = vi.fn();
    const onCleanup = vi.fn();
    const { rerender } = render(<DiskUsageModal usage={{ ok: false, free_space: "", free_bytes: 0, total_space: "", total_bytes: 0, used_percent: 0, largest_dirs: [], timestamp: "now" }} loading={false} onClose={onClose} onCleanup={onCleanup} cleanupLoading />);
    expect(screen.getByText("Disk Usage Details")).toBeInTheDocument();
    expect(screen.queryByText("Largest directories")).not.toBeInTheDocument();
    rerender(<PortStopModal bindings={[{ port: 80, pid: 9 }]} selections={{ services: {}, pids: { 9: true } }} loading onToggleService={vi.fn()} onTogglePID={vi.fn()} onConfirm={vi.fn()} onClose={onClose} />);
    expect(screen.getByRole("button", { name: /Stop Selected/ })).toBeDisabled();
    cleanup();
    const runPreflight = vi.fn();
    const base: ComponentProps<typeof StepPreflight>['deployment'] = {
      ...({} as any), preflightPassed: false, preflightChecks: null, preflightError: "VPS unavailable",
      isRunningPreflight: true, preflightOverride: true, setPreflightOverride: vi.fn(), runPreflight,
      parsedManifest: { ok: false, error: "invalid" }, sshKeyPath: null,
    } as any;
    render(<StepPreflight deployment={base} />);
    expect(screen.getByRole("button", { name: "Running Checks..." })).toBeInTheDocument();
    expect(screen.getByText("VPS unavailable")).toBeInTheDocument();
  });
});
