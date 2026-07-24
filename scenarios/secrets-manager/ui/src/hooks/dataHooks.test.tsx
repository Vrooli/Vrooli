import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useCampaigns } from "./useCampaigns";
import { useScenarios } from "./useScenarios";
import { useSecretsData } from "./useSecretsData";
import { useVulnerabilities } from "./useVulnerabilities";

const apiMocks = vi.hoisted(() => ({
  fetchCampaigns: vi.fn(),
  fetchScenarios: vi.fn(),
  fetchHealth: vi.fn(),
  fetchVaultStatus: vi.fn(),
  fetchCompliance: vi.fn(),
  fetchOrientationSummary: vi.fn(),
  fetchVulnerabilities: vi.fn()
}));

vi.mock("../lib/api", () => apiMocks);

function testWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } }
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("dashboard data hooks", () => {
  beforeEach(() => {
    apiMocks.fetchCampaigns.mockImplementation(({ includeReadiness }: { includeReadiness?: boolean } = {}) => Promise.resolve(
      includeReadiness
        ? { campaigns: [{ id: "campaign-1", scenario: "secrets-manager", tier: "tier-2-desktop", status: "ready", progress: 100, blockers: 0, updated_at: "now" }], count: 1 }
        : { campaigns: [{ id: "campaign-1", scenario: "secrets-manager", tier: "tier-2-desktop", status: "draft", progress: 20, blockers: 2, updated_at: "earlier" }, { id: "campaign-2", scenario: "api-gateway", tier: "tier-4-saas", status: "ready", progress: 100, blockers: 0, updated_at: "now" }], count: 2 }
    ));
    apiMocks.fetchScenarios.mockResolvedValue({ scenarios: [{ name: "secrets-manager" }, { name: "api-gateway" }], count: 2 });
    apiMocks.fetchHealth.mockResolvedValue({ status: "ok" });
    apiMocks.fetchVaultStatus.mockResolvedValue({ resource_statuses: [{ resource_name: "vault" }] });
    apiMocks.fetchCompliance.mockResolvedValue({ overall_score: 100 });
    apiMocks.fetchOrientationSummary.mockResolvedValue({ hero_stats: { missing_secrets: 0 } });
    apiMocks.fetchVulnerabilities.mockResolvedValue({
      vulnerabilities: [{ id: "vuln-1", component_name: "api-gateway" }], total_count: 1
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("merges scenario readiness campaigns over the base list and filters by scenario", async () => {
    const { result } = renderHook(() => useCampaigns("secrets-manager"), { wrapper: testWrapper() });
    await waitFor(() => expect(result.current.readinessQuery.isSuccess).toBe(true));

    expect(result.current.campaigns).toHaveLength(2);
    expect(result.current.campaigns.find((campaign) => campaign.id === "campaign-1")?.status).toBe("ready");
    act(() => result.current.setSearch("gateway"));
    expect(result.current.filtered.map((campaign) => campaign.id)).toEqual(["campaign-2"]);
  });

  it("filters scenarios and composes resource and vulnerability options", async () => {
    const scenarios = renderHook(() => useScenarios(), { wrapper: testWrapper() });
    await waitFor(() => expect(scenarios.result.current.query.isSuccess).toBe(true));
    act(() => scenarios.result.current.setSearch("gateway"));
    expect(scenarios.result.current.filtered.map((scenario) => scenario.name)).toEqual(["api-gateway"]);

    const vulnerabilities = renderHook(
      () => useVulnerabilities({ resource_statuses: [{ resource_name: "vault" }] }),
      { wrapper: testWrapper() }
    );
    await waitFor(() => expect(vulnerabilities.result.current.vulnerabilityQuery.isSuccess).toBe(true));
    expect(vulnerabilities.result.current.componentOptions).toEqual(["api-gateway", "vault"]);
    act(() => vulnerabilities.result.current.setSeverityFilter("critical"));
    await waitFor(() => expect(apiMocks.fetchVulnerabilities).toHaveBeenCalledWith({
      componentType: undefined,
      component: undefined,
      severity: "critical"
    }));
  });

  it("loads all dashboard data sources and refreshes every source on request", async () => {
    const { result } = renderHook(() => useSecretsData(), { wrapper: testWrapper() });
    await waitFor(() => expect(result.current.orientationQuery.isSuccess).toBe(true));
    expect(result.current.isInitialLoading).toBe(false);

    act(() => result.current.refreshAll());
    await waitFor(() => {
      expect(apiMocks.fetchHealth.mock.calls.length).toBeGreaterThan(1);
      expect(apiMocks.fetchVaultStatus.mock.calls.length).toBeGreaterThan(1);
      expect(apiMocks.fetchCompliance.mock.calls.length).toBeGreaterThan(1);
      expect(apiMocks.fetchOrientationSummary.mock.calls.length).toBeGreaterThan(1);
    });
  });
});
