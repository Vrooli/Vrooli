import { useCallback, useRef, useState } from "react";
import { Monitor, Loader2, AlertCircle, RotateCcw } from "lucide-react";
import { Drawer, DrawerHeader, DrawerBody } from "../ui/drawer";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { useLiveDesktopStore } from "../../store/liveDesktopStore";
import { VncCanvas } from "./VncCanvas";
import { DesktopToolbar } from "./DesktopToolbar";
import { PlatformSelector } from "./PlatformSelector";
import { selectors } from "../../consts/selectors";
import { Platform } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

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
  const [platform, setPlatform] = useState(Platform.LINUX);
  const [width, setWidth] = useState(1280);
  const [height, setHeight] = useState(720);

  const isDesktopActive =
    activeSession &&
    (connectionStatus === "connected" || connectionStatus === "connecting");

  const handleStart = useCallback(async () => {
    if (!scenarioName) return;
    await startSession({
      width,
      height,
      scenarioName,
      artifactPath: appPath ?? undefined,
      platform,
    });
  }, [scenarioName, appPath, width, height, platform, startSession]);

  const handleRetry = useCallback(() => {
    setError(null);
    void handleStart();
  }, [setError, handleStart]);

  return (
    <Drawer
      open={isOpen}
      onClose={close}
      side="right"
      panelClassName="w-full md:w-[90vw] md:max-w-6xl"
    >
      {/* When desktop is active, skip the header — the toolbar handles everything */}
      {!isDesktopActive && (
        <DrawerHeader className="px-4 py-3">
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-2 min-w-0">
              <Monitor className="h-4 w-4 text-blue-400 shrink-0" />
              <h2 className="text-sm font-semibold text-slate-50 truncate">
                {scenarioName
                  ? `Desktop — ${scenarioName}`
                  : "Interactive Desktop"}
              </h2>
            </div>
            <button
              type="button"
              onClick={close}
              className="rounded p-1 text-slate-400 transition hover:bg-slate-800 hover:text-slate-200"
              aria-label="Close"
              data-testid={selectors.liveDesktop.close}
            >
              <span className="text-lg leading-none">&times;</span>
            </button>
          </div>
        </DrawerHeader>
      )}

      <DrawerBody className="flex flex-col p-0 overflow-hidden">
        {/* No session — config form */}
        {!activeSession && connectionStatus === "disconnected" && !error && (
          <div className="p-5 space-y-4">
            <p className="text-sm text-slate-400">
              Start an interactive desktop session to control the virtual
              display from your browser.
            </p>
            <div className="space-y-1">
              <label className="text-xs text-slate-500">Platform</label>
              <PlatformSelector value={platform} onChange={setPlatform} />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <label
                  htmlFor="desktop-width"
                  className="text-xs text-slate-500"
                >
                  Width
                </label>
                <Input
                  id="desktop-width"
                  type="number"
                  value={width}
                  onChange={(e) => {
                    setWidth(Number(e.target.value) || 1280);
                  }}
                  min={800}
                  max={1920}
                />
              </div>
              <div className="space-y-1">
                <label
                  htmlFor="desktop-height"
                  className="text-xs text-slate-500"
                >
                  Height
                </label>
                <Input
                  id="desktop-height"
                  type="number"
                  value={height}
                  onChange={(e) => {
                    setHeight(Number(e.target.value) || 720);
                  }}
                  min={600}
                  max={1080}
                />
              </div>
            </div>
            <Button
              onClick={() => void handleStart()}
              className="w-full"
              data-testid={selectors.liveDesktop.start}
            >
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
              {activeSession
                ? "Connecting to desktop..."
                : "Starting desktop session..."}
            </p>
          </div>
        )}

        {/* Connected — single toolbar + VNC canvas, no header */}
        {isDesktopActive && (
          <div ref={desktopAreaRef} className="flex flex-col flex-1 min-h-0">
            <DesktopToolbar
              fullscreenTargetRef={desktopAreaRef}
              onClose={close}
            />
            <div className="flex-1 min-h-0 relative">
              <div
                data-testid={selectors.liveDesktop.canvas}
                className="h-full"
              >
                <VncCanvas sessionId={activeSession.sessionId} />
              </div>
            </div>
          </div>
        )}

        {/* Error state */}
        {error && (
          <div
            data-testid={selectors.liveDesktop.error}
            className="p-5 space-y-4"
          >
            <div className="flex items-start gap-2 rounded-lg border border-red-800/60 bg-red-950/30 p-4">
              <AlertCircle className="h-5 w-5 text-red-400 shrink-0 mt-0.5" />
              <div className="space-y-1 min-w-0">
                <p className="text-sm font-medium text-red-300">
                  Desktop Session Error
                </p>
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
