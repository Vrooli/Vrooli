import { useCallback, useState } from 'react';
import { addToHistory, loadHistory, saveHistory, type HistoryEntry } from './browserUrlHistory';

/**
 * Hook to manage URL history separately if needed.
 */
export function useUrlHistory() {
  const [history, setHistory] = useState<HistoryEntry[]>(loadHistory);

  const addUrl = useCallback((url: string, title?: string) => {
    const updated = addToHistory(history, url, title);
    setHistory(updated);
    saveHistory(updated);
  }, [history]);

  const removeUrl = useCallback((url: string) => {
    const updated = history.filter((h) => h.url !== url);
    setHistory(updated);
    saveHistory(updated);
  }, [history]);

  const clearHistory = useCallback(() => {
    setHistory([]);
    saveHistory([]);
  }, []);

  return { history, addUrl, removeUrl, clearHistory };
}
