/**
 * @libraryId react-component-library:useDirection
 * @displayName useDirection
 * @description Subscribes to the writing direction in force for a subtree so a direction change re-renders its consumers.
 * @version 2.1.3
 * @tags ["runtime","accessibility","internationalization"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-direction */
import { useEffect, useState, type RefObject } from "react";

import {
  normalizeWritingDirection,
  type WritingDirection,
} from "@vrooli/react-component-library/GestureDirection/1";

export type { WritingDirection };

const readDirection = (element: Element | null | undefined): WritingDirection => {
  if (typeof document === "undefined") return "ltr";
  const target = element ?? document.documentElement;

  // The nearest declared `dir` is checked first, and it is the authoritative
  // one: HTML directionality is defined by that attribute, it is what an author
  // sets to mirror a region, and `closest` resolves the inheritance that
  // reading the element's own attribute would miss.
  const declared = target.closest("[dir]");
  if (declared) {
    const value = declared.getAttribute("dir");
    // `dir="auto"` defers to the content, which only the engine can resolve, so
    // fall through to the computed value rather than guessing.
    if (value === "rtl" || value === "ltr") return value;
  }

  // Direction set through a stylesheet rather than an attribute. This is also
  // the read SidebarShell performed privately before it was routed through this
  // hook, so the fallback preserves the answer that component already had.
  if (typeof window !== "undefined" && typeof window.getComputedStyle === "function") {
    return normalizeWritingDirection(window.getComputedStyle(target).direction);
  }
  return normalizeWritingDirection(target.getAttribute("dir"));
};

/**
 * The writing direction in force for `elementRef`, re-read whenever it changes.
 *
 * Pass the ref of the surface being mirrored. Without one the hook answers for
 * the document as a whole, which is right for app-level chrome and wrong for
 * anything that can sit inside a mirrored region.
 *
 * The first render cannot see the element: refs are attached after it, so a
 * render-time read resolves against the document and reports the wrong side for
 * a surface inside a mirrored region. The effect below re-reads once the ref is
 * live, which is also where the observer that watches for later changes is set
 * up — one code path for "read it now" and "read it again", so they cannot
 * drift apart.
 *
 * This answers a locale question only. It does not say which side a panel is
 * anchored to — that is an ergonomic preference carried by `useHandedness`, and
 * the two are combined by `resolveGestureDirection`.
 */
export function useDirection(elementRef?: RefObject<Element | null>): WritingDirection {
  const [direction, setDirection] = useState<WritingDirection>(() =>
    readDirection(elementRef?.current),
  );

  useEffect(() => {
    const read = () => {
      const next = readDirection(elementRef?.current);
      // Guarded so an unrelated attribute change cannot re-render every
      // consumer for a value that did not move.
      setDirection((previous) => (previous === next ? previous : next));
    };
    read();

    if (typeof document === "undefined" || typeof MutationObserver === "undefined") return;
    const observer = new MutationObserver(read);
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["dir", "lang"],
      subtree: true,
    });
    return () => {
      observer.disconnect();
    };
  }, [elementRef]);

  return direction;
}
