import { useCallback, useEffect, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  fetchResourceDetail,
  updateResourceSecret,
  updateSecretStrategy,
  updateVulnerabilityStatus,
  setScenarioOverride,
  deleteScenarioOverride,
  fetchScenarioTierOverrides,
  type UpdateResourceSecretPayload,
  type UpdateSecretStrategyPayload,
  type ResourceDetail,
  type SetOverridePayload,
  type ScenarioSecretOverride
} from "../lib/api";

export interface UseResourcePanelOptions {
  selectedScenario?: string;
}

export const useResourcePanel = (options?: UseResourcePanelOptions) => {
  const selectedScenario = options?.selectedScenario;

  const [activeResource, setActiveResource] = useState<string | null>(null);
  const [selectedSecretKey, setSelectedSecretKey] = useState<string | null>(null);
  const [strategyTier, setStrategyTier] = useState("tier-2-desktop");
  const [strategyHandling, setStrategyHandling] = useState("prompt");
  const [strategyPrompt, setStrategyPrompt] = useState("Desktop pairing prompt");
  const [strategyDescription, setStrategyDescription] = useState("Prompt operator for credentials during install");
  const [overrideReason, setOverrideReason] = useState("");
  const [isOverrideMode, setIsOverrideMode] = useState(false);

  const resourceDetailQuery = useQuery<ResourceDetail>({
    queryKey: ["resource-detail", activeResource],
    queryFn: () => fetchResourceDetail(activeResource as string),
    enabled: Boolean(activeResource),
    refetchOnWindowFocus: false
  });

  const updateSecretMutation = useMutation({
    mutationFn: ({ resource, secret, payload }: { resource: string; secret: string; payload: UpdateResourceSecretPayload }) =>
      updateResourceSecret(resource, secret, payload),
    onSuccess: () => {
      resourceDetailQuery.refetch();
    }
  });

  const updateStrategyMutation = useMutation({
    mutationFn: ({ resource, secret, payload }: { resource: string; secret: string; payload: UpdateSecretStrategyPayload }) =>
      updateSecretStrategy(resource, secret, payload),
    onSuccess: () => {
      resourceDetailQuery.refetch();
    }
  });

  const updateVulnerabilityStatusMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: { status: string; assigned_to?: string } }) =>
      updateVulnerabilityStatus(id, payload),
    onSuccess: () => {
      resourceDetailQuery.refetch();
    }
  });

  // Query for scenario overrides (when a scenario is selected)
  const scenarioOverridesQuery = useQuery({
    queryKey: ["scenario-overrides", selectedScenario, strategyTier],
    queryFn: () => fetchScenarioTierOverrides(selectedScenario as string, strategyTier),
    enabled: Boolean(selectedScenario) && Boolean(strategyTier),
    refetchOnWindowFocus: false
  });

  // Mutation for setting scenario overrides
  const setOverrideMutation = useMutation({
    mutationFn: ({ scenario, tier, resource, secret, payload }: {
      scenario: string;
      tier: string;
      resource: string;
      secret: string;
      payload: SetOverridePayload;
    }) => setScenarioOverride(scenario, tier, resource, secret, payload),
    onSuccess: () => {
      scenarioOverridesQuery.refetch();
      resourceDetailQuery.refetch();
    }
  });

  // Mutation for deleting scenario overrides
  const deleteOverrideMutation = useMutation({
    mutationFn: ({ scenario, tier, resource, secret }: {
      scenario: string;
      tier: string;
      resource: string;
      secret: string;
    }) => deleteScenarioOverride(scenario, tier, resource, secret),
    onSuccess: () => {
      scenarioOverridesQuery.refetch();
      resourceDetailQuery.refetch();
    }
  });

  const openResourcePanel = useCallback((resourceName?: string, secretKey?: string, tier?: string) => {
    if (!resourceName) return;
    setActiveResource(resourceName);
    setSelectedSecretKey(secretKey ?? null);
    if (tier) {
      setStrategyTier(tier);
    }
  }, []);

  const closeResourcePanel = useCallback(() => {
    setActiveResource(null);
    setSelectedSecretKey(null);
  }, []);

  const handleSecretUpdate = useCallback(
    (secretKey: string, payload: UpdateResourceSecretPayload) => {
      if (!activeResource) return;
      updateSecretMutation.mutate({ resource: activeResource, secret: secretKey, payload });
    },
    [activeResource, updateSecretMutation]
  );

  const handleStrategyApply = useCallback(() => {
    if (!activeResource || !selectedSecretKey) return;

    // If in override mode and a scenario is selected, create a scenario override
    if (isOverrideMode && selectedScenario) {
      const payload: SetOverridePayload = {
        handling_strategy: strategyHandling,
        requires_user_input: strategyHandling === "prompt",
        prompt_label: strategyHandling === "prompt" ? strategyPrompt : undefined,
        prompt_description: strategyHandling === "prompt" ? strategyDescription : undefined,
        override_reason: overrideReason || undefined
      };
      setOverrideMutation.mutate({
        scenario: selectedScenario,
        tier: strategyTier,
        resource: activeResource,
        secret: selectedSecretKey,
        payload
      });
      return;
    }

    // Default: update the resource-level strategy
    const payload: UpdateSecretStrategyPayload = {
      tier: strategyTier,
      handling_strategy: strategyHandling,
      requires_user_input: strategyHandling === "prompt",
      prompt_label: strategyPrompt,
      prompt_description: strategyDescription
    };
    updateStrategyMutation.mutate({ resource: activeResource, secret: selectedSecretKey, payload });
  }, [activeResource, selectedSecretKey, strategyTier, strategyHandling, strategyPrompt, strategyDescription, overrideReason, isOverrideMode, selectedScenario, updateStrategyMutation, setOverrideMutation]);

  const handleDeleteOverride = useCallback(() => {
    if (!activeResource || !selectedSecretKey || !selectedScenario) return;
    deleteOverrideMutation.mutate({
      scenario: selectedScenario,
      tier: strategyTier,
      resource: activeResource,
      secret: selectedSecretKey
    });
  }, [activeResource, selectedSecretKey, selectedScenario, strategyTier, deleteOverrideMutation]);

  // Find the current override for the selected secret (if any)
  const currentOverride = (scenarioOverridesQuery.data?.overrides ?? []).find(
    (o: ScenarioSecretOverride) => o.resource_name === activeResource && o.secret_key === selectedSecretKey
  );

  const handleVulnerabilityStatus = useCallback(
    (id: string, status: string) => {
      updateVulnerabilityStatusMutation.mutate({ id, payload: { status } });
    },
    [updateVulnerabilityStatusMutation]
  );

  useEffect(() => {
    if (!resourceDetailQuery.data || !resourceDetailQuery.data.secrets.length) {
      setSelectedSecretKey(null);
      return;
    }
    const exists = resourceDetailQuery.data.secrets.some((secret) => secret.secret_key === selectedSecretKey);
    if (!exists) {
      setSelectedSecretKey(resourceDetailQuery.data.secrets[0].secret_key);
    }
  }, [resourceDetailQuery.data, selectedSecretKey]);

  return {
    activeResource,
    selectedSecretKey,
    strategyTier,
    strategyHandling,
    strategyPrompt,
    strategyDescription,
    overrideReason,
    isOverrideMode,
    currentOverride,
    selectedScenario,
    resourceDetailQuery,
    scenarioOverridesQuery,
    openResourcePanel,
    closeResourcePanel,
    setSelectedSecretKey,
    setStrategyTier,
    setStrategyHandling,
    setStrategyPrompt,
    setStrategyDescription,
    setOverrideReason,
    setIsOverrideMode,
    handleSecretUpdate,
    handleStrategyApply,
    handleDeleteOverride,
    handleVulnerabilityStatus
  };
};
