import { ARTIFACT_VIEWER_MIN_WIDTH, ARTIFACT_WORKSPACE_MIN_WIDTH, type PreviewSourceContext } from "@vrooli/file-viewer";
import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export interface FilePreviewRequest {
  sessionId: string;
  path: string;
  source: PreviewSourceContext;
}

/** A file this browser has opened recently, offered from the viewer's path menu. */
export interface RecentFilePreview {
  sessionId: string;
  /** The resolved path, so reopening does not re-resolve a relative link. */
  path: string;
  basename: string;
  kind: string;
  openedAt: number;
}

const MAX_RECENT_FILES = 12;

export const FILE_PREVIEW_DEFAULT_WIDTH = 440;
export const FILE_PREVIEW_MAX_WIDTH = 900;

export function clampFilePreviewWidth(width: number): number {
  if (!Number.isFinite(width)) return FILE_PREVIEW_DEFAULT_WIDTH;
  return Math.min(FILE_PREVIEW_MAX_WIDTH, Math.max(ARTIFACT_VIEWER_MIN_WIDTH, Math.round(width)));
}

/**
 * A persisted width belongs to the old viewport. On a narrower screen the same
 * pixels would starve the terminal, so cap the dock at the width that still
 * leaves the workspace its own minimum. The stored value is untouched: a
 * desktop load after a phone session gets its chosen width back.
 */
export function fitFilePreviewWidth(width: number, viewportWidth: number): number {
  const available = Math.max(ARTIFACT_VIEWER_MIN_WIDTH, viewportWidth - ARTIFACT_WORKSPACE_MIN_WIDTH);
  return Math.min(clampFilePreviewWidth(width), available);
}

interface FilePreviewState {
  request: FilePreviewRequest | null;
  width: number;
  recent: RecentFilePreview[];
  setRequest: (request: FilePreviewRequest) => void;
  clear: () => void;
  setWidth: (width: number) => void;
  rememberRecent: (entry: Omit<RecentFilePreview, "openedAt"> & { openedAt?: number }) => void;
  forgetRecent: (sessionId: string, path: string) => void;
}

const recentKey = (sessionId: string, path: string) => `${sessionId}\u0000${path}`;

// A quota/private-mode failure must not escape an open/close/resize handler.
// The in-memory store keeps working; only the reload memory is lost.
const guardedLocalStorage = {
  getItem: (name: string): string | null => {
    try { return window.localStorage.getItem(name); } catch { return null; }
  },
  setItem: (name: string, value: string): void => {
    try { window.localStorage.setItem(name, value); } catch { /* not persisted */ }
  },
  removeItem: (name: string): void => {
    try { window.localStorage.removeItem(name); } catch { /* not persisted */ }
  },
};

/**
 * Only layout width is durable. Open requests and recent paths are session
 * scoped and can contain workspace/session identifiers, so keep them in
 * memory and require a fresh user/deep-link action after reload.
 */
export const useFilePreviewStore = create<FilePreviewState>()(
  persist(
    (set) => ({
      request: null,
      width: FILE_PREVIEW_DEFAULT_WIDTH,
      recent: [],
      setRequest: (request) => { set({ request }); },
      clear: () => { set({ request: null }); },
      setWidth: (width) => { set({ width: clampFilePreviewWidth(width) }); },
      rememberRecent: (entry) => {
        set((state) => {
          const next: RecentFilePreview = { ...entry, openedAt: entry.openedAt ?? Date.now() };
          return {
            recent: [
              next,
              ...state.recent.filter((item) => recentKey(item.sessionId, item.path) !== recentKey(next.sessionId, next.path)),
            ].slice(0, MAX_RECENT_FILES),
          };
        });
      },
      forgetRecent: (sessionId, path) => {
        set((state) => ({ recent: state.recent.filter((item) => recentKey(item.sessionId, item.path) !== recentKey(sessionId, path)) }));
      },
    }),
    {
      name: "web-console.file-preview.v1",
      version: 2,
      storage: createJSONStorage(() => guardedLocalStorage),
      partialize: (state) => ({ width: state.width }),
      migrate: (persisted) => {
        const stored = persisted as Partial<Pick<FilePreviewState, "width">> | undefined;
        return { width: clampFilePreviewWidth(stored?.width ?? FILE_PREVIEW_DEFAULT_WIDTH) };
      },
      merge: (persisted, current) => {
        const stored = persisted as Partial<Pick<FilePreviewState, "width">> | undefined;
        return {
          ...current,
          request: null,
          width: clampFilePreviewWidth(stored?.width ?? current.width),
          recent: [],
        };
      },
    },
  ),
);
