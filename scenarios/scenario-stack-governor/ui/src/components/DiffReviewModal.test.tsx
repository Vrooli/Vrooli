import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { DiffReviewModal } from "./DiffReviewModal";
import type { FixResult } from "../lib/api";

function makeResult(overrides: Partial<FixResult> = {}): FixResult {
  return {
    scenario_name: "test-scenario",
    rule_id: "makefile-targets",
    fixed: true,
    file_path: "scenarios/test/Makefile",
    changes: [{ type: "add", detail: "added target" }],
    diff: { before: "old content", after: "new content" },
    ...overrides
  };
}

describe("DiffReviewModal", () => {
  it("renders nothing when open is false", () => {
    const { container } = render(
      <DiffReviewModal open={false} results={[]} onApply={vi.fn()} onCancel={vi.fn()} applying={false} />
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders review header when open", () => {
    render(
      <DiffReviewModal open={true} results={[makeResult()]} onApply={vi.fn()} onCancel={vi.fn()} applying={false} />
    );
    expect(screen.getByText("Review Changes")).toBeInTheDocument();
    expect(screen.getByText(/1 file will be modified/)).toBeInTheDocument();
  });

  it("deduplicates results by file_path", () => {
    const results = [
      makeResult({ file_path: "a/Makefile" }),
      makeResult({ file_path: "a/Makefile" }),
      makeResult({ file_path: "b/Makefile" })
    ];
    render(
      <DiffReviewModal open={true} results={results} onApply={vi.fn()} onCancel={vi.fn()} applying={false} />
    );
    expect(screen.getByText(/2 files will be modified/)).toBeInTheDocument();
  });

  it("filters out results without diff or not fixed", () => {
    const results = [
      makeResult({ diff: undefined }),
      makeResult({ fixed: false }),
      makeResult({ file_path: "good/Makefile" })
    ];
    render(
      <DiffReviewModal open={true} results={results} onApply={vi.fn()} onCancel={vi.fn()} applying={false} />
    );
    expect(screen.getByText(/1 file will be modified/)).toBeInTheDocument();
  });

  it("calls onApply when Apply is clicked", () => {
    const onApply = vi.fn();
    render(
      <DiffReviewModal open={true} results={[makeResult()]} onApply={onApply} onCancel={vi.fn()} applying={false} />
    );
    fireEvent.click(screen.getByText("Apply"));
    expect(onApply).toHaveBeenCalledOnce();
  });

  it("calls onCancel when Cancel is clicked", () => {
    const onCancel = vi.fn();
    render(
      <DiffReviewModal open={true} results={[makeResult()]} onApply={vi.fn()} onCancel={onCancel} applying={false} />
    );
    fireEvent.click(screen.getByText("Cancel"));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("shows Applying... and disables buttons when applying", () => {
    render(
      <DiffReviewModal open={true} results={[makeResult()]} onApply={vi.fn()} onCancel={vi.fn()} applying={true} />
    );
    expect(screen.getByText("Applying...")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeDisabled();
    expect(screen.getByText("Applying...")).toBeDisabled();
  });

  it("shows empty state when no valid diffs", () => {
    render(
      <DiffReviewModal open={true} results={[makeResult({ diff: undefined })]} onApply={vi.fn()} onCancel={vi.fn()} applying={false} />
    );
    expect(screen.getByText("No file changes to review.")).toBeInTheDocument();
  });
});
