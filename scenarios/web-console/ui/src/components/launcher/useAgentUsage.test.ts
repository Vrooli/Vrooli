import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({ getAgentUsage: vi.fn() }));
vi.mock("../../api/agentUsage", () => api);
import { useAgentUsage } from "./useAgentUsage";

const response = { targetId: "local", agentId: "codex", match: "likely", reason: "Account estimated", source: "provider", observedAt: new Date().toISOString(), windows: [] };

describe("useAgentUsage", () => {
  beforeEach(() => vi.resetAllMocks());

  it("does not request when disabled and refreshes the selected target", async () => {
    api.getAgentUsage.mockResolvedValue(response);
    const { result, rerender } = renderHook(({ enabled }) => useAgentUsage(enabled, "local", "codex"), { initialProps: { enabled: false } });
    expect(api.getAgentUsage).not.toHaveBeenCalled();
    rerender({ enabled: true });
    await waitFor(() => { expect(result.current.usages.codex?.match).toBe("likely"); });
    expect(api.getAgentUsage.mock.calls[0]?.[0]).toEqual({ targetId: "local", agentId: "codex" });
  });

  it("does not display a reading from a different target or unknown match state", async () => {
    api.getAgentUsage.mockResolvedValueOnce({ ...response, targetId: "remote" }).mockResolvedValueOnce({ ...response, targetId: "next", match: "new-state" });
    const { result, rerender } = renderHook(({ target }) => useAgentUsage(true, target, "codex"), { initialProps: { target: "local" } });
    await waitFor(() => { expect(result.current.usages.codex).toBeNull(); });
    rerender({ target: "next" });
    await waitFor(() => { expect(result.current.usages.codex).toBeNull(); });
  });

  it("discards an old target response after target change", async () => {
    let resolveOld!: (value: typeof response) => void;
    api.getAgentUsage.mockImplementation(({ targetId }: { targetId: string }) => targetId === "local" ? new Promise((resolve) => { resolveOld = resolve; }) : Promise.resolve({ ...response, targetId: "remote" }));
    const { result, rerender } = renderHook(({ target }) => useAgentUsage(true, target, "codex"), { initialProps: { target: "local" } });
    rerender({ target: "remote" });
    await waitFor(() => { expect(result.current.usages.codex?.targetId).toBe("remote"); });
    resolveOld(response);
    await Promise.resolve();
    expect(result.current.usages.codex?.targetId).toBe("remote");
  });
});
