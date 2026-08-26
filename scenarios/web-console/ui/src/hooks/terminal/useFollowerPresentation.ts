import { useMemo } from "react";
import type { Terminal } from "@xterm/xterm";
import { archetypeForGrid } from "../../lib/deviceArchetype";
import { chromeTier, fitDeviceGrid, fitDeviceGridWithControls, fitGrid, screenAperture, surplusRatio, type ChromeTier, type FollowerRect } from "../../lib/followerViewport";

export interface FollowerFrame {
  rect: FollowerRect;
  screenRect: FollowerRect;
  tier: ChromeTier;
  archetype: ReturnType<typeof archetypeForGrid>;
  cols: number;
  rows: number;
}

export function useFollowerPresentation(options: {
  terminal: Terminal | null;
  serverSize: { cols: number; rows: number } | null;
  isFollower: boolean;
  paneSize: { width: number; height: number };
}): FollowerFrame | null {
  const { terminal, serverSize, isFollower, paneSize } = options;
  const frame = useMemo<FollowerFrame | null>(() => {
    const size = serverSize;
    if (!isFollower || !size || paneSize.width <= 0 || paneSize.height <= 0) return null;
    const screen = terminal?.element?.querySelector(".xterm-screen") as HTMLElement | null;
    const measuredAspect = screen && terminal && terminal.cols > 0 && terminal.rows > 0 && screen.clientHeight > 0
      ? (screen.clientWidth / terminal.cols) / (screen.clientHeight / terminal.rows)
      : 0.5;
    const fitted = fitGrid(size.cols, size.rows, paneSize.width, paneSize.height, measuredAspect);
    let tier = chromeTier(surplusRatio(fitted, paneSize.width, paneSize.height), fitted.scale);
    const archetype = archetypeForGrid(size.cols, size.rows, measuredAspect);
    let device = fitDeviceGridWithControls(size.cols, size.rows, paneSize.width, paneSize.height, measuredAspect, archetype, tier);
    if (tier !== "strip" && device.frame.height < 140) {
      tier = "strip";
      device = fitDeviceGrid(size.cols, size.rows, paneSize.width, paneSize.height, measuredAspect, screenAperture(archetype, tier));
    }
    return { rect: device.frame, screenRect: device.screen, tier, archetype, cols: size.cols, rows: size.rows };
  }, [serverSize, isFollower, paneSize, terminal]);

  return frame;
}
