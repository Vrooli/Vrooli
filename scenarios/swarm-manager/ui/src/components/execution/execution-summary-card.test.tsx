import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ExecutionSummaryCard } from "./execution-summary-card";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord } from "../../types";

function makeExecution(overrides: Partial<ExecutionRecord> = {}): ExecutionRecord {
  return {
    executionId: "exec-1",
    backlogKind: "idea",
    backlogName: "test-feature",
    status: "completed",
    mode: "manual",
    createdAt: "2026-03-20T00:00:00Z",
    updatedAt: "2026-03-20T01:00:00Z",
    ...overrides,
  };
}

describe("ExecutionSummaryCard", () => {
  it("sidebar mode: opens on click, renders no checkbox", async () => {
    const onOpen = vi.fn();
    render(<ExecutionSummaryCard item={makeExecution()} onOpen={onOpen} />);
    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
    await userEvent.click(screen.getByTestId("sidebar-execution-item"));
    expect(onOpen).toHaveBeenCalledTimes(1);
  });

  it("pick mode: renders the context row and toggles on click", async () => {
    const onToggleSelect = vi.fn();
    render(
      <ExecutionSummaryCard
        item={makeExecution()}
        selection={{ selectionMode: true, selected: false, onToggleSelect }}
      />,
    );
    const row = screen.getByTestId(selectors.agentSessions.contextRow);
    await userEvent.click(row);
    expect(onToggleSelect).toHaveBeenCalledTimes(1);
  });

  it("pick mode disabled: does not toggle, exposes the reason", async () => {
    const onToggleSelect = vi.fn();
    render(
      <ExecutionSummaryCard
        item={makeExecution()}
        selection={{ selectionMode: true, selected: false, disabled: true, disabledReason: "Cap reached", onToggleSelect }}
      />,
    );
    const row = screen.getByTestId(selectors.agentSessions.contextRow);
    expect(row).toBeDisabled();
    expect(row).toHaveAttribute("title", "Cap reached");
    await userEvent.click(row);
    expect(onToggleSelect).not.toHaveBeenCalled();
  });
});
