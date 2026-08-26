import { vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { TreeView, type TreeNode } from "@vrooli/react-component-library/TreeView/1.0.0";
import { renderWithProviders } from "../test-utils";

describe("TreeView", () => {
  it("expands default folders when async data arrives after the initial render", () => {
    const { rerender } = renderWithProviders(<TreeView items={[]} label="Library assets" />);

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
    renderWithProviders(<TreeView items={[folder]} label="Library assets" onSelect={onSelect} />);

    const folderItem = screen.getByRole("treeitem", { name: "Components" });
    await user.click(screen.getByRole("button", { name: "Expand Components" }));
    expect(screen.getByRole("treeitem", { name: "Button" })).toBeVisible();

    folderItem.focus();
    await user.keyboard("{ArrowRight}");
    await user.keyboard("{ArrowDown}{Enter}");
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: "button" }));
  });

  it("supports legacy string nodes, explicit defaults, disabled nodes, and collapse navigation", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    const folder: TreeNode = {
      id: "folder",
      label: (
        <>
          <span>Folder</span>
          <small>meta</small>
        </>
      ),
      children: [
        { id: "disabled", label: "Disabled", disabled: true },
        { id: "file", label: "File", icon: <span>custom icon</span> },
      ],
    };
    renderWithProviders(
      <TreeView
        items={[folder]}
        label="Files"
        defaultExpandedIds={["folder"]}
        defaultSelectedId="file"
        onSelect={onSelect}
      />,
    );

    const folderItem = screen.getByRole("treeitem", { name: /Folder meta/ });
    expect(folderItem).toHaveAttribute("tabindex", "-1");
    await user.click(screen.getByRole("treeitem", { name: "Disabled" }));
    expect(onSelect).not.toHaveBeenCalled();
    folderItem.focus();
    await user.keyboard("{ArrowLeft}");
    expect(screen.queryByRole("treeitem", { name: "File" })).not.toBeInTheDocument();

    const { rerender } = renderWithProviders(<TreeView nodes={["Alpha", "Beta"]} label="Simple" />);
    expect(screen.getByRole("treeitem", { name: "Alpha" })).toBeInTheDocument();
    rerender(<TreeView items={[]} label="Simple" />);
    expect(screen.getByRole("status")).toHaveTextContent("Nothing to display.");
  });
});
