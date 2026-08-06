/** @vrooliComponentSource hooks.use-roving-focus */
import {
  useCallback,
  type KeyboardEvent as ReactKeyboardEvent,
  type RefObject,
} from "react";

export function useRovingFocus<T extends HTMLElement>(
  items: RefObject<T>[],
  activeIndex: number,
  setActiveIndex: (index: number) => void,
) {
  return useCallback(
    (event: ReactKeyboardEvent) => {
      const delta =
        event.key === "ArrowRight" || event.key === "ArrowDown"
          ? 1
          : event.key === "ArrowLeft" || event.key === "ArrowUp"
            ? -1
            : 0;
      if (!delta) return;
      event.preventDefault();
      const next = (activeIndex + delta + items.length) % items.length;
      setActiveIndex(next);
      items[next]?.current?.focus();
    },
    [activeIndex, items, setActiveIndex],
  );
}
