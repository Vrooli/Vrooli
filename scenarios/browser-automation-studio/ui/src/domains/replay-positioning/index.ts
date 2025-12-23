import { createContext, useContext, type CSSProperties } from 'react';
import type { ReplayRect, ReplaySize } from '@/domains/replay-layout';

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

export interface OverlayPointRequest {
  point: { x: number; y: number };
  source?: ReplaySize;
  containerId?: string;
}

export interface OverlayRectRequest {
  rect: ReplayRect;
  source?: ReplaySize;
  containerId?: string;
}

const isFiniteNumber = (value: number) => Number.isFinite(value) && !Number.isNaN(value);

const sanitizeSize = (source: ReplaySize | undefined, fallback: ReplaySize): ReplaySize => {
  const width = isFiniteNumber(source?.width ?? 0) ? source?.width ?? fallback.width : fallback.width;
  const height = isFiniteNumber(source?.height ?? 0) ? source?.height ?? fallback.height : fallback.height;
  return {
    width: width > 0 ? width : fallback.width,
    height: height > 0 ? height : fallback.height,
  };
};

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
    let x = 0;
    let y = 0;

    switch (request.position) {
      case 'top-left':
        x = margin;
        y = margin;
        break;
      case 'top-right':
        x = container.width - width - margin;
        y = margin;
        break;
      case 'bottom-left':
        x = margin;
        y = container.height - height - margin;
        break;
      case 'bottom-right':
        x = container.width - width - margin;
        y = container.height - height - margin;
        break;
      case 'center':
      default:
        x = (container.width - width) / 2;
        y = (container.height - height) / 2;
        break;
    }

    return {
      x: container.x + x,
      y: container.y + y,
      width,
      height,
    };
  }

  resolvePoint(request: OverlayPointRequest): { x: number; y: number } | null {
    const container = this.rects.get(request.containerId ?? 'overlay-root');
    if (!container) {
      return null;
    }
    const source = sanitizeSize(request.source, container);
    const { point } = request;
    if (!isFiniteNumber(point.x) || !isFiniteNumber(point.y)) {
      return null;
    }
    return {
      x: container.x + (point.x / source.width) * container.width,
      y: container.y + (point.y / source.height) * container.height,
    };
  }

  resolveRect(request: OverlayRectRequest): ReplayRect | null {
    const container = this.rects.get(request.containerId ?? 'overlay-root');
    if (!container) {
      return null;
    }
    const source = sanitizeSize(request.source, container);
    const { rect } = request;
    if (
      !isFiniteNumber(rect.x)
      || !isFiniteNumber(rect.y)
      || !isFiniteNumber(rect.width)
      || !isFiniteNumber(rect.height)
    ) {
      return null;
    }
    return {
      x: container.x + (rect.x / source.width) * container.width,
      y: container.y + (rect.y / source.height) * container.height,
      width: (rect.width / source.width) * container.width,
      height: (rect.height / source.height) * container.height,
    };
  }
}

export const OverlayRegistryContext = createContext<OverlayRegistry | null>(null);

export const useOverlayRegistry = (): OverlayRegistry | null => {
  return useContext(OverlayRegistryContext);
};

export const toOverlayPointStyle = (
  point: { x: number; y: number } | null | undefined,
): CSSProperties | undefined => {
  if (!point) {
    return undefined;
  }
  return {
    left: `${point.x}px`,
    top: `${point.y}px`,
  };
};

export const toOverlayRectStyle = (
  rect: ReplayRect | null | undefined,
): CSSProperties | undefined => {
  if (!rect) {
    return undefined;
  }
  return {
    left: `${rect.x}px`,
    top: `${rect.y}px`,
    width: `${rect.width}px`,
    height: `${rect.height}px`,
  };
};
