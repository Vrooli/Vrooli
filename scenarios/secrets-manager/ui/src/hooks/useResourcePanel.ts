import { useCallback, useEffect, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  fetchResourceDetail,
  updateResourceSecret,
  updateSecretStrategy,
  updateVulnerabilityStatus,
  type UpdateResourceSecretPayload,
  type UpdateSecretStrategyPayload,
  type ResourceDetail
} from "../lib/api";

export const useResourcePanel = () => {
  const [activeResource, setActiveResource] = useState<string | null>(null);
  const [selectedSecretKey, setSelectedSecretKey] = useState<string | null>(null);
  const [strategyTier, setStrategyTier] = useState("tier-2-desktop");
  const [strategyHandling, setStrategyHandling] = useState("prompt");
  const [strategyPrompt, setStrategyPrompt] = useState("Desktop pairing prompt");
  const [strategyDescription, setStrategyDescription] = useState("Prompt operator for credentials during install");

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

  const openResourcePanel = useCallback((resourceName?: string, secretKey?: string) => {
    if (!resourceName) return;
    setActiveResource(resourceName);
    setSelectedSecretKey(secretKey ?? null);
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
    const payload: UpdateSecretStrategyPayload = {
      tier: strategyTier,
      handling_strategy: strategyHandling,
      requires_user_input: strategyHandling === "prompt",
      prompt_label: strategyPrompt,
      prompt_description: strategyDescription
    };
    updateStrategyMutation.mutate({ resource: activeResource, secret: selectedSecretKey, payload });
  }, [activeResource, selectedSecretKey, strategyTier, strategyHandling, strategyPrompt, strategyDescription, updateStrategyMutation]);

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
    resourceDetailQuery,
    openResourcePanel,
    closeResourcePanel,
    setSelectedSecretKey,
    setStrategyTier,
    setStrategyHandling,
    setStrategyPrompt,
    setStrategyDescription,
    handleSecretUpdate,
    handleStrategyApply,
    handleVulnerabilityStatus
  };
};
