import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import {
  distance,
  fitTransform,
  midpoint,
  panBy,
  parseViewBox,
  resetTransform,
  zoomAroundPoint,
  ZOOM_STEP,
  type Size,
  type Transform,
} from "./zoomTransform";

export interface MermaidZoomSurfaceHandle {
  zoomIn: () => void;
  zoomOut: () => void;
  fit: () => void;
  reset: () => void;
}

interface MermaidZoomSurfaceProps {
  /** Pre-rendered Mermaid SVG markup. */
  svgHtml: string;
  /** Reports the current scale so a parent toolbar can display it. */
  onScaleChange?: (scale: number) => void;
  /** Accessible label for the focusable zoom surface. */
  ariaLabel?: string;
  /** Keyboard pan step in CSS pixels. */
  panStep?: number;
}

/**
 * MermaidZoomSurface renders pre-rendered Mermaid SVG inside a zoom/pan surface.
 * It owns the transform state and translates raw wheel/pointer/pinch/keyboard
 * input into transform updates via the pure helpers in zoomTransform. Imperative
 * zoom/fit/reset controls are exposed through the forwarded ref so the viewer's
 * toolbar can drive it without lifting transform state.
 */
export const MermaidZoomSurface = forwardRef<MermaidZoomSurfaceHandle, MermaidZoomSurfaceProps>(
  function MermaidZoomSurface({ svgHtml, onScaleChange, ariaLabel, panStep = 48 }, ref) {
    const surfaceRef = useRef<HTMLDivElement | null>(null);
    const contentRef = useRef<HTMLDivElement | null>(null);
    const [transform, setTransform] = useState<Transform>(() => resetTransform());
    const pointersRef = useRef<Map<number, { x: number; y: number }>>(new Map());
    const pinchDistRef = useRef<number | null>(null);
    const transformRef = useRef<Transform>(transform);
    transformRef.current = transform;

    useEffect(() => {
      onScaleChange?.(transform.scale);
    }, [transform.scale, onScaleChange]);

    const measureViewport = useCallback((): Size => {
      const el = surfaceRef.current;
      if (!el) return { width: 0, height: 0 };
      const rect = el.getBoundingClientRect();
      return { width: rect.width, height: rect.height };
    }, []);

    const measureContent = useCallback((): Size => {
      const svg = contentRef.current?.querySelector("svg");
      if (!svg) return { width: 0, height: 0 };
      const fromViewBox = parseViewBox(svg.getAttribute("viewBox"));
      if (fromViewBox) return fromViewBox;
      const attrW = Number(svg.getAttribute("width"));
      const attrH = Number(svg.getAttribute("height"));
      if (Number.isFinite(attrW) && Number.isFinite(attrH) && attrW > 0 && attrH > 0) {
        return { width: attrW, height: attrH };
      }
      const rect = svg.getBoundingClientRect();
      const scale = transformRef.current.scale || 1;
      if (rect.width > 0 && rect.height > 0) {
        return { width: rect.width / scale, height: rect.height / scale };
      }
      return { width: 0, height: 0 };
    }, []);

    // Mermaid emits SVGs with a `max-width` style and no explicit pixel size, so
    // their laid-out box doesn't match the viewBox — which breaks fit centering.
    // Pin the SVG to its intrinsic viewBox dimensions so the wrapper box equals
    // the content we scale against.
    const normalizeSvg = useCallback((): void => {
      const svg = contentRef.current?.querySelector("svg");
      if (!svg) return;
      const size = parseViewBox(svg.getAttribute("viewBox"));
      svg.style.display = "block";
      svg.style.maxWidth = "none";
      if (size) {
        svg.style.width = `${size.width}px`;
        svg.style.height = `${size.height}px`;
      }
    }, []);

    const fit = useCallback(() => {
      setTransform(fitTransform(measureViewport(), measureContent()));
    }, [measureContent, measureViewport]);

    const reset = useCallback(() => {
      setTransform(resetTransform());
    }, []);

    const zoomAtCenter = useCallback((factor: number) => {
      const viewport = measureViewport();
      setTransform((t) => zoomAroundPoint(t, factor, viewport.width / 2, viewport.height / 2));
    }, [measureViewport]);

    useImperativeHandle(
      ref,
      () => ({
        zoomIn: () => zoomAtCenter(ZOOM_STEP),
        zoomOut: () => zoomAtCenter(1 / ZOOM_STEP),
        fit,
        reset,
      }),
      [zoomAtCenter, fit, reset],
    );

    // Normalize and fit to the diagram each time new SVG is injected. Fitting in
    // the layout effect (before paint) avoids a flash of the un-fitted diagram; a
    // deferred re-fit covers the case where the surface hadn't been laid out yet.
    useLayoutEffect(() => {
      if (!svgHtml) return;
      normalizeSvg();
      fit();
      const raf = requestAnimationFrame(() => {
        if (measureViewport().width > 0) fit();
      });
      return () => cancelAnimationFrame(raf);
    }, [svgHtml, fit, measureViewport, normalizeSvg]);

    // Wheel/trackpad zoom around the pointer. Registered non-passively so we can
    // prevent the underlying messages list from scrolling.
    useEffect(() => {
      const el = surfaceRef.current;
      if (!el) return;
      const onWheel = (event: WheelEvent) => {
        event.preventDefault();
        const rect = el.getBoundingClientRect();
        const px = event.clientX - rect.left;
        const py = event.clientY - rect.top;
        const factor = Math.exp(-event.deltaY * 0.0015);
        setTransform((t) => zoomAroundPoint(t, factor, px, py));
      };
      el.addEventListener("wheel", onWheel, { passive: false });
      return () => el.removeEventListener("wheel", onWheel);
    }, []);

    const relativePoint = useCallback((clientX: number, clientY: number) => {
      const rect = surfaceRef.current?.getBoundingClientRect();
      if (!rect) return { x: clientX, y: clientY };
      return { x: clientX - rect.left, y: clientY - rect.top };
    }, []);

    const onPointerDown = useCallback((event: React.PointerEvent<HTMLDivElement>) => {
      surfaceRef.current?.setPointerCapture(event.pointerId);
      pointersRef.current.set(event.pointerId, { x: event.clientX, y: event.clientY });
      if (pointersRef.current.size === 2) {
        const [a, b] = [...pointersRef.current.values()];
        if (a && b) pinchDistRef.current = distance(a.x, a.y, b.x, b.y);
      }
    }, []);

    const onPointerMove = useCallback((event: React.PointerEvent<HTMLDivElement>) => {
      const pointers = pointersRef.current;
      const prev = pointers.get(event.pointerId);
      if (!prev) return;
      pointers.set(event.pointerId, { x: event.clientX, y: event.clientY });

      if (pointers.size >= 2) {
        const [a, b] = [...pointers.values()];
        if (!a || !b) return;
        const newDist = distance(a.x, a.y, b.x, b.y);
        const prevDist = pinchDistRef.current;
        pinchDistRef.current = newDist;
        if (prevDist && prevDist > 0) {
          const mid = midpoint(a.x, a.y, b.x, b.y);
          const rel = relativePoint(mid.x, mid.y);
          const factor = newDist / prevDist;
          setTransform((t) => zoomAroundPoint(t, factor, rel.x, rel.y));
        }
        return;
      }

      const dx = event.clientX - prev.x;
      const dy = event.clientY - prev.y;
      setTransform((t) => panBy(t, dx, dy));
    }, [relativePoint]);

    const endPointer = useCallback((event: React.PointerEvent<HTMLDivElement>) => {
      pointersRef.current.delete(event.pointerId);
      if (pointersRef.current.size < 2) pinchDistRef.current = null;
      if (surfaceRef.current?.hasPointerCapture(event.pointerId)) {
        surfaceRef.current.releasePointerCapture(event.pointerId);
      }
    }, []);

    const onKeyDown = useCallback((event: React.KeyboardEvent<HTMLDivElement>) => {
      switch (event.key) {
        case "+":
        case "=":
          event.preventDefault();
          zoomAtCenter(ZOOM_STEP);
          break;
        case "-":
        case "_":
          event.preventDefault();
          zoomAtCenter(1 / ZOOM_STEP);
          break;
        case "0":
          event.preventDefault();
          reset();
          break;
        case "f":
        case "F":
          event.preventDefault();
          fit();
          break;
        case "ArrowUp":
          event.preventDefault();
          setTransform((t) => panBy(t, 0, panStep));
          break;
        case "ArrowDown":
          event.preventDefault();
          setTransform((t) => panBy(t, 0, -panStep));
          break;
        case "ArrowLeft":
          event.preventDefault();
          setTransform((t) => panBy(t, panStep, 0));
          break;
        case "ArrowRight":
          event.preventDefault();
          setTransform((t) => panBy(t, -panStep, 0));
          break;
        default:
          break;
      }
    }, [fit, panStep, reset, zoomAtCenter]);

    const isPanning = pointersRef.current.size > 0;

    return (
      <div
        ref={surfaceRef}
        data-testid="mermaid-zoom-surface"
        role="img"
        aria-label={ariaLabel}
        tabIndex={0}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={endPointer}
        onPointerCancel={endPointer}
        onKeyDown={onKeyDown}
        className={`relative h-full w-full touch-none select-none overflow-hidden bg-wc-surface outline-none focus-visible:ring-1 focus-visible:ring-wc-accent/40 ${
          isPanning ? "cursor-grabbing" : "cursor-grab"
        }`}
      >
        <div
          ref={contentRef}
          // A plain 2D transform (no translate3d / will-change) keeps the SVG off
          // a cached GPU raster layer, so vector text re-renders crisply at every
          // zoom level instead of being scaled up blurry.
          className="absolute left-0 top-0 origin-top-left"
          style={{ transform: `translate(${transform.x}px, ${transform.y}px) scale(${transform.scale})` }}
          dangerouslySetInnerHTML={{ __html: svgHtml }}
        />
      </div>
    );
  },
);
