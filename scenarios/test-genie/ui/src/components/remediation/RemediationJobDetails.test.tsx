import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";
import { RemediationJobDetails } from "./RemediationJobDetails";

describe("RemediationJobDetails", () => {
  it("renders durable selections, attempts, and requirement verification without inferring success", () => {
    render(<RemediationJobDetails job={{ id: "job-1", scenario: "demo", status: "failed", source: { sourceExecutionId: "execution-1", sourceRunId: "run-1", scenario: "demo", createdAt: "2026-01-01T00:00:00Z", phases: [], findings: [], bundles: [], degraded: false }, sourceHash: "source", selectionHash: "selection", selectedFindingIds: ["afid:1"], selectedRequirementIds: ["REQ-1"], attempts: [{ id: "attempt-1", kind: "launch", state: "failed", idempotencyKey: "key", detail: "Agent Manager timed out", createdAt: "2026-01-01T00:00:00Z" }], verification: { delta: { remaining: ["afid:1"] }, requirementDelta: { unverifiable: ["REQ-1"] }, degraded: "requirements evidence unavailable" } }} />);
    expect(screen.getByText("afid:1")).toBeInTheDocument();
    expect(screen.getByText("REQ-1")).toBeInTheDocument();
    expect(screen.getByText(/launch failed/i)).toBeInTheDocument();
    expect(screen.getByText(/Requirements: 0 verified, 0 remaining, 1 unverifiable/i)).toBeInTheDocument();
  });

  it("offers retry only for terminal retryable states", () => {
    const retry = vi.fn();
    const job = { id: "job-1", scenario: "demo", status: "failed" as const, source: { sourceExecutionId: "execution-1", sourceRunId: "run-1", scenario: "demo", createdAt: "2026-01-01T00:00:00Z", phases: [], findings: [], bundles: [], degraded: false }, sourceHash: "source", selectionHash: "selection", selectedFindingIds: [], selectedRequirementIds: [] };
    render(<RemediationJobDetails job={job} onRetry={retry} />);
    fireEvent.click(screen.getByRole("button", { name: "Retry remediation" }));
    expect(retry).toHaveBeenCalledWith("job-1");
  });
});
