import { fireEvent } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithProviders, screen } from "../../test-utils";
import { fetchScenarios } from "../../api/selection";
import { StepOperatingMode } from "./StepOperatingMode";
import type { ListScenariosResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/selection/selection_pb";

vi.mock("../../api/selection", () => ({ fetchScenarios: vi.fn() }));

describe("StepOperatingMode", () => {
  it("groups selected scenarios, preserves system requirements, and reports overrides", async () => {
    vi.mocked(fetchScenarios).mockResolvedValueOnce({ scenarios: [
      { name: "core", systemRequired: true, autoRestart: true },
      { name: "selected-on-demand", systemRequired: false, autoRestart: false },
      { name: "selected-always-on", systemRequired: false, autoRestart: true },
      { name: "not-selected", systemRequired: false, autoRestart: true },
    ] } as unknown as ListScenariosResponse);
    const onAutoRestart = vi.fn();
    renderWithProviders(<StepOperatingMode
      selected={new Set(["selected-on-demand", "selected-always-on"])}
      overrides={{ "selected-always-on": { autoRestart: false }, "selected-on-demand": { autoRestart: true } }}
      onAutoRestart={onAutoRestart}
    />);

    expect(await screen.findByText("core")).toBeInTheDocument();
    expect(screen.getByText("selected-on-demand")).toBeInTheDocument();
    expect(screen.getByText("selected-always-on")).toBeInTheDocument();
    expect(screen.queryByText("not-selected")).not.toBeInTheDocument();
    expect(screen.getAllByTestId("override-indicator")).toHaveLength(2);

    const toggles = screen.getAllByTestId("keep-running-toggle");
    fireEvent.click(toggles[0]!);
    expect(onAutoRestart).toHaveBeenCalled();
  });

  it("shows a disabled loading row before scenarios are available", () => {
    vi.mocked(fetchScenarios).mockReturnValueOnce(new Promise(() => undefined));
    renderWithProviders(<StepOperatingMode selected={new Set()} onAutoRestart={vi.fn()} />);

    expect(screen.getByText("Loading")).toBeInTheDocument();
    expect(screen.getByTestId("keep-running-toggle")).toBeDisabled();
  });
});
