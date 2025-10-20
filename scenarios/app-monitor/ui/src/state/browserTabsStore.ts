import { create } from 'zustand';
import { createJSONStorage, persist, type StateStorage } from 'zustand/middleware';

import { ensureDataUrl } from '@/utils/dataUrl';
import { useSurfaceMediaStore } from './surfaceMediaStore';

export interface BrowserTabRecord {
  id: string;
  title: string;
  url: string;
  createdAt: number;
  lastActiveAt: number;
  screenshotData?: string | null;
  screenshotWidth?: number | null;
  screenshotHeight?: number | null;
  screenshotNote?: string | null;
  faviconUrl?: string | null;
}

export interface BrowserTabHistoryRecord extends BrowserTabRecord {
  closedAt: number;
}

interface BrowserTabsState {
  activeTabId: string | null;
  tabs: BrowserTabRecord[];
  history: BrowserTabHistoryRecord[];
  openTab: (input: { url: string; title?: string }) => BrowserTabRecord;
  closeTab: (id: string) => void;
  activateTab: (id: string) => void;
  updateTab: (id: string, data: Partial<BrowserTabRecord>) => void;
  clear: () => void;
}

type BrowserTabsPersistedState = Pick<BrowserTabsState, 'activeTabId' | 'tabs' | 'history'>;

const HISTORY_LIMIT = 50;

const noopStorage: StateStorage = {
  getItem: () => null,
  setItem: () => undefined,
  removeItem: () => undefined,
};

const browserTabsStorage = createJSONStorage<BrowserTabsPersistedState>(() => {
  if (typeof window !== 'undefined' && window.localStorage) {
    return window.localStorage;
  }
  return noopStorage;
});

const createRecord = ({ url, title }: { url: string; title?: string }): BrowserTabRecord => {
  const now = Date.now();
  const identifier = typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID()
    : `tab-${Math.random().toString(36).slice(2)}`;
  return {
    id: identifier,
    url,
    title: title?.trim() || url,
    createdAt: now,
    lastActiveAt: now,
    screenshotData: null,
    screenshotWidth: null,
    screenshotHeight: null,
    screenshotNote: null,
    faviconUrl: null,
  };
};

const BROWSER_TABS_STORAGE_KEY = 'app-monitor:browser-tabs';

const isFiniteNumber = (value: unknown): value is number => typeof value === 'number' && Number.isFinite(value);

const normalizeTabRecord = (input: unknown): BrowserTabRecord | null => {
  if (!input || typeof input !== 'object') {
    return null;
  }
  const record = input as Partial<BrowserTabRecord>;
  if (typeof record.id !== 'string' || typeof record.url !== 'string') {
    return null;
  }
  const createdAt = isFiniteNumber(record.createdAt) ? record.createdAt : Date.now();
  const lastActiveAt = isFiniteNumber(record.lastActiveAt) ? record.lastActiveAt : createdAt;
  const normalizedScreenshot = ensureDataUrl(record.screenshotData ?? null);

  return {
    id: record.id,
    url: record.url,
    title: typeof record.title === 'string' && record.title.trim() ? record.title.trim() : record.url,
    createdAt,
    lastActiveAt,
    screenshotData: normalizedScreenshot ?? null,
    screenshotWidth: isFiniteNumber(record.screenshotWidth) ? record.screenshotWidth : null,
    screenshotHeight: isFiniteNumber(record.screenshotHeight) ? record.screenshotHeight : null,
    screenshotNote: typeof record.screenshotNote === 'string' ? record.screenshotNote : null,
    faviconUrl: typeof record.faviconUrl === 'string' ? record.faviconUrl : null,
  };
};

const normalizeHistoryRecord = (input: unknown): BrowserTabHistoryRecord | null => {
  const base = normalizeTabRecord(input);
  if (!base) {
    return null;
  }
  const record = input as Partial<BrowserTabHistoryRecord>;
  const closedAt = isFiniteNumber(record.closedAt) ? record.closedAt : base.lastActiveAt;
  return {
    ...base,
    closedAt,
  };
};

const syncSurfaceScreenshots = (records: Array<BrowserTabRecord | BrowserTabHistoryRecord>) => {
  const setSurfaceScreenshot = useSurfaceMediaStore.getState().setScreenshot;
  records.forEach(record => {
    const dataUrl = ensureDataUrl(record.screenshotData ?? null);
    if (!dataUrl) {
      return;
    }
    setSurfaceScreenshot('web', record.id, {
      dataUrl,
      width: record.screenshotWidth ?? 0,
      height: record.screenshotHeight ?? 0,
      capturedAt: record.lastActiveAt || record.createdAt,
      note: record.screenshotNote ?? 'Restored snapshot',
      source: 'restored',
    });
  });
};

export const useBrowserTabsStore = create<BrowserTabsState>()(persist(
  (set) => ({
    activeTabId: null,
    tabs: [],
    history: [],

    openTab: ({ url, title }) => {
      const record = createRecord({ url, title });
      set((state) => ({
        activeTabId: record.id,
        tabs: [...state.tabs, record],
      }));
      return record;
    },

    closeTab: (id) => {
      const setSurfaceScreenshot = useSurfaceMediaStore.getState().setScreenshot;
      set((state) => {
        const target = state.tabs.find(tab => tab.id === id);
        const remaining = state.tabs.filter(tab => tab.id !== id);
        const historyEntry = target
          ? [{ ...target, closedAt: Date.now() }, ...state.history].slice(0, HISTORY_LIMIT)
          : state.history;
        const nextActive = state.activeTabId === id ? (remaining[0]?.id ?? null) : state.activeTabId;
        return {
          tabs: remaining,
          history: historyEntry,
          activeTabId: nextActive,
        };
      });
      setSurfaceScreenshot('web', id, null);
    },

    activateTab: (id) => {
      set((state) => ({
        activeTabId: id,
        tabs: state.tabs.map(tab => tab.id === id ? { ...tab, lastActiveAt: Date.now() } : tab),
      }));
    },

    updateTab: (id, data) => {
      const hasScreenshotUpdate = Object.prototype.hasOwnProperty.call(data, 'screenshotData');
      const normalizedScreenshotData = hasScreenshotUpdate
        ? ensureDataUrl(data.screenshotData ?? null)
        : undefined;

      set((state) => {
        const patch = { ...data } as Partial<BrowserTabRecord>;
        if (hasScreenshotUpdate) {
          patch.screenshotData = normalizedScreenshotData ?? null;
        }
        return {
          tabs: state.tabs.map(tab => (tab.id === id ? { ...tab, ...patch } : tab)),
          history: state.history.map(entry => (entry.id === id ? { ...entry, ...patch } : entry)),
        };
      });

      if (!hasScreenshotUpdate) {
        return;
      }

      const setSurfaceScreenshot = useSurfaceMediaStore.getState().setScreenshot;
      if (normalizedScreenshotData) {
        setSurfaceScreenshot('web', id, {
          dataUrl: normalizedScreenshotData,
          width: data.screenshotWidth ?? 0,
          height: data.screenshotHeight ?? 0,
          capturedAt: Date.now(),
          note: data.screenshotNote ?? null,
          source: 'bridge',
        });
      } else {
        setSurfaceScreenshot('web', id, null);
      }
    },

    clear: () => {
      useSurfaceMediaStore.getState().clear('web');
      set({ tabs: [], history: [], activeTabId: null });
    },
  }),
  {
    name: BROWSER_TABS_STORAGE_KEY,
    version: 2,
    storage: browserTabsStorage,
    partialize: state => ({
      activeTabId: state.activeTabId,
      tabs: state.tabs,
      history: state.history,
    }),
    migrate: (persisted) => {
      if (!persisted || typeof persisted !== 'object') {
        return {
          activeTabId: null,
          tabs: [],
          history: [],
        } satisfies BrowserTabsPersistedState;
      }

      const baseState = persisted as Partial<BrowserTabsPersistedState>;
      const tabs = Array.isArray(baseState.tabs)
        ? baseState.tabs.map(normalizeTabRecord).filter((record): record is BrowserTabRecord => Boolean(record))
        : [];
      const history = Array.isArray(baseState.history)
        ? baseState.history
          .map(normalizeHistoryRecord)
          .filter((entry): entry is BrowserTabHistoryRecord => Boolean(entry))
          .slice(0, HISTORY_LIMIT)
        : [];
      const activeTabId = typeof baseState.activeTabId === 'string'
        && tabs.some(tab => tab.id === baseState.activeTabId)
        ? baseState.activeTabId
        : (tabs[0]?.id ?? null);

      return {
        activeTabId,
        tabs,
        history,
      } satisfies BrowserTabsPersistedState;
    },
    onRehydrateStorage: () => (state) => {
      if (!state) {
        return;
      }
      queueMicrotask(() => {
        syncSurfaceScreenshots([...state.tabs, ...state.history]);
      });
    },
  },
));
