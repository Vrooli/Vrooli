import { create } from 'zustand';

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

export const useBrowserTabsStore = create<BrowserTabsState>((set) => ({
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
    set((state) => {
      const target = state.tabs.find(tab => tab.id === id);
      const remaining = state.tabs.filter(tab => tab.id !== id);
      const historyEntry = target
        ? [{ ...target, closedAt: Date.now() }, ...state.history].slice(0, 50)
        : state.history;
      const nextActive = state.activeTabId === id ? (remaining[0]?.id ?? null) : state.activeTabId;
      return {
        tabs: remaining,
        history: historyEntry,
        activeTabId: nextActive,
      };
    });
  },

  activateTab: (id) => {
    set((state) => ({
      activeTabId: id,
      tabs: state.tabs.map(tab => tab.id === id ? { ...tab, lastActiveAt: Date.now() } : tab),
    }));
  },

  updateTab: (id, data) => {
    set((state) => ({
      tabs: state.tabs.map(tab => tab.id === id ? { ...tab, ...data } : tab),
      history: state.history.map(entry => entry.id === id ? { ...entry, ...data } : entry),
    }));

    if (Object.prototype.hasOwnProperty.call(data, 'screenshotData')) {
      const setSurfaceScreenshot = useSurfaceMediaStore.getState().setScreenshot;
      if (data.screenshotData) {
        setSurfaceScreenshot('web', id, {
          dataUrl: data.screenshotData,
          width: data.screenshotWidth ?? 0,
          height: data.screenshotHeight ?? 0,
          capturedAt: Date.now(),
          note: data.screenshotNote ?? null,
        });
      } else {
        setSurfaceScreenshot('web', id, null);
      }
    }
  },

  clear: () => set({ tabs: [], history: [], activeTabId: null }),
}));
