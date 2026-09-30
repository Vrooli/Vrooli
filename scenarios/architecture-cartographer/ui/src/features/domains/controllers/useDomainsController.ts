import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { domainsClient } from "../../../api/domains";
import { signalsClient } from "../../../api/signals";

export const domainsKeys = {
  all: () => ["domains"] as const,
  get: (scenario: string) => [...domainsKeys.all(), "get", scenario] as const,
  convergence: (scenario: string) => [...domainsKeys.all(), "convergence", scenario] as const,
  boundaries: (scenario: string) => [...domainsKeys.all(), "boundaries", scenario] as const,
};

export function useGetDomainMap(scenario: string, enabled = true) {
  return useQuery({
    queryKey: domainsKeys.get(scenario),
    queryFn: () => domainsClient.getDomainMap({ scenario }),
    enabled: enabled && scenario.length > 0,
  });
}

export function useConvergenceReport(scenario: string, enabled = true) {
  return useQuery({
    queryKey: domainsKeys.convergence(scenario),
    queryFn: () => domainsClient.convergenceReport({ scenario }),
    enabled: enabled && scenario.length > 0,
  });
}

export function useBoundaryHealth(scenario: string, enabled = true) {
  return useQuery({
    queryKey: domainsKeys.boundaries(scenario),
    queryFn: () => signalsClient.boundaryHealth({ scenario }),
    enabled: enabled && scenario.length > 0,
  });
}

export function useExtractDomains(scenario: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => domainsClient.extractDomains({ scenario }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: domainsKeys.get(scenario) });
      void queryClient.invalidateQueries({ queryKey: domainsKeys.convergence(scenario) });
      void queryClient.invalidateQueries({ queryKey: domainsKeys.boundaries(scenario) });
    },
  });
}
