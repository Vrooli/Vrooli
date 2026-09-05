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
  focus?: string;
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
export function AmbientCanvas({ composition, readings, forcedTier, quietRefs, seed, focus }: AmbientCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [state, setState] = useState<"pending" | "ready" | "fallback">("pending");
  const [tier, setTier] = useState<ProbeTier>("still");
  const readingsRef = useRef(readings);
  readingsRef.current = readings;
  const compositionRef = useRef(composition);
  const seedRef = useRef(seed);
  const focusRef = useRef(focus);
  compositionRef.current = composition;
  seedRef.current = seed;
  focusRef.current = focus;
  const repaintRef = useRef<(() => void) | null>(null);

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
    let activeScene = createScene(compositionRef.current);
    let activeComposition = compositionRef.current;
    let activeSeed = seedRef.current;
    let incomingScene: ReturnType<typeof createScene> | null = null;
    let incomingComposition = "";
    let incomingSeed = "";
    let transitionStarted = 0;
    // Beat content changes in one coordinated visual beat; a long scene-only
    // fade leaves the previous composition visibly hanging behind the new one.
    const transitionDuration = Math.min(readMotionDuration(canvas), 420);
    const ratio = Math.min(2, Math.max(1, window.devicePixelRatio || 1));
    let width = 0;
    let height = 0;
    let started = 0;
    let last = 0;
    let raf = 0;
    let activeInitialised = false;
    let incomingInitialised = false;
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
      activeInitialised = false;
      incomingInitialised = false;
    };

    const frame = (nowMs: number, rng: () => number): Frame => {
      const t = started ? (nowMs - started) / 1000 : 0;
      const dt = last ? Math.min(0.1, (nowMs - last) / 1000) : 1 / 60;
      return { ctx: context, w: width, h: height, t, dt, quiet: quietRects(), tier: drawTier, palette: readPalette(canvas), data: sceneData(readingsRef.current, focusRef.current), rng };
    };

    const paint = (nowMs: number, still: boolean) => {
      if (!started) started = nowMs;
      const desiredComposition = compositionRef.current;
      if (desiredComposition !== activeComposition && !incomingScene) {
        incomingComposition = desiredComposition;
        incomingSeed = seedRef.current;
        incomingScene = createScene(incomingComposition);
        incomingInitialised = false;
        transitionStarted = nowMs;
        if (probed === "still" || window.matchMedia?.("(prefers-reduced-motion: reduce)").matches) {
          activeScene = incomingScene;
          activeComposition = incomingComposition;
          activeSeed = incomingSeed;
          activeInitialised = false;
          incomingScene = null;
        }
      }
      const drawAt = still ? started + 14_000 : nowMs;
      context.clearRect(0, 0, width, height);
      const activeRng = mulberry32(seedFrom(`${activeComposition}:${activeSeed}`));
      const activeFrame = frame(drawAt, activeRng);
      if (!activeInitialised) { activeScene.init(activeFrame); activeInitialised = true; }
      const blending = incomingScene !== null;
      const progress = blending ? Math.min(1, Math.max(0, (nowMs - transitionStarted) / transitionDuration)) : 1;
      context.save(); context.globalAlpha = blending ? 1 - progress : 1; activeScene.draw(activeFrame); context.restore();
      if (incomingScene) {
        const incomingRng = mulberry32(seedFrom(`${incomingComposition}:${incomingSeed}`));
        const incomingFrame = frame(drawAt, incomingRng);
        if (!incomingInitialised) { incomingScene.init(incomingFrame); incomingInitialised = true; }
        context.save(); context.globalAlpha = progress; incomingScene.draw(incomingFrame); context.restore();
        if (progress >= 1) {
          activeScene = incomingScene; activeComposition = incomingComposition; activeSeed = incomingSeed;
          activeInitialised = true; incomingScene = null; incomingInitialised = false;
        }
      }
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

    repaintRef.current = () => paint(performance.now(), probed === "still");

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
      repaintRef.current = null;
    };
  }, [forcedTier, quietRefs]);

  useEffect(() => { repaintRef.current?.(); }, [composition, seed, focus]);

  return (
    <div className={`cc-scene cc-scene-${state}`} data-testid="scene-canvas" data-scene-state={state} data-scene-tier={tier} data-composition={composition} aria-hidden="true">
      <canvas ref={canvasRef} />
      {state === "fallback" ? <div className="cc-scene-still" data-testid="scene-still" /> : null}
    </div>
  );
}

function readMotionDuration(element: HTMLElement): number {
  const value = getComputedStyle(element).getPropertyValue("--motion-room").trim();
  const milliseconds = value.endsWith("ms") ? Number.parseFloat(value) : value.endsWith("s") ? Number.parseFloat(value) * 1000 : 900;
  return Number.isFinite(milliseconds) && milliseconds > 0 ? milliseconds : 900;
}
