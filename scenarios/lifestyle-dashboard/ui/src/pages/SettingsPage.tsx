/**
 * Settings Page
 *
 * Storage management and configuration settings.
 *
 * [REQ:LD-STORAGE-MANAGE] - Storage management UI
 */
import { useQuery } from "@tanstack/react-query";
import { ChevronLeft, Database, HardDrive, Trash2, RefreshCw } from "lucide-react";
import { Link } from "react-router-dom";

import { fetchHealth, fetchSummary } from "../lib/api";

export default function SettingsPage() {
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 30000,
  });

  const summaryQuery = useQuery({
    queryKey: ["summary"],
    queryFn: fetchSummary,
    refetchInterval: 30000,
  });

  const dbInfo = healthQuery.data?.dependencies?.database;

  return (
    <div className="space-y-6">
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

      {/* Storage overview */}
      <div className="rounded-xl border border-white/10 bg-white/5 p-6">
        <div className="flex items-center gap-3 mb-6">
          <HardDrive className="w-6 h-6 text-blue-400" />
          <h2 className="text-lg font-medium">Storage Overview</h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Database status */}
          <div className="rounded-lg bg-slate-800/50 p-4">
            <div className="flex items-center gap-2 mb-2">
              <Database className="w-4 h-4 text-slate-400" />
              <span className="text-sm text-slate-400">Database</span>
            </div>
            <p className="text-lg font-medium">
              {dbInfo?.connected ? "Connected" : "Disconnected"}
            </p>
            {dbInfo?.database && (
              <p className="text-sm text-slate-500 truncate" title={dbInfo.database}>
                {dbInfo.database}
              </p>
            )}
          </div>

          {/* Total events */}
          <div className="rounded-lg bg-slate-800/50 p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-sm text-slate-400">Total Events</span>
            </div>
            <p className="text-lg font-medium">
              {summaryQuery.data?.total_events ?? "-"}
            </p>
            <p className="text-sm text-slate-500">
              Stored in database
            </p>
          </div>

          {/* Active domains */}
          <div className="rounded-lg bg-slate-800/50 p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-sm text-slate-400">Active Domains</span>
            </div>
            <p className="text-lg font-medium">
              {summaryQuery.data?.active_domains ?? "-"}
            </p>
            <p className="text-sm text-slate-500">
              Currently registered
            </p>
          </div>
        </div>
      </div>

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
              {dbInfo?.latency_ms
                ? `${dbInfo.latency_ms.toFixed(1)}ms`
                : "-"}
            </span>
          </div>
          <div className="flex justify-between py-2">
            <span className="text-slate-400">Status</span>
            <span className={`font-medium ${healthQuery.data?.status === "ok" ? "text-green-400" : "text-red-400"}`}>
              {healthQuery.data?.status ?? "unknown"}
            </span>
          </div>
        </div>
      </div>

      {/* Data management - placeholder for P0-006 */}
      <div className="rounded-xl border border-white/10 bg-white/5 p-6">
        <div className="flex items-center gap-3 mb-6">
          <Trash2 className="w-6 h-6 text-red-400" />
          <h2 className="text-lg font-medium">Data Management</h2>
        </div>

        <p className="text-slate-400 mb-4">
          Data cleanup functionality will be available in a future update.
        </p>

        <div className="flex gap-3">
          <button
            disabled
            className="px-4 py-2 rounded-lg bg-red-500/10 text-red-400 border border-red-500/20 opacity-50 cursor-not-allowed"
          >
            Clear All Events
          </button>
          <button
            disabled
            className="px-4 py-2 rounded-lg bg-slate-500/10 text-slate-400 border border-slate-500/20 opacity-50 cursor-not-allowed"
          >
            Export Data
          </button>
        </div>

        <p className="mt-4 text-xs text-slate-500">
          [REQ:LD-STORAGE-MANAGE] Storage management features - planned for P0-006
        </p>
      </div>
    </div>
  );
}
