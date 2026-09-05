/**
 * @libraryId react-component-library:useDirection
 * @displayName useDirection
 * @description Production-ready useDirection hook with SSR-safe lifecycle behavior.
 * @version 2.1.0
 * @tags ["runtime","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-direction */
import { useCallback, useSyncExternalStore, type RefObject } from "react";

import {
  normalizeWritingDirection,
  type WritingDirection,
} from "@vrooli/react-component-library/GestureDirection/1.0.0";

export type { WritingDirection };

/**
 * 1.x read `document.documentElement.dir` inline during render and returned it.
 * That is correct exactly once: nothing re-runs when the attribute changes, so
 * a locale switch left every mirrored surface pointing the old way until some
 * unrelated state happened to re-render it. Direction is external mutable state
 * and belongs behind a subscription, the same way `useReducedMotion` already
 * treats the reduced-motion media query.
 *
 * The subtree flag matters as much as the subscription. `dir` is inherited, so
 * a mirrored region anywhere in the document changes what its descendants
 * resolve to; watching only the root would miss every such change and report
 * the document's direction to a component sitting inside an `rtl` island.
 */
const subscribe = (onChange: () => void) => {
  if (
    typeof document === "undefined" ||
    typeof MutationObserver === "undefined"
  ) {
    return () => {};
  }
  const observer = new MutationObserver(onChange);
  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ["dir", "lang"],
    subtree: true,
  });
  return () => {
    observer.disconnect();
  };
};

const readDirection = (
  element: Element | null | undefined,
): WritingDirection => {
  if (typeof document === "undefined") return "ltr";
  const target = element ?? document.documentElement;
  // The computed direction, not the `dir` attribute: an element reports its own
  // declaration only, while what actually applies to it is inherited. Reading
  // the computed value is also what SidebarShell did when it resolved direction
  // privately, so routing that component through this hook cannot change the
  // answer it was already getting.
  if (
    typeof window !== "undefined" &&
    typeof window.getComputedStyle === "function"
  ) {
    return normalizeWritingDirection(window.getComputedStyle(target).direction);
  }
  return normalizeWritingDirection(target.getAttribute?.("dir"));
};

/** Server renders have no document; left-to-right is the safe neutral default. */
const getServerSnapshot = (): WritingDirection => "ltr";

/**
 * The writing direction in force for `elementRef`, re-read whenever it changes.
 *
 * Pass the ref of the surface being mirrored. Without one the hook answers for
 * the document as a whole, which is right for app-level chrome and wrong for
 * anything that can sit inside a mirrored region.
 *
 * This answers a locale question only. It does not say which side a panel is
 * anchored to — that is an ergonomic preference carried by `useHandedness`, and
 * the two are combined by `resolveGestureDirection`.
 */
export function useDirection(
  elementRef?: RefObject<Element | null>,
): WritingDirection {
  const getSnapshot = useCallback(
    () => readDirection(elementRef?.current),
    [elementRef],
  );
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
