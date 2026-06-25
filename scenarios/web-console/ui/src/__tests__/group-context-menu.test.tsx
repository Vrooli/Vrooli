import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import GroupContextMenu from "../components/GroupContextMenu";
import { HEADER_COLORS } from "../consts/config";
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
    onRename: vi.fn(),
    onRecolor: vi.fn(),
    onNewSession: vi.fn(),
    onToggleCollapse: vi.fn(),
    onUngroupAll: vi.fn(),
    onDelete: vi.fn(),
    onDismiss: vi.fn(),
    ...overrides,
  };
  render(<GroupContextMenu {...props} />);
  return props;
}

describe("GroupContextMenu", () => {
  it("fires rename and dismisses", () => {
    const props = renderMenu();
    fireEvent.click(screen.getByTestId("group-ctx-rename"));
    expect(props.onRename).toHaveBeenCalled();
    expect(props.onDismiss).toHaveBeenCalled();
  });

  it("recolor opens the palette and applies a color", () => {
    const props = renderMenu();
    expect(screen.queryByTestId("group-ctx-palette")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("group-ctx-recolor"));
    expect(screen.getByTestId("group-ctx-palette")).toBeInTheDocument();
    const color = HEADER_COLORS[0];
    fireEvent.click(screen.getByTestId(`group-ctx-color-${color}`));
    expect(props.onRecolor).toHaveBeenCalledWith(color);
    expect(props.onDismiss).toHaveBeenCalled();
  });

  it("fires ungroup-all and collapse toggle", () => {
    const props = renderMenu();
    fireEvent.click(screen.getByTestId("group-ctx-ungroup-all"));
    expect(props.onUngroupAll).toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("group-ctx-toggle-collapse"));
    expect(props.onToggleCollapse).toHaveBeenCalled();
  });

  it("fires new-session-in-group and dismisses", () => {
    const props = renderMenu();
    fireEvent.click(screen.getByTestId("group-ctx-new-session"));
    expect(props.onNewSession).toHaveBeenCalled();
    expect(props.onDismiss).toHaveBeenCalled();
  });

  it("fires delete", () => {
    const props = renderMenu();
    fireEvent.click(screen.getByTestId("group-ctx-delete"));
    expect(props.onDelete).toHaveBeenCalled();
  });
});
