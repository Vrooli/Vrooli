import type { FileViewerViewPreferences } from "@vrooli/file-viewer";

/**
 * The viewer's view options live under one namespaced prefix so a future
 * schema change can be migrated without guessing which localStorage keys were
 * the viewer's. A read/write failure (quota, privacy mode, disabled storage)
 * is silent by design: the option still applies for the current page.
 */
const PREFIX = "web-console.file-viewer.v1.";

export const fileViewerPreferences: FileViewerViewPreferences = {
  read: (key) => {
    try {
      return window.localStorage.getItem(`${PREFIX}${key}`);
    } catch {
      return null;
    }
  },
  write: (key, value) => {
    try {
      window.localStorage.setItem(`${PREFIX}${key}`, value);
    } catch {
      /* not persisted; the in-memory value still applies */
    }
  },
};
