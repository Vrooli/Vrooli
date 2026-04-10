/**
 * useAutoResizeTextarea – auto-sizes a <textarea> to fit its content up to a max height.
 *
 * Extracted from the identical inline implementations in quick-capture-input
 * and clarification-panel.
 */

import { useEffect, type RefObject } from "react";

const DEFAULT_MAX_HEIGHT = 200;

/**
 * Runs a side-effect that syncs the textarea's height to its scrollHeight
 * every time `value` changes.
 *
 * @param ref   – ref to the textarea element
 * @param value – the current text value (triggers re-measure on change)
 * @param options.maxHeight – clamp height at this value (default 200px)
 */
export function useAutoResizeTextarea(
  ref: RefObject<HTMLTextAreaElement | null>,
  value: string,
  options?: { maxHeight?: number },
): void {
  const maxHeight = options?.maxHeight ?? DEFAULT_MAX_HEIGHT;

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, maxHeight)}px`;
  }, [ref, value, maxHeight]);
}
