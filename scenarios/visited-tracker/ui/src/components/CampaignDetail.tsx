import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, Clock, FileText, TrendingUp, RefreshCw, AlertCircle, CheckCircle, Target } from "lucide-react";
import { fetchCampaign, fetchLeastVisited, fetchMostStale } from "../lib/api";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { HelpButton } from "./HelpButton";

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
        <div className="text-center">
          <p className="text-red-400">Failed to load campaign</p>
          <p className="mt-2 text-sm text-slate-500">{(error as Error).message}</p>
          <div className="mt-4 flex gap-3 justify-center">
            <Button variant="outline" onClick={onBack}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back
            </Button>
            <Button onClick={() => refetch()}>
              <RefreshCw className="mr-2 h-4 w-4" />
              Retry
            </Button>
          </div>
        </div>
      </div>
    );
  }

  if (isLoading || !campaign) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-950">
        <div className="text-center">
          <div className="mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-4 border-slate-700 border-t-slate-50" />
          <p className="text-slate-400">Loading campaign...</p>
        </div>
      </div>
    );
  }

  const coveragePercent = Math.round(campaign.coverage_percent || 0);
  const leastVisited = leastVisitedData?.files || [];
  const mostStale = mostStaleData?.files || [];

  return (
    <div className="min-h-screen bg-slate-950 p-6">
      <div className="mx-auto max-w-6xl">
        {/* Header */}
        <div className="mb-6 flex flex-wrap items-center gap-4">
          <Button variant="outline" onClick={onBack}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back
          </Button>
          <div className="flex-1 min-w-[200px]">
            <div className="flex items-center gap-3 flex-wrap">
              <h1 className="text-2xl sm:text-3xl font-bold text-slate-50">{campaign.name}</h1>
              <Badge variant={campaign.status}>{campaign.status}</Badge>
            </div>
            {campaign.description && (
              <p className="mt-2 text-slate-400">{campaign.description}</p>
            )}
          </div>
          <Button onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
        </div>

        {/* Stats Grid */}
        <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium text-slate-400 flex items-center gap-2">
                <FileText className="h-4 w-4" />
                Total Files
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold text-slate-50">{campaign.total_files || 0}</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium text-slate-400 flex items-center gap-2">
                <CheckCircle className="h-4 w-4" />
                Visited Files
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold text-green-400">{campaign.visited_files || 0}</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium text-slate-400 flex items-center gap-2">
                <Target className="h-4 w-4" />
                Coverage
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold text-blue-400">{coveragePercent}%</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium text-slate-400">Agent</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-lg font-semibold text-slate-50">{campaign.from_agent}</div>
            </CardContent>
          </Card>
        </div>

        {/* Coverage Progress */}
        <Card className="mb-8">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <TrendingUp className="h-5 w-5" />
              Coverage Progress
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="h-4 w-full overflow-hidden rounded-full bg-slate-800">
              <div
                className="h-full rounded-full bg-gradient-to-r from-green-500 to-blue-500 transition-all duration-500"
                style={{ width: `${coveragePercent}%` }}
              />
            </div>
            <div className="mt-2 flex justify-between text-sm text-slate-400">
              <span>
                {campaign.visited_files} of {campaign.total_files} files visited
              </span>
              <span>{coveragePercent}%</span>
            </div>
          </CardContent>
        </Card>

        {/* Actionable Files Section */}
        <div className="mb-8 grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Least Visited Files */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  <AlertCircle className="h-5 w-5 text-orange-400" />
                  Least Visited Files
                </CardTitle>
                <HelpButton content="Files that have been visited the fewest times. These are good candidates for your next review iteration." />
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
                        <div className="flex-1 min-w-0">
                          <div className="font-mono text-sm text-slate-300 truncate" title={file.path}>
                            {file.path}
                          </div>
                          {file.notes && file.notes.length > 0 && (
                            <div className="mt-1 text-xs text-slate-500 line-clamp-1">
                              {file.notes[file.notes.length - 1]}
                            </div>
                          )}
                        </div>
                        <div className="flex items-center gap-3 text-xs text-slate-500 whitespace-nowrap">
                          <span className="font-semibold text-orange-400">
                            {file.visit_count}x
                          </span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="py-8 text-center text-slate-500">
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
                <HelpButton content="Files with the highest staleness scores based on time since last visit and modification recency. Prioritize these for review." />
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
                        <div className="flex-1 min-w-0">
                          <div className="font-mono text-sm text-slate-300 truncate" title={file.path}>
                            {file.path}
                          </div>
                          {file.notes && file.notes.length > 0 && (
                            <div className="mt-1 text-xs text-slate-500 line-clamp-1">
                              {file.notes[file.notes.length - 1]}
                            </div>
                          )}
                        </div>
                        <div className="flex items-center gap-3 text-xs text-slate-500 whitespace-nowrap">
                          <span className="font-semibold text-red-400">
                            {file.staleness_score.toFixed(1)}
                          </span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="py-8 text-center text-slate-500">
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
              Recent Visits
            </CardTitle>
          </CardHeader>
          <CardContent>
            {campaign.visits && campaign.visits.length > 0 ? (
              <div className="space-y-2">
                {campaign.visits.slice(0, 20).map((visit, i) => (
                  <div
                    key={i}
                    className="flex flex-col sm:flex-row sm:items-center justify-between rounded-lg border border-white/5 bg-white/[0.02] p-3 hover:bg-white/[0.05] gap-2"
                  >
                    <div className="flex-1 min-w-0">
                      <div className="font-mono text-sm text-slate-300 truncate" title={visit.path}>
                        {visit.path}
                      </div>
                      {visit.notes && visit.notes.length > 0 && (
                        <div className="mt-1 text-xs text-slate-500 line-clamp-2">
                          {visit.notes[visit.notes.length - 1]}
                        </div>
                      )}
                    </div>
                    <div className="flex items-center gap-4 text-xs text-slate-500">
                      <span>Visits: {visit.visit_count}</span>
                      <span>Score: {visit.staleness_score.toFixed(1)}</span>
                      {visit.last_visited && (
                        <span>{formatRelativeTime(visit.last_visited)}</span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="py-8 text-center text-slate-500">
                No visits recorded yet
              </div>
            )}
          </CardContent>
        </Card>

        {/* Metadata */}
        <div className="mt-6 text-xs text-slate-600 space-y-1">
          <div>Created: {formatDate(campaign.created_at)}</div>
          <div>Updated: {formatDate(campaign.updated_at)}</div>
        </div>
      </div>
    </div>
  );
}
