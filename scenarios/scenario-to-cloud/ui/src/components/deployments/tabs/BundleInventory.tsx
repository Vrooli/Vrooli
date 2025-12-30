import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Package,
  Trash2,
  RefreshCw,
  Loader2,
  AlertCircle,
  CheckCircle2,
  HardDrive,
} from "lucide-react";
import { listBundles, getBundleStats, cleanupBundles } from "../../../lib/api";
import type { BundleInfo, BundleStats, BundleCleanupRequest } from "../../../lib/api";
import { cn } from "../../../lib/utils";
import { formatBytes } from "../../../hooks/useLiveState";

interface BundleInventoryProps {
  deploymentId: string;
}

export function BundleInventory({ deploymentId }: BundleInventoryProps) {
  const queryClient = useQueryClient();
  const [showCleanupConfirm, setShowCleanupConfirm] = useState(false);

  const {
    data: bundlesData,
    isLoading: isLoadingBundles,
    error: bundlesError,
    refetch: refetchBundles,
    isFetching: isFetchingBundles,
  } = useQuery({
    queryKey: ["bundles"],
    queryFn: listBundles,
  });

  const {
    data: statsData,
    isLoading: isLoadingStats,
  } = useQuery({
    queryKey: ["bundleStats"],
    queryFn: getBundleStats,
  });

  const cleanupMutation = useMutation({
    mutationFn: (request: BundleCleanupRequest) => cleanupBundles(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["bundles"] });
      queryClient.invalidateQueries({ queryKey: ["bundleStats"] });
      setShowCleanupConfirm(false);
    },
  });

  const handleCleanup = () => {
    cleanupMutation.mutate({
      keep_latest: 3,
    });
  };

  const handleRefresh = () => {
    refetchBundles();
    queryClient.invalidateQueries({ queryKey: ["bundleStats"] });
  };

  if (isLoadingBundles || isLoadingStats) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
      </div>
    );
  }

  if (bundlesError) {
    return (
      <div className="p-4 rounded-lg border border-red-500/30 bg-red-500/10 text-red-400">
        <div className="flex items-center gap-2">
          <AlertCircle className="h-5 w-5" />
          <span>Failed to load bundles: {bundlesError.message}</span>
        </div>
      </div>
    );
  }

  const bundles = bundlesData?.bundles || [];
  const stats = statsData?.stats;

  // Group bundles by scenario
  const bundlesByScenario: Record<string, BundleInfo[]> = {};
  for (const bundle of bundles) {
    if (!bundlesByScenario[bundle.scenario_id]) {
      bundlesByScenario[bundle.scenario_id] = [];
    }
    bundlesByScenario[bundle.scenario_id].push(bundle);
  }

  // Sort bundles within each scenario by date (newest first)
  for (const scenarioId of Object.keys(bundlesByScenario)) {
    bundlesByScenario[scenarioId].sort((a, b) =>
      new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
  }

  return (
    <div className="space-y-4">
      {/* Stats summary */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard
          label="Total Bundles"
          value={stats?.total_count?.toString() || bundles.length.toString()}
          icon={Package}
        />
        <StatCard
          label="Total Size"
          value={stats?.total_size_bytes ? formatBytes(stats.total_size_bytes) : formatBytes(bundles.reduce((acc, b) => acc + b.size_bytes, 0))}
          icon={HardDrive}
        />
        <StatCard
          label="Scenarios"
          value={Object.keys(bundlesByScenario).length.toString()}
          icon={Package}
        />
        <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4 flex items-center justify-center">
          <button
            onClick={handleRefresh}
            disabled={isFetchingBundles}
            className={cn(
              "flex items-center gap-2 px-4 py-2 rounded-lg border border-white/10",
              "hover:bg-white/5 transition-colors text-sm font-medium",
              isFetchingBundles && "opacity-50 cursor-not-allowed"
            )}
          >
            {isFetchingBundles ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4" />
            )}
            Refresh
          </button>
        </div>
      </div>

      {/* Bundle list by scenario */}
      <div className="border border-white/10 rounded-lg bg-slate-900/50">
        <div className="p-4 border-b border-white/10 flex items-center justify-between">
          <h3 className="text-sm font-medium text-white">Bundle Inventory</h3>
          <div className="flex items-center gap-2">
            {!showCleanupConfirm ? (
              <button
                onClick={() => setShowCleanupConfirm(true)}
                className={cn(
                  "flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm",
                  "border border-amber-500/30 text-amber-400",
                  "hover:bg-amber-500/10 transition-colors"
                )}
              >
                <Trash2 className="h-4 w-4" />
                Cleanup Old
              </button>
            ) : (
              <div className="flex items-center gap-2">
                <span className="text-sm text-slate-400">Keep 3 newest per scenario?</span>
                <button
                  onClick={handleCleanup}
                  disabled={cleanupMutation.isPending}
                  className={cn(
                    "flex items-center gap-1 px-3 py-1.5 rounded-lg text-sm",
                    "bg-amber-500 text-white hover:bg-amber-600 transition-colors",
                    cleanupMutation.isPending && "opacity-50 cursor-not-allowed"
                  )}
                >
                  {cleanupMutation.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <CheckCircle2 className="h-4 w-4" />
                  )}
                  Confirm
                </button>
                <button
                  onClick={() => setShowCleanupConfirm(false)}
                  className="px-3 py-1.5 rounded-lg text-sm border border-white/10 hover:bg-white/5"
                >
                  Cancel
                </button>
              </div>
            )}
          </div>
        </div>

        {bundles.length === 0 ? (
          <div className="p-8 text-center text-slate-500">
            <Package className="h-12 w-12 mx-auto mb-2 opacity-50" />
            <p>No bundles found</p>
          </div>
        ) : (
          <div className="divide-y divide-white/10">
            {Object.entries(bundlesByScenario).map(([scenarioId, scenarioBundles]) => (
              <div key={scenarioId} className="p-4">
                <div className="flex items-center justify-between mb-2">
                  <h4 className="text-sm font-medium text-white">{scenarioId}</h4>
                  <span className="text-xs text-slate-500">
                    {scenarioBundles.length} bundle{scenarioBundles.length !== 1 ? "s" : ""} •{" "}
                    {formatBytes(scenarioBundles.reduce((acc, b) => acc + b.size_bytes, 0))}
                  </span>
                </div>
                <div className="space-y-1">
                  {scenarioBundles.map((bundle, index) => (
                    <BundleRow
                      key={bundle.path}
                      bundle={bundle}
                      isLatest={index === 0}
                    />
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Cleanup result message */}
      {cleanupMutation.isSuccess && cleanupMutation.data && (
        <div className="p-4 rounded-lg border border-emerald-500/30 bg-emerald-500/10 text-emerald-400">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="h-5 w-5" />
            <span>
              Cleaned up {cleanupMutation.data.local_deleted?.length || 0} bundles,
              freed {formatBytes(cleanupMutation.data.local_freed_bytes)}
            </span>
          </div>
        </div>
      )}
    </div>
  );
}

interface StatCardProps {
  label: string;
  value: string;
  icon: React.ElementType;
}

function StatCard({ label, value, icon: Icon }: StatCardProps) {
  return (
    <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4">
      <div className="flex items-center gap-2 text-slate-400 mb-1">
        <Icon className="h-4 w-4" />
        <span className="text-xs">{label}</span>
      </div>
      <div className="text-xl font-semibold text-white">{value}</div>
    </div>
  );
}

interface BundleRowProps {
  bundle: BundleInfo;
  isLatest: boolean;
}

function BundleRow({ bundle, isLatest }: BundleRowProps) {
  const date = new Date(bundle.created_at);
  const shortHash = bundle.sha256.substring(0, 8);

  return (
    <div className={cn(
      "flex items-center justify-between px-3 py-2 rounded text-sm",
      "bg-slate-800/50"
    )}>
      <div className="flex items-center gap-3">
        <Package className="h-4 w-4 text-slate-500" />
        <span className="text-slate-300 font-mono text-xs">{shortHash}</span>
        {isLatest && (
          <span className="px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-400 text-xs">
            Latest
          </span>
        )}
      </div>
      <div className="flex items-center gap-4 text-slate-500 text-xs">
        <span>{formatBytes(bundle.size_bytes)}</span>
        <span>{date.toLocaleDateString()}</span>
      </div>
    </div>
  );
}
