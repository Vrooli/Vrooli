/**
 * Grid component showing service readiness status with health check support.
 */

import { useState } from "react";
import { fetchBundlePreflightHealth } from "../../lib/api";
import type { HealthPeekState } from "../../lib/preflight-constants";
import {
  formatDuration,
  getListenURL,
  getManifestHealthConfig,
  getServiceURL,
  normalizeHealthPath,
  parseTimestamp
} from "../../lib/preflight-utils";

interface ServiceStatus {
  ready: boolean;
  skipped?: boolean;
  message?: string;
  started_at?: string;
  ready_at?: string;
  updated_at?: string;
  exit_code?: number;
}

interface ServicesReadinessGridProps {
  readinessDetails: Array<[string, ServiceStatus]>;
  ports?: Record<string, Record<string, number>>;
  bundleManifest?: unknown;
  preflightSessionId?: string | null;
  snapshotTs: number;
  tick: number;
}

export function ServicesReadinessGrid({
  readinessDetails,
  ports,
  bundleManifest,
  preflightSessionId,
  snapshotTs,
  tick
}: ServicesReadinessGridProps) {
  const [healthPeek, setHealthPeek] = useState<Record<string, HealthPeekState>>({});

  const toggleHealthPeek = async (serviceId: string, healthSupported: boolean) => {
    if (!healthSupported) {
      setHealthPeek((prev) => ({
        ...prev,
        [serviceId]: { open: true, status: "error", error: "Health proxy only supports http checks with a configured health path." }
      }));
      return;
    }
    if (!preflightSessionId) {
      setHealthPeek((prev) => ({
        ...prev,
        [serviceId]: { open: true, status: "error", error: "Preflight session missing. Re-run preflight with a session TTL to fetch health responses." }
      }));
      return;
    }
    const current = healthPeek[serviceId];
    if (current?.open) {
      setHealthPeek((prev) => ({
        ...prev,
        [serviceId]: { ...current, open: false }
      }));
      return;
    }
    if (current?.status === "ready" && current.body) {
      setHealthPeek((prev) => ({
        ...prev,
        [serviceId]: { ...current, open: true }
      }));
      return;
    }
    setHealthPeek((prev) => ({
      ...prev,
      [serviceId]: { open: true, status: "loading" }
    }));
    try {
      const response = await fetchBundlePreflightHealth({
        session_id: preflightSessionId,
        service_id: serviceId
      });
      const raw = response.body ?? "";
      let body = raw.trim();
      try {
        const json = JSON.parse(raw);
        body = JSON.stringify(json, null, 2);
      } catch {
        // Leave body as raw text when not JSON.
      }
      setHealthPeek((prev) => ({
        ...prev,
        [serviceId]: {
          open: true,
          status: "ready",
          body,
          statusCode: response.status_code,
          url: response.url,
          contentType: response.content_type,
          truncated: response.truncated
        }
      }));
    } catch (err) {
      setHealthPeek((prev) => ({
        ...prev,
        [serviceId]: { open: true, status: "error", error: err instanceof Error ? err.message : String(err) }
      }));
    }
  };

  return (
    <div className="mt-2 grid gap-2 sm:grid-cols-2">
      {readinessDetails.map(([serviceId, status]) => {
        const referenceTs = snapshotTs || tick;
        const updatedAt = parseTimestamp(status.updated_at);
        const startedAt = parseTimestamp(status.started_at);
        const readyAt = parseTimestamp(status.ready_at);
        const isSkipped = Boolean(status.skipped);
        const isUpdatedAhead = updatedAt ? updatedAt > tick + 5000 : false;
        const statusAge = updatedAt && !isUpdatedAhead ? formatDuration(Math.max(0, referenceTs - updatedAt)) : "";
        const startedAge = startedAt ? formatDuration(Math.max(0, referenceTs - startedAt)) : "";
        const readyAge = readyAt ? formatDuration(Math.max(0, referenceTs - readyAt)) : "";
        const statusLabel = isSkipped ? "Skipped" : status.ready ? "Ready" : "Waiting";
        const statusClass = isSkipped ? "text-slate-300" : status.ready ? "text-emerald-200" : "text-amber-200";
        const healthConfig = getManifestHealthConfig(bundleManifest, serviceId);
        const healthType = healthConfig?.type?.toLowerCase();
        const healthPath = normalizeHealthPath(healthConfig?.path);
        const healthPortName = healthConfig?.portName;
        const portInfo = getServiceURL(serviceId, ports);
        const healthPortInfo = healthPortName
          ? getServiceURL(serviceId, ports, healthPortName)
          : null;
        const healthSupported = healthType === "http" && Boolean(healthPath) && Boolean(healthPortInfo);
        const listenURL = portInfo?.url ?? getListenURL(status.message);
        const healthURL = healthPortInfo?.url && healthPath
          ? `${healthPortInfo.url}${healthPath}`
          : null;
        const peekState = healthPeek[serviceId];

        return (
          <div key={serviceId} className="rounded-md border border-slate-800/70 bg-slate-950/80 p-2 space-y-1">
            <p className="text-[11px] uppercase tracking-wide text-slate-400">{serviceId}</p>
            <p className={`text-xs ${statusClass}`}>
              {statusLabel}
            </p>
            {status.message && (
              <p className="text-[11px] text-slate-300">
                <span>{status.message}</span>
              </p>
            )}
            <div className="text-[10px] text-slate-400 space-y-0.5">
              {startedAt && <p>Started {new Date(startedAt).toLocaleTimeString()} ({startedAge} ago)</p>}
              {readyAt && <p>Ready {new Date(readyAt).toLocaleTimeString()} ({readyAge} ago)</p>}
              {updatedAt && <p>Updated {new Date(updatedAt).toLocaleTimeString()} ({statusAge} ago)</p>}
              {typeof status.exit_code === "number" && <p>Exit code {status.exit_code}</p>}
            </div>
            {listenURL && (
              <div className="flex flex-wrap items-center gap-2 text-[10px]">
                <a
                  className="inline-flex items-center gap-1 rounded-sm border border-sky-900/60 bg-sky-950/40 px-2 py-0.5 font-semibold uppercase tracking-wide text-sky-200 hover:text-sky-100"
                  href={listenURL}
                  target="_blank"
                  rel="noreferrer"
                >
                  Open
                </a>
                {healthURL && (
                  <a
                    className="inline-flex items-center gap-1 rounded-sm border border-slate-800/60 bg-slate-950/60 px-2 py-0.5 font-semibold uppercase tracking-wide text-slate-200 hover:text-slate-100"
                    href={healthURL}
                    target="_blank"
                    rel="noreferrer"
                  >
                    Health
                  </a>
                )}
                {healthSupported && (
                  <button
                    type="button"
                    className="inline-flex items-center gap-1 rounded-sm border border-slate-800/60 bg-slate-950/60 px-2 py-0.5 font-semibold uppercase tracking-wide text-slate-200 hover:text-slate-100"
                    onClick={() => toggleHealthPeek(serviceId, healthSupported)}
                  >
                    {peekState?.open ? "Hide" : "View"} response
                  </button>
                )}
                <span className="text-slate-400">{listenURL}</span>
                {portInfo && (
                  <span className="text-slate-400">
                    {portInfo.portName}:{portInfo.port}
                  </span>
                )}
              </div>
            )}
            {!healthSupported && (
              <p className="text-[10px] text-slate-500">
                Health check not configured for http proxy inspection.
              </p>
            )}
            {peekState?.open && (
              <div className="rounded-md border border-slate-800/70 bg-slate-950/70 p-2 text-[10px] text-slate-200">
                {peekState.status === "loading" && (
                  <p className="text-slate-400">Loading health response...</p>
                )}
                {peekState.status === "error" && (
                  <p className="text-red-200">
                    Failed to fetch health response via the runtime proxy. Open the Health link to inspect directly.
                    {peekState.error ? ` (${peekState.error})` : ""}
                  </p>
                )}
                {peekState.status === "ready" && (
                  <div className="space-y-1">
                    <p className="text-[10px] text-slate-400">
                      {peekState.statusCode ? `HTTP ${peekState.statusCode}` : "HTTP response"}
                      {peekState.url ? ` · ${peekState.url}` : ""}
                      {peekState.truncated ? " · truncated" : ""}
                    </p>
                    {peekState.body && (
                      <pre className="whitespace-pre-wrap break-words">{peekState.body}</pre>
                    )}
                  </div>
                )}
              </div>
            )}
            {status.message === "pending start" && !isSkipped && (
              <p className="text-[11px] text-slate-500">
                Not launched yet; waiting on dependencies or secrets.
              </p>
            )}
            {status.ready && !isSkipped && readyAge && (
              <p className="text-[11px] text-emerald-200/80">Ready for {readyAge}</p>
            )}
            {!status.ready && !isSkipped && startedAge && (
              <p className="text-[11px] text-amber-200/80">Starting for {startedAge}</p>
            )}
            {!status.ready && !isSkipped && !startedAge && statusAge && (
              <p className="text-[11px] text-slate-400">Status set {statusAge} ago</p>
            )}
          </div>
        );
      })}
    </div>
  );
}
