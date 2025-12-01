import { useCallback, useEffect, useMemo, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import {
  generateDeploymentManifest,
  provisionSecrets,
  type DeploymentManifestRequest,
  type DeploymentManifestResponse,
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

  const parsedResources = useMemo(() => {
    if (!resourceInput.trim()) return undefined;
    return resourceInput
      .split(/[\n,]+/)
      .map((value) => value.trim())
      .filter(Boolean);
  }, [resourceInput]);

  const handleManifestRequest = useCallback(() => {
    manifestMutation.mutate({
      scenario: deploymentScenario,
      tier: deploymentTier,
      resources: parsedResources,
      include_optional: false
    });
  }, [manifestMutation, deploymentScenario, deploymentTier, parsedResources]);

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
      options.topResourceNeedingAttention,
      options.onOpenResource,
      options.onRefetchVulnerabilities,
      handleManifestRequest,
      handleProvisionSubmit,
      handleSetDeploymentScenario,
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

  return {
    activeJourney,
    journeyStep,
    journeySteps,
    handleJourneySelect,
    handleJourneyExit,
    handleJourneyNext,
    handleJourneyBack,
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
