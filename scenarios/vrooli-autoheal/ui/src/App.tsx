// Vrooli Autoheal Dashboard
// [REQ:UI-HEALTH-001] [REQ:UI-HEALTH-002] [REQ:UI-EVENTS-001] [REQ:UI-REFRESH-001] [REQ:UI-RESPONSIVE-001]
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState, useMemo, useCallback } from "react";
import { RefreshCw, Play, Shield, AlertCircle, CheckCircle, AlertTriangle, HardDrive, Activity, TrendingUp, LayoutDashboard } from "lucide-react";
import { Button } from "./components/ui/button";
import { fetchStatus, fetchChecks, runTick, groupChecksByStatus, statusToEmoji } from "./lib/api";
import type { CheckInfo, HealthResult, CheckCategory } from "./lib/api";
import { selectors } from "./consts/selectors";
import { StatusBadge, SummaryCard, CheckCard, PlatformInfo, EventsTimeline, UptimeStats, ErrorDisplay, TrendsPage, SystemProtection } from "./components";
import { APIError } from "./lib/api";

const AUTO_REFRESH_INTERVAL = 30000; // 30 seconds

type TabType = "dashboard" | "trends";

// Extended type for checks with metadata
interface EnrichedCheck extends HealthResult {
  title?: string;
  description?: string;
  importance?: string;
  category?: CheckCategory;
  intervalSeconds?: number;
}

// Helper to get tab from URL hash
function getTabFromHash(): TabType {
  const hash = window.location.hash.slice(1);
  if (hash === "trends") return "trends";
  return "dashboard";
}

export default function App() {
  const queryClient = useQueryClient();
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [activeTab, setActiveTab] = useState<TabType>(getTabFromHash);

  // Sync tab state with URL hash
  const handleTabChange = useCallback((tab: TabType) => {
    setActiveTab(tab);
    window.location.hash = tab === "dashboard" ? "" : tab;
  }, []);

  // Listen for hash changes (back/forward navigation)
  useEffect(() => {
    const handleHashChange = () => {
      setActiveTab(getTabFromHash());
    };
    window.addEventListener("hashchange", handleHashChange);
    return () => window.removeEventListener("hashchange", handleHashChange);
  }, []);

  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["status"],
    queryFn: fetchStatus,
    refetchInterval: autoRefresh ? AUTO_REFRESH_INTERVAL : false,
  });

  // Fetch check metadata (description, interval) - doesn't need to refresh as often
  const { data: checksMetadata } = useQuery({
    queryKey: ["checks-metadata"],
    queryFn: fetchChecks,
    staleTime: 60000, // Cache for 60s since check metadata rarely changes
  });

  // Build a lookup map of check metadata
  const checksMetadataMap = useMemo(() => {
    const map: Record<string, CheckInfo> = {};
    if (checksMetadata) {
      for (const check of checksMetadata) {
        map[check.id] = check;
      }
    }
    return map;
  }, [checksMetadata]);

  const tickMutation = useMutation({
    mutationFn: () => runTick(true),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["status"] });
    },
  });

  // Enrich checks with metadata (title, description, importance, category, interval)
  // Note: Must be before early returns to maintain hook order
  const enrichedChecks: EnrichedCheck[] = useMemo(() => {
    const checks = data?.checks || [];
    return checks.map((check) => {
      const metadata = checksMetadataMap[check.checkId];
      return {
        ...check,
        title: metadata?.title,
        description: metadata?.description,
        importance: metadata?.importance,
        category: metadata?.category,
        intervalSeconds: metadata?.intervalSeconds,
      };
    });
  }, [data?.checks, checksMetadataMap]);

  // Use centralized grouping helper for consistent status-based classification
  const { critical: critChecks, warning: warnChecks, ok: okChecks } = groupChecksByStatus(enrichedChecks);

  // Update page title based on status using centralized emoji mapping
  useEffect(() => {
    if (data) {
      const emoji = statusToEmoji(data.status);
      document.title = `${emoji} Autoheal - ${data.status.toUpperCase()}`;
    }
  }, [data?.status]);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-slate-950 text-slate-50 flex items-center justify-center">
        <div className="text-center">
          <RefreshCw className="animate-spin mx-auto mb-4" size={32} />
          <p className="text-slate-400">Loading health status...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-slate-950 text-slate-50 flex items-center justify-center p-6">
        <div className="max-w-md rounded-xl border border-white/10 bg-white/5 p-8">
          <ErrorDisplay
            error={error}
            onRetry={() => refetch()}
            title="Connection Error"
          />
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50" data-testid={selectors.dashboard}>
      {/* Header */}
      <header className="border-b border-white/10 bg-slate-900/50 backdrop-blur sticky top-0 z-10">
        <div className="max-w-6xl mx-auto px-4 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Shield className="text-blue-400" size={28} />
            <div>
              <h1 className="text-xl font-semibold">Vrooli Autoheal</h1>
              <p className="text-xs text-slate-400">Self-healing infrastructure supervisor</p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            {data && <StatusBadge status={data.status} />}
            <SystemProtection compact />

            <Button
              variant="outline"
              size="sm"
              onClick={() => setAutoRefresh(!autoRefresh)}
              className={autoRefresh ? "text-blue-400 border-blue-400/30" : ""}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${isFetching ? "animate-spin" : ""}`} />
              {autoRefresh ? "Auto" : "Manual"}
            </Button>

            <Button
              size="sm"
              onClick={() => tickMutation.mutate()}
              disabled={tickMutation.isPending}
              data-testid={selectors.runTickButton}
            >
              <Play className={`h-4 w-4 mr-2 ${tickMutation.isPending ? "animate-pulse" : ""}`} />
              Run Tick
            </Button>
          </div>
        </div>

        {/* Tab Navigation */}
        <div className="max-w-6xl mx-auto px-4">
          <nav className="flex gap-1">
            <button
              onClick={() => handleTabChange("dashboard")}
              className={`flex items-center gap-2 px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                activeTab === "dashboard"
                  ? "border-blue-400 text-blue-400"
                  : "border-transparent text-slate-400 hover:text-slate-200"
              }`}
              data-testid="autoheal-tab-dashboard"
            >
              <LayoutDashboard size={16} />
              Dashboard
            </button>
            <button
              onClick={() => handleTabChange("trends")}
              className={`flex items-center gap-2 px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                activeTab === "trends"
                  ? "border-blue-400 text-blue-400"
                  : "border-transparent text-slate-400 hover:text-slate-200"
              }`}
              data-testid="autoheal-tab-trends"
            >
              <TrendingUp size={16} />
              Trends
            </button>
          </nav>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-6xl mx-auto px-4 py-6">
        {activeTab === "dashboard" ? (
          <>
            {/* Summary Cards */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
              <SummaryCard
                title="Total Checks"
                value={data?.summary.total || 0}
                icon={HardDrive}
                color="bg-slate-800 text-slate-300"
              />
              <SummaryCard
                title="Healthy"
                value={data?.summary.ok || 0}
                icon={CheckCircle}
                color="bg-emerald-500/20 text-emerald-400"
              />
              <SummaryCard
                title="Warnings"
                value={data?.summary.warning || 0}
                icon={AlertTriangle}
                color="bg-amber-500/20 text-amber-400"
              />
              <SummaryCard
                title="Critical"
                value={data?.summary.critical || 0}
                icon={AlertCircle}
                color="bg-red-500/20 text-red-400"
              />
            </div>

            <div className="grid md:grid-cols-3 gap-6">
              {/* Health Checks */}
              <div className="md:col-span-2 space-y-4">
                <h2 className="text-lg font-medium flex items-center gap-2">
                  <Activity size={20} className="text-blue-400" />
                  Health Checks
                  {enrichedChecks.length === 0 && checksMetadata && checksMetadata.length > 0 && (
                    <span className="text-sm text-slate-500 font-normal">
                      ({checksMetadata.length} registered - click &quot;Run Tick&quot; to execute)
                    </span>
                  )}
                </h2>

                {/* Critical checks first */}
                {critChecks.length > 0 && (
                  <div className="space-y-2">
                    <h3 className="text-sm font-medium text-red-400">Critical Issues</h3>
                    {critChecks.map((check) => (
                      <CheckCard key={check.checkId} check={check} />
                    ))}
                  </div>
                )}

                {/* Warning checks */}
                {warnChecks.length > 0 && (
                  <div className="space-y-2">
                    <h3 className="text-sm font-medium text-amber-400">Warnings</h3>
                    {warnChecks.map((check) => (
                      <CheckCard key={check.checkId} check={check} />
                    ))}
                  </div>
                )}

                {/* OK checks */}
                {okChecks.length > 0 && (
                  <div className="space-y-2">
                    <h3 className="text-sm font-medium text-emerald-400">Healthy</h3>
                    {okChecks.map((check) => (
                      <CheckCard key={check.checkId} check={check} />
                    ))}
                  </div>
                )}
              </div>

              {/* Sidebar */}
              <div className="space-y-4">
                {/* System Protection Status */}
                <SystemProtection />

                {/* Uptime Stats - clickable to go to trends */}
                <UptimeStats onShowTrends={() => handleTabChange("trends")} />

                {data?.platform && <PlatformInfo platform={data.platform} />}

                {/* Last Updated */}
                <div className="rounded-xl border border-white/10 bg-white/5 p-4">
                  <p className="text-sm text-slate-400">
                    Last updated:{" "}
                    <span className="text-slate-200">
                      {data?.timestamp ? new Date(data.timestamp).toLocaleTimeString() : "Never"}
                    </span>
                  </p>
                  {autoRefresh && (
                    <p className="text-xs text-slate-500 mt-1">Auto-refresh every {AUTO_REFRESH_INTERVAL / 1000}s</p>
                  )}
                </div>
              </div>
            </div>

            {/* Events Timeline - Full Width */}
            <div className="mt-6">
              <EventsTimeline />
            </div>
          </>
        ) : (
          <TrendsPage />
        )}
      </main>
    </div>
  );
}
