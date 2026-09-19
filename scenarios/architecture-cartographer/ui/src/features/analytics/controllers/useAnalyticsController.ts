import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { analyticsClient } from "../../../api/analytics";

export const analyticsKeys = {
  all: () => ["analytics"] as const,
  stats: (scenario: string) => [...analyticsKeys.all(), "stats", scenario] as const,
  events: (scenario: string) => [...analyticsKeys.all(), "events", scenario] as const,
  placements: (scenario: string) => [...analyticsKeys.all(), "placements", scenario] as const,
};

export function useStats(scenario: string) {
  return useQuery({
    queryKey: analyticsKeys.stats(scenario),
    queryFn: () => analyticsClient.getStats({ scenario }),
    enabled: scenario.length > 0,
  });
}

export function useEvents(scenario: string) {
  return useQuery({
    queryKey: analyticsKeys.events(scenario),
    queryFn: () =>
      analyticsClient.listEvents({
        scenario,
        kinds: [],
        pageSize: 100,
        pageToken: "",
      }),
    enabled: scenario.length > 0,
  });
}

export function usePlacements(scenario: string) {
  return useQuery({
    queryKey: analyticsKeys.placements(scenario),
    queryFn: () =>
      analyticsClient.listPlacements({
        scenario,
        outcomes: [],
        pageSize: 100,
        pageToken: "",
      }),
    enabled: scenario.length > 0,
  });
}

export interface RecordOverrideArgs {
  scenario: string;
  chunkId: string;
  verdictDomain: string;
  chosenDomain: string;
  note: string;
  verdictEventId?: string;
}

export function useRecordOverride() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (args: RecordOverrideArgs) =>
      analyticsClient.recordOverride({
        scenario: args.scenario,
        chunkId: args.chunkId,
        verdictDomain: args.verdictDomain,
        chosenDomain: args.chosenDomain,
        note: args.note,
        verdictEventId: args.verdictEventId ?? "",
        idempotencyKey: "",
        dryRun: false,
      }),
    onSuccess: (_d, vars) => {
      void queryClient.invalidateQueries({ queryKey: analyticsKeys.placements(vars.scenario) });
      void queryClient.invalidateQueries({ queryKey: analyticsKeys.events(vars.scenario) });
    },
  });
}
