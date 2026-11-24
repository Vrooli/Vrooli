import { useEffect, useMemo, useRef, useState } from "react";
import type { DesignFrame } from "../design-types";
import { FRAME_SIZE } from "../studio/model";
import { useElementWidth, useFrameAudit } from "../studio/useFrameAudit";

export function useWorkspaceStudioViewport(
  frames: DesignFrame[],
  frame: DesignFrame | undefined,
  zoomOverride: number | null,
) {
  const [viewportWidth, setViewportWidth] = useState(() =>
    typeof window === "undefined" ? 1440 : window.innerWidth,
  );
  const canvasRef = useRef<HTMLDivElement>(null);
  const canvasWidth = useElementWidth(canvasRef);
  const narrow = canvasWidth > 0 && canvasWidth < 420;
  const fitScale =
    canvasWidth > 0
      ? Math.min(
          1,
          Math.max(
            0.1,
            Math.min(
              (canvasWidth - 44) / FRAME_SIZE.desktop.width,
              (canvasWidth - 104) / (FRAME_SIZE.tablet.width + FRAME_SIZE.phone.width + 40),
            ),
          ),
        )
      : 0.4;

  const scales = useMemo(() => {
    const map: Record<string, number> = {};
    for (const entry of frames) {
      const base = narrow
        ? Math.min(1, Math.max(0.1, (canvasWidth - 28) / FRAME_SIZE[entry.device].width))
        : fitScale;
      map[entry.id] = Math.min(1, Math.max(0.1, zoomOverride ?? base));
    }
    return map;
  }, [canvasWidth, fitScale, frames, narrow, zoomOverride]);

  const audits = useFrameAudit(canvasRef, frames, scales);
  const frameScale = frame ? (scales[frame.id] ?? fitScale) : fitScale;
  const presentScale = frame
    ? Math.min(1, Math.max(0.1, (viewportWidth - 48) / FRAME_SIZE[frame.device].width))
    : 1;

  useEffect(() => {
    const onResize = () => setViewportWidth(window.innerWidth);
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);

  return { canvasRef, narrow, fitScale, scales, audits, frameScale, presentScale };
}
