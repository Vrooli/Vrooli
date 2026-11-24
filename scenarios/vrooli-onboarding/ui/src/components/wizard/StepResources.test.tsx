import { fireEvent, screen } from "../../test-utils";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { vi } from "vitest";
import { DerivedResourceStep } from "./DerivedResourceStep";

const resourcesApi = vi.hoisted(() => ({ fetchDerivedResources: vi.fn() }));
const coreAuthorityApi = vi.hoisted(() => ({ readCoreAuthority: vi.fn().mockResolvedValue({ revision: "r1", authority: { seed: [], trusted_base: [], important: [], members: [] } }) }));
vi.mock("../../api/resources", () => resourcesApi);
vi.mock("../../api/coreAuthority", () => coreAuthorityApi);

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
    expect(await screen.findByRole("checkbox", { name: "Enable postgres" })).toBeDisabled();
    fireEvent.click(screen.getByRole("checkbox", { name: "Enable ollama" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "Enable qdrant" }));
    expect(onToggle).toHaveBeenNthCalledWith(1, "ollama", true);
    expect(onToggle).toHaveBeenNthCalledWith(2, "qdrant", true);
  });

  it("offers monitor-only and auto-start tiers only for selected resources", async () => {
    resourcesApi.fetchDerivedResources.mockResolvedValue({ required: [], optional: [], standalone: [{ name: "qdrant", enabled: false }], resources: [], count: 1 });
    const onTierChange = vi.fn();
    renderWithProviders(<DerivedResourceStep selected={new Set()} operatorState={{ version: "1", updatedAt: "now", resources: { qdrant: { enabled: true } } }} onToggle={vi.fn()} onTierChange={onTierChange} />);
    const important = await screen.findByRole("checkbox", { name: "Monitor important resource qdrant" });
    expect(important).toBeEnabled();
    fireEvent.click(important);
    expect(onTierChange).toHaveBeenLastCalledWith([{ kind: "resource", name: "qdrant" }], [], "r1");
  });
});
