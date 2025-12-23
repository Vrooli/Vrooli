import type { ReplayInset, ReplayLayoutInput, ReplayLayoutModel, ReplaySize } from './types';

const FALLBACK_SIZE: ReplaySize = { width: 1280, height: 720 };

// Layout contract:
// - canvas: the presentation surface (export size).
// - viewport: the recorded browser viewport (logical size).
// - display: the rendered canvas size after fitting to container.
// - viewportRect: the browser viewport placement inside the display.
// - frameRect: the browser frame placement including chrome header height.
// - contentInset: padding reserved by the background container before layout.

const normalizeSize = (value: ReplaySize | undefined, fallback: ReplaySize): ReplaySize => {
  const width = value?.width ?? fallback.width;
  const height = value?.height ?? fallback.height;
  return {
    width: Number.isFinite(width) && width > 0 ? width : fallback.width,
    height: Number.isFinite(height) && height > 0 ? height : fallback.height,
  };
};

const normalizeInset = (value?: ReplayInset): ReplayInset => {
  const x = Number.isFinite(value?.x) ? Math.max(0, value?.x ?? 0) : 0;
  const y = Number.isFinite(value?.y) ? Math.max(0, value?.y ?? 0) : 0;
  return { x, y };
};

export const computeReplayLayout = (input: ReplayLayoutInput): ReplayLayoutModel => {
  const canvas = normalizeSize(input.canvas, FALLBACK_SIZE);
  const viewport = normalizeSize(input.viewport, canvas);
  const contentInset = normalizeInset(input.contentInset);
  const containerOuter = input.container ? normalizeSize(input.container, canvas) : undefined;
  const container = containerOuter
    ? {
        width: Math.max(1, containerOuter.width - contentInset.x * 2),
        height: Math.max(1, containerOuter.height - contentInset.y * 2),
      }
    : undefined;
  const fit = input.fit ?? 'contain';
  const chromeHeaderHeight = Number.isFinite(input.chromeHeaderHeight)
    ? Math.max(0, input.chromeHeaderHeight ?? 0)
    : 0;

  const scale = fit === 'contain' && container
    ? Math.min(container.width / canvas.width, container.height / canvas.height, 1)
    : 1;

  const display = {
    width: Math.max(1, Math.round(canvas.width * scale)),
    height: Math.max(1, Math.round(canvas.height * scale)),
  };

  const availableHeight = Math.max(1, display.height - chromeHeaderHeight);
  const aspect = viewport.width > 0 ? viewport.height / viewport.width : 0.5625;
  const rawBrowserScale = typeof input.browserScale === 'number' ? input.browserScale : 1;
  const targetViewportWidth = display.width * Math.max(0, Math.min(1, rawBrowserScale));
  const maxViewportWidthByHeight = aspect > 0
    ? availableHeight / aspect
    : display.width;
  const viewportWidth = Math.max(1, Math.min(targetViewportWidth, maxViewportWidthByHeight));
  const viewportHeight = Math.max(1, viewportWidth * aspect);
  const viewportRect = {
    x: Math.max(0, (display.width - viewportWidth) / 2),
    y: chromeHeaderHeight + Math.max(0, (availableHeight - viewportHeight) / 2),
    width: viewportWidth,
    height: viewportHeight,
  };
  const viewportScale = viewport.width > 0 ? viewportRect.width / viewport.width : 1;
  const frameRect = {
    x: viewportRect.x,
    y: Math.max(0, viewportRect.y - chromeHeaderHeight),
    width: viewportRect.width,
    height: viewportRect.height + chromeHeaderHeight,
  };

  return {
    canvas,
    viewport,
    display,
    scale,
    viewportRect,
    frameRect,
    chromeHeaderHeight,
    contentInset,
    viewportScale,
  };
};
