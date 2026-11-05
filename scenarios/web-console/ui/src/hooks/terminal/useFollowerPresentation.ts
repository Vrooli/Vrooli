import { useMemo } from "react";
import { resolveArchetype, type DeviceArchetype } from "../../lib/deviceArchetype";
import { FALLBACK_CELL_ASPECT, fitFollowerPresentation, type ApertureRect, type ChromeTier, type FollowerRect } from "../../lib/followerViewport";
import type { FollowerMode } from "../../lib/terminalProtocol";

/**
 * The slice of xterm this hook reads.
 *
 * Narrowing to the three members actually used keeps the hook testable with a
 * plain object instead of a real `Terminal`, which is what the `as never` casts
 * in the suite were paying for.
 */
export interface MeasurableTerminal {
  element?: HTMLElement | undefined;
  cols: number;
  rows: number;
}

export interface FollowerFrame {
  rect: FollowerRect;
  screenRect: FollowerRect;
  /** The device's screen opening. Everything "on screen" is clipped to it. */
  apertureRect: ApertureRect;
  tier: ChromeTier;
  archetype: DeviceArchetype;
  cols: number;
  rows: number;
  /** The leader's virtual keyboard covers part of its viewport. */
  kbOpen: boolean;
  /** Share of the screen aperture the leader's keyboard covers, from the bottom. */
  keyboardShare: number;
  /** Distance below the panel where the caption clears any stand or base. */
  captionOffset: number;
}

/**
 * Measure the rendered cell aspect so a follower's letterboxing matches the
 * leader's real character cell rather than a guess.
 */
export function measureCellAspect(terminal: MeasurableTerminal | null): number {
  const screen = terminal?.element?.querySelector<HTMLElement>(".xterm-screen");
  if (!screen || !terminal || terminal.cols <= 0 || terminal.rows <= 0) return FALLBACK_CELL_ASPECT;
  if (screen.clientHeight <= 0 || screen.clientWidth <= 0) return FALLBACK_CELL_ASPECT;
  return (screen.clientWidth / terminal.cols) / (screen.clientHeight / terminal.rows);
}

export function useFollowerPresentation(options: {
  terminal: MeasurableTerminal | null;
  serverSize: { cols: number; rows: number } | null;
	followerMode: FollowerMode;
	viewMode?: "terminal" | "messages";
  paneSize: { width: number; height: number };
  /** Leader-declared device family. Absent leaders fall back to grid geometry. */
  leaderClass?: string;
  leaderKbOpen?: boolean;
}): FollowerFrame | null {
	const { terminal, serverSize, followerMode, viewMode = "terminal", paneSize, leaderClass, leaderKbOpen } = options;
  return useMemo<FollowerFrame | null>(() => {
    const size = serverSize;
		if (viewMode === "messages" || followerMode !== "follower" || !size || paneSize.width <= 0 || paneSize.height <= 0) return null;
    const cellAspect = measureCellAspect(terminal);
    const archetype = resolveArchetype({
      declaredClass: leaderClass,
      cols: size.cols,
      rows: size.rows,
      cellAspect,
    });
    const layout = fitFollowerPresentation({
      archetype,
      gridCols: size.cols,
      gridRows: size.rows,
      paneWidth: paneSize.width,
      paneHeight: paneSize.height,
      cellAspect,
      kbOpen: leaderKbOpen === true,
    });
    return {
      rect: layout.frame,
      screenRect: layout.screen,
      apertureRect: layout.aperture,
      tier: layout.tier,
      archetype,
      cols: size.cols,
      rows: size.rows,
      kbOpen: leaderKbOpen === true,
      keyboardShare: layout.keyboardShare,
      captionOffset: layout.captionOffset,
    };
	}, [serverSize, followerMode, viewMode, paneSize, terminal, leaderClass, leaderKbOpen]);
}
