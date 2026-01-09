import { useQuery } from "@tanstack/react-query";
import { fetchHealth, fetchVaultStatus, fetchCompliance, fetchOrientationSummary } from "../lib/api";

export const useSecretsData = () => {
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 60000
  });

  const vaultQuery = useQuery({
    queryKey: ["vault-status"],
    queryFn: () => fetchVaultStatus(),
    refetchInterval: 60000
  });

  const complianceQuery = useQuery({
    queryKey: ["compliance"],
    queryFn: fetchCompliance,
    refetchInterval: 60000
  });

  const orientationQuery = useQuery({
    queryKey: ["orientation-summary"],
    queryFn: fetchOrientationSummary,
    refetchInterval: 120000
  });

  const isRefreshing =
    healthQuery.isFetching || vaultQuery.isFetching || complianceQuery.isFetching;

  const isInitialLoading =
    healthQuery.isLoading || vaultQuery.isLoading || complianceQuery.isLoading || orientationQuery.isLoading;

  const refreshAll = () => {
    healthQuery.refetch();
    vaultQuery.refetch();
    complianceQuery.refetch();
    orientationQuery.refetch();
  };

  return {
    healthQuery,
    vaultQuery,
    complianceQuery,
    orientationQuery,
    isRefreshing,
    isInitialLoading,
    refreshAll
  };
};
