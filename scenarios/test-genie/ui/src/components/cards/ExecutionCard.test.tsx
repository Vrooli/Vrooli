import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import type { SuiteExecutionResult } from "../../lib/api";
import { ExecutionCard } from "./ExecutionCard";

function execution(overrides: Partial<SuiteExecutionResult> = {}): SuiteExecutionResult {
  return {
    scenarioName: "search-hub",
    startedAt: "2026-11-04T12:00:00Z",
    completedAt: "2026-11-04T12:01:00Z",
    success: true,
    preset: "quick",
    phases: [],
    phaseSummary: {
      total: 3,
      passed: 2,
      failed: 1,
      durationSeconds: 90,
      observationCount: 0,
    },
    ...overrides,
  };
}

describe("ExecutionCard", () => {
  it("shows the completed scenario summary, preset, phase counts, and duration", () => {
    render(<ExecutionCard execution={execution()} />);

    expect(screen.getByRole("heading", { name: "search-hub" })).toBeInTheDocument();
    expect(screen.getByText("completed")).toBeInTheDocument();
    expect(screen.getByText(/quick preset · Completed/)).toBeInTheDocument();
    expect(screen.getByText("2/3 passed")).toBeInTheDocument();
    expect(screen.getByText("1.5m")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("shows a failed status and omits the preset when the execution has none", () => {
    render(
      <ExecutionCard
        execution={execution({ success: false, preset: undefined, completedAt: "invalid" })}
      />,
    );

    expect(screen.getByText("failed")).toBeInTheDocument();
    expect(screen.getByText("Completed —")).toBeInTheDocument();
    expect(screen.queryByText(/preset/)).not.toBeInTheDocument();
  });

  it("runs the supplied rerun and scenario-focus actions", () => {
    const onPrefill = vi.fn();
    const onFocus = vi.fn();
    render(<ExecutionCard execution={execution()} onPrefill={onPrefill} onFocus={onFocus} />);

    fireEvent.click(screen.getByRole("button", { name: "Rerun tests" }));
    fireEvent.click(screen.getByRole("button", { name: "View scenario" }));

    expect(onPrefill).toHaveBeenCalledOnce();
    expect(onFocus).toHaveBeenCalledOnce();
  });
});
