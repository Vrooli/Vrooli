import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ArchitectureFinding } from "@vrooli/proto-types/architecture/v1/findings_pb";
import { RankProfile } from "@vrooli/proto-types/architecture-cartographer/v1/campaign/campaign_pb";

import { campaignClient } from "../../../api/campaign";

export { RankProfile };

/**
 * Stable React Query cache keys for the campaign feature. One key-builder
 * per surface so cache-key drift can't sneak in via inline string assembly.
 */
export const campaignKeys = {
  all: () => ["campaign"] as const,
  list: (scenario: string) => [...campaignKeys.all(), "list", scenario] as const,
  status: (id: string) => [...campaignKeys.all(), "status", id] as const,
  next: (id: string, profile: RankProfile) => [...campaignKeys.all(), "next", id, profile] as const,
};

export interface UseListCampaignsArgs {
  scenario: string;
  enabled?: boolean;
}

/** List the campaigns for one scenario (newest first). */
export function useListCampaigns({ scenario, enabled = true }: UseListCampaignsArgs) {
  return useQuery({
    queryKey: campaignKeys.list(scenario),
    queryFn: () => campaignClient.listCampaigns({ scenario }),
    enabled: enabled && scenario.length > 0,
  });
}

export interface UseCampaignStatusArgs {
  id: string;
  enabled?: boolean;
}

/** Full status projection (campaign + tracked items + rollups). */
export function useCampaignStatus({ id, enabled = true }: UseCampaignStatusArgs) {
  return useQuery({
    queryKey: campaignKeys.status(id),
    queryFn: () => campaignClient.getCampaignStatus({ campaignId: id }),
    enabled: enabled && id.length > 0,
  });
}

/** Prioritized worklist of open items (regressions → cycles → severity). */
export function useNextStep({ id, profile = RankProfile.BALANCED, enabled = true }: UseCampaignStatusArgs & { profile?: RankProfile }) {
  return useQuery({
    queryKey: campaignKeys.next(id, profile),
    queryFn: () => campaignClient.nextCampaignStep({ campaignId: id, profile }),
    enabled: enabled && id.length > 0,
  });
}

export interface CreateCampaignArgs {
  name: string;
  findings: ArchitectureFinding[];
}

/** Open a campaign for the scenario, ingesting the parsed findings. */
export function useCreateCampaign(scenario: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ name, findings }: CreateCampaignArgs) =>
      campaignClient.createCampaign({ scenario, name, findings }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: campaignKeys.list(scenario) });
    },
  });
}

/** Invalidate the per-campaign caches after a lifecycle mutation. */
function invalidateCampaign(queryClient: ReturnType<typeof useQueryClient>, id: string) {
  void queryClient.invalidateQueries({ queryKey: campaignKeys.status(id) });
  // Invalidate all next-step queries for this campaign (any profile)
  void queryClient.invalidateQueries({ queryKey: [...campaignKeys.all(), "next", id] });
}

export interface ResolveItemArgs {
  stableId: string;
  note?: string;
}

export function useResolveItem(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ stableId, note }: ResolveItemArgs) =>
      campaignClient.resolveItem({ campaignId: id, stableId, note: note ?? "" }),
    onSuccess: () => invalidateCampaign(queryClient, id),
  });
}

export function useApplyItem(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (stableId: string) => campaignClient.applyItem({ campaignId: id, stableId }),
    onSuccess: () => invalidateCampaign(queryClient, id),
  });
}

/** Reconcile a fresh audit photograph against the tracked items. */
export function useReauditCampaign(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (findings: ArchitectureFinding[]) =>
      campaignClient.reauditCampaign({ campaignId: id, findings }),
    onSuccess: () => invalidateCampaign(queryClient, id),
  });
}

export function useCloseCampaign(id: string, scenario: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => campaignClient.closeCampaign({ campaignId: id }),
    onSuccess: () => {
      invalidateCampaign(queryClient, id);
      void queryClient.invalidateQueries({ queryKey: campaignKeys.list(scenario) });
    },
  });
}
