import { fireEvent, screen } from "../../test-utils";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { vi } from "vitest";
import { DerivedResourceStep } from "./DerivedResourceStep";

const resourcesApi = vi.hoisted(() => ({ fetchDerivedResources: vi.fn() }));
vi.mock("../../api/resources", () => resourcesApi);

describe("DerivedResourceStep", () => {
  it("keeps closure resources locked and emits optional resource choices", async () => {
    resourcesApi.fetchDerivedResources.mockResolvedValue({
      required: [{ name: "postgres", category: "database", enabled: true }],
      optional: [{ name: "ollama", category: "ai", enabled: false }],
      standalone: [{ name: "qdrant", category: "search", enabled: false }],
      resources: [],
      count: 3,
    });
    const onToggle = vi.fn();
    renderWithProviders(<DerivedResourceStep selected={new Set(["writer"])} operatorState={null} onToggle={onToggle} />);
    expect(await screen.findByRole("checkbox", { name: /postgres/i })).toBeDisabled();
    fireEvent.click(screen.getByRole("checkbox", { name: /ollama/i }));
    fireEvent.click(screen.getByRole("checkbox", { name: /qdrant/i }));
    expect(onToggle).toHaveBeenNthCalledWith(1, "ollama", true);
    expect(onToggle).toHaveBeenNthCalledWith(2, "qdrant", true);
  });
});
