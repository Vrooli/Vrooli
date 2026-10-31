import { queryClient } from "../providers/QueryProvider";
import { apiRequest } from "./useApiTransport";
import { useApiMutation, useApiQuery } from "./useApiQuery";

export interface OrchestrationSettings {
  runExecution: {
    runTimeoutMinutes: number;
    maxConcurrentRuns: number;
    maxTurns: number;
  };
  safetyIsolation: {
    requireSandbox: boolean;
    requireApproval: boolean;
    networkAccess: "none" | "localhost" | "full";
  };
  healthDetection: {
    heartbeatIntervalSeconds: number;
    staleThresholdSeconds: number;
    maxRecoveryAgeSeconds: number;
    reconcilerIntervalSeconds: number;
  };
  processTermination: {
    gracePeriodSeconds: number;
    killProcessGroup: boolean;
    killOrphans: boolean;
    orphanGracePeriodSeconds: number;
    terminationMaxRetries: number;
  };
}

const SETTINGS_KEY = ["orchestration-settings"] as const;

export function useOrchestrationSettings() {
  const query = useApiQuery<OrchestrationSettings>(SETTINGS_KEY, (signal) =>
    apiRequest<OrchestrationSettings>("/orchestration-settings", { signal }));
  // One mutation for both writes, so `error` is always the most recent write's.
  const write = useApiMutation(async (settings?: OrchestrationSettings) => {
    const result = settings
      ? await apiRequest<OrchestrationSettings>("/orchestration-settings", { method: "PUT", body: JSON.stringify(settings) })
      : await apiRequest<OrchestrationSettings>("/orchestration-settings/reset", { method: "POST" });
    queryClient.setQueryData(SETTINGS_KEY, result);
    return result;
  });

  return {
    data: query.data,
    loading: query.loading || write.loading,
    error: write.error ?? query.error,
    refetch: query.refetch,
    updateSettings: (settings: OrchestrationSettings) => write.run(settings),
    resetSettings: () => write.run(),
  };
}
