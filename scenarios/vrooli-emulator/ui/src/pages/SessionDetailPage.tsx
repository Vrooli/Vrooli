import { useCallback, useEffect, useRef } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { AlertCircle, Loader2 } from "lucide-react";
import { VncCanvas } from "../components/sessions/VncCanvas";
import { SessionToolbar } from "../components/sessions/SessionToolbar";
import { useSessionStore } from "../store/sessionStore";

export function SessionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const desktopAreaRef = useRef<HTMLDivElement>(null);

  const activeSession = useSessionStore((s) => s.activeSession);
  const connectionStatus = useSessionStore((s) => s.connectionStatus);
  const error = useSessionStore((s) => s.error);
  const lastCapture = useSessionStore((s) => s.lastCapture);
  const loadSession = useSessionStore((s) => s.loadSession);
  const stopSession = useSessionStore((s) => s.stopSession);
  const reset = useSessionStore((s) => s.reset);

  useEffect(() => {
    if (!id) return;
    void loadSession(id);
    return () => {
      reset();
    };
  }, [id, loadSession, reset]);

  const goBack = useCallback(() => {
    navigate("/sessions");
  }, [navigate]);

  const handleStop = useCallback(async () => {
    await stopSession();
    navigate("/sessions");
  }, [navigate, stopSession]);

  if (!id) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-950 text-slate-50">
        <p>Missing session id.</p>
      </div>
    );
  }

  if (error && !activeSession) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-slate-950 p-6 text-slate-50">
        <div className="flex w-full max-w-md items-start gap-2 rounded-lg border border-red-800/60 bg-red-950/30 p-4">
          <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-red-400" />
          <div className="space-y-1">
            <p className="text-sm font-medium text-red-300">Unable to load session</p>
            <p className="text-xs text-red-400/80 break-all">{error}</p>
          </div>
        </div>
        <button
          type="button"
          onClick={goBack}
          className="text-sm text-slate-300 underline hover:text-slate-100"
        >
          Back to sessions
        </button>
      </div>
    );
  }

  if (!activeSession) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-slate-950 text-slate-50">
        <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
        <p className="text-sm text-slate-400">Loading session...</p>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen flex-col bg-slate-950 text-slate-50">
      <div ref={desktopAreaRef} className="flex flex-1 min-h-0 flex-col">
        <SessionToolbar fullscreenTargetRef={desktopAreaRef} onStop={() => void handleStop()} onBack={goBack} />

        <div className="relative flex-1 min-h-0">
          <VncCanvas sessionId={activeSession.id} />
        </div>

        {connectionStatus === "connecting" && (
          <div className="border-t border-slate-800 bg-slate-900/60 px-3 py-1.5 text-xs text-slate-400">
            <Loader2 className="mr-2 inline h-3 w-3 animate-spin" />
            Connecting to desktop...
          </div>
        )}

        {error && (
          <div className="flex items-center gap-2 border-t border-red-800/60 bg-red-950/30 px-3 py-1.5 text-xs text-red-300">
            <AlertCircle className="h-3 w-3" />
            <span className="truncate">{error}</span>
          </div>
        )}

        {lastCapture && (
          <div className="border-t border-slate-800 bg-slate-900/60 px-3 py-1.5 text-xs text-slate-300">
            Last {lastCapture.type}
            {lastCapture.path ? ` · ${lastCapture.path}` : ""}
            {lastCapture.url ? ` · ${lastCapture.url}` : ""}
            {" · "}
            {new Date(lastCapture.takenAt).toLocaleTimeString()}
          </div>
        )}
      </div>
    </div>
  );
}
