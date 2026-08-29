import { useId } from "react";
import type { DeviceArchetype } from "../../../lib/deviceArchetype";
import { DEVICE_GEOMETRY, screenBox } from "../../../lib/deviceGeometry";
import { Camera, ChinMark, Enclosure, HomeBar, Island, KeyPlate, MonitorStand, SideButtons, WedgeBase } from "./silhouetteParts";

export interface DeviceSilhouetteProps {
  archetype: DeviceArchetype;
  /** Fraction of the screen aperture the leader's keyboard covers, from the bottom. */
  keyboardShare: number;
  /** Draw the leader's virtual keyboard in the space its grid vacated. */
  kbOpen: boolean;
  /** Light the screen when this device owns a session's lease. */
  screenLit?: boolean;
}

/**
 * One archetype's enclosure, drawn in device units.
 *
 * The viewBox aspect equals the element aspect — the frame is sized from the
 * same geometry table — so the default `preserveAspectRatio` scales everything
 * uniformly. Corner radii stay circular and details keep their proportions at
 * every size, which the previous stretched unit-square viewBox could not do.
 */
export function DeviceSilhouette({ archetype, keyboardShare, kbOpen, screenLit = false }: DeviceSilhouetteProps) {
  const geometry = DEVICE_GEOMETRY[archetype];
  const box = screenBox(geometry);
  const plateHeight = box.height * Math.min(1, Math.max(0, keyboardShare));
  const plateTop = box.y + box.height - plateHeight;
  // Anything presented as being on the leader's screen is clipped to the screen
  // opening, so a square plate cannot overhang the rounded bezel corners.
  const screenClip = useId().replace(/:/g, "");

  return <svg
    data-testid="device-silhouette"
    data-screen-lit={screenLit ? "true" : "false"}
    aria-hidden="true"
    className="absolute inset-0 h-full w-full overflow-visible"
    viewBox={`0 0 ${String(geometry.width)} ${String(geometry.height)}`}
  >
    <defs>
      <clipPath id={screenClip}>
        <rect x={box.x} y={box.y} width={box.width} height={box.height} rx={geometry.screenRadius} />
      </clipPath>
    </defs>
    {geometry.base === "wedge" && <WedgeBase geometry={geometry} />}
    {geometry.base === "stand" && <MonitorStand geometry={geometry} />}
    <Enclosure geometry={geometry} screenLit={screenLit} />
    {kbOpen && plateHeight > 0 && <g clipPath={`url(#${screenClip})`}>
      <KeyPlate x={box.x} y={plateTop} width={box.width} height={plateHeight} />
    </g>}
    {archetype === "phone" && <><Island geometry={geometry} /><HomeBar geometry={geometry} /><SideButtons geometry={geometry} /></>}
    {archetype === "tablet" && <><Camera geometry={geometry} /><HomeBar geometry={geometry} /></>}
    {archetype === "laptop" && <Camera geometry={geometry} />}
    {(archetype === "monitor" || archetype === "ultrawide") && <ChinMark geometry={geometry} />}
  </svg>;
}
