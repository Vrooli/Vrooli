import { createContext } from 'react';
import type { ActualViewportOptional, ViewportDimensions, ViewportSyncState } from '../types/viewport';

export type ActualViewportWithSource = ActualViewportOptional;

export interface ViewportContextState {
  browserViewport: ViewportDimensions | null;
  actualViewport: ActualViewportWithSource | null;
  hasMismatch: boolean;
  mismatchReason: string | null;
  syncState: ViewportSyncState;
  sessionId: string | null;
}

export interface ViewportContextActions {
  updateFromBounds: (bounds: ViewportDimensions) => void;
  setActualViewport: (viewport: ViewportDimensions | null) => void;
  forceSync: () => Promise<void>;
  reset: () => void;
  getClampedViewport: (bounds: ViewportDimensions) => ViewportDimensions;
}

export interface ViewportContextValue extends ViewportContextState, ViewportContextActions {}

export const ViewportContext = createContext<ViewportContextValue | null>(null);
