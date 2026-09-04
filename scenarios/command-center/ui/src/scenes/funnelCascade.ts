import { clipOutsideQuiet, focalPoint, rgba, type Scene } from "./engine";

/** Conversion path: each band width is proportional to its observed value. */
export function funnelCascade(): Scene { return { init() {}, draw(frame) {
  const { ctx, w, h, palette, data } = frame; const focus = data.readings[data.focus ?? ""];
  const values = focus?.rows?.map((row) => Math.max(0, row.value)) ?? [1, 0.4, 0.1]; const max = Math.max(1, ...values); const center = focalPoint(frame); const height = Math.min(h * 0.62, 420); const band = height / values.length;
  clipOutsideQuiet(frame); values.forEach((value, index) => { const width = w * 0.62 * (value / max); const y = center.y - height / 2 + index * band; ctx.fillStyle = rgba(ctx, palette.primary, 0.12 + (1 - index / values.length) * 0.2); ctx.strokeStyle = palette.accent; ctx.lineWidth = 1.5; ctx.beginPath(); ctx.moveTo(center.x - width / 2, y); ctx.lineTo(center.x + width / 2, y); ctx.lineTo(center.x + Math.max(4, width * 0.78) / 2, y + band - 3); ctx.lineTo(center.x - Math.max(4, width * 0.78) / 2, y + band - 3); ctx.closePath(); ctx.fill(); ctx.stroke(); }); ctx.restore();
} }; }
