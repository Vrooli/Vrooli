import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
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

function renderLayers(onReparent = vi.fn()) {
  renderWithProviders(
    <StudioLayers
      design={design}
      frame={frame}
      rows={visibleLayerRows(frame.spec, new Set())}
      selectedId="item"
      coverage={{ rcl: 0, custom: 3, total: 3, ratio: 0 }}
      onToggleCollapse={vi.fn()}
      onSelect={vi.fn()}
      onMove={vi.fn()}
      onReparent={onReparent}
      onDuplicate={vi.fn()}
      onRemove={vi.fn()}
      onAdd={vi.fn()}
      onRename={vi.fn()}
    />,
  );
  return onReparent;
}

describe("StudioLayers drag and drop", () => {
  it("supports keyboard reparenting and protects the frame root", () => {
    const move = renderLayers();
    const item = screen.getAllByRole("treeitem")[2]?.querySelector(".studio-tree-select");
    if (!item) throw new Error("Layer selection button was not rendered.");
    fireEvent.keyDown(item, { key: " " });
    fireEvent.keyDown(item, { key: "ArrowDown" });
    fireEvent.keyDown(item, { key: "ArrowRight" });
    fireEvent.keyDown(item, { key: "Enter" });
    expect(move).toHaveBeenCalledWith("item", "target", "inside");
    const rootRow = screen.getAllByRole("treeitem")[0];
    expect(rootRow?.querySelector(".studio-tree-select")).toHaveAttribute("draggable", "false");
  });
});
