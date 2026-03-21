import { Square, Maximize2, Minimize2, Play, AlertCircle, X, Monitor } from "lucide-react";
import { useState, useCallback, useEffect, useRef, type RefObject } from "react";
import { useLiveDesktopStore } from "../../store/liveDesktopStore";
import { launchAppOnDesktop } from "../../lib/api/livedesktop";
import type { ConnectionStatus } from "../../lib/api/livedesktop";

const STATUS_COLORS: Record<ConnectionStatus, string> = {
  disconnected: "bg-slate-500",
  connecting: "bg-amber-400 animate-pulse",
  connected: "bg-emerald-400",
  error: "bg-red-400",
};

interface DesktopToolbarProps {
  fullscreenTargetRef: RefObject<HTMLDivElement | null>;
  onClose: () => void;
}

export function DesktopToolbar({ fullscreenTargetRef, onClose }: DesktopToolbarProps) {
  const connectionStatus = useLiveDesktopStore((s) => s.connectionStatus);
  const activeSession = useLiveDesktopStore((s) => s.activeSession);
  const scenarioName = useLiveDesktopStore((s) => s.scenarioName);
  const appPath = useLiveDesktopStore((s) => s.appPath);
  const stopSession = useLiveDesktopStore((s) => s.stopSession);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [launching, setLaunching] = useState(false);
  const [launchError, setLaunchError] = useState<string | null>(null);
  const launchInFlight = useRef(false);

  useEffect(() => {
    const onChange = () => setIsFullscreen(!!document.fullscreenElement);
    document.addEventListener("fullscreenchange", onChange);
    return () => document.removeEventListener("fullscreenchange", onChange);
  }, []);

  const toggleFullscreen = useCallback(() => {
    const target = fullscreenTargetRef.current;
    if (!target) return;

    if (!document.fullscreenElement) {
      target.requestFullscreen().catch(() => {});
    } else {
      document.exitFullscreen().catch(() => {});
    }
  }, [fullscreenTargetRef]);

  const handleLaunchApp = useCallback(async () => {
    if (!activeSession || launchInFlight.current) return;
    launchInFlight.current = true;
    setLaunching(true);
    setLaunchError(null);
    try {
      await launchAppOnDesktop(activeSession.id, appPath ?? undefined);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to launch app";
      setLaunchError(msg);
    } finally {
      setLaunching(false);
      launchInFlight.current = false;
    }
  }, [activeSession, appPath]);

  return (
    <div className="border-b border-slate-800 bg-slate-900/80 shrink-0">
      {/* Single compact bar: identity | status | actions | close */}
      <div className="flex items-center gap-2 px-3 py-1.5">
        {/* Left: scenario identity + connection status */}
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <Monitor className="h-3.5 w-3.5 text-blue-400 shrink-0" />
          {scenarioName && (
            <span className="text-xs font-medium text-slate-300 truncate">{scenarioName}</span>
          )}
          <div className="flex items-center gap-1.5 shrink-0">
            <div className={`h-2 w-2 rounded-full ${STATUS_COLORS[connectionStatus]}`} />
            {activeSession && (
              <span className="text-[11px] text-slate-500">
                {activeSession.width}&times;{activeSession.height}
              </span>
            )}
          </div>
        </div>

        {/* Right: action buttons */}
        <div className="flex items-center gap-1 shrink-0">
          {/* Launch App */}
          {connectionStatus === "connected" && (
            <button
              type="button"
              onClick={() => void handleLaunchApp()}
              disabled={launching}
              className="flex items-center gap-1.5 rounded px-2 py-1 text-xs font-medium text-emerald-300 border border-emerald-800/50 hover:bg-emerald-950/30 hover:text-emerald-200 disabled:opacity-50 disabled:cursor-not-allowed transition"
            >
              <Play className="h-3 w-3" />
              {launching ? "Launching..." : "Launch App"}
            </button>
          )}

          {/* Fullscreen toggle */}
          <button
            type="button"
            onClick={toggleFullscreen}
            className="rounded p-1.5 text-slate-400 hover:bg-slate-800 hover:text-slate-200 transition"
            title={isFullscreen ? "Exit fullscreen" : "Fullscreen"}
          >
            {isFullscreen ? <Minimize2 className="h-3.5 w-3.5" /> : <Maximize2 className="h-3.5 w-3.5" />}
          </button>

          {/* Stop session */}
          <button
            type="button"
            onClick={() => void stopSession()}
            className="rounded p-1.5 text-red-400 hover:bg-red-950/40 hover:text-red-300 transition"
            title="Stop session"
          >
            <Square className="h-3.5 w-3.5" />
          </button>

          {/* Divider */}
          <div className="w-px h-4 bg-slate-700 mx-0.5" />

          {/* Close drawer */}
          <button
            type="button"
            onClick={onClose}
            className="rounded p-1.5 text-slate-400 hover:bg-slate-800 hover:text-slate-200 transition"
            aria-label="Close"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      {/* Inline launch error banner */}
      {launchError && (
        <div className="flex items-center gap-2 px-3 py-1 bg-red-950/40 border-t border-red-800/40">
          <AlertCircle className="h-3 w-3 text-red-400 shrink-0" />
          <span className="text-[11px] text-red-300 truncate flex-1">{launchError}</span>
          <button
            type="button"
            onClick={() => setLaunchError(null)}
            className="text-red-400 hover:text-red-200 p-0.5"
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      )}
    </div>
  );
}
