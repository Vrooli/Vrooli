import { drawGlow, inQuiet, read, rgba, type Frame, type Scene } from "./engine";

interface Spark { x: number; lane: number; speed: number; size: number; age: number; pooled: boolean; swirl: number }

/** The Forge: a current of sparks, one per unit of throughput; blocked work pools behind a dam. */
export function flowCurrent(): Scene {
  let sparks: Spark[] = [];
  let emitDebt = 0;
  let rate = 4;
  let pool = 0;
  const lanes = 3;
  const laneY = (frame: Frame, lane: number, x: number): number => {
    const { h, t, w } = frame;
    const base = h * (0.42 + lane * 0.12);
    return base + Math.sin(x / w * 5.2 + t * 0.35 + lane) * h * 0.05 + Math.sin(x / w * 11 - t * 0.2) * h * 0.015;
  };
  return {
    init(frame) {
      const { data, tier } = frame;
      const created = read(data, "throughput_stats", 40);
      rate = Math.max(4, Math.min(tier === "full" ? 18 : 8, created / 3));
      pool = Math.min(40, Math.round(read(data, "blocking_stats", 0)) * 3);
      // Pre-warm so the first frame, and the still tier, already show a current in motion.
      const { w, rng } = frame;
      sparks = Array.from({ length: Math.round(rate * 9) }, () => ({ x: rng() * w, lane: Math.floor(rng() * lanes), speed: w * (0.05 + rng() * 0.06), size: 1 + rng() * 2.2, age: rng() * 8, pooled: false, swirl: rng() * Math.PI * 2 }));
      emitDebt = 0;
    },
    draw(frame) {
      const { ctx, w, h, dt, t, palette, quiet, rng } = frame;
      const damX = w * 0.8;
      const heat = ctx.createLinearGradient(0, h * 0.55, 0, h);
      heat.addColorStop(0, rgba(ctx, palette.primary, 0));
      heat.addColorStop(1, rgba(ctx, palette.primary, 0.12));
      ctx.fillStyle = heat;
      ctx.fillRect(0, h * 0.55, w, h * 0.45);
      ctx.lineWidth = 1;
      for (let lane = 0; lane < lanes; lane += 1) {
        ctx.strokeStyle = rgba(ctx, palette.primary, 0.22);
        ctx.lineWidth = lane === 1 ? 1.6 : 1;
        ctx.beginPath();
        for (let x = 0; x <= w; x += 12) {
          const y = laneY(frame, lane, x);
          if (x === 0) ctx.moveTo(x, y);
          else ctx.lineTo(x, y);
        }
        ctx.stroke();
      }
      if (pool > 0) {
        ctx.setLineDash([4, 6]);
        ctx.strokeStyle = rgba(ctx, palette.foreground, 0.35);
        ctx.beginPath();
        ctx.moveTo(damX, h * 0.3);
        ctx.lineTo(damX, h * 0.85);
        ctx.stroke();
        ctx.setLineDash([]);
      }
      emitDebt += rate * dt;
      while (emitDebt >= 1) {
        emitDebt -= 1;
        sparks.push({ x: -10, lane: Math.floor(rng() * lanes), speed: w * (0.05 + rng() * 0.06), size: 1 + rng() * 2.2, age: 0, pooled: false, swirl: rng() * Math.PI * 2 });
      }
      const pooled = sparks.filter((spark) => spark.pooled).length;
      ctx.globalCompositeOperation = "lighter";
      sparks = sparks.filter((spark) => {
        spark.age += dt;
        if (!spark.pooled) {
          spark.x += spark.speed * dt;
          if (pool > 0 && spark.x > damX - 16 && pooled < pool) spark.pooled = true;
        }
        const y = spark.pooled
          ? laneY(frame, spark.lane, damX - 40) + Math.sin(t * 0.8 + spark.swirl) * h * 0.06
          : laneY(frame, spark.lane, spark.x) + Math.sin(spark.age * 6 + spark.swirl) * 3;
        const x = spark.pooled ? damX - 40 - Math.abs(Math.cos(t * 0.5 + spark.swirl)) * 70 : spark.x;
        const life = spark.pooled ? 1 : 1 - Math.max(0, (spark.x - w * 0.6) / (w * 0.45));
        if (life <= 0) return false;
        if (spark.pooled && spark.age > 60) return false;
        if (!inQuiet(quiet, x, y, 6)) {
          drawGlow(frame, x, y, spark.size * 9, spark.pooled ? palette.warning : palette.primary, 0.9 * life);
          ctx.fillStyle = rgba(ctx, palette.accent, life);
          ctx.beginPath();
          ctx.arc(x, y, spark.size * 0.9, 0, Math.PI * 2);
          ctx.fill();
        }
        return spark.x < w + 20;
      });
      for (let i = 0; i < 40; i += 1) {
        const ex = ((i * 173 + t * (6 + (i % 5))) % w);
        const ey = h - ((t * (10 + (i % 7)) + i * 97) % (h * 0.95));
        if (inQuiet(quiet, ex, ey, 4)) continue;
        drawGlow(frame, ex, ey, 2 + (i % 3), palette.accent, 0.5 * (ey / h));
      }
      ctx.globalCompositeOperation = "source-over";
    },
  };
}
