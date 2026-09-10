import { fireEvent } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithProviders, screen, waitFor } from "../../test-utils";
import { fetchCoreSet, fetchScenarios } from "../../api/selection";
import { StepCoreSet } from "./StepCoreSet";
import type { GetCoreSetResponse, ListScenariosResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/selection/selection_pb";

vi.mock("../../api/selection", () => ({
  fetchCoreSet: vi.fn(),
  fetchScenarios: vi.fn(),
}));

describe("StepCoreSet", () => {
  it("shows loading and request failures for the scenario and preview queries", async () => {
    vi.mocked(fetchScenarios).mockRejectedValueOnce(new Error("scenario request failed"));
    vi.mocked(fetchCoreSet).mockRejectedValueOnce(new Error("preview request failed"));
    renderWithProviders(<StepCoreSet seed={new Set()} trustedBase={new Set()} onChange={vi.fn()} />);

    expect(await screen.findByText("Core-set choices could not be loaded.")).toBeInTheDocument();
    expect(screen.getAllByRole("alert")).toHaveLength(2);
  });

  it("renders an unavailable preview and keeps the confirm action disabled", async () => {
    vi.mocked(fetchScenarios).mockResolvedValueOnce({ scenarios: [] } as unknown as ListScenariosResponse);
    vi.mocked(fetchCoreSet).mockResolvedValueOnce({ available: false, error: "Capacity check required", seed: ["alpha"] } as unknown as GetCoreSetResponse);
    renderWithProviders(<StepCoreSet seed={new Set(["alpha"])} trustedBase={new Set()} onChange={vi.fn()} />);

    expect(await screen.findByText(/Capacity check required/)).toBeInTheDocument();
    expect(screen.getByTestId("core-set-confirm")).toBeDisabled();
  });

  it("allows optional scenario changes while protecting trusted scenarios", async () => {
    vi.mocked(fetchScenarios).mockResolvedValueOnce({ scenarios: [
      { name: "trusted", systemRequired: true },
      { name: "optional", systemRequired: false },
    ] } as unknown as ListScenariosResponse);
    vi.mocked(fetchCoreSet).mockResolvedValue({ available: true, seed: [], members: [{ kind: "scenario", name: "optional", supervisionIntent: "on-demand" }], memberCounts: { scenario: 1, resource: 0 } } as unknown as GetCoreSetResponse);
    const onChange = vi.fn();
    renderWithProviders(<StepCoreSet seed={new Set()} trustedBase={new Set(["trusted"])} onChange={onChange} />);

    expect(await screen.findByText("trusted")).toBeInTheDocument();
    expect(screen.getAllByText("Trusted-base member; cannot be removed")).toHaveLength(2);
    fireEvent.click(screen.getAllByTestId("core-set-toggle")[1]!);
    await waitFor(() => expect(screen.getByTestId("core-set-confirm")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("core-set-confirm"));
    expect(onChange).toHaveBeenCalledWith(["optional"]);
  });
});
