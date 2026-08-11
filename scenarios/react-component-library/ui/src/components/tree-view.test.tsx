import { vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { TreeView, type TreeNode } from "./TreeView";

describe("TreeView", () => {
  it("expands default folders when async data arrives after the initial render", () => {
    const { rerender } = render(<TreeView items={[]} label="Library assets" />);

    const folder: TreeNode = {
      id: "components",
      label: "Components",
      defaultExpanded: true,
      children: [{ id: "button", label: "Button" }],
    };
    rerender(<TreeView items={[folder]} label="Library assets" />);

    expect(screen.getByRole("treeitem", { name: "Button" })).toBeVisible();
  });

  it("supports disclosure and selection with the keyboard", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    const folder: TreeNode = {
      id: "components",
      label: "Components",
      children: [{ id: "button", label: "Button" }],
    };
    render(<TreeView items={[folder]} label="Library assets" onSelect={onSelect} />);

    const folderItem = screen.getByRole("treeitem", { name: "Components" });
    await user.click(screen.getByRole("button", { name: "Expand Components" }));
    expect(screen.getByRole("treeitem", { name: "Button" })).toBeVisible();

    folderItem.focus();
    await user.keyboard("{ArrowRight}");
    await user.keyboard("{ArrowDown}{Enter}");
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: "button" }));
  });
});
