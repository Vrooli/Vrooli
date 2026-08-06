import { interpolatePath, type NormalizedIconGeometry } from "./geometry";

export function animatePathMorph(
  element: SVGPathElement | null,
  from: NormalizedIconGeometry,
  to: NormalizedIconGeometry,
  durationMs: number,
  reducedMotion: boolean,
  onComplete: () => void,
) {
  if (!element) return () => {};
  let frame: number | undefined;
  let stopped = false;
  const finish = () => {
    if (stopped) return;
    element.setAttribute("d", to.d);
    onComplete();
  };

  if (reducedMotion || durationMs <= 0 || typeof window === "undefined") {
    finish();
    return () => {
      stopped = true;
    };
  }

  const start =
    typeof performance === "undefined" ? Date.now() : performance.now();
  const tick = (now: number) => {
    if (stopped) return;
    const elapsed = now - start;
    const raw = Math.min(1, elapsed / durationMs);
    const eased = 1 - Math.pow(1 - raw, 3);
    element.setAttribute("d", interpolatePath(from, to, eased));
    if (raw >= 1) {
      finish();
      return;
    }
    frame = window.requestAnimationFrame(tick);
  };

  element.setAttribute("d", from.d);
  frame = window.requestAnimationFrame(tick);
  return () => {
    stopped = true;
    if (frame !== undefined) window.cancelAnimationFrame(frame);
  };
}
