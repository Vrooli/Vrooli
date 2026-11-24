import { useCallback, useEffect, useLayoutEffect, useState, type RefObject } from "react";

import type { FixtureStatus } from "./FixtureFrame";
import {
  isTouchTargetSmall,
  measureTouchTargets,
  overflowAt,
} from "./instruments";
import { useInContextContrast } from "./useInContextContrast";
import type { Template } from "../templates/spec";
import type { ViewportPreset } from "./viewports";

interface Mark {
  x: number;
  y: number;
  w: number;
  h: number;
  tone: "red" | "amber";
}

interface TypographyReading {
  fontFamily: string;
  fontSize: string;
  color: string;
}

interface InContextMeasurementsOptions {
  pageRef: RefObject<HTMLDivElement | null>;
  slotWrapRef: RefObject<HTMLDivElement | null>;
  probeRef: RefObject<HTMLDivElement | null>;
  scale: number;
  viewport: ViewportPreset;
  safeTop: number;
  safeBottom: number;
  overflowEnabled: boolean;
  targetsEnabled: boolean;
  contrastEnabled: boolean;
  renderSpec: Template["spec"];
  templateId: string;
  themeSignature: string;
  status: FixtureStatus;
  storyId: string | null;
}

/** Owns DOM measurement, overlay, and contrast-probe state for the context stage. */
export function useInContextMeasurements({
  pageRef,
  slotWrapRef,
  probeRef,
  scale,
  viewport,
  safeTop,
  safeBottom,
  overflowEnabled,
  targetsEnabled,
  contrastEnabled,
  renderSpec,
  templateId,
  themeSignature,
  status,
  storyId,
}: InContextMeasurementsOptions) {
  const [overflow, setOverflow] = useState<
    Array<{ width: number; overflow: boolean; scrollWidth: number }>
  >([]);
  const [marks, setMarks] = useState<Mark[]>([]);
  const [targets, setTargets] = useState<{ total: number; small: number } | null>(null);
  const [typeInfo, setTypeInfo] = useState<{
    inherited: boolean;
    fontFamily: string;
    fontSize: string;
  } | null>(null);
  const [pageType, setPageType] = useState<TypographyReading | null>(null);
  const [safeArea, setSafeArea] = useState<{
    top: number;
    gap: number;
    height: number;
    safeTop: number;
    safeBottom: number;
  } | null>(null);
  const contrast = useInContextContrast({
    pageRef,
    scale,
    enabled: contrastEnabled,
    renderSpec,
    templateId,
    themeSignature,
    status,
    storyId,
  });

  /* ── page-only overflow at the three reference widths ──────────────────── */
  useLayoutEffect(() => {
    const root = probeRef.current;
    if (!root) return;
    const read = () => {
      const rows = Array.from(root.querySelectorAll<HTMLElement>("[data-ic-probe]")).map(
        (frame) => {
          const width = Number(frame.dataset.icProbe);
          // Overflow is scrollWidth against the logical page width. The probe is
          // never scaled, and a screen rect would describe the probe, not the page.
          const scrollWidth = frame.scrollWidth;
          return { width, scrollWidth, overflow: overflowAt(width, scrollWidth) };
        },
      );
      setOverflow((current) => {
        const same =
          current.length === rows.length &&
          current.every((row) => {
            const next = rows.find((entry) => entry.width === row.width);
            return next?.overflow === row.overflow && next.scrollWidth === row.scrollWidth;
          });
        return same ? current : rows;
      });
    };
    read();
    const timer = window.setTimeout(read, 120);
    return () => window.clearTimeout(timer);
  }, [probeRef, renderSpec, templateId]);

  /* ── live measurements inside the frame ───────────────────────────────── */
  const measure = useCallback(() => {
    const host = pageRef.current;
    const root = host?.querySelector<HTMLElement>(".tpl");
    if (!host || !root) return;
    const hostRect = host.getBoundingClientRect();
    const rootRect = root.getBoundingClientRect();
    const factor = scale || 1;
    const toLocal = (value: number) => value / factor;

    const nextMarks: Mark[] = [];

    if (overflowEnabled) {
      const pageWidth = viewport.width;
      for (const element of Array.from(root.querySelectorAll<HTMLElement>("*"))) {
        const rect = element.getBoundingClientRect();
        if (rect.width === 0 || rect.height === 0) continue;
        const left = toLocal(rect.left - rootRect.left);
        const right = left + toLocal(rect.width);
        if (right > pageWidth + 1 || left < -1) {
          nextMarks.push({
            x: left,
            y: toLocal(rect.top - rootRect.top),
            w: toLocal(rect.width),
            h: toLocal(rect.height),
            tone: "red",
          });
        }
        if (nextMarks.length > 80) break;
      }
    }

    if (targetsEnabled) {
      const elements = root.querySelectorAll<HTMLElement>(
        "button, a[href], input, select, textarea, [role='button'], [role='link'], [role='tab'], [tabindex='0']",
      );
      // The frame is zoomed, so the screen rect is not the target's size. The
      // scan divides by the frame's scale to report logical CSS pixels.
      const scan = measureTouchTargets(elements, factor);
      setTargets((current) => {
        if (!scan) return null;
        const next = { total: scan.total, small: scan.small };
        return current && current.total === next.total && current.small === next.small
          ? current
          : next;
      });
      for (const entry of scan?.targets ?? []) {
        if (!isTouchTargetSmall(entry.width, entry.height)) continue;
        nextMarks.push({
          x: toLocal(entry.rect.left - rootRect.left),
          y: toLocal(entry.rect.top - rootRect.top),
          w: entry.width,
          h: entry.height,
          tone: "amber",
        });
      }
    } else {
      setTargets(null);
    }

    setMarks((current) => {
      if (current.length !== nextMarks.length) return nextMarks;
      const same = current.every((mark, index) => {
        const next = nextMarks[index];
        return (
          next &&
          mark.tone === next.tone &&
          Math.round(mark.x) === Math.round(next.x) &&
          Math.round(mark.y) === Math.round(next.y) &&
          Math.round(mark.w) === Math.round(next.w) &&
          Math.round(mark.h) === Math.round(next.h)
        );
      });
      return same ? current : nextMarks;
    });

    // Type inheritance: the slot placeholder carries mono placeholder styles,
    // so the slot body is reset to the page's computed type before we compare.
    const rootStyle = getComputedStyle(root);
    setPageType((current) =>
      current &&
      current.fontFamily === rootStyle.fontFamily &&
      current.fontSize === rootStyle.fontSize &&
      current.color === rootStyle.color
        ? current
        : {
            fontFamily: rootStyle.fontFamily,
            fontSize: rootStyle.fontSize,
            color: rootStyle.color,
          },
    );

    const typeElement = slotWrapRef.current ?? host.querySelector<HTMLElement>(".tpl-slot");
    if (typeElement) {
      const slotStyle = getComputedStyle(typeElement);
      const family = slotStyle.fontFamily || "";
      const rootFamily = rootStyle.fontFamily || "";
      const inherited = family.length > 0 && family === rootFamily;
      const next = { inherited, fontFamily: family || "—", fontSize: slotStyle.fontSize || "—" };
      setTypeInfo((current) =>
        current &&
        current.inherited === next.inherited &&
        current.fontFamily === next.fontFamily &&
        current.fontSize === next.fontSize
          ? current
          : next,
      );

      // Safe area: the frame insets the page; the content must start after it.
      const page = root.querySelector<HTMLElement>(".tpl-page") ?? root;
      const first = page.firstElementChild as HTMLElement | null;
      const scroll = host.scrollTop || 0;
      const top = first ? toLocal(first.getBoundingClientRect().top - hostRect.top) + scroll : 0;
      const gap = Number.parseFloat(getComputedStyle(host).paddingBottom) || 0;
      const nextSafe = { top, gap, height: viewport.height, safeTop, safeBottom };
      setSafeArea((current) =>
        current &&
        Math.abs(current.top - nextSafe.top) < 0.5 &&
        Math.abs(current.gap - nextSafe.gap) < 0.5 &&
        current.height === nextSafe.height &&
        current.safeTop === nextSafe.safeTop
          ? current
          : nextSafe,
      );
    }
  }, [
    overflowEnabled,
    pageRef,
    safeBottom,
    safeTop,
    scale,
    slotWrapRef,
    targetsEnabled,
    viewport.height,
    viewport.width,
  ]);

  useLayoutEffect(() => {
    if (typeof window === "undefined") return;
    const raf = window.requestAnimationFrame(measure);
    return () => window.cancelAnimationFrame(raf);
  }, [measure, pageType, renderSpec, status, storyId, contrast]);

  useEffect(() => {
    const host = pageRef.current;
    if (!host || typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(() => measure());
    observer.observe(host);
    const root = host.querySelector<HTMLElement>(".tpl");
    if (root) observer.observe(root);
    return () => observer.disconnect();
  }, [measure, pageRef, renderSpec]);

  return { overflow, marks, targets, typeInfo, pageType, safeArea, contrast };
}
