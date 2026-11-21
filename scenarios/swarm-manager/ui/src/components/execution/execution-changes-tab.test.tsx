import { describe, it, expect } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { renderWithProviders as renderWithCanonicalProviders } from "../../test-utils/renderWithProviders";
import { ExecutionChangesTab } from "./execution-changes-tab";
import type { Finalization } from "../../types";
import { selectors } from "../../consts/selectors";

const makeFinalization = (overrides?: Partial<Finalization>): Finalization => ({
  eligible: true,
  status: "completed",
  phase: "completed",
  scopeSource: "sandbox_diff",
  warnings: [],
  affectedScenarios: ["app-a"],
  aggregateClassification: "ready",
  scenarios: [
    {
      scenarioName: "app-a",
      changedPaths: ["src/main.ts", "src/utils.ts"],
      restart: { status: "completed", attempts: 1 },
      health: { status: "completed", scenarioStatus: "running", healthStatus: "healthy", schemaValid: true },
      review: { status: "completed" },
    },
  ],
  ...overrides,
});

describe("ExecutionChangesTab", () => {
  it("shows empty state when no finalization", () => {
    renderWithCanonicalProviders(<ExecutionChangesTab finalization={undefined} isActive={false} />);
    expect(screen.getByTestId(selectors.executionDetails.changesEmpty)).toBeInTheDocument();
    expect(screen.getByText(/No sandbox changes available/)).toBeInTheDocument();
  });

  it("shows pending message when execution is active", () => {
    renderWithCanonicalProviders(<ExecutionChangesTab finalization={undefined} isActive={true} />);
    expect(screen.getByText(/Changes will be available after/)).toBeInTheDocument();
  });

  it("shows empty state when finalization has no sandbox scope", () => {
    const fin = makeFinalization({
      scopeSource: "acceptance_allow",
      scenarios: [
        {
          scenarioName: "app-a",
          changedPaths: [],
          restart: { status: "completed", attempts: 1 },
          health: { status: "completed", scenarioStatus: "running", healthStatus: "healthy", schemaValid: true },
          review: { status: "completed" },
        },
      ],
    });
    renderWithCanonicalProviders(<ExecutionChangesTab finalization={fin} isActive={false} />);
    expect(screen.getByTestId(selectors.executionDetails.changesEmpty)).toBeInTheDocument();
  });

  it("renders changed files grouped by scenario", () => {
    renderWithCanonicalProviders(<ExecutionChangesTab finalization={makeFinalization()} isActive={false} />);
    expect(screen.getByTestId(selectors.executionDetails.changesFileList)).toBeInTheDocument();
    expect(screen.getByText("app-a")).toBeInTheDocument();
    expect(screen.getByText("src/main.ts")).toBeInTheDocument();
    expect(screen.getByText("src/utils.ts")).toBeInTheDocument();
    expect(screen.getByText("2 files")).toBeInTheDocument();
  });

  it("shows file count for single file", () => {
    const fin = makeFinalization({
      scenarios: [
        {
          scenarioName: "app-a",
          changedPaths: ["src/main.ts"],
          restart: { status: "completed", attempts: 1 },
          health: { status: "completed", scenarioStatus: "running", healthStatus: "healthy", schemaValid: true },
          review: { status: "completed" },
        },
      ],
    });
    renderWithCanonicalProviders(<ExecutionChangesTab finalization={fin} isActive={false} />);
    expect(screen.getByText("1 file")).toBeInTheDocument();
  });

  it("collapses and expands scenario file list", () => {
    renderWithCanonicalProviders(<ExecutionChangesTab finalization={makeFinalization()} isActive={false} />);

    // Files visible by default (expanded)
    expect(screen.getByText("src/main.ts")).toBeInTheDocument();

    // Click to collapse
    fireEvent.click(screen.getByText("app-a"));
    expect(screen.queryByText("src/main.ts")).not.toBeInTheDocument();

    // Click to expand again
    fireEvent.click(screen.getByText("app-a"));
    expect(screen.getByText("src/main.ts")).toBeInTheDocument();
  });

  it("renders multiple scenarios", () => {
    const fin = makeFinalization({
      scenarios: [
        {
          scenarioName: "app-a",
          changedPaths: ["src/a.ts"],
          restart: { status: "completed", attempts: 1 },
          health: { status: "completed", scenarioStatus: "running", healthStatus: "healthy", schemaValid: true },
          review: { status: "completed" },
        },
        {
          scenarioName: "app-b",
          changedPaths: ["src/b.ts"],
          restart: { status: "completed", attempts: 1 },
          health: { status: "completed", scenarioStatus: "running", healthStatus: "healthy", schemaValid: true },
          review: { status: "completed" },
        },
      ],
    });
    renderWithCanonicalProviders(<ExecutionChangesTab finalization={fin} isActive={false} />);
    expect(screen.getByText("app-a")).toBeInTheDocument();
    expect(screen.getByText("app-b")).toBeInTheDocument();
  });
});
