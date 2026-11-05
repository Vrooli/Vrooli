import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "./apiErrors";
import {
  applyDeploymentPlan,
  cancelOperation,
  clearOperationPointer,
  compileDeploymentPlan,
  getAuthzMatrix,
  getOperation,
  listDeploymentOperations,
  listRecoveryPoints,
  loadOperationPointer,
  newRequestKey,
  operationPointerKey,
  requestJson,
  restoreRecoveryPoint,
  rollbackAdmission,
  saveOperationPointer,
  waitOperation,
} from "./consoleApi";

function initOf(calls: unknown[][], index: number): RequestInit {
  const call = calls[index] ?? [];
  return (call[1] ?? {}) as RequestInit;
}

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("consoleApi", () => {
  const fetchMock = vi.fn();
  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
    sessionStorage.clear();
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("turns typed refusals into ApiError with code, next_action and details", async () => {
    fetchMock.mockImplementation(async () =>
      jsonResponse({ error: { code: "forbidden_scope", message: "denied", retryable: false, next_action: { owner: "operator", kind: "sign_in", reference: "x" }, details: { required_scope: "scenario-to-cloud:write" } } }, 403),
    );
    await expect(requestJson("/deployments/d/plan", { method: "POST" })).rejects.toMatchObject({ code: "forbidden_scope", status: 403 });
    try {
      await requestJson("/deployments/d/plan", { method: "POST" });
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).nextAction?.kind).toBe("sign_in");
      expect((err as ApiError).typed.details).toMatchObject({ required_scope: "scenario-to-cloud:write" });
    }
  });

  it("never caches and sends JSON content type", async () => {
    fetchMock.mockImplementation(async () => jsonResponse({ schema_version: "1", scopes: [], routes: [], scenario: "scenario-to-cloud" }));
    await getAuthzMatrix();
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/authz/matrix");
    expect(init.cache).toBe("no-store");
    expect((init.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
  });

  it("addresses every console endpoint by its documented path and method", async () => {
    fetchMock.mockImplementation(async () => jsonResponse({}));
    await compileDeploymentPlan("dep 1", "install");
    await applyDeploymentPlan("dep 1", { plan_digest: "sha256:x", request_key: "k" });
    await getOperation("op/1");
    await waitOperation("op-1", 20);
    await cancelOperation("op-1");
    await listDeploymentOperations("dep-1");
    await listRecoveryPoints("dep-1");
    await rollbackAdmission("dep-1", { current_schema: "12", target_schema: "11" });
    await restoreRecoveryPoint("dep-1", "rp-1", { into: { "postgres:main": "postgres://x" } });
    const calls = fetchMock.mock.calls.map(([url, init]) => `${(init as RequestInit).method ?? "GET"} ${String(url).replace(/^.*\/api\/v1/, "")}`);
    expect(calls).toEqual([
      "POST /deployments/dep%201/plan",
      "POST /deployments/dep%201/plan/apply",
      "GET /operations/op%2F1",
      "GET /operations/op-1/wait?timeout=20",
      "POST /operations/op-1/cancel",
      "GET /deployments/dep-1/operations",
      "GET /deployments/dep-1/recovery-points",
      "POST /deployments/dep-1/recovery-points/rollback-admission",
      "POST /deployments/dep-1/recovery-points/rp-1/restore",
    ]);
    expect(JSON.parse(initOf(fetchMock.mock.calls, 0).body as string)).toEqual({ scope: "install" });
    expect(JSON.parse(initOf(fetchMock.mock.calls, 1).body as string)).toEqual({ plan_digest: "sha256:x", request_key: "k" });
    expect(JSON.parse(initOf(fetchMock.mock.calls, 8).body as string)).toMatchObject({ deployment_id: "dep-1", recovery_point_id: "rp-1", into: { "postgres:main": "postgres://x" } });
  });

  it("compiles the full scope by default with an empty body", async () => {
    fetchMock.mockImplementation(async () => jsonResponse({}));
    await compileDeploymentPlan("dep-1");
    expect(initOf(fetchMock.mock.calls, 0).body).toBe("{}");
  });

  it("mints version 4 uuid request keys with and without randomUUID", () => {
    expect(newRequestKey()).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
    const original = globalThis.crypto;
    vi.stubGlobal("crypto", { getRandomValues: (arr: Uint8Array) => arr.fill(7) });
    expect(newRequestKey()).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
    vi.stubGlobal("crypto", undefined);
    expect(newRequestKey()).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
    vi.stubGlobal("crypto", original);
  });

  // [REQ:STC-P0-039] The durable pointer round-trips and rejects malformed data.
  it("stores, loads and clears the durable operation pointer", () => {
    const pointer = { deployment_id: "dep-1", operation_id: "op-1", plan_digest: "sha256:x" };
    saveOperationPointer(pointer);
    expect(sessionStorage.getItem(operationPointerKey("dep-1"))).toBe(JSON.stringify(pointer));
    expect(loadOperationPointer("dep-1")).toEqual(pointer);
    sessionStorage.setItem(operationPointerKey("dep-2"), JSON.stringify({ deployment_id: "dep-2" }));
    expect(loadOperationPointer("dep-2")).toBeNull();
    sessionStorage.setItem(operationPointerKey("dep-3"), "not json");
    expect(loadOperationPointer("dep-3")).toBeNull();
    expect(loadOperationPointer("dep-4")).toBeNull();
    clearOperationPointer("dep-1");
    expect(loadOperationPointer("dep-1")).toBeNull();
  });

  it("tolerates unavailable storage", () => {
    const setItem = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("quota");
    });
    const removeItem = vi.spyOn(Storage.prototype, "removeItem").mockImplementation(() => {
      throw new Error("quota");
    });
    const getItem = vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("quota");
    });
    expect(() => saveOperationPointer({ deployment_id: "d", operation_id: "o", plan_digest: "p" })).not.toThrow();
    expect(() => clearOperationPointer("d")).not.toThrow();
    expect(loadOperationPointer("d")).toBeNull();
    setItem.mockRestore();
    removeItem.mockRestore();
    getItem.mockRestore();
  });
});
