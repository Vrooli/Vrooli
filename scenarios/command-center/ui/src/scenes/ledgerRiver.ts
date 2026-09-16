import { clipOutsideQuiet, drawGlow, freeBand, rgba, type Frame, type Scene } from "./engine";

export interface LedgerLayout {
  left: number;
  right: number;
  baseline: number;
  burnY: number;
  revenueY: number;
  crossing: boolean;
}

/** Pure geometry for the waterline: all magnitudes come from the posture reading. */
export function ledgerLayout(frame: Pick<Frame, "w" | "h" | "quiet" | "data">): LedgerLayout {
  const band = freeBand(frame.h, frame.quiet.filter((rect) => rect.w >= frame.w * 0.7));
  const width = Math.min(frame.w * 0.72, band.size * 1.25);
  const left = frame.w * 0.5 - width / 2;
  const baseline = band.bottom - band.size * 0.12;
  const posture = frame.data.readings.offer_posture?.meta ?? {};
  const burn = Number(posture.burnMinor ?? 0);
  const revenue = Number(posture.revenueMinor ?? 0);
  const max = Math.max(1, burn, revenue);
  const scale = band.size * 0.62;
  const burnY = baseline - (burn / max) * scale;
  const revenueY = baseline - (revenue / max) * scale;
  return { left, right: left + width, baseline, burnY, revenueY, crossing: revenue >= burn && burn > 0 };
}

/** Ledger waterline: burn is the fixed reference; revenue rises toward it. */
export function ledgerRiver(): Scene {
  let bloomed = false;
  return {
    init() { bloomed = false; },
    draw(frame) {
      const { ctx, h, palette, data } = frame;
      const layout = ledgerLayout(frame);
      const measured = data.readings.offer_posture?.ink === "solid" || data.readings.offer_posture?.ink === "dimmed";
      clipOutsideQuiet(frame);
      ctx.strokeStyle = rgba(ctx, palette.primary, 0.12);
      ctx.lineWidth = 1;
      for (let y = h * 0.16; y < h; y += h * 0.08) {
        ctx.beginPath(); ctx.moveTo(layout.left, y); ctx.lineTo(layout.right, y); ctx.stroke();
      }
      ctx.restore();
      if (!measured) return;
      const drawLine = (y: number, color: string, alpha: number) => {
        ctx.strokeStyle = rgba(ctx, color, alpha); ctx.setLineDash([]); ctx.lineWidth = 2;
        ctx.beginPath(); ctx.moveTo(layout.left, y); ctx.lineTo(layout.right, y); ctx.stroke();
      };
      drawLine(layout.burnY, palette.warning, 0.78);
      drawLine(layout.revenueY, palette.accent, 0.85);
      ctx.font = "11px ui-monospace, SFMono-Regular, monospace";
      ctx.fillStyle = rgba(ctx, palette.warning, 0.8); ctx.fillText("BURN", layout.left, layout.burnY - 8);
      ctx.fillStyle = rgba(ctx, palette.accent, 0.9); ctx.fillText("REVENUE", layout.left, layout.revenueY - 8);
      if (layout.crossing && !bloomed) {
        bloomed = true;
        drawGlow(frame, (layout.left + layout.right) / 2, layout.burnY, 28, palette.accent, 0.9);
      }
    },
  };
}
