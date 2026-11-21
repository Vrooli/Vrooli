import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import GroupContextMenu from "../components/GroupContextMenu";
import type { TabGroupMeta } from "../stores/useWorkspaceStore";

const group: TabGroupMeta = {
  id: "g1",
  name: "Work",
  color: "#123456",
  isCollapsed: false,
};

function renderMenu(overrides: Partial<React.ComponentProps<typeof GroupContextMenu>> = {}) {
  const props = {
    position: { x: 10, y: 10 },
    group,
    onNewSession: vi.fn(),
    onToggleCollapse: vi.fn(),
    onManageGroups: vi.fn(),
    onCloseGroup: vi.fn(),
    onDismiss: vi.fn(),
    ...overrides,
  };
  render(<GroupContextMenu {...props} />);
  return props;
}

describe("GroupContextMenu", () => {
  // Bulk administration stays in the manager; what belongs here is what an
  // operator wants to do to THIS group without leaving the list it is in.
  it("keeps bulk administration out: no rename/recolor/ungroup items", () => {
    renderMenu();
    expect(screen.queryByTestId("group-ctx-rename")).not.toBeInTheDocument();
    expect(screen.queryByTestId("group-ctx-recolor")).not.toBeInTheDocument();
    expect(screen.queryByTestId("group-ctx-ungroup-all")).not.toBeInTheDocument();
  });

  // Closing a group had no entry point on the group itself, so the way it got
  // done was closing every session by hand — which left the group behind,
  // because a group outlives its members.
  it("offers closing the group from the header itself", () => {
    const props = renderMenu();
    fireEvent.click(screen.getByTestId("group-ctx-close-group"));
    expect(props.onCloseGroup).toHaveBeenCalled();
  });

  it("fires collapse toggle", () => {
    const props = renderMenu();
    fireEvent.click(screen.getByTestId("group-ctx-toggle-collapse"));
    expect(props.onToggleCollapse).toHaveBeenCalled();
    expect(props.onDismiss).toHaveBeenCalled();
  });

  it("fires new-session-in-group and dismisses", () => {
    const props = renderMenu();
    fireEvent.click(screen.getByTestId("group-ctx-new-session"));
    expect(props.onNewSession).toHaveBeenCalled();
    expect(props.onDismiss).toHaveBeenCalled();
  });

  it("hides new-session when not provided", () => {
    renderMenu({ onNewSession: undefined });
    expect(screen.queryByTestId("group-ctx-new-session")).not.toBeInTheDocument();
  });

  it("opens the Manage Groups drawer entry", () => {
    const props = renderMenu();
    fireEvent.click(screen.getByTestId("group-ctx-manage-groups"));
    expect(props.onManageGroups).toHaveBeenCalled();
    expect(props.onDismiss).toHaveBeenCalled();
  });
});
