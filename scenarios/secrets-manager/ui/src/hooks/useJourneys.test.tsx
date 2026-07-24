import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { DeploymentManifestResponse, DeploymentReadinessResponse } from "../lib/api";
import { useJourneys } from "./useJourneys";

const apiMocks = vi.hoisted(() => ({
  fetchDeploymentReadiness: vi.fn(),
  generateDeploymentManifest: vi.fn(),
  provisionSecrets: vi.fn()
}));

vi.mock("../lib/api", () => apiMocks);

const summary = {
  total_secrets: 2,
  strategized_secrets: 2,
  requires_action: 0,
  blocking_secrets: [],
  classification_weights: {},
  strategy_breakdown: {},
  scope_readiness: {}
};

const manifest: DeploymentManifestResponse = {
  scenario: "secrets-manager",
  tier: "tier-2-desktop",
  generated_at: "2026-07-23T00:00:00Z",
  resources: ["vault", "redis"],
  secrets: [],
  summary
};

const readiness: DeploymentReadinessResponse = {
  scenario: "secrets-manager",
  tier: "tier-2-desktop",
  resources: ["vault", "redis"],
  summary,
  generated_at: "2026-07-23T00:00:00Z"
};

function testWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } }
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

const options = {
  selectedScenario: "secrets-manager",
  tierReadiness: [{ tier: "tier-2-desktop", label: "Desktop", ready_percent: 100, strategized: 2, total: 2 }],
  heroStats: { vault_configured: 1, vault_total: 1, missing_secrets: 0, risk_score: 4 },
  onOpenResource: vi.fn(),
  onRefetchVulnerabilities: vi.fn(),
  onNavigateTab: vi.fn()
};

describe("useJourneys", () => {
  beforeEach(() => {
    apiMocks.generateDeploymentManifest.mockResolvedValue(manifest);
    apiMocks.fetchDeploymentReadiness.mockResolvedValue(readiness);
    apiMocks.provisionSecrets.mockResolvedValue({ success: true, path: "secret/vault/token" });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("auto-generates the selected deployment manifest and collects readiness by tier", async () => {
    const { result } = renderHook(() => useJourneys(options), { wrapper: testWrapper() });

    await waitFor(() => expect(result.current.deploymentFlow.manifestData).toEqual(manifest));
    await waitFor(() => expect(apiMocks.fetchDeploymentReadiness).toHaveBeenCalledTimes(1));
    expect(apiMocks.generateDeploymentManifest).toHaveBeenCalledWith({
      scenario: "secrets-manager",
      tier: "tier-2-desktop",
      resources: undefined,
      include_optional: false
    });
    expect(apiMocks.fetchDeploymentReadiness).toHaveBeenCalledWith({
      scenario: "secrets-manager",
      tier: "tier-2-desktop",
      resources: undefined,
      include_optional: false
    });

    act(() => result.current.deploymentFlow.onSetResourcesInput("vault,\nredis"));
    await waitFor(() => expect(result.current.deploymentFlow.resourcesInput).toBe("vault,\nredis"));
    act(() => result.current.deploymentFlow.onGenerateManifest());
    await waitFor(() => expect(apiMocks.generateDeploymentManifest).toHaveBeenCalledWith({
      scenario: "secrets-manager",
      tier: "tier-2-desktop",
      resources: ["vault", "redis"],
      include_optional: false
    }));
  });

  it("drives journey navigation and only enables deployment progression after readiness and manifest data exist", async () => {
    const { result } = renderHook(() => useJourneys(options), { wrapper: testWrapper() });
    await waitFor(() => expect(result.current.deploymentFlow.manifestData).toEqual(manifest));

    expect(result.current.activeJourney).toBe("orientation");
    act(() => result.current.handleJourneyNext());
    expect(result.current.journeyStep).toBe(1);
    act(() => result.current.handleJourneyBack());
    expect(result.current.journeyStep).toBe(0);

    act(() => result.current.handleJourneySelect("prep-deployment"));
    expect(result.current.journeyStep).toBe(0);
    expect(result.current.journeyNextDisabled).toBe(false);

    act(() => result.current.setJourneyStep(3));
    await waitFor(() => expect(result.current.journeyNextDisabled).toBe(false));
    act(() => result.current.setJourneyStep(4));
    expect(result.current.journeyNextDisabled).toBe(false);

    act(() => result.current.handleJourneyExit());
    expect(result.current.activeJourney).toBeNull();
    expect(result.current.journeySteps).toEqual([]);
  });

  it("propagates scenario and tier changes through the deployment flow", async () => {
    const onDeploymentScenarioChange = vi.fn();
    const { result } = renderHook(
      () => useJourneys({ ...options, onDeploymentScenarioChange }),
      { wrapper: testWrapper() }
    );
    await waitFor(() => expect(result.current.deploymentFlow.manifestData).toEqual(manifest));

    act(() => result.current.deploymentFlow.onSetScenario("api-gateway"));
    expect(result.current.deploymentFlow.scenario).toBe("api-gateway");
    expect(onDeploymentScenarioChange).toHaveBeenCalledWith("api-gateway");

    act(() => result.current.deploymentFlow.onSetTier("tier-4-saas"));
    expect(result.current.deploymentFlow.tier).toBe("tier-4-saas");
  });

  it("prevents advancing a failed deployment until scenario, readiness, and manifest requirements are met", async () => {
    apiMocks.generateDeploymentManifest.mockRejectedValue(new Error("manifest unavailable"));
    const { result } = renderHook(
      () => useJourneys({ ...options, tierReadiness: [] }),
      { wrapper: testWrapper() }
    );

    await waitFor(() => expect(result.current.deploymentFlow.manifestIsError).toBe(true));
    act(() => result.current.handleJourneySelect("prep-deployment"));
    act(() => result.current.deploymentFlow.onSetScenario(""));
    act(() => result.current.setJourneyStep(1));
    expect(result.current.journeyNextDisabled).toBe(true);
    act(() => result.current.deploymentFlow.onSetScenario("secrets-manager"));
    act(() => result.current.setJourneyStep(3));
    expect(result.current.journeyNextDisabled).toBe(true);
    act(() => result.current.setJourneyStep(4));
    expect(result.current.journeyNextDisabled).toBe(true);
  });

  it("records readiness failures without discarding the selected scenario", async () => {
    apiMocks.fetchDeploymentReadiness.mockRejectedValue(new Error("readiness unavailable"));
    const { result } = renderHook(() => useJourneys(options), { wrapper: testWrapper() });
    await waitFor(() => expect(apiMocks.fetchDeploymentReadiness).toHaveBeenCalled());
    await waitFor(() => expect(result.current.journeySteps.length).toBeGreaterThan(0));
    expect(result.current.deploymentFlow.scenario).toBe("secrets-manager");
  });

  it("keeps deployment progression disabled when readiness responds without a body", async () => {
    apiMocks.fetchDeploymentReadiness.mockResolvedValue(undefined);
    const { result } = renderHook(() => useJourneys(options), { wrapper: testWrapper() });

    await waitFor(() => expect(apiMocks.fetchDeploymentReadiness).toHaveBeenCalled());
    act(() => result.current.handleJourneySelect("prep-deployment"));
    act(() => result.current.setJourneyStep(3));

    await waitFor(() => expect(result.current.journeyNextDisabled).toBe(true));
  });
});
