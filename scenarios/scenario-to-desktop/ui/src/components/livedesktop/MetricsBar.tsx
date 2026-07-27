import { Cpu, HardDrive, Zap } from "lucide-react";
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
    </div>
  );
}
