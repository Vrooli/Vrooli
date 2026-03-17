import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import TabBar from "../components/TabBar";
import { useWorkspaceStore, type PaneMetadata, type TabGroupMeta } from "../stores/useWorkspaceStore";

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

vi.mock("../lib/api", () => ({
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

describe("TabBar create group", () => {
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

  it("creates group via API, assigns pane, and enters group rename mode", async () => {
    renderTabBar();

    // Open context menu on tab "a"
    fireEvent.contextMenu(screen.getByTestId("tab-a"), { clientX: 50, clientY: 10 });

    // Hover "Add to Group" to open submenu
    const addToGroupBtn = screen.getByTestId("tab-ctx-add-to-group");
    fireEvent.pointerEnter(addToGroupBtn);

    // Click "New Group..."
    const newGroupBtn = screen.getByTestId("tab-ctx-new-group");
    fireEvent.click(newGroupBtn);

    // Wait for async group creation
    await waitFor(() => {
      expect(mockCreateTabGroup).toHaveBeenCalledWith({ name: "New Group", color: "#3b82f6" });
    });

    // Group should be added to store
    await waitFor(() => {
      const groups = useWorkspaceStore.getState().groups;
      expect(groups).toHaveLength(1);
      expect(groups[0]?.id).toBe("g-new");
    });

    // Pane "a" should be assigned to the group
    const pane = useWorkspaceStore.getState().panes.find((p) => p.sessionId === "a");
    expect(pane?.groupId).toBe("g-new");

    // Backend sync for pane update should have been called
    expect(mockUpdateWorkspacePane).toHaveBeenCalledWith("a", { group_id: "g-new" });
  });
});
