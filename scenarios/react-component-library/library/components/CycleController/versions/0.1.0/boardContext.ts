import { createContext, useContext } from "react";
import type { GamepadAction } from "@vrooli/iframe-bridge/spatial";
import type { BoardResponse, BoardRoom } from "./api";

/** Four input classes resolve to this one vocabulary before anything reacts. */
export type BoardIntent = GamepadAction | "pause-cycle" | "reveal-controls" | "show-help" | "inspect" | "toggle-fullscreen";
export type SamplesMode = "hide" | "mark" | "full";

export interface BoardControllerValue {
  rooms: BoardRoom[];
  board: BoardResponse | undefined;
  samples: SamplesMode;
  paused: boolean;
  controlsVisible: boolean;
  helpVisible: boolean;
  acknowledgement: string;
  /** 0..1 progress through the current cycle interval. */
  progress: number;
  cycleSeconds: number;
  transitioning: boolean;
  dispatch: (intent: BoardIntent) => void;
  setSamples: (mode: SamplesMode) => void;
  goTo: (path: string) => void;
}

export const BoardContext = createContext<BoardControllerValue | null>(null);

export function useBoardController(): BoardControllerValue {
  const value = useContext(BoardContext);
  if (!value) throw new Error("useBoardController must be used inside BoardController");
  return value;
}

export const parseSamples = (value: string | null): SamplesMode => (value === "hide" || value === "full" ? value : "mark");