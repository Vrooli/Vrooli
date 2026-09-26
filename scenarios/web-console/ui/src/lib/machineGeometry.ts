// [REQ:P0-002e] Machine chassis geometry.
//
// The sibling of `deviceGeometry.ts`, and deliberately a separate table rather
// than extra rows in `DEVICE_GEOMETRY`. `followerViewport.ts` iterates
// `DEVICE_ARCHETYPES` to fit the live follower viewport — screen apertures,
// stand extents, frame aspects — and its tests loop over every member. A
// machine never enters that path, so a machine shape must never enter that
// enum.
//
// What machines and devices *do* share is the drawing: both are an
// `EnclosureGeometry` rendered by the parts in `silhouetteParts.tsx`, in device
// units, through a viewBox whose aspect matches its element.
//
// The honesty rule from the device side applies with less signal here. A
// machine declares only `kind`, `os` and `arch`; nothing tells us what the
// hardware physically is. So the shipped vocabulary is one neutral chassis, and
// reachability is carried by the status lamp rather than by inventing a shape
// per guess. The extra archetypes below exist for a caller that can one day
// prove which one applies — none can today.
import type { EnclosureGeometry } from "./deviceGeometry";

export const MACHINE_ARCHETYPES = ["chassis", "rack", "mini", "tower"] as const;

export type MachineArchetype = (typeof MACHINE_ARCHETYPES)[number];

export interface MachineGeometry extends EnclosureGeometry {
  /** How the chassis meets its surface, drawn outside the panel bounds. */
  base: "none" | "feet" | "ears";
  /** Extra device units the base occupies below the panel. */
  baseHeight: number;
  /** Number of slots in the vent grille. */
  vents: number;
  /** Share of the face width the grille runs across, leaving room for the lamp. */
  ventRun: number;
  /** Slot direction. A tall chassis vents in rows, a wide one in columns. */
  ventDirection: "horizontal" | "vertical";
}

export const MACHINE_GEOMETRY: Record<MachineArchetype, MachineGeometry> = {
  chassis: { width: 392, height: 176, radius: 10, bezel: 16, chin: 30, screenRadius: 4, base: "feet", baseHeight: 10, vents: 9, ventRun: 0.6, ventDirection: "vertical" },
  rack: { width: 468, height: 112, radius: 6, bezel: 11, chin: 16, screenRadius: 3, base: "ears", baseHeight: 0, vents: 14, ventRun: 0.58, ventDirection: "vertical" },
  mini: { width: 236, height: 236, radius: 20, bezel: 20, chin: 34, screenRadius: 6, base: "feet", baseHeight: 8, vents: 6, ventRun: 0.55, ventDirection: "vertical" },
  tower: { width: 176, height: 320, radius: 8, bezel: 14, chin: 26, screenRadius: 4, base: "feet", baseHeight: 9, vents: 7, ventRun: 0.62, ventDirection: "horizontal" },
};

/**
 * How reachable a machine is, in the three states the chassis draws.
 *
 * This is the presentation vocabulary, not the transport's. `unenrolled` means
 * "linked but never confirmed" and is what a machine that has never answered
 * gets — visually distinct from one that answered and later stopped, because
 * the two need different actions from the operator.
 */
export type MachineState = "dispatchable" | "offline" | "unenrolled";

/** The face rect in device units. The chassis analogue of {@link screenBox}. */
export { screenBox as faceBox } from "./deviceGeometry";
