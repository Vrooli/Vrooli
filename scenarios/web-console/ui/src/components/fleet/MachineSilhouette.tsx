import { MACHINE_GEOMETRY, type MachineArchetype, type MachineState } from "../../lib/machineGeometry";
import { ChassisFeet, Enclosure, RackEars, StatusLamp, VentFace } from "../terminal/device/silhouetteParts";

export interface MachineSilhouetteProps {
  /**
   * Which chassis to draw. Only `chassis` is reachable from the fleet drawer
   * today: the target contract carries `kind`, `os` and `arch`, and none of
   * them proves a machine is rack-mounted, a mini PC or a tower. The rest of
   * the table is drawn only by a caller that can prove which one applies.
   */
  archetype?: MachineArchetype;
  /** Whether the machine answers, which is what the lamp reports. */
  state: MachineState;
}

/**
 * [REQ:P0-002e] A machine's enclosure, drawn in device units.
 *
 * The sibling of `DeviceSilhouette`, sharing its parts, its tokens and its
 * viewBox discipline — the viewBox aspect equals the element aspect, so corner
 * radii stay circular and details keep their proportions at every size.
 *
 * It differs in what it is willing to claim. A device silhouette draws a screen
 * because a device *is* a screen somebody is looking at; a machine is compute,
 * usually headless, so its recess is a vent face and reachability is carried by
 * the status lamp instead of by inventing a shape per guess.
 */
export function MachineSilhouette({ archetype = "chassis", state }: MachineSilhouetteProps) {
  const geometry = MACHINE_GEOMETRY[archetype];
  const live = state === "dispatchable";
  // Ears are drawn outside the panel on the horizontal axis, so they need
  // margin rather than the extra height a base occupies.
  const margin = geometry.base === "ears" ? 22 : 16;
  const below = geometry.base === "feet" ? geometry.baseHeight : 0;

  return <svg
    data-testid="machine-silhouette"
    data-archetype={archetype}
    data-state={state}
    aria-hidden="true"
    className="absolute inset-0 h-full w-full"
    viewBox={`${String(-margin)} ${String(-margin)} ${String(geometry.width + margin * 2)} ${String(geometry.height + below + margin * 2)}`}
  >
    {geometry.base === "ears" && <RackEars geometry={geometry} />}
    {geometry.base === "feet" && <ChassisFeet geometry={geometry} />}
    <Enclosure geometry={geometry} faceFill={live ? "var(--wc-machine-face-live)" : undefined} />
    <VentFace geometry={geometry} live={live} />
    <StatusLamp geometry={geometry} state={state} />
  </svg>;
}

export default MachineSilhouette;
