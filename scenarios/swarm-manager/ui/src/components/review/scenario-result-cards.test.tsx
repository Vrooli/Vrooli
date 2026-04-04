import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ScenarioResultCards } from "./scenario-result-cards";
import { useDetailSelectionStore } from "../../stores/detail-selection-store";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord, Finalization, ScenarioFinalization } from "../../types";

function makeScenarioFinalization(
  overrides?: Partial<ScenarioFinalization>,
): ScenarioFinalization {
  return {
    scenarioName: "test-scenario",
    changedPaths: ["src/main.ts"],
    restart: { status: "completed", attempts: 1 },
    health: { status: "completed", schemaValid: true },
    review: { status: "completed" },
    ...overrides,
  } as ScenarioFinalization;
}

function makeFinalization(overrides?: Partial<Finalization>): Finalization {
  return {
    eligible: true,
    status: "completed",
    phase: "completed",
    scopeSource: "none",
    aggregateClassification: "ready",
    aggregateSummary: "All checks passed",
    warnings: [],
    affectedScenarios: ["test-scenario"],
    scenarios: [makeScenarioFinalization()],
    ...overrides,
  } as Finalization;
}

function makeExecution(overrides?: Partial<ExecutionRecord>): ExecutionRecord {
  return {
    executionId: "exec-1",
    backlogKind: "task",
    backlogName: "test-item",
    status: "completed",
    mode: "yolo",
    createdAt: new Date().toISOString(),
    ...overrides,
  } as ExecutionRecord;
}

const defaultProps = {
  execution: makeExecution({ finalization: makeFinalization() }),

};

describe("ScenarioResultCards", () => {
  beforeEach(() => {
    useDetailSelectionStore.setState({ selection: null });
  });

  it("returns null when no finalization", () => {
    const { container } = render(
      <ScenarioResultCards
        execution={makeExecution({ finalization: undefined })}

      />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("returns null when finalization has empty scenarios", () => {
    const { container } = render(
      <ScenarioResultCards
        execution={makeExecution({
          finalization: makeFinalization({ scenarios: [] }),
        })}

      />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("returns null when finalization is running", () => {
    const { container } = render(
      <ScenarioResultCards
        execution={makeExecution({
          finalization: makeFinalization({ status: "running", phase: "reviewing" }),
        })}

      />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("returns null when finalization is pending", () => {
    const { container } = render(
      <ScenarioResultCards
        execution={makeExecution({
          finalization: makeFinalization({ status: "pending", phase: "scope_detection" }),
        })}

      />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders scenario cards when finalization is complete with scenarios", () => {
    render(<ScenarioResultCards {...defaultProps} />);
    expect(screen.getByTestId(selectors.review.scenarioResultCards)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.review.scenarioResultCard)).toBeInTheDocument();
    expect(screen.getByText("test-scenario")).toBeInTheDocument();
    expect(screen.getByText("Scenario Results")).toBeInTheDocument();
  });

  it("renders multiple scenario cards", () => {
    const execution = makeExecution({
      finalization: makeFinalization({
        scenarios: [
          makeScenarioFinalization({ scenarioName: "alpha" }),
          makeScenarioFinalization({ scenarioName: "beta" }),
        ],
      }),
    });
    render(<ScenarioResultCards execution={execution} />);
    const cards = screen.getAllByTestId(selectors.review.scenarioResultCard);
    expect(cards).toHaveLength(2);
    expect(screen.getByText("alpha")).toBeInTheDocument();
    expect(screen.getByText("beta")).toBeInTheDocument();
  });

  it("shows classification badge for completed review with classification", () => {
    const execution = makeExecution({
      finalization: makeFinalization({
        scenarios: [
          makeScenarioFinalization({
            scenarioName: "my-scenario",
            review: {
              status: "completed",
              result: {
                jobId: "j1",
                classification: "ready",
                dimensions: [],
                summary: "Looks good",
                reviewedAt: new Date().toISOString(),
              },
            },
          }),
        ],
      }),
    });
    render(<ScenarioResultCards execution={execution} />);
    expect(screen.getByText("Ready")).toBeInTheDocument();
  });

  it("shows needs_work classification badge", () => {
    const execution = makeExecution({
      finalization: makeFinalization({
        scenarios: [
          makeScenarioFinalization({
            review: {
              status: "completed",
              result: {
                jobId: "j1",
                classification: "needs_work",
                dimensions: [],
                summary: "Needs fixes",
                reviewedAt: new Date().toISOString(),
              },
            },
          }),
        ],
      }),
    });
    render(<ScenarioResultCards execution={execution} />);
    expect(screen.getByText("Needs work")).toBeInTheDocument();
  });

  it("shows restart and health status indicators", () => {
    render(<ScenarioResultCards {...defaultProps} />);
    expect(screen.getByText("restart")).toBeInTheDocument();
    expect(screen.getByText("health")).toBeInTheDocument();
  });

  it("shows failed restart status", () => {
    const execution = makeExecution({
      finalization: makeFinalization({
        scenarios: [
          makeScenarioFinalization({
            restart: { status: "failed", attempts: 2, lastError: "timeout" },
          }),
        ],
      }),
    });
    render(<ScenarioResultCards execution={execution} />);
    expect(screen.getByText("restart")).toBeInTheDocument();
  });

  it("clicking scenario name navigates to scenario detail", () => {
    render(
      <ScenarioResultCards
        execution={makeExecution({
          finalization: makeFinalization({
            scenarios: [makeScenarioFinalization({ scenarioName: "click-me" })],
          }),
        })}
      />,
    );
    fireEvent.click(screen.getByText("click-me"));
    const selection = useDetailSelectionStore.getState().selection;
    expect(selection).toMatchObject({ entityType: "scenario", name: "click-me" });
  });

  it("renders cards when finalization status is failed (terminal)", () => {
    const execution = makeExecution({
      finalization: makeFinalization({ status: "failed", phase: "failed" }),
    });
    render(<ScenarioResultCards execution={execution} />);
    expect(screen.getByTestId(selectors.review.scenarioResultCards)).toBeInTheDocument();
  });

  it("shows skipped review status indicator", () => {
    const execution = makeExecution({
      finalization: makeFinalization({
        scenarios: [
          makeScenarioFinalization({
            review: { status: "skipped", skipReason: "No GCT configured" },
          }),
        ],
      }),
    });
    render(<ScenarioResultCards execution={execution} />);
    // When review is skipped, the component renders a StatusIndicator with "review" label
    expect(screen.getByText("review")).toBeInTheDocument();
  });
});
