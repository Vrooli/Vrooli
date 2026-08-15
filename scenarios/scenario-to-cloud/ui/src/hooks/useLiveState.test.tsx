import "@testing-library/jest-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { vi } from "vitest";

const api = vi.hoisted(() => ({
  getLiveState: vi.fn(),
  getFiles: vi.fn(),
  getFileContent: vi.fn(),
  getDrift: vi.fn(),
  killProcess: vi.fn(),
  restartProcess: vi.fn(),
  controlProcess: vi.fn(),
  executeVPSAction: vi.fn(),
  getHistory: vi.fn(),
  getLogs: vi.fn(),
  checkDNS: vi.fn(),
  controlCaddy: vi.fn(),
  getTLSInfo: vi.fn(),
  renewTLS: vi.fn(),
}));

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../lib/api")>("../lib/api");
  return { ...actual, ...api };
});

import {
  formatBytes,
  formatDuration,
  formatMB,
  formatUptime,
  getEventTypeInfo,
  getLogLevelInfo,
  getTimeSince,
  useCaddyControl,
  useDNSCheck,
  useDrift,
  useFileContent,
  useFiles,
  useHistory,
  useKillProcess,
  useLiveState,
  useLogs,
  useProcessControl,
  useRestartProcess,
  useTLSInfo,
  useTLSRenew,
  useVPSAction,
} from "./useLiveState";

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("live-state hooks and formatting", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getLiveState.mockResolvedValue({ result: { processes: [], ports: [] } });
    api.getFiles.mockResolvedValue({ path: "/var/app", entries: [] });
    api.getFileContent.mockResolvedValue({ path: "/var/app/a.txt", content: "hello", size_bytes: 5, truncated: false });
    api.getDrift.mockResolvedValue({ result: { drifted: false } });
    api.getHistory.mockResolvedValue({ history: [] });
    api.getLogs.mockResolvedValue({ logs: [], total: 0, filtered: 0, sources: [] });
    api.checkDNS.mockResolvedValue({ ok: true });
    api.getTLSInfo.mockResolvedValue({ ok: true });
    api.killProcess.mockResolvedValue({ ok: true });
    api.restartProcess.mockResolvedValue({ ok: true });
    api.controlProcess.mockResolvedValue({ ok: true });
    api.executeVPSAction.mockResolvedValue({ ok: true });
    api.controlCaddy.mockResolvedValue({ ok: true });
    api.renewTLS.mockResolvedValue({ ok: true });
  });

  it("formats uptime, sizes, durations, timestamps, event types, and log levels", () => {
    expect(formatUptime(5)).toBe("5s");
    expect(formatUptime(65)).toBe("1m");
    expect(formatUptime(3665)).toBe("1h 1m");
    expect(formatUptime(90000)).toBe("1d 1h");
    expect(formatBytes(5)).toBe("5 B");
    expect(formatBytes(2048)).toBe("2.0 KB");
    expect(formatBytes(2 * 1024 * 1024)).toBe("2.0 MB");
    expect(formatBytes(2 * 1024 * 1024 * 1024)).toBe("2.0 GB");
    expect(formatMB(5)).toBe("5 MB");
    expect(formatMB(2048)).toBe("2.0 GB");
    expect(formatDuration(5)).toBe("5ms");
    expect(formatDuration(2000)).toBe("2s");
    expect(formatDuration(61000)).toBe("1m 1s");
    expect(formatDuration(3661000)).toBe("1h 1m");
    expect(getTimeSince(new Date().toISOString())).toBe("just now");
    expect(getEventTypeInfo("deployment_created").label).toBe("Created");
    expect(getEventTypeInfo("unknown")).toEqual({ label: "unknown", color: "slate", icon: "info" });
    for (const level of ["ERROR", "WARN", "INFO", "DEBUG", "other"]) {
      expect(getLogLevelInfo(level)).toHaveProperty("color");
    }
  });

  it("loads live state, files, content, drift, history, logs, DNS, and TLS", async () => {
    const { result } = renderHook(() => ({
      live: useLiveState("deployment-1"),
      files: useFiles("deployment-1", "/var/app"),
      content: useFileContent("deployment-1", "/var/app/a.txt"),
      drift: useDrift("deployment-1"),
      history: useHistory("deployment-1"),
      logs: useLogs("deployment-1", { source: "api", level: "INFO", tail: 10, search: "health" }),
      dns: useDNSCheck("deployment-1"),
      tls: useTLSInfo("deployment-1"),
    }), { wrapper });
    await waitFor(() => expect(result.current.live.data).toEqual({ processes: [], ports: [] }));
    expect(result.current.files.data).toEqual({ path: "/var/app", entries: [] });
    expect(result.current.content.data).toEqual({ path: "/var/app/a.txt", content: "hello", sizeBytes: 5, truncated: false });
    expect(result.current.drift.data).toEqual({ drifted: false });
    expect(result.current.history.data).toEqual([]);
    expect(result.current.logs.data).toEqual({ logs: [], total: 0, filtered: 0, sources: [] });
    expect(result.current.dns.data).toEqual({ ok: true });
    expect(result.current.tls.data).toEqual({ ok: true });
    expect(api.getFiles).toHaveBeenCalledWith("deployment-1", "/var/app");
    expect(api.getLogs).toHaveBeenCalledWith("deployment-1", { source: "api", level: "INFO", tail: 10, search: "health" });
  });

  it("keeps null query inputs disabled", () => {
    const { result } = renderHook(() => ({
      live: useLiveState(null),
      files: useFiles(null),
      content: useFileContent(null, null),
      drift: useDrift(null),
      history: useHistory(null),
      logs: useLogs(null),
      dns: useDNSCheck(null),
      tls: useTLSInfo(null),
    }), { wrapper });
    expect(result.current.live.data).toBeUndefined();
    expect(result.current.content.data).toBeUndefined();
    expect(api.getLiveState).not.toHaveBeenCalled();
    expect(api.getTLSInfo).not.toHaveBeenCalled();
  });

  it("executes process, VPS, Caddy, and TLS mutations", async () => {
    const { result } = renderHook(() => ({
      kill: useKillProcess("deployment-1"),
      restart: useRestartProcess("deployment-1"),
      control: useProcessControl("deployment-1"),
      action: useVPSAction("deployment-1"),
      caddy: useCaddyControl("deployment-1"),
      renew: useTLSRenew("deployment-1"),
    }), { wrapper });
    await act(async () => {
      await result.current.kill.mutateAsync({ pid: 10 });
      await result.current.restart.mutateAsync({ type: "scenario", id: "api" });
      await result.current.control.mutateAsync({ type: "scenario", id: "api", action: "stop" });
      await result.current.action.mutateAsync({ action: "reboot", confirmation: "REBOOT" });
      await result.current.caddy.mutateAsync("reload");
      await result.current.renew.mutateAsync();
    });
    expect(api.killProcess).toHaveBeenCalledWith("deployment-1", { pid: 10 });
    expect(api.restartProcess).toHaveBeenCalledWith("deployment-1", { type: "scenario", id: "api" });
    expect(api.controlProcess).toHaveBeenCalledWith("deployment-1", { type: "scenario", id: "api", action: "stop" });
    expect(api.executeVPSAction).toHaveBeenCalledWith("deployment-1", { action: "reboot", confirmation: "REBOOT" });
    expect(api.controlCaddy).toHaveBeenCalledWith("deployment-1", "reload");
    expect(api.renewTLS).toHaveBeenCalledWith("deployment-1");
  });
});
