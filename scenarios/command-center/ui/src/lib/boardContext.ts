import { createContext, useContext } from "react";
import type { GamepadAction } from "@vrooli/iframe-bridge/spatial";
import type { BoardResponse, BoardRoom } from "./api";

/** Four input classes resolve to this one vocabulary before anything reacts. */
export type BoardIntent = GamepadAction | "pause-cycle" | "reveal-controls" | "show-help" | "inspect" | "toggle-fullscreen" | "navigate-beat-prev" | "navigate-beat-next";
export type SamplesMode = "hide" | "mark" | "full";

export interface BoardControllerValue {
  rooms: BoardRoom[];
  board: BoardResponse | undefined;
  samples: SamplesMode;
  paused: boolean;
  controlsVisible: boolean;
  helpVisible: boolean;
  acknowledgement: string;
  cycleSeconds: number;
  beatIndex: number;
  beatDurations: number[];
  transitioning: boolean;
  dispatch: (intent: BoardIntent) => void;
  setSamples: (mode: SamplesMode) => void;
  goTo: (path: string) => void;
  seekCycle: (progress: number) => void;
  selectBeat: (index: number) => void;
  /**
   * Keep the current beat on screen past its authored dwell while `holding`.
   * A paged strip or an auto-scrolling list holds until it has been read once.
   */
  holdBeat: (id: string, holding: boolean) => void;
}

export const BoardContext = createContext<BoardControllerValue | null>(null);

const noHold = () => undefined;

/** The beat hold for components that also render outside a board (Focus, tests): a no-op there. */
export const useBeatHold = (): BoardControllerValue["holdBeat"] => useContext(BoardContext)?.holdBeat ?? noHold;

/**
 * Cycle progress ticks four times a second. It has its own context so a tick
 * re-renders only the cycle rail, not every room surface under the board.
 */
export interface BoardProgress {
  /** 0..1 progress through the current cycle interval. */
  progress: number;
  /** 0..1 progress through the current beat. */
  beatProgress: number;
}

export const BoardProgressContext = createContext<BoardProgress>({ progress: 0, beatProgress: 0 });

export const useBoardProgress = (): BoardProgress => useContext(BoardProgressContext);

export function useBoardController(): BoardControllerValue {
  const value = useContext(BoardContext);
  if (!value) throw new Error("useBoardController must be used inside BoardController");
  return value;
}

export const parseSamples = (value: string | null): SamplesMode => (value === "hide" || value === "full" ? value : "mark");
