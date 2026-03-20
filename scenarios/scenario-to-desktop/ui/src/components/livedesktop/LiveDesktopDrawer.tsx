import { useCallback, useRef, useState } from "react";
import { Monitor, Loader2, AlertCircle, RotateCcw } from "lucide-react";
import { Drawer, DrawerHeader, DrawerBody } from "../ui/drawer";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { useLiveDesktopStore } from "../../store/liveDesktopStore";
import { VncCanvas } from "./VncCanvas";
import { DesktopToolbar } from "./DesktopToolbar";

export function LiveDesktopDrawer() {
  const isOpen = useLiveDesktopStore((s) => s.isOpen);
  const close = useLiveDesktopStore((s) => s.close);
  const activeSession = useLiveDesktopStore((s) => s.activeSession);
  const connectionStatus = useLiveDesktopStore((s) => s.connectionStatus);
  const error = useLiveDesktopStore((s) => s.error);
  const scenarioName = useLiveDesktopStore((s) => s.scenarioName);
  const appPath = useLiveDesktopStore((s) => s.appPath);
  const startSession = useLiveDesktopStore((s) => s.startSession);
  const setError = useLiveDesktopStore((s) => s.setError);

  const desktopAreaRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(1280);
  const [height, setHeight] = useState(720);

  const handleStart = useCallback(async () => {
    if (!scenarioName) return;
    await startSession({
      width,
      height,
      scenario_name: scenarioName,
      app_path: appPath ?? undefined,
    });
  }, [scenarioName, appPath, width, height, startSession]);

  const handleRetry = useCallback(() => {
    setError(null);
    void handleStart();
  }, [setError, handleStart]);

  return (
    <Drawer open={isOpen} onClose={close} side="right" panelClassName="w-full md:w-[90vw] md:max-w-6xl">
      <DrawerHeader>
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-2 min-w-0">
            <Monitor className="h-5 w-5 text-blue-400 shrink-0" />
            <div className="min-w-0">
              <h2 className="text-lg font-semibold text-slate-50 truncate">Interactive Desktop</h2>
              {scenarioName && (
                <p className="text-sm text-slate-400 truncate">{scenarioName}</p>
              )}
            </div>
          </div>
          <button
            type="button"
            onClick={close}
            className="rounded-lg p-1.5 text-slate-400 transition hover:bg-slate-800 hover:text-slate-200"
            aria-label="Close"
          >
            <span className="text-xl leading-none">&times;</span>
          </button>
        </div>
      </DrawerHeader>

      <DrawerBody className="flex flex-col gap-4 p-0 overflow-hidden">
        {/* No session — config form */}
        {!activeSession && connectionStatus === "disconnected" && !error && (
          <div className="p-5 space-y-4">
            <p className="text-sm text-slate-400">
              Start an interactive desktop session to control the virtual display from your browser.
            </p>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <label className="text-xs text-slate-500">Width</label>
                <Input
                  type="number"
                  value={width}
                  onChange={(e) => setWidth(Number(e.target.value) || 1280)}
                  min={800}
                  max={1920}
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs text-slate-500">Height</label>
                <Input
                  type="number"
                  value={height}
                  onChange={(e) => setHeight(Number(e.target.value) || 720)}
                  min={600}
                  max={1080}
                />
              </div>
            </div>
            <Button onClick={() => void handleStart()} className="w-full">
              <Monitor className="mr-2 h-4 w-4" />
              Start Session
            </Button>
          </div>
        )}

        {/* Connecting */}
        {connectionStatus === "connecting" && (
          <div className="flex-1 flex flex-col items-center justify-center gap-3 p-5">
            <Loader2 className="h-8 w-8 text-blue-400 animate-spin" />
            <p className="text-sm text-slate-400">
              {activeSession ? "Connecting to desktop..." : "Starting desktop session..."}
            </p>
          </div>
        )}

        {/* Connected — VNC canvas */}
        {activeSession && (connectionStatus === "connected" || connectionStatus === "connecting") && (
          <div ref={desktopAreaRef} className="flex flex-col flex-1 min-h-0">
            <DesktopToolbar fullscreenTargetRef={desktopAreaRef} />
            <div className="flex-1 min-h-0 relative">
              <VncCanvas sessionId={activeSession.id} />
            </div>
          </div>
        )}

        {/* Error state */}
        {error && (
          <div className="p-5 space-y-4">
            <div className="flex items-start gap-2 rounded-lg border border-red-800/60 bg-red-950/30 p-4">
              <AlertCircle className="h-5 w-5 text-red-400 shrink-0 mt-0.5" />
              <div className="space-y-1 min-w-0">
                <p className="text-sm font-medium text-red-300">Desktop Session Error</p>
                <p className="text-xs text-red-400/80 break-all">{error}</p>
              </div>
            </div>
            <Button onClick={handleRetry} variant="outline" className="w-full">
              <RotateCcw className="mr-2 h-4 w-4" />
              Retry
            </Button>
          </div>
        )}
      </DrawerBody>
    </Drawer>
  );
}
