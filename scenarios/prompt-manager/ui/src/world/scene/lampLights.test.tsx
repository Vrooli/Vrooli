import { campfireState } from '../config/architecture'
import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { PerspectiveCamera } from 'three'
import { scenes, resolvePeriod, tuning } from '../config'
import { LampLights } from './LampLights'
import { LampLightPool, lampPoolSize } from './lampPool'

const placements = [
  { position: [0, 0] as const, y: 2, scale: 1 },
  { position: [10, 0] as const, y: 2, scale: 1 },
  { position: [-10, 0] as const, y: 2, scale: 1 },
]
const settings = { color: '#ffe4bc', intensity: 8, distance: 14, height: 1.8 }

describe('pooled lamp lights', () => {
  it('mounts no lights in daylight, low profile, or an undeclared emissive role', () => {
    for (const scene of Object.values(scenes)) {
      const day = resolvePeriod(scene, 'day')
      expect(lampPoolSize(tuning.quality.profiles.high, day, scene.emissive?.lamp)).toBe(0)
      expect(renderToStaticMarkup(<LampLights placements={placements} scene={scene} period={day} lighting={tuning.lighting} profile={tuning.quality.profiles.high} camera={tuning.camera} />)).toBe('')
      expect(lampPoolSize(tuning.quality.profiles.low, resolvePeriod(scene, 'night'), scene.emissive?.lamp)).toBe(0)
      expect(lampPoolSize(tuning.quality.profiles.high, resolvePeriod(scene, 'night'))).toBe(0)
    }
  })
  it('selects nearest lamps, breaks ties by placement order, and disables unused slots', () => {
    const camera = new PerspectiveCamera()
    camera.position.set(0, 5, 5)
    camera.updateMatrixWorld()
    const pool = new LampLightPool(2)
    pool.update(camera, placements, settings, 0.05, 0.002)
    expect(pool.lights.map(light => light.position.x)).toEqual([0, 10])
    expect(pool.lights[0]?.position.y).toBe(3.8)
    const larger = new LampLightPool(5)
    larger.update(camera, placements, settings, 0.05, 0.002)
    expect(larger.lights.map(light => light.intensity)).toEqual([8, 8, 8, 0, 0])
  })
  it('retains the exact pool and light identities across 100 camera moves, then skips at rest', () => {
    const camera = new PerspectiveCamera()
    const pool = new LampLightPool(tuning.quality.profiles.high.lampLights)
    const original = [...pool.lights]
    for (let frame = 0; frame < 100; frame += 1) {
      camera.position.x = frame
      camera.updateMatrixWorld()
      expect(pool.update(camera, placements, settings, 0.05, 0.002)).toBe(true)
      pool.lights.forEach((light, index) => expect(light).toBe(original[index]))
      expect(pool.lights.length).toBe(tuning.quality.profiles.high.lampLights)
    }
    for (let frame = 0; frame < 100; frame += 1) expect(pool.update(camera, placements, settings, 0.05, 0.002)).toBe(false)
    expect(pool.runs).toBe(100)
    expect(pool.skips).toBe(100)
  })
  it('refreshes light settings without camera movement or pool replacement', () => {
    const camera = new PerspectiveCamera()
    const pool = new LampLightPool(1)
    pool.update(camera, placements, settings, 0.05, 0.002)
    const light = pool.lights[0]
    pool.update(camera, placements, { ...settings, intensity: 12, distance: 20 }, 0.05, 0.002)
    expect(pool.lights[0]).toBe(light)
    expect(light?.intensity).toBe(12)
    expect(light?.distance).toBe(20)
    pool.update(camera, [], settings, 0.05, 0.002)
    expect(light?.intensity).toBe(0)
  })
  it('illuminates the nearest fire at flame height and retains the same budget when moving to lamps', () => {
    const camera = new PerspectiveCamera()
    const pool = new LampLightPool(1)
    const emitters = [{ position: [10, 0] as const, y: 2, scale: 1 }, { position: [0, 0] as const, y: 2, scale: 1,
      light: { color: '#ff7c36', height: .65, intensityScale: 2 } }]
    pool.update(camera, emitters, settings, .05, .002)
    const light = pool.lights[0]
    expect(light?.position.toArray()).toEqual([0, 2.65, 0])
    expect(light?.color.getHexString()).toBe('ff7c36')
    expect(light?.intensity).toBe(16)
    camera.position.x = 10
    camera.updateMatrixWorld()
    pool.update(camera, emitters, settings, .05, .002)
    expect(pool.lights).toHaveLength(1)
    expect(pool.lights[0]).toBe(light)
    expect(light?.position.toArray()).toEqual([10, 3.8, 0])
    expect(light?.color.getHexString()).toBe('ffe4bc')
    expect(light?.intensity).toBe(8)
  })
})

it('fades flames to embers before extinguishing and puts out exposed fires in wet weather', () => {
  const burning = campfireState(4, 0, false)
  expect(burning.flame).toBe(1)
  expect(burning.embers).toBe(1)
  const dying = campfireState(.8, .8, false)
  expect(dying.flame).toBe(0)
  expect(dying.embers).toBeGreaterThan(0)
  expect(dying.light).toBeGreaterThan(0)
  expect(campfireState(0, 1, false)).toEqual({ light: 0, flame: 0, embers: 0 })
  expect(campfireState(4, 0, true)).toEqual({ light: 0, flame: 0, embers: 0 })
  expect(campfireState(0, 0, false).flame).toBe(0)
})
