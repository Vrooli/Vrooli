import { useCallback, useEffect, useState } from 'react';

const SHOW_LINE_NUMBERS_KEY = 'ecosystem-manager.editor.showLineNumbers';
const PREFERENCE_EVENT = 'ecosystem-manager:editor-preferences-updated';

function readShowLineNumbers(): boolean {
  if (typeof window === 'undefined') return true;
  const stored = localStorage.getItem(SHOW_LINE_NUMBERS_KEY);
  if (stored === null) return true;
  return stored === 'true';
}

export function useEditorPreferences() {
  const [showLineNumbers, setShowLineNumbersState] = useState<boolean>(() => readShowLineNumbers());

  useEffect(() => {
    const syncFromStorage = () => {
      setShowLineNumbersState(readShowLineNumbers());
    };

    const handleStorage = (event: StorageEvent) => {
      if (!event.key || event.key === SHOW_LINE_NUMBERS_KEY) {
        syncFromStorage();
      }
    };

    window.addEventListener('storage', handleStorage);
    window.addEventListener(PREFERENCE_EVENT, syncFromStorage);

    return () => {
      window.removeEventListener('storage', handleStorage);
      window.removeEventListener(PREFERENCE_EVENT, syncFromStorage);
    };
  }, []);

  const setShowLineNumbers = useCallback((value: boolean) => {
    if (typeof window === 'undefined') return;
    localStorage.setItem(SHOW_LINE_NUMBERS_KEY, String(value));
    setShowLineNumbersState(value);
    window.dispatchEvent(new Event(PREFERENCE_EVENT));
  }, []);

  return {
    showLineNumbers,
    setShowLineNumbers,
  };
}
