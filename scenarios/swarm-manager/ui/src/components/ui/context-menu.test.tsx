import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextMenu } from "./context-menu";
import { useContextMenu } from "./use-context-menu";

function Harness({ onSelect }: { onSelect: () => void }) {
  const menu = useContextMenu();
  return (
    <>
      <div {...menu.triggerProps} data-testid="row">
        row
      </div>
      <ContextMenu
        origin={menu.origin}
        onClose={menu.close}
        items={[{ label: "Set as goal", onSelect, testId: "ctx-goal" }]}
        testId="ctx-menu"
      />
    </>
  );
}

describe("ContextMenu", () => {
  it("opens on right-click and runs the selected item, then closes", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(<Harness onSelect={onSelect} />);

    expect(screen.queryByTestId("ctx-menu")).not.toBeInTheDocument();

    fireEvent.contextMenu(screen.getByTestId("row"), { clientX: 20, clientY: 30 });

    const menu = screen.getByTestId("ctx-menu");
    // Shares the canonical Popover surface used by every other menu.
    expect(menu.className).toContain("bg-slate-900");
    expect(screen.getByRole("menu")).toBeInTheDocument();

    await user.click(screen.getByTestId("ctx-goal"));
    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId("ctx-menu")).not.toBeInTheDocument();
  });

  it("renders nothing when there are no items", () => {
    render(<ContextMenu origin={{ x: 0, y: 0 }} onClose={() => {}} items={[]} testId="empty-menu" />);
    expect(screen.queryByTestId("empty-menu")).not.toBeInTheDocument();
  });
});
