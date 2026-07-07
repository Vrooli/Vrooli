import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Trash2 } from "lucide-react";
import { ActionMenu, ActionMenuSheetContent } from "./action-menu";

describe("ActionMenu", () => {
  it("opens a standardized menu and runs item actions", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();

    render(
      <ActionMenu
        label="Test actions"
        triggerTestId="actions-trigger"
        menuTestId="actions-menu"
        items={[
          {
            label: "Delete",
            icon: <Trash2 />,
            onSelect,
            destructive: true,
            testId: "delete-action",
          },
        ]}
      />,
    );

    await user.click(screen.getByTestId("actions-trigger"));

    // The menu renders through the shared Popover primitive: the testId
    // container carries the canonical popover surface, and the item list is
    // the role="menu" region.
    const menu = screen.getByTestId("actions-menu");
    expect(menu.className).toContain("bg-slate-900");
    expect(screen.getByRole("menu")).toBeInTheDocument();

    await user.click(screen.getByTestId("delete-action"));
    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId("actions-menu")).not.toBeInTheDocument();
  });

  it("does not run disabled item actions", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();

    render(
      <ActionMenu
        label="Test actions"
        triggerTestId="actions-trigger"
        items={[
          {
            label: "Disabled",
            onSelect,
            disabled: true,
            testId: "disabled-action",
          },
        ]}
      />,
    );

    await user.click(screen.getByTestId("actions-trigger"));
    await user.click(screen.getByTestId("disabled-action"));

    expect(onSelect).not.toHaveBeenCalled();
  });
});

describe("ActionMenuSheetContent", () => {
  it("renders the same button styling for bottom-sheet content", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    const onItemSelected = vi.fn();

    render(
      <ActionMenuSheetContent
        onItemSelected={onItemSelected}
        items={[
          {
            label: "Archive",
            onSelect,
            testId: "archive-action",
          },
        ]}
      />,
    );

    const item = screen.getByTestId("archive-action");
    expect(screen.getByRole("button", { name: "Archive" })).toBe(item);

    await user.click(item);
    expect(onItemSelected).toHaveBeenCalledTimes(1);
    expect(onSelect).toHaveBeenCalledTimes(1);
  });
});
