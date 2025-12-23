import { createContext, useContext } from 'react';
import type { ReplayRect } from './types';

export type OverlayAnchor =
  | 'top-left'
  | 'top-right'
  | 'bottom-left'
  | 'bottom-right'
  | 'center';

export interface OverlayAnchorRequest {
  id: string;
  containerId?: string;
  position: OverlayAnchor;
  size: { width: number; height: number };
  margin?: number;
}

export class OverlayRegistry {
  private rects = new Map<string, ReplayRect>();

  setRect(id: string, rect: ReplayRect) {
    this.rects.set(id, rect);
  }

  getRect(id: string): ReplayRect | undefined {
    return this.rects.get(id);
  }

  resolveAnchor(request: OverlayAnchorRequest): ReplayRect | null {
    const container = this.rects.get(request.containerId ?? 'overlay-root');
    if (!container) {
      return null;
    }
    const margin = Math.max(0, request.margin ?? 0);
    const width = Math.max(0, request.size.width);
    const height = Math.max(0, request.size.height);
    let x = container.x;
    let y = container.y;

    switch (request.position) {
      case 'top-left':
        x = container.x + margin;
        y = container.y + margin;
        break;
      case 'top-right':
        x = container.x + container.width - width - margin;
        y = container.y + margin;
        break;
      case 'bottom-left':
        x = container.x + margin;
        y = container.y + container.height - height - margin;
        break;
      case 'bottom-right':
        x = container.x + container.width - width - margin;
        y = container.y + container.height - height - margin;
        break;
      case 'center':
      default:
        x = container.x + (container.width - width) / 2;
        y = container.y + (container.height - height) / 2;
        break;
    }

    return {
      x,
      y,
      width,
      height,
    };
  }
}

export const OverlayRegistryContext = createContext<OverlayRegistry | null>(null);

export const useOverlayRegistry = (): OverlayRegistry | null => {
  return useContext(OverlayRegistryContext);
};
