import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { probeRunner, usePermissionPolicy, useRolePolicyCatalog } from "./useApi.js";
import { RunnerType } from "../types.js";
import { rpcBody, rpcCalls, rpcMethod } from "../test-utils/rpc.js";

afterEach(() => vi.unstubAllGlobals());

function json(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });
}

test("policy controls and the runner probe use canonical read and whole-document action routes", async () => {
  const fetch = vi.fn(async () => json({}));
  vi.stubGlobal("fetch", fetch);
  const roleCatalog = renderHook(() => useRolePolicyCatalog());
  const policy = renderHook(() => usePermissionPolicy());
  await waitFor(() => assert.ok(fetch.mock.calls.length >= 3));
  await waitFor(() => assert.ok(roleCatalog.result.current.data));
  assert.ok(roleCatalog.result.current.data);
  await act(async () => {
    await policy.result.current.validate();
    await policy.result.current.reload();
    await policy.result.current.plan();
    await policy.result.current.doctor();
    await policy.result.current.reconcile();
    await probeRunner(RunnerType.CODEX);
  });
  const calls = fetch.mock.calls as unknown as unknown[][];
  for (const method of ["ValidatePermissionPolicyCatalog", "ReloadPermissionPolicyCatalog", "PlanPermissionPolicy", "DoctorPermissionPolicy", "ReconcilePermissionPolicy"]) {
    assert.equal(rpcCalls(calls, method).length, 1, `${method} is called once over Connect`);
  }
  assert.equal(rpcBody(rpcCalls(calls, "ReconcilePermissionPolicy")[0]!).explicitlyAuthorized, true);
  assert.equal(rpcBody(rpcCalls(calls, "ProbeRunner")[0]!).runnerType, "RUNNER_TYPE_CODEX");
  assert.ok(calls.every((call) => rpcMethod(call) !== ""), "no REST route is used for policy controls or the probe");
});
