import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ResourceDetail } from "../lib/api";
import { useResourcePanel } from "./useResourcePanel";

const apiMocks = vi.hoisted(() => ({
  fetchResourceDetail: vi.fn(),
  updateResourceSecret: vi.fn(),
  updateSecretStrategy: vi.fn(),
  updateVulnerabilityStatus: vi.fn(),
  fetchScenarioTierOverrides: vi.fn(),
  setScenarioOverride: vi.fn(),
  deleteScenarioOverride: vi.fn()
}));

vi.mock("../lib/api", () => apiMocks);

const detail: ResourceDetail = {
  resource_name: "vault",
  valid_secrets: 1,
  missing_secrets: 0,
  total_secrets: 1,
  secrets: [{
    id: "vault-token",
    secret_key: "VAULT_TOKEN",
    secret_type: "token",
    description: "Vault token",
    classification: "service",
    required: true,
    owner_team: "platform",
    owner_contact: "platform@example.test",
    tier_strategies: {},
    validation_state: "valid"
  }],
  open_vulnerabilities: []
};

function testWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } }
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function renderPanel() {
  return renderHook(() => useResourcePanel({ selectedScenario: "secrets-manager" }), { wrapper: testWrapper() });
}

describe("useResourcePanel", () => {
  beforeEach(() => {
    apiMocks.fetchResourceDetail.mockResolvedValue(detail);
    apiMocks.fetchScenarioTierOverrides.mockResolvedValue({
      scenario: "secrets-manager",
      tier: "tier-2-desktop",
      overrides: [],
      count: 0
    });
    apiMocks.updateResourceSecret.mockResolvedValue(detail.secrets[0]);
    apiMocks.updateSecretStrategy.mockResolvedValue(detail.secrets[0]);
    apiMocks.updateVulnerabilityStatus.mockResolvedValue({ id: "vuln-1", status: "resolved" });
    apiMocks.setScenarioOverride.mockResolvedValue({ id: "override-1" });
    apiMocks.deleteScenarioOverride.mockResolvedValue({ success: true, message: "deleted" });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("opens a resource, defaults its selected secret, and issues resource-level updates", async () => {
    const { result } = renderPanel();
    act(() => result.current.openResourcePanel("vault", undefined, "tier-4-saas"));
    await waitFor(() => expect(result.current.resourceDetailQuery.data).toEqual(detail));

    expect(result.current.activeResource).toBe("vault");
    expect(result.current.selectedSecretKey).toBe("VAULT_TOKEN");
    expect(result.current.strategyTier).toBe("tier-4-saas");

    act(() => result.current.handleSecretUpdate("VAULT_TOKEN", { classification: "credential" }));
    await waitFor(() => expect(apiMocks.updateResourceSecret).toHaveBeenCalledWith(
      "vault", "VAULT_TOKEN", { classification: "credential" }
    ));

    act(() => result.current.handleStrategyApply());
    await waitFor(() => expect(apiMocks.updateSecretStrategy).toHaveBeenCalledWith(
      "vault",
      "VAULT_TOKEN",
      expect.objectContaining({ tier: "tier-4-saas", handling_strategy: "prompt", requires_user_input: true })
    ));
  });

  it("uses a scenario override when requested and can remove that override", async () => {
    const { result } = renderPanel();
    act(() => result.current.openResourcePanel("vault", "VAULT_TOKEN"));
    await waitFor(() => expect(result.current.resourceDetailQuery.data).toEqual(detail));

    act(() => {
      result.current.setIsOverrideMode(true);
      result.current.setStrategyHandling("generate");
      result.current.setOverrideReason("desktop policy");
    });
    act(() => result.current.handleStrategyApply());
    await waitFor(() => expect(apiMocks.setScenarioOverride).toHaveBeenCalledWith(
      "secrets-manager",
      "tier-2-desktop",
      "vault",
      "VAULT_TOKEN",
      expect.objectContaining({ handling_strategy: "generate", requires_user_input: false, override_reason: "desktop policy" })
    ));

    act(() => result.current.handleDeleteOverride());
    await waitFor(() => expect(apiMocks.deleteScenarioOverride).toHaveBeenCalledWith(
      "secrets-manager", "tier-2-desktop", "vault", "VAULT_TOKEN"
    ));
  });

  it("exposes matching server overrides, updates vulnerability status, and clears selection on close", async () => {
    apiMocks.fetchScenarioTierOverrides.mockResolvedValueOnce({
      scenario: "secrets-manager",
      tier: "tier-2-desktop",
      overrides: [{ resource_name: "vault", secret_key: "VAULT_TOKEN", id: "override-1" }],
      count: 1
    });
    const { result } = renderPanel();
    act(() => result.current.openResourcePanel("vault", "VAULT_TOKEN"));
    await waitFor(() => expect(result.current.currentOverride).toMatchObject({ id: "override-1" }));

    act(() => result.current.handleVulnerabilityStatus("vuln-1", "resolved"));
    await waitFor(() => expect(apiMocks.updateVulnerabilityStatus).toHaveBeenCalledWith("vuln-1", { status: "resolved" }));

    act(() => result.current.closeResourcePanel());
    expect(result.current.activeResource).toBeNull();
    expect(result.current.selectedSecretKey).toBeNull();
  });
});
