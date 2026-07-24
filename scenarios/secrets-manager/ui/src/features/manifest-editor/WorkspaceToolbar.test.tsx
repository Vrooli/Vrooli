import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { WorkspaceToolbar } from "./WorkspaceToolbar";

describe("WorkspaceToolbar", () => {
  afterEach(cleanup);

  it("routes manifest selection, filter, save, export, and copy-from-tier actions", async () => {
    const onScenarioChange = vi.fn();
    const onTierChange = vi.fn();
    const onSearchChange = vi.fn();
    const onFilterChange = vi.fn();
    const onSaveAll = vi.fn();
    const onExport = vi.fn();
    const onCopyFromTier = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(
      <WorkspaceToolbar
        scenario="secrets-manager"
        tier="tier-2-desktop"
        availableTiers={[{ value: "tier-1-local", label: "Tier 1" }, { value: "tier-2-desktop", label: "Tier 2" }]}
        availableScenarios={[{ name: "secrets-manager", display_name: "Secrets Manager" }, { name: "api-gateway" }]}
        searchQuery=""
        filter="all"
        hasPendingChanges
        isSaving={false}
        onScenarioChange={onScenarioChange}
        onTierChange={onTierChange}
        onSearchChange={onSearchChange}
        onFilterChange={onFilterChange}
        onSaveAll={onSaveAll}
        onExport={onExport}
        onCopyFromTier={onCopyFromTier}
      />
    );

    const [scenarioSelect, tierSelect, filterSelect] = screen.getAllByRole("combobox");
    if (!scenarioSelect || !tierSelect || !filterSelect) throw new Error("Expected workspace selectors");
    fireEvent.change(scenarioSelect, { target: { value: "api-gateway" } });
    fireEvent.change(tierSelect, { target: { value: "tier-1-local" } });
    fireEvent.change(screen.getByPlaceholderText("Search secrets..."), { target: { value: "vault" } });
    fireEvent.change(filterSelect, { target: { value: "blocking" } });
    expect(onScenarioChange).toHaveBeenCalledWith("api-gateway");
    expect(onTierChange).toHaveBeenCalledWith("tier-1-local");
    expect(onSearchChange).toHaveBeenCalledWith("vault");
    expect(onFilterChange).toHaveBeenCalledWith("blocking");
    fireEvent.click(screen.getByText("Save All"));
    fireEvent.click(screen.getByText("Export"));
    expect(onSaveAll).toHaveBeenCalledOnce();
    expect(onExport).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByText("Copy from tier"));
    fireEvent.click(screen.getByRole("button", { name: "Tier 1" }));
    await waitFor(() => expect(onCopyFromTier).toHaveBeenCalledWith("tier-1-local"));
  });

  it("surfaces failed copy operations and disables actions without a scenario", async () => {
    const { rerender } = renderWithProviders(
      <WorkspaceToolbar
        scenario="secrets-manager"
        tier="tier-2-desktop"
        availableTiers={[{ value: "tier-1-local", label: "Tier 1" }, { value: "tier-2-desktop", label: "Tier 2" }]}
        availableScenarios={[]}
        searchQuery=""
        filter="all"
        hasPendingChanges={false}
        isSaving={false}
        onScenarioChange={() => {}}
        onTierChange={() => {}}
        onSearchChange={() => {}}
        onFilterChange={() => {}}
        onSaveAll={() => {}}
        onExport={() => {}}
        onCopyFromTier={async () => { throw new Error("copy unavailable"); }}
      />
    );
    fireEvent.click(screen.getByText("Copy from tier"));
    fireEvent.click(screen.getByRole("button", { name: "Tier 1" }));
    await waitFor(() => expect(screen.getByText("copy unavailable")).toBeInTheDocument());

    rerender(
      <WorkspaceToolbar
        scenario=""
        tier="tier-2-desktop"
        availableTiers={[{ value: "tier-1-local", label: "Tier 1" }, { value: "tier-2-desktop", label: "Tier 2" }]}
        availableScenarios={[]}
        searchQuery=""
        filter="all"
        hasPendingChanges={false}
        isSaving={false}
        onScenarioChange={() => {}}
        onTierChange={() => {}}
        onSearchChange={() => {}}
        onFilterChange={() => {}}
        onSaveAll={() => {}}
        onExport={() => {}}
        onCopyFromTier={async () => {}}
      />
    );
    expect(screen.getByRole("button", { name: "Export" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Copy from tier" })).toBeDisabled();
  });
});
