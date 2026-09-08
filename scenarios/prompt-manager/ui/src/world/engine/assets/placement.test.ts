import { describe, expect, it } from 'vitest'
import { propRecord } from './registry'
import { propOrigin } from './placement'

describe('furniture anchors', () => {
  it.each(['desk', 'chair_desk', 'table_round'])('centres the actual %s mesh at its layout anchor at every orientation', id => {
    const record = propRecord('office', id)
    if (!record) throw new Error('Missing test asset')
    for (const rotation of [0, Math.PI / 2, Math.PI, Math.PI / 4]) {
      for (const scale of [.8, 1.7, 2.2]) {
        const origin = propOrigin(record, [3, 2, 4], rotation, scale)
        const x = (record.bounds.min[0] + record.bounds.max[0]) * scale / 2
        const z = (record.bounds.min[2] + record.bounds.max[2]) * scale / 2
        expect(origin[0] + x * Math.cos(rotation) + z * Math.sin(rotation)).toBeCloseTo(3)
        expect(origin[2] - x * Math.sin(rotation) + z * Math.cos(rotation)).toBeCloseTo(4)
        expect(origin[1] + record.bounds.min[1] * scale).toBeCloseTo(2)
      }
    }
  })
})
