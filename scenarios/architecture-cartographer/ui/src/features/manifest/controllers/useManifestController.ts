import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { manifestClient } from "../../../api/manifest";

export const manifestKeys = {
  all: () => ["manifest"] as const,
  get: (scenario: string) => [...manifestKeys.all(), "get", scenario] as const,
  domains: (scenario: string) => [...manifestKeys.all(), "domains", scenario] as const,
};

export function useGetManifest(scenario: string, enabled = true) {
  return useQuery({
    queryKey: manifestKeys.get(scenario),
    queryFn: () => manifestClient.getManifest({ scenario }),
    enabled: enabled && scenario.length > 0,
  });
}

export function useListDomains(scenario: string, enabled = true) {
  return useQuery({
    queryKey: manifestKeys.domains(scenario),
    queryFn: () => manifestClient.listDomains({ scenario }),
    enabled: enabled && scenario.length > 0,
  });
}

export function useValidateManifest(scenario: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      manifestClient.validateManifest({
        scenario,
        source: new Uint8Array(),
        contentType: "",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: manifestKeys.get(scenario) });
      void queryClient.invalidateQueries({ queryKey: manifestKeys.domains(scenario) });
    },
  });
}
