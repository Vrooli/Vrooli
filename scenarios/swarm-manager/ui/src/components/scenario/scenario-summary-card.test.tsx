import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ScenarioSummaryCard } from "./scenario-summary-card";
import { selectors } from "../../consts/selectors";
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
  it("non-selectable mode: renders a plain summary, no context row", () => {
    render(<ScenarioSummaryCard scenario={makeScenario()} />);
    expect(screen.getByText("API Server")).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.agentSessions.contextRow)).not.toBeInTheDocument();
  });

  it("pick mode: renders the context row and toggles on click", async () => {
    const onToggleSelect = vi.fn();
    render(
      <ScenarioSummaryCard
        scenario={makeScenario()}
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
      <ScenarioSummaryCard
        scenario={makeScenario()}
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
