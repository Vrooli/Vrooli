import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { DeploymentManifestResponse } from "../../lib/api";
import { useManifestEditor } from "./useManifestEditor";

const apiMocks = vi.hoisted(() => ({
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
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function renderEditor() {
  return renderHook(
    () => useManifestEditor({ scenario: "secrets-manager", tier: "tier-2-desktop", initialManifest: manifest }),
    { wrapper: testWrapper() }
  );
}

describe("useManifestEditor", () => {
  beforeEach(() => {
    apiMocks.fetchScenarioTierOverrides.mockResolvedValue({ overrides: [], count: 0 });
    apiMocks.setScenarioOverride.mockResolvedValue({ id: "override-1" });
    apiMocks.deleteScenarioOverride.mockResolvedValue({ success: true });
    apiMocks.copyOverridesFromTier.mockResolvedValue({ success: true, copied: 1 });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("keeps resource selection and session exclusions local while updating the deployment summary", async () => {
    const { result } = renderEditor();
    await waitFor(() => expect(result.current.overridesQuery.isSuccess).toBe(true));

    expect(result.current.expandedResources).toEqual(new Set(["vault", "redis"]));
    expect(result.current.summary.blockingSecrets).toBe(1);

    act(() => result.current.toggleResource("vault"));
    expect(result.current.expandedResources.has("vault")).toBe(false);

    act(() => result.current.selectSecret("vault", "VAULT_TOKEN"));
    expect(result.current.selectedSecret).toMatchObject({ resource_name: "vault", secret_key: "VAULT_TOKEN" });
    expect(result.current.expandedResources.has("vault")).toBe(true);

    act(() => result.current.excludeSecret("vault", "VAULT_TOKEN"));
    expect(result.current.isExcluded("vault", "VAULT_TOKEN")).toBe(true);
    expect(result.current.summary.blockingSecrets).toBe(0);
    expect(result.current.getExportPreview().secrets.map((secret) => secret.secret_key)).toEqual(["REDIS_PASSWORD"]);

    act(() => result.current.toggleResourceExclusion("redis"));
    expect(result.current.isExcluded("redis")).toBe(true);
    expect(result.current.getExportPreview().secrets).toEqual([]);

    act(() => result.current.includeSecret("vault", "VAULT_TOKEN"));
    expect(result.current.isExcluded("vault", "VAULT_TOKEN")).toBe(false);
  });

  it("persists only dirty override edits and clears them after save or reset", async () => {
    const { result } = renderEditor();
    await waitFor(() => expect(result.current.overridesQuery.isSuccess).toBe(true));

    await act(async () => {
      await result.current.saveOverride("vault", "VAULT_TOKEN");
    });
    expect(apiMocks.setScenarioOverride).not.toHaveBeenCalled();

    act(() => result.current.updatePendingChange("vault", "VAULT_TOKEN", {
      handling_strategy: "generate",
      requires_user_input: false,
      generator_template: { length: 32 }
    }));
    expect(result.current.hasPendingChanges).toBe(true);
    expect(result.current.pendingOverrides.get("vault:VAULT_TOKEN")?.changes).toMatchObject({ handling_strategy: "generate" });

    await act(async () => {
      await result.current.saveOverride("vault", "VAULT_TOKEN");
    });
    expect(apiMocks.setScenarioOverride).toHaveBeenCalledWith(
      "secrets-manager",
      "tier-2-desktop",
      "vault",
      "VAULT_TOKEN",
      expect.objectContaining({ handling_strategy: "generate", requires_user_input: false, generator_template: { length: 32 } })
    );
    expect(result.current.hasPendingChanges).toBe(false);

    act(() => result.current.updatePendingChange("redis", "REDIS_PASSWORD", { handling_strategy: "delegate" }));
    await act(async () => {
      await result.current.resetOverride("redis", "REDIS_PASSWORD");
    });
    expect(apiMocks.deleteScenarioOverride).toHaveBeenCalledWith("secrets-manager", "tier-2-desktop", "redis", "REDIS_PASSWORD");
    expect(result.current.pendingOverrides.has("redis:REDIS_PASSWORD")).toBe(false);
  });

  it("surfaces server override state and sends the selected source tier for a copy", async () => {
    apiMocks.fetchScenarioTierOverrides.mockResolvedValueOnce({
      overrides: [{ resource_name: "vault", secret_key: "VAULT_TOKEN" }],
      count: 1
    });
    const { result } = renderEditor();
    await waitFor(() => expect(result.current.overridesQuery.isSuccess).toBe(true));

    expect(result.current.isSecretOverridden("vault", "VAULT_TOKEN")).toBe(true);
    expect(result.current.isSecretOverridden("redis", "REDIS_PASSWORD")).toBe(false);

    await act(async () => {
      await result.current.copyFromTier("tier-1-local-dev", true);
    });
    expect(apiMocks.copyOverridesFromTier).toHaveBeenCalledWith("secrets-manager", {
      source_tier: "tier-1-local-dev",
      target_tier: "tier-2-desktop",
      overwrite: true
    });
  });
});
