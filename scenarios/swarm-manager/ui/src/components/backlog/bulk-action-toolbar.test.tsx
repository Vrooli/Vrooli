import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { BulkActionToolbar } from "./bulk-action-toolbar";

describe("BulkActionToolbar", () => {
  const renderToolbar = (selectedCount = 2, onApproveSelected = vi.fn(), onFlagSelected = vi.fn()) =>
    render(
      <BulkActionToolbar
        selectedCount={selectedCount}
        onApproveSelected={onApproveSelected}
        onFlagSelected={onFlagSelected}
        onClearSelection={vi.fn()}
      />,
    );

  it("stays hidden without a selection", () => {
    const { container } = renderToolbar(0);
    expect(container.firstChild).toBeNull();
  });

  it("offers explicit approve and flag review actions", () => {
    const onApprove = vi.fn();
    const onFlag = vi.fn();
    renderToolbar(2, onApprove, onFlag);

    fireEvent.click(screen.getByRole("button", { name: "Approve" }));
    fireEvent.click(screen.getByRole("button", { name: "Flag" }));
    expect(onApprove).toHaveBeenCalledOnce();
    expect(onFlag).toHaveBeenCalledOnce();
    expect(screen.queryByRole("button", { name: /agent/i })).not.toBeInTheDocument();
  });
});
