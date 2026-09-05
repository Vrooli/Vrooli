import { clipOutsideQuiet, drawGlow, focalPoint, rgba, type Scene } from "./engine";

/** Release ladder: explicit ascending rungs make the delivery stages legible. */
export function funnelCascade(): Scene { return { init() {}, draw(frame) {
  const { ctx, w, h, palette, data } = frame; const focus = data.readings[data.focus ?? ""];
  const rows = focus?.rows?.length ? focus.rows : [{ value: 1, share: 1 }, { value: 0.7, share: 0.7 }, { value: 0.4, share: 0.4 }];
  const center = focalPoint(frame); const rungCount = Math.min(5, rows.length); const rungGap = Math.min(72, h * 0.13); const startY = center.y - ((rungCount - 1) * rungGap) / 2; const left = Math.max(w * 0.38, center.x - w * 0.28);
  clipOutsideQuiet(frame);
  ctx.strokeStyle = rgba(ctx, palette.accent, 0.48); ctx.lineWidth = 1.2;
  ctx.beginPath(); ctx.moveTo(left, startY + 12); ctx.lineTo(left, startY + (rungCount - 1) * rungGap + 12); ctx.stroke();
  rows.slice(0, rungCount).forEach((row, index) => {
    const progress = Math.max(0.18, Math.min(1, row.share || row.value)); const y = startY + index * rungGap; const length = w * (0.2 + progress * 0.34); const x = left + index * Math.min(20, w * 0.018);
    ctx.strokeStyle = rgba(ctx, palette.primary, 0.38 + progress * 0.45); ctx.lineWidth = 2;
    ctx.beginPath(); ctx.moveTo(x, y); ctx.lineTo(x + length, y); ctx.stroke();
    drawGlow(frame, x + length, y, 5 + progress * 5, palette.accent, 0.28 + progress * 0.4);
    ctx.fillStyle = palette.accent; ctx.beginPath(); ctx.arc(x, y, 2.5, 0, Math.PI * 2); ctx.fill();
  });
  ctx.restore();
} }; }
