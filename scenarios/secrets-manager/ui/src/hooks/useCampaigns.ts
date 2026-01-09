import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchCampaigns, type CampaignSummary } from "../lib/api";

export const useCampaigns = (selectedScenario?: string) => {
  const [search, setSearch] = useState("");

  const baseQuery = useQuery({
    queryKey: ["campaigns", "base"],
    queryFn: () => fetchCampaigns({ includeReadiness: false }),
    staleTime: 5 * 60 * 1000
  });

  const readinessQuery = useQuery({
    enabled: !!selectedScenario,
    queryKey: ["campaigns", "readiness", selectedScenario],
    queryFn: () => fetchCampaigns({ includeReadiness: true, scenario: selectedScenario }),
    staleTime: 60 * 1000
  });

  const campaigns: CampaignSummary[] = baseQuery.data?.campaigns ?? [];
  const readinessCampaigns: CampaignSummary[] = readinessQuery.data?.campaigns ?? [];

  const campaignsById = new Map<string, CampaignSummary>();
  campaigns.forEach((c) => campaignsById.set(c.id, c));
  readinessCampaigns.forEach((c) => campaignsById.set(c.id, c));
  const merged = Array.from(campaignsById.values());

  const filtered = useMemo(() => {
    if (!search.trim()) return merged;
    const needle = search.toLowerCase();
    return merged.filter((campaign) => campaign.scenario.toLowerCase().includes(needle));
  }, [merged, search]);

  return {
    search,
    setSearch,
    campaigns: merged,
    filtered,
    query: baseQuery,
    readinessQuery
  };
};
