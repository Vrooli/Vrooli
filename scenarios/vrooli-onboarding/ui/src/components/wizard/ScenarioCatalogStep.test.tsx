import { fireEvent, screen, waitFor } from "../../test-utils";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { vi } from "vitest";
import { ScenarioCatalogStep } from "./ScenarioCatalogStep";

const selectionApi = vi.hoisted(() => ({ fetchScenarios: vi.fn(), fetchClosure: vi.fn() }));
vi.mock("../../api/selection", () => selectionApi);

describe("ScenarioCatalogStep", () => {
  beforeEach(() => {
    selectionApi.fetchScenarios.mockResolvedValue({
      scenarios: [
        { name: "control-plane", systemRequired: true, enabled: true, resources: ["postgres"], description: "Required control plane" },
        { name: "writer", systemRequired: false, enabled: false, resources: ["ollama"], description: "Writes documents" },
      ],
      count: 2,
    });
    selectionApi.fetchClosure.mockResolvedValue({
      scenarios: [{ name: "helper", direct: false }],
      resources: [{ name: "postgres" }, { name: "ollama" }],
    });
  });

  it("renders dependency cascade evidence and protects system-required scenarios", async () => {
    renderWithProviders(<ScenarioCatalogStep selected={new Set()} onToggle={vi.fn()} />);
    expect(await screen.findByTestId("scenario-card-control-plane")).toBeDisabled();
    expect(screen.getByTestId("cascade-note")).toHaveTextContent("helper");
    expect(screen.getByTestId("resource-rollup")).toHaveTextContent("postgres");
  });

  it("filters the rendered catalog without losing the selected callback contract", async () => {
    const onToggle = vi.fn();
    renderWithProviders(<ScenarioCatalogStep selected={new Set()} onToggle={onToggle} />);
    const search = await screen.findByRole("searchbox", { name: "Search scenarios" });
    fireEvent.change(search, { target: { value: "writer" } });
    expect(screen.queryByTestId("scenario-card-control-plane")).not.toBeInTheDocument();
    expect(screen.getByTestId("scenario-card-writer")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("scenario-card-writer"));
    await waitFor(() => expect(onToggle).toHaveBeenCalledWith("writer"));
  });
});
