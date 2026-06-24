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
  onOpenSettings: vi.fn(),
};

beforeEach(() => {
  useWorkspaceStore.setState({ groups: [], sidebarSortMode: "manual", panes: [] });
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
});
