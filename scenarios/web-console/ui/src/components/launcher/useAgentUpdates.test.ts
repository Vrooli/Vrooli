import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({ agentUpdate: vi.fn() }));
vi.mock("../../api/capabilities", () => ({ capabilitiesClient: api }));
import { useAgentUpdates } from "./useAgentUpdates";

const response = { targetId: "local", agentId: "codex", installed: "1.0", latest: "2.0", pinned: "1.0", status: "behind", checkedAt: "2026-10-29T12:00:00Z", sourceKind: "release", sourceId: "codex", probeVerified: true, note: "" };

describe("useAgentUpdates", () => {
  beforeEach(() => { vi.resetAllMocks(); });

  it("does not request update checks until enabled", async () => {
    api.agentUpdate.mockResolvedValue(response);
    const { result, rerender } = renderHook(({ open }) => useAgentUpdates(open, "local", "codex"), { initialProps: { open: false } });
    expect(api.agentUpdate).not.toHaveBeenCalled();
    rerender({ open: true });
    await waitFor(() => { expect(result.current.updates.codex?.status).toBe("behind"); });
    expect(api.agentUpdate).toHaveBeenCalledOnce();
  });

  it("shows a completed check without waiting for another agent", async () => {
    api.agentUpdate.mockImplementation(({ agentId }: { agentId: string }) => agentId === "codex" ? Promise.resolve(response) : new Promise(() => {}));
    const { result } = renderHook(() => useAgentUpdates(true, "local", "codex,claude"));
    await waitFor(() => { expect(result.current.updates.codex?.status).toBe("behind"); });
    expect(result.current.loading).toBe(true);
  });

  it("does not show a response for a different target", async () => {
    api.agentUpdate.mockResolvedValue({ ...response, targetId: "other" });
    const { result } = renderHook(() => useAgentUpdates(true, "local", "codex"));
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    expect(result.current.updates.codex).toBeNull();
  });

  it("treats a failed check as unknown", async () => {
    api.agentUpdate.mockRejectedValue(new Error("offline"));
    const { result } = renderHook(() => useAgentUpdates(true, "local", "codex"));
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    expect(result.current.updates.codex).toBeNull();
  });
});
