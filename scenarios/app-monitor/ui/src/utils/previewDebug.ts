const PREVIEW_DEBUG_QUERY_KEY = 'debugPreviewEvents';
const PREVIEW_DEBUG_STORAGE_KEY = 'app-monitor:debug-preview-events';

export const isPreviewDebugEventsEnabled = (): boolean => {
  if (typeof window === 'undefined') {
    return false;
  }

  const params = new URLSearchParams(window.location.search);
  if (params.get(PREVIEW_DEBUG_QUERY_KEY) === '1') {
    return true;
  }

  return window.localStorage.getItem(PREVIEW_DEBUG_STORAGE_KEY) === '1';
};

export const previewDebugKeys = {
  query: PREVIEW_DEBUG_QUERY_KEY,
  storage: PREVIEW_DEBUG_STORAGE_KEY,
} as const;
