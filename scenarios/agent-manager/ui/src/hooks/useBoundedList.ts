import { useEffect, useMemo, useState } from "react";

/**
 * Shows the first `initialCount` items and reveals `increment` more on demand,
 * so a long list does not render (or scroll) all at once. The count follows the
 * list when it grows or shrinks.
 */
export function useBoundedList<T>(items: T[], initialCount: number, increment = initialCount) {
  const [visibleCount, setVisibleCount] = useState(() => Math.min(items.length, initialCount));

  useEffect(() => {
    setVisibleCount((current) => {
      if (items.length <= initialCount) return items.length;
      return Math.min(Math.max(current, initialCount), items.length);
    });
  }, [initialCount, items.length]);

  const visibleItems = useMemo(() => items.slice(0, visibleCount), [items, visibleCount]);
  return {
    visibleItems,
    hiddenCount: items.length - visibleItems.length,
    total: items.length,
    showMore: () => setVisibleCount((current) => Math.min(items.length, current + increment)),
  };
}
