import type { PropRecord } from './registry'

/** Measured kit-facing axes: chairs face -Z; a log seat faces across its long Z axis.
 * The layout facing belongs to the sitter, independent of the mesh's modelling axis.
 */
export function seatRotation(record: Pick<PropRecord, 'scene' | 'id'>, facing: number): number {
  const modelFacing = record.scene === 'park' && record.id === 'log_seat' ? Math.PI / 2 : Math.PI
  return facing - modelFacing
}

/** Layout anchors name the footprint centre at ground height, independent of a kit's modelling origin. */
export function propOrigin(record: Pick<PropRecord, 'bounds'>, ground: readonly [number, number, number], rotation: number, scale: number): [number, number, number] {
  const x = (record.bounds.min[0] + record.bounds.max[0]) * scale / 2
  const z = (record.bounds.min[2] + record.bounds.max[2]) * scale / 2
  return [ground[0] - x * Math.cos(rotation) - z * Math.sin(rotation), ground[1] - record.bounds.min[1] * scale,
    ground[2] + x * Math.sin(rotation) - z * Math.cos(rotation)]
}
