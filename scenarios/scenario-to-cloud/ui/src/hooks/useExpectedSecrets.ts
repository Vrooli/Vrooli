import { useQuery } from "@tanstack/react-query";
import {
  getExpectedSecrets,
  type ExpectedSecret,
  type ExpectedSecretsSummary,
} from "../lib/api";

// Re-export types for convenience
export type { ExpectedSecret, ExpectedSecretsSummary };

/**
 * Hook to fetch expected secrets for a deployment based on the scenario's service.json.
 * This returns what secrets the scenario expects, which can be compared with
 * actual VPS secrets from useVPSSecrets to show configuration status.
 */
export function useExpectedSecrets(deploymentId: string | null, tier?: string) {
  const expectedQuery = useQuery({
    queryKey: ["expectedSecrets", deploymentId, tier],
    queryFn: async () => {
      if (!deploymentId) return null;
      return getExpectedSecrets(deploymentId, tier);
    },
    enabled: !!deploymentId,
    staleTime: 60000, // Secret definitions don't change often
  });

  return {
    // Data
    expectedSecrets: expectedQuery.data?.expected_secrets ?? [],
    scenarioId: expectedQuery.data?.scenario_id ?? null,
    tier: expectedQuery.data?.tier ?? null,
    summary: expectedQuery.data?.summary ?? null,

    // Loading/error states
    isLoading: expectedQuery.isLoading,
    error: expectedQuery.error,

    // Refetch
    refetch: expectedQuery.refetch,
  };
}
