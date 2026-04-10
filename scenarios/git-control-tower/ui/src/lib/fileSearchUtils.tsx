import type { ContentSearchMatch } from "./api";

export type SearchMode = "files" | "content";

// localStorage keys
export const FILE_HISTORY_KEY = "git-control-tower:file-search-history";
export const SEARCH_MODE_KEY = "git-control-tower:search-mode";
export const MAX_HISTORY_ITEMS = 20;

// Helper to get history from localStorage
export function getFileHistory(): string[] {
  try {
    const stored = localStorage.getItem(FILE_HISTORY_KEY);
    if (!stored) return [];
    const parsed: unknown = JSON.parse(stored);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item): item is string => typeof item === "string");
  } catch {
    return [];
  }
}

// Helper to add a file to history
export function addToFileHistory(path: string): void {
  try {
    const history = getFileHistory();
    const filtered = history.filter((p) => p !== path);
    const updated = [path, ...filtered].slice(0, MAX_HISTORY_ITEMS);
    localStorage.setItem(FILE_HISTORY_KEY, JSON.stringify(updated));
  } catch {
    // Ignore localStorage errors
  }
}

// Get saved search mode preference
export function getSavedSearchMode(): SearchMode {
  try {
    const saved = localStorage.getItem(SEARCH_MODE_KEY);
    if (saved === "files" || saved === "content") return saved;
  } catch {
    // Ignore
  }
  return "files";
}

// Save search mode preference
export function saveSearchMode(mode: SearchMode): void {
  try {
    localStorage.setItem(SEARCH_MODE_KEY, mode);
  } catch {
    // Ignore
  }
}

// Simple fuzzy matching for file paths
export function fuzzyMatch(path: string, query: string): boolean {
  if (!query) return true;
  const lowerPath = path.toLowerCase();
  const lowerQuery = query.toLowerCase();

  let pathIdx = 0;
  for (let i = 0; i < lowerQuery.length; i++) {
    const char = lowerQuery.charAt(i);
    const found = lowerPath.indexOf(char, pathIdx);
    if (found === -1) return false;
    pathIdx = found + 1;
  }
  return true;
}

// Score fuzzy match for sorting (higher is better)
export function fuzzyScore(path: string, query: string): number {
  if (!query) return 0;
  const lowerPath = path.toLowerCase();
  const lowerQuery = query.toLowerCase();

  let score = 0;
  let pathIdx = 0;
  let consecutive = 0;

  for (let i = 0; i < lowerQuery.length; i++) {
    const char = lowerQuery.charAt(i);
    const found = lowerPath.indexOf(char, pathIdx);
    if (found === -1) return -1;

    if (found === pathIdx) {
      consecutive++;
      score += consecutive * 2;
    } else {
      consecutive = 0;
    }

    if (found === 0 || lowerPath[found - 1] === "/") {
      score += 5;
    }

    score -= (found - pathIdx) * 0.5;
    pathIdx = found + 1;
  }

  score -= path.length * 0.1;

  const filename = path.split("/").pop() || "";
  if (filename.toLowerCase().includes(lowerQuery)) {
    score += 10;
  }

  return score;
}

// Get file icon color based on language
export function getFileIconColor(language?: string): string {
  switch (language) {
    case "typescript":
      return "text-blue-400";
    case "javascript":
      return "text-yellow-400";
    case "go":
      return "text-cyan-400";
    case "python":
      return "text-green-400";
    case "rust":
      return "text-orange-400";
    case "java":
    case "kotlin":
      return "text-red-400";
    case "css":
    case "scss":
    case "less":
      return "text-pink-400";
    case "html":
      return "text-orange-300";
    case "json":
    case "yaml":
    case "toml":
      return "text-purple-400";
    case "markdown":
      return "text-slate-400";
    default:
      return "text-slate-500";
  }
}

// Group content search results by file
export interface GroupedMatches {
  path: string;
  matches: ContentSearchMatch[];
}

export function groupMatchesByFile(matches: ContentSearchMatch[]): GroupedMatches[] {
  const groups: Map<string, ContentSearchMatch[]> = new Map();

  for (const match of matches) {
    const existing = groups.get(match.path) || [];
    existing.push(match);
    groups.set(match.path, existing);
  }

  return Array.from(groups.entries()).map(([path, matches]) => ({
    path,
    matches
  }));
}

// Extract a window of content centered around the match
export function extractMatchWindow(
  content: string,
  query: string,
  isRegex: boolean,
  caseSensitive: boolean,
  maxLength: number = 80
): { text: string; matchStart: number; matchEnd: number; hasEllipsisBefore: boolean; hasEllipsisAfter: boolean } | null {
  if (!query) return null;

  try {
    const flags = caseSensitive ? "" : "i";
    const pattern = isRegex ? query : query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const regex = new RegExp(pattern, flags);
    const match = content.match(regex);

    if (!match || match.index === undefined) {
      return null;
    }

    const matchStart = match.index;
    const matchEnd = matchStart + match[0].length;

    if (content.length <= maxLength) {
      return {
        text: content,
        matchStart,
        matchEnd,
        hasEllipsisBefore: false,
        hasEllipsisAfter: false
      };
    }

    const matchCenter = matchStart + match[0].length / 2;
    const halfWindow = Math.floor(maxLength / 2);

    let windowStart = Math.floor(matchCenter - halfWindow);
    let windowEnd = Math.ceil(matchCenter + halfWindow);

    if (windowStart < 0) {
      windowEnd = Math.min(content.length, windowEnd - windowStart);
      windowStart = 0;
    } else if (windowEnd > content.length) {
      windowStart = Math.max(0, windowStart - (windowEnd - content.length));
      windowEnd = content.length;
    }

    if (matchStart < windowStart) {
      windowStart = matchStart;
      windowEnd = Math.min(content.length, windowStart + maxLength);
    }
    if (matchEnd > windowEnd) {
      windowEnd = matchEnd;
      windowStart = Math.max(0, windowEnd - maxLength);
    }

    return {
      text: content.slice(windowStart, windowEnd),
      matchStart: matchStart - windowStart,
      matchEnd: matchEnd - windowStart,
      hasEllipsisBefore: windowStart > 0,
      hasEllipsisAfter: windowEnd < content.length
    };
  } catch {
    return null;
  }
}

// Highlight search query in content, centered around the match
export function highlightContent(content: string, query: string, isRegex: boolean, caseSensitive: boolean): React.ReactNode {
  if (!query) return content;

  const window = extractMatchWindow(content, query, isRegex, caseSensitive);

  if (!window) {
    return content.length > 80 ? content.slice(0, 80) + "\u2026" : content;
  }

  const { text, matchStart, matchEnd, hasEllipsisBefore, hasEllipsisAfter } = window;

  const before = text.slice(0, matchStart);
  const matchText = text.slice(matchStart, matchEnd);
  const after = text.slice(matchEnd);

  return (
    <>
      {hasEllipsisBefore && <span className="text-slate-500">{"\u2026"}</span>}
      {before}
      <mark className="bg-amber-500/30 text-amber-200 rounded px-0.5">
        {matchText}
      </mark>
      {after}
      {hasEllipsisAfter && <span className="text-slate-500">{"\u2026"}</span>}
    </>
  );
}
