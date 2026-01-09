import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type SurfaceType = 'app' | 'web' | 'resource';

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
const SURFACE_MEDIA_STORAGE_KEY = 'app-monitor:surface-media';
type SurfaceMediaPersistedState = Pick<SurfaceMediaState, 'screenshots'>;

const buildKey = (type: SurfaceType, id: string) => `${type}:${id}`;

const pruneScreenshots = (input: Record<string, SurfaceScreenshotMeta>): Record<string, SurfaceScreenshotMeta> => {
  const entries = Object.entries(input);
  if (entries.length <= MAX_SURFACE_SCREENSHOTS) {
    return input;
  }
  entries.sort((a, b) => b[1].capturedAt - a[1].capturedAt);
  return Object.fromEntries(entries.slice(0, MAX_SURFACE_SCREENSHOTS));
};

const resolveStorage = () => (
  typeof window !== 'undefined'
    ? createJSONStorage<SurfaceMediaPersistedState>(() => window.localStorage)
    : undefined
);

export const useSurfaceMediaStore = create<SurfaceMediaState>()(persist(
  (set) => ({
    screenshots: {},
    setScreenshot: (type, id, meta) => set((state) => {
      const key = buildKey(type, id);
      const next = { ...state.screenshots };

      if (!meta) {
        delete next[key];
        return { screenshots: next };
      }

      next[key] = meta;
      return { screenshots: pruneScreenshots(next) };
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
  }),
  {
    name: SURFACE_MEDIA_STORAGE_KEY,
    version: 1,
    storage: resolveStorage(),
    partialize: state => ({ screenshots: state.screenshots }),
    migrate: (persisted) => {
      if (!persisted || typeof persisted !== 'object') {
        return { screenshots: {} } satisfies SurfaceMediaPersistedState;
      }
      const rawScreenshots = (persisted as SurfaceMediaPersistedState).screenshots ?? {};
      return { screenshots: pruneScreenshots(rawScreenshots) } satisfies SurfaceMediaPersistedState;
    },
  },
));

export const selectScreenshotBySurface = (type: SurfaceType, id: string | null | undefined) => {
  if (!id) {
    return () => null;
  }
  const key = buildKey(type, id);
  return (state: SurfaceMediaState) => state.screenshots[key] ?? null;
};

export type { SurfaceMediaState };
