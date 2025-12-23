import { describe, expect, it } from 'vitest';
import { computeReplayLayout } from '../compute';

describe('computeReplayLayout', () => {
  it('scales to fit container and centers the viewport', () => {
    const layout = computeReplayLayout({
      canvas: { width: 1280, height: 720 },
      viewport: { width: 1920, height: 1080 },
      browserScale: 0.8,
      container: { width: 640, height: 360 },
      fit: 'contain',
    });

    expect(layout.scale).toBeCloseTo(0.5, 4);
    expect(layout.display).toEqual({ width: 640, height: 360 });
    expect(layout.viewportRect.width).toBeCloseTo(512, 1);
    expect(layout.viewportRect.height).toBeCloseTo(288, 1);
    expect(layout.viewportRect.x).toBeCloseTo(64, 1);
  });

  it('clamps viewport width to fit display height', () => {
    const layout = computeReplayLayout({
      canvas: { width: 1280, height: 720 },
      viewport: { width: 1000, height: 2000 },
      browserScale: 1,
      container: { width: 1280, height: 720 },
      fit: 'contain',
    });

    expect(layout.display).toEqual({ width: 1280, height: 720 });
    expect(layout.viewportRect.width).toBeCloseTo(360, 1);
    expect(layout.viewportRect.height).toBeCloseTo(720, 1);
    expect(layout.viewportRect.x).toBeCloseTo(460, 1);
  });

  it('accounts for chrome header height', () => {
    const layout = computeReplayLayout({
      canvas: { width: 1280, height: 720 },
      viewport: { width: 1280, height: 720 },
      browserScale: 1,
      chromeHeaderHeight: 64,
      container: { width: 1280, height: 720 },
      fit: 'contain',
    });

    expect(layout.viewportRect.y).toBeCloseTo(64, 1);
    expect(layout.frameRect.y).toBeCloseTo(0, 1);
    expect(layout.frameRect.height).toBeCloseTo(720, 1);
  });

  it('fits inside container after applying content inset', () => {
    const layout = computeReplayLayout({
      canvas: { width: 1000, height: 500 },
      viewport: { width: 1000, height: 500 },
      browserScale: 1,
      contentInset: { x: 20, y: 10 },
      container: { width: 1000, height: 500 },
      fit: 'contain',
    });

    expect(layout.display).toEqual({ width: 960, height: 480 });
    expect(layout.viewportRect.x).toBeCloseTo(0, 1);
    expect(layout.viewportRect.y).toBeCloseTo(0, 1);
  });
});
