/** @vrooliComponentSource hooks.use-roving-focus */
import {
  useCallback,
  type KeyboardEvent as ReactKeyboardEvent,
  type RefObject,
} from "react";

export interface RovingFocusOptions {
  orientation?: "horizontal" | "vertical" | "both";
  loop?: boolean;
  disabledIndices?: number[];
}

export function useRovingFocus<T extends HTMLElement>(
  items: RefObject<T>[],
  activeIndex: number,
  setActiveIndex: (index: number) => void,
  {
    orientation = "both",
    loop = true,
    disabledIndices = [],
  }: RovingFocusOptions = {},
) {
  return useCallback(
    (event: ReactKeyboardEvent) => {
      const horizontal =
        event.key === "ArrowRight" || event.key === "ArrowLeft";
      const vertical = event.key === "ArrowDown" || event.key === "ArrowUp";
      const allowed =
        orientation === "both" ||
        (orientation === "horizontal" ? horizontal : vertical);
      const delta =
        event.key === "ArrowRight" || event.key === "ArrowDown"
          ? 1
          : event.key === "ArrowLeft" || event.key === "ArrowUp"
            ? -1
            : 0;
      const directIndex =
        event.key === "Home" ? 0 : event.key === "End" ? items.length - 1 : -1;
      if (
        (!delta && directIndex < 0) ||
        (delta !== 0 && !allowed) ||
        items.length === 0
      )
        return;
      event.preventDefault();
      const blocked = new Set(disabledIndices);
      const move = (candidate: number, direction: number) => {
        let next = candidate;
        for (let attempts = 0; attempts < items.length; attempts += 1) {
          if (!blocked.has(next)) return next;
          next += direction;
          if (loop) next = (next + items.length) % items.length;
          else if (next < 0 || next >= items.length) return activeIndex;
        }
        return activeIndex;
      };
      const next =
        directIndex >= 0
          ? move(directIndex, directIndex === 0 ? 1 : -1)
          : move(
              loop
                ? (activeIndex + delta + items.length) % items.length
                : activeIndex + delta,
              delta,
            );
      setActiveIndex(next);
      items[next]?.current?.focus();
    },
    [activeIndex, disabledIndices, items, loop, orientation, setActiveIndex],
  );
}
