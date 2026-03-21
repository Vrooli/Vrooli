import { Square, Maximize2, Minimize2, Play, AlertCircle, X } from "lucide-react";
import { useState, useCallback, useEffect, useRef, type RefObject } from "react";
import { Button } from "../ui/button";
import { useLiveDesktopStore } from "../../store/liveDesktopStore";
import { launchAppOnDesktop } from "../../lib/api/livedesktop";
import type { ConnectionStatus } from "../../lib/api/livedesktop";

const STATUS_COLORS: Record<ConnectionStatus, string> = {
  disconnected: "bg-slate-500",
  connecting: "bg-amber-500 animate-pulse",
  connected: "bg-emerald-500",
  error: "bg-red-500",
};

const STATUS_LABELS: Record<ConnectionStatus, string> = {
  disconnected: "Disconnected",
  connecting: "Connecting...",
  connected: "Connected",
  error: "Error",
};

interface DesktopToolbarProps {
  fullscreenTargetRef: RefObject<HTMLDivElement | null>;
}

export function DesktopToolbar({ fullscreenTargetRef }: DesktopToolbarProps) {
  const connectionStatus = useLiveDesktopStore((s) => s.connectionStatus);
  const activeSession = useLiveDesktopStore((s) => s.activeSession);
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
    <div className="border-b border-slate-800 bg-slate-900/80">
      <div className="flex items-center justify-between gap-3 px-4 py-2">
        <div className="flex items-center gap-3">
          {/* Connection status */}
          <div className="flex items-center gap-2">
            <div className={`h-2.5 w-2.5 rounded-full ${STATUS_COLORS[connectionStatus]}`} />
            <span className="text-xs text-slate-400">{STATUS_LABELS[connectionStatus]}</span>
          </div>

          {/* Resolution */}
          {activeSession && (
            <span className="text-xs text-slate-500">
              {activeSession.width}x{activeSession.height}
            </span>
          )}
        </div>

        <div className="flex items-center gap-2">
          {/* Launch App */}
          {connectionStatus === "connected" && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => void handleLaunchApp()}
              disabled={launching}
              className="border-emerald-800/60 text-emerald-300 hover:bg-emerald-950/30 hover:text-emerald-200"
            >
              <Play className="mr-1.5 h-3.5 w-3.5" />
              {launching ? "Launching..." : "Launch App"}
            </Button>
          )}

          {/* Fullscreen toggle */}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={toggleFullscreen}
            className="text-slate-400 hover:text-slate-200"
          >
            {isFullscreen ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
          </Button>

          {/* Stop session */}
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => void stopSession()}
            className="border-red-800/60 text-red-300 hover:bg-red-950/30 hover:text-red-200"
          >
            <Square className="mr-1.5 h-3.5 w-3.5" />
            Stop Session
          </Button>
        </div>
      </div>

      {/* Inline launch error banner — does NOT destroy the VNC view */}
      {launchError && (
        <div className="flex items-center gap-2 px-4 py-1.5 bg-red-950/40 border-t border-red-800/40">
          <AlertCircle className="h-3.5 w-3.5 text-red-400 shrink-0" />
          <span className="text-xs text-red-300 truncate flex-1">{launchError}</span>
          <button
            type="button"
            onClick={() => setLaunchError(null)}
            className="text-red-400 hover:text-red-200 p-0.5"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      )}
    </div>
  );
}
