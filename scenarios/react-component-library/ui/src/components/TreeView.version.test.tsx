import { Fragment } from "react";
import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { TreeView, type TreeNode } from "@vrooli/react-component-library/TreeView/1";

describe("released TreeView 1.0.0", () => {
  it("renders the empty state and legacy string nodes", () => {
    const { rerender } = renderWithProviders(<TreeView label="Assets" />);
    expect(screen.getByRole("status")).toHaveTextContent("Nothing to display.");

    rerender(<TreeView nodes={["Alpha", "Beta"]} label="Assets" />);
    expect(screen.getByRole("treeitem", { name: "Alpha" })).toBeVisible();
    expect(screen.getByRole("treeitem", { name: "Beta" })).toHaveAttribute("aria-level", "1");
  });

  it("supports default expansion, custom labels, icons, and selection", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    const folder: TreeNode = {
      id: "folder",
      label: (
        <Fragment>
          <span>Folder</span>
          <small>metadata</small>
        </Fragment>
      ),
      ariaLabel: "Project folder",
      defaultExpanded: true,
      icon: <span>folder icon</span>,
      children: [{ id: "file", label: "Readme", icon: <span>file icon</span> }],
    };

    renderWithProviders(<TreeView items={[folder]} defaultSelectedId="file" onSelect={onSelect} />);
    const folderItem = screen.getByRole("treeitem", { name: "Project folder" });
    expect(screen.getByRole("treeitem", { name: "Readme" })).toBeVisible();
    expect(screen.getByRole("treeitem", { name: "Readme" })).toHaveAttribute(
      "aria-selected",
      "true",
    );

    await user.click(folderItem);
    expect(onSelect).toHaveBeenCalledWith(folder);
    expect(folderItem).toHaveAttribute("aria-selected", "true");
  });

  it("opens and closes folders through disclosure and arrow keys", async () => {
    const user = userEvent.setup();
    const parent: TreeNode = {
      id: "parent",
      label: "Parent",
      children: [{ id: "child", label: "Child" }],
    };
    renderWithProviders(<TreeView items={[parent]} />);

    const parentItem = screen.getByRole("treeitem", { name: "Parent" });
    await user.click(screen.getByRole("button", { name: "Expand Parent" }));
    expect(screen.getByRole("treeitem", { name: "Child" })).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Collapse Parent" }));
    expect(screen.queryByRole("treeitem", { name: "Child" })).not.toBeInTheDocument();

    parentItem.focus();
    await user.keyboard("{ArrowRight}");
    expect(screen.getByRole("treeitem", { name: "Child" })).toBeVisible();
    await user.keyboard("{ArrowLeft}");
    expect(screen.queryByRole("treeitem", { name: "Child" })).not.toBeInTheDocument();
  });

  it("supports keyboard traversal, activation, and disabled nodes", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    const nodes: TreeNode[] = [
      { id: "first", label: "First" },
      { id: "disabled", label: "Disabled", disabled: true },
      { id: "last", label: "Last" },
    ];
    renderWithProviders(<TreeView items={nodes} onSelect={onSelect} />);

    const first = screen.getByRole("treeitem", { name: "First" });
    first.focus();
    await user.keyboard("{ArrowDown}{ArrowDown}{ArrowUp}{Home}{End}{Enter}");
    expect(onSelect).toHaveBeenCalledWith(nodes[2]);
    await user.click(screen.getByRole("treeitem", { name: "Disabled" }));
    expect(onSelect).toHaveBeenCalledTimes(1);
  });
});
