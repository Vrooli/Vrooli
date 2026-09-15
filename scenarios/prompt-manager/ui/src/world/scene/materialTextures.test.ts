import { describe, expect, it } from 'vitest'
import { MeshStandardMaterial, RGBAFormat, SRGBColorSpace } from 'three'
import { applyWorldTexture, worldTexture, worldTextureKind } from './materialTextures'

describe('world material texture library', () => {
  it('provides cached albedo, roughness, and micro-normal channels for surface families', () => {
    for (const kind of ['wood', 'bark', 'canvas', 'fabric', 'metal', 'leaf', 'stone', 'floor', 'path'] as const) {
      const set = worldTexture(kind)
      expect(set.map.image.width).toBe(96)
      expect(set.map.image.height).toBe(96)
      expect(set.map.colorSpace).toBe(SRGBColorSpace)
      expect(set.roughnessMap?.image.width).toBe(96)
      expect(set.roughnessMap?.format).toBe(RGBAFormat)
      expect(set.envMapIntensity).toBe(0)
      if (['wood', 'bark', 'fabric', 'stone'].includes(kind)) expect(set.normalMap?.image.width).toBe(96)
      expect(worldTexture(kind)).toBe(set)
    }
  })

  it('classifies source-kit material names by physical surface', () => {
    expect(worldTextureKind('woodBark')).toBe('bark')
    expect(worldTextureKind('leafsGreen')).toBe('leaf')
    expect(worldTextureKind('metalMedium')).toBe('metal')
    expect(worldTextureKind('carpetDarker')).toBe('fabric')
    expect(worldTextureKind('wallPaint')).toBe('wall')
  })

  it('keeps source vegetation variation while shifting cyan kit leaves into green', () => {
    const material = new MeshStandardMaterial({ color: '#70e6d6' })
    applyWorldTexture(material, 'leaf')
    const hsl = { h: 0, s: 0, l: 0 }
    material.color.getHSL(hsl)
    expect(hsl.h).toBeGreaterThan(.2)
    expect(hsl.h).toBeLessThan(.4)
    expect(material.envMapIntensity).toBe(0)
    material.dispose()
  })

  it('desaturates blue-grey kit metals before applying controlled reflections', () => {
    const material = new MeshStandardMaterial({ color: '#bdcfd6' })
    applyWorldTexture(material, 'metal')
    const hsl = { h: 0, s: 0, l: 0 }
    material.color.getHSL(hsl)
    expect(hsl.s).toBeLessThan(.14)
    expect(material.envMapIntensity).toBe(0)
    material.dispose()
  })
})
