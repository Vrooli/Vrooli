import "@testing-library/jest-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor, act } from "@testing-library/react";
import type { ReactNode } from "react";
import { vi } from "vitest";

const api = vi.hoisted(() => ({
  listDeployments: vi.fn(),
  getDeployment: vi.fn(),
  createDeployment: vi.fn(),
  executeDeployment: vi.fn(),
  inspectDeployment: vi.fn(),
  stopDeployment: vi.fn(),
  startDeployment: vi.fn(),
  deleteDeployment: vi.fn(),
}));

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../lib/api")>("../lib/api");
  return { ...actual, ...api };
});

import {
  getStatusInfo,
  useCreateDeployment,
  useDeleteDeployment,
  useDeployment,
  useDeployments,
  useExecuteDeployment,
  useInspectDeployment,
  useStartDeployment,
  useStopDeployment,
} from "./useDeployments";

const deployment = {
  id: "deployment-1",
  name: "Demo",
  scenario_id: "demo",
  status: "deployed" as const,
  created_at: "2026-08-14T00:00:00Z",
  updated_at: "2026-08-14T00:00:00Z",
};

function createWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

describe("deployment hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.listDeployments.mockResolvedValue({ deployments: [deployment] });
    api.getDeployment.mockResolvedValue({ deployment });
    api.createDeployment.mockResolvedValue({ deployment });
    api.executeDeployment.mockResolvedValue({ run_id: "run-1" });
    api.inspectDeployment.mockResolvedValue({ result: { ok: true } });
    api.stopDeployment.mockResolvedValue({ ok: true });
    api.startDeployment.mockResolvedValue({ ok: true });
    api.deleteDeployment.mockResolvedValue({ ok: true });
  });

  it("maps every deployment status to a stable presentation contract", () => {
    expect(getStatusInfo("pending")).toEqual({ label: "Pending", color: "slate", icon: "clock" });
    expect(getStatusInfo("setup_running")).toEqual({ label: "Setting up...", color: "blue", icon: "loader" });
    expect(getStatusInfo("setup_complete")).toEqual({ label: "Setup complete", color: "blue", icon: "check" });
    expect(getStatusInfo("deploying")).toEqual({ label: "Deploying...", color: "blue", icon: "loader" });
    expect(getStatusInfo("deployed")).toEqual({ label: "Deployed", color: "emerald", icon: "check-circle" });
    expect(getStatusInfo("failed")).toEqual({ label: "Failed", color: "red", icon: "x-circle" });
    expect(getStatusInfo("stopped")).toEqual({ label: "Stopped", color: "amber", icon: "pause" });
    expect(getStatusInfo("future" as never)).toEqual({ label: "future", color: "slate", icon: "help" });
  });

  it("loads deployment collections and individual records, including the disabled null path", async () => {
    const collection = renderHook(() => useDeployments(), { wrapper: createWrapper() });
    await waitFor(() => expect(collection.result.current.data).toEqual([deployment]));
    expect(api.listDeployments).toHaveBeenCalledOnce();

    const disabled = renderHook(() => useDeployment(null), { wrapper: createWrapper() });
    expect(disabled.result.current.data).toBeUndefined();
    expect(api.getDeployment).not.toHaveBeenCalled();

    const detail = renderHook(() => useDeployment("deployment-1"), { wrapper: createWrapper() });
    await waitFor(() => expect(detail.result.current.data).toEqual(deployment));
    expect(api.getDeployment).toHaveBeenCalledWith("deployment-1");
  });

  it("executes create, deploy, inspect, start, stop, and delete mutations", async () => {
    const create = renderHook(() => useCreateDeployment(), { wrapper: createWrapper() });
    await act(async () => {
      await create.result.current.mutateAsync({ manifest: { scenario: "demo" }, name: "Demo" });
    });
    expect(api.createDeployment).toHaveBeenCalledWith({ scenario: "demo" }, { name: "Demo" });

    const execute = renderHook(() => useExecuteDeployment(), { wrapper: createWrapper() });
    await act(async () => {
      await execute.result.current.mutateAsync({ id: "deployment-1", options: { runPreflight: true } });
    });
    expect(api.executeDeployment).toHaveBeenCalledWith("deployment-1", { runPreflight: true });

    const inspect = renderHook(() => useInspectDeployment(), { wrapper: createWrapper() });
    await act(async () => { await inspect.result.current.mutateAsync("deployment-1"); });
    expect(api.inspectDeployment).toHaveBeenCalledWith("deployment-1");

    const stop = renderHook(() => useStopDeployment(), { wrapper: createWrapper() });
    await act(async () => { await stop.result.current.mutateAsync("deployment-1"); });
    expect(api.stopDeployment).toHaveBeenCalledWith("deployment-1");

    const start = renderHook(() => useStartDeployment(), { wrapper: createWrapper() });
    await act(async () => { await start.result.current.mutateAsync("deployment-1"); });
    expect(api.startDeployment).toHaveBeenCalledWith("deployment-1");

    const remove = renderHook(() => useDeleteDeployment(), { wrapper: createWrapper() });
    await act(async () => {
      await remove.result.current.mutateAsync({ id: "deployment-1", stopOnVPS: true, cleanupBundles: true });
    });
    expect(api.deleteDeployment).toHaveBeenCalledWith("deployment-1", { stopOnVPS: true, cleanupBundles: true });
  });
});
