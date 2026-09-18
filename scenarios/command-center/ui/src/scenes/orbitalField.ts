import { drawGlow, focalPoint, inQuiet, slot, rgba, type Scene, type SlotManifest } from "./engine";

export const slotManifest: SlotManifest = {
  running: { shape: "scalar", role: "primary", whenUnbound: "decorative" },
  healthy: { shape: "scalar", role: "secondary", whenUnbound: "decorative" },
};

interface Star { x: number; y: number; z: number; twinkle: number }
interface Body { rx: number; ry: number; period: number; phase: number; tilt: number; size: number; healthy: boolean; trail: Array<[number, number]>; x: number; y: number; wobble: number }
interface RingLayer { canvas: HTMLCanvasElement; key: string }

/** Mission Control: every running scenario is a body on its own orbit; health is emission. */
export function orbitalField(): Scene {
  let stars: Star[] = [];
  let bodies: Body[] = [];
  let rings: RingLayer | null = null;
  return {
    init(frame) {
      const { rng, tier, data } = frame;
      const starCount = tier === "full" ? 2400 : 900;
      stars = Array.from({ length: starCount }, () => ({ x: rng() * 1.4 - 0.2, y: rng() * 1.4 - 0.2, z: 0.15 + rng() ** 2 * 0.85, twinkle: rng() * Math.PI * 2 }));
      const runningValue = slot(data, "running");
      const healthyValue = slot(data, "healthy");
      const running = Math.max(0, Math.round(runningValue ?? 0));
      const healthy = Math.max(0, Math.round(healthyValue ?? 0));
      const count = runningValue === null ? 0 : Math.min(running, tier === "full" ? 110 : 48);
      const unhealthyEvery = running > healthy ? Math.max(1, Math.round(count / (running - healthy))) : Infinity;
      bodies = Array.from({ length: count }, (_, i) => {
        const r = 0.14 + rng() ** 0.8 * 0.36;
        return { rx: r, ry: r * (0.28 + rng() * 0.4), period: 50 + r * 320 + rng() * 60, phase: rng() * Math.PI * 2, tilt: (rng() - 0.5) * 0.9, size: 1.6 + rng() * 2.2, healthy: (i + 1) % unhealthyEvery !== 0, trail: [], x: 0, y: 0, wobble: 0 };
      });
      rings = null;
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
      rings = drawRings(ctx, bodies, focal, scale, rgba(ctx, palette.primary, 0.05), rings);
      ctx.globalCompositeOperation = "lighter";
      const trailCap = tier === "full" ? 26 : 12;
      for (const body of bodies) {
        const angle = body.phase + (t / body.period) * Math.PI * 2;
        const ox = Math.cos(angle) * body.rx * scale;
        const oy = Math.sin(angle) * body.ry * scale;
        body.x = focal.x + ox * Math.cos(body.tilt) - oy * Math.sin(body.tilt);
        body.y = focal.y + ox * Math.sin(body.tilt) + oy * Math.cos(body.tilt);
        body.wobble = body.healthy ? 0 : Math.sin(t * 3 + body.phase) * 3;
        body.trail.push([body.x + body.wobble, body.y]);
        if (body.trail.length > trailCap) body.trail.shift();
      }
      // Every body advances once per frame, so all trails share one length and a
      // segment's fade depends only on its age: one path per colour and age
      // replaces a separate stroke (and colour parse) per segment.
      const trailLength = bodies[0]?.trail.length ?? 0;
      for (const healthy of [true, false]) {
        ctx.strokeStyle = rgba(ctx, healthy ? palette.primary : palette.warning, 1);
        for (let i = 1; i < trailLength; i += 1) {
          ctx.globalAlpha = (i / trailLength) * 0.35;
          ctx.beginPath();
          for (const body of bodies) {
            if (body.healthy !== healthy) continue;
            const from = body.trail[i - 1];
            const to = body.trail[i];
            if (!from || !to) continue;
            ctx.moveTo(from[0], from[1]);
            ctx.lineTo(to[0], to[1]);
          }
          ctx.stroke();
        }
      }
      ctx.globalAlpha = 1;
      const core = rgba(ctx, palette.foreground, 0.9);
      for (const body of bodies) {
        if (inQuiet(quiet, body.x, body.y, 8)) continue;
        const x = body.x + body.wobble;
        drawGlow(frame, x, body.y, body.size * 4.5, body.healthy ? palette.primary : palette.warning, body.healthy ? 0.7 : 0.9);
        ctx.fillStyle = core;
        ctx.beginPath();
        ctx.arc(x, body.y, body.size * 0.5, 0, Math.PI * 2);
        ctx.fill();
      }
      ctx.globalCompositeOperation = "source-over";
    },
  };
}

/**
 * The orbit rings stay put while the layout holds, yet stroking every full
 * ellipse each frame was most of this scene's cost. They are drawn once into a
 * layer in device pixels and copied at an integer offset, so the pixels match a
 * direct stroke; the layer redraws when the focal point, scale, colour or
 * backing transform changes.
 */
function drawRings(ctx: CanvasRenderingContext2D, bodies: Body[], focal: { x: number; y: number }, scale: number, color: string, cached: RingLayer | null): RingLayer | null {
  const { a, d, e, f } = ctx.getTransform();
  const radius = Math.max(0, ...bodies.map((body) => body.rx)) * scale + 2;
  const left = Math.floor((focal.x - radius) * a + e);
  const top = Math.floor((focal.y - radius) * d + f);
  const key = `${focal.x},${focal.y},${scale},${a},${d},${e},${f},${color}`;
  let layer = cached;
  if (layer?.key !== key) {
    const canvas = layer?.canvas ?? document.createElement("canvas");
    canvas.width = Math.ceil(2 * radius * a) + 2;
    canvas.height = Math.ceil(2 * radius * d) + 2;
    const layerCtx = canvas.getContext("2d");
    if (!layerCtx) {
      strokeRings(ctx, bodies, focal, scale, color);
      return null;
    }
    layerCtx.setTransform(a, 0, 0, d, e - left, f - top);
    strokeRings(layerCtx, bodies, focal, scale, color);
    layer = { canvas, key };
  }
  ctx.save();
  ctx.setTransform(1, 0, 0, 1, 0, 0);
  ctx.drawImage(layer.canvas, left, top);
  ctx.restore();
  return layer;
}

function strokeRings(ctx: CanvasRenderingContext2D, bodies: Body[], focal: { x: number; y: number }, scale: number, color: string): void {
  ctx.lineWidth = 1;
  ctx.strokeStyle = color;
  for (const body of bodies) {
    ctx.beginPath();
    ctx.ellipse(focal.x, focal.y, body.rx * scale, body.ry * scale, body.tilt, 0, Math.PI * 2);
    ctx.stroke();
  }
}
