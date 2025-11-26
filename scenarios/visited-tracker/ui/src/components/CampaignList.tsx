import { useState, useEffect, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, RefreshCw, Search, Inbox, Terminal, HelpCircle, AlertCircle, X } from "lucide-react";
import { fetchCampaigns, deleteCampaign } from "../lib/api";
import { CampaignCard } from "./CampaignCard";
import { CampaignCardSkeleton } from "./CampaignCardSkeleton";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Badge } from "./ui/badge";
import { CliCommand } from "./CliCommand";
import { useToast } from "./ui/toast";
import { ConfirmDialog } from "./ui/confirm-dialog";

interface CampaignListProps {
  onViewCampaign: (id: string) => void;
  onCreateClick?: () => void;
  onHelpClick?: () => void;
}

export function CampaignList({ onViewCampaign, onCreateClick, onHelpClick }: CampaignListProps) {
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [agentFilter, setAgentFilter] = useState<string>("");
  const [confirmDelete, setConfirmDelete] = useState<{ id: string; name: string } | null>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);

  const queryClient = useQueryClient();
  const { showToast } = useToast();

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["campaigns"],
    queryFn: fetchCampaigns
  });

  // Keyboard shortcuts for better UX
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT') {
        return;
      }

      // N to create new campaign
      if (e.key === 'n' || e.key === 'N') {
        e.preventDefault();
        onCreateClick?.();
      }
      // / to focus search
      else if (e.key === '/') {
        e.preventDefault();
        searchInputRef.current?.focus();
      }
      // R to refresh
      else if (e.key === 'r' || e.key === 'R') {
        e.preventDefault();
        refetch();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onCreateClick, refetch]);

  const deleteMutation = useMutation({
    mutationFn: deleteCampaign,
    onSuccess: (_, campaignId) => {
      queryClient.invalidateQueries({ queryKey: ["campaigns"] });
      showToast("Campaign deleted successfully", "success");
    },
    onError: (error) => {
      showToast(`Failed to delete campaign: ${(error as Error).message}`, "error");
    }
  });

  const handleDelete = (id: string, name: string) => {
    setConfirmDelete({ id, name });
  };

  const handleConfirmDelete = () => {
    if (confirmDelete) {
      deleteMutation.mutate(confirmDelete.id);
      setConfirmDelete(null);
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
      <div className="flex min-h-screen items-center justify-center bg-slate-950 p-4">
        <div className="text-center max-w-md" role="alert" aria-live="assertive">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-red-500/10">
            <AlertCircle className="h-8 w-8 text-red-400" aria-hidden="true" />
          </div>
          <h2 className="text-xl font-semibold text-red-400 mb-2">Failed to load campaigns</h2>
          <p className="mt-2 text-sm text-slate-300 mb-4">{(error as Error).message}</p>
          <div className="rounded-lg border border-slate-700/50 bg-slate-800/30 p-4 mb-4 text-left">
            <div className="text-xs text-slate-300 space-y-2">
              <div>
                <strong className="text-slate-200">Troubleshooting steps:</strong>
              </div>
              <ol className="list-decimal list-inside space-y-1 text-slate-400">
                <li>Check if the visited-tracker API is running</li>
                <li>Verify the API port matches the expected value</li>
                <li>Try refreshing your browser</li>
                <li>Check the browser console for detailed errors</li>
              </ol>
            </div>
          </div>
          <Button className="mt-2" onClick={() => refetch()} aria-label="Retry loading campaigns">
            <RefreshCw className="mr-2 h-4 w-4" aria-hidden="true" />
            Retry
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-950 p-3 sm:p-4 md:p-6 lg:p-8">
      <a href="#main-content" className="skip-to-content">
        Skip to main content
      </a>
      <div className="mx-auto max-w-7xl">
        {/* Header */}
        <header className="mb-4 sm:mb-6 md:mb-8">
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1">
              <h1 className="text-xl sm:text-2xl md:text-3xl lg:text-4xl font-bold text-slate-50">Visited Tracker</h1>
              <p className="mt-1.5 sm:mt-2 text-xs sm:text-sm md:text-base text-slate-300">
                Manage file visit tracking campaigns across your codebase
              </p>
              <div className="mt-2 flex items-center gap-2 flex-wrap">
                <span className="text-[10px] sm:text-xs text-slate-500" role="note" aria-label="Keyboard shortcuts available">
                  <kbd className="px-1 sm:px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 text-[10px] sm:text-xs">N</kbd> new, <kbd className="px-1 sm:px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 text-[10px] sm:text-xs">/</kbd> search, <kbd className="px-1 sm:px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 text-[10px] sm:text-xs">R</kbd> refresh
                </span>
                {onHelpClick && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={onHelpClick}
                    className="h-6 px-2 text-[10px] sm:text-xs text-slate-400 hover:text-slate-200"
                    aria-label="View all keyboard shortcuts"
                  >
                    <HelpCircle className="h-3 w-3 mr-1" aria-hidden="true" />
                    All shortcuts
                  </Button>
                )}
              </div>
            </div>
          </div>
        </header>

        {/* Stats */}
        <div className="mb-4 sm:mb-6 md:mb-8 grid grid-cols-1 gap-2.5 sm:gap-3 md:gap-4 sm:grid-cols-3" role="region" aria-label="Campaign statistics">
          <article className="rounded-lg sm:rounded-xl border border-white/10 bg-white/5 p-3 sm:p-4 md:p-5 hover:bg-white/[0.07] transition-colors focus-within:ring-2 focus-within:ring-blue-400/50" tabIndex={0} aria-labelledby="stat-total" role="group">
            <div id="stat-total" className="text-xl sm:text-2xl md:text-3xl font-bold text-slate-50">
              {totalCampaigns}
            </div>
            <div className="text-[10px] sm:text-xs md:text-sm text-slate-300 mt-0.5 sm:mt-1" aria-label="Total number of campaigns">Total Campaigns</div>
          </article>
          <article className="rounded-lg sm:rounded-xl border border-white/10 bg-white/5 p-3 sm:p-4 md:p-5 hover:bg-white/[0.07] transition-colors focus-within:ring-2 focus-within:ring-blue-400/50" tabIndex={0} aria-labelledby="stat-active" role="group">
            <div id="stat-active" className="text-xl sm:text-2xl md:text-3xl font-bold text-green-400">
              {activeCampaigns}
            </div>
            <div className="text-[10px] sm:text-xs md:text-sm text-slate-300 mt-0.5 sm:mt-1" aria-label="Number of active campaigns">Active Campaigns</div>
          </article>
          <article className="rounded-lg sm:rounded-xl border border-white/10 bg-white/5 p-3 sm:p-4 md:p-5 hover:bg-white/[0.07] transition-colors focus-within:ring-2 focus-within:ring-blue-400/50" tabIndex={0} aria-labelledby="stat-files" role="group">
            <div id="stat-files" className="text-xl sm:text-2xl md:text-3xl font-bold text-blue-400">
              {totalFiles.toLocaleString()}
            </div>
            <div className="text-[10px] sm:text-xs md:text-sm text-slate-300 mt-0.5 sm:mt-1" aria-label="Total number of tracked files">Tracked Files</div>
          </article>
        </div>

        {/* Controls */}
        <div className="mb-4 sm:mb-6 flex flex-col sm:flex-row sm:flex-wrap items-stretch sm:items-center gap-2 sm:gap-3" role="toolbar" aria-label="Campaign controls">
          <Button onClick={onCreateClick} className="w-full sm:w-auto order-1" size="default">
            <Plus className="mr-2 h-4 w-4" aria-hidden="true" />
            <span className="hidden xs:inline">New Campaign</span>
            <span className="xs:hidden">New</span>
          </Button>

          <div className="relative flex-1 min-w-[180px] sm:min-w-[200px] max-w-md order-3 sm:order-2">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400 pointer-events-none" aria-hidden="true" />
            <Input
              ref={searchInputRef}
              placeholder="Search campaigns..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="pl-9 h-10 sm:h-11 text-sm"
              aria-label="Search campaigns"
            />
          </div>

          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="h-10 sm:h-11 rounded-lg border border-white/10 bg-white/5 px-3 sm:px-4 text-xs sm:text-sm text-slate-50 focus-visible:border-white/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/20 w-full sm:w-auto order-4 sm:order-3"
            aria-label="Filter by status"
          >
            <option value="">All Status</option>
            <option value="active">Active</option>
            <option value="completed">Completed</option>
            <option value="archived">Archived</option>
          </select>

          <select
            value={agentFilter}
            onChange={(e) => setAgentFilter(e.target.value)}
            className="h-10 sm:h-11 rounded-lg border border-white/10 bg-white/5 px-3 sm:px-4 text-xs sm:text-sm text-slate-50 focus-visible:border-white/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/20 w-full sm:w-auto order-5 sm:order-4"
            aria-label="Filter by agent"
          >
            <option value="">All Agents</option>
            {uniqueAgents.map((agent) => (
              <option key={agent} value={agent}>
                {agent}
              </option>
            ))}
          </select>

          <Button
            variant="outline"
            onClick={() => refetch()}
            aria-label="Refresh campaign list"
            className="w-full sm:w-auto order-2 sm:order-5"
            size="default"
            disabled={isLoading}
          >
            <RefreshCw className={`mr-2 h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} aria-hidden="true" />
            <span className="hidden xs:inline">{isLoading ? 'Loading...' : 'Refresh'}</span>
            <span className="xs:hidden">↻</span>
          </Button>
        </div>

        {/* Active Filters */}
        {(searchTerm || statusFilter || agentFilter) && (
          <div className="mb-4 flex flex-wrap items-center gap-2" role="status" aria-live="polite" aria-label="Active filters">
            <span className="text-xs text-slate-400">Active filters:</span>
            {searchTerm && (
              <Badge variant="secondary" className="flex items-center gap-1 px-2 py-1">
                <span className="text-xs">Search: {searchTerm}</span>
                <button
                  onClick={() => setSearchTerm("")}
                  className="ml-1 hover:bg-white/10 rounded-full p-0.5"
                  aria-label="Clear search filter"
                >
                  <X className="h-3 w-3" aria-hidden="true" />
                </button>
              </Badge>
            )}
            {statusFilter && (
              <Badge variant="secondary" className="flex items-center gap-1 px-2 py-1">
                <span className="text-xs">Status: {statusFilter}</span>
                <button
                  onClick={() => setStatusFilter("")}
                  className="ml-1 hover:bg-white/10 rounded-full p-0.5"
                  aria-label="Clear status filter"
                >
                  <X className="h-3 w-3" aria-hidden="true" />
                </button>
              </Badge>
            )}
            {agentFilter && (
              <Badge variant="secondary" className="flex items-center gap-1 px-2 py-1">
                <span className="text-xs">Agent: {agentFilter}</span>
                <button
                  onClick={() => setAgentFilter("")}
                  className="ml-1 hover:bg-white/10 rounded-full p-0.5"
                  aria-label="Clear agent filter"
                >
                  <X className="h-3 w-3" aria-hidden="true" />
                </button>
              </Badge>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setSearchTerm("");
                setStatusFilter("");
                setAgentFilter("");
              }}
              className="h-auto py-1 px-2 text-xs"
              aria-label="Clear all filters"
            >
              Clear all
            </Button>
          </div>
        )}

        {/* Content */}
        <main id="main-content">
        {isLoading ? (
          <div className="grid grid-cols-1 gap-3 sm:gap-4 md:gap-5 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3" role="status" aria-live="polite" aria-busy="true">
            <span className="sr-only">Loading campaigns...</span>
            {[...Array(6)].map((_, i) => (
              <CampaignCardSkeleton key={i} />
            ))}
          </div>
        ) : filteredCampaigns.length === 0 ? (
          <div className="flex items-center justify-center py-8 sm:py-12">
            <div className="max-w-2xl text-center px-4">
              <Inbox className="mx-auto mb-4 sm:mb-6 h-12 w-12 sm:h-16 sm:w-16 text-slate-600" aria-hidden="true" />
              <h3 className="mb-2 sm:mb-3 text-lg sm:text-xl md:text-2xl font-semibold text-slate-50">
                {campaigns.length === 0 ? "Welcome to Visited Tracker" : "No Matching Campaigns"}
              </h3>

              {campaigns.length === 0 ? (
                <>
                  <p className="mb-4 sm:mb-6 text-sm sm:text-base text-slate-400 leading-relaxed max-w-xl mx-auto">
                    Visited Tracker helps agents maintain perfect memory across conversations by tracking file visits with staleness detection.
                  </p>

                  <div className="mb-6 sm:mb-8 rounded-lg sm:rounded-xl border border-blue-500/20 bg-blue-500/5 p-4 sm:p-6 text-left max-w-3xl mx-auto">
                    <div className="flex items-center gap-2 mb-3 sm:mb-4">
                      <Terminal className="h-4 w-4 sm:h-5 sm:w-5 text-blue-400 shrink-0" aria-hidden="true" />
                      <h4 className="text-sm sm:text-base font-semibold text-slate-50">
                        Quick Start for Agents
                      </h4>
                    </div>
                    <div className="space-y-3 sm:space-y-4" role="list" aria-label="Getting started steps">
                      <div role="listitem">
                        <div className="text-xs sm:text-sm font-medium text-slate-200 mb-2 flex items-center gap-2">
                          <span className="flex items-center justify-center w-5 h-5 rounded-full bg-blue-500/20 text-blue-300 text-xs font-bold" aria-hidden="true">1</span>
                          Get files to work on (auto-creates campaign):
                        </div>
                        <CliCommand command='visited-tracker least-visited --location scenarios/X/ui --pattern "**/*.tsx" --tag ux --limit 5' />
                      </div>
                      <div role="listitem">
                        <div className="text-xs sm:text-sm font-medium text-slate-200 mb-2 flex items-center gap-2">
                          <span className="flex items-center justify-center w-5 h-5 rounded-full bg-blue-500/20 text-blue-300 text-xs font-bold" aria-hidden="true">2</span>
                          Record your work with context:
                        </div>
                        <CliCommand command='visited-tracker visit src/App.tsx --tag ux --note "Improved accessibility"' />
                      </div>
                      <div role="listitem">
                        <div className="text-xs sm:text-sm font-medium text-slate-200 mb-2 flex items-center gap-2">
                          <span className="flex items-center justify-center w-5 h-5 rounded-full bg-blue-500/20 text-blue-300 text-xs font-bold" aria-hidden="true">3</span>
                          Continue with least-visited files:
                        </div>
                        <CliCommand command='visited-tracker least-visited --tag ux --limit 3' />
                      </div>
                      <div className="pt-2 sm:pt-3 border-t border-white/10">
                        <div className="text-[10px] sm:text-xs text-slate-400 leading-relaxed">
                          <strong className="text-slate-300">Zero friction:</strong> Campaigns auto-create based on <code className="px-1 py-0.5 rounded bg-slate-800 text-blue-300">--location</code> + <code className="px-1 py-0.5 rounded bg-slate-800 text-blue-300">--tag</code>. No manual setup needed.
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="flex flex-col sm:flex-row gap-2 sm:gap-3 justify-center">
                    <Button onClick={onCreateClick} variant="outline" size="lg" className="w-full sm:w-auto" aria-label="Create a campaign manually (optional)">
                      <Plus className="mr-2 h-4 w-4" aria-hidden="true" />
                      Create Campaign Manually
                    </Button>
                  </div>

                  <p className="mt-4 sm:mt-6 text-[10px] sm:text-xs text-slate-500" role="note">
                    Manual creation is optional - most agents use CLI auto-creation for zero-friction workflows.
                  </p>
                </>
              ) : (
                <div className="space-y-4">
                  <p className="mb-4 sm:mb-6 text-sm sm:text-base text-slate-400">
                    No campaigns match your current filters
                  </p>
                  <div className="rounded-lg border border-slate-700/50 bg-slate-800/30 p-4 text-left">
                    <div className="text-xs sm:text-sm text-slate-300 space-y-2">
                      <div>
                        <strong className="text-slate-200">Active filters:</strong>
                      </div>
                      <ul className="list-disc list-inside space-y-1 text-slate-400">
                        {searchTerm && <li>Search: "{searchTerm}"</li>}
                        {statusFilter && <li>Status: {statusFilter}</li>}
                        {agentFilter && <li>Agent: {agentFilter}</li>}
                      </ul>
                    </div>
                  </div>
                  <Button variant="outline" onClick={() => {
                    setSearchTerm("");
                    setStatusFilter("");
                    setAgentFilter("");
                  }} aria-label="Clear all filters to show all campaigns">
                    Clear All Filters
                  </Button>
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-3 sm:gap-4 md:gap-5 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3">
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
        </main>
      </div>

      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={(open) => !open && setConfirmDelete(null)}
        title="Delete Campaign"
        description={
          <>
            Are you sure you want to delete campaign <strong>"{confirmDelete?.name}"</strong>?
            <br /><br />
            This will permanently delete all visit tracking data for this campaign.
          </>
        }
        confirmLabel="Delete Campaign"
        onConfirm={handleConfirmDelete}
        destructive
      />
    </div>
  );
}
