import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { parseTuning, tuning, withTuningOverride } from './tuning'
import { collectLevers, extractTuningDocs, renderTuningDocs, undocumentedLevers } from './tuningDocs'
import { scenes, resolvePeriod } from './scenes'
import { periodForHour } from './periods'
import { PERIOD_IDS, QUALITY_PROFILE_IDS, SCENE_IDS } from './tuning.schema'
import raw from './world.tuning.json'

const CONFIG_DOC = resolve(import.meta.dirname, '../../../../docs/reference/configuration.md')

describe('world.tuning.json', () => {
  it('parses against the schema', () => {
    expect(parseTuning(raw)).toEqual(tuning)
  })

  it('rejects an out-of-bounds lever with the lever path in the message', () => {
    const broken = structuredClone(raw) as { sim: { walkSpeed: number } }
    broken.sim.walkSpeed = 999
    expect(() => parseTuning(broken)).toThrow(/sim\.walkSpeed/)
  })

  it('rejects an unknown quality profile reference', () => {
    const broken = structuredClone(raw) as { quality: { defaultProfile: string } }
    broken.quality.defaultProfile = 'insane'
    expect(() => parseTuning(broken)).toThrow(/quality\.defaultProfile/)
  })

  it('keeps camera, governor and period bands internally consistent', () => {
    expect(tuning.camera.polarMinDeg).toBeLessThan(tuning.camera.polarMaxDeg)
    expect(tuning.camera.minDistance).toBeLessThan(tuning.camera.maxDistance)
    expect(tuning.quality.degradedRatio).toBeLessThan(tuning.quality.recoverRatio)
    expect(tuning.labels.minScreenPx).toBeLessThanOrEqual(tuning.labels.maxScreenPx)
    for (const id of QUALITY_PROFILE_IDS) expect(tuning.quality.profiles[id]).toBeDefined()
    for (const id of PERIOD_IDS) expect(tuning.lighting.periods[id]).toBeDefined()
    for (const id of SCENE_IDS) expect(tuning.budgets.scenes[id]).toBeDefined()
  })

  it('every lever is documented in the schema', () => {
    expect(undocumentedLevers()).toEqual([])
  })

  it('docs/reference/configuration.md carries the current lever table', () => {
    const doc = readFileSync(CONFIG_DOC, 'utf8')
    const current = extractTuningDocs(doc)
    expect(current, 'run `pnpm world:tuning-docs`').toBe(renderTuningDocs())
  })

  it('renders one row per leaf lever and groups by top-level key', () => {
    const rows = collectLevers()
    expect(rows.length).toBeGreaterThan(100)
    const groups = new Set(rows.map((r) => r.path.split('.')[0]))
    expect([...groups]).toEqual(Object.keys(raw))
  })

  it('withTuningOverride merges deeply and re-validates', () => {
    const next = withTuningOverride({ sim: { walkSpeed: 3 }, camera: { fov: 40 } })
    expect(next.sim.walkSpeed).toBe(3)
    expect(next.camera.fov).toBe(40)
    expect(next.sim.hurrySpeed).toBe(tuning.sim.hurrySpeed)
    expect(() => withTuningOverride({ sim: { walkSpeed: -1 } })).toThrow(/sim\.walkSpeed/)
  })
})

describe('scenes', () => {
  it('parses both scene files', () => {
    expect(scenes.park.id).toBe('park')
    expect(scenes.office.id).toBe('office')
    expect(scenes.park.props.trees.length).toBeGreaterThan(0)
    expect(scenes.office.props.trees).toEqual([])
  })

  it('applies indoor period overrides over the global preset', () => {
    const night = resolvePeriod(scenes.office, 'night')
    expect(night.lampEmissive).toBe(scenes.office.lighting?.periods?.night?.lampEmissive)
    expect(night.keyColor).toBe(tuning.lighting.periods.night.keyColor)
    expect(resolvePeriod(scenes.park, 'night')).toEqual(tuning.lighting.periods.night)
  })

  it('hero poses satisfy the camera clamps', () => {
    for (const id of SCENE_IDS) {
      const pose = scenes[id].camera.hero
      expect(pose.polarDeg).toBeGreaterThanOrEqual(tuning.camera.polarMinDeg)
      expect(pose.polarDeg).toBeLessThanOrEqual(tuning.camera.polarMaxDeg)
      expect(pose.distanceFactor).toBeGreaterThan(0)
    }
  })
})

describe('periodForHour', () => {
  it('maps every hour to a band and wraps the night band past midnight', () => {
    expect(periodForHour(6, tuning.lighting)).toBe('dawn')
    expect(periodForHour(12, tuning.lighting)).toBe('day')
    expect(periodForHour(18.5, tuning.lighting)).toBe('dusk')
    expect(periodForHour(23, tuning.lighting)).toBe('night')
    expect(periodForHour(2, tuning.lighting)).toBe('night')
    expect(periodForHour(26, tuning.lighting)).toBe('night')
    expect(periodForHour(-1, tuning.lighting)).toBe('night')
  })
})

describe('scene slabs', () => {
  it('keep the bevel within half the thickness so the top face stays at ground level', () => {
    for (const id of SCENE_IDS) {
      expect(scenes[id].slab.cornerRadius).toBeLessThanOrEqual(scenes[id].slab.thickness / 2)
    }
  })
})
