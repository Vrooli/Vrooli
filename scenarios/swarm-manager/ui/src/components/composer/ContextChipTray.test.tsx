import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ContextChipTray, type ComposerContextChip } from "./ContextChipTray";

const CHIP: ComposerContextChip = {
  type: "backlog_item",
  ref: "fix/broken-thing",
  title: "Broken thing",
  subtitle: "fix · ready",
  nodeId: "backlog-item/fix/broken-thing",
};

describe("ContextChipTray", () => {
  it("opens a detail view with contents and available actions when a chip is clicked", async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();
    const onRemove = vi.fn();

    render(
      <ContextChipTray items={[CHIP]} onRemove={onRemove} onOpen={onOpen} testId="tray" />,
    );

    // The chip label is a button (not just a title-attribute tooltip).
    await user.click(screen.getByTestId("tray-chip"));

    const detail = screen.getByTestId("tray-detail");
    // Contents: title and summary are rendered in the detail view.
    expect(detail).toHaveTextContent("Broken thing");
    expect(detail).toHaveTextContent("fix · ready");

    // Available actions: Open navigates via the resolved detail path.
    await user.click(screen.getByTestId("tray-detail-open"));
    expect(onOpen).toHaveBeenCalledWith("/backlog/fix/broken-thing");
  });

  it("keeps the remove control working", async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();

    render(<ContextChipTray items={[CHIP]} onRemove={onRemove} testId="tray" />);

    await user.click(screen.getByRole("button", { name: "Remove Broken thing" }));
    expect(onRemove).toHaveBeenCalledWith("backlog_item", "fix/broken-thing");
  });

  it("can opt out of the composer height constraint for sent-message context", () => {
    const { container } = render(<ContextChipTray items={[CHIP]} constrainHeight={false} testId="tray" />);

    expect(container.firstChild).not.toHaveClass("max-h-20");
    expect(container.firstChild).not.toHaveClass("overflow-y-auto");
  });
});
