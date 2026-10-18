import { useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";
import { boundImage, fitImage, rotatedDimensions, zoomImage, type ImageDimensions, type ImageView } from "./imageGeometry";

interface Point { x: number; y: number }
interface Gesture { view: ImageView; center: Point; distance: number }

export function useImageInspection(stage: ImageDimensions, image: ImageDimensions, rotation: number) {
  const [manualView, setManualView] = useState<ImageView | null>(null);
  const dimensions = rotatedDimensions(image, rotation);
  const fit = fitImage(dimensions, stage);
  const view = boundImage(manualView ?? { scale: fit, x: 0, y: 0 }, dimensions, stage);
  const current = useRef(view);
  current.current = view;
  const points = useRef(new Map<number, Point>());
  const gesture = useRef<Gesture | null>(null);
  const stageElement = useRef<HTMLDivElement | null>(null);
  const origin = useRef<Point>({ x: 0, y: 0 });
  const pending = useRef<ImageView | null>(null);
  const frame = useRef(0);

  const paint = (next: ImageView) => {
    current.current = next;
    pending.current = next;
    if (!frame.current) frame.current = requestAnimationFrame(() => {
      frame.current = 0;
      if (pending.current) setManualView(pending.current);
    });
  };
  const reset = () => {
    cancelAnimationFrame(frame.current);
    frame.current = 0;
    pending.current = null;
    points.current.clear();
    gesture.current = null;
    setManualView(null);
  };
  useEffect(() => {
    // A resized stage invalidates active pointer coordinates, never image data.
    points.current.clear();
    gesture.current = null;
    return () => { cancelAnimationFrame(frame.current); frame.current = 0; pending.current = null; };
  }, [stage.width, stage.height, rotation]);

  const baseline = () => {
    const [a, b] = Array.from(points.current.values());
    if (!a) { gesture.current = null; return; }
    gesture.current = {
      view: current.current,
      center: b ? { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 } : a,
      distance: b ? Math.hypot(b.x - a.x, b.y - a.y) : 0,
    };
  };
  const down = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (event.pointerType === "mouse" && event.button !== 0) return;
    event.stopPropagation();
    const rect = event.currentTarget.getBoundingClientRect();
    origin.current = { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
    points.current.set(event.pointerId, { x: event.clientX - origin.current.x, y: event.clientY - origin.current.y });
    event.currentTarget.setPointerCapture(event.pointerId);
    baseline();
  };
  const move = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (!points.current.has(event.pointerId)) return;
    points.current.set(event.pointerId, { x: event.clientX - origin.current.x, y: event.clientY - origin.current.y });
    const start = gesture.current;
    const [a, b] = Array.from(points.current.values());
    if (!start || !a) return;
    const center = b ? { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 } : a;
    const scale = b && start.distance > 0 ? Math.max(0.01, Math.min(8, start.view.scale * Math.hypot(b.x - a.x, b.y - a.y) / start.distance)) : start.view.scale;
    const ratio = scale / start.view.scale;
    paint(boundImage({ scale, x: center.x - (start.center.x - start.view.x) * ratio, y: center.y - (start.center.y - start.view.y) * ratio }, dimensions, stage));
  };
  const up = (event: ReactPointerEvent<HTMLDivElement>) => {
    points.current.delete(event.pointerId);
    baseline();
  };
  useEffect(() => {
    const element = stageElement.current;
    if (!element) return;
    const wheel = (event: WheelEvent) => {
      if (!image.width) return;
      event.preventDefault();
      const rect = element.getBoundingClientRect();
      const previous = current.current;
      const factor = event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? stage.height : 1;
      const next = zoomImage(previous, previous.scale * Math.exp(-event.deltaY * factor * 0.002), { x: event.clientX - rect.left - rect.width / 2, y: event.clientY - rect.top - rect.height / 2 }, rotatedDimensions(image, rotation), stage);
      current.current = next;
      setManualView(next);
    };
    element.addEventListener("wheel", wheel, { passive: false });
    return () => { element.removeEventListener("wheel", wheel); };
  }, [image, rotation, stage]);

  const zoom = (scale: number) => {
    const next = zoomImage(current.current, scale, { x: 0, y: 0 }, dimensions, stage);
    current.current = next;
    setManualView(next);
  };
  const pan = (x: number, y: number) => {
    const next = boundImage({ ...current.current, x: current.current.x + x, y: current.current.y + y }, dimensions, stage);
    current.current = next;
    setManualView(next);
  };
  return { stageElement, view, fitMode: manualView === null, fit, zoom, reset, pan, pointerEvents: { onPointerDown: down, onPointerMove: move, onPointerUp: up, onPointerCancel: up, onLostPointerCapture: up } };
}
