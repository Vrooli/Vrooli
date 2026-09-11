import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Trash2 } from "lucide-react";
import { ActionMenu, ActionMenuSheetContent } from "./action-menu";

/**
 * The anchored menu, and its outside-press dismissal, is the desktop (≥ md)
 * presentation; phone widths get a bottom sheet whose backdrop dismisses it.
 */
function mockDesktopViewport() {
  vi.spyOn(window, "matchMedia").mockImplementation(
    (query: string) =>
      ({
        matches: query.includes("min-width"),
        media: query,
        onchange: null,
        addListener: () => undefined,
        removeListener: () => undefined,
        addEventListener: () => undefined,
        removeEventListener: () => undefined,
        dispatchEvent: () => false,
      }) as MediaQueryList,
  );
}

afterEach(() => {
  vi.restoreAllMocks();
});

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
    expect(menu.className).toContain("rcl-context-menu__surface");
    expect(screen.getByRole("menu")).toBeInTheDocument();

    await user.click(screen.getByTestId("delete-action"));
    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId("actions-menu")).not.toBeInTheDocument();
  });

  it("closes when the operator clicks outside the menu", async () => {
    mockDesktopViewport();
    const user = userEvent.setup();

    render(
      <>
        <p>Elsewhere</p>
        <ActionMenu label="Test actions" triggerTestId="actions-trigger" menuTestId="actions-menu" items={[{ label: "Archive", onSelect: vi.fn() }]} />
      </>,
    );

    await user.click(screen.getByTestId("actions-trigger"));
    expect(screen.getByTestId("actions-menu")).toBeInTheDocument();

    await user.click(screen.getByText("Elsewhere"));
    expect(screen.queryByTestId("actions-menu")).not.toBeInTheDocument();
  });

  it("closes, rather than reopens, when the trigger is pressed again", async () => {
    mockDesktopViewport();
    const user = userEvent.setup();

    render(<ActionMenu label="Test actions" triggerTestId="actions-trigger" menuTestId="actions-menu" items={[{ label: "Archive", onSelect: vi.fn() }]} />);

    await user.click(screen.getByTestId("actions-trigger"));
    expect(screen.getByTestId("actions-menu")).toBeInTheDocument();

    await user.click(screen.getByTestId("actions-trigger"));
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
