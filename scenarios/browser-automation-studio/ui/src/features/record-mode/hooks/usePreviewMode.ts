import { useEffect, useState } from 'react';

type PreviewMode = 'live' | 'snapshot';

const STORAGE_KEY = 'record-mode-preview-mode';

/**
 * Persisted preview mode selector for Record Mode.
 */
export function usePreviewMode(): [PreviewMode, (mode: PreviewMode) => void] {
  const [mode, setMode] = useState<PreviewMode>('live');

  useEffect(() => {
    try {
      const stored = window.localStorage.getItem(STORAGE_KEY) as PreviewMode | null;
      if (stored === 'live' || stored === 'snapshot') {
        setMode(stored);
      }
    } catch {
      // Ignore storage errors (private mode, etc.)
    }
  }, []);

  const updateMode = (next: PreviewMode) => {
    setMode(next);
    try {
      window.localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // Non-fatal if we cannot persist
    }
  };

  return [mode, updateMode];
}

export type { PreviewMode };
