import type { LayoutSection } from "./components/LayoutSettingsModal";

export const layoutOrder: LayoutSection[] = ["changes", "history", "diff", "commit"];

/** Allocate each sidebar slot from the remaining panels. In particular,
 * choosing Commit as the main panel must leave Diff reachable in the stack. */
export function selectStackSlots(primary: LayoutSection) {
  const remaining = new Set(layoutOrder.filter(section => section !== primary));
  const take = (preferred: LayoutSection, last = false): LayoutSection => {
    const candidates = Array.from(remaining);
    const chosen = remaining.has(preferred) ? preferred : candidates[last ? candidates.length - 1 : 0];
    if (!chosen) throw new Error("Workspace layout requires three distinct stack panels");
    remaining.delete(chosen);
    return chosen;
  };
  const middle = take(remaining.has("history") ? "history" : "diff");
  const bottom = take("commit", true);
  const top = take("changes");
  return { top, middle, bottom };
}

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
