import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, RefreshCw, Search, Inbox } from "lucide-react";
import { fetchCampaigns, deleteCampaign } from "../lib/api";
import { CampaignCard } from "./CampaignCard";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

interface CampaignListProps {
  onViewCampaign: (id: string) => void;
  onCreateClick?: () => void;
}

export function CampaignList({ onViewCampaign, onCreateClick }: CampaignListProps) {
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [agentFilter, setAgentFilter] = useState<string>("");

  const queryClient = useQueryClient();

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["campaigns"],
    queryFn: fetchCampaigns
  });

  const deleteMutation = useMutation({
    mutationFn: deleteCampaign,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["campaigns"] });
    }
  });

  const handleDelete = (id: string, name: string) => {
    if (
      confirm(
        `Are you sure you want to delete campaign "${name}"?\n\nThis will permanently delete all visit tracking data for this campaign.`
      )
    ) {
      deleteMutation.mutate(id);
    }
  };

  const campaigns = data?.campaigns || [];

  // Extract unique agents for filter
  const uniqueAgents = Array.from(new Set(campaigns.map((c) => c.from_agent))).sort();

  // Apply filters
  const filteredCampaigns = campaigns.filter((campaign) => {
    const matchesSearch =
      !searchTerm ||
      campaign.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (campaign.description && campaign.description.toLowerCase().includes(searchTerm.toLowerCase()));

    const matchesStatus = !statusFilter || campaign.status === statusFilter;
    const matchesAgent = !agentFilter || campaign.from_agent === agentFilter;

    return matchesSearch && matchesStatus && matchesAgent;
  });

  // Calculate stats
  const totalCampaigns = campaigns.length;
  const activeCampaigns = campaigns.filter((c) => c.status === "active").length;
  const totalFiles = campaigns.reduce((sum, c) => sum + (c.total_files || 0), 0);

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-center">
          <p className="text-red-400">Failed to load campaigns</p>
          <p className="mt-2 text-sm text-slate-500">{(error as Error).message}</p>
          <Button className="mt-4" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Retry
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-950 p-6">
      <div className="mx-auto max-w-7xl">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-slate-50">Visited Tracker</h1>
          <p className="mt-2 text-slate-400">
            Manage file visit tracking campaigns across your codebase
          </p>
        </div>

        {/* Stats */}
        <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div className="rounded-xl border border-white/10 bg-white/5 p-4">
            <div className="text-2xl font-bold text-slate-50">{totalCampaigns}</div>
            <div className="text-sm text-slate-400">Total Campaigns</div>
          </div>
          <div className="rounded-xl border border-white/10 bg-white/5 p-4">
            <div className="text-2xl font-bold text-green-400">{activeCampaigns}</div>
            <div className="text-sm text-slate-400">Active Campaigns</div>
          </div>
          <div className="rounded-xl border border-white/10 bg-white/5 p-4">
            <div className="text-2xl font-bold text-blue-400">{totalFiles.toLocaleString()}</div>
            <div className="text-sm text-slate-400">Tracked Files</div>
          </div>
        </div>

        {/* Controls */}
        <div className="mb-6 flex flex-wrap items-center gap-3">
          <Button onClick={onCreateClick}>
            <Plus className="mr-2 h-4 w-4" />
            New Campaign
          </Button>

          <div className="relative flex-1 min-w-[200px] max-w-md">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
            <Input
              placeholder="Search campaigns..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="pl-10"
            />
          </div>

          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="h-11 rounded-lg border border-white/10 bg-white/5 px-4 text-sm text-slate-50 focus-visible:border-white/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/20"
          >
            <option value="">All Status</option>
            <option value="active">Active</option>
            <option value="completed">Completed</option>
            <option value="archived">Archived</option>
          </select>

          <select
            value={agentFilter}
            onChange={(e) => setAgentFilter(e.target.value)}
            className="h-11 rounded-lg border border-white/10 bg-white/5 px-4 text-sm text-slate-50 focus-visible:border-white/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/20"
          >
            <option value="">All Agents</option>
            {uniqueAgents.map((agent) => (
              <option key={agent} value={agent}>
                {agent}
              </option>
            ))}
          </select>

          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
        </div>

        {/* Content */}
        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <div className="text-center">
              <div className="mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-4 border-slate-700 border-t-slate-50" />
              <p className="text-slate-400">Loading campaigns...</p>
            </div>
          </div>
        ) : filteredCampaigns.length === 0 ? (
          <div className="flex items-center justify-center py-12">
            <div className="max-w-2xl text-center">
              <Inbox className="mx-auto mb-6 h-16 w-16 text-slate-600" />
              <h3 className="mb-3 text-2xl font-semibold text-slate-50">
                {campaigns.length === 0 ? "Welcome to Visited Tracker" : "No Matching Campaigns"}
              </h3>

              {campaigns.length === 0 ? (
                <>
                  <p className="mb-6 text-slate-400 leading-relaxed">
                    Visited Tracker helps agents maintain perfect memory across conversations by tracking file visits with staleness detection.
                  </p>

                  <div className="mb-8 rounded-xl border border-white/10 bg-white/5 p-6 text-left">
                    <h4 className="mb-3 text-sm font-semibold text-slate-50 uppercase tracking-wide">
                      Quick Start for Agents
                    </h4>
                    <div className="space-y-3 text-sm text-slate-300">
                      <div>
                        <div className="font-medium text-slate-200 mb-1">1. Auto-create campaigns with CLI:</div>
                        <code className="block bg-slate-900/80 rounded px-3 py-2 font-mono text-xs text-blue-300">
                          visited-tracker least-visited --location scenarios/X/ui --pattern "**/*.tsx" --tag ux --limit 5
                        </code>
                      </div>
                      <div>
                        <div className="font-medium text-slate-200 mb-1">2. Record work with notes:</div>
                        <code className="block bg-slate-900/80 rounded px-3 py-2 font-mono text-xs text-blue-300">
                          visited-tracker visit src/App.tsx --tag ux --note "Improved accessibility"
                        </code>
                      </div>
                      <div>
                        <div className="font-medium text-slate-200 mb-1">3. Get next files to work on:</div>
                        <code className="block bg-slate-900/80 rounded px-3 py-2 font-mono text-xs text-blue-300">
                          visited-tracker least-visited --location scenarios/X --tag refactor --limit 3
                        </code>
                      </div>
                    </div>
                  </div>

                  <div className="flex gap-3 justify-center">
                    <Button onClick={onCreateClick} size="lg">
                      <Plus className="mr-2 h-4 w-4" />
                      Create Campaign Manually
                    </Button>
                  </div>

                  <p className="mt-6 text-xs text-slate-500">
                    Campaigns are typically auto-created via CLI. Manual creation is useful for testing or non-agent workflows.
                  </p>
                </>
              ) : (
                <>
                  <p className="mb-6 text-slate-400">
                    Try adjusting your search or filters
                  </p>
                </>
              )}
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
            {filteredCampaigns.map((campaign) => (
              <CampaignCard
                key={campaign.id}
                campaign={campaign}
                onView={onViewCampaign}
                onDelete={handleDelete}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
