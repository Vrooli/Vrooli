import { useQuery } from "@tanstack/react-query";

import type { GraphSnapshot } from "@vrooli/proto-types/architecture-cartographer/v1/graph/graph_pb";

import { graphClient } from "../../../api/graph";
import { domainsClient } from "../../../api/domains";
import { conflictsClient } from "../../../api/conflicts";

/**
 * Stable React Query cache keys for the graph feature. One key-builder per
 * surface so cache-key drift can't sneak in via inline string assembly.
 */
export const graphKeys = {
  all: () => ["graph"] as const,
  snapshot: (scenario: string, snapshotId: string) =>
    [...graphKeys.all(), "snapshot", scenario, snapshotId] as const,
  latestSnapshot: (scenario: string) =>
    [...graphKeys.all(), "latest", scenario] as const,
  domains: (scenario: string) => [...graphKeys.all(), "domains", scenario] as const,
};

export interface UseGetGraphSnapshotArgs {
  scenario: string;
  /** When omitted, the latest snapshot for the scenario is fetched. */
  snapshotId?: string;
  enabled?: boolean;
}

/**
 * Resolve a graph snapshot. If `snapshotId` is provided, `GetGraphSnapshot`
 * is called directly; otherwise the most recent snapshot is found via
 * `ListGraphSnapshots`. Both code paths return the same `GraphSnapshot`
 * shape so callers don't branch on the source.
 */
export function useGetGraphSnapshot({
  scenario,
  snapshotId,
  enabled = true,
}: UseGetGraphSnapshotArgs) {
  const id = snapshotId ?? "";
  return useQuery({
    queryKey: id.length > 0
      ? graphKeys.snapshot(scenario, id)
      : graphKeys.latestSnapshot(scenario),
    queryFn: async (): Promise<GraphSnapshot | null> => {
      if (id.length > 0) {
        const response = await graphClient.getGraphSnapshot({ id });
        return response.snapshot ?? null;
      }
      const list = await graphClient.listGraphSnapshots({
        scenario,
        pageSize: 1,
        pageToken: "",
      });
      return list.snapshots[0] ?? null;
    },
    enabled: enabled && scenario.length > 0,
  });
}

export interface UseListDomainsArgs {
  scenario: string;
  enabled?: boolean;
}

export function useListDomains({ scenario, enabled = true }: UseListDomainsArgs) {
  return useQuery({
    queryKey: graphKeys.domains(scenario),
    queryFn: () => domainsClient.getDomainMap({ scenario }),
    enabled: enabled && scenario.length > 0,
  });
}

/**
 * Composed loader: pulls the latest graph snapshot + the active conflicts
 * for the same scenario so the canvas overlay matches the workbench.
 *
 * Each underlying query is independently cached so reusing the conflicts
 * data on the workbench page is free.
 */
export function useGraphWorkspace(scenario: string, snapshotId?: string) {
  const snapshot = useGetGraphSnapshot({ scenario, snapshotId });
  const domains = useListDomains({ scenario });
  const conflicts = useQuery({
    queryKey: ["graph", "conflicts", scenario] as const,
    queryFn: () =>
      conflictsClient.listConflicts({
        scenario,
        statuses: [],
        types: [],
        pageSize: 200,
        pageToken: "",
      }),
    enabled: scenario.length > 0,
  });
  return { snapshot, domains, conflicts };
}
