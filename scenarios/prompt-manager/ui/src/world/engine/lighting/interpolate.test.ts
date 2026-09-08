import { describe, expect, it } from 'vitest'
import { Color } from 'three'
import { scenes, tuning, resolvePeriod } from '../../config'
import { continuousPeriod } from './interpolate'

describe('continuous civil-time lighting', () => {
  it('matches every scene-resolved preset at its band centre', () => {
    for (const scene of Object.values(scenes)) {
      for (const [id, minutes] of [['night', 30], ['dawn', 390], ['day', 750], ['dusk', 1110]] as const) {
        expect(continuousPeriod(scene, minutes, tuning)).toEqual(resolvePeriod(scene, id, tuning))
      }
    }
  })
  it('blends numeric values and colors in linear light at the midpoint', () => {
    const a = resolvePeriod(scenes.park, 'dawn', tuning), b = resolvePeriod(scenes.park, 'day', tuning)
    const result = continuousPeriod(scenes.park, 570, tuning)
    expect(result.exposure).toBeCloseTo((a.exposure + b.exposure) / 2)
    expect(result.keyIntensity).toBeCloseTo((a.keyIntensity + b.keyIntensity) / 2)
    expect(result.keyColor).toBe(`#${new Color(a.keyColor).lerp(new Color(b.keyColor), .5).getHexString()}`)
  })
  it('is continuous at midnight, anchors and former period boundaries', () => {
    for (const minute of [0, 30, 300, 390, 480, 750, 1020, 1110, 1200, 1440]) {
      const before = continuousPeriod(scenes.park, minute - .001, tuning)
      const after = continuousPeriod(scenes.park, minute + .001, tuning)
      expect(Math.abs(before.exposure - after.exposure)).toBeLessThan(.0001)
      expect(Math.abs(before.sunElevationDeg - after.sunElevationDeg)).toBeLessThan(.001)
      const left = new Color(before.keyColor), right = new Color(after.keyColor)
      expect(Math.hypot(left.r - right.r, left.g - right.g, left.b - right.b)).toBeLessThan(.015)
    }
    expect(continuousPeriod(scenes.park, 0, tuning)).toEqual(continuousPeriod(scenes.park, 1440, tuning))
  })
})
