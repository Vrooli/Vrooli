import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({ getAgentPreferences: vi.fn(), updateAgentPreferences: vi.fn() }));
vi.mock("../../api/shortcuts", () => ({ shortcutsClient: api }));
import { useAgentPreferences } from "./useAgentPreferences";

const value = { profileId: "workspace", profileInstance: "instance-1", orderedAgentIds: ["codex", "claude"], hiddenAgentIds: [], revision: 2n, orderMode: "custom", lastLaunchedAt: {} };

describe("useAgentPreferences", () => {
  beforeEach(() => { vi.resetAllMocks(); api.getAgentPreferences.mockResolvedValue({ preferences: value }); });

  it("saves the owner revision and profile instance without touching shortcut commands", async () => {
    api.updateAgentPreferences.mockResolvedValue({ preferences: { ...value, hiddenAgentIds: ["codex"], revision: 3n } });
    const { result } = renderHook(() => useAgentPreferences(true, "workspace"));
    await waitFor(() => { expect(result.current.preferences?.revision).toBe(2n); });
    await act(async () => { await result.current.update({ orderedAgentIds: ["codex", "claude"], hiddenAgentIds: ["codex"] }); });
    expect(api.updateAgentPreferences).toHaveBeenCalledWith({ profileId: "workspace", profileInstance: "instance-1", expectedRevision: 2n, orderedAgentIds: ["codex", "claude"], hiddenAgentIds: ["codex"], orderMode: "custom" });
    expect(result.current.preferences?.hiddenAgentIds).toEqual(["codex"]);
  });

  it("reloads the owner's order and reports a conflicting write", async () => {
    api.updateAgentPreferences.mockRejectedValue(new Error("conflict"));
    api.getAgentPreferences.mockResolvedValueOnce({ preferences: value }).mockResolvedValueOnce({ preferences: { ...value, revision: 3n, orderedAgentIds: ["claude", "codex"] } });
    const { result } = renderHook(() => useAgentPreferences(true, "workspace"));
    await waitFor(() => { expect(result.current.preferences?.revision).toBe(2n); });
    await act(async () => { await result.current.update({ orderedAgentIds: ["codex"], hiddenAgentIds: [] }); });
    expect(result.current.error).toBe(true);
    expect(result.current.preferences?.orderedAgentIds).toEqual(["claude", "codex"]);
    expect(result.current.preferences?.revision).toBe(3n);
  });

  it("persists the order mode and keeps the fields a change does not name", async () => {
    api.updateAgentPreferences.mockResolvedValue({ preferences: { ...value, orderMode: "recent", revision: 3n } });
    const { result } = renderHook(() => useAgentPreferences(true, "workspace"));
    await waitFor(() => { expect(result.current.preferences?.revision).toBe(2n); });
    await act(async () => { await result.current.update({ orderMode: "recent" }); });
    expect(api.updateAgentPreferences).toHaveBeenCalledWith(expect.objectContaining({ orderMode: "recent", orderedAgentIds: ["codex", "claude"], hiddenAgentIds: [] }));
    expect(result.current.preferences?.orderMode).toBe("recent");
  });

  it("records a launch without reporting a failure, retrying a conflict once from fresh state", async () => {
    const { result } = renderHook(() => useAgentPreferences(true, "workspace"));
    await waitFor(() => { expect(result.current.preferences?.revision).toBe(2n); });
    api.updateAgentPreferences.mockRejectedValueOnce(new Error("conflict")).mockResolvedValueOnce({ preferences: value });
    api.getAgentPreferences.mockResolvedValueOnce({ preferences: { ...value, revision: 9n } });
    act(() => { result.current.recordLaunch("codex"); });
    await waitFor(() => { expect(api.updateAgentPreferences).toHaveBeenCalledTimes(2); });
    expect(api.updateAgentPreferences).toHaveBeenLastCalledWith(expect.objectContaining({ expectedRevision: 9n, launchedAgentId: "codex" }));
    expect(result.current.error).toBe(false);
  });

  it("does not apply preferences from a different effective profile", async () => {
    api.getAgentPreferences.mockResolvedValue({ preferences: { ...value, profileId: "other" } });
    const { result } = renderHook(() => useAgentPreferences(true, "workspace"));
    await waitFor(() => { expect(result.current.error).toBe(true); });
    expect(result.current.preferences).toBeNull();
  });
});
