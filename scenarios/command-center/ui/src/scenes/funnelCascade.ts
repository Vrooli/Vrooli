import { clipOutsideQuiet, drawGlow, focalPoint, rgba, slotRows, type Frame, type Scene, type SlotManifest } from "./engine";

const rowColumns = { key: { type: "string" }, value: { type: "number" }, share: { type: "number" } };
export const slotManifest: SlotManifest = {
  ladder: { shape: "rows", role: "primary", whenUnbound: "decorative", columns: rowColumns },
};

export interface SceneRung { rank: number; label: string; status: string; next: boolean }

/**
 * The rungs the scene draws: the last shipped rung for footing, then the next
 * unshipped rung and those above it, at most `limit`. Rows arrive in rank order.
 */
export function visibleRungs(rows: Array<{ value: number; label?: string; detail?: string }>, limit = 7): { rungs: SceneRung[]; above: number } {
  if (!rows.length) return { rungs: [], above: 0 };
  const nextIndex = rows.findIndex((row) => row.detail !== "SHIPPED");
  const start = nextIndex < 0 ? Math.max(0, rows.length - limit) : Math.max(0, nextIndex - 1);
  const window = rows.slice(start, start + limit);
  return {
    rungs: window.map((row, index) => ({ rank: row.value, label: row.label ?? "", status: row.detail ?? "", next: start + index === nextIndex })),
    above: Math.max(0, rows.length - start - window.length),
  };
}

const LIT = new Set(["TRIGGER_MET", "PROPOSED", "ACTIVE"]);

/** The figure layer's edge padding (`--viewport-pad` in design-tokens.css), in CSS pixels. */
const viewportPad = (w: number): number => Math.min(52, Math.max(16, w * 0.026));

/**
 * The rung spacing and foot position that keep the whole ladder, its labels and
 * "+N more above" between the eyebrow and the bottom edge, centred on `centerY`
 * where the band allows it.
 */
export function ladderSpan(count: number, centerY: number, w: number, h: number): { gap: number; footY: number } {
  const top = viewportPad(w) * 3.2;
  const bottom = h - viewportPad(w) * 1.6;
  const gap = Math.max(8, Math.min(66, (h * 0.62) / Math.max(count, 3), (bottom - top) / (count + 1)));
  const centred = centerY + ((count - 1) * gap) / 2;
  return { gap, footY: Math.min(bottom - gap * 0.5, Math.max(centred, top + count * gap)) };
}

function drawUnlabelled(frame: Frame): void {
  const { ctx, w, h, palette } = frame;
  const center = focalPoint(frame);
  const gap = Math.min(64, h * 0.12);
  const left = center.x - Math.min(w * 0.14, 160);
  ctx.strokeStyle = rgba(ctx, palette.primary, 0.3);
  ctx.lineWidth = 1.5;
  for (let index = 0; index < 3; index += 1) {
    const y = center.y + gap - index * gap;
    ctx.beginPath();
    ctx.moveTo(left, y);
    ctx.lineTo(left + Math.min(w * 0.28, 320), y);
    ctx.stroke();
  }
}

/** Release ladder: labelled rungs climbing from the next release, the one the hero names, lit at the foot. */
export function funnelCascade(): Scene {
  return {
    init() {},
    draw(frame) {
      const { ctx, w, h, t, tier, palette, data } = frame;
      const { rungs, above } = visibleRungs(slotRows(data, "ladder") ?? []);
      clipOutsideQuiet(frame);
      if (!rungs.length) {
        drawUnlabelled(frame);
        ctx.restore();
        return;
      }
      const center = focalPoint(frame);
      const fontPx = Math.round(Math.max(11, Math.min(16, h * 0.018)));
      ctx.font = `500 ${fontPx}px ui-monospace, SFMono-Regular, Menlo, Consolas, monospace`;
      ctx.textBaseline = "middle";
      const widest = Math.max(...rungs.map((rung) => ctx.measureText(`${rung.rank}  ${rung.label}`).width));
      const railWidth = Math.min(w * 0.16, 190);
      const { gap, footY } = ladderSpan(rungs.length, center.y, w, h);
      let left = center.x - (railWidth + 16 + widest) / 2;
      left = Math.min(left, w - 24 - widest - 16 - railWidth);
      const right = left + railWidth;
      const topY = footY - (rungs.length - 1) * gap;

      for (const x of [left, right]) {
        const rail = ctx.createLinearGradient(0, topY - gap, 0, footY + gap * 0.5);
        rail.addColorStop(0, rgba(ctx, palette.primary, 0));
        rail.addColorStop(0.25, rgba(ctx, palette.primary, 0.42));
        rail.addColorStop(1, rgba(ctx, palette.primary, 0.42));
        ctx.strokeStyle = rail;
        ctx.lineWidth = 1.4;
        ctx.beginPath();
        ctx.moveTo(x, topY - gap);
        ctx.lineTo(x, footY + gap * 0.5);
        ctx.stroke();
      }

      rungs.forEach((rung, index) => {
        const y = footY - index * gap;
        const shipped = rung.status === "SHIPPED";
        const alpha = rung.next ? 0.95 : shipped ? 0.28 : 0.5;
        ctx.strokeStyle = rgba(ctx, rung.next ? palette.primary : shipped ? palette.foreground : palette.primary, alpha);
        ctx.lineWidth = rung.next ? 3 : 1.6;
        ctx.beginPath();
        ctx.moveTo(left, y);
        ctx.lineTo(right, y);
        ctx.stroke();
        if (rung.next) {
          const pulse = tier === "full" ? 0.5 + 0.5 * Math.sin(t * 1.4) : 0.6;
          drawGlow(frame, left, y, 10 + pulse * 6, palette.primary, 0.35 + pulse * 0.25);
          drawGlow(frame, right, y, 10 + pulse * 6, palette.primary, 0.35 + pulse * 0.25);
        }
        const dotX = left - 14;
        ctx.beginPath();
        ctx.arc(dotX, y, 4, 0, Math.PI * 2);
        if (LIT.has(rung.status) || shipped) {
          ctx.fillStyle = shipped ? rgba(ctx, palette.foreground, 0.5) : palette.accent;
          ctx.fill();
        } else {
          ctx.strokeStyle = rgba(ctx, palette.foreground, 0.5);
          ctx.lineWidth = 1.2;
          ctx.stroke();
        }
        ctx.fillStyle = rgba(ctx, palette.foreground, rung.next ? 1 : shipped ? 0.4 : 0.66);
        ctx.fillText(`${rung.rank}  ${rung.label}`, right + 16, y);
      });

      if (above > 0) {
        ctx.fillStyle = rgba(ctx, palette.foreground, 0.4);
        ctx.fillText(`+${above} more above`, right + 16, topY - gap * 0.8);
      }
      ctx.restore();
    },
  };
}
