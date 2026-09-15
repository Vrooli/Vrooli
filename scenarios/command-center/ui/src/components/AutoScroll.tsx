import { useEffect, useId, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { useBeatHold, useCycleScale } from "../lib/boardContext";
import { AUTOSCROLL_TIMING, scrollStops, visibleRange, type RowBox } from "../lib/fit";
import { useLandscapeRoom, useReducedMotion } from "../lib/media";

type Phase = "run" | "fading" | "return";

interface Geometry {
  stops: number[];
  /** Rows the position counter names, which may be fewer than the rows it stops at. */
  counted: RowBox[];
  viewport: number;
}

interface AutoScrollProps {
  children: ReactNode;
  /** The rows it steps between. */
  rowSelector: string;
  /** The rows the position counter counts; defaults to the stepped rows. */
  countSelector?: string;
  className?: string;
  label?: string;
  /** Show "3–6 of 9" under the list; off where the rows are unlike each other. */
  counter?: boolean;
}

const boxesOf = (content: HTMLElement, selector: string, origin: number): RowBox[] =>
  Array.from(content.querySelectorAll<HTMLElement>(selector), (row) => {
    const box = row.getBoundingClientRect();
    return { top: box.top - origin, height: box.height };
  }).filter((row) => row.height > 0);

/** Tolerant of sub-pixel churn: a measurement that differs by at most a pixel is the same geometry. */
const sameGeometry = (a: Geometry | null, b: Geometry): boolean =>
  a !== null &&
  Math.abs(a.viewport - b.viewport) <= 1 &&
  a.counted.length === b.counted.length &&
  a.stops.length === b.stops.length &&
  a.stops.every((stop, index) => Math.abs(stop - (b.stops[index] ?? 0)) <= 1);

/**
 * A list that is tall by nature, shown on a screen that never scrolls. It
 * holds the first rows, steps up one row at a time, holds the end, fades back
 * to the top and holds the beat until it has made one full pass. A list that
 * fits never moves, and in portrait the page scrolls instead.
 */
export function AutoScroll({ children, rowSelector, countSelector, className, label, counter = true }: AutoScrollProps) {
  const id = useId();
  const hold = useBeatHold();
  const scale = useCycleScale();
  const landscape = useLandscapeRoom();
  const reduced = useReducedMotion();
  const viewportRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const [geometry, setGeometry] = useState<Geometry | null>(null);
  const [stop, setStop] = useState(0);
  const [phase, setPhase] = useState<Phase>("run");
  const [read, setRead] = useState(false);

  useLayoutEffect(() => {
    const viewport = viewportRef.current;
    const content = contentRef.current;
    if (!landscape || !viewport || !content) {
      setGeometry(null);
      return;
    }
    const measure = () => {
      const box = content.getBoundingClientRect();
      const rows = boxesOf(content, rowSelector, box.top);
      const next: Geometry = {
        stops: scrollStops(rows, box.height, viewport.clientHeight, reduced),
        counted: countSelector ? boxesOf(content, countSelector, box.top) : rows,
        viewport: viewport.clientHeight,
      };
      setGeometry((previous) => (sameGeometry(previous, next) ? previous : next));
    };
    measure();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(measure);
    observer.observe(viewport);
    observer.observe(content);
    return () => observer.disconnect();
  }, [countSelector, landscape, reduced, rowSelector]);

  const last = geometry ? geometry.stops.length - 1 : 0;
  const overflowing = last > 0;
  const current = Math.min(stop, last);

  useEffect(() => {
    hold(id, overflowing && !read);
    return () => hold(id, false);
  }, [hold, id, overflowing, read]);

  useEffect(() => {
    if (!overflowing) {
      setStop(0);
      setPhase("run");
      return;
    }
    // One full pass per reading set: after it has been read the list stays at
    // the top instead of looping and flickering for the rest of the beat.
    if (read) return;
    const { holdStartMs, stepMs, holdEndMs, fadeMs } = AUTOSCROLL_TIMING;
    const fade = reduced ? 0 : fadeMs * scale;
    let timer: number;
    if (phase === "fading") {
      timer = window.setTimeout(() => {
        setStop(0);
        setPhase("return");
      }, fade);
    } else if (phase === "return") {
      timer = window.setTimeout(() => {
        setPhase("run");
        setRead(true);
      }, fade);
    } else {
      const delay = (current === 0 ? holdStartMs : current < last ? stepMs : holdEndMs) * scale;
      timer = window.setTimeout(() => (current < last ? setStop(current + 1) : setPhase("fading")), delay);
    }
    return () => window.clearTimeout(timer);
  }, [current, last, overflowing, phase, read, reduced, scale]);

  const offset = overflowing && geometry ? geometry.stops[current] ?? 0 : 0;
  const end = overflowing && geometry ? geometry.stops[last] ?? 0 : 0;
  const range = overflowing && geometry ? visibleRange(geometry.counted, offset, geometry.viewport) : null;
  return (
    <div className={className ? `cc-autoscroll ${className}` : "cc-autoscroll"} data-overflowing={overflowing || undefined} data-read={read || undefined}>
      <div
        ref={viewportRef}
        className="cc-autoscroll__viewport"
        data-autoscroll-viewport
        data-phase={phase}
        data-more-above={offset > 0.5 || undefined}
        data-more-below={(overflowing && offset < end - 0.5) || undefined}
        aria-label={label}
      >
        <div ref={contentRef} className="cc-autoscroll__content" style={overflowing ? { transform: `translate3d(0, ${-offset}px, 0)` } : undefined}>
          {children}
        </div>
      </div>
      {/* Once scrolling, the counter line always holds its height: if its range
          or presence depended on what fitted, the counter would resize the
          viewport it is measured from and the geometry would never settle. A
          list that fits still shows no counter. */}
      {counter && overflowing && geometry ? <span className="cc-autoscroll__position" data-testid="autoscroll-position" aria-hidden="true">{range ? `${range[0]}–${range[1]} of ${geometry.counted.length}` : "\u00A0"}</span> : null}
    </div>
  );
}
