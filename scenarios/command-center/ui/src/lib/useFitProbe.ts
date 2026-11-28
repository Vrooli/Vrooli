import { useEffect, type RefObject } from "react";
import { PORTRAIT_QUERY } from "./media";

/** Long enough for the beat entrance, the strip's measure pass and a digit roll to settle. */
const SETTLE_MS = 1200;

export interface FitReport {
  state: "ok" | "overflow";
  detail: string;
}

interface Box {
  top: number;
  left: number;
  right: number;
  bottom: number;
}

const describe = (element: Element): string => {
  const name = typeof element.className === "string" && element.className ? `.${element.className.split(" ")[0]}` : element.tagName.toLowerCase();
  const text = element.textContent.trim().slice(0, 24);
  return text ? `${name} "${text}"` : name;
};

/**
 * Where a landscape room's figures leave the viewport or the hero runs into the
 * strip. What a clipping ancestor hides (an ellipsis, an auto-scroll viewport)
 * is not on screen to begin with, so only the drawn part of each leaf counts.
 */
export function measureFit(room: Element, viewport = { width: window.innerWidth, height: window.innerHeight }): FitReport {
  const hero = room.querySelector("[data-testid='room-hero']");
  const strip = room.querySelector("[data-testid='room-supporting']");
  const clips = new Map<Element, DOMRect | null>();
  const clipOf = (element: Element): DOMRect | null => {
    if (!clips.has(element)) {
      const style = getComputedStyle(element);
      clips.set(element, style.overflowX !== "visible" || style.overflowY !== "visible" ? element.getBoundingClientRect() : null);
    }
    return clips.get(element) ?? null;
  };
  const drawn = (element: Element, region: Element): Box | null => {
    const rect = element.getBoundingClientRect();
    const box = { top: rect.top, left: rect.left, right: rect.right, bottom: rect.bottom };
    for (let parent = element.parentElement; parent && parent !== region; parent = parent.parentElement) {
      const clip = clipOf(parent);
      if (clip) {
        box.top = Math.max(box.top, clip.top);
        box.left = Math.max(box.left, clip.left);
        box.right = Math.min(box.right, clip.right);
        box.bottom = Math.min(box.bottom, clip.bottom);
      }
      if (box.right - box.left < 2 || box.bottom - box.top < 2) return null;
    }
    return box.right - box.left < 2 || box.bottom - box.top < 2 ? null : box;
  };
  const problems: string[] = [];
  let heroBottom = Number.NEGATIVE_INFINITY;
  for (const [region, name] of [[hero, "hero"], [strip, "strip"]] as const) {
    if (!region) continue;
    for (const element of region.querySelectorAll("*")) {
      if (element.childElementCount > 0 && !element.matches("[data-rcl-figure]")) continue;
      const box = drawn(element, region);
      if (!box) continue;
      if (name === "hero") heroBottom = Math.max(heroBottom, box.bottom);
      if (box.bottom > viewport.height + 1 || box.right > viewport.width + 1 || box.top < -1 || box.left < -1) problems.push(`${name} ${describe(element)} leaves the screen`);
    }
  }
  const stripTop = strip?.querySelector("[data-testid='metric-list']")?.getBoundingClientRect().top;
  if (stripTop !== undefined && heroBottom > stripTop + 2) problems.push(`hero runs ${Math.round(heroBottom - stripTop)}px into the strip`);
  return problems.length ? { state: "overflow", detail: problems.slice(0, 3).join("; ") } : { state: "ok", detail: "" };
}

/**
 * Stamps the room with its own fit: data-fit is "ok", "overflow" (with the
 * first problems in data-fit-detail), "scroll" in portrait, or "pending" while
 * the beat settles. Workflow cases read it; nothing on screen changes.
 */
export function useFitProbe(ref: RefObject<HTMLElement | null>, beatKey: string, dataKey: string): void {
  // Only a new beat is unknown until it settles; a data refresh keeps the last verdict until it is re-checked.
  useEffect(() => {
    if (ref.current) ref.current.dataset.fit = "pending";
  }, [ref, beatKey]);
  useEffect(() => {
    const room = ref.current;
    if (!room) return;
    let timer = 0;
    const check = () => {
      window.clearTimeout(timer);
      timer = window.setTimeout(() => {
        if (window.matchMedia(PORTRAIT_QUERY).matches) {
          room.dataset.fit = "scroll";
          delete room.dataset.fitDetail;
          return;
        }
        const report = measureFit(room);
        room.dataset.fit = report.state;
        if (report.detail) room.dataset.fitDetail = report.detail;
        else delete room.dataset.fitDetail;
      }, SETTLE_MS);
    };
    check();
    window.addEventListener("resize", check);
    // Strip pages and auto-scroll steps change what is on screen; re-check once they settle.
    const observer = typeof MutationObserver === "undefined" ? null : new MutationObserver(check);
    observer?.observe(room, { subtree: true, childList: true, attributes: true, attributeFilter: ["data-phase", "data-paged", "data-density"] });
    return () => {
      window.clearTimeout(timer);
      window.removeEventListener("resize", check);
      observer?.disconnect();
    };
  }, [ref, beatKey, dataKey]);
}
