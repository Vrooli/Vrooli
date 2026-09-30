import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { ScenarioSummaryCard } from "./scenario-summary-card";
import type { Scenario } from "../../types";

function makeScenario(overrides: Partial<Scenario> = {}): Scenario {
  return {
    name: "api-server",
    displayName: "API Server",
    description: "Backend REST API",
    status: "running",
    priority: 1,
    completenessScore: 85,
    isGreenfield: false,
    tags: [],
    ...overrides,
  };
}

describe("ScenarioSummaryCard", () => {
  it("renders the scenario summary as row content without its own button", () => {
    render(<ScenarioSummaryCard scenario={makeScenario()} />);
    expect(screen.getByText("API Server")).toBeInTheDocument();
    expect(screen.getByText("Backend REST API")).toBeInTheDocument();
    expect(screen.getByText("api-server")).toBeInTheDocument();
    // The CollectionList row owns opening and selection.
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("falls back to the scenario name when there is no display name", () => {
    render(<ScenarioSummaryCard scenario={makeScenario({ displayName: "" })} />);
    expect(screen.getAllByText("api-server")).toHaveLength(2);
  });
});
