import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { DeploymentManifestResponse } from "../../lib/api";
import { useManifestWorkspace } from "./useManifestWorkspace";

const apiMocks = vi.hoisted(() => ({
  generateDeploymentManifest: vi.fn(),
  fetchScenarioTierOverrides: vi.fn(),
  setScenarioOverride: vi.fn(),
  deleteScenarioOverride: vi.fn(),
  copyOverridesFromTier: vi.fn()
}));

vi.mock("../../lib/api", () => apiMocks);

const manifest: DeploymentManifestResponse = {
  scenario: "secrets-manager",
  tier: "tier-2-desktop",
  generated_at: "2026-07-23T00:00:00Z",
  resources: ["vault", "redis"],
  secrets: [
    {
      resource_name: "vault",
      secret_key: "VAULT_TOKEN",
      secret_type: "token",
      required: true,
      classification: "service",
      handling_strategy: "none",
      requires_user_input: false
    },
    {
      resource_name: "redis",
      secret_key: "REDIS_PASSWORD",
      secret_type: "password",
      required: true,
      classification: "service",
      handling_strategy: "prompt",
      requires_user_input: true
    }
  ],
  summary: {
    total_secrets: 2,
    strategized_secrets: 1,
    requires_action: 1,
    blocking_secrets: ["vault/VAULT_TOKEN"],
    classification_weights: {},
    strategy_breakdown: {},
    scope_readiness: {}
  }
};

function testWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } }
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function renderWorkspace(onScenarioChange = vi.fn()) {
  return renderHook(
    () => useManifestWorkspace({
      initialScenario: "secrets-manager",
      initialTier: "tier-2-desktop",
      onScenarioChange
    }),
    { wrapper: testWrapper() }
  );
}

describe("useManifestWorkspace", () => {
  beforeEach(() => {
    apiMocks.generateDeploymentManifest.mockResolvedValue(manifest);
    apiMocks.fetchScenarioTierOverrides.mockResolvedValue({ overrides: [], count: 0 });
    apiMocks.setScenarioOverride.mockResolvedValue({ id: "override-1" });
    apiMocks.deleteScenarioOverride.mockResolvedValue({ success: true });
    apiMocks.copyOverridesFromTier.mockResolvedValue({ success: true, copied: 1 });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("loads a manifest, expands its resources, and applies session exclusions to the export", async () => {
    const { result } = renderWorkspace();
    await waitFor(() => expect(result.current.manifest).toEqual(manifest));

    expect(result.current.expandedResources).toEqual(new Set(["vault", "redis"]));
    expect(result.current.summary.blockingSecrets).toBe(1);

    act(() => result.current.selectSecret("vault", "VAULT_TOKEN"));
    expect(result.current.selectedSecret).toMatchObject({ resource_name: "vault", secret_key: "VAULT_TOKEN" });
    expect(result.current.jsonPanelOpen).toBe(false);

    act(() => result.current.toggleSecretExclusion("vault", "VAULT_TOKEN"));
    expect(result.current.isExcluded("vault", "VAULT_TOKEN")).toBe(true);
    expect(result.current.summary.blockingSecrets).toBe(0);
    expect(result.current.getExportPreview()?.secrets.map((secret) => secret.secret_key)).toEqual(["REDIS_PASSWORD"]);

    act(() => result.current.toggleResourceExclusion("redis"));
    expect(result.current.isExcluded("redis")).toBe(true);
    expect(result.current.getExportPreview()?.secrets).toEqual([]);
  });

  it("guards scenario and tier changes until dirty changes are explicitly discarded", async () => {
    const onScenarioChange = vi.fn();
    const { result } = renderWorkspace(onScenarioChange);
    await waitFor(() => expect(result.current.manifest).toEqual(manifest));

    act(() => result.current.updatePendingChange("vault", "VAULT_TOKEN", { handling_strategy: "generate" }));
    expect(result.current.hasPendingChanges).toBe(true);

    act(() => result.current.setScenario("another-scenario"));
    expect(result.current.scenario).toBe("secrets-manager");
    expect(result.current.confirmDialog?.title).toBe("Unsaved changes");

    act(() => result.current.confirmDialog?.onConfirm());
    expect(result.current.scenario).toBe("another-scenario");
    expect(onScenarioChange).toHaveBeenCalledWith("another-scenario");
    expect(result.current.hasPendingChanges).toBe(false);

    act(() => result.current.updatePendingChange("vault", "VAULT_TOKEN", { prompt_label: "Vault token" }));
    expect(result.current.hasPendingChanges).toBe(true);
    act(() => result.current.setTier("tier-1-local"));
    expect(result.current.tier).toBe("tier-2-desktop");
    act(() => result.current.confirmDialog?.onConfirm());
    expect(result.current.tier).toBe("tier-1-local");
  });

  it("persists pending overrides, resets them, and copies selected tier settings", async () => {
    const { result } = renderWorkspace();
    await waitFor(() => expect(result.current.manifest).toEqual(manifest));

    act(() => result.current.updatePendingChange("vault", "VAULT_TOKEN", {
      handling_strategy: "generate",
      generator_template: { length: 32 }
    }));
    await act(async () => {
      await result.current.saveAllPending();
    });
    expect(apiMocks.setScenarioOverride).toHaveBeenCalledWith(
      "secrets-manager",
      "tier-2-desktop",
      "vault",
      "VAULT_TOKEN",
      expect.objectContaining({ handling_strategy: "generate", generator_template: { length: 32 } })
    );
    expect(result.current.hasPendingChanges).toBe(false);

    act(() => result.current.updatePendingChange("redis", "REDIS_PASSWORD", { handling_strategy: "delegate" }));
    await act(async () => {
      await result.current.resetOverride("redis", "REDIS_PASSWORD");
    });
    expect(apiMocks.deleteScenarioOverride).toHaveBeenCalledWith(
      "secrets-manager",
      "tier-2-desktop",
      "redis",
      "REDIS_PASSWORD"
    );

    await act(async () => {
      await result.current.copyFromTier("tier-1-local", true);
    });
    expect(apiMocks.copyOverridesFromTier).toHaveBeenCalledWith("secrets-manager", {
      source_tier: "tier-1-local",
      target_tier: "tier-2-desktop",
      overwrite: true
    });
  });
});
