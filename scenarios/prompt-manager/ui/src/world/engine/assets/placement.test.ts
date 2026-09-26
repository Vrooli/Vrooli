import { describe, expect, it } from 'vitest'
import { propRecord } from './registry'
import { propOrigin, seatRotation } from './placement'

describe('furniture anchors', () => {
  it('places the actual long axis of log seats tangent to the gathering circle while chairs face its centre', () => {
    const log = propRecord('park', 'log_seat')
    if (!log) throw new Error('Missing log asset')
    expect(log.size[2]).toBeGreaterThan(log.size[0] * 2)
    for (const facing of [0, .7, Math.PI, -1.8]) {
      const yaw = seatRotation(log, facing)
      expect(Math.sin(yaw) * Math.sin(facing) + Math.cos(yaw) * Math.cos(facing)).toBeCloseTo(0)
      for (const id of ['chair_desk', 'chair_lounge']) {
        const chair = propRecord('office', id)
        if (!chair) throw new Error('Missing chair asset')
        const yaw = seatRotation(chair, facing)
        expect(-Math.sin(yaw)).toBeCloseTo(Math.sin(facing))
        expect(-Math.cos(yaw)).toBeCloseTo(Math.cos(facing))
      }
    }
  })
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
