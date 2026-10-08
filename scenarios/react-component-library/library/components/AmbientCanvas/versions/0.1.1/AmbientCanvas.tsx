/**
 * @libraryId react-component-library:AmbientCanvas
 * @displayName Ambient Canvas
 * @description Tier-aware live ambient scene canvas
 * @version 0.1.1
 * @tags ["ambient","canvas","display","war-room"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { useEffect, useRef } from "react";

export interface AmbientCanvasFrame {
  context: CanvasRenderingContext2D;
  width: number;
  height: number;
  elapsedSeconds: number;
  deltaSeconds: number;
}

export interface AmbientCanvasProps {
  draw: (frame: AmbientCanvasFrame) => void;
  className?: string;
  paused?: boolean;
  still?: boolean;
}

/** A host-agnostic canvas loop. The host supplies the scene and live data. */
export function AmbientCanvas({
  draw,
  className,
  paused = false,
  still = false,
}: AmbientCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const drawRef = useRef(draw);
  drawRef.current = draw;
  useEffect(() => {
    const canvas = canvasRef.current;
    const context = canvas?.getContext("2d");
    if (!canvas || !context) return;
    let frameID = 0;
    let last = 0;
    const started = performance.now();
    const resize = () => {
      const bounds = canvas.getBoundingClientRect();
      const ratio = Math.min(2, Math.max(1, window.devicePixelRatio || 1));
      canvas.width = Math.max(1, Math.floor(bounds.width * ratio));
      canvas.height = Math.max(1, Math.floor(bounds.height * ratio));
      context.setTransform(ratio, 0, 0, ratio, 0, 0);
    };
    const paint = (now: number) => {
      resize();
      const bounds = canvas.getBoundingClientRect();
      const deltaSeconds = last ? Math.min(0.1, (now - last) / 1000) : 1 / 60;
      drawRef.current({
        context,
        width: Math.max(1, bounds.width),
        height: Math.max(1, bounds.height),
        elapsedSeconds: (now - started) / 1000,
        deltaSeconds,
      });
      last = now;
      if (!still && !paused && !document.hidden) frameID = requestAnimationFrame(paint);
    };
    const observer = new ResizeObserver(resize);
    observer.observe(canvas);
    paint(performance.now());
    return () => {
      observer.disconnect();
      cancelAnimationFrame(frameID);
    };
  }, [paused, still]);
  return (
    <canvas
      ref={canvasRef}
      className={className}
      data-rcl-ambient-canvas
      data-paused={paused || undefined}
      data-still={still || undefined}
    />
  );
}
