import type { ExecutionViewportSettings, ViewportPreset } from '@stores/workflowStore';
import { VIEWPORT_PRESETS } from '@shared/ui';

const DEFAULT_DESKTOP_VIEWPORT: ExecutionViewportSettings = {
  width: 1920,
  height: 1080,
  preset: 'desktop',
};

const MIN_VIEWPORT_DIMENSION = 320;
const MAX_VIEWPORT_DIMENSION = 3840;

export const clampViewportDimension = (value: number): number => {
  if (!Number.isFinite(value)) {
    return MIN_VIEWPORT_DIMENSION;
  }
  return Math.min(
    Math.max(Math.round(value), MIN_VIEWPORT_DIMENSION),
    MAX_VIEWPORT_DIMENSION
  );
};

export const determineViewportPreset = (width: number, height: number): ViewportPreset => {
  if (!Number.isFinite(width) || !Number.isFinite(height)) {
    return 'custom';
  }
  const matchingPreset = VIEWPORT_PRESETS.find(
    (p) => p.width === width && p.height === height
  );
  if (matchingPreset) {
    if (width === 1920 && height === 1080) return 'desktop';
    if (width <= 500) return 'mobile';
  }
  return 'custom';
};

export const normalizeViewportSetting = (
  viewport?: ExecutionViewportSettings | null
): ExecutionViewportSettings => {
  if (
    !viewport ||
    !Number.isFinite(viewport.width) ||
    !Number.isFinite(viewport.height)
  ) {
    return { ...DEFAULT_DESKTOP_VIEWPORT };
  }
  const width = clampViewportDimension(viewport.width);
  const height = clampViewportDimension(viewport.height);
  const preset = viewport.preset ?? determineViewportPreset(width, height);
  return { width, height, preset };
};
