import { useLayoutEffect, useMemo, type RefObject } from "react";
import { FitAddon } from "@xterm/addon-fit";
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
  fitRef: RefObject<FitAddon | null>;
  serverSize: { cols: number; rows: number } | null;
  isFollower: boolean;
  paneSize: { width: number; height: number };
}): FollowerFrame | null {
  const { terminal, fitRef, serverSize, isFollower, paneSize } = options;
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

  useLayoutEffect(() => {
    const element = terminal?.element;
    const fit = fitRef.current;
    if (!element || !fit || !frame) return;
    const { screenRect } = frame;
    element.style.position = "absolute";
    element.style.left = `${String(screenRect.x)}px`;
    element.style.top = `${String(screenRect.y)}px`;
    element.style.width = `${String(screenRect.width / screenRect.scale)}px`;
    element.style.height = `${String(screenRect.height / screenRect.scale)}px`;
    element.style.transformOrigin = "top left";
    element.style.transform = screenRect.scale < 1 ? `scale(${String(screenRect.scale)})` : "";
    element.style.transition = "left 240ms ease, top 240ms ease, width 240ms ease, height 240ms ease, transform 240ms ease";
    terminal.options.fontSize = screenRect.fontSize;
    fit.fit();
    if (terminal.cols !== frame.cols || terminal.rows !== frame.rows) terminal.resize(frame.cols, frame.rows);
    return () => {
      element.style.position = "";
      element.style.left = "";
      element.style.top = "";
      element.style.width = "";
      element.style.height = "";
      element.style.transformOrigin = "";
      element.style.transform = "";
      element.style.transition = "";
    };
  }, [terminal, fitRef, frame]);
  return frame;
}
