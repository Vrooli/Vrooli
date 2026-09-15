import { clipOutsideQuiet, drawGlow, focalPoint, rgba, type Scene } from "./engine";

export function conversionFunnel(): Scene {
  return { init() {}, draw(frame) {
    const focus = frame.data.readings[frame.data.focus ?? ""];
    const rows = focus?.rows ?? [];
    if (!rows.length || rows.some((row) => row.value < 0)) return;
    const max = Math.max(...rows.map((row) => row.value), 1);
    const center = focalPoint(frame);
    const gap = Math.min(18, frame.h / Math.max(rows.length * 5, 1));
    const height = Math.min(64, frame.h / Math.max(rows.length * 1.6, 1));
    clipOutsideQuiet(frame);
    rows.forEach((row, index) => {
      const width = Math.max(2, frame.w * 0.42 * row.value / max);
      const y = center.y - (rows.length * (height + gap)) / 2 + index * (height + gap);
      frame.ctx.fillStyle = rgba(frame.ctx, frame.palette.primary, 0.18);
      frame.ctx.strokeStyle = rgba(frame.ctx, frame.palette.primary, 0.75);
      frame.ctx.beginPath(); frame.ctx.roundRect(center.x - width / 2, y, width, height, 8); frame.ctx.fill(); frame.ctx.stroke();
      drawGlow(frame, center.x, y + height / 2, Math.min(width / 3, 18), frame.palette.accent, 0.16);
    });
    frame.ctx.restore();
  }};
}
