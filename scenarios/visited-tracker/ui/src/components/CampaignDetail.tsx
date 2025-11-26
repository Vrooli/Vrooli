import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { ArrowLeft, Clock, FileText, TrendingUp, RefreshCw, AlertCircle, CheckCircle, Target, Terminal } from "lucide-react";
import { fetchCampaign, fetchLeastVisited, fetchMostStale } from "../lib/api";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { HelpButton } from "./HelpButton";
import { FilePathWithCopy } from "./FilePathWithCopy";
import { CliCommand } from "./CliCommand";

interface CampaignDetailProps {
  campaignId: string;
  onBack: () => void;
}

export function CampaignDetail({ campaignId, onBack }: CampaignDetailProps) {
  const { data: campaign, isLoading, error, refetch } = useQuery({
    queryKey: ["campaign", campaignId],
    queryFn: () => fetchCampaign(campaignId)
  });

  const { data: leastVisitedData } = useQuery({
    queryKey: ["leastVisited", campaignId],
    queryFn: () => fetchLeastVisited(campaignId, 5),
    enabled: !!campaign
  });

  const { data: mostStaleData } = useQuery({
    queryKey: ["mostStale", campaignId],
    queryFn: () => fetchMostStale(campaignId, 5),
    enabled: !!campaign
  });

  // Keyboard shortcuts for better UX
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Esc to go back
      if (e.key === 'Escape') {
        onBack();
      }
      // R to refresh (when not in an input)
      if (e.key === 'r' && !e.metaKey && !e.ctrlKey) {
        const target = e.target as HTMLElement;
        if (target.tagName !== 'INPUT' && target.tagName !== 'TEXTAREA') {
          e.preventDefault();
          refetch();
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onBack, refetch]);

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const formatRelativeTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffInSeconds = (now.getTime() - date.getTime()) / 1000;

    if (diffInSeconds < 60) return "just now";
    if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}m ago`;
    if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}h ago`;
    if (diffInSeconds < 604800) return `${Math.floor(diffInSeconds / 86400)}d ago`;
    if (diffInSeconds < 2592000) return `${Math.floor(diffInSeconds / 604800)}w ago`;

    return date.toLocaleDateString();
  };

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-950">
        <div className="text-center max-w-md">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-red-500/10">
            <AlertCircle className="h-8 w-8 text-red-400" />
          </div>
          <h2 className="mb-2 text-xl font-semibold text-slate-50">Failed to Load Campaign</h2>
          <p className="mb-6 text-sm text-slate-400">{(error as Error).message}</p>
          <div className="flex gap-3 justify-center">
            <Button variant="outline" onClick={onBack}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to List
            </Button>
            <Button onClick={() => refetch()}>
              <RefreshCw className="mr-2 h-4 w-4" />
              Try Again
            </Button>
          </div>
          <p className="mt-4 text-xs text-slate-600">Press Esc to go back</p>
        </div>
      </div>
    );
  }

  if (isLoading || !campaign) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-950">
        <div className="text-center" role="status" aria-live="polite" aria-busy="true">
          <div
            className="mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-4 border-slate-700 border-t-slate-50"
            aria-hidden="true"
          />
          <p className="text-slate-300">Loading campaign...</p>
          <span className="sr-only">Please wait while the campaign details are loading</span>
        </div>
      </div>
    );
  }

  const coveragePercent = Math.round(campaign.coverage_percent || 0);
  const leastVisited = leastVisitedData?.files || [];
  const mostStale = mostStaleData?.files || [];

  return (
    <div className="min-h-screen bg-slate-950 p-3 sm:p-4 md:p-6 lg:p-8">
      <a href="#main-content" className="skip-to-content">
        Skip to main content
      </a>
      <div className="mx-auto max-w-6xl">
        {/* Header */}
        <header className="mb-4 sm:mb-6 flex flex-col sm:flex-row sm:flex-wrap items-start sm:items-center gap-2 sm:gap-3 md:gap-4">
          <Button variant="outline" onClick={onBack} aria-label="Go back to campaign list" className="w-full sm:w-auto shrink-0">
            <ArrowLeft className="mr-2 h-4 w-4" aria-hidden="true" />
            Back
          </Button>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 sm:gap-3 flex-wrap">
              <h1 className="text-lg sm:text-xl md:text-2xl lg:text-3xl font-bold text-slate-50 break-words">{campaign.name}</h1>
              <Badge variant={campaign.status} aria-label={`Campaign status: ${campaign.status}`} className="shrink-0">{campaign.status}</Badge>
            </div>
            {campaign.description && (
              <p className="mt-1.5 sm:mt-2 text-xs sm:text-sm md:text-base text-slate-400">{campaign.description}</p>
            )}
          </div>
          <Button onClick={() => refetch()} aria-label="Refresh campaign data" className="w-full sm:w-auto shrink-0">
            <RefreshCw className="mr-2 h-4 w-4" aria-hidden="true" />
            <span className="hidden xs:inline">Refresh</span>
            <span className="xs:hidden">↻</span>
          </Button>
        </header>

        {/* Keyboard shortcuts hint */}
        <div className="mb-3 sm:mb-4 text-[10px] sm:text-xs text-slate-400" role="note" aria-label="Keyboard shortcuts available">
          <kbd className="px-1 sm:px-1.5 py-0.5 rounded bg-slate-800 text-slate-300 text-[10px] sm:text-xs">Esc</kbd> to go back, <kbd className="px-1 sm:px-1.5 py-0.5 rounded bg-slate-800 text-slate-300 text-[10px] sm:text-xs">R</kbd> to refresh
        </div>

        {/* Stats Grid */}
        <main id="main-content">
        <section className="mb-6 sm:mb-8 grid grid-cols-2 gap-2.5 sm:gap-3 md:gap-4 lg:grid-cols-4" role="region" aria-label="Campaign statistics">
          <Card className="hover:bg-white/[0.02] transition-colors">
            <CardHeader className="pb-2 sm:pb-3">
              <CardTitle className="text-xs sm:text-sm font-medium text-slate-400 flex items-center gap-1.5 sm:gap-2">
                <FileText className="h-3 w-3 sm:h-4 sm:w-4" aria-hidden="true" />
                <span className="hidden xs:inline">Total Files</span>
                <span className="xs:hidden">Files</span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-xl sm:text-2xl md:text-3xl font-bold text-slate-50" aria-label={`${campaign.total_files || 0} total files`}>{campaign.total_files || 0}</div>
            </CardContent>
          </Card>

          <Card className="hover:bg-white/[0.02] transition-colors">
            <CardHeader className="pb-2 sm:pb-3">
              <CardTitle className="text-xs sm:text-sm font-medium text-slate-400 flex items-center gap-1.5 sm:gap-2">
                <CheckCircle className="h-3 w-3 sm:h-4 sm:w-4" aria-hidden="true" />
                <span className="hidden xs:inline">Visited</span>
                <span className="xs:hidden">Done</span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-xl sm:text-2xl md:text-3xl font-bold text-green-400" aria-label={`${campaign.visited_files || 0} visited files`}>{campaign.visited_files || 0}</div>
            </CardContent>
          </Card>

          <Card className="hover:bg-white/[0.02] transition-colors">
            <CardHeader className="pb-2 sm:pb-3">
              <CardTitle className="text-xs sm:text-sm font-medium text-slate-400 flex items-center gap-1.5 sm:gap-2">
                <Target className="h-3 w-3 sm:h-4 sm:w-4" aria-hidden="true" />
                Coverage
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-xl sm:text-2xl md:text-3xl font-bold text-blue-400" aria-label={`${coveragePercent}% coverage`}>{coveragePercent}%</div>
            </CardContent>
          </Card>

          <Card className="hover:bg-white/[0.02] transition-colors col-span-2 lg:col-span-1">
            <CardHeader className="pb-2 sm:pb-3">
              <CardTitle className="text-xs sm:text-sm font-medium text-slate-400">Agent</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-base sm:text-lg font-semibold text-slate-50 truncate" title={campaign.from_agent} aria-label={`Created by agent: ${campaign.from_agent}`}>{campaign.from_agent}</div>
            </CardContent>
          </Card>
        </section>

        {/* Coverage Progress */}
        <Card className="mb-6 sm:mb-8">
          <CardHeader>
            <CardTitle className="flex items-center gap-1.5 sm:gap-2 text-sm sm:text-base">
              <TrendingUp className="h-4 w-4 sm:h-5 sm:w-5" aria-hidden="true" />
              Coverage Progress
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div
              className="h-3 sm:h-4 w-full overflow-hidden rounded-full bg-slate-800"
              role="progressbar"
              aria-valuenow={coveragePercent}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-label={`Coverage progress: ${coveragePercent}%`}
            >
              <div
                className="h-full rounded-full bg-gradient-to-r from-green-500 to-blue-500 transition-all duration-500"
                style={{ width: `${coveragePercent}%` }}
              />
            </div>
            <div className="mt-1.5 sm:mt-2 flex justify-between text-xs sm:text-sm text-slate-300">
              <span>
                {campaign.visited_files} of {campaign.total_files} files visited
              </span>
              <span className="font-semibold">{coveragePercent}%</span>
            </div>
          </CardContent>
        </Card>

        {/* CLI Integration - Agent Quick Start */}
        <Card className="mb-6 sm:mb-8 border-blue-500/20 bg-blue-500/5">
          <CardHeader>
            <div className="flex items-center justify-between gap-2">
              <CardTitle className="flex items-center gap-1.5 sm:gap-2 text-sm sm:text-base">
                <Terminal className="h-4 w-4 sm:h-5 sm:w-5 text-blue-400 shrink-0" aria-hidden="true" />
                <span>CLI Commands (for agents)</span>
              </CardTitle>
              <HelpButton content="Copy these commands to interact with this campaign via CLI. All commands use the campaign ID automatically when working with this campaign's files." />
            </div>
          </CardHeader>
          <CardContent className="space-y-3 sm:space-y-4">
            <div>
              <div className="text-xs sm:text-sm font-semibold text-slate-200 mb-2">Get next files to work on:</div>
              <CliCommand command={`visited-tracker least-visited --campaign-id ${campaign.id} --limit 5`} />
            </div>

            <div>
              <div className="text-xs sm:text-sm font-semibold text-slate-200 mb-2">Record file visit with context:</div>
              <CliCommand command={`visited-tracker visit PATH --campaign-id ${campaign.id} --note "Your work description"`} />
            </div>

            <div>
              <div className="text-xs sm:text-sm font-semibold text-slate-200 mb-2">Check most stale files:</div>
              <CliCommand command={`visited-tracker most-stale --campaign-id ${campaign.id} --limit 5`} />
            </div>

            <div>
              <div className="text-xs sm:text-sm font-semibold text-slate-200 mb-2">View coverage stats:</div>
              <CliCommand command={`visited-tracker coverage --campaign-id ${campaign.id} --json`} />
            </div>

            <div className="pt-2 sm:pt-3 border-t border-white/10">
              <div className="text-[10px] sm:text-xs text-slate-400 leading-relaxed">
                <strong className="text-slate-300">Pro tip:</strong> Set <code className="px-1.5 py-0.5 rounded bg-slate-800 text-blue-300 text-[10px] sm:text-xs">export VISITED_TRACKER_CAMPAIGN_ID={campaign.id}</code> in your shell to omit <code className="px-1 py-0.5 rounded bg-slate-800 text-slate-300 text-[10px]">--campaign-id</code> from all commands.
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Actionable Files Section */}
        <div className="mb-6 sm:mb-8 grid grid-cols-1 lg:grid-cols-2 gap-4 sm:gap-6">
          {/* Least Visited Files */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between gap-2">
                <CardTitle className="flex items-center gap-1.5 sm:gap-2 text-sm sm:text-base">
                  <AlertCircle className="h-4 w-4 sm:h-5 sm:w-5 text-orange-400 shrink-0" aria-hidden="true" />
                  <span className="hidden sm:inline">Least Visited Files</span>
                  <span className="sm:hidden">Least Visited</span>
                </CardTitle>
                <HelpButton content="Files that have been visited the fewest times. These are good candidates for your next review iteration. Hover over paths to copy them." />
              </div>
            </CardHeader>
            <CardContent>
              {leastVisited.length > 0 ? (
                <div className="space-y-2">
                  {leastVisited.map((file, i) => (
                    <div
                      key={i}
                      className="rounded-lg border border-orange-500/20 bg-orange-500/5 p-3 hover:bg-orange-500/10 transition-colors"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <FilePathWithCopy path={file.file_path} className="flex-1 min-w-0" />
                        <div className="flex items-center gap-3 text-xs text-slate-500 whitespace-nowrap">
                          <span className="font-semibold text-orange-400">
                            {file.visit_count}x
                          </span>
                        </div>
                      </div>
                      {file.notes && (
                        <div className="mt-2 text-xs text-slate-500 line-clamp-2">
                          Note: {file.notes}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="py-8 text-center text-slate-400" role="status">
                  No files tracked yet
                </div>
              )}
            </CardContent>
          </Card>

          {/* Most Stale Files */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  <Clock className="h-5 w-5 text-red-400" />
                  Most Stale Files
                </CardTitle>
                <HelpButton content="Files with the highest staleness scores based on time since last visit and modification recency. Prioritize these for review. Hover over paths to copy them." />
              </div>
            </CardHeader>
            <CardContent>
              {mostStale.length > 0 ? (
                <div className="space-y-2">
                  {mostStale.map((file, i) => (
                    <div
                      key={i}
                      className="rounded-lg border border-red-500/20 bg-red-500/5 p-3 hover:bg-red-500/10 transition-colors"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <FilePathWithCopy path={file.file_path} className="flex-1 min-w-0" />
                        <div className="flex items-center gap-3 text-xs text-slate-500 whitespace-nowrap">
                          <span className="font-semibold text-red-400">
                            {file.staleness_score?.toFixed(1) ?? 'N/A'}
                          </span>
                        </div>
                      </div>
                      {file.notes && (
                        <div className="mt-2 text-xs text-slate-500 line-clamp-2">
                          Note: {file.notes}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="py-8 text-center text-slate-400" role="status">
                  No stale files found
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Patterns */}
        <Card className="mb-8">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5" />
              File Patterns
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {campaign.patterns.map((pattern, i) => (
                <span
                  key={i}
                  className="rounded-lg border border-white/10 bg-slate-800/60 px-3 py-1.5 font-mono text-sm text-slate-300"
                >
                  {pattern}
                </span>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Recent Visits */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Clock className="h-5 w-5" />
              Recently Visited Files
            </CardTitle>
          </CardHeader>
          <CardContent>
            {campaign.tracked_files && campaign.tracked_files.filter(f => f.last_visited && !f.deleted).length > 0 ? (
              <div className="space-y-2">
                {campaign.tracked_files
                  .filter(f => f.last_visited && !f.deleted)
                  .sort((a, b) => {
                    const aTime = a.last_visited ? new Date(a.last_visited).getTime() : 0;
                    const bTime = b.last_visited ? new Date(b.last_visited).getTime() : 0;
                    return bTime - aTime;
                  })
                  .slice(0, 20)
                  .map((file, i) => (
                  <div
                    key={i}
                    className="flex flex-col sm:flex-row sm:items-center justify-between rounded-lg border border-white/5 bg-white/[0.02] p-3 hover:bg-white/[0.05] gap-2"
                  >
                    <FilePathWithCopy path={file.file_path} className="flex-1 min-w-0" />
                    <div className="flex items-center gap-4 text-xs text-slate-500 flex-wrap">
                      <span>Visits: {file.visit_count}</span>
                      <span>Score: {file.staleness_score?.toFixed(1) ?? 'N/A'}</span>
                      {file.last_visited && (
                        <span>{formatRelativeTime(file.last_visited)}</span>
                      )}
                    </div>
                    {file.notes && (
                      <div className="text-xs text-slate-500 line-clamp-1 sm:max-w-xs">
                        {file.notes}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <div className="py-8 text-center text-slate-400" role="status">
                No visits recorded yet
              </div>
            )}
          </CardContent>
        </Card>

        {/* Metadata */}
        <footer className="mt-6 text-xs text-slate-400 space-y-1" role="contentinfo">
          <div>Created: {formatDate(campaign.created_at)}</div>
          <div>Updated: {formatDate(campaign.updated_at)}</div>
        </footer>
        </main>
      </div>
    </div>
  );
}
