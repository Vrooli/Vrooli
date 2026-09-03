import { describe, expect, it } from 'vitest'
import { scenes, tuning } from '../../config'
import { allProps, propRecord } from './registry'

/** Each material is one instanced draw per prop; keep props cheap. */
const MAX_PARTS = 3

describe('prop registry', () => {
  it('has a baked entry for every prop each scene names, within the triangle budget', () => {
    for (const scene of Object.values(scenes)) {
      const p = scene.props
      const ids = [p.desk, p.chair, p.table, p.seat, p.campfire, p.lamp, p.board, ...p.trees, ...p.decor]
      for (const id of ids) {
        const record = propRecord(scene.assetSet, id)
        expect(record, `${scene.id}/${id} missing from registry; run pnpm world:assets`).toBeDefined()
        expect(record?.triangles ?? Infinity).toBeLessThanOrEqual(tuning.budgets.propTriangles)
        expect(record?.materials ?? 0).toBeLessThanOrEqual(MAX_PARTS)
      }
    }
  })

  it('every record carries real bounds', () => {
    for (const record of allProps()) {
      expect(record.size.every((v) => v > 0 && v < 10)).toBe(true)
      expect(record.bytes).toBeGreaterThan(0)
    }
  })
})
