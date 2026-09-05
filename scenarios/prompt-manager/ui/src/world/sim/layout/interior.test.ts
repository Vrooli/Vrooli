import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { interiorDesks, interiorFor, interiorTablePosition } from './interior'

describe('interiorFor', () => {
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
