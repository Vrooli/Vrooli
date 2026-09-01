import { drawGlow, focalPoint, inQuiet, rgba, type Scene } from "./engine";

const NODES: Array<{ id: string; label: string; color: string; metric: string }> = [
  { id: "mission-control", label: "MISSION CONTROL", color: "#33d6ff", metric: "composite_system_health" },
  { id: "hive", label: "THE HIVE", color: "#2ee6b3", metric: "composite_portfolio" },
  { id: "forge", label: "THE FORGE", color: "#ff9b3d", metric: "composite_throughput" },
  { id: "ledger", label: "LEDGER", color: "#d9b64a", metric: "composite_revenue" },
  { id: "broadcast", label: "BROADCAST", color: "#ff44b0", metric: "composite_reach" },
];

/** Panorama: the five rooms as one slowly turning object; each ring carries its room's provenance. */
export function panoramaConstellation(): Scene {
  return {
    init() {},
    draw(frame) {
      const { ctx, w, h, t, palette, quiet, data } = frame;
      const focal = focalPoint(frame);
      const radius = Math.min(w, h) * 0.3;
      const positions = NODES.map((node, i) => {
        const angle = (i / NODES.length) * Math.PI * 2 + t * (Math.PI * 2 / 220) - Math.PI / 2;
        return { node, x: focal.x + Math.cos(angle) * radius * 0.98, y: focal.y + Math.sin(angle) * radius * 0.7 };
      });
      ctx.lineWidth = 1;
      for (let i = 0; i < positions.length; i += 1) {
        const a = positions[i];
        const b = positions[(i + 1) % positions.length];
        if (!a || !b) continue;
        const sag = 18 + Math.sin(t * 0.3 + i) * 6;
        ctx.strokeStyle = rgba(ctx, palette.primary, 0.18);
        ctx.beginPath();
        ctx.moveTo(a.x, a.y);
        ctx.quadraticCurveTo((a.x + b.x) / 2, (a.y + b.y) / 2 + sag, b.x, b.y);
        ctx.stroke();
        ctx.strokeStyle = rgba(ctx, palette.primary, 0.1);
        ctx.beginPath();
        ctx.moveTo(focal.x, focal.y);
        ctx.quadraticCurveTo((a.x + focal.x) / 2, (a.y + focal.y) / 2 + sag * 0.6, a.x, a.y);
        ctx.stroke();
      }
      ctx.globalCompositeOperation = "lighter";
      drawGlow(frame, focal.x, focal.y, radius * 0.5, palette.primary, 0.35);
      drawGlow(frame, focal.x, focal.y, 12, palette.accent, 1);
      ctx.globalCompositeOperation = "source-over";
      for (const { node, x, y } of positions) {
        if (inQuiet(quiet, x, y, 40)) continue;
        const ink = data.readings[node.metric]?.ink ?? "dotted";
        const ringR = 22 + Math.sin(t * 0.5 + x) * 1.5;
        ctx.setLineDash(ink === "dotted" ? [2, 5] : []);
        ctx.lineWidth = ink === "hollow" || ink === "dotted" ? 1.4 : 3;
        ctx.strokeStyle = rgba(ctx, node.color, ink === "dimmed" ? 0.5 : 0.95);
        ctx.beginPath();
        ctx.arc(x, y, ringR, 0, Math.PI * 2);
        ctx.stroke();
        ctx.setLineDash([]);
        if (ink === "solid" || ink === "dimmed") {
          ctx.globalCompositeOperation = "lighter";
          drawGlow(frame, x, y, ringR * 1.8, node.color, ink === "solid" ? 0.5 : 0.2);
          ctx.globalCompositeOperation = "source-over";
        }
        ctx.fillStyle = rgba(ctx, palette.foreground, 0.75);
        ctx.font = `600 ${Math.max(9, Math.min(12, w / 120))}px "JetBrains Mono", monospace`;
        ctx.textAlign = "center";
        ctx.fillText(node.label, x, y + ringR + 18);
      }
    },
  };
}
