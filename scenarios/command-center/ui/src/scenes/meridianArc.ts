import { geoGraticule10, geoOrthographic, geoPath } from "d3-geo";
import { feature } from "topojson-client";
import type { Topology } from "topojson-specification";
import worldTopology from "world-atlas/countries-110m.json";
import { clipOutsideQuiet, drawGlow, focalPoint, mulberry32, rgba, seedFrom, type Scene } from "./engine";

type GeoPoint = [longitude: number, latitude: number];

const WORLD_TOPOLOGY = worldTopology as unknown as Topology;
const WORLD_OBJECT = WORLD_TOPOLOGY.objects.countries;
if (!WORLD_OBJECT) throw new Error("world-atlas countries topology is missing");
const WORLD = feature(WORLD_TOPOLOGY, WORLD_OBJECT);
const GRATICULE = geoGraticule10();
const VIRGINIA: GeoPoint = [-77.487, 39.043];
const DESTINATIONS: GeoPoint[] = [
  [-0.1276, 51.5072], [-46.6333, -23.5505], [103.8198, 1.3521], [151.2093, -33.8688],
  [139.6917, 35.6895], [18.4241, -33.9249], [37.6173, 55.7558], [-99.1332, 19.4326],
  [-74.006, 40.7128], [2.3522, 48.8566], [72.8777, 19.076], [114.1694, 22.3193],
];

const EARTH_STYLES = [
  { landAlpha: 0.72, gridAlpha: 0.18, coastWidth: 1.25, routeAlpha: 0.92, routeWidth: 2.2 },
  { landAlpha: 0.5, gridAlpha: 0.32, coastWidth: 0.9, routeAlpha: 1, routeWidth: 1.6 },
  { landAlpha: 0.9, gridAlpha: 0.1, coastWidth: 1.65, routeAlpha: 0.78, routeWidth: 3.1 },
] as const;

/** Geographic traffic rendered as a restrained neon Earth, not invented screen polygons. */
export function meridianArc(): Scene {
  let receivers: Array<{ x: number; y: number }> = [];
  return {
    init(frame) {
      const stableRng = mulberry32(seedFrom("broadcast-receivers"));
      receivers = Array.from({ length: frame.tier === "full" ? 140 : 60 }, () => ({ x: stableRng(), y: stableRng() }));
    },
    draw(frame) {
      const { ctx, w, h, palette, data, t } = frame;
      const focus = data.readings[data.focus ?? ""];
      const shares = focus?.rows?.map((row) => Math.max(0, Math.min(1, row.share))) ?? [1];
      const focal = focalPoint(frame);
      // Broadcast is a room-scale panorama. Keep the globe behind the figures,
      // but give it enough presence to read as the room's world rather than a
      // small widget in the open column.
      const origin = { x: Math.min(w * 0.72, focal.x + w * 0.06), y: h * 0.42 };
      const globeRadius = Math.min(w, h) * 0.34;
      const style = EARTH_STYLES[Math.floor(t / 12) % EARTH_STYLES.length] ?? EARTH_STYLES[0];
      const projection = geoOrthographic()
        .scale(globeRadius)
        .translate([origin.x, origin.y])
        .center(VIRGINIA)
        .rotate([t * 4, -18, 8])
        .clipAngle(90);
      const path = geoPath(projection, ctx);

      ctx.lineCap = "round";
      ctx.save();
      ctx.translate(origin.x, origin.y);
      ctx.rotate(-0.16);
      ctx.translate(-origin.x, -origin.y);
      drawGlow(frame, origin.x, origin.y, globeRadius * 1.35, palette.primary, 0.12);

      ctx.strokeStyle = rgba(ctx, palette.accent, style.landAlpha);
      ctx.lineWidth = style.coastWidth;
      path(WORLD);
      ctx.stroke();

      ctx.strokeStyle = rgba(ctx, palette.primary, style.gridAlpha);
      ctx.lineWidth = 0.75;
      path(GRATICULE);
      ctx.stroke();

      const source = projection(VIRGINIA);
      if (source) {
        drawGlow(frame, source[0], source[1], 10, palette.warning, 0.95);
        ctx.fillStyle = palette.warning;
        ctx.beginPath();
        ctx.arc(source[0], source[1], 2.7, 0, Math.PI * 2);
        ctx.fill();
      }

      shares.slice(0, frame.tier === "full" ? DESTINATIONS.length : 6).forEach((share, index) => {
        const destination = DESTINATIONS[index];
        if (!destination || !source) return;
        drawGreatCircleRoute(frame, projection, origin.x, origin.y, globeRadius, VIRGINIA, destination, share, style.routeAlpha, style.routeWidth);
      });
      // End the globe's tilt transform before drawing the room-wide ambient
      // layer. Leaving this save open made every subsequent frame inherit the
      // globe rotation, which appeared as rotating rectangular borders.
      ctx.restore();
      // Keep ambient receiver noise out of the figures, but never carve the
      // geographic object itself into pieces around those quiet zones.
      clipOutsideQuiet(frame);
      receivers.forEach((receiver, index) => {
        const pulse = 0.45 + 0.55 * Math.max(0, Math.sin(t * 2.8 + index * 0.73));
        drawGlow(frame, receiver.x * w, receiver.y * h, 2 + pulse * 4, palette.accent, 0.18 + pulse * 0.62);
      });
      ctx.restore();
    },
  };
}

function drawGreatCircleRoute(
  frame: Parameters<Scene["draw"]>[0],
  projection: ReturnType<typeof geoOrthographic>,
  centerX: number,
  centerY: number,
  globeRadius: number,
  source: GeoPoint,
  destination: GeoPoint,
  share: number,
  alpha: number,
  width: number,
): void {
  const { ctx, palette } = frame;
  const sourcePoint = projection(source);
  const target = projection(destination);
  if (!sourcePoint || !target) return;
  ctx.strokeStyle = rgba(ctx, palette.primary, alpha * (0.35 + share * 0.65));
  // Routes are hairlines in the atmosphere, never a second solid shape on
  // the globe. The control point lifts the route away from the surface.
  ctx.lineWidth = Math.max(0.7, width * 0.32 + share * 0.35);
  const midX = (sourcePoint[0] + target[0]) / 2;
  const midY = (sourcePoint[1] + target[1]) / 2;
  const awayX = midX - centerX;
  const awayY = midY - centerY;
  const distance = Math.max(1, Math.hypot(awayX, awayY));
  const lift = globeRadius * (0.18 + share * 0.08);
  const control: [number, number] = [midX + (awayX / distance) * lift, midY + (awayY / distance) * lift];
  ctx.beginPath();
  ctx.moveTo(sourcePoint[0], sourcePoint[1]);
  ctx.quadraticCurveTo(control[0], control[1], target[0], target[1]);
  ctx.stroke();
  if (target) drawGlow(frame, target[0], target[1], 3 + share * 8, palette.accent, 0.3 + share * 0.6);
}
