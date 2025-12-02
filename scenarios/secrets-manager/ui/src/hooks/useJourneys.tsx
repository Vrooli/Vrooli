import { useCallback, useEffect, useMemo, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import {
  fetchDeploymentReadiness,
  generateDeploymentManifest,
  provisionSecrets,
  type DeploymentManifestRequest,
  type DeploymentManifestResponse,
  type DeploymentReadinessResponse,
  type ProvisionSecretsPayload,
  type ProvisionSecretsResponse,
  type ScenarioSummary
} from "../lib/api";
import { buildJourneySteps, type JourneyId } from "../features/journeys/journeySteps";
import { ScenarioSelector } from "../sections/ScenarioSelector";

interface UseJourneysOptions {
  selectedScenario?: string;
  onDeploymentScenarioChange?: (scenario: string) => void;
  scenarioSelection?: {
    scenarios: ScenarioSummary[];
    filtered: ScenarioSummary[];
    search: string;
    isLoading: boolean;
    selectedScenario?: string;
    onSearchChange: (value: string) => void;
    onSelect: (scenario: string) => void;
  };
  heroStats?: {
    vault_configured: number;
    vault_total: number;
    missing_secrets: number;
    risk_score: number;
  };
  orientationData?: {
    hero_stats: {
      risk_score: number;
    };
  };
  tierReadiness: Array<{
    tier: string;
    label: string;
    ready_percent: number;
    strategized: number;
    total: number;
  }>;
  topResourceNeedingAttention?: string;
  onOpenResource: (resourceName?: string, secretKey?: string) => void;
  onRefetchVulnerabilities: () => void;
}

export const useJourneys = (options: UseJourneysOptions) => {
  // Default to orientation journey on mount
  const [activeJourney, setActiveJourney] = useState<JourneyId | null>("orientation");
  const [journeyStep, setJourneyStep] = useState(0);
  const [deploymentScenario, setDeploymentScenario] = useState(options.selectedScenario || "picker-wheel");
  const [deploymentTier, setDeploymentTier] = useState("tier-2-desktop");
  const [resourceInput, setResourceInput] = useState("");
  const [provisionResourceInput, setProvisionResourceInput] = useState("");
  const [provisionSecretKey, setProvisionSecretKey] = useState("");
  const [provisionSecretValue, setProvisionSecretValue] = useState("");
  const [readinessRefreshKey, setReadinessRefreshKey] = useState(0);
  const [autoManifestDisabled, setAutoManifestDisabled] = useState(false);
  const [tierSnapshots, setTierSnapshots] = useState<
    Record<
      string,
      {
        summary?: DeploymentReadinessResponse["summary"];
        generatedAt?: string;
        loading?: boolean;
        error?: string;
      }
    >
  >({});

  const manifestMutation = useMutation<DeploymentManifestResponse, Error, DeploymentManifestRequest>({
    mutationFn: (payload) => generateDeploymentManifest(payload)
  });

  const provisionMutation = useMutation<ProvisionSecretsResponse, Error, ProvisionSecretsPayload>({
    mutationFn: (payload) => provisionSecrets(payload)
  });

  useEffect(() => {
    if (options.selectedScenario && options.selectedScenario !== deploymentScenario) {
      setDeploymentScenario(options.selectedScenario);
    }
  }, [options.selectedScenario, deploymentScenario]);

  const parseResources = useCallback((value: string) => {
    if (!value.trim()) return undefined;
    return value
      .split(/[\n,]+/)
      .map((item) => item.trim())
      .filter(Boolean);
  }, []);

  const handleManifestRequest = useCallback(() => {
    const resources = parseResources(resourceInput);
    manifestMutation.mutate({
      scenario: deploymentScenario,
      tier: deploymentTier,
      resources,
      include_optional: false
    });
  }, [manifestMutation.mutate, deploymentScenario, deploymentTier, parseResources, resourceInput]);

  // Auto-refresh manifest whenever scenario/tier/resources change (skip if user-facing errors disable it)
  useEffect(() => {
    if (!deploymentScenario || autoManifestDisabled) return;
    handleManifestRequest();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [deploymentScenario, deploymentTier, resourceInput, autoManifestDisabled]);

  // If manifest fetch fails, stop auto-loop and let the user click regenerate manually.
  useEffect(() => {
    if (manifestMutation.isError) {
      setAutoManifestDisabled(true);
    }
  }, [manifestMutation.isError]);

  // Re-enable auto manifest when inputs change or a success occurs.
  useEffect(() => {
    setAutoManifestDisabled(false);
  }, [deploymentScenario, deploymentTier]);

  useEffect(() => {
    if (manifestMutation.isSuccess) {
      setAutoManifestDisabled(false);
    }
  }, [manifestMutation.isSuccess]);

  // Fetch readiness snapshots for all tiers for this scenario
  useEffect(() => {
    if (!deploymentScenario || options.tierReadiness.length === 0) return;
    const tiers = options.tierReadiness.map((tier) => tier.tier);
    setTierSnapshots((prev) => {
      const next = { ...prev };
      tiers.forEach((tier) => {
        next[tier] = { ...next[tier], loading: true, error: undefined };
      });
      return next;
    });

    let cancelled = false;
    const resources = parseResources(resourceInput);

    const fetchTiers = async () => {
      for (const tier of tiers) {
        try {
          const data = await fetchDeploymentReadiness({
            scenario: deploymentScenario,
            tier,
            resources,
            include_optional: false
          });
          if (cancelled) return;
          setTierSnapshots((prev) => ({
            ...prev,
            [tier]: {
              summary: data.summary,
              generatedAt: data.generated_at,
              loading: false,
              error: undefined
            }
          }));
        } catch (err) {
          if (cancelled) return;
          const message = err instanceof Error ? err.message : "Failed to load readiness";
          setTierSnapshots((prev) => ({
            ...prev,
            [tier]: { ...prev[tier], loading: false, error: message }
          }));
        }
      }
    };

    fetchTiers();
    return () => {
      cancelled = true;
    };
  }, [deploymentScenario, resourceInput, options.tierReadiness, parseResources, readinessRefreshKey]);

  const handleProvisionSubmit = useCallback(() => {
    provisionMutation.mutate({
      resource: provisionResourceInput,
      secret_key: provisionSecretKey,
      secret_value: provisionSecretValue
    });
  }, [provisionMutation, provisionResourceInput, provisionSecretKey, provisionSecretValue]);

  // Clear provision form on success
  useEffect(() => {
    if (provisionMutation.isSuccess) {
      setProvisionSecretKey("");
      setProvisionSecretValue("");
    }
  }, [provisionMutation.isSuccess]);

  const handleSetDeploymentScenario = useCallback(
    (scenario: string) => {
      setDeploymentScenario(scenario);
      options.onDeploymentScenarioChange?.(scenario);
    },
    [options]
  );

  const selectedTierSnapshot = tierSnapshots[deploymentTier];
  const selectedTierHasData = !!selectedTierSnapshot?.summary;
  const selectedTierLoading = !!selectedTierSnapshot?.loading;

  const handleReadinessRefresh = useCallback(() => {
    setReadinessRefreshKey((value) => value + 1);
  }, []);

  const journeySteps = useMemo(
    () =>
      buildJourneySteps(activeJourney, {
        heroStats: options.heroStats,
        orientationData: options.orientationData,
        tierReadiness: options.tierReadiness,
        deploymentScenario,
        deploymentTier,
        resourceInput,
        provisionResourceInput,
        provisionSecretKey,
        provisionSecretValue,
        provisionIsLoading: provisionMutation.isPending,
        provisionIsSuccess: provisionMutation.isSuccess,
        provisionError: provisionMutation.error?.message,
        manifestData: manifestMutation.data,
        manifestIsLoading: manifestMutation.isPending,
        manifestIsError: manifestMutation.isError,
        manifestError: manifestMutation.error ?? undefined,
        readinessSummary: selectedTierSnapshot?.summary,
        readinessGeneratedAt: selectedTierSnapshot?.generatedAt,
        readinessIsLoading: selectedTierSnapshot?.loading,
        readinessIsError: !!selectedTierSnapshot?.error,
        readinessError: selectedTierSnapshot?.error ? new Error(selectedTierSnapshot.error) : undefined,
        readinessByTier: tierSnapshots,
        onRefreshReadiness: handleReadinessRefresh,
        topResourceNeedingAttention: options.topResourceNeedingAttention,
        scenarioSelectionContent: options.scenarioSelection ? (
          <ScenarioSelector
            scenarios={options.scenarioSelection.scenarios}
            filtered={options.scenarioSelection.filtered}
            search={options.scenarioSelection.search}
            isLoading={options.scenarioSelection.isLoading}
            selectedScenario={options.scenarioSelection.selectedScenario || deploymentScenario}
            onSearchChange={options.scenarioSelection.onSearchChange}
            onSelect={handleSetDeploymentScenario}
          />
        ) : null,
        onOpenResource: options.onOpenResource,
        onRefetchVulnerabilities: options.onRefetchVulnerabilities,
        onManifestRequest: handleManifestRequest,
        onSetDeploymentScenario: handleSetDeploymentScenario,
        onSetDeploymentTier: setDeploymentTier,
        onSetResourceInput: setResourceInput,
        onSetProvisionResourceInput: setProvisionResourceInput,
        onSetProvisionSecretKey: setProvisionSecretKey,
        onSetProvisionSecretValue: setProvisionSecretValue,
        onProvisionSubmit: handleProvisionSubmit,
        scenarioSelection: options.scenarioSelection
      }),
    [
      activeJourney,
      options.heroStats,
      options.orientationData,
      options.tierReadiness,
      deploymentScenario,
      deploymentTier,
      resourceInput,
      provisionResourceInput,
      provisionSecretKey,
      provisionSecretValue,
      provisionMutation.isPending,
      provisionMutation.isSuccess,
      provisionMutation.error,
      manifestMutation.data,
      manifestMutation.isPending,
      manifestMutation.isError,
      manifestMutation.error,
      tierSnapshots,
      options.topResourceNeedingAttention,
      options.onOpenResource,
      options.onRefetchVulnerabilities,
      handleManifestRequest,
      handleProvisionSubmit,
      handleSetDeploymentScenario,
      handleReadinessRefresh,
      options.scenarioSelection
    ]
  );

  useEffect(() => {
    setJourneyStep(0);
  }, [activeJourney]);

  const handleJourneySelect = useCallback((journey: JourneyId) => {
    setActiveJourney(journey);
  }, []);

  const handleJourneyExit = useCallback(() => {
    setActiveJourney(null);
  }, []);

  const handleJourneyNext = useCallback(() => {
    setJourneyStep((value) => Math.min(journeySteps.length - 1, value + 1));
  }, [journeySteps.length]);

  const handleJourneyBack = useCallback(() => {
    setJourneyStep((value) => Math.max(0, value - 1));
  }, []);

  const journeyNextDisabled =
    (activeJourney === "prep-deployment" && journeyStep === 0 && !deploymentScenario) ||
    (activeJourney === "prep-deployment" && journeyStep === 1 && !selectedTierHasData && !selectedTierLoading) ||
    (activeJourney === "prep-deployment" && journeyStep >= 2 && !manifestMutation.data && !manifestMutation.isPending);

  return {
    activeJourney,
    journeyStep,
    journeySteps,
    handleJourneySelect,
    handleJourneyExit,
    handleJourneyNext,
    handleJourneyBack,
    journeyNextDisabled,
    deploymentFlow: {
      scenario: deploymentScenario,
      tier: deploymentTier,
      resourcesInput: resourceInput,
      manifestData: manifestMutation.data,
      manifestIsLoading: manifestMutation.isPending,
      manifestIsError: manifestMutation.isError,
      manifestError: manifestMutation.error ?? undefined,
      onSetScenario: handleSetDeploymentScenario,
      onSetTier: setDeploymentTier,
      onSetResourcesInput: setResourceInput,
      onGenerateManifest: handleManifestRequest
    }
  };
};
