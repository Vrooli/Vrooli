import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { interiorDesks, interiorFor, interiorTablePosition } from './interior'

describe('interiorFor', () => {
  it('places decorations against room edges or beneath a table, with aligned rotations and clear workstation access', () => {
    for (const id of ['alpha', 'beta', 'gamma', 'delta']) {
      const choice = interiorFor(7, id, 6, [12, 12], tuning.layout, 4)
      const desks = interiorDesks(choice, 6, [12, 12], tuning.layout)
      const table = interiorTablePosition(choice, [12, 12], tuning.layout)
      expect(choice.fillers.length).toBeGreaterThan(1)
      for (const item of choice.fillers) {
        expect(item.rotation / (Math.PI / 2)).toBeCloseTo(Math.round(item.rotation / (Math.PI / 2)))
        if (item.propIndex === 1) { expect(item.local).toEqual(table); continue }
        expect(Math.max(Math.abs(item.local[0]), Math.abs(item.local[1]))).toBeGreaterThan(4.5)
        for (const desk of desks) {
          expect(Math.hypot(item.local[0] - desk.position[0], item.local[1] - desk.position[1])).toBeGreaterThan(1.2)
          expect(Math.hypot(item.local[0] - desk.seat[0], item.local[1] - desk.seat[1])).toBeGreaterThan(1.2)
        }
      }
    }
  })
  it('is stable by seed and team identity', () => {
    expect(interiorFor(7, 'alpha', 5, [8, 6], tuning.layout)).toEqual(interiorFor(7, 'alpha', 5, [8, 6], tuning.layout))
  })

  it('varies across team identities', () => {
    const records = Array.from({ length: 4 }, (_, index) => JSON.stringify(interiorFor(7, `team-${index}`, 5, [8, 6], tuning.layout)))
    expect(new Set(records).size).toBeGreaterThan(1)
  })

  it('turns the seeded choices into distinct desk, table, and lamp layouts', () => {
    const records = Array.from({ length: 4 * 5 }, (_, index) => interiorFor(7, `team-${index}`, 5, [8, 6], tuning.layout, 4))
    expect(new Set(records.map((record) => record.deskWall)).size).toBeGreaterThan(1)
    expect(new Set(records.map((record) => record.table)).size).toBeGreaterThan(1)
    expect(records.every((record) => record.lampCorners[0] !== record.lampCorners[1])).toBe(true)
    expect(new Set(records.map((record) => JSON.stringify(interiorDesks(record, 5, [8, 6], tuning.layout)))).size).toBeGreaterThan(1)
    expect(new Set(records.map((record) => JSON.stringify(interiorTablePosition(record, [8, 6], tuning.layout)))).size).toBeGreaterThan(1)
    expect(new Set(records.flatMap((record) => record.fillers.map((filler) => filler.propIndex))).size).toBeGreaterThan(1)
  })

  it('changes only the desk grid when member count changes above the table threshold', () => {
    const five = interiorFor(7, 'alpha', 5, [12, 12], tuning.layout, 4)
    const ten = interiorFor(7, 'alpha', 5 + 5, [12, 12], tuning.layout, 4)
    expect(ten.columns).not.toBe(five.columns)
    expect({ ...ten, columns: five.columns }).toEqual(five)
  })
})
