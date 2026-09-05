import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { UseManifestWorkspaceReturn } from "./useManifestWorkspace";
import { renderWithProviders } from "../../test-utils";
import { ManifestWorkspace } from "./ManifestWorkspace";

const useManifestWorkspaceMock = vi.hoisted(() => vi.fn());

vi.mock("./useManifestWorkspace", () => ({
  useManifestWorkspace: useManifestWorkspaceMock
}));

const manifest = {
  scenario: "secrets-manager",
  tier: "tier-2-desktop",
  generated_at: "2026-07-23T00:00:00Z",
  resources: [],
  secrets: [],
  summary: {
    total_secrets: 0,
    strategized_secrets: 0,
    requires_action: 0,
    blocking_secrets: [],
    classification_weights: {},
    strategy_breakdown: {},
    scope_readiness: {}
  }
};

function workspace(overrides: Partial<UseManifestWorkspaceReturn> = {}): UseManifestWorkspaceReturn {
  return {
    scenario: "secrets-manager",
    tier: "tier-2-desktop",
    setScenario: vi.fn(),
    setTier: vi.fn(),
    availableTiers: [{ value: "tier-2-desktop", label: "Tier 2" }],
    availableScenarios: [],
    manifest,
    manifestIsLoading: false,
    manifestIsError: false,
    manifestError: null,
    refreshManifest: vi.fn(),
    resources: [],
    allResources: [],
    selectedSecret: null,
    summary: {
      totalSecrets: 0,
      strategizedSecrets: 0,
      blockingSecrets: 0,
      excludedSecrets: 0,
      overriddenSecrets: 0,
      resourceCount: 0
    },
    filter: "all",
    setFilter: vi.fn(),
    searchQuery: "",
    setSearchQuery: vi.fn(),
    expandedResources: new Set(),
    toggleResource: vi.fn(),
    selectSecret: vi.fn(),
    selectedSecretId: null,
    isExcluded: vi.fn(() => false),
    toggleSecretExclusion: vi.fn(),
    toggleResourceExclusion: vi.fn(),
    excludedResources: new Set(),
    excludedSecrets: new Set(),
    overriddenSecrets: new Set(),
    isSecretOverridden: vi.fn(() => false),
    overridesQuery: {} as UseManifestWorkspaceReturn["overridesQuery"],
    pendingOverrides: new Map(),
    updatePendingChange: vi.fn(),
    saveOverride: vi.fn(),
    resetOverride: vi.fn(),
    saveAllPending: vi.fn(),
    hasPendingChanges: false,
    isSaving: false,
    isDeleting: false,
    copyFromTier: vi.fn(),
    isCopying: false,
    copyError: null,
    exportManifest: vi.fn(),
    getExportPreview: vi.fn(() => manifest),
    jsonPanelOpen: false,
    setJsonPanelOpen: vi.fn(),
    confirmDialog: null,
    closeConfirmDialog: vi.fn(),
    ...overrides
  };
}

describe("ManifestWorkspace", () => {
  afterEach(cleanup);

  it("shows loading, error retry, and scenario-selection states", () => {
    const refreshManifest = vi.fn();
    useManifestWorkspaceMock.mockReturnValue(workspace({ manifest: undefined, manifestIsLoading: true }));
    const { rerender } = renderWithProviders(<ManifestWorkspace />);
    expect(screen.getByText("Loading manifest...")).toBeInTheDocument();

    useManifestWorkspaceMock.mockReturnValue(workspace({
      manifest: undefined,
      manifestIsError: true,
      manifestError: new Error("manifest service unavailable"),
      refreshManifest
    }));
    rerender(<ManifestWorkspace />);
    expect(screen.getByText("Failed to load manifest")).toBeInTheDocument();
    expect(screen.getByText("manifest service unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(refreshManifest).toHaveBeenCalledOnce();

    useManifestWorkspaceMock.mockReturnValue(workspace({ scenario: "", manifest: undefined }));
    rerender(<ManifestWorkspace />);
    expect(screen.getByText("Select a scenario")).toBeInTheDocument();
  });

  it("routes refresh, view switches, and collapsed workspace controls", () => {
    const refreshManifest = vi.fn();
    const setJsonPanelOpen = vi.fn();
    useManifestWorkspaceMock.mockReturnValue(workspace({ refreshManifest, setJsonPanelOpen }));
    const onToggleCollapse = vi.fn();
    const { rerender } = renderWithProviders(
      <ManifestWorkspace initialScenario="secrets-manager" onToggleCollapse={onToggleCollapse} />
    );

    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    fireEvent.click(screen.getByRole("button", { name: "JSON Preview" }));
    fireEvent.click(screen.getByRole("button", { name: "Details" }));
    expect(refreshManifest).toHaveBeenCalledOnce();
    expect(setJsonPanelOpen).toHaveBeenNthCalledWith(1, true);
    expect(setJsonPanelOpen).toHaveBeenNthCalledWith(2, false);
    expect(screen.getByText("Resources:")).toBeInTheDocument();

    rerender(<ManifestWorkspace initialScenario="secrets-manager" isCollapsed onToggleCollapse={onToggleCollapse} />);
    expect(screen.getByText("secrets-manager (tier-2-desktop)")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Manifest Workspace"));
    expect(onToggleCollapse).toHaveBeenCalledOnce();
  });
});
