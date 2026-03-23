import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ReviewCard } from "./review-card";
import type { ArchiveRequirement, ArchiveTarget } from "../../types";

const makeRequirement = (overrides?: Partial<ArchiveRequirement>): ArchiveRequirement => ({
  id: "REQ-001",
  title: "Test requirement",
  description: "A test description",
  status: "pending",
  category: "foundation",
  prd_ref: "OT-P0-001",
  ...overrides,
});

const makeTarget = (overrides?: Partial<ArchiveTarget>): ArchiveTarget => ({
  id: "OT-P0-001",
  criticality: "P0",
  title: "Test target",
  notes: "Some notes",
  status: "pending",
  linked_requirement_ids: ["REQ-001"],
  ...overrides,
});

describe("ReviewCard", () => {
  it("renders requirement with unreviewed status", () => {
    render(
      <ReviewCard
        item={makeRequirement()}
        itemType="requirement"
        currentStatus="unreviewed"
        onApprove={vi.fn()}
        onFlag={vi.fn()}
        onUnreview={vi.fn()}
        onEdit={vi.fn()}
        onRemove={vi.fn()}
      />,
    );

    expect(screen.getByText("REQ-001")).toBeInTheDocument();
    expect(screen.getByText("Test requirement")).toBeInTheDocument();
    expect(screen.getByText("Unreviewed")).toBeInTheDocument();
  });

  it("renders target with criticality badge", () => {
    render(
      <ReviewCard
        item={makeTarget()}
        itemType="target"
        currentStatus="unreviewed"
        onApprove={vi.fn()}
        onFlag={vi.fn()}
        onUnreview={vi.fn()}
        onEdit={vi.fn()}
        onRemove={vi.fn()}
      />,
    );

    expect(screen.getByText("P0")).toBeInTheDocument();
    expect(screen.getByText("OT-P0-001")).toBeInTheDocument();
    expect(screen.getByText("Test target")).toBeInTheDocument();
  });

  it("calls onApprove when Approve button is clicked", () => {
    const onApprove = vi.fn();
    render(
      <ReviewCard
        item={makeRequirement()}
        itemType="requirement"
        currentStatus="unreviewed"
        onApprove={onApprove}
        onFlag={vi.fn()}
        onUnreview={vi.fn()}
        onEdit={vi.fn()}
        onRemove={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByText("Approve"));
    expect(onApprove).toHaveBeenCalledOnce();
  });

  it("calls onFlag and shows comment field when Flag button is clicked", () => {
    const onFlag = vi.fn();
    render(
      <ReviewCard
        item={makeRequirement()}
        itemType="requirement"
        currentStatus="unreviewed"
        onApprove={vi.fn()}
        onFlag={onFlag}
        onUnreview={vi.fn()}
        onEdit={vi.fn()}
        onRemove={vi.fn()}
      />,
    );

    // Comment field should not be visible initially
    expect(screen.queryByPlaceholderText("Comment (optional)...")).not.toBeInTheDocument();

    fireEvent.click(screen.getByText("Flag"));
    expect(onFlag).toHaveBeenCalledOnce();

    // Comment field should now be visible
    expect(screen.getByPlaceholderText("Comment (optional)...")).toBeInTheDocument();
  });

  it("shows Reset button when status is not unreviewed", () => {
    render(
      <ReviewCard
        item={makeRequirement()}
        itemType="requirement"
        currentStatus="approved"
        onApprove={vi.fn()}
        onFlag={vi.fn()}
        onUnreview={vi.fn()}
        onEdit={vi.fn()}
        onRemove={vi.fn()}
      />,
    );

    expect(screen.getByText("Reset")).toBeInTheDocument();
  });

  it("does not show Reset button when status is unreviewed", () => {
    render(
      <ReviewCard
        item={makeRequirement()}
        itemType="requirement"
        currentStatus="unreviewed"
        onApprove={vi.fn()}
        onFlag={vi.fn()}
        onUnreview={vi.fn()}
        onEdit={vi.fn()}
        onRemove={vi.fn()}
      />,
    );

    expect(screen.queryByText("Reset")).not.toBeInTheDocument();
  });

  it("uses correct border color classes for each status", () => {
    const { container, rerender } = render(
      <ReviewCard
        item={makeRequirement()}
        itemType="requirement"
        currentStatus="approved"
        onApprove={vi.fn()}
        onFlag={vi.fn()}
        onUnreview={vi.fn()}
        onEdit={vi.fn()}
        onRemove={vi.fn()}
      />,
    );

    expect(container.firstChild).toHaveClass("border-emerald-500/40");

    rerender(
      <ReviewCard
        item={makeRequirement()}
        itemType="requirement"
        currentStatus="flagged"
        onApprove={vi.fn()}
        onFlag={vi.fn()}
        onUnreview={vi.fn()}
        onEdit={vi.fn()}
        onRemove={vi.fn()}
      />,
    );

    expect(container.firstChild).toHaveClass("border-amber-500/40");
  });

  it("disables buttons when disabled prop is true", () => {
    render(
      <ReviewCard
        item={makeRequirement()}
        itemType="requirement"
        currentStatus="unreviewed"
        onApprove={vi.fn()}
        onFlag={vi.fn()}
        onUnreview={vi.fn()}
        onEdit={vi.fn()}
        onRemove={vi.fn()}
        disabled
      />,
    );

    const approveBtn = screen.getByText("Approve").closest("button");
    expect(approveBtn).toBeDisabled();
  });

  it("optimistically updates visual state on approve click", () => {
    const { container } = render(
      <ReviewCard
        item={makeRequirement()}
        itemType="requirement"
        currentStatus="unreviewed"
        onApprove={vi.fn()}
        onFlag={vi.fn()}
        onUnreview={vi.fn()}
        onEdit={vi.fn()}
        onRemove={vi.fn()}
      />,
    );

    // Initially unreviewed (slate border)
    expect(container.firstChild).toHaveClass("border-slate-700");

    // Click approve — should optimistically switch to approved (emerald border)
    fireEvent.click(screen.getByText("Approve"));
    expect(container.firstChild).toHaveClass("border-emerald-500/40");
  });

  it("shows error banner when error prop is set", () => {
    render(
      <ReviewCard
        item={makeRequirement()}
        itemType="requirement"
        currentStatus="unreviewed"
        onApprove={vi.fn()}
        onFlag={vi.fn()}
        onUnreview={vi.fn()}
        onEdit={vi.fn()}
        onRemove={vi.fn()}
        error="Network request failed"
      />,
    );

    expect(screen.getByText("Network request failed")).toBeInTheDocument();
  });

  it("disables action buttons when saving", () => {
    render(
      <ReviewCard
        item={makeRequirement()}
        itemType="requirement"
        currentStatus="unreviewed"
        onApprove={vi.fn()}
        onFlag={vi.fn()}
        onUnreview={vi.fn()}
        onEdit={vi.fn()}
        onRemove={vi.fn()}
        saving
      />,
    );

    const approveBtn = screen.getByText("Approve").closest("button");
    expect(approveBtn).toBeDisabled();
  });
});
