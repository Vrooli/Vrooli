import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import SessionSidebar from "../components/SessionSidebar";
import { useWorkspaceStore, type PaneMetadata } from "../stores/useWorkspaceStore";
import { buildOriginBucketedNavigation } from "../lib/workspaceNavigation";

/**
 * "Mark as unread" is a flag the user sets, not a state the app derives.
 *
 * It cannot reuse the conversation read cursor for two independent reasons:
 * the cursor only ever moves forward (it records what was actually displayed),
 * and it exists only for message-capable sessions — so a plain terminal, which
 * is most of what a user wants to come back to, could never carry a badge.
 */

const mockUpdateWorkspacePane = vi.fn().mockResolvedValue(undefined);
vi.mock("../api/workspace", () => ({
  saveWorkspaceLayout: vi.fn().mockResolvedValue(undefined),
  updateWorkspacePane: (...args: unknown[]) => mockUpdateWorkspacePane(...args) as unknown,
  createTabGroup: vi.fn().mockResolvedValue(undefined),
  updateTabGroup: vi.fn().mockResolvedValue(undefined),
  deleteTabGroup: vi.fn().mockResolvedValue(undefined),
}));

const pane = (sessionId: string, manuallyUnread = false): PaneMetadata => ({
  sessionId,
  name: sessionId,
  headerColor: "transparent",
  themeId: "default",
  fontSize: 14,
  groupId: null,
  supportsMessagesView: false,
  manuallyUnread,
});

function renderSidebar() {
  const { panes, activePane } = useWorkspaceStore.getState();
  const buckets = buildOriginBucketedNavigation({
    panes,
    groups: [],
    activePane,
    originBySession: Object.fromEntries(panes.map((p) => [p.sessionId, "ui" as const])),
  });
  return render(
    <SessionSidebar
      buckets={buckets}
      containerRef={{ current: null }}
      isMobile={false}
      mobileOpen={false}
      onCloseMobile={() => {}}
      onActivatePane={() => {}}
      onClosePane={() => {}}
      onNewTerminal={() => {}}
      onOpenLauncher={() => {}}
      onNewSessionInGroup={() => {}}
      onOpenSettings={() => {}}
    />,
  );
}

describe("manual unread flag", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWorkspaceStore.setState({ panes: [pane("a"), pane("b")], activePane: "a", groups: [] });
  });

  it("shows a dot — with no number — for a flagged terminal session", () => {
    useWorkspaceStore.setState({ panes: [pane("a", true), pane("b")] });
    renderSidebar();

    const dot = screen.getByTestId("sidebar-manual-unread-a");
    expect(dot).toBeTruthy();
    expect(dot.textContent).toBe("");
    expect(screen.queryByTestId("sidebar-manual-unread-b")).toBeNull();
    cleanup();
  });

  it("marks a session unread from the context menu and persists it", () => {
    renderSidebar();
    fireEvent.contextMenu(screen.getByTestId("sidebar-session-b"), { clientX: 10, clientY: 10 });
    fireEvent.click(screen.getByTestId("tab-ctx-toggle-unread"));

    expect(useWorkspaceStore.getState().panes.find((p) => p.sessionId === "b")?.manuallyUnread).toBe(true);
    expect(mockUpdateWorkspacePane).toHaveBeenCalledWith("b", { manually_unread: true });
    cleanup();
  });

  it("clears the flag again from the same menu entry", () => {
    useWorkspaceStore.setState({ panes: [pane("a"), pane("b", true)] });
    renderSidebar();
    fireEvent.contextMenu(screen.getByTestId("sidebar-session-b"), { clientX: 10, clientY: 10 });
    fireEvent.click(screen.getByTestId("tab-ctx-toggle-unread"));

    expect(useWorkspaceStore.getState().panes.find((p) => p.sessionId === "b")?.manuallyUnread).toBe(false);
    expect(mockUpdateWorkspacePane).toHaveBeenCalledWith("b", { manually_unread: false });
    cleanup();
  });
});

describe("manual unread clearing rule", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ panes: [pane("a"), pane("b")], activePane: "a", groups: [] });
  });

  it("survives being set on the session you are already looking at", () => {
    // The whole point of flagging is "come back to this". If activating the
    // pane cleared it unconditionally, flagging the current session would be
    // undone by the next click inside that very pane.
    useWorkspaceStore.getState().setPaneManuallyUnread("a", true);
    const cleared = useWorkspaceStore.getState().setActivePane("a");

    expect(cleared).toBe(false);
    expect(useWorkspaceStore.getState().panes.find((p) => p.sessionId === "a")?.manuallyUnread).toBe(true);
  });

  it("clears when you leave and come back", () => {
    useWorkspaceStore.getState().setPaneManuallyUnread("a", true);
    useWorkspaceStore.getState().setActivePane("b");
    expect(useWorkspaceStore.getState().panes.find((p) => p.sessionId === "a")?.manuallyUnread).toBe(true);

    const cleared = useWorkspaceStore.getState().setActivePane("a");
    expect(cleared).toBe(true);
    expect(useWorkspaceStore.getState().panes.find((p) => p.sessionId === "a")?.manuallyUnread).toBe(false);
  });

  it("reports no clear when activating an unflagged pane", () => {
    // The caller uses this to decide whether to spend a backend write.
    expect(useWorkspaceStore.getState().setActivePane("b")).toBe(false);
  });

  it("leaves other panes' flags alone", () => {
    useWorkspaceStore.setState({ panes: [pane("a", true), pane("b", true)], activePane: "a" });
    useWorkspaceStore.getState().setActivePane("b");
    expect(useWorkspaceStore.getState().panes.find((p) => p.sessionId === "a")?.manuallyUnread).toBe(true);
    expect(useWorkspaceStore.getState().panes.find((p) => p.sessionId === "b")?.manuallyUnread).toBe(false);
  });
});
