import { describe, expect, it } from 'vitest'
import { Color, CubeTexture, Scene } from 'three'
import { applyPeriodBackground } from './background'

describe('period background ownership', () => {
  it('updates the outdoor fallback without replacing the reflection environment', () => {
    const scene = new Scene()
    const environment = new CubeTexture()
    const renderer = { toneMappingExposure: 1 }
    scene.environment = environment
    scene.background = environment
    for (const period of [
      { backgroundColor: '#abcdef', exposure: 0.8 },
      { backgroundColor: '#123456', exposure: 0.4 },
    ]) {
      applyPeriodBackground(scene, renderer, true, period)
      expect(scene.background).toEqual(new Color(period.backgroundColor))
      expect(scene.environment).toBe(environment)
      expect(renderer.toneMappingExposure).toBe(period.exposure)
    }
  })

  it('sets an indoor color and exposure, including a transition from outdoors', () => {
    const scene = new Scene()
    scene.background = new CubeTexture()
    const renderer = { toneMappingExposure: 1 }
    const period = { backgroundColor: '#123456', exposure: 0.4 }
    applyPeriodBackground(scene, renderer, false, period)
    expect(scene.background).toEqual(new Color(period.backgroundColor))
    expect(renderer.toneMappingExposure).toBe(period.exposure)
  })
})
