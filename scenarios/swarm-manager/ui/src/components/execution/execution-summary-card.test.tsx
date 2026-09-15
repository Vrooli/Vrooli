import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { ExecutionSummaryCard } from "./execution-summary-card";
import type { ExecutionRecord } from "../../types";

function makeExecution(overrides: Partial<ExecutionRecord> = {}): ExecutionRecord {
  return {
    executionId: "exec-1",
    backlogKind: "idea",
    backlogName: "test-feature",
    status: "needs_review",
    mode: "manual",
    createdAt: "2026-03-20T00:00:00Z",
    updatedAt: "2026-03-20T01:00:00Z",
    ...overrides,
  };
}

describe("ExecutionSummaryCard", () => {
  it("renders the execution summary as row content without its own button", () => {
    render(<ExecutionSummaryCard item={makeExecution()} />);
    expect(screen.getByText("test-feature")).toBeInTheDocument();
    expect(screen.getByText("needs review")).toBeInTheDocument();
    expect(screen.getByText("Manual")).toBeInTheDocument();
    // The CollectionList row owns opening; nested buttons would swallow its click.
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
