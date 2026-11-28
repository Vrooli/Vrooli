import { describe, expect, it, vi } from "vitest";

const calls = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn() }));
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:9999/api/v1",
  createScenarioConnectTransport: () => ({}),
}));
vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    getOperatorState: calls.get,
    patchOperatorState: calls.patch,
  }),
}));

import { fetchOperatorState, saveOperatorState } from "./operatorstate";

describe("typed operator-state client", () => {
  it("reads the typed state envelope", async () => {
    calls.get.mockResolvedValueOnce({ state: { version: "1.0.0", trustPosture: "shared" } });
    await expect(fetchOperatorState()).resolves.toMatchObject({ version: "1.0.0", trustPosture: "shared" });
    expect(calls.get).toHaveBeenCalledWith({ target: "local" });
  });

  it("sends only the top-level fields present in a patch mask", async () => {
    calls.patch.mockResolvedValueOnce({ state: { version: "1.0.0", scenarios: { alpha: { enabled: true } } } });
    const patch = { scenarios: { alpha: { enabled: true } } };
    await saveOperatorState(patch);
    expect(calls.patch).toHaveBeenCalledWith(expect.objectContaining({ target: "local", state: patch }));
    expect(calls.patch.mock.calls[0]?.[0]?.updateMask?.paths).toEqual(["scenarios"]);
    calls.patch.mockClear();
  });

  it("refuses posture fields before calling the generic operator-state writer", async () => {
    await expect(saveOperatorState({ updateControl: "guard" })).rejects.toThrow(/typed install-posture permission/);
    await expect(saveOperatorState({ sessionMode: "shared" })).rejects.toThrow(/enrollment-owned/);
    await expect(saveOperatorState({ trustPosture: "shared" })).rejects.toThrow(/control-plane-owned/);
    expect(calls.patch).not.toHaveBeenCalled();
  });
});
