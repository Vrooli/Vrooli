import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { BulkActionBar } from "../../../../components/BulkActionBar/versions/1.0.9/BulkActionBar";

describe("BulkActionBar selection contract", () => {
  it("exposes selection scope, runs the declared action, and clears it", () => {
    const onAction = vi.fn();
    const onClear = vi.fn();
    render(<BulkActionBar selectedCount={2} totalCount={5} actionLabel="Archive" onAction={onAction} onClear={onClear} />);
    expect(screen.getByText("2 items selected")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Clear selection" }));
    fireEvent.click(screen.getByRole("button", { name: "Archive" }));
    expect(onAction).toHaveBeenCalledOnce();
    expect(onClear).toHaveBeenCalledOnce();
  });

  it("reports progress and failure states accessibly", () => {
    render(<BulkActionBar selectedCount={2} status="submitting" progress={{ completed: 1, total: 2, label: "Archiving 1 of 2" }} />);
    expect(screen.getByRole("progressbar", { name: "Archiving 1 of 2" })).toHaveValue(1);
    expect(screen.queryByRole("button", { name: "Clear selection" })).not.toBeInTheDocument();
  });
});
