import "@testing-library/jest-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { vi } from "vitest";

const api = vi.hoisted(() => ({
  getAgentManagerStatus: vi.fn(),
  listInvestigations: vi.fn(),
  getInvestigation: vi.fn(),
  triggerInvestigation: vi.fn(),
  stopInvestigation: vi.fn(),
  applyFixes: vi.fn(),
}));

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../lib/api")>("../lib/api");
  return { ...actual, ...api };
});

import { useDeploymentInvestigation } from "./useInvestigation";

const detail = {
  id: "investigation-1",
  deployment_id: "deployment-1",
  status: "running" as const,
  progress: 40,
  details: { source: "agent", operation_mode: "investigate", trigger_reason: "health" },
  created_at: "2026-08-14T00:00:00Z",
  updated_at: "2026-08-14T00:00:00Z",
};
const summary = { ...detail, has_findings: false };
const fixed = { ...detail, id: "fix-1", status: "pending" as const };

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("useDeploymentInvestigation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getAgentManagerStatus.mockResolvedValue({ enabled: true, available: true });
    api.listInvestigations.mockResolvedValue({ investigations: [summary] });
    api.getInvestigation.mockResolvedValue({ investigation: detail });
    api.triggerInvestigation.mockResolvedValue({ investigation: detail });
    api.stopInvestigation.mockResolvedValue({ ok: true });
    api.applyFixes.mockResolvedValue({ investigation: fixed });
  });

  it("tracks the latest investigation and exposes agent status", async () => {
    const { result } = renderHook(() => useDeploymentInvestigation("deployment-1"), { wrapper });
    await waitFor(() => expect(result.current.activeInvestigation?.id).toBe("investigation-1"));
    expect(result.current.isAgentAvailable).toBe(true);
    expect(result.current.isAgentEnabled).toBe(true);
    expect(result.current.isRunning).toBe(true);
    expect(result.current.investigations).toHaveLength(1);
    expect(api.listInvestigations).toHaveBeenCalledWith("deployment-1", 50);
    expect(api.getInvestigation).toHaveBeenCalledWith("deployment-1", "investigation-1");
  });

  it("triggers, stops, applies fixes, views reports, and refreshes", async () => {
    const { result } = renderHook(() => useDeploymentInvestigation("deployment-1"), { wrapper });
    await waitFor(() => expect(result.current.activeInvestigation?.id).toBe("investigation-1"));

    await act(async () => {
      await result.current.trigger({ auto_fix: true, note: "inspect logs" });
    });
    expect(api.triggerInvestigation).toHaveBeenCalledWith("deployment-1", { auto_fix: true, note: "inspect logs" });

    await act(async () => { await result.current.stop(); });
    expect(api.stopInvestigation).toHaveBeenCalledWith("deployment-1", "investigation-1");

    await act(async () => {
      await result.current.applyFixes("investigation-1", {
        immediate: true,
        permanent: false,
        prevention: true,
        note: "apply now",
      });
    });
    expect(api.applyFixes).toHaveBeenCalledWith("deployment-1", "investigation-1", {
      immediate: true,
      permanent: false,
      prevention: true,
      note: "apply now",
    });

    act(() => result.current.viewReport("investigation-1"));
    expect(result.current.showReport).toBe(true);
    expect(result.current.activeInvestigationId).toBe("investigation-1");
    act(() => result.current.closeReport());
    expect(result.current.showReport).toBe(false);
    act(() => result.current.refresh());
  });

  it("keeps null deployments inert and returns unavailable defaults", async () => {
    const { result } = renderHook(() => useDeploymentInvestigation(null), { wrapper });
    expect(result.current.isAgentAvailable).toBe(false);
    expect(result.current.isAgentEnabled).toBe(false);
    expect(result.current.investigations).toEqual([]);
    await act(async () => {
      await expect(result.current.trigger()).resolves.toBeUndefined();
      await expect(result.current.stop()).resolves.toBeUndefined();
      await expect(result.current.applyFixes("missing", { immediate: true, permanent: false, prevention: false })).resolves.toBeUndefined();
    });
    expect(api.listInvestigations).not.toHaveBeenCalled();
    expect(api.triggerInvestigation).not.toHaveBeenCalled();
  });
});
