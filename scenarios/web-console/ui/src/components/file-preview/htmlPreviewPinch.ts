import runtimeSource from "./htmlPreviewPinchRuntime.js?raw";

export function htmlPreviewPinchScript(): string {
  return `<script type="module" data-wc-html-pinch>${runtimeSource}\ninstallHtmlPreviewPinch(window);</script>`;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
function number(value: unknown): number | null {
  return typeof value === "number" && Number.isFinite(value) && Math.abs(value) < 1e8 ? value : null;
}

/** One compositor transform per frame; no document measurements in the gesture path. */
export function attachHtmlPreviewInspection(
  frame: HTMLIFrameElement,
  surface: HTMLElement,
  stage: HTMLElement,
  baseScale: number,
  width: number,
  height: number,
): () => void {
  let scale = 1;
  let x = 0;
  let y = 0;
  let raf: number | null = null;
  let scrollX = 0;
  let scrollY = 0;
  let panEnabled = false;
  let pinch: { distance: number; scale: number; screenX: number; screenY: number; pointX: number; pointY: number; anchorX: number; anchorY: number } | null = null;
  let pan: { x: number; y: number } | null = null;
  const post = (message: { pan?: boolean; scrollX?: number; scrollY?: number }) => {
    frame.contentWindow?.postMessage({ type: "wc-html-inspection", ...message }, "*");
  };
  const clamp = () => {
    x = Math.min(0, Math.max(width * (1 - scale), x));
    y = Math.min(0, Math.max(height * (1 - scale), y));
  };
  const paint = () => {
    raf = null;
    frame.style.transform = scale === 1 ? `scale(${String(baseScale)})` : `translate3d(${String(x)}px, ${String(y)}px, 0) scale(${String(baseScale * scale)})`;
    surface.dataset.wcHtmlInspectionScale = String(scale);
    const nextPan = scale > 1;
    if (nextPan !== panEnabled) { panEnabled = nextPan; post({ pan: panEnabled }); }
    // Once the magnified viewport reaches an edge, continue through the actual
    // document. Native frame scrolling stays entirely out of the pinch path.
    if (scrollX || scrollY) {
      const previous = stage.scrollLeft;
      stage.scrollLeft += scrollX;
      post({ scrollX: (scrollX - (stage.scrollLeft - previous)) / (baseScale * scale), scrollY: scrollY / (baseScale * scale) });
      scrollX = 0;
      scrollY = 0;
    }
  };
  const schedule = () => { raf ??= requestAnimationFrame(paint); };
  const zoom = (requested: number, pointX: number, pointY: number, anchorX: number, anchorY: number) => {
    scale = Math.min(5, Math.max(1, requested));
    if (scale < 1.01) scale = 1;
    x = pointX - anchorX * scale;
    y = pointY - anchorY * scale;
    clamp();
    schedule();
  };
  const onMessage = (event: MessageEvent<unknown>) => {
    if (event.source !== frame.contentWindow || !isRecord(event.data) || event.data.type !== "wc-html-gesture") return;
    const data = event.data;
    if (data.kind === "ready") { post({ pan: panEnabled }); return; }
    if (data.kind === "end" || data.kind === "cancel") { pinch = null; pan = null; return; }
    const sx = number(data.x);
    const sy = number(data.y);
    if (data.kind === "wheel") {
      const localX = number(data.localX);
      const localY = number(data.localY);
      const delta = number(data.delta);
      if (localX === null || localY === null || delta === null) return;
      const pointX = x + localX * baseScale * scale;
      const pointY = y + localY * baseScale * scale;
      zoom(scale * Math.exp(-Math.max(-500, Math.min(500, delta)) * 0.01), pointX, pointY, localX * baseScale, localY * baseScale);
      return;
    }
    if (sx === null || sy === null) return;
    if (data.kind === "pinch-start") {
      const distance = number(data.distance);
      const localX = number(data.localX);
      const localY = number(data.localY);
      if (distance === null || distance <= 0 || localX === null || localY === null) return;
      pinch = { distance, scale, screenX: sx, screenY: sy, pointX: x + localX * baseScale * scale, pointY: y + localY * baseScale * scale, anchorX: localX * baseScale, anchorY: localY * baseScale };
      pan = null;
    } else if (data.kind === "pinch-move" && pinch) {
      const distance = number(data.distance);
      if (distance === null || distance <= 0) return;
      zoom(pinch.scale * distance / pinch.distance, pinch.pointX + sx - pinch.screenX, pinch.pointY + sy - pinch.screenY, pinch.anchorX, pinch.anchorY);
    } else if (data.kind === "pan-start") {
      pan = { x: sx, y: sy };
      pinch = null;
    } else if (data.kind === "pan-move" && pan && scale > 1) {
      const dx = sx - pan.x;
      const dy = sy - pan.y;
      const oldX = x;
      const oldY = y;
      x += dx;
      y += dy;
      clamp();
      scrollX -= dx - (x - oldX);
      scrollY -= dy - (y - oldY);
      pan = { x: sx, y: sy };
      schedule();
    }
  };
  window.addEventListener("message", onMessage);
  frame.style.transform = `scale(${String(baseScale)})`;
  surface.dataset.wcHtmlInspectionScale = "1";
  return () => {
    window.removeEventListener("message", onMessage);
    if (raf !== null) cancelAnimationFrame(raf);
    post({ pan: false });
    delete surface.dataset.wcHtmlInspectionScale;
  };
}
