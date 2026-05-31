import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { InitiativeSummaryCard } from "./initiative-summary-card";
import { selectors } from "../../consts/selectors";
import type { InitiativeWithRollup } from "../../types";

function makeItem(overrides: Partial<InitiativeWithRollup["initiative"]> = {}): InitiativeWithRollup {
  return {
    initiative: {
      name: "test-initiative",
      title: "Test Initiative",
      description: "A test initiative",
      status: "active",
      priority: 0,
      dependsOn: [],
      items: [],
      mode: "item-level",
      acceptanceCriteria: [],
      created: "2026-03-27T00:00:00Z",
      updated: "2026-03-28T00:00:00Z",
      ...overrides,
    },
    rollup: { total: 2, completed: 1, inProgress: 1, failed: 0, pending: 0, archived: 0 },
  };
}

describe("InitiativeSummaryCard", () => {
  it("sidebar mode: opens on click, renders no checkbox", async () => {
    const onOpen = vi.fn();
    render(<InitiativeSummaryCard item={makeItem()} onOpen={onOpen} />);
    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
    await userEvent.click(screen.getByTestId("sidebar-initiative-item"));
    expect(onOpen).toHaveBeenCalledTimes(1);
  });

  it("pick mode: renders the context row and toggles on click", async () => {
    const onToggleSelect = vi.fn();
    render(
      <InitiativeSummaryCard
        item={makeItem()}
        selection={{ selectionMode: true, selected: true, onToggleSelect }}
      />,
    );
    const row = screen.getByTestId(selectors.agentSessions.contextRow);
    expect(row).toHaveAttribute("aria-pressed", "true");
    await userEvent.click(row);
    expect(onToggleSelect).toHaveBeenCalledTimes(1);
  });

  it("pick mode disabled: does not toggle, exposes the reason", async () => {
    const onToggleSelect = vi.fn();
    render(
      <InitiativeSummaryCard
        item={makeItem()}
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
