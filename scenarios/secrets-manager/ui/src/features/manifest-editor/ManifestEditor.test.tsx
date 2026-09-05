import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { ManifestEditor } from "./ManifestEditor";

const editorMock = vi.hoisted(() => vi.fn());

vi.mock("./useManifestEditor", () => ({ useManifestEditor: editorMock }));
vi.mock("./ManifestToolbar", () => ({
  ManifestToolbar: (props: { onSearchChange: (value: string) => void; onFilterChange: (value: string) => void; onSaveAll: () => void; onExport: () => void; onClose: () => void; onCopyFromTier: (tier: string) => void }) => (
    <div>
      <button onClick={() => props.onSearchChange("token")}>search</button>
      <button onClick={() => props.onFilterChange("blocking")}>filter</button>
      <button onClick={props.onSaveAll}>save all</button>
      <button onClick={props.onExport}>export</button>
      <button onClick={props.onClose}>close</button>
      <button onClick={() => props.onCopyFromTier("tier-1-local")}>copy</button>
    </div>
  )
}));
vi.mock("./ResourceTree", () => ({
  ResourceTree: (props: { onToggleResource: (resource: string) => void; onSelectSecret: (resource: string, key: string) => void; onToggleResourceExclusion: (resource: string) => void; onToggleSecretExclusion: (resource: string, key: string) => void }) => (
    <div>
      <button onClick={() => props.onToggleResource("vault")}>toggle resource</button>
      <button onClick={() => props.onSelectSecret("vault", "TOKEN")}>select secret</button>
      <button onClick={() => props.onToggleResourceExclusion("vault")}>exclude resource</button>
      <button onClick={() => props.onToggleSecretExclusion("vault", "TOKEN")}>exclude secret</button>
    </div>
  )
}));
vi.mock("./ManifestSecretDetailPanel", () => ({
  ManifestSecretDetailPanel: (props: { onUpdatePendingChange: (change: { handling_strategy: string }) => void; onSave: () => void; onReset: () => void; onToggleExclude: () => void }) => (
    <div>
      <button onClick={() => props.onUpdatePendingChange({ handling_strategy: "prompt" })}>change detail</button>
      <button onClick={props.onSave}>save detail</button>
      <button onClick={props.onReset}>reset detail</button>
      <button onClick={props.onToggleExclude}>exclude detail</button>
    </div>
  )
}));
vi.mock("./ManifestSummaryBar", () => ({ ManifestSummaryBar: () => <div>summary</div> }));

const manifest = {
  scenario: "secrets-manager", tier: "tier-2-desktop", generated_at: "2026-07-23T00:00:00Z", resources: ["vault"], secrets: [],
  summary: { total_secrets: 0, strategized_secrets: 0, requires_action: 0, blocking_secrets: [], classification_weights: {}, strategy_breakdown: {}, scope_readiness: {} }
};

describe("ManifestEditor", () => {
  it("routes toolbar, tree, and selected-secret actions through the editor state", () => {
    const state = {
      searchQuery: "", filter: "all", hasPendingChanges: true, isSaving: false, isCopying: false, copyError: null,
      resources: [], expandedResources: new Set(), selectedSecret: null, selectedSecretId: { resource: "vault", key: "TOKEN" },
      excludedResources: new Set(), excludedSecrets: new Set(), overriddenSecrets: new Set(), pendingOverrides: new Map([["vault:TOKEN", { changes: {} }]]),
      setSearchQuery: vi.fn(), setFilter: vi.fn(), saveAllPending: vi.fn(), exportManifest: vi.fn(), copyFromTier: vi.fn(),
      toggleResource: vi.fn(), selectSecret: vi.fn(), toggleResourceExclusion: vi.fn(), toggleSecretExclusion: vi.fn(),
      isExcluded: vi.fn(() => false), isSecretOverridden: vi.fn(() => false), updatePendingChange: vi.fn(), saveOverride: vi.fn(), resetOverride: vi.fn(),
      isDeleting: false, summary: { resourceCount: 1, totalSecrets: 1, strategizedSecrets: 0, blockingSecrets: 1, excludedSecrets: 0, overriddenSecrets: 0 }, getExportPreview: vi.fn(() => manifest)
    };
    editorMock.mockReturnValue(state);
    const onClose = vi.fn();
    const onExport = vi.fn();
    renderWithProviders(<ManifestEditor scenario="secrets-manager" tier="tier-2-desktop" initialManifest={manifest} onClose={onClose} onExport={onExport} />);

    for (const action of ["search", "filter", "save all", "export", "close", "copy", "toggle resource", "select secret", "exclude resource", "exclude secret", "change detail", "save detail", "reset detail", "exclude detail"]) {
      fireEvent.click(screen.getByRole("button", { name: action }));
    }
    expect(state.setSearchQuery).toHaveBeenCalledWith("token");
    expect(state.setFilter).toHaveBeenCalledWith("blocking");
    expect(state.exportManifest).toHaveBeenCalledOnce();
    expect(onExport).toHaveBeenCalledOnce();
    expect(onClose).toHaveBeenCalledOnce();
    expect(state.copyFromTier).toHaveBeenCalledWith("tier-1-local");
    expect(state.updatePendingChange).toHaveBeenCalledWith("vault", "TOKEN", { handling_strategy: "prompt" });
    expect(state.saveOverride).toHaveBeenCalledWith("vault", "TOKEN");
    expect(state.resetOverride).toHaveBeenCalledWith("vault", "TOKEN");
    expect(state.toggleSecretExclusion).toHaveBeenCalledWith("vault", "TOKEN");
  });
});
