import { useEffect, useRef, useState } from "react";
import type { Reading } from "../lib/api";
import { createScene } from "../scenes";
import { frameIsBlank, probeTier, type ProbeTier } from "../lib/sceneProbe";
import { mulberry32, sceneData, seedFrom, type Frame, type Palette, type Rect, type SceneTier } from "../scenes/engine";


interface AmbientCanvasProps {
  composition: string;
  readings: Reading[];
  forcedTier: string | null;
  /** Elements the scene must keep bright bodies out of: the hero and the supporting readings. */
  quietRefs: Array<React.RefObject<HTMLElement>>;
  /** Seed so adjacent displays never run in sync. */
  seed: string;
}

const readPalette = (element: HTMLElement): Palette => {
  const style = getComputedStyle(element);
  const token = (name: string, fallback: string) => style.getPropertyValue(name).trim() || fallback;
  return {
    primary: token("--color-primary", "#33d6ff"),
    accent: token("--color-accent", "#9be9ff"),
    foreground: token("--color-foreground", "#eaf3ff"),
    glow: token("--color-glow", "rgba(51,214,255,.5)"),
    gap: token("--color-gap", "#b7a6ff"),
    warning: token("--color-warning", "#f5b544"),
    background: token("--color-background", "#04060d"),
  };
};

/**
 * The scene layer. Draws the room's composition into a transparent canvas so
 * the theme ground paints beneath it and the figure layer composites above.
 * Still tier draws one composed frame; every tier checks its first frame.
 */
export function AmbientCanvas({ composition, readings, forcedTier, quietRefs, seed }: AmbientCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [state, setState] = useState<"pending" | "ready" | "fallback">("pending");
  const [tier, setTier] = useState<ProbeTier>("still");
  const readingsRef = useRef(readings);
  readingsRef.current = readings;

  useEffect(() => {
    const canvas = canvasRef.current;
    const context = canvas?.getContext("2d");
    if (!canvas || !context) {
      setState("fallback");
      return;
    }
    const probed = probeTier(forcedTier);
    setTier(probed);
    const drawTier: SceneTier = probed === "still" ? "reduced" : probed;
    const scene = createScene(composition);
    const rng = mulberry32(seedFrom(`${composition}:${seed}`));
    const palette = readPalette(canvas);
    const data = sceneData(readingsRef.current);
    const ratio = Math.min(2, Math.max(1, window.devicePixelRatio || 1));
    let width = 0;
    let height = 0;
    let started = 0;
    let last = 0;
    let raf = 0;
    let initialised = false;
    let checked = false;

    const quietRects = (): Rect[] => {
      const own = canvas.getBoundingClientRect();
      return quietRefs.flatMap((ref) => {
        const element = ref.current;
        if (!element) return [];
        const box = element.getBoundingClientRect();
        return box.width > 0 && box.height > 0 ? [{ x: box.left - own.left, y: box.top - own.top, w: box.width, h: box.height }] : [];
      });
    };

    const resize = () => {
      const box = canvas.getBoundingClientRect();
      width = Math.max(1, Math.floor(box.width));
      height = Math.max(1, Math.floor(box.height));
      canvas.width = Math.floor(width * ratio);
      canvas.height = Math.floor(height * ratio);
      context.setTransform(ratio, 0, 0, ratio, 0, 0);
      initialised = false;
    };

    const frame = (nowMs: number): Frame => {
      const t = started ? (nowMs - started) / 1000 : 0;
      const dt = last ? Math.min(0.1, (nowMs - last) / 1000) : 1 / 60;
      return { ctx: context, w: width, h: height, t, dt, quiet: quietRects(), tier: drawTier, palette, data, rng };
    };

    const paint = (nowMs: number, still: boolean) => {
      if (!started) started = nowMs;
      const f = frame(still ? started + 14_000 : nowMs);
      if (!initialised) {
        scene.init(f);
        initialised = true;
      }
      context.clearRect(0, 0, width, height);
      scene.draw(f);
      last = nowMs;
      if (!checked) {
        checked = true;
        if (frameIsBlank(context, canvas.width, canvas.height)) {
          setState("fallback");
        } else {
          setState("ready");
        }
      }
    };

    const loop = (nowMs: number) => {
      if (!document.hidden) paint(nowMs, false);
      raf = window.requestAnimationFrame(loop);
    };

    resize();
    const observer = new ResizeObserver(() => {
      resize();
      if (probed === "still") paint(performance.now(), true);
    });
    observer.observe(canvas);
    if (probed === "still") {
      paint(performance.now(), true);
    } else {
      raf = window.requestAnimationFrame(loop);
    }
    return () => {
      observer.disconnect();
      window.cancelAnimationFrame(raf);
    };
  }, [composition, forcedTier, quietRefs, seed]);

  return (
    <div className={`cc-scene cc-scene-${state}`} data-testid="scene-canvas" data-scene-state={state} data-scene-tier={tier} data-composition={composition} aria-hidden="true">
      <canvas ref={canvasRef} />
      {state === "fallback" ? <div className="cc-scene-still" data-testid="scene-still" /> : null}
    </div>
  );
}
