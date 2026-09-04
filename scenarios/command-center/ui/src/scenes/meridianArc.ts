import { clipOutsideQuiet, drawGlow, focalPoint, rgba, type Scene } from "./engine";

/** Geographic traffic: arc brightness and reach follow the panel's shares. */
export function meridianArc(): Scene {
  return { init() {}, draw(frame) {
    const { ctx, w, h, palette, data, t } = frame; const focus = data.readings[data.focus ?? ""];
    const shares = focus?.rows?.map((row) => Math.max(0, Math.min(1, row.share))) ?? [1]; const origin = focalPoint(frame);
    clipOutsideQuiet(frame); ctx.lineCap = "round";
    shares.slice(0, frame.tier === "full" ? 12 : 6).forEach((share, index) => {
      const endX = w * (0.12 + ((index * 0.37) % 0.76)); const endY = h * (0.18 + ((index * 0.23) % 0.64));
      const bend = (endX - origin.x) * 0.35; ctx.strokeStyle = rgba(ctx, palette.primary, 0.18 + share * 0.72); ctx.lineWidth = 1 + share * 5;
      ctx.beginPath(); ctx.moveTo(origin.x, origin.y); ctx.quadraticCurveTo(origin.x + bend, Math.min(origin.y, endY) - h * (0.12 + share * 0.18) + Math.sin(t + index) * 3, endX, endY); ctx.stroke();
      drawGlow(frame, endX, endY, 3 + share * 10, palette.accent, 0.3 + share * 0.5);
    }); ctx.restore();
  } };
}
