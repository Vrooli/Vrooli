/**
 * Hook for managing code signing configuration queries and state.
 */

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { signingConnectClient } from "../lib/api/connect";
import {
  presentSigningReadiness,
  signingConfigFromProto,
  type SigningConfig,
  type SigningReadinessResponse,
} from "../domain/signing";

export interface UseSigningConfigOptions {
  scenarioName: string;
}

export interface UseSigningConfigResult {
  // Config query
  config: SigningConfig | null;
  configLoading: boolean;
  refetchConfig: () => void;

  // Readiness query
  readiness: SigningReadinessResponse | undefined;
  readinessLoading: boolean;
  refetchReadiness: () => void;

  // Combined loading state
  loading: boolean;

  // Enabled state for build
  enabledForBuild: boolean;
  setEnabledForBuild: (enabled: boolean) => void;

  // Validation helpers
  isReady: boolean;
  firstIssue: string | undefined;

  // Combined refresh
  refreshAll: () => void;
}

/**
 * Hook for managing signing configuration and readiness checks.
 */
export function useSigningConfig({
  scenarioName,
}: UseSigningConfigOptions): UseSigningConfigResult {
  const [enabledForBuild, setEnabledForBuild] = useState(false);

  // Fetch signing config
  const {
    data: configResp,
    isFetching: configLoading,
    refetch: refetchConfig,
  } = useQuery({
    queryKey: ["signing-config-inline", scenarioName],
    queryFn: () => signingConnectClient.getSigningConfig({ scenarioName }),
    enabled: Boolean(scenarioName),
  });

  // Fetch signing readiness
  const {
    data: readiness,
    isFetching: readinessLoading,
    refetch: refetchReadiness,
  } = useQuery<SigningReadinessResponse>({
    queryKey: ["signing-readiness-inline", scenarioName],
    queryFn: async () =>
      presentSigningReadiness(
        await signingConnectClient.getSigningReadiness({ scenarioName }),
      ),
    enabled: Boolean(scenarioName),
  });

  // Sync enabledForBuild with saved config
  useEffect(() => {
    if (configResp?.config) {
      setEnabledForBuild(configResp.config.enabled);
    } else if (scenarioName) {
      setEnabledForBuild(false);
    }
  }, [scenarioName, configResp?.config]);

  const config = configResp?.config
    ? signingConfigFromProto(configResp.config)
    : null;
  const loading = configLoading || readinessLoading;
  const isReady = Boolean(readiness?.ready);
  const firstIssue = readiness?.issues?.[0];

  const refreshAll = () => {
    if (scenarioName) {
      void refetchConfig();
      void refetchReadiness();
    }
  };

  return {
    config,
    configLoading,
    refetchConfig: () => {
      void refetchConfig();
    },
    readiness,
    readinessLoading,
    refetchReadiness: () => {
      void refetchReadiness();
    },
    loading,
    enabledForBuild,
    setEnabledForBuild,
    isReady,
    firstIssue,
    refreshAll,
  };
}
