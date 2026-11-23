import type { LayoutSection } from "./components/LayoutSettingsModal";

export const layoutOrder: LayoutSection[] = ["changes", "history", "diff", "commit"];

export type DesktopPane = Exclude<LayoutSection, "review">;
export type DesktopPaneOrder = [DesktopPane, DesktopPane, DesktopPane, DesktopPane];

const desktopPanes = new Set<DesktopPane>(layoutOrder as DesktopPane[]);

export function normalizeDesktopPaneOrder(value: unknown, fallback: DesktopPaneOrder = ["diff", "changes", "history", "commit"]): DesktopPaneOrder {
  if (!Array.isArray(value)) return fallback;
  const result: DesktopPane[] = [];
  for (const item of value) {
    if (typeof item === "string" && desktopPanes.has(item as DesktopPane) && !result.includes(item as DesktopPane)) {
      result.push(item as DesktopPane);
    }
  }
  for (const pane of fallback) {
    if (!result.includes(pane)) result.push(pane);
  }
  return result.slice(0, 4) as DesktopPaneOrder;
}

export function moveDesktopPane(order: DesktopPaneOrder, from: DesktopPane, to: DesktopPane): DesktopPaneOrder {
  if (from === to) return order;
  const next = [...order];
  const fromIndex = next.indexOf(from);
  const toIndex = next.indexOf(to);
  if (fromIndex < 0 || toIndex < 0) return order;
  const fromPane = next[fromIndex];
  const toPane = next[toIndex];
  if (!fromPane || !toPane) return order;
  next[fromIndex] = toPane;
  next[toIndex] = fromPane;
  return next as DesktopPaneOrder;
}

/** Allocate each sidebar slot from the remaining panels. In particular,
 * choosing Commit as the main panel must leave Diff reachable in the stack. */
export function selectStackSlots(primary: LayoutSection, order: DesktopPaneOrder = ["diff", "changes", "history", "commit"]) {
  const remaining = primary === "review" ? order.filter((pane) => pane !== "diff") : order.filter((pane) => pane !== primary);
  const [top, middle, bottom] = remaining;
  if (!top || !middle || !bottom) throw new Error("Workspace layout requires three distinct stack panels");
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
