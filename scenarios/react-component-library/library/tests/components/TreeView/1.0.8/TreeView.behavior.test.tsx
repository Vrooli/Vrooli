import { Fragment } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import {
  TreeView,
  type TreeNode,
} from "@vrooli/react-component-library/TreeView/1";

describe("released TreeView 1.0.0", () => {
  afterEach(() => vi.restoreAllMocks());

  it("renders the empty state and legacy string nodes", () => {
    const warning = vi
      .spyOn(console, "warn")
      .mockImplementation(() => undefined);
    const { rerender } = renderWithProviders(<TreeView label="Assets" />);
    expect(screen.getByRole("status")).toHaveTextContent("Nothing to display.");

    rerender(<TreeView nodes={["Alpha", "Beta"]} label="Assets" />);
    expect(screen.getByRole("treeitem", { name: "Alpha" })).toBeVisible();
    expect(screen.getByRole("treeitem", { name: "Beta" })).toHaveAttribute(
      "aria-level",
      "1",
    );
    expect(warning).toHaveBeenCalledWith(
      expect.stringContaining("object item"),
    );
  });

  it("supports default expansion, custom labels, icons, and selection", async () => {
    const warning = vi
      .spyOn(console, "warn")
      .mockImplementation(() => undefined);
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

    renderWithProviders(
      <TreeView
        items={[folder]}
        defaultSelectedId="file"
        onSelect={onSelect}
      />,
    );
    const folderItem = screen.getByRole("treeitem", {
      name: "Folder metadata",
    });
    expect(screen.getByRole("treeitem", { name: "Readme" })).toBeVisible();
    expect(screen.getByRole("treeitem", { name: "Readme" })).toHaveAttribute(
      "aria-selected",
      "true",
    );

    await user.click(folderItem);
    expect(onSelect).toHaveBeenCalledWith(folder);
    expect(folderItem).toHaveAttribute("aria-selected", "true");
    expect(warning).toHaveBeenCalledWith(
      expect.stringContaining("string label"),
    );
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
    expect(
      screen.queryByRole("treeitem", { name: "Child" }),
    ).not.toBeInTheDocument();

    parentItem.focus();
    await user.keyboard("{ArrowRight}");
    expect(screen.getByRole("treeitem", { name: "Child" })).toBeVisible();
    await user.keyboard("{ArrowLeft}");
    expect(
      screen.queryByRole("treeitem", { name: "Child" }),
    ).not.toBeInTheDocument();
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

  it("expands default folders when async data arrives after the initial render", () => {
    const { rerender } = renderWithProviders(
      <TreeView items={[]} label="Library assets" />,
    );

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
    renderWithProviders(
      <TreeView items={[folder]} label="Library assets" onSelect={onSelect} />,
    );

    const folderItem = screen.getByRole("treeitem", { name: "Components" });
    await user.click(screen.getByRole("button", { name: "Expand Components" }));
    expect(screen.getByRole("treeitem", { name: "Button" })).toBeVisible();

    folderItem.focus();
    await user.keyboard("{ArrowRight}");
    await user.keyboard("{ArrowDown}{Enter}");
    expect(onSelect).toHaveBeenCalledWith(
      expect.objectContaining({ id: "button" }),
    );
  });

  it("supports legacy string nodes, explicit defaults, disabled nodes, and collapse navigation", async () => {
    // Bare string nodes are a documented legacy form, while the released
    // contract declares object nodes. The warning is the contract speaking;
    // assert it instead of treating it as console noise.
    const warning = vi
      .spyOn(console, "warn")
      .mockImplementation(() => undefined);
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
    expect(
      screen.queryByRole("treeitem", { name: "File" }),
    ).not.toBeInTheDocument();

    const { rerender } = renderWithProviders(
      <TreeView nodes={["Alpha", "Beta"]} label="Simple" />,
    );
    expect(screen.getByRole("treeitem", { name: "Alpha" })).toBeInTheDocument();
    rerender(<TreeView items={[]} label="Simple" />);
    expect(screen.getByRole("status")).toHaveTextContent("Nothing to display.");
    expect(warning).toHaveBeenCalledWith(
      expect.stringContaining("object item"),
    );
  });
});
