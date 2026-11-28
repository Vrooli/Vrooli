import type { ResizeHandle } from "./region";
import type { RegionCaptureAspect } from "./RegionCaptureControls";

export const HANDLES: readonly ResizeHandle[] = ["nw", "ne", "sw", "se"];
export const ASPECT_RATIOS: Record<RegionCaptureAspect, number | null> = {
  free: null,
  "1:1": 1,
  "4:3": 4 / 3,
  "16:9": 16 / 9,
};

export interface Point {
  x: number;
  y: number;
}

export interface DragState {
  mode: "new" | "move" | "resize";
  pointerId: number;
  start: Point;
  origin: import("./region").Region;
  handle: ResizeHandle;
}

export function percent(value: number, total: number): string {
  if (total <= 0) return "0%";
  return `${(value / total) * 100}%`;
}
