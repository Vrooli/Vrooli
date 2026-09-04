import { drawGlow, inQuiet, read, rgba, type Scene } from "./engine";

interface Cell { x: number; y: number; phase: number; lit: boolean; healthy: boolean }
interface Wave { x: number; y: number; born: number }

/** The Hive: one cell per scenario, breathing on its own phase; an invocation is a wavefront. */
export function hiveLattice(): Scene {
  let cells: Cell[] = [];
  let waves: Wave[] = [];
  let nextWave = 2;
  let radius = 24;
  return {
    init(frame) {
      const { w, h, rng, data } = frame;
      const total = Math.max(12, Math.round(read(data, "total_scenarios", 120)));
      const running = Math.round(read(data, "composite_portfolio", read(data, "scenario_completeness", total * 0.5)));
      const healthy = Math.round(read(data, "scenario_completeness", running * 0.7));
      const area = (w * h) / (total * 2.4);
      radius = Math.max(14, Math.min(Math.sqrt(area / 2.6), Math.min(w, h) / 12));
      const dx = radius * Math.sqrt(3);
      const dy = radius * 1.5;
      const all: Array<[number, number]> = [];
      for (let row = -1; row * dy < h + radius; row += 1) {
        for (let col = -1; col * dx < w + radius; col += 1) {
          all.push([col * dx + (row % 2 ? dx / 2 : 0), row * dy]);
        }
      }
      for (let i = all.length - 1; i > 0; i -= 1) {
        const j = Math.floor(rng() * (i + 1));
        const a = all[i];
        const b = all[j];
        if (a && b) {
          all[i] = b;
          all[j] = a;
        }
      }
      cells = all.map(([x, y], i) => ({ x, y, phase: rng() * Math.PI * 2, lit: i < running, healthy: i < healthy }));
      waves = [];
    },
    draw(frame) {
      const { ctx, t, palette, quiet, tier } = frame;
      if (t > nextWave) {
        const lit = cells.filter((cell) => cell.lit && !inQuiet(quiet, cell.x, cell.y, radius));
        const origin = lit[Math.floor(frame.rng() * lit.length)];
        if (origin) waves.push({ x: origin.x, y: origin.y, born: t });
        nextWave = t + 3.5 + frame.rng() * 4;
      }
      waves = waves.filter((wave) => t - wave.born < 4);
      ctx.lineWidth = 1;
      for (const cell of cells) {
        const quietHere = inQuiet(quiet, cell.x, cell.y, radius * 0.6);
        const breath = 0.5 + 0.5 * Math.sin(t * 0.35 + cell.phase);
        let pulse = 0;
        for (const wave of waves) {
          const age = t - wave.born;
          const dist = Math.hypot(cell.x - wave.x, cell.y - wave.y);
          const ring = age * radius * 5;
          const delta = Math.abs(dist - ring);
          if (delta < radius * 1.4) pulse = Math.max(pulse, (1 - delta / (radius * 1.4)) * (1 - age / 4));
        }
        const base = cell.lit ? (cell.healthy ? 0.32 : 0.16) : 0.045;
        const alpha = (base + breath * (cell.lit ? 0.16 : 0.02) + pulse * 0.6) * (quietHere ? 0.3 : 1);
        ctx.beginPath();
        for (let side = 0; side < 6; side += 1) {
          const angle = Math.PI / 6 + (Math.PI / 3) * side;
          const px = cell.x + Math.cos(angle) * radius * 0.86;
          const py = cell.y + Math.sin(angle) * radius * 0.86;
          if (side === 0) ctx.moveTo(px, py);
          else ctx.lineTo(px, py);
        }
        ctx.closePath();
        ctx.strokeStyle = rgba(ctx, palette.primary, Math.min(0.9, alpha + 0.06));
        ctx.stroke();
        if (cell.lit) {
          ctx.fillStyle = rgba(ctx, palette.primary, alpha * 0.35);
          ctx.fill();
        }
        if (cell.lit && !quietHere && tier === "full" && (cell.healthy || pulse > 0.2)) {
          ctx.globalCompositeOperation = "lighter";
          drawGlow(frame, cell.x, cell.y, radius * (1.1 + pulse), cell.healthy ? palette.primary : palette.warning, 0.22 + pulse * 0.6);
          ctx.globalCompositeOperation = "source-over";
        }
      }
    },
  };
}