import { Cpu, HardDrive, Zap } from "lucide-react";
import { useSessionStore } from "../../store/sessionStore";

function cpuColor(percent: number): string {
  if (percent >= 80) return "text-red-400";
  if (percent >= 50) return "text-amber-400";
  return "text-emerald-400";
}

function formatMB(mb: number): string {
  if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`;
  return `${Math.round(mb)} MB`;
}

function formatSeconds(ms: number): string {
  return `${(ms / 1000).toFixed(1)}s`;
}

export function MetricsBar() {
  const metrics = useSessionStore((s) => s.activeSession?.metrics);
  const appRunning = useSessionStore((s) => s.activeSession?.app_running);

  if (!appRunning) return null;

  const splashMs = metrics?.splash_duration_ms;
  const readyMs = metrics?.ready_duration_ms;
  const hasSplash = metrics?.splash_detected && splashMs != null;
  const hasReady = metrics?.ready_detected && readyMs != null;

  return (
    <div className="flex items-center gap-3 px-3 py-1 border-t border-slate-800/50 bg-slate-900/40 text-[11px]">
      <div className="flex items-center gap-1">
        <Zap className="h-3 w-3 text-slate-500" />
        {!hasSplash && !hasReady && <span className="text-amber-400 animate-pulse">Starting...</span>}
        {hasSplash && !hasReady && (
          <>
            <span className="text-emerald-400" title="Time to first window (splash screen)">
              Splash {formatSeconds(splashMs)}
            </span>
            <span className="text-slate-600 mx-0.5">|</span>
            <span className="text-amber-400 animate-pulse">Loading...</span>
          </>
        )}
        {hasReady && hasSplash && splashMs !== readyMs && (
          <>
            <span className="text-slate-400" title="Time to first window (splash screen)">
              Splash {formatSeconds(splashMs)}
            </span>
            <span className="text-slate-600 mx-0.5">|</span>
            <span className="text-emerald-400" title="Time to main application window">
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

      {metrics && metrics.sample_count > 0 && (
        <>
          {metrics.current_cpu_percent != null && (
            <div className="flex items-center gap-1">
              <Cpu className="h-3 w-3 text-slate-500" />
              <span className={cpuColor(metrics.current_cpu_percent)}>
                {Math.round(metrics.current_cpu_percent)}%
              </span>
            </div>
          )}

          {metrics.current_rss_mb != null && (
            <div
              className="flex items-center gap-1"
              title={metrics.peak_rss_mb != null ? `Peak: ${formatMB(metrics.peak_rss_mb)}` : undefined}
            >
              <HardDrive className="h-3 w-3 text-slate-500" />
              <span className="text-slate-300">{formatMB(metrics.current_rss_mb)}</span>
            </div>
          )}
        </>
      )}
    </div>
  );
}
