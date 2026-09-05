import { Cpu, HardDrive, Zap } from "lucide-react";
import type { DesktopProcessRoleMetric } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/evidence_pb";
import { useLiveDesktopStore } from "../../store/liveDesktopStore";

function cpuColor(percent: number): string {
  if (percent >= 80) return "text-red-400";
  if (percent >= 50) return "text-amber-400";
  return "text-emerald-400";
}

function formatMB(mb: number): string {
  if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`;
  return `${String(Math.round(mb))} MB`;
}

  function formatSeconds(ms: bigint): string {
  return `${(Number(ms) / 1000).toFixed(1)}s`;
}

export function MetricsBar() {
  const metrics = useLiveDesktopStore((s) => s.activeSession?.metrics);
  const appRunning = useLiveDesktopStore((s) => s.activeSession?.appRunning);

  if (!appRunning) return null;

  // Build the startup timing display:
  // - "Starting..." while waiting for any window
  // - "Splash 0.8s | Loading..." after splash but before main window
  // - "Splash 0.8s | Ready 2.1s" when main window is visible
  // - "Ready 1.2s" if splash and ready are the same (no splash screen)
  const splashMs = metrics?.splashDurationMs;
  const readyMs = metrics?.readyDurationMs;
  const hasSplash = metrics?.splashDetected && splashMs != null;
  const hasReady = metrics?.readyDetected && readyMs != null;

  return (
    <div className="flex items-center gap-3 px-3 py-1 border-t border-slate-800/50 bg-slate-900/40 text-[11px]">
      {/* Startup timing */}
      <div className="flex items-center gap-1">
        <Zap className="h-3 w-3 text-slate-500" />
        {!hasSplash && !hasReady && (
          <span className="text-amber-400 animate-pulse">Starting...</span>
        )}
        {hasSplash && !hasReady && (
          <>
            <span
              className="text-emerald-400"
              title="Time to first window (splash screen)"
            >
              Splash {formatSeconds(splashMs)}
            </span>
            <span className="text-slate-600 mx-0.5">|</span>
            <span className="text-amber-400 animate-pulse">Loading...</span>
          </>
        )}
        {hasReady && hasSplash && splashMs !== readyMs && (
          <>
            <span
              className="text-slate-400"
              title="Time to first window (splash screen)"
            >
              Splash {formatSeconds(splashMs)}
            </span>
            <span className="text-slate-600 mx-0.5">|</span>
            <span
              className="text-emerald-400"
              title="Time to main application window"
            >
              Ready {formatSeconds(readyMs)}
            </span>
          </>
        )}
        {hasReady && (!hasSplash || splashMs === readyMs) && (
          <span className="text-emerald-400" title="Time to application window">
            Ready {formatSeconds(readyMs)}
          </span>
        )}
      </div>

      {metrics && metrics.sampleCount > 0 && (
        <>
          {/* CPU */}
          {metrics.currentCpuPercent != null && (
            <div className="flex items-center gap-1">
              <Cpu className="h-3 w-3 text-slate-500" />
              <span className={cpuColor(metrics.currentCpuPercent)}>
                {Math.round(metrics.currentCpuPercent)}%
              </span>
            </div>
          )}

          {/* Memory */}
          {metrics.currentRssMb != null && (
            <div
              className="flex items-center gap-1"
              title={
                metrics.peakRssMb != null
                  ? `Peak: ${formatMB(metrics.peakRssMb)}`
                  : undefined
              }
            >
              <HardDrive className="h-3 w-3 text-slate-500" />
              <span className="text-slate-300">
                {formatMB(metrics.currentRssMb)}
              </span>
            </div>
          )}
        </>
      )}

      {metrics && (metrics.processRoles?.length ?? 0) > 0 && (
        <details className="relative">
          <summary className="cursor-pointer list-none text-slate-400 hover:text-slate-200">
            Attribution
          </summary>
          <div className="absolute right-0 bottom-6 z-20 min-w-64 rounded border border-slate-700 bg-slate-950 p-2 shadow-xl">
            <div className="mb-1 text-[10px] uppercase tracking-wide text-slate-500">
              Process roles
            </div>
            {(metrics.processRoles ?? []).map((role: DesktopProcessRoleMetric) => (
              <div
                key={role.role}
                className="flex items-center justify-between gap-4 py-1 text-[11px]"
              >
                <span className="text-slate-300">{role.role || "unknown"}</span>
                {role.unsupported ? (
                  <span className="text-slate-500">Unavailable</span>
                ) : role.available ? (
                  <span className="text-slate-400">
                    {role.processCount} proc · {Math.round(Number(role.rssBytes) / 1024 / 1024)} MB
                  </span>
                ) : (
                  <span className="text-slate-500">Not observed</span>
                )}
              </div>
            ))}
          </div>
        </details>
      )}

      {metrics && (metrics.performanceStatus || metrics.protocolStartupDurationMs != null || metrics.demoStartupDurationMs != null) && (
        <details className="relative">
          <summary className="cursor-pointer list-none text-slate-400 hover:text-slate-200">
            Performance
          </summary>
          <div className="absolute right-0 bottom-6 z-20 min-w-64 rounded border border-slate-700 bg-slate-950 p-2 shadow-xl">
            <div className="mb-1 text-[10px] uppercase tracking-wide text-slate-500">
              Launch evidence
            </div>
            <div className="flex justify-between gap-4 py-1 text-[11px]">
              <span className="text-slate-300">Status</span>
              <span className={metrics.performanceStatus === "measured" ? "text-emerald-400" : "text-amber-400"}>
                {metrics.performanceStatus || "Unmeasured"}
              </span>
            </div>
            {metrics.performanceReason && <div className="py-1 text-[10px] text-slate-500">{metrics.performanceReason}</div>}
            <div className="flex justify-between gap-4 py-1 text-[11px] text-slate-400">
              <span>Protocol</span>
              <span>{metrics.protocolStartupDurationMs != null ? formatSeconds(metrics.protocolStartupDurationMs) : "Unavailable"}</span>
            </div>
            <div className="flex justify-between gap-4 py-1 text-[11px] text-slate-400">
              <span>Demo</span>
              <span>{metrics.demoStartupDurationMs != null ? formatSeconds(metrics.demoStartupDurationMs) : "Unavailable"}</span>
            </div>
          </div>
        </details>
      )}
    </div>
  );
}
