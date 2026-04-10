export interface HistoryEntry {
  url: string;
  title?: string;
  visitCount: number;
  lastVisited: number;
}

const URL_HISTORY_STORAGE_KEY = 'browser-automation-studio:url-history';
const MAX_HISTORY_ITEMS = 50;

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const parseHistoryEntry = (value: unknown): HistoryEntry | null => {
  if (!isRecord(value)) return null;
  if (typeof value.url !== 'string' || value.url.length === 0) return null;
  const visitCount = typeof value.visitCount === 'number' ? value.visitCount : 1;
  const lastVisited = typeof value.lastVisited === 'number' ? value.lastVisited : Date.now();
  const entry: HistoryEntry = {
    url: value.url,
    visitCount,
    lastVisited,
  };
  if (typeof value.title === 'string') {
    entry.title = value.title;
  }
  return entry;
};

export function loadHistory(): HistoryEntry[] {
  try {
    const stored = localStorage.getItem(URL_HISTORY_STORAGE_KEY);
    if (!stored) {
      return [];
    }
    const parsed: unknown = JSON.parse(stored);
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed
      .map(parseHistoryEntry)
      .filter((entry): entry is HistoryEntry => entry !== null)
      .slice(0, MAX_HISTORY_ITEMS);
  } catch {
    // Invalid or unavailable
  }
  return [];
}

export function saveHistory(history: HistoryEntry[]): void {
  try {
    localStorage.setItem(
      URL_HISTORY_STORAGE_KEY,
      JSON.stringify(history.slice(0, MAX_HISTORY_ITEMS))
    );
  } catch {
    // localStorage unavailable
  }
}

export function addToHistory(history: HistoryEntry[], url: string, title?: string): HistoryEntry[] {
  const normalizedUrl = url.toLowerCase();
  const existingIndex = history.findIndex((e) => e.url.toLowerCase() === normalizedUrl);
  const existingEntry = existingIndex >= 0 ? history[existingIndex] : undefined;

  const newEntry: HistoryEntry = {
    url,
    title,
    visitCount: existingEntry ? existingEntry.visitCount + 1 : 1,
    lastVisited: Date.now(),
  };

  const updated = existingIndex >= 0
    ? [...history.slice(0, existingIndex), ...history.slice(existingIndex + 1)]
    : [...history];

  updated.unshift(newEntry);

  return updated.slice(0, MAX_HISTORY_ITEMS);
}

export function scoreHistoryMatch(entry: HistoryEntry, query: string): number {
  const lowerUrl = entry.url.toLowerCase();
  const lowerQuery = query.toLowerCase();

  let score = 0;

  // Exact match bonus
  if (lowerUrl === lowerQuery) {
    score += 1000;
  }

  // Starts with query (after protocol)
  const urlWithoutProtocol = lowerUrl.replace(/^https?:\/\//, '');
  if (urlWithoutProtocol.startsWith(lowerQuery)) {
    score += 500;
  }

  // Contains query
  if (lowerUrl.includes(lowerQuery)) {
    score += 100;
  }

  // Recency bonus (decay over 30 days)
  const daysSinceVisit = (Date.now() - entry.lastVisited) / (1000 * 60 * 60 * 24);
  score += Math.max(0, 50 - daysSinceVisit);

  // Frequency bonus
  score += Math.min(entry.visitCount * 5, 50);

  return score;
}
