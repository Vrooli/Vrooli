/** Geometry the engine needs from the world without knowing the sim. */
export interface WorldExtent {
  /** Extent along x (metres). */
  width: number
  /** Extent along z (metres). */
  depth: number
  /** Centre in world space. */
  center: readonly [x: number, z: number]
}

export interface WorldBounds extends WorldExtent {
  /** Extent of the placed layout without the slab margin. */
  footprint: WorldExtent
  /** Ground points on the edge of everything placed; what the camera frames. */
  outline: ReadonlyArray<readonly [x: number, z: number]>
}
