import { describe, expect, it } from 'vitest';
import { OverlayRegistry } from '../index';

describe('OverlayRegistry', () => {
  it('resolves anchors within the default overlay root', () => {
    const registry = new OverlayRegistry();
    registry.setRect('overlay-root', { x: 0, y: 0, width: 200, height: 100 });

    const rect = registry.resolveAnchor({
      id: 'watermark',
      position: 'top-right',
      size: { width: 20, height: 10 },
      margin: 10,
    });

    expect(rect).toEqual({ x: 170, y: 10, width: 20, height: 10 });
  });

  it('maps points from source dimensions into overlay space', () => {
    const registry = new OverlayRegistry();
    registry.setRect('overlay-root', { x: 0, y: 0, width: 300, height: 150 });

    const resolved = registry.resolvePoint({
      point: { x: 50, y: 25 },
      source: { width: 100, height: 50 },
    });

    expect(resolved?.x).toBeCloseTo(150, 3);
    expect(resolved?.y).toBeCloseTo(75, 3);
  });

  it('maps rects from source dimensions into overlay space', () => {
    const registry = new OverlayRegistry();
    registry.setRect('overlay-root', { x: 0, y: 0, width: 400, height: 200 });

    const resolved = registry.resolveRect({
      rect: { x: 50, y: 20, width: 100, height: 40 },
      source: { width: 200, height: 100 },
    });

    expect(resolved?.x).toBeCloseTo(100, 3);
    expect(resolved?.y).toBeCloseTo(40, 3);
    expect(resolved?.width).toBeCloseTo(200, 3);
    expect(resolved?.height).toBeCloseTo(80, 3);
  });
});
