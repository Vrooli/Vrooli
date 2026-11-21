// [REQ:P0-002e] Device archetypes for follower presentation.
//
// `navigator.userAgent` is not a trustworthy device identity and deliberately
// never participates here. Neither does the terminal grid, for a different
// reason: it is not *stable*. A leader's grid shrinks whenever its virtual
// keyboard opens, so classifying from grid geometry made a phone reclassify
// itself as a laptop — silhouette, stand and all — the moment somebody typed.
//
// The leader therefore declares its own class from `screen`, and grid geometry
// survives only as the fallback for a leader that declares nothing.

export const DEVICE_ARCHETYPES = ["phone", "tablet", "laptop", "monitor", "ultrawide"] as const;

export type DeviceArchetype = (typeof DEVICE_ARCHETYPES)[number];

export function isDeviceArchetype(value: string): value is DeviceArchetype {
  return (DEVICE_ARCHETYPES as readonly string[]).includes(value);
}

export function aspectForGrid(cols: number, rows: number, cellAspect: number): number {
  return (cols * cellAspect) / rows;
}

/**
 * Fallback classification for a leader that declares no device class.
 *
 * Callers must prefer {@link resolveArchetype}: this reads the live grid, and
 * the live grid moves for reasons that have nothing to do with the hardware.
 */
export function archetypeForGrid(cols: number, rows: number, cellAspect: number): DeviceArchetype {
  const aspect = aspectForGrid(cols, rows, cellAspect);
  if (aspect < 1.1) return "phone";
  if (aspect < 1.6 && cols <= 110) return "tablet";
  if (aspect < 2.1) return "laptop";
  if (aspect < 3) return "monitor";
  return "ultrawide";
}

/**
 * Choose the archetype a follower frames the session with.
 *
 * The declared class wins whenever the leader sent one, which is what keeps
 * the silhouette stable across a keyboard opening, a font-size change, or any
 * other resize. Grid geometry is consulted only when nothing was declared.
 */
export function resolveArchetype(options: {
  declaredClass?: string;
  cols: number;
  rows: number;
  cellAspect: number;
}): DeviceArchetype {
  const declared = options.declaredClass;
  if (declared && isDeviceArchetype(declared)) return declared;
  return archetypeForGrid(options.cols, options.rows, options.cellAspect);
}
