import { fireEvent, screen, waitFor } from "../../test-utils";
import { createTestQueryClient, renderWithProviders } from "@vrooli/api-base/testing";
import { act } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { StepOperatingMode } from "./StepOperatingMode";

const selectionApi = vi.hoisted(() => ({ fetchScenarios: vi.fn() }));
vi.mock("../../api/selection", () => selectionApi);

describe("operating-mode scenario catalog recovery", () => {
  beforeEach(() => {
    selectionApi.fetchScenarios.mockReset();
  });

  it("keeps saved target overrides read-only during catalog loss and restores them on explicit retry", async () => {
    const cachedCatalog = {
      count: 1,
      scenarios: [{ name: "worker", systemRequired: false, enabled: true, autoRestart: true, resources: [], description: "Worker" }],
    };
    let completeRetry: ((catalog: typeof cachedCatalog) => void) | undefined;
    selectionApi.fetchScenarios
      .mockRejectedValueOnce(new Error("scenario catalog unavailable"))
      .mockImplementationOnce(() => new Promise((resolve) => { completeRetry = resolve; }));
    const onAutoRestart = vi.fn();
    const props = {
      target: "local",
      selected: new Set(["worker"]),
      overrides: { worker: { autoRestart: false } },
      onAutoRestart,
    };

    const queryClient = createTestQueryClient();
    queryClient.setQueryData(["selection-scenarios", "local"], cachedCatalog);
    const view = renderWithProviders(<StepOperatingMode {...props} />, { queryClient });
    expect(await screen.findByTestId("scenario-list-unavailable")).toHaveTextContent("Your saved operating-mode choices are retained");
    expect(screen.getByRole("button", { name: "Retry scenario list" })).toBeEnabled();
    const unavailableToggle = screen.getByRole("switch", { name: "Keep worker running" });
    expect(unavailableToggle).toBeDisabled();
    expect(unavailableToggle).not.toBeChecked();
    expect(screen.getByTestId("override-indicator")).toHaveTextContent("Override saved.");
    expect(selectionApi.fetchScenarios).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole("button", { name: "Retry scenario list" }));
    expect(await screen.findByRole("button", { name: "Retrying scenario list…" })).toBeDisabled();
    expect(unavailableToggle).toBeDisabled();
    await act(async () => { completeRetry?.(cachedCatalog); });
    const toggle = await screen.findByRole("switch", { name: "Keep worker running" });
    await waitFor(() => expect(toggle).not.toBeChecked());
    expect(toggle).toBeEnabled();
    expect(await screen.findByTestId("override-indicator")).toHaveTextContent("Override saved.");
    expect(screen.getByTestId("recommendation-note")).toHaveTextContent("Operator choice differs from the manifest recommendation (always on).");
    expect(onAutoRestart).not.toHaveBeenCalled();
    expect(selectionApi.fetchScenarios).toHaveBeenNthCalledWith(1, "local");
    expect(selectionApi.fetchScenarios).toHaveBeenNthCalledWith(2, "local");

    view.unmount();
  });

  it("does not invent local recommendations when a remote target has no cached catalog", async () => {
    selectionApi.fetchScenarios
      .mockRejectedValueOnce(new Error("remote scenario catalog unavailable"))
      .mockResolvedValueOnce({
        count: 1,
        scenarios: [{ name: "worker", systemRequired: false, enabled: true, autoRestart: false, resources: [], description: "Worker" }],
      });
    const onAutoRestart = vi.fn();
    renderWithProviders(<StepOperatingMode
      target="node-7"
      selected={new Set(["worker"])}
      overrides={{ worker: { autoRestart: true } }}
      onAutoRestart={onAutoRestart}
    />);

    expect(await screen.findByTestId("scenario-list-unavailable")).toHaveTextContent("Your saved operating-mode choices are retained");
    expect(screen.queryByRole("switch", { name: "Keep worker running" })).not.toBeInTheDocument();
    expect(selectionApi.fetchScenarios).toHaveBeenCalledTimes(1);
    expect(selectionApi.fetchScenarios).toHaveBeenCalledWith("node-7");

    fireEvent.click(screen.getByRole("button", { name: "Retry scenario list" }));
    const toggle = await screen.findByRole("switch", { name: "Keep worker running" });
    expect(toggle).toBeChecked();
    expect(toggle).toBeEnabled();
    expect(selectionApi.fetchScenarios).toHaveBeenNthCalledWith(2, "node-7");
    expect(onAutoRestart).not.toHaveBeenCalled();
  });
});
