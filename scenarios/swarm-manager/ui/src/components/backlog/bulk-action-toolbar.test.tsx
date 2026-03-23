import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { BulkActionToolbar } from "./bulk-action-toolbar";

describe("BulkActionToolbar", () => {
  it("does not render when selectedCount is 0", () => {
    const { container } = render(
      <BulkActionToolbar
        selectedCount={0}
        onApproveSelected={vi.fn()}
        onFlagSelected={vi.fn()}
        onSendToAgent={vi.fn()}
        onClearSelection={vi.fn()}
      />,
    );

    expect(container.firstChild).toBeNull();
  });

  it("renders when selectedCount is greater than 0", () => {
    render(
      <BulkActionToolbar
        selectedCount={3}
        onApproveSelected={vi.fn()}
        onFlagSelected={vi.fn()}
        onSendToAgent={vi.fn()}
        onClearSelection={vi.fn()}
      />,
    );

    expect(screen.getByText("3 selected")).toBeInTheDocument();
    expect(screen.getByText("Approve")).toBeInTheDocument();
    expect(screen.getByText("Flag")).toBeInTheDocument();
    expect(screen.getByText("Send to Agent")).toBeInTheDocument();
  });

  it("calls onApproveSelected when Approve is clicked", () => {
    const onApprove = vi.fn();
    render(
      <BulkActionToolbar
        selectedCount={2}
        onApproveSelected={onApprove}
        onFlagSelected={vi.fn()}
        onSendToAgent={vi.fn()}
        onClearSelection={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByText("Approve"));
    expect(onApprove).toHaveBeenCalledOnce();
  });

  it("calls onFlagSelected when Flag is clicked", () => {
    const onFlag = vi.fn();
    render(
      <BulkActionToolbar
        selectedCount={2}
        onApproveSelected={vi.fn()}
        onFlagSelected={onFlag}
        onSendToAgent={vi.fn()}
        onClearSelection={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByText("Flag"));
    expect(onFlag).toHaveBeenCalledOnce();
  });

  it("calls onSendToAgent when Send to Agent is clicked", () => {
    const onSend = vi.fn();
    render(
      <BulkActionToolbar
        selectedCount={2}
        onApproveSelected={vi.fn()}
        onFlagSelected={vi.fn()}
        onSendToAgent={onSend}
        onClearSelection={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByText("Send to Agent"));
    expect(onSend).toHaveBeenCalledOnce();
  });

  it("calls onClearSelection when clear button is clicked", () => {
    const onClear = vi.fn();
    render(
      <BulkActionToolbar
        selectedCount={2}
        onApproveSelected={vi.fn()}
        onFlagSelected={vi.fn()}
        onSendToAgent={vi.fn()}
        onClearSelection={onClear}
      />,
    );

    fireEvent.click(screen.getByTitle("Clear selection"));
    expect(onClear).toHaveBeenCalledOnce();
  });
});
