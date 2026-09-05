import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { ExecutionCard } from "./ExecutionCard";
import { selectors } from "@constants/selectors";
import { renderWithProviders } from "@/test-utils";

describe("ExecutionCard", () => {
  it("exposes stable workflow identity for deterministic selection", () => {
    renderWithProviders(
      <ExecutionCard
        testId={selectors.executions.list.item}
        execution={{
          id: "execution-1",
          workflowId: "workflow-1",
          workflowName: "Seeded workflow",
          status: "completed",
          startedAt: new Date("2026-01-01T00:00:00Z"),
        }}
      />,
    );

    const card = screen.getByTestId(selectors.executions.list.item);
    expect(card).toHaveAttribute("data-workflow-id", "workflow-1");
    expect(card).toHaveAttribute("data-workflow-name", "Seeded workflow");
  });
});
