import { describe, it, expect, vi, beforeEach } from "vitest";
import { createRef } from "react";
import { render, screen, fireEvent, act } from "@testing-library/react";
import SessionSidebar from "../components/SessionSidebar";
import { buildWorkspaceNavigationItems } from "../lib/workspaceNavigation";
import { useWorkspaceStore, type PaneMetadata } from "../stores/useWorkspaceStore";

vi.mock("../hooks/useWorkspaceSync", () => ({
  useWorkspaceSync: () => ({
    syncCreateGroup: vi.fn(),
    syncPaneUpdate: vi.fn(),
    syncPaneOrder: vi.fn(),
    syncUpdateGroup: vi.fn(),
    syncDeleteGroup: vi.fn(),
  }),
}));

const pane = (sessionId: string, headerColor: string): PaneMetadata => ({
  sessionId,
  name: sessionId,
  headerColor,
  themeId: "default",
  fontSize: 14,
  groupId: null,
  supportsMessagesView: false,
});

const baseProps = {
  containerRef: createRef<HTMLElement>(),
  isMobile: false,
  mobileOpen: false,
  onCloseMobile: vi.fn(),
  onActivatePane: vi.fn(),
  onClosePane: vi.fn(),
  onNewTerminal: vi.fn(),
  onOpenLauncher: vi.fn(),
  onNewSessionInGroup: vi.fn(),
  onOpenSettings: vi.fn(),
};

beforeEach(() => {
  useWorkspaceStore.setState({ groups: [], sidebarSortMode: "manual", plusButtonBehavior: "launcher", panes: [] });
  vi.clearAllMocks();
});

describe("SessionSidebar", () => {
  it("renders a 2-color accent as a gradient", () => {
    const items = buildWorkspaceNavigationItems({
      panes: [pane("a", "#aaaaaa|#bbbbbb")],
      groups: [],
      activePane: "a",
    });
    const { container } = render(<SessionSidebar {...baseProps} items={items} />);
    expect(container.innerHTML).toContain("linear-gradient");
  });

  it("has a sort control that updates the store sort mode", () => {
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    render(<SessionSidebar {...baseProps} items={items} />);
    const select = screen.getByTestId("sidebar-sort-select");
    expect(select).toHaveValue("manual");
    fireEvent.change(select, { target: { value: "name" } });
    expect(useWorkspaceStore.getState().sidebarSortMode).toBe("name");
  });

  it("shows drag handles only in manual sort mode", () => {
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    const { rerender } = render(<SessionSidebar {...baseProps} items={items} />);
    expect(screen.getByTestId("sidebar-drag-handle-a")).toBeInTheDocument();

    act(() => useWorkspaceStore.setState({ sidebarSortMode: "name" }));
    rerender(<SessionSidebar {...baseProps} items={items} />);
    expect(screen.queryByTestId("sidebar-drag-handle-a")).not.toBeInTheDocument();
  });

  it("closes the mobile drawer when the plus button starts a terminal", () => {
    useWorkspaceStore.setState({ plusButtonBehavior: "new-terminal" });
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    render(<SessionSidebar {...baseProps} items={items} isMobile mobileOpen />);

    fireEvent.pointerDown(screen.getByTestId("workspace-sidebar-new"), { pointerType: "mouse", button: 0 });
    fireEvent.pointerUp(screen.getByTestId("workspace-sidebar-new"), { pointerType: "mouse", button: 0 });

    expect(baseProps.onNewTerminal).toHaveBeenCalledOnce();
    expect(baseProps.onCloseMobile).toHaveBeenCalledOnce();
  });

  it("renders grouped panes with explicit boundary rows", () => {
    const group = { id: "g1", name: "Work", color: "#123456", isCollapsed: false };
    const panes = [
      { ...pane("a", "transparent"), groupId: "g1" },
      { ...pane("b", "transparent"), groupId: "g1" },
      pane("c", "transparent"),
    ];
    const items = buildWorkspaceNavigationItems({ panes, groups: [group], activePane: "a" });
    render(<SessionSidebar {...baseProps} items={items} />);

    expect(screen.getByTestId("sidebar-group-header-g1")).toBeInTheDocument();
    expect(screen.getByTestId("sidebar-session-a").closest("[data-group-id]")).toHaveAttribute("data-group-id", "g1");
    expect(screen.getByTestId("sidebar-session-b").closest("[data-group-id]")).toHaveAttribute("data-group-id", "g1");
    expect(screen.getByTestId("sidebar-session-c").closest("[data-group-id]")).toBeNull();
  });

  it("exposes new-session-in-group from the group context menu", () => {
    const group = { id: "g1", name: "Work", color: "#123456", isCollapsed: false };
    const items = buildWorkspaceNavigationItems({
      panes: [{ ...pane("a", "transparent"), groupId: "g1" }],
      groups: [group],
      activePane: "a",
    });
    useWorkspaceStore.setState({ groups: [group] });
    render(<SessionSidebar {...baseProps} items={items} />);

    fireEvent.contextMenu(screen.getByTestId("sidebar-group-header-g1"), { clientX: 20, clientY: 40 });
    fireEvent.click(screen.getByTestId("group-ctx-new-session"));

    expect(baseProps.onNewSessionInGroup).toHaveBeenCalledWith("g1");
  });
});
