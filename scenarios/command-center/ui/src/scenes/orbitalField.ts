import { drawGlow, focalPoint, inQuiet, read, rgba, type Scene } from "./engine";

interface Star { x: number; y: number; z: number; twinkle: number }
interface Body { rx: number; ry: number; period: number; phase: number; tilt: number; size: number; healthy: boolean; trail: Array<[number, number]> }

/** Mission Control: every running scenario is a body on its own orbit; health is emission. */
export function orbitalField(): Scene {
  let stars: Star[] = [];
  let bodies: Body[] = [];
  return {
    init(frame) {
      const { rng, tier, data } = frame;
      const starCount = tier === "full" ? 2400 : 900;
      stars = Array.from({ length: starCount }, () => ({ x: rng() * 1.4 - 0.2, y: rng() * 1.4 - 0.2, z: 0.15 + rng() ** 2 * 0.85, twinkle: rng() * Math.PI * 2 }));
      const running = Math.round(read(data, "active_scenarios", 48));
      const healthy = Math.round(read(data, "scenario_health", running));
      const count = Math.max(6, Math.min(running, tier === "full" ? 110 : 48));
      const unhealthyEvery = running > healthy ? Math.max(1, Math.round(count / (running - healthy))) : Infinity;
      bodies = Array.from({ length: count }, (_, i) => {
        const r = 0.14 + rng() ** 0.8 * 0.36;
        return { rx: r, ry: r * (0.28 + rng() * 0.4), period: 50 + r * 320 + rng() * 60, phase: rng() * Math.PI * 2, tilt: (rng() - 0.5) * 0.9, size: 1.6 + rng() * 2.2, healthy: (i + 1) % unhealthyEvery !== 0, trail: [] };
      });
    },
    draw(frame) {
      const { ctx, w, h, t, palette, quiet, tier } = frame;
      const focal = focalPoint(frame);
      const scale = Math.min(w, h);
      const driftX = Math.sin(t * 0.021) * 18 + Math.sin(t * 0.0083) * 12;
      const driftY = Math.cos(t * 0.017) * 12;
      ctx.fillStyle = rgba(ctx, palette.foreground, 1);
      for (const star of stars) {
        const px = star.x * w + driftX * star.z;
        const py = star.y * h + driftY * star.z;
        if (px < 0 || px > w || py < 0 || py > h) continue;
        const flicker = 0.55 + 0.45 * Math.sin(t * 0.6 + star.twinkle);
        ctx.globalAlpha = (0.08 + star.z * 0.5) * flicker * (inQuiet(quiet, px, py) ? 0.35 : 1);
        const size = star.z * 1.6;
        ctx.fillRect(px, py, size, size);
      }
      ctx.globalAlpha = 1;
      ctx.globalCompositeOperation = "lighter";
      drawGlow(frame, focal.x, focal.y, scale * 0.22, palette.primary, 0.35);
      drawGlow(frame, focal.x, focal.y, scale * 0.05, palette.accent, 0.9);
      ctx.globalCompositeOperation = "source-over";
      ctx.lineWidth = 1;
      for (const body of bodies) {
        ctx.strokeStyle = rgba(ctx, palette.primary, 0.05);
        ctx.beginPath();
        ctx.ellipse(focal.x, focal.y, body.rx * scale, body.ry * scale, body.tilt, 0, Math.PI * 2);
        ctx.stroke();
      }
      ctx.globalCompositeOperation = "lighter";
      for (const body of bodies) {
        const angle = body.phase + (t / body.period) * Math.PI * 2;
        const ox = Math.cos(angle) * body.rx * scale;
        const oy = Math.sin(angle) * body.ry * scale;
        const x = focal.x + ox * Math.cos(body.tilt) - oy * Math.sin(body.tilt);
        const y = focal.y + ox * Math.sin(body.tilt) + oy * Math.cos(body.tilt);
        const wobble = body.healthy ? 0 : Math.sin(t * 3 + body.phase) * 3;
        body.trail.push([x + wobble, y]);
        if (body.trail.length > (tier === "full" ? 26 : 12)) body.trail.shift();
        const color = body.healthy ? palette.primary : palette.warning;
        for (let i = 1; i < body.trail.length; i += 1) {
          const from = body.trail[i - 1];
          const to = body.trail[i];
          if (!from || !to) continue;
          const [ax, ay] = from;
          const [bx, by] = to;
          ctx.strokeStyle = rgba(ctx, color, (i / body.trail.length) * 0.35);
          ctx.beginPath();
          ctx.moveTo(ax, ay);
          ctx.lineTo(bx, by);
          ctx.stroke();
        }
        if (inQuiet(quiet, x, y, 8)) continue;
        drawGlow(frame, x + wobble, y, body.size * 4.5, color, body.healthy ? 0.7 : 0.9);
        ctx.fillStyle = rgba(ctx, palette.foreground, 0.9);
        ctx.beginPath();
        ctx.arc(x + wobble, y, body.size * 0.5, 0, Math.PI * 2);
        ctx.fill();
      }
      ctx.globalCompositeOperation = "source-over";
    },
  };
}
