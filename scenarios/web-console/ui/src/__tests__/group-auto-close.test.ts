import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";

import { useGroupActions } from "../hooks/useGroupActions";
import { useWorkspaceStore, type PaneMetadata, type RoleMeta, type TabGroupMeta } from "../stores/useWorkspaceStore";

// [REQ:P0-014f] Group Auto-Close With Undo

const mockCreateTabGroup = vi.fn();
const mockDeleteTabGroup = vi.fn().mockResolvedValue(undefined);
const mockUpdateWorkspacePane = vi.fn().mockResolvedValue(undefined);
const mockCreateRole = vi.fn();

vi.mock("../api/workspace", () => ({
  saveWorkspaceLayout: vi.fn().mockResolvedValue(undefined),
  updateWorkspacePane: (...args: unknown[]) => mockUpdateWorkspacePane(...args) as unknown,
  createTabGroup: (...args: unknown[]) => mockCreateTabGroup(...args) as unknown,
  updateTabGroup: vi.fn().mockResolvedValue(undefined),
  deleteTabGroup: (...args: unknown[]) => mockDeleteTabGroup(...args) as unknown,
}));

vi.mock("../api/workspaceRoles", () => ({
  createRole: (...args: unknown[]) => mockCreateRole(...args) as unknown,
  updateRole: vi.fn().mockResolvedValue({}),
  deleteRole: vi.fn().mockResolvedValue(undefined),
  listRoles: vi.fn().mockResolvedValue([]),
}));

const pane = (sessionId: string, groupId: string | null): PaneMetadata => ({
  sessionId,
  name: sessionId,
  headerColor: "transparent",
  themeId: "default",
  fontSize: 14,
  groupId,
  supportsMessagesView: false,
  manuallyUnread: false,
});

const role = (id: string, groupId: string, sessionId: string | null = null): RoleMeta => ({
  id,
  groupId,
  label: id,
  command: "agent",
  workingDir: "",
  incomingPrompt: "Do {{payload}}",
  backend: "",
  targetId: "",
  sessionId,
  sortOrder: 0,
});

const group: TabGroupMeta = { id: "g1", name: "Ship it", color: "#22d3ee", isCollapsed: false };

function seed(overrides: Partial<ReturnType<typeof useWorkspaceStore.getState>> = {}) {
  useWorkspaceStore.setState({
    panes: [],
    groups: [group],
    roles: [],
    activePane: null,
    closedGroupUndo: null,
    autoCloseEmptyGroups: true,
    ...overrides,
  });
}

describe("auto-close", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCreateTabGroup.mockResolvedValue({ id: "g-restored", name: "Ship it", color: "#22d3ee", sort_order: 0, is_collapsed: false });
    mockCreateRole.mockImplementation((input: { group_id: string; label: string }) =>
      Promise.resolve({
        id: `restored-${input.label}`,
        group_id: input.group_id,
        label: input.label,
        command: "agent",
        working_dir: "",
        incoming_prompt: "Do {{payload}}",
        backend: "",
        target_id: "",
        session_id: null,
        sort_order: 0,
      }));
    seed();
  });

  it("closes a group with no panes and no waiting roles", () => {
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroupIfFinished("g1"); });
    expect(useWorkspaceStore.getState().groups).toHaveLength(0);
    expect(mockDeleteTabGroup).toHaveBeenCalledWith("g1");
  });

  // The exemption is the whole safety argument for closing without asking.
  it("keeps a group that still holds a waiting role", () => {
    seed({ roles: [role("r1", "g1")] });
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroupIfFinished("g1"); });
    expect(useWorkspaceStore.getState().groups).toHaveLength(1);
    expect(mockDeleteTabGroup).not.toHaveBeenCalled();
  });

  it("keeps a group that still holds a pane", () => {
    seed({ panes: [pane("s1", "g1")] });
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroupIfFinished("g1"); });
    expect(useWorkspaceStore.getState().groups).toHaveLength(1);
  });

  it("does nothing when the preference is off", () => {
    seed({ autoCloseEmptyGroups: false });
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroupIfFinished("g1"); });
    expect(useWorkspaceStore.getState().groups).toHaveLength(1);
  });

  it("still closes by hand when the preference is off", () => {
    seed({ autoCloseEmptyGroups: false });
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroup("g1"); });
    expect(useWorkspaceStore.getState().groups).toHaveLength(0);
  });

  it("ignores a group id that is already gone", () => {
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroupIfFinished("missing"); });
    expect(mockDeleteTabGroup).not.toHaveBeenCalled();
  });

  it("ignores a null group id", () => {
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroupIfFinished(null); });
    expect(mockDeleteTabGroup).not.toHaveBeenCalled();
  });
});

describe("undo", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCreateTabGroup.mockResolvedValue({ id: "g-restored", name: "Ship it", color: "#22d3ee", sort_order: 0, is_collapsed: false });
    mockCreateRole.mockImplementation((input: { group_id: string; label: string }) =>
      Promise.resolve({
        id: `restored-${input.label}`,
        group_id: input.group_id,
        label: input.label,
        command: "agent",
        working_dir: "",
        incoming_prompt: "Do {{payload}}",
        backend: "",
        target_id: "",
        session_id: null,
        sort_order: 0,
      }));
    seed();
  });

  it("captures the group, its roles, and its members before deleting", () => {
    seed({ panes: [pane("s1", "g1")], roles: [role("r1", "g1"), role("r2", "g1")] });
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroup("g1"); });

    const snapshot = useWorkspaceStore.getState().closedGroupUndo;
    expect(snapshot?.group.name).toBe("Ship it");
    expect(snapshot?.roles.map((r) => r.id)).toEqual(["r1", "r2"]);
    expect(snapshot?.memberSessionIds).toEqual(["s1"]);
  });

  it("restores the group, every role, and the pane assignments", async () => {
    seed({ panes: [pane("s1", "g1")], roles: [role("r1", "g1"), role("r2", "g1")] });
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroup("g1"); });

    await act(async () => { await result.current.restoreClosedGroup(); });

    const state = useWorkspaceStore.getState();
    expect(state.groups.map((g) => g.name)).toEqual(["Ship it"]);
    expect(state.roles).toHaveLength(2);
    expect(state.panes[0]?.groupId).toBe("g-restored");
    expect(state.closedGroupUndo).toBeNull();
  });

  it("keeps the snapshot when the replay fails, so the operator can retry", async () => {
    mockCreateTabGroup.mockRejectedValueOnce(new Error("offline"));
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => { /* expected */ });
    seed({ roles: [role("r1", "g1")] });
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroup("g1"); });

    let restored = true;
    await act(async () => { restored = await result.current.restoreClosedGroup(); });

    expect(restored).toBe(false);
    // The snapshot is the only copy of the group; dropping it here would
    // lose it for good.
    expect(useWorkspaceStore.getState().closedGroupUndo).not.toBeNull();
    consoleError.mockRestore();
  });

  it("does not resurrect a session that closed during the undo window", async () => {
    seed({ panes: [pane("s1", "g1")], roles: [] });
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroup("g1"); });
    // The session goes away while the banner is up.
    act(() => { useWorkspaceStore.setState({ panes: [] }); });

    await act(async () => { await result.current.restoreClosedGroup(); });
    expect(useWorkspaceStore.getState().panes).toHaveLength(0);
  });

  it("brings a role back waiting when its session is gone", async () => {
    seed({ panes: [], roles: [role("r1", "g1", "dead-session")] });
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroup("g1"); });

    await act(async () => { await result.current.restoreClosedGroup(); });
    expect(mockCreateRole).toHaveBeenCalledWith(expect.objectContaining({ session_id: null }));
  });

  it("replaces the undo slot rather than stacking two banners", () => {
    seed({ groups: [group, { id: "g2", name: "Second", color: "#f00", isCollapsed: false }] });
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroup("g1"); });
    act(() => { result.current.closeGroup("g2"); });
    expect(useWorkspaceStore.getState().closedGroupUndo?.group.id).toBe("g2");
  });

  it("clears the slot when dismissed", () => {
    const { result } = renderHook(() => useGroupActions());
    act(() => { result.current.closeGroup("g1"); });
    act(() => { result.current.dismissClosedGroupUndo(); });
    expect(useWorkspaceStore.getState().closedGroupUndo).toBeNull();
  });

  it("reports false when there is nothing to restore", async () => {
    const { result } = renderHook(() => useGroupActions());
    let restored = true;
    await act(async () => { restored = await result.current.restoreClosedGroup(); });
    expect(restored).toBe(false);
  });
});
