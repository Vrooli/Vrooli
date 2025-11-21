import { useCallback, useEffect, useMemo, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { generateDeploymentManifest, type DeploymentManifestRequest, type DeploymentManifestResponse } from "../lib/api";
import { buildJourneySteps, type JourneyId } from "../features/journeys/journeySteps";

interface UseJourneysOptions {
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
  onOpenResource: (resourceName?: string) => void;
  onRefetchVulnerabilities: () => void;
}

export const useJourneys = (options: UseJourneysOptions) => {
  const [activeJourney, setActiveJourney] = useState<JourneyId | null>(null);
  const [journeyStep, setJourneyStep] = useState(0);
  const [deploymentScenario, setDeploymentScenario] = useState("picker-wheel");
  const [deploymentTier, setDeploymentTier] = useState("tier-2-desktop");
  const [resourceInput, setResourceInput] = useState("");

  const manifestMutation = useMutation<DeploymentManifestResponse, Error, DeploymentManifestRequest>({
    mutationFn: (payload) => generateDeploymentManifest(payload)
  });

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

  const journeySteps = useMemo(
    () =>
      buildJourneySteps(activeJourney, {
        heroStats: options.heroStats,
        orientationData: options.orientationData,
        tierReadiness: options.tierReadiness,
        deploymentScenario,
        deploymentTier,
        resourceInput,
        manifestData: manifestMutation.data,
        manifestIsLoading: manifestMutation.isPending,
        manifestIsError: manifestMutation.isError,
        manifestError: manifestMutation.error ?? undefined,
        topResourceNeedingAttention: options.topResourceNeedingAttention,
        onOpenResource: options.onOpenResource,
        onRefetchVulnerabilities: options.onRefetchVulnerabilities,
        onManifestRequest: handleManifestRequest,
        onSetDeploymentScenario: setDeploymentScenario,
        onSetDeploymentTier: setDeploymentTier,
        onSetResourceInput: setResourceInput
      }),
    [
      activeJourney,
      options.heroStats,
      options.orientationData,
      options.tierReadiness,
      deploymentScenario,
      deploymentTier,
      resourceInput,
      manifestMutation.data,
      manifestMutation.isPending,
      manifestMutation.isError,
      manifestMutation.error,
      options.topResourceNeedingAttention,
      options.onOpenResource,
      options.onRefetchVulnerabilities,
      handleManifestRequest
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
    handleJourneyBack
  };
};
