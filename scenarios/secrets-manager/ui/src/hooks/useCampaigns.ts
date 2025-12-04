import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchCampaigns, type CampaignSummary } from "../lib/api";

export const useCampaigns = () => {
  const [search, setSearch] = useState("");

  const query = useQuery({
    queryKey: ["campaigns"],
    queryFn: () => fetchCampaigns(true),
    staleTime: 5 * 60 * 1000
  });

  const campaigns: CampaignSummary[] = query.data?.campaigns ?? [];

  const filtered = useMemo(() => {
    if (!search.trim()) return campaigns;
    const needle = search.toLowerCase();
    return campaigns.filter((campaign) => campaign.scenario.toLowerCase().includes(needle));
  }, [campaigns, search]);

  return {
    search,
    setSearch,
    campaigns,
    filtered,
    query
  };
};
