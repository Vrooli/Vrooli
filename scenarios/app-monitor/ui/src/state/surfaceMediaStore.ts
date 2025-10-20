import { create } from 'zustand';

export type SurfaceType = 'app' | 'web';

export interface SurfaceScreenshotMeta {
  dataUrl: string;
  width: number;
  height: number;
  capturedAt: number;
  note?: string | null;
  source?: 'bridge' | 'client' | 'restored';
}

interface SurfaceMediaState {
  screenshots: Record<string, SurfaceScreenshotMeta>;
  setScreenshot: (type: SurfaceType, id: string, meta: SurfaceScreenshotMeta | null) => void;
  clear: (type?: SurfaceType) => void;
}

const MAX_SURFACE_SCREENSHOTS = 48;

const buildKey = (type: SurfaceType, id: string) => `${type}:${id}`;

export const useSurfaceMediaStore = create<SurfaceMediaState>((set) => ({
  screenshots: {},
  setScreenshot: (type, id, meta) => set((state) => {
    const key = buildKey(type, id);
    const next = { ...state.screenshots };

    if (!meta) {
      delete next[key];
      return { screenshots: next };
    }

    next[key] = meta;
    const entries = Object.entries(next);

    if (entries.length > MAX_SURFACE_SCREENSHOTS) {
      entries.sort((a, b) => b[1].capturedAt - a[1].capturedAt);
      const trimmed = entries.slice(0, MAX_SURFACE_SCREENSHOTS);
      return { screenshots: Object.fromEntries(trimmed) };
    }

    return { screenshots: next };
  }),
  clear: (type) => set((state) => {
    if (!type) {
      return { screenshots: {} };
    }
    const prefix = `${type}:`;
    const filtered = Object.fromEntries(
      Object.entries(state.screenshots).filter(([key]) => !key.startsWith(prefix)),
    );
    return { screenshots: filtered };
  }),
}));

export const selectScreenshotBySurface = (type: SurfaceType, id: string | null | undefined) => {
  if (!id) {
    return () => null;
  }
  const key = buildKey(type, id);
  return (state: SurfaceMediaState) => state.screenshots[key] ?? null;
};

export type { SurfaceMediaState };
