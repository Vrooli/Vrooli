import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import TabBar from "../components/TabBar";
import { useWorkspaceStore, type PaneMetadata, type TabGroupMeta } from "../stores/useWorkspaceStore";

// [REQ:P0-001a] Tab reorder persistence — regression test for drag-drop sync

// jsdom doesn't implement these DOM methods
Element.prototype.scrollIntoView = vi.fn();
Element.prototype.setPointerCapture = vi.fn();
Element.prototype.releasePointerCapture = vi.fn();

// Mock the api module so saveWorkspaceLayout can be spied on
const mockSaveWorkspaceLayout = vi.fn().mockResolvedValue(undefined);

vi.mock("../api/workspace", () => ({
  saveWorkspaceLayout: (...args: unknown[]) => mockSaveWorkspaceLayout(...args) as unknown,
  updateWorkspacePane: vi.fn().mockResolvedValue(undefined),
  createTabGroup: vi.fn().mockResolvedValue({ id: "g1", name: "Group", color: "#3b82f6" }),
  updateTabGroup: vi.fn().mockResolvedValue(undefined),
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
  }));

describe("TabBar reorder sync", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();

    // Reset store to known state
    useWorkspaceStore.setState({
      panes: makePanes("a", "b", "c"),
      activePane: "a",
      groups: [] as TabGroupMeta[],
      displayMode: "tabs",
      tabContextMenu: null,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("calls saveWorkspaceLayout after drag-drop reorder", async () => {
    render(
      <TabBar
        panes={useWorkspaceStore.getState().panes}
        activePane="a"
        onNewTerminal={vi.fn()}
        onOpenLauncher={vi.fn()}
        onClosePane={vi.fn()}
      />,
    );

    const tabA = screen.getByTestId("tab-a");

    // Simulate drag: pointerdown on tab A, move past threshold, drop
    const tabARect = { x: 50, y: 10 };

    // Start drag on tab A
    fireEvent.pointerDown(tabA, {
      button: 0,
      pointerType: "mouse",
      clientX: tabARect.x,
      clientY: tabARect.y,
      pointerId: 1,
    });

    // Move past drag threshold (>5px)
    fireEvent.pointerMove(window, {
      clientX: tabARect.x + 10,
      clientY: tabARect.y,
      pointerId: 1,
    });

    // Drop
    fireEvent.pointerUp(window);

    // Flush the debounce timer (300ms)
    act(() => {
      vi.advanceTimersByTime(400);
    });

    // saveWorkspaceLayout should have been called with the new pane order
    expect(mockSaveWorkspaceLayout).toHaveBeenCalled();
    const call = mockSaveWorkspaceLayout.mock.calls[0] as unknown[] | undefined;
    expect(call?.[0]).toHaveProperty("pane_order");
    expect((call?.[0] as Record<string, unknown>)?.pane_order).toBeInstanceOf(Array);
  });

  it("does not reorder or activate a tab after a touch swipe", () => {
    useWorkspaceStore.setState({ activePane: "b" });

    render(
      <TabBar
        panes={useWorkspaceStore.getState().panes}
        activePane="b"
        onNewTerminal={vi.fn()}
        onOpenLauncher={vi.fn()}
        onClosePane={vi.fn()}
      />,
    );

    const tabA = screen.getByTestId("tab-a");
    const startPoint = { x: 50, y: 10 };

    fireEvent.pointerDown(tabA, {
      button: 0,
      pointerType: "touch",
      clientX: startPoint.x,
      clientY: startPoint.y,
      pointerId: 1,
    });

    fireEvent.pointerMove(window, {
      clientX: startPoint.x + 10,
      clientY: startPoint.y,
      pointerId: 1,
    });

    fireEvent.pointerUp(tabA, {
      button: 0,
      pointerType: "touch",
      clientX: startPoint.x + 10,
      clientY: startPoint.y,
      pointerId: 1,
    });

    act(() => {
      vi.runOnlyPendingTimers();
    });

    expect(useWorkspaceStore.getState().panes.map((pane) => pane.sessionId)).toEqual(["a", "b", "c"]);
    expect(useWorkspaceStore.getState().activePane).toBe("b");
    expect(mockSaveWorkspaceLayout).not.toHaveBeenCalled();
  });

  it("persists active pane when the user switches tabs", () => {
    render(
      <TabBar
        panes={useWorkspaceStore.getState().panes}
        activePane="a"
        onNewTerminal={vi.fn()}
        onOpenLauncher={vi.fn()}
        onClosePane={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByTestId("tab-b"));

    expect(mockSaveWorkspaceLayout).toHaveBeenCalledWith({
      active_pane: "b",
      pane_order: ["a", "b", "c"],
    });
  });
});
