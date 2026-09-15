import { clipOutsideQuiet, drawGlow, rgba, type Frame, type Scene } from "./engine";

export interface FunnelLayout {
  centerX: number;
  centerY: number;
  maxWidth: number;
  height: number;
  gap: number;
}

/** Keep the atmosphere in the open band beside the hero, with a deliberately small footprint. */
export function funnelLayout(frame: Pick<Frame, "w" | "h" | "quiet">, rowCount: number): FunnelLayout {
  const { w, h, quiet } = frame;
  const narrow = quiet.filter((rect) => rect.w < w * 0.7);
  const heroRight = narrow.length ? Math.max(...narrow.map((rect) => rect.x + rect.w)) : w * 0.54;
  const sideBand = Math.max(0, w - heroRight);
  const side = sideBand >= w * 0.22;
  const bandWidth = side ? sideBand : w * 0.32;
  const centerX = side ? heroRight + sideBand * 0.52 : w * 0.76;
  const centerY = h * 0.48;
  const height = Math.min(44, Math.max(20, h / Math.max(rowCount * 2.6, 1)));
  const gap = Math.min(10, Math.max(4, h / Math.max(rowCount * 8, 1)));
  return { centerX, centerY, maxWidth: Math.max(96, Math.min(280, bandWidth * 0.72)), height, gap };
}

export function conversionFunnel(): Scene {
  return { init() {}, draw(frame) {
    const focus = frame.data.readings[frame.data.focus ?? ""];
    const rows = focus?.rows ?? [];
    if (!rows.length || rows.some((row) => row.value < 0)) return;
    const max = Math.max(...rows.map((row) => row.value), 1);
    const { centerX, centerY, maxWidth, height, gap } = funnelLayout(frame, rows.length);
    const totalHeight = rows.length * height + (rows.length - 1) * gap;
    clipOutsideQuiet(frame);
    rows.forEach((row, index) => {
      const width = Math.max(6, maxWidth * row.value / max);
      const y = centerY - totalHeight / 2 + index * (height + gap);
      frame.ctx.fillStyle = rgba(frame.ctx, frame.palette.primary, 0.07);
      frame.ctx.strokeStyle = rgba(frame.ctx, frame.palette.primary, 0.42);
      frame.ctx.beginPath();
      frame.ctx.roundRect(centerX - width / 2, y, width, height, 6);
      frame.ctx.fill(); frame.ctx.stroke();
      drawGlow(frame, centerX, y + height / 2, Math.min(width / 4, 12), frame.palette.accent, 0.1);
    });
    frame.ctx.restore();
  }};
}
