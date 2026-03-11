/**
 * Settings Page
 *
 * Storage management, score configuration, and system settings.
 *
 * [REQ:LD-UI-STORAGE] - Storage management UI
 * [REQ:LD-SCORE-CALC] - Score configuration UI
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
  Scale,
} from "lucide-react";
import { Link } from "react-router-dom";

import { Card, StatBox } from "../components/ui";
import {
  fetchHealth,
  fetchStorageInfo,
  cleanupEvents,
  fetchScoreConfig,
  updateDomainWeight,
  type StorageInfo,
  type DomainWeightConfig,
} from "../lib/api";
import { formatBytes, formatDate } from "../lib/format";
import { DATA_SELECTORS } from "../consts/selectors";

const WEIGHT_OPTIONS = [
  { value: "high", label: "High", color: "text-green-400", description: "3x weight" },
  { value: "medium", label: "Medium", color: "text-yellow-400", description: "2x weight" },
  { value: "low", label: "Low", color: "text-gray-400", description: "1x weight" },
  { value: "none", label: "None", color: "text-red-400", description: "Excluded" },
];

export default function SettingsPage() {
  const queryClient = useQueryClient();
  const [showConfirmClear, setShowConfirmClear] = useState(false);
  const [selectedDomain, setSelectedDomain] = useState<string | null>(null);
  const [cleanupResult, setCleanupResult] = useState<string | null>(null);
  const [weightResult, setWeightResult] = useState<string | null>(null);

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

  // [REQ:LD-SCORE-CALC] Score configuration query
  const scoreConfigQuery = useQuery({
    queryKey: ["scoreConfig"],
    queryFn: fetchScoreConfig,
    refetchInterval: 60000,
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

  // [REQ:LD-SCORE-CALC] Weight update mutation
  const weightMutation = useMutation({
    mutationFn: ({ domain, weight }: { domain: string; weight: string }) =>
      updateDomainWeight(domain, weight),
    onSuccess: (data) => {
      setWeightResult(`Updated ${data.display_name} to ${data.weight} weight`);
      queryClient.invalidateQueries({ queryKey: ["scoreConfig"] });
      queryClient.invalidateQueries({ queryKey: ["score"] });
      setTimeout(() => setWeightResult(null), 3000);
    },
    onError: (err) => {
      setWeightResult(`Error: ${err instanceof Error ? err.message : "Failed to update weight"}`);
      setTimeout(() => setWeightResult(null), 5000);
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

  const handleWeightChange = useCallback(
    (domain: string, weight: string) => {
      weightMutation.mutate({ domain, weight });
    },
    [weightMutation]
  );

  const dbInfo = healthQuery.data?.dependencies?.database;
  const storage: StorageInfo | undefined = storageQuery.data;
  const scoreConfig = scoreConfigQuery.data;

  return (
    <div className="space-y-6" data-testid={DATA_SELECTORS.SETTINGS_PAGE}>
      {/* Page header */}
      <div className="flex items-center gap-4">
        <Link to="/" className="text-slate-400 hover:text-white transition-colors">
          <ChevronLeft className="w-5 h-5" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold">Settings</h1>
          <p className="text-slate-400">Storage management, score weights, and configuration</p>
        </div>
      </div>

      {/* Notifications */}
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

      {weightResult && (
        <div
          className={`p-4 rounded-lg flex items-center gap-3 ${
            weightResult.startsWith("Error")
              ? "bg-red-500/10 border border-red-500/20 text-red-400"
              : "bg-green-500/10 border border-green-500/20 text-green-400"
          }`}
        >
          {weightResult.startsWith("Error") ? (
            <AlertTriangle className="w-5 h-5 flex-shrink-0" />
          ) : (
            <CheckCircle className="w-5 h-5 flex-shrink-0" />
          )}
          <span>{weightResult}</span>
        </div>
      )}

      {/* Score Configuration [REQ:LD-SCORE-CALC] */}
      <Card padding="lg">
        <div className="flex items-center gap-3 mb-6">
          <Scale className="w-6 h-6 text-purple-400" />
          <div>
            <h2 className="text-lg font-medium">Score Configuration</h2>
            <p className="text-sm text-slate-400">Adjust how each domain contributes to your Lifestyle Score</p>
          </div>
        </div>

        {scoreConfigQuery.isLoading ? (
          <div className="text-center py-8 text-slate-500">Loading score configuration...</div>
        ) : scoreConfig && scoreConfig.weights.length > 0 ? (
          <div className="space-y-3">
            {scoreConfig.weights.map((config: DomainWeightConfig) => (
              <div
                key={config.domain}
                className="flex items-center justify-between py-3 px-4 rounded-lg bg-slate-800/30"
                data-testid={`weight-config-${config.domain}`}
              >
                <div>
                  <p className="font-medium">{config.display_name}</p>
                  <p className="text-sm text-slate-500">{config.domain}</p>
                </div>
                <div className="flex items-center gap-2">
                  <select
                    value={config.weight}
                    onChange={(e) => handleWeightChange(config.domain, e.target.value)}
                    disabled={weightMutation.isPending}
                    className="bg-slate-800 border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-purple-500 disabled:opacity-50"
                  >
                    {WEIGHT_OPTIONS.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label} ({option.description})
                      </option>
                    ))}
                  </select>
                  {weightMutation.isPending && (
                    <Loader2 className="w-4 h-4 animate-spin text-purple-400" />
                  )}
                </div>
              </div>
            ))}

            <p className="text-xs text-slate-500 mt-4 pt-4 border-t border-white/5">
              <strong>Weight multipliers:</strong> High = 3x, Medium = 2x, Low = 1x, None = excluded from score.
              New domains default to {scoreConfig.default_weight} weight.
            </p>
          </div>
        ) : (
          <div className="text-center py-8 text-slate-400">
            <Scale className="w-12 h-12 mx-auto mb-3 text-slate-700" />
            <p>No domains registered yet</p>
            <p className="text-sm text-slate-500 mt-1">
              Domain score weights will appear here when domains are registered
            </p>
          </div>
        )}
      </Card>

      {/* Storage overview */}
      <Card padding="lg">
        <div className="flex items-center gap-3 mb-6">
          <HardDrive className="w-6 h-6 text-blue-400" />
          <h2 className="text-lg font-medium">Storage Overview</h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {/* Database size */}
          <StatBox>
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
          </StatBox>

          {/* Total events */}
          <StatBox>
            <div className="flex items-center gap-2 mb-2">
              <span className="text-sm text-slate-400">Total Events</span>
            </div>
            <p className="text-lg font-medium">{storage?.total_events ?? "-"}</p>
            <p className="text-sm text-slate-500">Stored in database</p>
          </StatBox>

          {/* Active domains */}
          <StatBox>
            <div className="flex items-center gap-2 mb-2">
              <span className="text-sm text-slate-400">Active Domains</span>
            </div>
            <p className="text-lg font-medium">{storage?.total_domains ?? "-"}</p>
            <p className="text-sm text-slate-500">Currently registered</p>
          </StatBox>

          {/* Date range */}
          <StatBox>
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
          </StatBox>
        </div>
      </Card>

      {/* Events by domain */}
      {storage?.events_by_domain && storage.events_by_domain.length > 0 && (
        <Card padding="lg">
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
        </Card>
      )}

      {/* Service info */}
      <Card padding="lg">
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
      </Card>

      {/* Data management */}
      <Card padding="lg">
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
          [REQ:LD-UI-STORAGE] [REQ:LD-SCORE-CALC] Storage and score configuration
        </p>
      </Card>

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
