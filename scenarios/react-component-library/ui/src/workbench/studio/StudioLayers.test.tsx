import { cleanup, createEvent, fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import type { DesignDocument, DesignFrame } from "../design-types";
import { node } from "../templates/spec";
import { visibleLayerRows } from "./model";
import { StudioLayers } from "./StudioLayers";

const frame: DesignFrame = {
  id: "frame",
  name: "Desktop",
  device: "desktop",
  spec: node("root", "page", {}, [
    node("group", "stack", { label: "Group" }, [node("item", "text", { text: "Item" })]),
    node("target", "card", { label: "Target" }),
  ]),
};
const design: DesignDocument = {
  id: "design",
  name: "Design",
  updatedAt: "2026-01-01T00:00:00Z",
  revision: 1,
  frames: [frame],
  theme: {
    bg: "",
    surface: "",
    surfaceAlt: "",
    ink: "",
    muted: "",
    line: "",
    accent: "",
    accentInk: "",
    accentSoft: "",
    secondary: "",
  },
};

function renderLayers(selectedId: string | null = "item") {
  const actions = {
    onToggleCollapse: vi.fn(),
    onSelect: vi.fn(),
    onMove: vi.fn(),
    onReparent: vi.fn(),
    onDuplicate: vi.fn(),
    onRemove: vi.fn(),
    onAdd: vi.fn(),
    onRename: vi.fn(),
  };
  const render = (collapsed = new Set<string>()) =>
    renderWithProviders(
      <StudioLayers
        design={design}
        frame={frame}
        rows={visibleLayerRows(frame.spec, collapsed)}
        selectedId={selectedId}
        coverage={{ rcl: 1, custom: 2, total: 3, ratio: 1 / 3 }}
        {...actions}
      />,
    );
  render();
  return { ...actions, render };
}

describe("StudioLayers controls", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  it("commits a trimmed name and restores blank or cancelled edits", () => {
    const { onRename } = renderLayers();
    const name = screen.getByRole("textbox", { name: "Design name" });
    fireEvent.change(name, { target: { value: "  Updated design  " } });
    fireEvent.blur(name);
    expect(onRename).toHaveBeenCalledWith("Updated design");

    fireEvent.change(name, { target: { value: "   " } });
    fireEvent.blur(name);
    expect(name).toHaveValue("Design");
    fireEvent.change(name, { target: { value: "Discard me" } });
    fireEvent.keyDown(name, { key: "Escape" });
    expect(name).toHaveValue("Design");
    fireEvent.change(name, { target: { value: "Design" } });
    fireEvent.blur(name);
    expect(onRename).toHaveBeenCalledTimes(1);
  });

  it("selects and collapses layers, and adds a new node under the selected container", () => {
    const { onSelect, onToggleCollapse, onAdd } = renderLayers("group");
    fireEvent.click(screen.getByRole("button", { name: "Collapse Group" }));
    expect(onToggleCollapse).toHaveBeenCalledWith("group");

    fireEvent.click(screen.getByRole("button", { name: /Group stack/ }));
    expect(onSelect).toHaveBeenCalledWith("group");

    fireEvent.change(screen.getByRole("combobox", { name: "Add a layer" }), {
      target: { value: "action" },
    });
    expect(onAdd).toHaveBeenCalledWith(expect.objectContaining({ id: "action" }), "group");
  });

  it("uses the root as the add destination when a non-container layer is selected", () => {
    const { onAdd } = renderLayers("item");
    fireEvent.change(screen.getByRole("combobox", { name: "Add a layer" }), {
      target: { value: "action" },
    });
    expect(onAdd).toHaveBeenCalledWith(expect.objectContaining({ id: "action" }), "root");
  });

  it("runs the selected-layer actions and disables them for the frame root", () => {
    const actions = renderLayers("item");
    fireEvent.click(screen.getByRole("button", { name: "Move up" }));
    fireEvent.click(screen.getByRole("button", { name: "Move down" }));
    fireEvent.click(screen.getByRole("button", { name: "Duplicate layer" }));
    fireEvent.click(screen.getByRole("button", { name: "Remove" }));
    expect(actions.onMove).toHaveBeenNthCalledWith(1, -1);
    expect(actions.onMove).toHaveBeenNthCalledWith(2, 1);
    expect(actions.onDuplicate).toHaveBeenCalledOnce();
    expect(actions.onRemove).toHaveBeenCalledOnce();

    cleanup();
    renderLayers("root");
    for (const label of ["Move up", "Move down", "Duplicate layer", "Remove"]) {
      expect(screen.getByRole("button", { name: label })).toBeDisabled();
    }
  });

  it("shows drag destinations, reparents inside containers, and clears drag state", () => {
    const { onReparent } = renderLayers();
    const controls = screen
      .getAllByRole("treeitem")
      .map((row) => row.querySelector<HTMLButtonElement>(".studio-tree-select"));
    const root = controls[0]!;
    const source = controls[2]!;
    const groupRow = screen.getAllByRole("treeitem")[1]!;
    const group = controls[1]!;
    const dataTransfer = {
      effectAllowed: "",
      dropEffect: "",
      getData: vi.fn(() => "item"),
      setData: vi.fn(),
    };
    const dragOverAt = (clientY: number) => {
      const event = createEvent.dragOver(group);
      Object.defineProperty(event, "dataTransfer", { value: dataTransfer });
      Object.defineProperty(event, "clientY", { value: clientY });
      fireEvent(group, event);
    };
    const dropAt = (clientY: number) => {
      const event = createEvent.drop(group);
      Object.defineProperty(event, "dataTransfer", { value: dataTransfer });
      Object.defineProperty(event, "clientY", { value: clientY });
      fireEvent(group, event);
    };
    vi.spyOn(group, "getBoundingClientRect").mockReturnValue({
      top: 0,
      bottom: 100,
      left: 0,
      right: 100,
      width: 100,
      height: 100,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    });

    const protectedDrag = fireEvent.dragStart(root, { dataTransfer });
    expect(protectedDrag).toBe(false);

    fireEvent.dragStart(source, { dataTransfer });
    expect(dataTransfer.setData).toHaveBeenCalledWith("text/plain", "item");
    expect(screen.getAllByRole("treeitem")[2]).toHaveAttribute("data-drag-source", "true");

    dragOverAt(50);
    expect(groupRow).toHaveAttribute("data-drop-position", "inside");
    fireEvent.dragLeave(group);
    expect(groupRow).not.toHaveAttribute("data-drop-position");

    dragOverAt(10);
    expect(groupRow).toHaveAttribute("data-drop-position", "before");
    dropAt(50);
    expect(onReparent).toHaveBeenCalledWith("item", "group", "inside");
    expect(groupRow).not.toHaveAttribute("data-drop-position");
    fireEvent.dragEnd(source);
    expect(screen.getAllByRole("treeitem")[2]).not.toHaveAttribute("data-drag-source");
  });
});
