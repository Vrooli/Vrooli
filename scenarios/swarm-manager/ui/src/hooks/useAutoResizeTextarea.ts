/**
 * useAutoResizeTextarea – auto-sizes a <textarea> to fit its content up to a max height.
 *
 * Extracted from the identical inline implementations in quick-capture-input
 * and clarification-panel.
 */

import { useEffect, type RefObject } from "react";

const DEFAULT_MAX_HEIGHT = 200;
const DEFAULT_LINE_HEIGHT = 24;

/**
 * Runs a side-effect that syncs the textarea's height to its scrollHeight
 * every time `value` changes.
 *
 * @param ref   – ref to the textarea element
 * @param value – the current text value (triggers re-measure on change)
 * @param options.maxHeight – clamp height at this value (default 200px)
 * @param options.maxHeightVh – clamp height to this fraction of the viewport
 * @param options.fillHeight – let the surrounding flex layout size the textarea
 * @param options.minRows – minimum number of text rows for this textarea
 * @param options.maxRows – maximum number of text rows before scrolling
 */
export function useAutoResizeTextarea(
  ref: RefObject<HTMLTextAreaElement | null>,
  value: string,
  options?: { maxHeight?: number; maxHeightVh?: number; fillHeight?: boolean; minRows?: number; maxRows?: number },
): void {
  const maxHeightVh = options?.maxHeightVh;
  const maxHeight = options?.maxHeight ?? (maxHeightVh === undefined ? DEFAULT_MAX_HEIGHT : Number.POSITIVE_INFINITY);
  const fillHeight = options?.fillHeight ?? false;
  const minRows = options?.minRows;
  const maxRows = options?.maxRows;

  useEffect(() => {
    const measure = () => {
      const el = ref.current;
      if (!el) return;
      if (fillHeight) {
        el.style.height = "100%";
        return;
      }
      el.style.height = "auto";
      if (minRows !== undefined || maxRows !== undefined) {
        // scrollHeight includes the textarea's padding, so retain it when
        // converting row counts into a pixel clamp. The fallback keeps this
        // hook deterministic in non-layout test environments.
        let lineHeight = DEFAULT_LINE_HEIGHT;
        let padding = 0;
        if (typeof window !== "undefined" && el instanceof window.HTMLElement) {
          const computed = window.getComputedStyle(el);
          const parsedLineHeight = Number.parseFloat(computed.lineHeight);
          if (Number.isFinite(parsedLineHeight)) lineHeight = parsedLineHeight;
          padding = Number.parseFloat(computed.paddingTop) + Number.parseFloat(computed.paddingBottom);
          if (!Number.isFinite(padding)) padding = 0;
        }
        const minimum = minRows === undefined ? 0 : (minRows * lineHeight) + padding;
        const maximum = maxRows === undefined ? Number.POSITIVE_INFINITY : (maxRows * lineHeight) + padding;
        const contentHeight = el.scrollHeight;
        el.style.height = `${Math.max(minimum, Math.min(contentHeight, maximum))}px`;
        el.style.overflowY = contentHeight > maximum ? "auto" : "hidden";
        return;
      }
      const viewportMax = maxHeightVh === undefined ? Number.POSITIVE_INFINITY : (window.innerHeight * maxHeightVh) / 100;
      el.style.height = `${Math.min(el.scrollHeight, maxHeight, viewportMax)}px`;
    };

    measure();
    if ((maxHeightVh === undefined && minRows === undefined && maxRows === undefined) || fillHeight) return;
    window.addEventListener("resize", measure);
    return () => window.removeEventListener("resize", measure);
  }, [ref, value, maxHeight, maxHeightVh, fillHeight, minRows, maxRows]);
}
