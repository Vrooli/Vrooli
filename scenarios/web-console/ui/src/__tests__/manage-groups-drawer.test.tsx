import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import ManageGroupsDrawer from "../components/ManageGroupsDrawer";
import { useWorkspaceStore, type PaneMetadata, type RoleMeta, type TabGroupMeta } from "../stores/useWorkspaceStore";
import { HEADER_COLORS } from "../consts/config";
import { strings } from "../consts/strings";

// [REQ:P0-014c] Group Assignment And Administration Split
// [REQ:P0-014f] Group Auto-Close With Undo

const mockUpdateWorkspacePane = vi.fn().mockResolvedValue(undefined);
const mockCreateTabGroup = vi.fn().mockResolvedValue({
  id: "g-new",
  name: "New Group",
  color: HEADER_COLORS[1],
  sort_order: 0,
  is_collapsed: false,
});
const mockUpdateTabGroup = vi.fn().mockResolvedValue(undefined);
const mockDeleteTabGroup = vi.fn().mockResolvedValue(undefined);

vi.mock("../api/workspace", () => ({
  saveWorkspaceLayout: vi.fn().mockResolvedValue(undefined),
  updateWorkspacePane: (...args: unknown[]) => mockUpdateWorkspacePane(...args) as unknown,
  createTabGroup: (...args: unknown[]) => mockCreateTabGroup(...args) as unknown,
  updateTabGroup: (...args: unknown[]) => mockUpdateTabGroup(...args) as unknown,
  deleteTabGroup: (...args: unknown[]) => mockDeleteTabGroup(...args) as unknown,
}));

vi.mock("../api/workspaceRoles", () => ({
  createRole: vi.fn().mockResolvedValue({}),
  updateRole: vi.fn().mockResolvedValue({}),
  deleteRole: vi.fn().mockResolvedValue(undefined),
  listRoles: vi.fn().mockResolvedValue([]),
}));

const makePanes = (assignments: Record<string, string | null>): PaneMetadata[] =>
  Object.entries(assignments).map(([id, groupId]) => ({
    sessionId: id,
    name: id,
    headerColor: "transparent",
    themeId: "default",
    fontSize: 14,
    groupId,
    supportsMessagesView: false,
    manuallyUnread: false,
  }));

const role = (id: string, groupId: string): RoleMeta => ({
  id,
  groupId,
  label: id,
  command: "agent",
  workingDir: "",
  incomingPrompt: "",
  backend: "",
  targetId: "",
  sessionId: null,
  sortOrder: 0,
});

const groupA: TabGroupMeta = { id: "ga", name: "Alpha", color: HEADER_COLORS[0] ?? "#111111", isCollapsed: false };
const groupB: TabGroupMeta = { id: "gb", name: "Beta", color: HEADER_COLORS[2] ?? "#222222", isCollapsed: false };

function setStore(overrides: Partial<ReturnType<typeof useWorkspaceStore.getState>> = {}) {
  useWorkspaceStore.setState({
    panes: makePanes({ s1: "ga", s2: "ga", s3: null }),
    activePane: "s1",
    groups: [groupA, groupB],
    roles: [],
    closedGroupUndo: null,
    autoCloseEmptyGroups: true,
    manageGroupsOpen: true,
    ...overrides,
  });
}

describe("ManageGroupsDrawer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setStore();
  });

  it("renders nothing while closed", () => {
    setStore({ manageGroupsOpen: false });
    render(<ManageGroupsDrawer />);
    expect(screen.queryByTestId("manage-groups-drawer")).not.toBeInTheDocument();
  });

  it("lists every group with its session count", () => {
    render(<ManageGroupsDrawer />);
    expect(screen.getByTestId("manage-groups-row-ga")).toBeInTheDocument();
    expect(screen.getByTestId("manage-groups-row-gb")).toBeInTheDocument();
    // The visible text is an interpolated translation; the count itself
    // rides on a data attribute so the assertion is about the number.
    expect(screen.getByTestId("manage-groups-count-ga")).toHaveAttribute("data-session-count", "2");
  });

  // The drawer is an administration surface only. Assignment moved to an
  // anchored picker beside the tab, which is why the drawer no longer has to
  // know about a session at all.
  it("offers no per-group assign control", () => {
    render(<ManageGroupsDrawer />);
    expect(screen.queryByTestId("manage-groups-assign-ga")).not.toBeInTheDocument();
    expect(screen.queryByTestId("manage-groups-unassign-ga")).not.toBeInTheDocument();
  });

  it("splits groups into Active and Empty", () => {
    render(<ManageGroupsDrawer />);
    const active = screen.getByTestId("manage-groups-section-active");
    const empty = screen.getByTestId("manage-groups-section-empty");
    expect(active).toContainElement(screen.getByTestId("manage-groups-row-ga"));
    expect(empty).toContainElement(screen.getByTestId("manage-groups-row-gb"));
  });

  it("names the waiting roles that keep an empty group alive", () => {
    setStore({ roles: [role("r1", "gb"), role("r2", "gb")] });
    render(<ManageGroupsDrawer />);
    expect(screen.getByTestId("manage-groups-waiting-gb")).toHaveAttribute("data-waiting-count", "2");
  });

  it("filters groups by name", () => {
    render(<ManageGroupsDrawer />);
    fireEvent.change(screen.getByTestId("manage-groups-filter"), { target: { value: "beta" } });
    expect(screen.queryByTestId("manage-groups-row-ga")).not.toBeInTheDocument();
    expect(screen.getByTestId("manage-groups-row-gb")).toBeInTheDocument();
  });

  it("reports when a filter matches nothing", () => {
    render(<ManageGroupsDrawer />);
    fireEvent.change(screen.getByTestId("manage-groups-filter"), { target: { value: "zzz" } });
    expect(screen.getByTestId("manage-groups-no-matches")).toBeInTheDocument();
  });

  // Closing routes through closeGroup, so it captures an undo snapshot and
  // needs no per-row confirm of its own.
  it("closes one group and leaves it undoable", async () => {
    render(<ManageGroupsDrawer />);
    fireEvent.click(screen.getByTestId("manage-groups-close-gb"));

    await waitFor(() => { expect(mockDeleteTabGroup).toHaveBeenCalledWith("gb"); });
    expect(useWorkspaceStore.getState().groups.map((g) => g.id)).toEqual(["ga"]);
    expect(useWorkspaceStore.getState().closedGroupUndo?.group.id).toBe("gb");
  });

  it("closes a whole selection in one action", async () => {
    render(<ManageGroupsDrawer />);
    fireEvent.click(screen.getByTestId("manage-groups-select-ga"));
    fireEvent.click(screen.getByTestId("manage-groups-select-gb"));
    expect(screen.getByTestId("manage-groups-bulk-bar")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /manageGroups\.closeSelected/ }));
    await waitFor(() => { expect(mockDeleteTabGroup).toHaveBeenCalledTimes(2); });
    expect(useWorkspaceStore.getState().groups).toHaveLength(0);
  });

  it("sweeps every empty group at once", async () => {
    setStore({ groups: [groupA, groupB, { id: "gc", name: "Gamma", color: "#333", isCollapsed: false }] });
    render(<ManageGroupsDrawer />);
    fireEvent.click(screen.getByTestId("manage-groups-close-all-empty"));

    await waitFor(() => { expect(mockDeleteTabGroup).toHaveBeenCalledTimes(2); });
    // The active group survives the sweep.
    expect(useWorkspaceStore.getState().groups.map((g) => g.id)).toEqual(["ga"]);
  });

  it("ungroups a closed group's sessions rather than closing them", async () => {
    render(<ManageGroupsDrawer />);
    fireEvent.click(screen.getByTestId("manage-groups-close-ga"));

    await waitFor(() => { expect(mockDeleteTabGroup).toHaveBeenCalledWith("ga"); });
    const panes = useWorkspaceStore.getState().panes;
    expect(panes).toHaveLength(3);
    expect(panes.every((p) => p.groupId === null)).toBe(true);
  });

  it("recolors a group in place", async () => {
    render(<ManageGroupsDrawer />);
    fireEvent.click(screen.getByTestId("manage-groups-recolor-ga"));
    const nextColor = HEADER_COLORS[3] ?? "#444444";
    fireEvent.click(screen.getByTestId(`manage-groups-color-${nextColor}`));

    await waitFor(() => { expect(mockUpdateTabGroup).toHaveBeenCalledWith("ga", { color: nextColor }); });
    expect(useWorkspaceStore.getState().groups.find((g) => g.id === "ga")?.color).toBe(nextColor);
  });

  it("creates a group from the footer", async () => {
    render(<ManageGroupsDrawer />);
    fireEvent.click(screen.getByTestId("manage-groups-create"));
    await waitFor(() => { expect(mockCreateTabGroup).toHaveBeenCalled(); });
  });

  it("turns automatic closing off", () => {
    render(<ManageGroupsDrawer />);
    fireEvent.click(screen.getByTestId("manage-groups-auto-close"));
    expect(useWorkspaceStore.getState().autoCloseEmptyGroups).toBe(false);
  });

  it("shows the empty state when no group exists", () => {
    setStore({ groups: [] });
    render(<ManageGroupsDrawer />);
    expect(screen.getByTestId("manage-groups-empty")).toBeInTheDocument();
  });

  // The header describes the workspace, not the filtered view: a filter that
  // hides clutter must not hide the count that says how much there is.
  it("states how many groups exist and how many are empty", () => {
    render(<ManageGroupsDrawer />);
    const summary = screen.getByTestId("manage-groups-summary");
    expect(summary).toHaveTextContent(strings.manageGroups.groupCount);
    expect(summary).toHaveTextContent(strings.manageGroups.emptyCount);
  });

  it("switches the list between recent and name order", () => {
    render(<ManageGroupsDrawer />);
    const sort = screen.getByTestId("manage-groups-sort");
    expect(sort).toHaveTextContent(strings.manageGroups.sortRecent);
    fireEvent.click(sort);
    expect(sort).toHaveTextContent(strings.manageGroups.sortName);
  });
});
