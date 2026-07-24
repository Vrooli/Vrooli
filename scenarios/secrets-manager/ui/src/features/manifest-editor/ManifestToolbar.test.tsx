import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { ManifestToolbar } from "./ManifestToolbar";

describe("ManifestToolbar", () => {
  afterEach(cleanup);

  it("routes search, filtering, save, export, close, and copying", async () => {
    const onSearchChange = vi.fn();
    const onFilterChange = vi.fn();
    const onSaveAll = vi.fn();
    const onExport = vi.fn();
    const onClose = vi.fn();
    const onCopyFromTier = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(
      <ManifestToolbar
        searchQuery=""
        filter="all"
        hasPendingChanges
        isSaving={false}
        currentTier="tier-2"
        availableTiers={["tier-1", "tier-2"]}
        onSearchChange={onSearchChange}
        onFilterChange={onFilterChange}
        onSaveAll={onSaveAll}
        onExport={onExport}
        onClose={onClose}
        onCopyFromTier={onCopyFromTier}
      />
    );

    fireEvent.change(screen.getByPlaceholderText("Search secrets..."), { target: { value: "token" } });
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "blocking" } });
    fireEvent.click(screen.getByRole("button", { name: "Save All" }));
    fireEvent.click(screen.getByRole("button", { name: "Export" }));
    fireEvent.click(screen.getByRole("button", { name: "Copy from tier" }));
    fireEvent.click(screen.getByRole("button", { name: "tier-1" }));
    fireEvent.click(screen.getByRole("button", { name: "Close manifest editor" }));
    expect(onSearchChange).toHaveBeenCalledWith("token");
    expect(onFilterChange).toHaveBeenCalledWith("blocking");
    expect(onSaveAll).toHaveBeenCalledOnce();
    expect(onExport).toHaveBeenCalledOnce();
    expect(onClose).toHaveBeenCalledOnce();
    await waitFor(() => expect(onCopyFromTier).toHaveBeenCalledWith("tier-1"));
  });

  it("reports copy errors and disables save while saving", async () => {
    renderWithProviders(
      <ManifestToolbar
        searchQuery=""
        filter="all"
        hasPendingChanges
        isSaving
        currentTier="tier-2"
        availableTiers={["tier-1", "tier-2"]}
        onSearchChange={() => {}}
        onFilterChange={() => {}}
        onSaveAll={() => {}}
        onExport={() => {}}
        onClose={() => {}}
        onCopyFromTier={async () => { throw new Error("copy unavailable"); }}
      />
    );
    expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Copy from tier" }));
    fireEvent.click(screen.getByRole("button", { name: "tier-1" }));
    await waitFor(() => expect(screen.getByText("copy unavailable")).toBeInTheDocument());
  });
});
