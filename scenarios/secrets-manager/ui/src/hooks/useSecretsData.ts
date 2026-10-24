import { useQuery } from "@tanstack/react-query";
import { fetchHealth, fetchCredentialCoverage, fetchCompliance, fetchOrientationSummary } from "../lib/api";

export const useSecretsData = () => {
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 60000
  });

  const credentialQuery = useQuery({
    queryKey: ["credential-coverage"],
    queryFn: () => fetchCredentialCoverage(),
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
    healthQuery.isFetching || credentialQuery.isFetching || complianceQuery.isFetching;

  const isInitialLoading =
    healthQuery.isLoading || credentialQuery.isLoading || complianceQuery.isLoading || orientationQuery.isLoading;

  const refreshAll = () => {
    healthQuery.refetch();
    credentialQuery.refetch();
    complianceQuery.refetch();
    orientationQuery.refetch();
  };

  return {
    healthQuery,
    credentialQuery,
    complianceQuery,
    orientationQuery,
    isRefreshing,
    isInitialLoading,
    refreshAll
  };
};
