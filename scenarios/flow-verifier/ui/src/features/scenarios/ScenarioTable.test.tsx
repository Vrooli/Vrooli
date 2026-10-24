import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import type { ScenarioSummary } from "../../api/scenarios";
import { ScenarioTable } from "./ScenarioTable";

const alpha: ScenarioSummary = {
  id: "alpha",
  displayName: "Alpha",
  description: "First scenario",
  path: "/repo/scenarios/alpha",
  flowCount: 2,
};
const beta: ScenarioSummary = {
  id: "beta",
  displayName: "Beta",
  path: "/repo/scenarios/beta",
  flowCount: 0,
  discoveryError: "permission denied",
};

describe("ScenarioTable", () => {
  afterEach(() => cleanup());

  it("renders one row per scenario with a drill-in link", () => {
    renderWithProviders(
      <ScenarioTable
        scenarios={[alpha, beta]}
        selectedIds={new Set()}
        onToggleOne={() => {}}
        onToggleAll={() => {}}
      />,
    );
    expect(screen.getByTestId("scenario-row-alpha")).toBeInTheDocument();
    expect(screen.getByTestId("scenario-row-beta")).toBeInTheDocument();
    expect(screen.getByTestId("scenario-link-alpha")).toHaveAttribute("href", "/scenarios/alpha");
    expect(screen.getByTestId("scenario-row-error-beta")).toHaveTextContent("permission denied");
  });

  it("calls onToggleOne when a row checkbox is clicked", async () => {
    const onToggleOne = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <ScenarioTable
        scenarios={[alpha]}
        selectedIds={new Set()}
        onToggleOne={onToggleOne}
        onToggleAll={() => {}}
      />,
    );
    await user.click(screen.getByTestId("scenario-select-alpha"));
    expect(onToggleOne).toHaveBeenCalledWith("alpha");
  });

  it("calls onToggleAll when header checkbox is clicked", async () => {
    const onToggleAll = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <ScenarioTable
        scenarios={[alpha, beta]}
        selectedIds={new Set()}
        onToggleOne={() => {}}
        onToggleAll={onToggleAll}
      />,
    );
    await user.click(screen.getByTestId("scenario-toggle-all"));
    expect(onToggleAll).toHaveBeenCalledWith(true);
  });

  it("marks the header checkbox as checked when every row is selected", () => {
    renderWithProviders(
      <ScenarioTable
        scenarios={[alpha, beta]}
        selectedIds={new Set(["alpha", "beta"])}
        onToggleOne={() => {}}
        onToggleAll={() => {}}
      />,
    );
    expect(screen.getByTestId("scenario-toggle-all")).toBeChecked();
  });
});
