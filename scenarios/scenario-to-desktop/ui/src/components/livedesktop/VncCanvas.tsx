import { useEffect, useRef, useCallback } from "react";
import RFB from "@novnc/novnc/lib/rfb";
import { useLiveDesktopStore } from "../../store/liveDesktopStore";
import { buildVncWsUrl } from "../../lib/api/livedesktop";

interface VncCanvasProps {
  sessionId: string;
}

export function VncCanvas({ sessionId }: VncCanvasProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const rfbRef = useRef<RFB | null>(null);
  const setConnectionStatus = useLiveDesktopStore((s) => s.setConnectionStatus);
  const setError = useLiveDesktopStore((s) => s.setError);

  const cleanup = useCallback(() => {
    if (rfbRef.current) {
      rfbRef.current.disconnect();
      rfbRef.current = null;
    }
  }, []);

  useEffect(() => {
    const container = containerRef.current;
    if (!container || !sessionId) return;

    const wsUrl = buildVncWsUrl(sessionId);

    try {
      const rfb = new RFB(container, wsUrl);
      rfbRef.current = rfb;

      rfb.scaleViewport = true;
      rfb.resizeSession = false;
      rfb.clipViewport = false;

      rfb.addEventListener("connect", () => {
        setConnectionStatus("connected");
      });

      rfb.addEventListener("disconnect", (e: CustomEvent) => {
        if (e.detail?.clean) {
          setConnectionStatus("disconnected");
        } else {
          setError("VNC connection lost");
        }
      });

      rfb.addEventListener("securityfailure", (e: CustomEvent) => {
        setError(`VNC security error: ${e.detail?.reason ?? "unknown"}`);
      });

      // Re-scale when container resizes (e.g. fullscreen toggle)
      const observer = new ResizeObserver(() => {
        if (rfbRef.current) {
          rfbRef.current.scaleViewport = true;
        }
      });
      observer.observe(container);

      return () => {
        observer.disconnect();
        cleanup();
      };
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to initialize VNC");
      return cleanup;
    }
  }, [sessionId, setConnectionStatus, setError, cleanup]);

  return (
    <div
      ref={containerRef}
      className="absolute inset-0 bg-black overflow-hidden"
    />
  );
}
