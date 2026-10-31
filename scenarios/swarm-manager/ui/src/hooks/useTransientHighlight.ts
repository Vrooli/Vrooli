/**
 * useTransientHighlight
 *
 * Apply a transient CSS class to a DOM element looked up by attribute, then
 * clear it after `durationMs`. Cleans up timers on unmount and on target
 * change. The hook only manages the side-effect — callers control which
 * highlight class to apply.
 *
 * Looking up by attribute (rather than an element ref) survives component
 * refactors: the orientation flow only needs to know
 * `data-orientation-target="acceptance-criteria"` exists somewhere in the
 * DOM, not which component renders it.
 */

import { useEffect } from "react";

export interface UseTransientHighlightOptions {
  /** Selector to look up the target. e.g. "[data-orientation-target='acceptance-criteria']". */
  targetSelector: string | null;
  /** Class added to the matched element while the highlight is active. */
  highlightClass: string;
  /** How long the highlight stays before being removed. Default 1500 ms. */
  durationMs?: number;
  /** When set to true, scroll the matched element into view before highlighting. */
  scrollIntoView?: boolean;
}

export function useTransientHighlight({
  targetSelector,
  highlightClass,
  durationMs = 1500,
  scrollIntoView = true,
}: UseTransientHighlightOptions) {
  useEffect(() => {
    if (!targetSelector) return;
    if (typeof document === "undefined") return;
    const element = document.querySelector(targetSelector);
    if (!element) return;
    if (scrollIntoView && typeof element.scrollIntoView === "function") {
      element.scrollIntoView({ behavior: "smooth", block: "center" });
    }
    // classList.add accepts only single class tokens; split on whitespace so
    // callers can pass a "ring-2 ring-cyan-400/60"-style multi-class string.
    const tokens = highlightClass.split(/\s+/).filter(Boolean);
    for (const token of tokens) element.classList.add(token);
    const timer = window.setTimeout(() => {
      for (const token of tokens) element.classList.remove(token);
    }, durationMs);
    return () => {
      window.clearTimeout(timer);
      for (const token of tokens) element.classList.remove(token);
    };
  }, [targetSelector, highlightClass, durationMs, scrollIntoView]);
}
