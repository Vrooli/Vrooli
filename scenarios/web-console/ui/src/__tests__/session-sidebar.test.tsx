import { describe, it, expect, vi, beforeEach } from "vitest";
import { createRef } from "react";
import { render, screen, fireEvent, act, waitFor } from "@testing-library/react";
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

const listArchivedSessions = vi.fn();
vi.mock("../api/sessions", () => ({
  listArchivedSessions: () => listArchivedSessions(),
}));

const pane = (sessionId: string, headerColor: string): PaneMetadata => ({
  sessionId,
  name: sessionId,
  headerColor,
  themeId: "default",
  fontSize: 14,
  groupId: null,
  supportsMessagesView: false,
  manuallyUnread: false,
});

// These tests exercise the (UI-origin) single-bucket path, where the sidebar
// renders exactly as it did before origin tabs. Wrap the flat item list in the
// one "ui" bucket the component now consumes.
const asBuckets = (items: ReturnType<typeof buildWorkspaceNavigationItems>) =>
  [{ bucket: "ui" as const, items }];

const baseProps = {
  containerRef: createRef<HTMLElement>(),
  isMobile: false,
  mobileOpen: false,
  onCloseMobile: vi.fn(),
  onActivatePane: vi.fn(),
  onClosePane: vi.fn(),
  onDeletePanePermanently: vi.fn(),
  onNewTerminal: vi.fn(),
  onOpenLauncher: vi.fn(),
  onNewSessionInGroup: vi.fn(),
  onOpenSettings: vi.fn(),
};

beforeEach(() => {
  useWorkspaceStore.setState({ groups: [], sidebarSortMode: "manual", sidebarView: "list", plusButtonBehavior: "launcher", panes: [] });
  vi.clearAllMocks();
  listArchivedSessions.mockResolvedValue({
    total: 2,
    sessions: [
      { id: "arc-1", pane_name: "Release planning", archived_at: "2026-08-18T18:00:00Z", created_at: "", agent_type: "codex", message_count: 12, restore_state: "reopenable" },
      { id: "arc-2", pane_name: "Design review", archived_at: "2026-08-18T17:00:00Z", created_at: "", agent_type: "claude", message_count: 4, restore_state: "read_only" },
    ],
  });
});

describe("SessionSidebar", () => {
  it("renders a 2-color accent as a gradient", () => {
    const items = buildWorkspaceNavigationItems({
      panes: [pane("a", "#aaaaaa|#bbbbbb")],
      groups: [],
      activePane: "a",
    });
    const { container } = render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);
    expect(container.innerHTML).toContain("linear-gradient");
  });

  it("has a sort control that updates the store sort mode", () => {
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);
    const select = screen.getByTestId("sidebar-sort-select");
    expect(select).toHaveValue("manual");
    fireEvent.change(select, { target: { value: "name" } });
    expect(useWorkspaceStore.getState().sidebarSortMode).toBe("name");
  });

  it("shows drag handles only in manual sort mode", () => {
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    const { rerender } = render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);
    expect(screen.getByTestId("sidebar-drag-handle-a")).toBeInTheDocument();

    act(() => useWorkspaceStore.setState({ sidebarSortMode: "name" }));
    rerender(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);
    expect(screen.queryByTestId("sidebar-drag-handle-a")).not.toBeInTheDocument();
  });

  it("closes the mobile drawer when the plus button starts a terminal", () => {
    useWorkspaceStore.setState({ plusButtonBehavior: "new-terminal" });
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} isMobile mobileOpen />);

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
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);

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
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);

    fireEvent.contextMenu(screen.getByTestId("sidebar-group-header-g1"), { clientX: 20, clientY: 40 });
    fireEvent.click(screen.getByTestId("group-ctx-new-session"));

    expect(baseProps.onNewSessionInGroup).toHaveBeenCalledWith("g1");
  });

  it("renders the archive footer when origin tabs are hidden", async () => {
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);
    expect(screen.queryByTestId("sidebar-origin-tabs")).not.toBeInTheDocument();
    expect(screen.getByTestId("sidebar-archive-footer")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("sidebar-archive-footer")).toHaveTextContent("2"));
  });

  it("filters the loaded shallow archive by name without another request", async () => {
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);
    fireEvent.click(screen.getByTestId("sidebar-archive-footer"));
    await screen.findByTestId("sidebar-archive-session-arc-1");
    expect(screen.getByTestId("sidebar-archive-session-arc-2")).toBeInTheDocument();
    fireEvent.change(screen.getByTestId("sidebar-archive-filter"), { target: { value: "release" } });
    expect(screen.getByTestId("sidebar-archive-session-arc-1")).toBeInTheDocument();
    expect(screen.queryByTestId("sidebar-archive-session-arc-2")).not.toBeInTheDocument();
    expect(listArchivedSessions).toHaveBeenCalledTimes(1);
  });

  it("leaves crash-orphan details to the archive drawer", async () => {
    listArchivedSessions.mockResolvedValue({
      total: 1,
      sessions: [{ id: "crash", pane_name: "Crash orphan", archived_at: "2026-08-18T18:00:00Z", created_at: "", agent_type: "codex", message_count: 2, restore_state: "reopenable", awaiting_recovery: true }],
    });
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);
    fireEvent.click(screen.getByTestId("sidebar-archive-footer"));
    await waitFor(() => expect(listArchivedSessions).toHaveBeenCalled());
    expect(screen.queryByTestId("sidebar-archive-session-crash")).not.toBeInTheDocument();
    expect(screen.getByTestId("sidebar-archive-search-all")).toBeInTheDocument();
  });
});
