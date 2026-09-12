import { clipOutsideQuiet, drawGlow, focalPoint, inQuiet, mulberry32, read, rgba, seedFrom, type Scene } from "./engine";

interface Receiver { x: number; y: number; lit: number }

/** Broadcast: waves radiate from a transmitter and dim by each funnel stage's drop-off. */
export function signalConstellation(): Scene {
  let receivers: Receiver[] = [];
  const period = 2.2;
  const travel = 9;
  return {
    init(frame) {
      const { tier } = frame;
      const stableRng = mulberry32(seedFrom("broadcast-receivers"));
      receivers = Array.from({ length: tier === "full" ? 140 : 60 }, () => ({ x: stableRng(), y: stableRng(), lit: 0 }));
    },
    draw(frame) {
      const { ctx, w, h, t, dt, palette, quiet, data } = frame;
      const focal = focalPoint(frame);
      const visitors = Math.max(1, read(data, "visitors", 2840));
      const stages = [1, read(data, "conversions", 61) / visitors, read(data, "cta_clicks", 412) / visitors];
      const stageRadius = Math.max(w, h) * 0.9;
      clipOutsideQuiet(frame);
      ctx.lineWidth = 1.2;
      for (let k = 0; k < travel / period; k += 1) {
        const age = ((t + k * period) % travel);
        const progress = age / travel;
        const radius = progress * stageRadius;
        const stage = progress < 0.33 ? 0 : progress < 0.66 ? 1 : 2;
        const dropoff = stage === 0 ? 1 : Math.min(1, Math.max(0.12, Math.sqrt((stages[stage] ?? 1) * 6)));
        const alpha = (1 - progress) * 0.55 * dropoff;
        ctx.strokeStyle = rgba(ctx, palette.primary, alpha);
        ctx.beginPath();
        ctx.arc(focal.x, focal.y, radius, 0, Math.PI * 2);
        ctx.stroke();
      }
      ctx.globalCompositeOperation = "lighter";
      for (const receiver of receivers) {
        const x = receiver.x * w;
        const y = receiver.y * h;
        if (inQuiet(quiet, x, y, 6)) continue;
        const dist = Math.hypot(x - focal.x, y - focal.y);
        let lit = 0;
        for (let k = 0; k < travel / period; k += 1) {
          const radius = (((t + k * period) % travel) / travel) * stageRadius;
          if (Math.abs(radius - dist) < 14) lit = 1;
        }
        receiver.lit = Math.max(lit, receiver.lit - dt * 0.6);
        drawGlow(frame, x, y, 2.5 + receiver.lit * 6, palette.accent, 0.18 + receiver.lit * 0.7);
      }
      for (let beam = 0; beam < 2; beam += 1) {
        const angle = t * 0.08 + beam * Math.PI;
        const gradient = ctx.createLinearGradient(focal.x, focal.y, focal.x + Math.cos(angle) * stageRadius, focal.y + Math.sin(angle) * stageRadius);
        gradient.addColorStop(0, rgba(ctx, palette.primary, 0.3));
        gradient.addColorStop(1, rgba(ctx, palette.primary, 0));
        ctx.strokeStyle = gradient;
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.moveTo(focal.x, focal.y);
        ctx.lineTo(focal.x + Math.cos(angle) * stageRadius, focal.y + Math.sin(angle) * stageRadius);
        ctx.stroke();
      }
      drawGlow(frame, focal.x, focal.y, Math.min(w, h) * 0.12, palette.primary, 0.6);
      drawGlow(frame, focal.x, focal.y, 10, palette.accent, 1);
      ctx.globalCompositeOperation = "source-over";
      ctx.restore();
    },
  };
}
