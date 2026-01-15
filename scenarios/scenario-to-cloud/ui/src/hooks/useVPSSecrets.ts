import { useState, useCallback, useEffect, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  listVPSSecrets,
  getVPSSecret,
  createVPSSecret,
  updateVPSSecret,
  deleteVPSSecret,
  type VPSSecretEntry,
  type VPSSecretsMetadata,
  type SecretOperationResponse,
} from "../lib/api";

// Re-export types for convenience
export type { VPSSecretEntry, VPSSecretsMetadata, SecretOperationResponse };

/**
 * Hook to manage VPS secrets for a deployment.
 * Provides CRUD operations and reveal functionality with auto-hide.
 */
export function useVPSSecrets(deploymentId: string | null) {
  const queryClient = useQueryClient();

  // Track which secrets are currently revealed (with auto-hide timers)
  const [revealedSecrets, setRevealedSecrets] = useState<Record<string, string>>({});
  const revealTimers = useRef<Record<string, NodeJS.Timeout>>({});

  // Clear all timers on unmount
  useEffect(() => {
    return () => {
      Object.values(revealTimers.current).forEach(clearTimeout);
    };
  }, []);

  // Fetch all secrets (masked by default)
  const secretsQuery = useQuery({
    queryKey: ["vpsSecrets", deploymentId],
    queryFn: async () => {
      if (!deploymentId) return null;
      return listVPSSecrets(deploymentId);
    },
    enabled: !!deploymentId,
    staleTime: 30000, // Consider stale after 30s
  });

  // Reveal a secret value (with auto-hide after 30 seconds)
  const revealSecret = useCallback(
    async (key: string) => {
      if (!deploymentId) return null;

      // Clear existing timer if any
      if (revealTimers.current[key]) {
        clearTimeout(revealTimers.current[key]);
      }

      try {
        const response = await getVPSSecret(deploymentId, key, true);
        const value = response.secret.value || "";

        setRevealedSecrets((prev) => ({ ...prev, [key]: value }));

        // Auto-hide after 30 seconds
        revealTimers.current[key] = setTimeout(() => {
          setRevealedSecrets((prev) => {
            const next = { ...prev };
            delete next[key];
            return next;
          });
          delete revealTimers.current[key];
        }, 30000);

        return value;
      } catch (error) {
        console.error("Failed to reveal secret:", error);
        throw error;
      }
    },
    [deploymentId]
  );

  // Hide a revealed secret
  const hideSecret = useCallback((key: string) => {
    if (revealTimers.current[key]) {
      clearTimeout(revealTimers.current[key]);
      delete revealTimers.current[key];
    }
    setRevealedSecrets((prev) => {
      const next = { ...prev };
      delete next[key];
      return next;
    });
  }, []);

  // Create a new secret
  const createMutation = useMutation({
    mutationFn: async ({
      key,
      value,
      restartScenario = false,
    }: {
      key: string;
      value: string;
      restartScenario?: boolean;
    }) => {
      if (!deploymentId) throw new Error("No deployment ID");
      return createVPSSecret(deploymentId, key, value, restartScenario);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vpsSecrets", deploymentId] });
    },
  });

  // Update an existing secret
  const updateMutation = useMutation({
    mutationFn: async ({
      key,
      value,
      restartScenario = false,
    }: {
      key: string;
      value: string;
      restartScenario?: boolean;
    }) => {
      if (!deploymentId) throw new Error("No deployment ID");
      return updateVPSSecret(deploymentId, key, value, restartScenario);
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["vpsSecrets", deploymentId] });
      // Clear revealed value if it was revealed
      hideSecret(variables.key);
    },
  });

  // Delete a secret
  const deleteMutation = useMutation({
    mutationFn: async ({
      key,
      restartScenario = false,
    }: {
      key: string;
      restartScenario?: boolean;
    }) => {
      if (!deploymentId) throw new Error("No deployment ID");
      return deleteVPSSecret(deploymentId, key, restartScenario);
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["vpsSecrets", deploymentId] });
      // Clear revealed value if it was revealed
      hideSecret(variables.key);
    },
  });

  // Get secret value (either from revealed cache or masked)
  const getSecretValue = useCallback(
    (key: string): { value: string; masked: boolean } => {
      if (revealedSecrets[key] !== undefined) {
        return { value: revealedSecrets[key], masked: false };
      }
      return { value: "********", masked: true };
    },
    [revealedSecrets]
  );

  // Check if a secret is currently revealed
  const isRevealed = useCallback(
    (key: string) => revealedSecrets[key] !== undefined,
    [revealedSecrets]
  );

  return {
    // Data
    secrets: secretsQuery.data?.secrets ?? [],
    metadata: secretsQuery.data?.metadata ?? null,

    // Loading/error states
    isLoading: secretsQuery.isLoading,
    error: secretsQuery.error,

    // Refetch
    refetch: secretsQuery.refetch,

    // Reveal functionality
    revealSecret,
    hideSecret,
    getSecretValue,
    isRevealed,

    // CRUD mutations
    create: createMutation.mutateAsync,
    update: updateMutation.mutateAsync,
    delete: deleteMutation.mutateAsync,

    // Mutation states
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,
    createError: createMutation.error,
    updateError: updateMutation.error,
    deleteError: deleteMutation.error,
  };
}
