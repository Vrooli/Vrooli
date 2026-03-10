/**
 * Settings Page
 *
 * Storage management and configuration settings.
 *
 * [REQ:LD-UI-STORAGE] - Storage management UI
 */
import { useState, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  ChevronLeft,
  Database,
  HardDrive,
  Trash2,
  RefreshCw,
  AlertTriangle,
  CheckCircle,
  Loader2,
} from "lucide-react";
import { Link } from "react-router-dom";

import { fetchHealth, fetchStorageInfo, cleanupEvents, type StorageInfo } from "../lib/api";
import { formatBytes, formatDate } from "../lib/format";
import { DATA_SELECTORS } from "../consts/selectors";

export default function SettingsPage() {
  const queryClient = useQueryClient();
  const [showConfirmClear, setShowConfirmClear] = useState(false);
  const [selectedDomain, setSelectedDomain] = useState<string | null>(null);
  const [cleanupResult, setCleanupResult] = useState<string | null>(null);

  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 30000,
  });

  const storageQuery = useQuery({
    queryKey: ["storage"],
    queryFn: fetchStorageInfo,
    refetchInterval: 30000,
  });

  const cleanupMutation = useMutation({
    mutationFn: (domains?: string[]) => cleanupEvents({ domains }),
    onSuccess: (data) => {
      setCleanupResult(data.message);
      setShowConfirmClear(false);
      setSelectedDomain(null);
      // Invalidate queries to refresh data
      queryClient.invalidateQueries({ queryKey: ["storage"] });
      queryClient.invalidateQueries({ queryKey: ["summary"] });
      queryClient.invalidateQueries({ queryKey: ["events"] });
      queryClient.invalidateQueries({ queryKey: ["timeline"] });
      queryClient.invalidateQueries({ queryKey: ["score"] });
      // Clear result after 5 seconds
      setTimeout(() => setCleanupResult(null), 5000);
    },
    onError: (err) => {
      setCleanupResult(`Error: ${err instanceof Error ? err.message : "Failed to clear events"}`);
      setTimeout(() => setCleanupResult(null), 5000);
    },
  });

  const handleClearAll = useCallback(() => {
    cleanupMutation.mutate(undefined);
  }, [cleanupMutation]);

  const handleClearDomain = useCallback(
    (domain: string) => {
      cleanupMutation.mutate([domain]);
    },
    [cleanupMutation]
  );

  const dbInfo = healthQuery.data?.dependencies?.database;
  const storage: StorageInfo | undefined = storageQuery.data;

  return (
    <div className="space-y-6" data-testid={DATA_SELECTORS.SETTINGS_PAGE}>
      {/* Page header */}
      <div className="flex items-center gap-4">
        <Link to="/" className="text-slate-400 hover:text-white transition-colors">
          <ChevronLeft className="w-5 h-5" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold">Settings</h1>
          <p className="text-slate-400">Storage management and configuration</p>
        </div>
      </div>

      {/* Cleanup result notification */}
      {cleanupResult && (
        <div
          className={`p-4 rounded-lg flex items-center gap-3 ${
            cleanupResult.startsWith("Error")
              ? "bg-red-500/10 border border-red-500/20 text-red-400"
              : "bg-green-500/10 border border-green-500/20 text-green-400"
          }`}
        >
          {cleanupResult.startsWith("Error") ? (
            <AlertTriangle className="w-5 h-5 flex-shrink-0" />
          ) : (
            <CheckCircle className="w-5 h-5 flex-shrink-0" />
          )}
          <span>{cleanupResult}</span>
        </div>
      )}

      {/* Storage overview */}
      <div className="rounded-xl border border-white/10 bg-white/5 p-6">
        <div className="flex items-center gap-3 mb-6">
          <HardDrive className="w-6 h-6 text-blue-400" />
          <h2 className="text-lg font-medium">Storage Overview</h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {/* Database size */}
          <div className="rounded-lg bg-slate-800/50 p-4">
            <div className="flex items-center gap-2 mb-2">
              <Database className="w-4 h-4 text-slate-400" />
              <span className="text-sm text-slate-400">Database Size</span>
            </div>
            <p className="text-lg font-medium">
              {storage?.database_size_bytes !== undefined
                ? formatBytes(storage.database_size_bytes)
                : "-"}
            </p>
            <p className="text-sm text-slate-500">
              {dbInfo?.connected ? "SQLite (WAL)" : "Disconnected"}
            </p>
          </div>

          {/* Total events */}
          <div className="rounded-lg bg-slate-800/50 p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-sm text-slate-400">Total Events</span>
            </div>
            <p className="text-lg font-medium">{storage?.total_events ?? "-"}</p>
            <p className="text-sm text-slate-500">Stored in database</p>
          </div>

          {/* Active domains */}
          <div className="rounded-lg bg-slate-800/50 p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-sm text-slate-400">Active Domains</span>
            </div>
            <p className="text-lg font-medium">{storage?.total_domains ?? "-"}</p>
            <p className="text-sm text-slate-500">Currently registered</p>
          </div>

          {/* Date range */}
          <div className="rounded-lg bg-slate-800/50 p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-sm text-slate-400">Data Range</span>
            </div>
            <p className="text-lg font-medium">
              {storage?.oldest_event ? formatDate(storage.oldest_event) : "-"}
            </p>
            <p className="text-sm text-slate-500">
              {storage?.newest_event
                ? `to ${formatDate(storage.newest_event)}`
                : "No events"}
            </p>
          </div>
        </div>
      </div>

      {/* Events by domain */}
      {storage?.events_by_domain && storage.events_by_domain.length > 0 && (
        <div className="rounded-xl border border-white/10 bg-white/5 p-6">
          <div className="flex items-center gap-3 mb-6">
            <Database className="w-6 h-6 text-purple-400" />
            <h2 className="text-lg font-medium">Events by Domain</h2>
          </div>

          <div className="space-y-2">
            {storage.events_by_domain.map((domainInfo) => (
              <div
                key={domainInfo.domain}
                className="flex items-center justify-between py-3 px-4 rounded-lg bg-slate-800/30 hover:bg-slate-800/50 transition-colors"
              >
                <div>
                  <p className="font-medium">{domainInfo.display_name}</p>
                  <p className="text-sm text-slate-500">{domainInfo.domain}</p>
                </div>
                <div className="flex items-center gap-4">
                  <span className="text-slate-400">
                    {domainInfo.event_count.toLocaleString()} events
                  </span>
                  <button
                    onClick={() => setSelectedDomain(domainInfo.domain)}
                    disabled={cleanupMutation.isPending}
                    className="px-3 py-1 text-sm rounded-lg bg-red-500/10 text-red-400 border border-red-500/20 hover:bg-red-500/20 transition-colors disabled:opacity-50"
                  >
                    Clear
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Service info */}
      <div className="rounded-xl border border-white/10 bg-white/5 p-6">
        <div className="flex items-center gap-3 mb-6">
          <RefreshCw className="w-6 h-6 text-green-400" />
          <h2 className="text-lg font-medium">Service Information</h2>
        </div>

        <div className="space-y-3 text-sm">
          <div className="flex justify-between py-2 border-b border-white/5">
            <span className="text-slate-400">Service Name</span>
            <span className="font-medium">{healthQuery.data?.service ?? "-"}</span>
          </div>
          <div className="flex justify-between py-2 border-b border-white/5">
            <span className="text-slate-400">Version</span>
            <span className="font-medium">{healthQuery.data?.version ?? "1.0.0"}</span>
          </div>
          <div className="flex justify-between py-2 border-b border-white/5">
            <span className="text-slate-400">Uptime</span>
            <span className="font-medium">
              {healthQuery.data?.uptime_seconds
                ? `${Math.floor(healthQuery.data.uptime_seconds / 60)} minutes`
                : "-"}
            </span>
          </div>
          <div className="flex justify-between py-2 border-b border-white/5">
            <span className="text-slate-400">Database Latency</span>
            <span className="font-medium">
              {dbInfo?.latency_ms ? `${dbInfo.latency_ms.toFixed(1)}ms` : "-"}
            </span>
          </div>
          <div className="flex justify-between py-2">
            <span className="text-slate-400">Status</span>
            <span
              className={`font-medium ${healthQuery.data?.status === "ok" ? "text-green-400" : "text-red-400"}`}
            >
              {healthQuery.data?.status ?? "unknown"}
            </span>
          </div>
        </div>
      </div>

      {/* Data management */}
      <div className="rounded-xl border border-white/10 bg-white/5 p-6">
        <div className="flex items-center gap-3 mb-6">
          <Trash2 className="w-6 h-6 text-red-400" />
          <h2 className="text-lg font-medium">Data Management</h2>
        </div>

        <p className="text-slate-400 mb-4">
          Clear all events from the database. This action cannot be undone.
        </p>

        <div className="flex gap-3">
          <button
            onClick={() => setShowConfirmClear(true)}
            disabled={cleanupMutation.isPending || (storage?.total_events ?? 0) === 0}
            className="px-4 py-2 rounded-lg bg-red-500/10 text-red-400 border border-red-500/20 hover:bg-red-500/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            {cleanupMutation.isPending ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Trash2 className="w-4 h-4" />
            )}
            Clear All Events
          </button>
        </div>

        <p className="mt-4 text-xs text-slate-500">
          [REQ:LD-UI-STORAGE] Storage management features
        </p>
      </div>

      {/* Confirm clear all modal */}
      {showConfirmClear && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-slate-900 border border-white/10 rounded-xl p-6 max-w-md w-full mx-4">
            <div className="flex items-center gap-3 mb-4">
              <AlertTriangle className="w-6 h-6 text-red-400" />
              <h3 className="text-lg font-medium">Clear All Events?</h3>
            </div>
            <p className="text-slate-400 mb-6">
              This will permanently delete all {storage?.total_events?.toLocaleString()} events
              from the database. This action cannot be undone.
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowConfirmClear(false)}
                className="px-4 py-2 rounded-lg bg-slate-700 text-white hover:bg-slate-600 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleClearAll}
                disabled={cleanupMutation.isPending}
                className="px-4 py-2 rounded-lg bg-red-600 text-white hover:bg-red-500 transition-colors disabled:opacity-50 flex items-center gap-2"
              >
                {cleanupMutation.isPending && <Loader2 className="w-4 h-4 animate-spin" />}
                Clear All
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Confirm clear domain modal */}
      {selectedDomain && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-slate-900 border border-white/10 rounded-xl p-6 max-w-md w-full mx-4">
            <div className="flex items-center gap-3 mb-4">
              <AlertTriangle className="w-6 h-6 text-red-400" />
              <h3 className="text-lg font-medium">Clear Domain Events?</h3>
            </div>
            <p className="text-slate-400 mb-6">
              This will permanently delete all events from the <strong>{selectedDomain}</strong>{" "}
              domain. This action cannot be undone.
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setSelectedDomain(null)}
                className="px-4 py-2 rounded-lg bg-slate-700 text-white hover:bg-slate-600 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => handleClearDomain(selectedDomain)}
                disabled={cleanupMutation.isPending}
                className="px-4 py-2 rounded-lg bg-red-600 text-white hover:bg-red-500 transition-colors disabled:opacity-50 flex items-center gap-2"
              >
                {cleanupMutation.isPending && <Loader2 className="w-4 h-4 animate-spin" />}
                Clear Domain
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
