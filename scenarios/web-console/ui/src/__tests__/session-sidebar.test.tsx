import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { createRef } from "react";
import { screen, fireEvent, act, waitFor, within } from "@testing-library/react";
import SessionSidebar from "../components/SessionSidebar";
import { buildWorkspaceNavigationItems } from "../lib/workspaceNavigation";
import { useWorkspaceStore, type PaneMetadata } from "../stores/useWorkspaceStore";
import { strings } from "../consts/strings";

vi.mock("../hooks/useWorkspaceSync", () => ({
  useWorkspaceSync: () => ({
    syncCreateGroup: vi.fn(),
    syncPaneUpdate: vi.fn(),
    syncPaneOrder: vi.fn(),
    syncUpdateGroup: vi.fn(),
    syncDeleteGroup: vi.fn(),
    syncPaneMove: vi.fn(),
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
  onStartRole: vi.fn(),
  onHandoffToRole: vi.fn(),
  onOpenRoleMenu: vi.fn(),
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

  it("opens the full archive with the clicked sidebar session selected", async () => {
    const onOpenArchiveDrawer = vi.fn();
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    render(<SessionSidebar {...baseProps} onOpenArchiveDrawer={onOpenArchiveDrawer} buckets={asBuckets(items)} />);
    fireEvent.click(screen.getByTestId("sidebar-archive-footer"));

    fireEvent.click(await screen.findByTestId("sidebar-archive-session-arc-1"));

    expect(onOpenArchiveDrawer).toHaveBeenCalledWith("arc-1");
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

  it("routes sidebar controls and returns from the archive view", async () => {
    const onOpenArchiveDrawer = vi.fn();
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    render(<SessionSidebar {...baseProps} onOpenArchiveDrawer={onOpenArchiveDrawer} buckets={asBuckets(items)} />);
    fireEvent.click(screen.getByTestId("sidebar-session-a"));
    fireEvent.click(screen.getByTestId("workspace-sidebar-settings"));
    fireEvent.pointerDown(screen.getByTestId("workspace-sidebar-new"), { pointerType: "mouse", button: 0 });
    fireEvent.pointerUp(screen.getByTestId("workspace-sidebar-new"), { pointerType: "mouse", button: 0 });
    expect(baseProps.onActivatePane).toHaveBeenCalledWith("a");
    expect(baseProps.onOpenSettings).toHaveBeenCalled();
    expect(baseProps.onOpenLauncher).toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("sidebar-archive-footer"));
    await screen.findByTestId("sidebar-archive-view");
    fireEvent.click(screen.getByTestId("sidebar-archive-search-all"));
    expect(onOpenArchiveDrawer).toHaveBeenCalledWith();
    fireEvent.click(screen.getByTestId("sidebar-archive-back"));
    expect(screen.getByTestId("sidebar-sort-select")).toBeInTheDocument();
  });

  it("routes pane context actions through the sidebar", () => {
    const onDeletePanePermanently = vi.fn();
    const onClosePane = vi.fn();
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    useWorkspaceStore.setState({ panes: [pane("a", "transparent")] });
    render(<SessionSidebar {...baseProps} onDeletePanePermanently={onDeletePanePermanently} onClosePane={onClosePane} buckets={asBuckets(items)} />);

    fireEvent.contextMenu(screen.getByTestId("sidebar-session-a"), { clientX: 10, clientY: 20 });
    fireEvent.click(screen.getByTestId("tab-ctx-rename"));
    const input = screen.getByTestId("sidebar-rename-input-a");
    fireEvent.change(input, { target: { value: "renamed" } });
    fireEvent.keyDown(input, { key: "Enter" });

    fireEvent.contextMenu(screen.getByTestId("sidebar-session-a"), { clientX: 10, clientY: 20 });
    fireEvent.click(screen.getByTestId("tab-ctx-toggle-unread"));
    fireEvent.contextMenu(screen.getByTestId("sidebar-session-a"), { clientX: 10, clientY: 20 });
    fireEvent.click(screen.getByTestId("tab-ctx-close"));
    expect(onClosePane).toHaveBeenCalledWith("a");
    fireEvent.contextMenu(screen.getByTestId("sidebar-session-a"), { clientX: 10, clientY: 20 });
    fireEvent.click(screen.getByTestId("tab-ctx-delete-permanently"));
    expect(onDeletePanePermanently).toHaveBeenCalledWith("a");
  });

  // Regression: the overlay used to be rendered inside the archive branch of
  // the sidebar, so "Add to group" from the (default) session list did
  // nothing at all — the menu closed and no surface appeared.
  it("opens the group overlay from the session list, not only the archive view", () => {
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    useWorkspaceStore.setState({
      panes: [pane("a", "transparent")],
      groups: [{ id: "g1", name: "Ship it", color: "#22d3ee", isCollapsed: false }],
    });
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);
    expect(useWorkspaceStore.getState().sidebarView).toBe("list");

    fireEvent.contextMenu(screen.getByTestId("sidebar-session-a"), { clientX: 10, clientY: 20 });
    fireEvent.click(screen.getByTestId("tab-ctx-add-to-group"));

    expect(screen.getByTestId("group-assign-picker")).toBeInTheDocument();
    expect(screen.getByTestId("group-picker-option-g1")).toHaveTextContent("Ship it");
  });

  it("assigns the session to the group whose card is pressed", async () => {
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    useWorkspaceStore.setState({
      panes: [pane("a", "transparent")],
      groups: [{ id: "g1", name: "Ship it", color: "#22d3ee", isCollapsed: false }],
    });
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);

    fireEvent.contextMenu(screen.getByTestId("sidebar-session-a"), { clientX: 10, clientY: 20 });
    fireEvent.click(screen.getByTestId("tab-ctx-add-to-group"));
    fireEvent.click(screen.getByTestId("group-picker-option-g1"));

    await waitFor(() => {
      expect(useWorkspaceStore.getState().panes.find((p) => p.sessionId === "a")?.groupId).toBe("g1");
    });
    // Choosing commits and closes; the overlay is not a place to linger.
    expect(screen.queryByTestId("group-assign-picker")).toBeNull();
  });

  // A group is a thing with a colour and a size, so its card says how big it
  // is. A name-only row made picking one feel like filling in a form field.
  // (The test i18n echoes keys, so the assertion is on which summary the card
  // chose, not on the interpolated number.)
  it("states each group's size on its card", () => {
    const panes = [pane("a", "transparent"), { ...pane("b", "transparent"), groupId: "g1" }];
    const items = buildWorkspaceNavigationItems({ panes, groups: [], activePane: "a" });
    useWorkspaceStore.setState({
      panes,
      groups: [
        { id: "g1", name: "Ship it", color: "#22d3ee", isCollapsed: false },
        { id: "g2", name: "Research", color: "#f59e0b", isCollapsed: false },
      ],
      roles: [{
        id: "r1", groupId: "g1", label: "Reviewer", command: "claude", workingDir: "",
        incomingPrompt: "", backend: "", targetId: "", sessionId: null, sortOrder: 0,
      }],
    });
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);

    fireEvent.contextMenu(screen.getByTestId("sidebar-session-a"), { clientX: 10, clientY: 20 });
    fireEvent.click(screen.getByTestId("tab-ctx-add-to-group"));

    const populated = screen.getByTestId("group-picker-option-g1");
    expect(populated).toHaveTextContent(strings.groupPicker.sessionCount);
    expect(populated).toHaveTextContent(strings.groupPicker.waitingCount);

    // An empty group says so rather than reading "0 sessions".
    const empty = screen.getByTestId("group-picker-option-g2");
    expect(empty).toHaveTextContent(strings.groupPicker.emptyGroup);
    expect(empty).not.toHaveTextContent(strings.groupPicker.sessionCount);
  });

  // Groups holding work and groups holding none are two different decisions:
  // one you pick from, the other you sweep.
  it("splits the picker into active and empty sections", () => {
    const panes = [pane("a", "transparent"), { ...pane("b", "transparent"), groupId: "g1" }];
    const items = buildWorkspaceNavigationItems({ panes, groups: [], activePane: "a" });
    useWorkspaceStore.setState({
      panes,
      groups: [
        { id: "g1", name: "Ship it", color: "#22d3ee", isCollapsed: false },
        { id: "g2", name: "Research", color: "#f59e0b", isCollapsed: false },
      ],
      roles: [],
    });
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);

    fireEvent.contextMenu(screen.getByTestId("sidebar-session-a"), { clientX: 10, clientY: 20 });
    fireEvent.click(screen.getByTestId("tab-ctx-add-to-group"));

    const active = screen.getByTestId("group-picker-section-active");
    const empty = screen.getByTestId("group-picker-section-empty");
    expect(within(active).getByTestId("group-picker-option-g1")).toBeInTheDocument();
    expect(within(empty).getByTestId("group-picker-option-g2")).toBeInTheDocument();
    expect(screen.getByTestId("group-picker-close-all-empty")).toBeInTheDocument();
  });

  // A group with only waiting roles holds no session, so it is still a
  // candidate for the sweep — its row says how many are waiting so the
  // decision is informed rather than blind.
  it("counts a group of waiting roles as empty, and says how many wait", () => {
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    useWorkspaceStore.setState({
      panes: [pane("a", "transparent")],
      groups: [{ id: "g1", name: "Ship it", color: "#22d3ee", isCollapsed: false }],
      roles: [{
        id: "r1", groupId: "g1", label: "Reviewer", command: "claude", workingDir: "",
        incomingPrompt: "", backend: "", targetId: "", sessionId: null, sortOrder: 0,
      }],
    });
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);

    fireEvent.contextMenu(screen.getByTestId("sidebar-session-a"), { clientX: 10, clientY: 20 });
    fireEvent.click(screen.getByTestId("tab-ctx-add-to-group"));

    const empty = screen.getByTestId("group-picker-section-empty");
    expect(within(empty).getByTestId("group-picker-option-g1")).toHaveTextContent(strings.groupPicker.waitingCount);
  });

  it("renames and recolours a group from the picker's edit mode", async () => {
    const items = buildWorkspaceNavigationItems({ panes: [pane("a", "transparent")], groups: [], activePane: "a" });
    useWorkspaceStore.setState({
      panes: [pane("a", "transparent")],
      groups: [{ id: "g1", name: "Ship it", color: "#22d3ee", isCollapsed: false }],
      roles: [],
    });
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);

    fireEvent.contextMenu(screen.getByTestId("sidebar-session-a"), { clientX: 10, clientY: 20 });
    fireEvent.click(screen.getByTestId("tab-ctx-add-to-group"));
    fireEvent.click(screen.getByTestId("group-picker-edit-toggle"));

    fireEvent.change(screen.getByTestId("group-picker-rename-g1"), { target: { value: "Shipped" } });
    await waitFor(() => {
      expect(useWorkspaceStore.getState().groups[0]?.name).toBe("Shipped");
    });

    fireEvent.click(screen.getByTestId("group-picker-recolor-g1"));
    const swatch = within(screen.getByTestId("group-picker-palette-g1")).getAllByRole("button")[1];
    fireEvent.click(swatch as HTMLElement);
    expect(useWorkspaceStore.getState().groups[0]?.color).not.toBe("#22d3ee");
  });

  // The consequence that matters: closing a group releases its sessions.
  it("closing a group from the picker ungroups its sessions instead of ending them", async () => {
    const panes = [pane("a", "transparent"), { ...pane("b", "transparent"), groupId: "g1" }];
    const items = buildWorkspaceNavigationItems({ panes, groups: [], activePane: "a" });
    useWorkspaceStore.setState({
      panes,
      groups: [{ id: "g1", name: "Ship it", color: "#22d3ee", isCollapsed: false }],
      roles: [],
    });
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);

    fireEvent.contextMenu(screen.getByTestId("sidebar-session-a"), { clientX: 10, clientY: 20 });
    fireEvent.click(screen.getByTestId("tab-ctx-add-to-group"));
    fireEvent.click(screen.getByTestId("group-picker-edit-toggle"));
    fireEvent.click(screen.getByTestId("group-picker-close-g1"));

    await waitFor(() => { expect(useWorkspaceStore.getState().groups).toHaveLength(0); });
    const survivors = useWorkspaceStore.getState().panes;
    expect(survivors).toHaveLength(2);
    expect(survivors.find((p) => p.sessionId === "b")?.groupId).toBeNull();
    // And it is undoable: the snapshot is what makes closing safe to offer
    // behind a single tap with no confirm dialog.
    expect(useWorkspaceStore.getState().closedGroupUndo).not.toBeNull();
  });

  it("sweeps every empty group in one action", async () => {
    const panes = [pane("a", "transparent"), { ...pane("b", "transparent"), groupId: "g1" }];
    const items = buildWorkspaceNavigationItems({ panes, groups: [], activePane: "a" });
    useWorkspaceStore.setState({
      panes,
      groups: [
        { id: "g1", name: "Ship it", color: "#22d3ee", isCollapsed: false },
        { id: "g2", name: "Research", color: "#f59e0b", isCollapsed: false },
        { id: "g3", name: "Spike", color: "#a78bfa", isCollapsed: false },
      ],
      roles: [],
    });
    render(<SessionSidebar {...baseProps} buckets={asBuckets(items)} />);

    fireEvent.contextMenu(screen.getByTestId("sidebar-session-a"), { clientX: 10, clientY: 20 });
    fireEvent.click(screen.getByTestId("tab-ctx-add-to-group"));
    fireEvent.click(screen.getByTestId("group-picker-close-all-empty"));

    // The one holding a session survives; both empties go.
    await waitFor(() => { expect(useWorkspaceStore.getState().groups.map((g) => g.id)).toEqual(["g1"]); });
  });
});
