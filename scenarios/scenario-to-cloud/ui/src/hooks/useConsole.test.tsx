import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { makeStanding } from "../test-utils/consoleFixtures";
import { operationPointerKey, saveOperationPointer } from "../lib/consoleApi";
import { useApplyPlan, useCancelOperation, useCompiledPlan, useDurableOperation, useOperationStanding, useRecoveryPoints, useRestoreRecoveryPoint, useRollbackAdmission } from "./useConsole";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function wrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return {
    client,
    Wrapper: ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
  };
}

describe("useConsole hooks", () => {
  const fetchMock = vi.fn();
  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
    sessionStorage.clear();
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  // [REQ:STC-P0-039] A reload rehydrates the operation from the durable pointer.
  it("rehydrates the operation from the stored pointer and marks it resumed", async () => {
    saveOperationPointer({ deployment_id: "dep-1", operation_id: "op-1234567890", plan_digest: "sha256:x" });
    fetchMock.mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/operations/op-1234567890")) return jsonResponse(makeStanding());
      return jsonResponse({ error: { code: "deployment_not_found", message: "nope" } }, 404);
    });
    const { Wrapper } = wrapper();
    const { result } = renderHook(() => useDurableOperation("dep-1"), { wrapper: Wrapper });
    expect(result.current.rehydrating).toBe(true);
    await waitFor(() => expect(result.current.rehydrating).toBe(false));
    expect(result.current.pointer?.operation_id).toBe("op-1234567890");
    expect(result.current.resumed).toBe(true);
    act(() => result.current.detach());
    expect(result.current.pointer).toBeNull();
    expect(sessionStorage.getItem(operationPointerKey("dep-1"))).toBeNull();
  });

  it("discards a pointer whose operation no longer exists and falls back to the listed active operation", async () => {
    saveOperationPointer({ deployment_id: "dep-1", operation_id: "op-gone", plan_digest: "sha256:x" });
    fetchMock.mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/operations/op-gone")) return jsonResponse({ error: { code: "operation_not_found", message: "gone" } }, 404);
      if (url.includes("/deployments/dep-1/operations")) {
        return jsonResponse({ schema_version: "1", deployment_id: "dep-1", operations: [makeStanding({ operation_id: "op-done", state: "succeeded", terminal: true }), makeStanding({ operation_id: "op-live" })], timestamp: "t" });
      }
      return jsonResponse({}, 500);
    });
    const { Wrapper } = wrapper();
    const { result } = renderHook(() => useDurableOperation("dep-1"), { wrapper: Wrapper });
    await waitFor(() => expect(result.current.rehydrating).toBe(false));
    expect(result.current.pointer?.operation_id).toBe("op-live");
    expect(result.current.resumed).toBe(true);
    expect(JSON.parse(sessionStorage.getItem(operationPointerKey("dep-1")) ?? "{}")).toMatchObject({ operation_id: "op-live" });
  });

  it("stays detached when nothing is stored and nothing is active, and attaches on demand", async () => {
    fetchMock.mockImplementation(async (input: RequestInfo | URL) => {
      if (String(input).includes("/operations")) return jsonResponse({ schema_version: "1", deployment_id: "dep-1", operations: [], timestamp: "t" });
      return jsonResponse({}, 500);
    });
    const { Wrapper } = wrapper();
    const { result } = renderHook(() => useDurableOperation("dep-1"), { wrapper: Wrapper });
    await waitFor(() => expect(result.current.rehydrating).toBe(false));
    expect(result.current.pointer).toBeNull();
    act(() => result.current.attach({ deployment_id: "dep-1", operation_id: "op-new", plan_digest: "sha256:y" }));
    expect(result.current.pointer?.operation_id).toBe("op-new");
    expect(result.current.resumed).toBe(false);
    expect(sessionStorage.getItem(operationPointerKey("dep-1"))).toContain("op-new");
  });

  it("does nothing without a deployment id and survives a failed listing", async () => {
    const { Wrapper } = wrapper();
    const idle = renderHook(() => useDurableOperation(null), { wrapper: Wrapper });
    expect(idle.result.current.rehydrating).toBe(false);
    fetchMock.mockRejectedValue(new Error("network"));
    const failing = renderHook(() => useDurableOperation("dep-9"), { wrapper: Wrapper });
    await waitFor(() => expect(failing.result.current.rehydrating).toBe(false));
    expect(failing.result.current.pointer).toBeNull();
  });

  it("reads the standing once, then uses the server-side wait while the operation is active", async () => {
    let calls = 0;
    fetchMock.mockImplementation(async (input: RequestInfo | URL) => {
      calls += 1;
      const url = String(input);
      if (url.includes("/wait?timeout=20")) return jsonResponse(makeStanding({ state: "succeeded", terminal: true }));
      return jsonResponse(makeStanding());
    });
    const { Wrapper } = wrapper();
    const { result } = renderHook(() => useOperationStanding("op-1234567890"), { wrapper: Wrapper });
    await waitFor(() => expect(result.current.data?.state).toBe("succeeded"));
    const urls = fetchMock.mock.calls.map(([u]) => String(u));
    expect(urls[0]).toMatch(/\/operations\/op-1234567890$/);
    expect(urls[1]).toContain("/operations/op-1234567890/wait?timeout=20");
    expect(calls).toBe(2);
  });

  it("is disabled without an operation id", () => {
    const { Wrapper } = wrapper();
    const { result } = renderHook(() => useOperationStanding(null), { wrapper: Wrapper });
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("compiles the plan as a read and exposes typed refusals", async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: { code: "closure_unavailable", message: "no closure" } }, 424));
    const { Wrapper } = wrapper();
    const { result } = renderHook(() => useCompiledPlan("dep-1"), { wrapper: Wrapper });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect((result.current.error as unknown as { code: string }).code).toBe("closure_unavailable");
    const disabled = renderHook(() => useCompiledPlan("dep-1", { enabled: false }), { wrapper: Wrapper });
    expect(disabled.result.current.fetchStatus).toBe("idle");
  });

  it("applies, cancels, checks rollback and restores through their endpoints and invalidates caches", async () => {
    fetchMock.mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith("/plan/apply")) return jsonResponse({ schema_version: "1", operation_id: "op-2", plan_digest: "sha256:x", state: "admitted" }, 202);
      if (url.endsWith("/cancel")) return jsonResponse(makeStanding({ state: "cancel_requested", cancel_requested: true }), 202);
      if (url.endsWith("/rollback-admission")) return jsonResponse({ schema_version: "1", verdict: { compatible: true, schema_strategy: "same", current_schema: "12", target_schema: "12" } });
      if (url.endsWith("/restore")) return jsonResponse({ schema_version: "1", receipt: { outcome: "succeeded" } });
      if (url.endsWith("/recovery-points")) return jsonResponse({ schema_version: "1", recovery_points: [] });
      return jsonResponse({ init: init?.method });
    });
    const { Wrapper, client } = wrapper();
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const apply = renderHook(() => useApplyPlan(), { wrapper: Wrapper });
    await act(async () => {
      await apply.result.current.mutateAsync({ deploymentId: "dep-1", planDigest: "sha256:x", requestKey: "k" });
    });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["deployment", "dep-1"] });
    const cancel = renderHook(() => useCancelOperation(), { wrapper: Wrapper });
    await act(async () => {
      await cancel.result.current.mutateAsync("op-1234567890");
    });
    expect((client.getQueryData(["operation", "op-1234567890"]) as { state: string }).state).toBe("cancel_requested");
    const rollback = renderHook(() => useRollbackAdmission(), { wrapper: Wrapper });
    await act(async () => {
      const res = await rollback.result.current.mutateAsync({ deploymentId: "dep-1", currentSchema: "12", targetSchema: "12" });
      expect(res.verdict.compatible).toBe(true);
    });
    const points = renderHook(() => useRecoveryPoints("dep-1"), { wrapper: Wrapper });
    await waitFor(() => expect(points.result.current.data?.recovery_points).toEqual([]));
    const restore = renderHook(() => useRestoreRecoveryPoint(), { wrapper: Wrapper });
    await act(async () => {
      await restore.result.current.mutateAsync({ deploymentId: "dep-1", recoveryPointId: "rp-1", into: { a: "b" } });
    });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["recoveryPoints", "dep-1"] });
  });
});
