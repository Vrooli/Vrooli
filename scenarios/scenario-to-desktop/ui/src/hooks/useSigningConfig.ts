/**
 * Hook for managing code signing configuration queries and state.
 */

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  fetchSigningConfig,
  checkSigningReadiness,
  type SigningConfig,
  type SigningReadinessResponse
} from "../lib/api";

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
  scenarioName
}: UseSigningConfigOptions): UseSigningConfigResult {
  const [enabledForBuild, setEnabledForBuild] = useState(false);

  // Fetch signing config
  const {
    data: configResp,
    isFetching: configLoading,
    refetch: refetchConfig
  } = useQuery({
    queryKey: ["signing-config-inline", scenarioName],
    queryFn: () => fetchSigningConfig(scenarioName),
    enabled: Boolean(scenarioName)
  });

  // Fetch signing readiness
  const {
    data: readiness,
    isFetching: readinessLoading,
    refetch: refetchReadiness
  } = useQuery<SigningReadinessResponse>({
    queryKey: ["signing-readiness-inline", scenarioName],
    queryFn: () => checkSigningReadiness(scenarioName),
    enabled: Boolean(scenarioName)
  });

  // Sync enabledForBuild with saved config
  useEffect(() => {
    if (configResp?.config) {
      setEnabledForBuild(Boolean(configResp.config.enabled));
    } else if (scenarioName) {
      setEnabledForBuild(false);
    }
  }, [scenarioName, configResp?.config]);

  const config = configResp?.config ?? null;
  const loading = configLoading || readinessLoading;
  const isReady = Boolean(readiness?.ready);
  const firstIssue = readiness?.issues?.[0];

  const refreshAll = () => {
    if (scenarioName) {
      refetchConfig();
      refetchReadiness();
    }
  };

  return {
    config,
    configLoading,
    refetchConfig,
    readiness,
    readinessLoading,
    refetchReadiness,
    loading,
    enabledForBuild,
    setEnabledForBuild,
    isReady,
    firstIssue,
    refreshAll
  };
}
