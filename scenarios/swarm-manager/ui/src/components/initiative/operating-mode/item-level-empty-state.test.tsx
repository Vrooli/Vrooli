import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ItemLevelEmptyState } from "./item-level-empty-state";
import { selectors } from "../../../consts/selectors";
import type { Initiative, InitiativeRollup } from "../../../types";

const initiative: Initiative = {
  name: "init-a",
  title: "Initiative A",
  description: "",
  status: "active",
  mode: "item-level",
  acceptanceCriteria: [],
  priority: 0,
  dependsOn: [],
  items: ["execute/a", "execute/b", "fix/typo"],
  created: "2026-04-30T00:00:00Z",
  updated: "2026-04-30T00:00:00Z",
};

const rollup: InitiativeRollup = {
  total: 3,
  pending: 1,
  inProgress: 1,
  completed: 1,
  failed: 0,
  blocked: 0,
  archived: 0,
} as InitiativeRollup;

describe("ItemLevelEmptyState", () => {
  it("renders the three-bullet explainer", () => {
    render(<ItemLevelEmptyState initiative={initiative} onSwitchClick={() => {}} />);
    expect(screen.getByText(/Each backlog item runs through the existing execution flow/)).toBeInTheDocument();
    expect(screen.getByText(/Agents only read initiative state/)).toBeInTheDocument();
    expect(screen.getByText(/Switch to a phase-capable mode/)).toBeInTheDocument();
  });

  it("shows item-count, completed, and in-flight stats from rollup", () => {
    render(<ItemLevelEmptyState initiative={initiative} rollup={rollup} onSwitchClick={() => {}} />);
    const stats = screen.getByTestId(selectors.initiativeDetails.itemLevelEmptyState);
    expect(stats).toHaveTextContent("3");
    expect(stats).toHaveTextContent("1");
  });

  it("falls back to zero stats when rollup is missing", () => {
    render(<ItemLevelEmptyState initiative={initiative} onSwitchClick={() => {}} />);
    const stats = screen.getByTestId(selectors.initiativeDetails.itemLevelEmptyState);
    // 3 items in initiative + 0 completed + 0 in-flight
    expect(stats).toHaveTextContent("3");
    expect(stats).toHaveTextContent("0");
  });

  it("calls onSwitchClick when the Switch button is clicked", async () => {
    const onSwitchClick = vi.fn();
    render(<ItemLevelEmptyState initiative={initiative} onSwitchClick={onSwitchClick} />);
    await userEvent.click(
      screen.getByTestId(selectors.initiativeDetails.itemLevelEmptyStateSwitchButton),
    );
    expect(onSwitchClick).toHaveBeenCalledTimes(1);
  });
});
