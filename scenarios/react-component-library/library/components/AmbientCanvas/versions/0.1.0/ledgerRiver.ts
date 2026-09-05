import { clipOutsideQuiet, drawGlow, focalPoint, freeBand, inQuiet, rgba, type Scene } from "./engine";

interface Mote { column: number; y: number; speed: number; x: number }

/** Ledger: tier columns whose strokes carry the room's provenance, and a specular sweep crossing gold. */
export function ledgerRiver(): Scene {
  const tiers = [0.58, 0.2, 0.13, 0.06, 0.03];
  const top = 0.58;
  let motes: Mote[] = [];
  return {
    init(frame) {
      const { rng, tier } = frame;
      motes = Array.from({ length: tier === "full" ? 160 : 60 }, () => ({ column: Math.floor(rng() * tiers.length), y: rng(), speed: 0.04 + rng() * 0.05, x: rng() }));
    },
    draw(frame) {
      const { ctx, w, h, t, dt, palette, quiet, data } = frame;
      const allIllustrative = data.order.every((id) => data.readings[id]?.ink !== "solid" && data.readings[id]?.ink !== "dimmed");
      const focal = focalPoint(frame);
      const rule = rgba(ctx, palette.primary, 0.09);
      clipOutsideQuiet(frame);
      ctx.lineWidth = 1;
      for (let y = h * 0.12; y < h; y += h * 0.07) {
        ctx.strokeStyle = rule;
        ctx.beginPath();
        ctx.moveTo(w * 0.04, y);
        ctx.lineTo(w * 0.96, y);
        ctx.stroke();
      }
      ctx.restore();
      const band = freeBand(h, quiet.filter((rect) => rect.w >= w * 0.7));
      const bandW = Math.min(w * 0.42, band.size * 1.1);
      const colW = bandW / tiers.length;
      const left = focal.x - bandW / 2;
      const base = band.bottom - band.size * 0.08;
      const maxH = band.size * 0.7;
      const column = (i: number, share: number) => ({ x: left + i * colW + colW * 0.18, cw: colW * 0.64, ch: Math.max(h * 0.06, maxH * Math.pow(share / top, 0.55)) });
      ctx.setLineDash(allIllustrative ? [3, 6] : []);
      tiers.forEach((share, i) => {
        const { x, cw, ch } = column(i, share);
        if (inQuiet(quiet, x + cw / 2, base - ch / 2, 10)) return;
        ctx.strokeStyle = rgba(ctx, palette.primary, 0.55);
        ctx.strokeRect(x, base - ch, cw, ch);
        ctx.setLineDash([]);
        ctx.strokeStyle = rgba(ctx, palette.primary, 0.22);
        ctx.beginPath();
        ctx.moveTo(x, base + 8);
        ctx.lineTo(x + cw, base + 8);
        ctx.stroke();
        ctx.setLineDash(allIllustrative ? [3, 6] : []);
      });
      ctx.setLineDash([]);
      ctx.globalCompositeOperation = "lighter";
      for (const mote of motes) {
        mote.y -= mote.speed * dt;
        if (mote.y < 0.1) mote.y = 1;
        const { x: cx, cw, ch } = column(mote.column, tiers[mote.column] ?? top);
        const x = cx + mote.x * cw;
        const y = base - mote.y * ch * 0.86;
        if (inQuiet(quiet, x, y, 6)) continue;
        drawGlow(frame, x, y, 5, palette.accent, 0.7 * (1 - mote.y));
      }
      // A slow specular pass: light crossing gold along the column strokes, one pass every eight seconds.
      const sweep = ((t % 8) / 8) * (bandW + 120) - 60 + left;
      ctx.setLineDash(allIllustrative ? [3, 6] : []);
      tiers.forEach((share, i) => {
        const { x, cw, ch } = column(i, share);
        const distance = Math.abs(x + cw / 2 - sweep);
        if (distance > colW) return;
        ctx.strokeStyle = rgba(ctx, palette.accent, 0.5 * (1 - distance / colW));
        ctx.strokeRect(x, base - ch, cw, ch);
      });
      ctx.setLineDash([]);
      ctx.globalCompositeOperation = "source-over";
    },
  };
}