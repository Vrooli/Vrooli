export type ReplaySize = { width: number; height: number };
export type ReplayRect = { x: number; y: number; width: number; height: number };
export type ReplayFitMode = 'contain' | 'none';
export type ReplayInset = { x: number; y: number };

export interface ReplayLayoutInput {
  canvas: ReplaySize;
  viewport: ReplaySize;
  browserScale: number;
  chromeHeaderHeight?: number;
  contentInset?: ReplayInset;
  container?: ReplaySize;
  fit?: ReplayFitMode;
}

export interface ReplayLayoutModel {
  canvas: ReplaySize;
  viewport: ReplaySize;
  display: ReplaySize;
  scale: number;
  viewportRect: ReplayRect;
  frameRect: ReplayRect;
  chromeHeaderHeight: number;
  contentInset: ReplayInset;
  viewportScale: number;
}
