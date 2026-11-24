import { fireEvent } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithProviders, screen, waitFor } from "../../test-utils";
import { fetchCoreSet, fetchScenarios } from "../../api/selection";
import { readCoreAuthority } from "../../api/coreAuthority";
import { StepCoreSet } from "./StepCoreSet";
import type { GetCoreSetResponse, ListScenariosResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/selection/selection_pb";

vi.mock("../../api/selection", () => ({
  fetchCoreSet: vi.fn(),
  fetchScenarios: vi.fn(),
}));
vi.mock("../../api/coreAuthority", () => ({ readCoreAuthority: vi.fn().mockResolvedValue({ revision: "r1", authority: { seed: [], trusted_base: [], important: [], members: [] } }) }));

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
    renderWithProviders(<StepCoreSet seed={new Set()} trustedBase={new Set(["trusted"])} selectedScenarios={new Set(["optional"])} onChange={onChange} />);

    expect(await screen.findByText("trusted")).toBeInTheDocument();
    expect(screen.getAllByText("Required for recovery · always included")).toHaveLength(1);
    expect(screen.queryByRole("checkbox", { name: "Supervise trusted" })).not.toBeInTheDocument();
    fireEvent.click(screen.getAllByTestId("core-set-toggle")[1]!);
    await waitFor(() => expect(screen.getByTestId("core-set-confirm")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("core-set-confirm"));
    expect(onChange).toHaveBeenCalledWith(["optional"], [], [], "r1");
  });

  it("offers important monitoring only for selected scenarios and writes its source revision", async () => {
    vi.mocked(fetchScenarios).mockResolvedValueOnce({ scenarios: [{ name: "selected-app", systemRequired: false }, { name: "unselected-app", systemRequired: false }] } as unknown as ListScenariosResponse);
    vi.mocked(fetchCoreSet).mockResolvedValueOnce({ available: true, seed: [], members: [], memberCounts: { scenario: 0, resource: 0 } } as unknown as GetCoreSetResponse);
    vi.mocked(readCoreAuthority).mockResolvedValueOnce({ revision: "tier-r7", authority: { seed: [], trusted_base: [], important: [], members: [] } });
    const onChange = vi.fn();
    renderWithProviders(<StepCoreSet seed={new Set()} trustedBase={new Set()} selectedScenarios={new Set(["selected-app"])} onChange={onChange} />);
    const important = await screen.findByRole("checkbox", { name: "Monitor important scenario selected-app" });
    expect(screen.queryByRole("checkbox", { name: "Monitor important scenario unselected-app" })).not.toBeInTheDocument();
    fireEvent.click(important);
    fireEvent.click(screen.getByTestId("core-set-confirm"));
    expect(onChange).toHaveBeenCalledWith([], [{ kind: "scenario", name: "selected-app" }], [], "tier-r7");
  });


  it("shows the operational recovery plane as included and protected", async () => {
    const required = ["agent-manager", "prompt-manager", "web-console", "vrooli-autoheal"];
    const expectedSeed = [...required].sort();
    vi.mocked(fetchScenarios).mockResolvedValueOnce({
      scenarios: required.map((name) => ({ name, systemRequired: false })),
    } as unknown as ListScenariosResponse);
    vi.mocked(fetchCoreSet).mockResolvedValue({
      available: true,
      seed: expectedSeed,
      members: required.map((name) => ({ kind: "scenario", name, supervisionIntent: "must_start" })),
      memberCounts: { scenario: required.length, resource: 0 },
    } as unknown as GetCoreSetResponse);

    renderWithProviders(<StepCoreSet seed={new Set()} trustedBase={new Set()} onChange={vi.fn()} />);

    await screen.findAllByText("agent-manager");
    const firstChoices = screen.getAllByTestId("core-set-toggle").slice(0, required.length);
    expect(firstChoices.map((card) => card.querySelector(".scenario-choice-card__name")?.textContent)).toEqual([...required].sort());
    for (const name of required) {
      await waitFor(() => expect(screen.getAllByText(name)).toHaveLength(2));
      expect(screen.queryByRole("checkbox", { name: `Supervise ${name}` })).not.toBeInTheDocument();
    }
    expect(screen.getAllByText("Required for recovery · always included")).toHaveLength(required.length);
    expect(fetchCoreSet).toHaveBeenCalledWith(expectedSeed, "local");
  });

  it("reports unconfirmed edits so the wizard can keep Next disabled", async () => {
    vi.mocked(fetchScenarios).mockResolvedValueOnce({ scenarios: [{ name: "selected-app", systemRequired: false }] } as unknown as ListScenariosResponse);
    vi.mocked(fetchCoreSet).mockResolvedValue({ available: true, seed: [], members: [], memberCounts: { scenario: 0, resource: 0 } } as unknown as GetCoreSetResponse);
    const onPendingChangesChange = vi.fn();
    renderWithProviders(<StepCoreSet seed={new Set()} trustedBase={new Set()} selectedScenarios={new Set(["selected-app"])} onPendingChangesChange={onPendingChangesChange} onChange={vi.fn()} />);

    expect(await screen.findByText("selected-app")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("checkbox", { name: "Supervise selected-app" }));
    await waitFor(() => expect(onPendingChangesChange).toHaveBeenLastCalledWith(true));
    expect(screen.getByRole("status")).toHaveTextContent("Confirm your changes below to save them before you continue.");
  });
});
