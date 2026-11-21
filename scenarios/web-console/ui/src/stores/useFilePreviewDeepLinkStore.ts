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

function acceptsDocumentHash(path: string | null): boolean {
  return path !== null && /^[^#]*\.html?$/i.test(path);
}

/** Read a `?file=` path without coupling the link to any particular session. */
export function parseFilePreviewDeepLink(search: string, hash = ""): string | null {
  const path = new URLSearchParams(search || "").get("file");
  if (!path?.trim()) return null;
  return hash && acceptsDocumentHash(path) ? path + hash : path;
}

/** Consume only the file parameter, retaining approval and unrelated URL state. */
export function stripFilePreviewDeepLink(href: string): string {
  const url = new URL(href);
  if (acceptsDocumentHash(url.searchParams.get("file"))) url.hash = "";
  url.searchParams.delete("file");
  return `${url.pathname}${url.search}${url.hash}`;
}
