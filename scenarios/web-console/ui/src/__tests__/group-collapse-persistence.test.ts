import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";

import { useGroupActions } from "../hooks/useGroupActions";
import { useWorkspaceStore, type TabGroupMeta } from "../stores/useWorkspaceStore";

const mockUpdateTabGroup = vi.fn().mockResolvedValue(undefined);

vi.mock("../api/workspace", () => ({
  saveWorkspaceLayout: vi.fn().mockResolvedValue(undefined),
  updateWorkspacePane: vi.fn().mockResolvedValue(undefined),
  createTabGroup: vi.fn().mockResolvedValue(undefined),
  updateTabGroup: (...args: unknown[]) => mockUpdateTabGroup(...args) as unknown,
  deleteTabGroup: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("../api/workspaceRoles", () => ({
  createRole: vi.fn().mockResolvedValue({}),
  updateRole: vi.fn().mockResolvedValue({}),
  deleteRole: vi.fn().mockResolvedValue(undefined),
  listRoles: vi.fn().mockResolvedValue([]),
}));

const group: TabGroupMeta = { id: "g1", name: "Ship it", color: "#22d3ee", isCollapsed: false };

describe("group collapse persistence", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWorkspaceStore.setState({ panes: [], groups: [{ ...group }], roles: [], activePane: null });
  });

  // Collapse is server-hydrated on reload, so a toggle that only reached the
  // store looked like it had never persisted: every group reopened on refresh.
  it("writes the collapsed state to the backend when a group is collapsed", () => {
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.toggleGroupCollapsed("g1"); });

    expect(useWorkspaceStore.getState().groups[0]?.isCollapsed).toBe(true);
    expect(mockUpdateTabGroup).toHaveBeenCalledWith("g1", { is_collapsed: true });
  });

  // The expand half has to persist too, or a collapsed group could never be
  // reopened across a reload.
  it("writes the expanded state to the backend when a group is expanded", () => {
    useWorkspaceStore.setState({ groups: [{ ...group, isCollapsed: true }] });
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.toggleGroupCollapsed("g1"); });

    expect(useWorkspaceStore.getState().groups[0]?.isCollapsed).toBe(false);
    expect(mockUpdateTabGroup).toHaveBeenCalledWith("g1", { is_collapsed: false });
  });

  // Guards the read-back: negating a stale value would persist the opposite of
  // what the store now holds.
  it("persists the value the store ended on, not the one it started from", () => {
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.toggleGroupCollapsed("g1"); });
    act(() => { result.current.toggleGroupCollapsed("g1"); });

    expect(mockUpdateTabGroup).toHaveBeenNthCalledWith(1, "g1", { is_collapsed: true });
    expect(mockUpdateTabGroup).toHaveBeenNthCalledWith(2, "g1", { is_collapsed: false });
  });

  it("does nothing for a group that is not in the store", () => {
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.toggleGroupCollapsed("missing"); });

    expect(mockUpdateTabGroup).not.toHaveBeenCalled();
  });
});
