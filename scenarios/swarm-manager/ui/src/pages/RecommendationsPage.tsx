/**
 * Recommendations Page - Displays system-generated improvement suggestions
 */

import { useMemo, useState, useEffect } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Filter, Zap, Settings, ArrowRight, CheckCircle2, XCircle, RefreshCw, Play } from "lucide-react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ErrorState } from "../components/ui/error-state";
import { ResponsiveList, ResponsiveListItem } from "../components/ui/responsive-list";
import { Select } from "../components/ui/select";
import { selectors } from "../consts/selectors";
import { defaultQueryOptions, formatRelativeTime } from "../lib";
import { agentManagerService, recommendationsService, settingsService } from "../services";
import type { Recommendation, RecommendationStatus } from "../types";
import { useRecommendationsStore } from "../stores";

const STATUS_OPTIONS: RecommendationStatus[] = ["pending", "approved", "rejected"];

export function RecommendationsPage() {
  const [statusFilter, setStatusFilter] = useState<RecommendationStatus | "">("");
  const [showFilters, setShowFilters] = useState(false);
  const recommendations = useRecommendationsStore((state) => state.recommendations);
  const status = useRecommendationsStore((state) => state.status);
  const error = useRecommendationsStore((state) => state.error);
  const fetchRecommendations = useRecommendationsStore((state) => state.fetchRecommendations);
  const hasLoaded = status !== "idle";

  const { data: settings } = useQuery({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });

  const { data: agentStatus } = useQuery({
    queryKey: ["agent-manager-status"],
    queryFn: () => agentManagerService.getStatus(),
    ...defaultQueryOptions,
  });

  useEffect(() => {
    void fetchRecommendations();
  }, [fetchRecommendations]);

  const refreshMutation = useMutation({
    mutationFn: recommendationsService.refresh,
    onSuccess: () => {
      void fetchRecommendations({ force: true });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: RecommendationStatus }) =>
      recommendationsService.updateStatus(id, status),
    onSuccess: () => {
      void fetchRecommendations({ force: true });
    },
  });

  const startMutation = useMutation({
    mutationFn: (id: string) => recommendationsService.start(id),
    onSuccess: () => {
      void fetchRecommendations({ force: true });
    },
  });

  const refreshError = refreshMutation.isError ? "Failed to refresh recommendations. Please try again." : null;
  const updateError = updateMutation.isError ? "Failed to update recommendation. Please try again." : null;
  const startError = startMutation.isError ? "Failed to start recommendation. Please try again." : null;

  const filteredRecommendations = useMemo(() => {
    if (!recommendations) return [];
    let result = recommendations;
    if (statusFilter) {
      result = result.filter((rec) => rec.status === statusFilter);
    }
    return result;
  }, [recommendations, statusFilter]);

  const statusSummary = useMemo(() => {
    if (!recommendations) return { pending: 0, approved: 0, rejected: 0 };
    return {
      pending: recommendations.filter((r) => r.status === "pending").length,
      approved: recommendations.filter((r) => r.status === "approved").length,
      rejected: recommendations.filter((r) => r.status === "rejected").length,
    };
  }, [recommendations]);

  const agentAvailable = agentStatus ? agentStatus.enabled && agentStatus.available : true;
  const agentStatusLabel = agentStatus
    ? agentAvailable
      ? "Agent manager online"
      : "Agent manager offline"
    : "Agent manager status unknown";

  if (settings?.recommendationMode === "off") {
    return (
      <div className="space-y-6" data-testid={selectors.recommendations.page}>
        <Card padding="lg" centered data-testid={selectors.recommendations.empty}>
          <Zap className="mx-auto h-12 w-12 text-slate-600" />
          <h3 className="mt-4 text-lg font-medium text-slate-300">Recommendations are off</h3>
          <p className="mt-2 text-sm text-slate-400">
            Enable the recommendation engine in Settings to start receiving suggestions.
          </p>
          <Link to="/settings" data-testid={selectors.recommendations.settingsLink}>
            <Button variant="outline" className="mt-4 group">
              <Settings className="mr-2 h-4 w-4" />
              Configure Settings
              <ArrowRight className="ml-2 h-4 w-4 opacity-0 transition group-hover:opacity-100" />
            </Button>
          </Link>
        </Card>
      </div>
    );
  }

  if ((status === "loading" || !hasLoaded) && recommendations.length === 0) {
    return (
      <div className="space-y-6" data-testid={selectors.recommendations.page}>
        <Card padding="lg" centered>
          <p className="text-slate-400">Loading recommendations...</p>
        </Card>
      </div>
    );
  }

  if (error && recommendations.length === 0 && hasLoaded) {
    return (
      <div className="space-y-6" data-testid={selectors.recommendations.page}>
        <ErrorState
          error={error}
          title="Unable to load recommendations"
          onRetry={() => fetchRecommendations({ force: true })}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6" data-testid={selectors.recommendations.page}>
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <h2 className="text-xl font-semibold">Recommendations</h2>
        <div className="flex items-center gap-2">
          <div className="hidden items-center gap-2 rounded-full border border-white/10 bg-slate-800 px-3 py-1 text-xs text-slate-400 sm:flex">
            <span
              className={`h-2 w-2 rounded-full ${agentAvailable ? "bg-emerald-400" : "bg-red-400"}`}
              aria-hidden="true"
            />
            <span>{agentStatusLabel}</span>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => refreshMutation.mutate()}
            data-testid={selectors.recommendations.filter}
            disabled={refreshMutation.isPending}
          >
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
          <div className="relative">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowFilters(!showFilters)}
              className={statusFilter ? "border-cyan-500/50" : ""}
              aria-label="Filter recommendations"
            >
              <Filter className="h-4 w-4" />
            </Button>
            {showFilters && (
              <div className="absolute right-0 top-full z-10 mt-2 w-56 rounded-lg border border-white/10 bg-slate-800 p-3 shadow-xl">
                <div className="mb-2 flex items-center justify-between">
                  <span className="text-sm font-medium text-slate-200">Filters</span>
                  {statusFilter && (
                    <button
                      onClick={() => setStatusFilter("")}
                      className="text-xs text-slate-400 hover:text-slate-200"
                    >
                      Clear
                    </button>
                  )}
                </div>
                <div className="space-y-1">
                  <label htmlFor="recommendations-status-filter" className="text-xs text-slate-400">
                    Status
                  </label>
                  <Select
                    id="recommendations-status-filter"
                    value={statusFilter}
                    onChange={(e) => setStatusFilter(e.target.value as RecommendationStatus | "")}
                    variant="filter"
                    withChevron
                  >
                    <option value="">All statuses</option>
                    {STATUS_OPTIONS.map((status) => (
                      <option key={status} value={status}>
                        {status}
                      </option>
                    ))}
                  </Select>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {(refreshError || updateError || startError) && (
        <div className="space-y-2">
          {refreshError && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
              {refreshError}
            </div>
          )}
          {updateError && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
              {updateError}
            </div>
          )}
          {startError && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
              {startError}
            </div>
          )}
        </div>
      )}

      {recommendations.length > 0 && (
        <div className="flex items-center gap-4 text-sm text-slate-400">
          <span>{statusSummary.pending} pending</span>
          <span>{statusSummary.approved} approved</span>
          <span>{statusSummary.rejected} rejected</span>
        </div>
      )}

      <div className="space-y-4">
        {recommendations.length === 0 && hasLoaded && (
          <Card padding="lg" centered data-testid={selectors.recommendations.empty}>
            <Zap className="mx-auto h-12 w-12 text-slate-600" />
            <h3 className="mt-4 text-lg font-medium text-slate-300">No recommendations</h3>
            <p className="mt-2 text-sm text-slate-400">
              The system didn’t find anything to suggest right now.
            </p>
          </Card>
        )}

        {recommendations.length > 0 && filteredRecommendations.length === 0 && (
          <Card padding="lg" centered>
            <Zap className="mx-auto h-12 w-12 text-slate-600" />
            <h3 className="mt-4 text-lg font-medium text-slate-300">No matching recommendations</h3>
            <p className="mt-2 text-sm text-slate-400">Try clearing the filters.</p>
            <Button
              variant="outline"
              size="sm"
              className="mt-4"
              onClick={() => setStatusFilter("")}
            >
              Clear filters
            </Button>
          </Card>
        )}

        {filteredRecommendations.length > 0 && (
          <ResponsiveList
            data-testid={selectors.recommendations.list}
            columns="md:grid-cols-2 xl:grid-cols-3"
          >
            {filteredRecommendations.map((rec) => (
              <RecommendationCard
                key={rec.id}
                recommendation={rec}
                onApprove={() => updateMutation.mutate({ id: rec.id, status: "approved" })}
                onReject={() => updateMutation.mutate({ id: rec.id, status: "rejected" })}
                onStart={() => startMutation.mutate(rec.id)}
                isUpdating={updateMutation.isPending}
                isStarting={startMutation.isPending}
                agentAvailable={agentAvailable}
              />
            ))}
          </ResponsiveList>
        )}
      </div>
    </div>
  );
}

function RecommendationCard({
  recommendation,
  onApprove,
  onReject,
  onStart,
  isUpdating,
  isStarting,
  agentAvailable,
}: {
  recommendation: Recommendation;
  onApprove: () => void;
  onReject: () => void;
  onStart: () => void;
  isUpdating: boolean;
  isStarting: boolean;
  agentAvailable: boolean;
}) {
  const statusColor =
    recommendation.status === "approved"
      ? "text-emerald-400"
      : recommendation.status === "rejected"
        ? "text-red-400"
        : "text-yellow-400";
  const isStarted = Boolean(recommendation.runId || recommendation.taskId);
  const canStart = !isStarted && recommendation.status !== "rejected" && agentAvailable;

  return (
    <ResponsiveListItem
      className="py-4 md:p-6 lg:p-8"
      data-testid={selectors.recommendations.cardByName({ name: recommendation.id })}
    >
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-2">
          <div className="flex items-center gap-3 text-sm">
            <span className={`uppercase tracking-wider ${statusColor}`}>{recommendation.status}</span>
            <span className="text-slate-400">{recommendation.type}</span>
            <span className="rounded-full bg-slate-700 px-2 py-0.5 text-xs text-slate-300">P{recommendation.priority}</span>
          </div>
          <h3 className="text-base font-medium text-slate-100">{recommendation.scenarioName}</h3>
          <p className="text-sm text-slate-400">{recommendation.description}</p>
          <span className="text-xs text-slate-500">{formatRelativeTime(recommendation.created)}</span>
          {isStarted && (
            <div className="text-xs text-slate-500">
              Started {recommendation.startedAt ? formatRelativeTime(recommendation.startedAt) : "recently"}
              {recommendation.startedBy ? ` • ${recommendation.startedBy}` : ""}
            </div>
          )}
        </div>
        <div className="flex flex-col gap-2">
          <Button
            size="sm"
            variant="outline"
            onClick={onStart}
            disabled={isStarting || !canStart}
          >
            <Play className="mr-2 h-4 w-4" />
            {isStarted ? "Started" : "Start"}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={onApprove}
            disabled={isUpdating || recommendation.status === "approved"}
          >
            <CheckCircle2 className="mr-2 h-4 w-4" />
            Approve
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={onReject}
            disabled={isUpdating || recommendation.status === "rejected"}
          >
            <XCircle className="mr-2 h-4 w-4" />
            Reject
          </Button>
        </div>
      </div>
    </ResponsiveListItem>
  );
}
