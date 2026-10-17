import { Square, Maximize2, Minimize2, ArrowLeft, Monitor } from "lucide-react";
import { useState, useCallback, useEffect, type RefObject } from "react";
import { useSessionStore } from "../../store/sessionStore";
import type { ConnectionStatus } from "../../lib/api/sessions";
import { SessionControlsMenu } from "./SessionControlsMenu";
import { MetricsBar } from "./MetricsBar";

const STATUS_COLORS: Record<ConnectionStatus, string> = {
  disconnected: "bg-slate-500",
  connecting: "bg-amber-400 animate-pulse",
  connected: "bg-emerald-400",
  reconnecting: "bg-amber-400 animate-pulse",
  failed: "bg-red-400",
};

interface SessionToolbarProps {
  fullscreenTargetRef: RefObject<HTMLDivElement | null>;
  onStop: () => void;
  onBack: () => void;
}

export function SessionToolbar({ fullscreenTargetRef, onStop, onBack }: SessionToolbarProps) {
  const connectionStatus = useSessionStore((s) => s.connectionStatus);
  const activeSession = useSessionStore((s) => s.activeSession);
  const [isFullscreen, setIsFullscreen] = useState(false);

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

  return (
    <div className="border-b border-slate-800 bg-slate-900/80 shrink-0">
      <div className="flex items-center gap-2 px-3 py-1.5">
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <button
            type="button"
            onClick={onBack}
            className="rounded p-1.5 text-slate-400 hover:bg-slate-800 hover:text-slate-200 transition"
            aria-label="Back to sessions"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
          </button>
          <Monitor className="h-3.5 w-3.5 text-blue-400 shrink-0" />
          {activeSession?.scenario_name && (
            <span className="text-xs font-medium text-slate-300 truncate">{activeSession.scenario_name}</span>
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

        <div className="flex items-center gap-1 shrink-0">
          {connectionStatus === "connected" && <SessionControlsMenu />}

          <button
            type="button"
            onClick={toggleFullscreen}
            className="rounded p-1.5 text-slate-400 hover:bg-slate-800 hover:text-slate-200 transition"
            title={isFullscreen ? "Exit fullscreen" : "Fullscreen"}
          >
            {isFullscreen ? <Minimize2 className="h-3.5 w-3.5" /> : <Maximize2 className="h-3.5 w-3.5" />}
          </button>

          <button
            type="button"
            onClick={onStop}
            className="rounded p-1.5 text-red-400 hover:bg-red-950/40 hover:text-red-300 transition"
            title="Stop session"
          >
            <Square className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>
      {connectionStatus === "connected" && <MetricsBar />}
    </div>
  );
}
