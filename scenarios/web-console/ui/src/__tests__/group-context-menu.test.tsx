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
    onDismiss: vi.fn(),
    ...overrides,
  };
  render(<GroupContextMenu {...props} />);
  return props;
}

describe("GroupContextMenu", () => {
  it("is a thin quick-path menu: no rename/recolor/ungroup/delete items", () => {
    renderMenu();
    expect(screen.queryByTestId("group-ctx-rename")).not.toBeInTheDocument();
    expect(screen.queryByTestId("group-ctx-recolor")).not.toBeInTheDocument();
    expect(screen.queryByTestId("group-ctx-ungroup-all")).not.toBeInTheDocument();
    expect(screen.queryByTestId("group-ctx-delete")).not.toBeInTheDocument();
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
