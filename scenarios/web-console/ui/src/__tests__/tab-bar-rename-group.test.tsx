import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import TabBar from "../components/TabBar";
import { useWorkspaceStore, type PaneMetadata, type TabGroupMeta } from "../stores/useWorkspaceStore";
import { HEADER_COLORS } from "../consts/config";

// [REQ:P0-001a] Tab rename + group creation — regression tests

// jsdom doesn't implement these DOM methods
Element.prototype.scrollIntoView = vi.fn();
Element.prototype.setPointerCapture = vi.fn();
Element.prototype.releasePointerCapture = vi.fn();

const mockUpdateWorkspacePane = vi.fn().mockResolvedValue(undefined);
const mockCreateTabGroup = vi.fn().mockResolvedValue({
  id: "g-new",
  name: "New Group",
  color: "#3b82f6",
  sort_order: 0,
  is_collapsed: false,
});
const mockUpdateTabGroup = vi.fn().mockResolvedValue(undefined);

vi.mock("../api/workspace", () => ({
  saveWorkspaceLayout: vi.fn().mockResolvedValue(undefined),
  updateWorkspacePane: (...args: unknown[]) => mockUpdateWorkspacePane(...args) as unknown,
  createTabGroup: (...args: unknown[]) => mockCreateTabGroup(...args) as unknown,
  updateTabGroup: (...args: unknown[]) => mockUpdateTabGroup(...args) as unknown,
  deleteTabGroup: vi.fn().mockResolvedValue(undefined),
}));

const makePanes = (...ids: string[]): PaneMetadata[] =>
  ids.map((id) => ({
    sessionId: id,
    name: id,
    headerColor: "transparent",
    themeId: "default",
    fontSize: 14,
    groupId: null,
    supportsMessagesView: false,
  manuallyUnread: false,
  }));

const renderTabBar = () =>
  render(
    <TabBar
      panes={useWorkspaceStore.getState().panes}
      activePane="a"
      onNewTerminal={vi.fn()}
      onOpenLauncher={vi.fn()}
      onClosePane={vi.fn()}
    />,
  );

describe("TabBar rename", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWorkspaceStore.setState({
      panes: makePanes("a", "b"),
      activePane: "a",
      groups: [] as TabGroupMeta[],
      displayMode: "tabs",
      tabContextMenu: null,
    });
  });

  it("shows inline rename input when Rename is clicked in context menu", () => {
    renderTabBar();

    // Open context menu on tab "a"
    const tabA = screen.getByTestId("tab-a");
    fireEvent.contextMenu(tabA, { clientX: 50, clientY: 10 });

    // Click Rename
    const renameBtn = screen.getByTestId("tab-ctx-rename");
    fireEvent.click(renameBtn);

    // Inline input should appear with current name
    const input = screen.getByTestId("tab-rename-input-a");
    expect(input).toBeInTheDocument();
    expect(input).toHaveValue("a");
  });

  it("commits rename on Enter and syncs to backend", () => {
    renderTabBar();

    // Open context menu and click Rename
    fireEvent.contextMenu(screen.getByTestId("tab-a"), { clientX: 50, clientY: 10 });
    fireEvent.click(screen.getByTestId("tab-ctx-rename"));

    const input = screen.getByTestId("tab-rename-input-a");

    // Type new name and press Enter
    fireEvent.change(input, { target: { value: "my-server" } });
    fireEvent.keyDown(input, { key: "Enter" });

    // Input should be gone
    expect(screen.queryByTestId("tab-rename-input-a")).not.toBeInTheDocument();

    // Store should be updated
    const pane = useWorkspaceStore.getState().panes.find((p) => p.sessionId === "a");
    expect(pane?.name).toBe("my-server");

    // Backend sync should have been called
    expect(mockUpdateWorkspacePane).toHaveBeenCalledWith("a", { name: "my-server" });
  });

  it("cancels rename on Escape without changing name", () => {
    renderTabBar();

    fireEvent.contextMenu(screen.getByTestId("tab-a"), { clientX: 50, clientY: 10 });
    fireEvent.click(screen.getByTestId("tab-ctx-rename"));

    const input = screen.getByTestId("tab-rename-input-a");
    fireEvent.change(input, { target: { value: "changed" } });
    fireEvent.keyDown(input, { key: "Escape" });

    // Input should be gone, name unchanged
    expect(screen.queryByTestId("tab-rename-input-a")).not.toBeInTheDocument();
    const pane = useWorkspaceStore.getState().panes.find((p) => p.sessionId === "a");
    expect(pane?.name).toBe("a");
    expect(mockUpdateWorkspacePane).not.toHaveBeenCalled();
  });
});

describe("TabBar group quick paths", () => {
  const groupMeta: TabGroupMeta = { id: "g1", name: "Work", color: HEADER_COLORS[0] ?? "#123456", isCollapsed: false };

  beforeEach(() => {
    vi.clearAllMocks();
    useWorkspaceStore.setState({
      panes: makePanes("a", "b"),
      activePane: "a",
      groups: [groupMeta],
      displayMode: "tabs",
      tabContextMenu: null,
      manageGroupsTarget: null,
    });
  });

  it("Add to Group opens the Manage Groups drawer with the session as context", () => {
    renderTabBar();

    // Ungrouped pane: no inline group list — assignment lives in the drawer.
    fireEvent.contextMenu(screen.getByTestId("tab-a"), { clientX: 50, clientY: 10 });
    expect(screen.queryByTestId("tab-ctx-group-g1")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tab-ctx-remove-from-group")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tab-ctx-manage-groups")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("tab-ctx-add-to-group"));

    expect(useWorkspaceStore.getState().manageGroupsTarget).toEqual({ sessionId: "a" });
  });

  it("grouped panes get one-tap remove plus the Manage Groups entry point", async () => {
    const panes = useWorkspaceStore.getState().panes.map((p) =>
      p.sessionId === "a" ? { ...p, groupId: "g1" } : p,
    );
    useWorkspaceStore.setState({ panes });
    renderTabBar();

    fireEvent.contextMenu(screen.getByTestId("tab-a"), { clientX: 50, clientY: 10 });
    expect(screen.queryByTestId("tab-ctx-add-to-group")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("tab-ctx-remove-from-group"));

    await waitFor(() => {
      const pane = useWorkspaceStore.getState().panes.find((p) => p.sessionId === "a");
      expect(pane?.groupId).toBeNull();
    });
    // Leaving a group is a move: the pane's membership and its color are
    // persisted together, so the surviving members stay a contiguous block.
    expect(mockUpdateWorkspacePane).toHaveBeenCalledWith("a", {
      group_id: null,
      header_color: "transparent",
    });
  });

  it("opens the Manage Groups drawer from a grouped pane's menu", () => {
    const panes = useWorkspaceStore.getState().panes.map((p) =>
      p.sessionId === "a" ? { ...p, groupId: "g1" } : p,
    );
    useWorkspaceStore.setState({ panes });
    renderTabBar();

    fireEvent.contextMenu(screen.getByTestId("tab-a"), { clientX: 50, clientY: 10 });
    fireEvent.click(screen.getByTestId("tab-ctx-manage-groups"));

    expect(useWorkspaceStore.getState().manageGroupsTarget).toEqual({ sessionId: "a" });
  });
});
