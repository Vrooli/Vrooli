/**
 * @libraryId react-component-library:useDirection
 * @displayName useDirection
 * @description Production-ready useDirection hook with SSR-safe lifecycle behavior.
 * @version 2.0.0
 * @tags ["runtime","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-direction */
import { useSyncExternalStore } from "react";

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
 */
const subscribe = (onChange: () => void) => {
  if (
    typeof document === "undefined" ||
    typeof MutationObserver === "undefined"
  ) {
    return () => {};
  }
  // `dir` is inherited, so a change on either the root or the body can alter
  // what a subtree resolves to. Watching both costs one observer and closes the
  // common case where an app sets direction on `body`.
  const observer = new MutationObserver(onChange);
  const options: MutationObserverInit = {
    attributes: true,
    attributeFilter: ["dir", "lang"],
  };
  observer.observe(document.documentElement, options);
  if (document.body) observer.observe(document.body, options);
  return () => {
    observer.disconnect();
  };
};

const getSnapshot = (): WritingDirection => {
  if (typeof document === "undefined") return "ltr";
  // `dir` on the element reports only what that element declares. The computed
  // direction is what actually applies, and it is what the gesture code in
  // SidebarShell has always compared against — reading the same source keeps
  // the hook and the components that bypassed it from disagreeing.
  const root = document.documentElement;
  if (
    typeof window !== "undefined" &&
    typeof window.getComputedStyle === "function"
  ) {
    return normalizeWritingDirection(window.getComputedStyle(root).direction);
  }
  return normalizeWritingDirection(root.dir);
};

/** Server renders have no document; left-to-right is the safe neutral default. */
const getServerSnapshot = (): WritingDirection => "ltr";

/**
 * The document's current writing direction, re-read whenever it changes.
 *
 * This answers a locale question only. It does not say which side a panel is
 * anchored to — that is an ergonomic preference carried by `useHandedness`, and
 * the two are combined by `resolveGestureDirection`.
 */
export function useDirection(): WritingDirection {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
