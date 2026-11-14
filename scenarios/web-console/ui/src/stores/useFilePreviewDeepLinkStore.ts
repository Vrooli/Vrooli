import { create } from "zustand";

export interface FilePreviewDeepLinkState {
  path: string | null;
  retryVersion: number;
  request: (path: string) => void;
  clear: () => void;
  retry: () => void;
}

export const useFilePreviewDeepLinkStore = create<FilePreviewDeepLinkState>((set) => ({
  path: null,
  retryVersion: 0,
  request: (path) => { set({ path, retryVersion: 0 }); },
  clear: () => { set({ path: null }); },
  retry: () => { set((state) => ({ retryVersion: state.retryVersion + 1 })); },
}));

/** Read a `?file=` path without coupling the link to any particular session. */
export function parseFilePreviewDeepLink(search: string): string | null {
  const path = new URLSearchParams(search || "").get("file");
  return path && path.trim() ? path : null;
}

/** Consume only the file parameter, retaining approval and unrelated URL state. */
export function stripFilePreviewDeepLink(href: string): string {
  const url = new URL(href);
  url.searchParams.delete("file");
  return `${url.pathname}${url.search}${url.hash}`;
}
