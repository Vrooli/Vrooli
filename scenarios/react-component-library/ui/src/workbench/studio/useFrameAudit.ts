import { useLayoutEffect, useRef, useState, type RefObject } from "react";

import type { DesignFrame } from "../design-types";
import { emptyAudit, type FrameAuditMap } from "./checks";
import { rclPrimitive } from "./rcl";
import type { TemplateNodeType } from "../templates/spec";

export interface PaintedArea {
  rcl: number;
  total: number;
}

// Transformed typography can produce a couple of logical pixels of scroll
// width even when its rendered content fits the page. Treat that tiny spill as
// layout noise while keeping larger measured overflow visible to health checks.
const FRAME_OVERFLOW_TOLERANCE = 2;

/** Count a mapped component's full footprint once, including its authored children. */
export function measurePaintedArea(
  page: HTMLElement,
  pageRect: DOMRect,
  scale: number,
): PaintedArea {
  let rcl = 0;
  let total = 0;
  const factor = (scale || 1) * (scale || 1);
  for (const node of Array.from(page.querySelectorAll<HTMLElement>("[data-node-id]"))) {
    let ancestor =
      node.parentElement?.closest<HTMLElement>("[data-node-id][data-node-type]") ?? null;
    let insideRcl = false;
    while (ancestor) {
      const ancestorType = ancestor.dataset.nodeType as TemplateNodeType | undefined;
      if (ancestorType && rclPrimitive(ancestorType)) {
        insideRcl = true;
        break;
      }
      ancestor =
        ancestor.parentElement?.closest<HTMLElement>("[data-node-id][data-node-type]") ?? null;
    }
    if (insideRcl) continue;

    const type = node.dataset.nodeType as TemplateNodeType | undefined;
    const isRcl = Boolean(type && rclPrimitive(type));
    if (node.querySelector("[data-node-id]") && !isRcl) continue;

    const rect = node.getBoundingClientRect();
    if (rect.width <= 0 || rect.height <= 0) continue;
    const width = Math.min(rect.right, pageRect.right) - Math.max(rect.left, pageRect.left);
    const height = Math.min(rect.bottom, pageRect.bottom) - Math.max(rect.top, pageRect.top);
    if (width <= 0 || height <= 0) continue;
    const area = (width * height) / factor;
    total += area;
    if (isRcl) rcl += area;
  }
  return { rcl, total };
}

/**
 * Measure the rendered frames, not the spec: horizontal overflow, real button
 * heights and painted area only exist once layout has happened. Heights and
 * areas are divided by the frame scale because layout is scaled as a whole.
 *
 * A ResizeObserver keeps the audit honest after fonts, images and responsive
 * reflow change the layout; re-measuring only on a spec edit left the checks
 * describing an older paint.
 */
export function useFrameAudit(
  container: RefObject<HTMLElement>,
  frames: DesignFrame[],
  scales: Record<string, number>,
): FrameAuditMap {
  const [audit, setAudit] = useState<FrameAuditMap>(() => emptyAudit(frames));
  const scalesRef = useRef(scales);
  scalesRef.current = scales;
  const framesRef = useRef(frames);
  framesRef.current = frames;
  // Zoom changes the transformed rects without changing the layout box, so
  // the ResizeObserver cannot see it; re-measure explicitly when it moves.
  const scaleKey = frames.map((frame) => `${frame.id}:${scales[frame.id] ?? 1}`).join("|");

  useLayoutEffect(() => {
    const root = container.current;
    if (!root) return;
    let frameHandle = 0;

    const measure = () => {
      const next: FrameAuditMap = {};
      for (const frame of framesRef.current) {
        const host = root.querySelector<HTMLElement>(`[data-studio-frame="${frame.id}"]`);
        const page = host?.querySelector<HTMLElement>(".tpl-page") ?? null;
        const scale = scalesRef.current[frame.id] ?? 1;
        const measurable = Boolean(page && page.clientWidth > 0);
        const measuredOverflow = page ? page.scrollWidth - page.clientWidth : 0;
        const overflow =
          measurable && measuredOverflow > FRAME_OVERFLOW_TOLERANCE ? measuredOverflow : 0;
        const buttons = page ? Array.from(page.querySelectorAll<HTMLElement>("button")) : [];
        let counted = 0;
        let under = 0;
        for (const button of buttons) {
          const height = button.getBoundingClientRect().height / (scale || 1);
          if (height < 1) continue;
          counted += 1;
          if (height < 44) under += 1;
        }

        // Painted area: only leaves, so a card and its text are not counted
        // twice. Rects are clipped to the page and divided by the scale²
        // because the whole frame is rendered at one zoom.
        let painted: PaintedArea | null = null;
        if (page && measurable) {
          painted = measurePaintedArea(page, page.getBoundingClientRect(), scale);
        }

        next[frame.id] = {
          overflow,
          buttons: counted > 0 ? { total: counted, under } : null,
          painted: painted && painted.total > 0 ? painted : null,
          measured: measurable,
        };
      }
      setAudit(next);
    };

    const schedule = () => {
      if (frameHandle) return;
      frameHandle = window.requestAnimationFrame(() => {
        frameHandle = 0;
        measure();
      });
    };

    measure();
    const observer = new ResizeObserver(schedule);
    for (const frame of framesRef.current) {
      const host = root.querySelector<HTMLElement>(`[data-studio-frame="${frame.id}"]`);
      const page = host?.querySelector<HTMLElement>(".tpl-page");
      if (page) observer.observe(page);
      if (host) observer.observe(host);
    }

    return () => {
      if (frameHandle) window.cancelAnimationFrame(frameHandle);
      observer.disconnect();
    };
  }, [container, frames, scaleKey]);

  return audit;
}

/** Track an element's content width for fit-to-canvas zoom. */
export function useElementWidth(ref: RefObject<HTMLElement>): number {
  const [width, setWidth] = useState(0);
  useLayoutEffect(() => {
    const element = ref.current;
    if (!element) return;
    const update = () => setWidth(element.clientWidth);
    update();
    const observer = new ResizeObserver(update);
    observer.observe(element);
    return () => observer.disconnect();
  }, [ref]);
  return width;
}
