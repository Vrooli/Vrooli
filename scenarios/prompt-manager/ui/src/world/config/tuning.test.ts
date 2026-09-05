import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { parseTuning, tuning, withTuningOverride } from './tuning'
import { collectCompositionLevers, collectLevers, extractTuningDocs, renderTuningDocs, undocumentedLevers } from './tuningDocs'
import { scenes, resolvePeriod } from './scenes'
import { periodForHour } from './periods'
import { PERIOD_IDS, QUALITY_PROFILE_IDS, SCENE_IDS } from './tuning.schema'
import raw from './world.tuning.json'
import weatherCatalog from './weather.json'

const CONFIG_DOC = resolve(import.meta.dirname, '../../../../docs/reference/configuration.md')

describe('world.tuning.json', () => {
  it('documents scene composition and vegetation metadata from their owning schemas', () => {
    const rows = collectCompositionLevers()
    expect(undocumentedLevers(rows)).toEqual([])
    expect(rows.find((r) => r.path === 'scenes.office.centre.blend')).toMatchObject({ value: '4', description: expect.stringContaining('metres') })
    expect(rows.find((r) => r.path === 'scenes.park.centre.blend')?.value).toBe('—')
    expect(rows.find((r) => r.path === 'scenes.office.emissive')?.description).toContain('hex colours')
    expect(rows.find((r) => r.path === 'vegetationEntry.density')?.description).toContain('instances per square metre')
    expect(rows.find((r) => r.path === 'vegetationEntry.class')).toBeDefined()
    expect(rows.find((r) => r.path === 'vegetationEntry.scaleRef')).toBeDefined()
  })
  it('keeps the weather catalogue aligned with authoritative runtime defaults', () => {
    expect(weatherCatalog).toEqual(raw.weather)
  })
  it('parses against the schema', () => {
    expect(parseTuning(raw)).toEqual(tuning)
  })

  it('preserves label appearance defaults and validates visual overrides', () => {
    expect(tuning.labels).toMatchObject({ color: '#ffffff', strokeColor: '#101423', strokePercent: 10, charWidthFactor: 0.58, refreshEveryFrames: 3, basePxPerUnit: 40, pinnedBonus: 10, syncSizeEpsilon: 0.0001, renderOrder: 10 })
    expect(tuning.labels.priorities).toEqual({ failed: 5, working: 4, walkingToDesk: 4, gathered: 3, walkingToTable: 3, socializing: 2, idle: 1 })
    const next = withTuningOverride({ labels: { strokePercent: 20, refreshEveryFrames: 1, priorities: { failed: 9 } } })
    expect(next.labels.strokePercent).toBe(20)
    expect(next.labels.refreshEveryFrames).toBe(1)
    expect(next.labels.priorities.failed).toBe(9)
    expect(next.labels.priorities.idle).toBe(1)
    expect(() => withTuningOverride({ labels: { refreshEveryFrames: 0 } })).toThrow(/labels.refreshEveryFrames/)
    expect(() => withTuningOverride({ actor: { extras: { tierSizes: [1, 2] } } })).toThrow(/actor.extras.tierSizes/)
  })

  it('preserves editor and lighting rig defaults and accepts bounded overrides', () => {
    expect(tuning.editor).toMatchObject({ handleLift: 0.05, handleOpacity: 0.28, selectedOpacity: 0.45, handleColor: '#ffffff', selectedColor: '#7c9cff' })
    expect(tuning.lighting.rig).toEqual({ environmentResolution: 256, sunDistance: 100, shadowExtentScale: 0.62, shadowExtentPadding: 6, hemisphereHeight: 50, keyPanel: { intensity: 1.2, position: [8, 6, -8], scale: [8, 4, 1] }, fillPanel: { intensity: 0.6, position: [-9, 4, 4], scale: [6, 3, 1] }, topPanel: { intensity: 0.4, position: [0, 10, 0], scale: [6, 6, 6] }, topPanelColor: '#ffffff' })
    const next = withTuningOverride({ lighting: { rig: { sunDistance: 150, keyPanel: { intensity: 2 } } }, editor: { handleOpacity: 0.6 } })
    expect(next.lighting.rig.sunDistance).toBe(150)
    expect(next.lighting.rig.keyPanel.intensity).toBe(2)
    expect(next.editor.handleOpacity).toBe(0.6)
    expect(() => withTuningOverride({ weather: { lightingLimits: { fogFarMin: 30, fogFarMax: 20 } } })).toThrow(/fogFarMin/)
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
    for (const id of QUALITY_PROFILE_IDS) {
      expect(tuning.quality.profiles[id].ao).toBe(tuning.quality.profiles[id].aoQuality !== 'off')
    }
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
    expect(scenes.park.biomeSet).toBe('park')
    expect(scenes.office.biomeSet).toBe('park')
    expect(scenes.office.terrain).toBeUndefined()
    expect(scenes.office.centre?.biomeSet).toBe('office')
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
