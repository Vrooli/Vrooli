import type { LayoutSection } from "./components/LayoutSettingsModal";

export const layoutOrder: LayoutSection[] = ["changes", "history", "diff", "commit"];

export type SelectionEntry = { path: string; staged: boolean };

/** Pure helper: compute the next selection given a mode ("single" | "toggle" | "range"). */
export function computeNextSelection(
  nextKey: string,
  lastKey: string | null,
  mode: "single" | "toggle" | "range",
  currentSelection: SelectionEntry[],
  orderedIndexMap: Map<string, number>,
  orderedKeys: string[],
  orderedKeyToEntry: Map<string, SelectionEntry>,
  selectionKey: (entry: SelectionEntry) => string,
): SelectionEntry[] {
  const nextEntry = orderedKeyToEntry.get(nextKey) ?? { path: nextKey.slice(2), staged: nextKey.startsWith("1:") };

  if (mode === "range" && lastKey && orderedIndexMap.has(lastKey) && orderedIndexMap.has(nextKey)) {
    const start = orderedIndexMap.get(lastKey) ?? 0;
    const end = orderedIndexMap.get(nextKey) ?? 0;
    const [from, to] = start < end ? [start, end] : [end, start];
    return orderedKeys
      .slice(from, to + 1)
      .map((key) => orderedKeyToEntry.get(key))
      .filter((entry): entry is SelectionEntry => Boolean(entry));
  }

  if (mode === "toggle") {
    const hasEntry = currentSelection.some((entry) => selectionKey(entry) === nextKey);
    if (hasEntry) {
      return currentSelection.filter((entry) => selectionKey(entry) !== nextKey);
    }
    return [...currentSelection, nextEntry].sort((a, b) => {
      const aIndex = orderedIndexMap.get(selectionKey(a)) ?? 0;
      const bIndex = orderedIndexMap.get(selectionKey(b)) ?? 0;
      return aIndex - bIndex;
    });
  }

  // single
  return [nextEntry];
}
