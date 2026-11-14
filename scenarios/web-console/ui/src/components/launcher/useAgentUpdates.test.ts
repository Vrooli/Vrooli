import { act, renderHook, waitFor } from "@testing-library/react";
import { Code, ConnectError } from "@connectrpc/connect";
import { beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({ agentUpdate: vi.fn(), runAgentUpdateAction: vi.fn() }));
vi.mock("../../api/capabilities", () => ({ capabilitiesClient: api }));
import { useAgentUpdates } from "./useAgentUpdates";

const response = {
  targetId: "local", agentId: "codex", installed: "1.0", latest: "2.0", pinned: "1.0", status: "behind", checkedAt: "2026-10-29T12:00:00Z", sourceKind: "release", sourceId: "codex", probeVerified: true, note: "",
  state: "update_ready", primaryAction: "update", operatorCommand: "", reason: "", checks: [], installKind: "managed", rollbackVersion: "", newerThanReviewed: false,
};

describe("useAgentUpdates", () => {
  beforeEach(() => { vi.resetAllMocks(); });

  it("does not request update checks until enabled", async () => {
    api.agentUpdate.mockResolvedValue(response);
    const { result, rerender } = renderHook(({ open }) => useAgentUpdates(open, "local", "codex"), { initialProps: { open: false } });
    expect(api.agentUpdate).not.toHaveBeenCalled();
    rerender({ open: true });
    await waitFor(() => { expect(result.current.updates.codex?.state).toBe("update_ready"); });
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

  it("runs an action once however often it is pressed, then adopts the fresh observation", async () => {
    api.agentUpdate.mockResolvedValue(response);
    let finish!: (value: unknown) => void;
    api.runAgentUpdateAction.mockImplementation(() => new Promise((resolve) => { finish = resolve; }));
    const { result } = renderHook(() => useAgentUpdates(true, "local", "codex"));
    await waitFor(() => { expect(result.current.updates.codex).toBeTruthy(); });
    act(() => { void result.current.run("codex", "update"); void result.current.run("codex", "update"); });
    expect(api.runAgentUpdateAction).toHaveBeenCalledOnce();
    expect(api.runAgentUpdateAction.mock.calls[0]?.[0]).toEqual({ targetId: "local", agentId: "codex", action: "update" });
    expect(result.current.runs.codex?.phase).toBe("running");
    await act(async () => { finish({ success: true, message: "Updated.", update: { ...response, installed: "2.0", state: "up_to_date", primaryAction: "" } }); await Promise.resolve(); });
    await waitFor(() => { expect(result.current.runs.codex?.phase).toBe("done"); });
    expect(result.current.runs.codex).toMatchObject({ previous: "1.0", current: "2.0" });
    expect(result.current.updates.codex?.state).toBe("up_to_date");
  });

  it("reports a refused action with the server's words and allows another try", async () => {
    api.agentUpdate.mockResolvedValue(response);
    api.runAgentUpdateAction.mockResolvedValueOnce({ success: false, message: "npm is unreachable", update: { ...response, reason: "fresh" } }).mockRejectedValueOnce(new ConnectError("not here", Code.Unimplemented));
    const { result } = renderHook(() => useAgentUpdates(true, "local", "codex"));
    await waitFor(() => { expect(result.current.updates.codex).toBeTruthy(); });
    await act(async () => { await result.current.run("codex", "update"); });
    expect(result.current.runs.codex).toMatchObject({ phase: "failed", message: "npm is unreachable", unsupported: false });
    await act(async () => { await result.current.run("codex", "update"); });
    expect(result.current.runs.codex).toMatchObject({ phase: "failed", unsupported: true });
  });

  describe("across machines", () => {
    const onTarget = (targetId: string) => ({ ...response, targetId, state: "update_ready" });
    const setup = async () => {
      api.agentUpdate.mockImplementation(({ targetId }: { targetId: string }) => Promise.resolve(onTarget(targetId)));
      const hook = renderHook(({ target }) => useAgentUpdates(true, target, "codex"), { initialProps: { target: "machine-a" } });
      await waitFor(() => { expect(hook.result.current.updates.codex?.state).toBe("update_ready"); });
      return hook;
    };

    it("does not let an action on one machine write into another's panel", async () => {
      const { result, rerender } = await setup();
      let finish!: (value: unknown) => void;
      api.runAgentUpdateAction.mockImplementationOnce(() => new Promise((resolve) => { finish = resolve; }));
      act(() => { void result.current.run("codex", "update"); });
      rerender({ target: "machine-b" });
      await waitFor(() => { expect(result.current.updates.codex?.installed).toBe("1.0"); });
      expect(result.current.runs.codex).toBeUndefined();
      await act(async () => { finish({ success: true, message: "Updated A.", update: { ...onTarget("machine-a"), installed: "9.9", state: "up_to_date" } }); await Promise.resolve(); });
      expect(result.current.updates.codex?.installed).toBe("1.0");
      expect(result.current.runs.codex).toBeUndefined();
      // Going back shows what happened on that machine.
      rerender({ target: "machine-a" });
      await waitFor(() => { expect(result.current.runs.codex?.phase).toBe("done"); });
    });

    it("does not block the same agent on another machine while one runs", async () => {
      const { result, rerender } = await setup();
      api.runAgentUpdateAction.mockImplementation(() => new Promise(() => undefined));
      act(() => { void result.current.run("codex", "update"); });
      rerender({ target: "machine-b" });
      act(() => { void result.current.run("codex", "update"); });
      expect(api.runAgentUpdateAction.mock.calls.map((call) => (call[0] as { targetId: string }).targetId)).toEqual(["machine-a", "machine-b"]);
      expect(result.current.runs.codex?.phase).toBe("running");
    });
  });

  it("looks again through the server on retry and leaves no run behind", async () => {
    api.agentUpdate.mockResolvedValueOnce({ ...response, state: "check_failed" });
    api.runAgentUpdateAction.mockResolvedValue({ success: true, message: "", update: response });
    const { result } = renderHook(() => useAgentUpdates(true, "local", "codex"));
    await waitFor(() => { expect(result.current.updates.codex?.state).toBe("check_failed"); });
    act(() => { result.current.retryCheck("codex"); });
    await waitFor(() => { expect(result.current.updates.codex?.state).toBe("update_ready"); });
    expect(api.runAgentUpdateAction.mock.calls[0]?.[0]).toMatchObject({ action: "retry" });
    await waitFor(() => { expect(result.current.runs.codex).toBeUndefined(); });
  });

  it("keeps a generous deadline, and repeats an action after a dropped connection to rejoin it", async () => {
    api.agentUpdate.mockResolvedValue(response);
    api.runAgentUpdateAction.mockRejectedValueOnce(new ConnectError("dropped", Code.Unavailable)).mockResolvedValueOnce({ success: true, message: "Already current.", update: response });
    const { result } = renderHook(() => useAgentUpdates(true, "local", "codex"));
    await waitFor(() => { expect(result.current.updates.codex).toBeTruthy(); });
    await act(async () => { await result.current.run("codex", "update"); });
    expect(api.runAgentUpdateAction).toHaveBeenCalledTimes(2);
    expect((api.runAgentUpdateAction.mock.calls[0]?.[1] as { timeoutMs: number }).timeoutMs).toBeGreaterThanOrEqual(9 * 60_000);
    expect(result.current.runs.codex?.phase).toBe("done");
  });

  it("reads a dropped rollback back and never repeats it", async () => {
    api.agentUpdate.mockResolvedValue(response);
    api.runAgentUpdateAction.mockRejectedValue(new ConnectError("dropped", Code.Unavailable));
    const { result } = renderHook(() => useAgentUpdates(true, "local", "codex"));
    await waitFor(() => { expect(result.current.updates.codex).toBeTruthy(); });
    const reads = api.agentUpdate.mock.calls.length;
    await act(async () => { await result.current.run("codex", "rollback"); });
    expect(api.runAgentUpdateAction).toHaveBeenCalledOnce();
    expect(api.agentUpdate.mock.calls.length).toBe(reads + 1);
    expect(result.current.runs.codex).toMatchObject({ phase: "failed", outcomeUnknown: true });
  });

  it("does not repeat an action a running one refused", async () => {
    api.agentUpdate.mockResolvedValue(response);
    api.runAgentUpdateAction.mockRejectedValue(new ConnectError("a different action is running", Code.FailedPrecondition));
    const { result } = renderHook(() => useAgentUpdates(true, "local", "codex"));
    await waitFor(() => { expect(result.current.updates.codex).toBeTruthy(); });
    await act(async () => { await result.current.run("codex", "update"); });
    expect(api.runAgentUpdateAction).toHaveBeenCalledOnce();
    expect(result.current.runs.codex).toMatchObject({ phase: "failed", message: "a different action is running" });
  });
});
