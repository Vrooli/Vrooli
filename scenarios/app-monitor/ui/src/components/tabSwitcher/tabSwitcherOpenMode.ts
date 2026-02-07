import type { WorkspaceOpenMode } from '@/features/preview-workspace/utils/navigationIntent';

export type AppOpenMode = 'single-preview' | WorkspaceOpenMode;
export type AppOpenModeShortcutResult = AppOpenMode | 'cycle' | null;
export const APP_OPEN_MODE_QUERY_KEY = 'appOpenMode';

export const APP_OPEN_MODES: AppOpenMode[] = ['single-preview', 'replace-focused', 'add-pane'];

export const APP_OPEN_MODE_LABELS: Record<AppOpenMode, string> = {
  'single-preview': 'Single Preview',
  'replace-focused': 'Focused Pane',
  'add-pane': 'New Pane',
};

export const isAppOpenMode = (value: string | null | undefined): value is AppOpenMode => (
  value === 'single-preview' || value === 'replace-focused' || value === 'add-pane'
);

export const cycleAppOpenMode = (current: AppOpenMode): AppOpenMode => {
  const defaultMode: AppOpenMode = 'single-preview';
  const currentIndex = APP_OPEN_MODES.indexOf(current);
  if (currentIndex < 0) {
    return defaultMode;
  }
  const nextIndex = (currentIndex + 1) % APP_OPEN_MODES.length;
  return APP_OPEN_MODES[nextIndex] ?? defaultMode;
};

export const resolveAppOpenModeShortcut = (event: KeyboardEvent): AppOpenModeShortcutResult => {
  if (!event.altKey || event.ctrlKey || event.metaKey) {
    return null;
  }

  const key = event.key.toLowerCase();

  if (key === 'o') {
    return 'cycle';
  }

  if (key === '1') {
    return 'single-preview';
  }

  if (key === '2') {
    return 'replace-focused';
  }

  if (key === '3') {
    return 'add-pane';
  }

  return null;
};
