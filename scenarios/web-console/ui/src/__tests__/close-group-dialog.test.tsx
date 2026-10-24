import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";

import CloseGroupDialog from "../components/CloseGroupDialog";
import { useWorkspaceStore, type PaneMetadata, type RoleMeta, type TabGroupMeta } from "../stores/useWorkspaceStore";
import { strings } from "../consts/strings";

// [REQ:P0-014c] Group Assignment And Administration Split
// [REQ:P0-014f] Group Auto-Close With Undo

const group: TabGroupMeta = { id: "g1", name: "Test", color: "#22d3ee", isCollapsed: false };

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

const role = (id: string): RoleMeta => ({
  id,
  groupId: "g1",
  label: id,
  command: "codex",
  workingDir: "",
  incomingPrompt: "",
  backend: "",
  targetId: "",
  sessionId: null,
  sortOrder: 0,
});

function setStore(overrides: Partial<ReturnType<typeof useWorkspaceStore.getState>> = {}) {
  useWorkspaceStore.setState({
    groups: [group],
    panes: [pane("s1", "g1"), pane("s2", "g1"), pane("s3", null)],
    roles: [],
    closeGroupTarget: "g1",
    ...overrides,
  });
}

describe("CloseGroupDialog", () => {
  let onCloseGroup: ReturnType<typeof vi.fn>;
  let onCloseSession: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.clearAllMocks();
    onCloseGroup = vi.fn();
    onCloseSession = vi.fn();
    setStore();
  });

  const renderDialog = () =>
    render(<CloseGroupDialog onCloseGroup={onCloseGroup} onCloseSession={onCloseSession} />);

  it("renders nothing when no group is targeted", () => {
    setStore({ closeGroupTarget: null });
    renderDialog();
    expect(screen.queryByTestId("close-group-also-sessions")).not.toBeInTheDocument();
  });

  it("states how many sessions and waiting roles the group holds", () => {
    setStore({ roles: [role("r1")] });
    renderDialog();
    const summary = screen.getByTestId("close-group-summary");
    expect(summary).toHaveTextContent(strings.manageGroups.sessionCount);
    expect(summary).toHaveTextContent(strings.roles.waitingCount);
  });

  // The safe path is the default: the group goes, the sessions survive.
  it("closes only the group when the box is left unchecked", () => {
    renderDialog();
    fireEvent.click(screen.getByTestId("close-group-confirm"));
    expect(onCloseGroup).toHaveBeenCalledWith("g1");
    expect(onCloseSession).not.toHaveBeenCalled();
    expect(useWorkspaceStore.getState().closeGroupTarget).toBeNull();
  });

  // The whole point of the feature: not having to close each session by hand.
  it("closes every member session when the box is checked", () => {
    renderDialog();
    fireEvent.click(screen.getByLabelText(strings.groupContextMenu.closeSessionsLabel));
    fireEvent.click(screen.getByTestId("close-group-confirm"));

    expect(onCloseGroup).toHaveBeenCalledWith("g1");
    expect(onCloseSession.mock.calls.map((call) => call[0] as string)).toEqual(["s1", "s2"]);
    // A session outside the group is never touched.
    expect(onCloseSession).not.toHaveBeenCalledWith("s3");
  });

  it("does nothing on cancel", () => {
    renderDialog();
    fireEvent.click(screen.getByTestId("close-group-cancel"));
    expect(onCloseGroup).not.toHaveBeenCalled();
    expect(onCloseSession).not.toHaveBeenCalled();
    expect(useWorkspaceStore.getState().closeGroupTarget).toBeNull();
  });

  // A group with nothing in it cannot destroy anything, so it is not asked.
  it("omits the checkbox for a group with no sessions", () => {
    setStore({ panes: [pane("s3", null)] });
    renderDialog();
    expect(screen.queryByTestId("close-group-also-sessions")).not.toBeInTheDocument();
    expect(screen.getByTestId("close-group-consequence")).toHaveTextContent(strings.groupContextMenu.closeUndoNote);
  });

  it("says which of the two outcomes is about to happen", () => {
    renderDialog();
    expect(screen.getByTestId("close-group-consequence")).toHaveTextContent(strings.groupContextMenu.closeKeepHint);
    fireEvent.click(screen.getByLabelText(strings.groupContextMenu.closeSessionsLabel));
    expect(screen.getByTestId("close-group-consequence")).not.toHaveTextContent(strings.groupContextMenu.closeKeepHint);
  });
});
