import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemberItemStrategyPanel } from "./member-item-strategy-panel";
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

describe("MemberItemStrategyPanel", () => {
  it("explains the strategy: items run their own workflows, initiative supplies configuration", () => {
    render(<MemberItemStrategyPanel initiative={initiative} onSwitchClick={() => {}} />);
    expect(screen.getByText(/Each member item runs its own workflow/)).toBeInTheDocument();
    expect(screen.getByText(/The initiative provides strategy configuration/)).toBeInTheDocument();
    expect(
      screen.getByText(/Switch to an operating mode to coordinate work across items/),
    ).toBeInTheDocument();
  });

  it("shows item-count, completed, and in-flight stats from rollup", () => {
    render(<MemberItemStrategyPanel initiative={initiative} rollup={rollup} onSwitchClick={() => {}} />);
    const stats = screen.getByTestId(selectors.initiativeDetails.memberItemStrategyPanel);
    expect(stats).toHaveTextContent("3");
    expect(stats).toHaveTextContent("1");
  });

  it("falls back to zero stats when rollup is missing", () => {
    render(<MemberItemStrategyPanel initiative={initiative} onSwitchClick={() => {}} />);
    const stats = screen.getByTestId(selectors.initiativeDetails.memberItemStrategyPanel);
    // 3 items in initiative + 0 completed + 0 in-flight
    expect(stats).toHaveTextContent("3");
    expect(stats).toHaveTextContent("0");
  });

  it("exposes the switch action as a button with an accessible name", () => {
    render(<MemberItemStrategyPanel initiative={initiative} onSwitchClick={() => {}} />);
    expect(
      screen.getByRole("button", { name: /switch to an operating mode/i }),
    ).toBeInTheDocument();
  });

  it("calls onSwitchClick when the Switch button is clicked", async () => {
    const onSwitchClick = vi.fn();
    render(<MemberItemStrategyPanel initiative={initiative} onSwitchClick={onSwitchClick} />);
    await userEvent.click(
      screen.getByTestId(selectors.initiativeDetails.memberItemStrategyPanelSwitchButton),
    );
    expect(onSwitchClick).toHaveBeenCalledTimes(1);
  });
});
