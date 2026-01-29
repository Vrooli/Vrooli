/**
 * GroundMaterial - Applies textured ground materials with optional triplanar projection.
 */

import { useEffect, useMemo } from 'react'
import * as THREE from 'three'
import { useThree } from '@react-three/fiber'
import type { GroundMaterialConfig } from '@/types/environment'
import { getGroundTextureSet } from '@/lib/groundTextures'

interface GroundMaterialProps {
  material: GroundMaterialConfig
  groundSize: number
  fallbackColor: string
}

const DEFAULT_TILE_SIZE = 4

const setTextureRepeat = (texture: THREE.Texture, repeat: number, rotation: number) => {
  texture.repeat.set(repeat, repeat)
  texture.rotation = rotation
  texture.center.set(0.5, 0.5)
  texture.needsUpdate = true
}

export function GroundMaterial({ material, groundSize, fallbackColor }: GroundMaterialProps) {
  const { gl } = useThree()
  const maxAnisotropy = useMemo(
    () => Math.min(8, gl.capabilities.getMaxAnisotropy()),
    [gl]
  )

  const textureConfig = material.type === 'texture' ? material.texture : undefined
  const textureId = textureConfig?.id
  const textureSet = useMemo(
    () => (textureId ? getGroundTextureSet(textureId) : null),
    [textureId]
  )

  const tileSize = Math.max(textureConfig?.tileSize ?? DEFAULT_TILE_SIZE, 0.1)
  const baseUvRepeat = groundSize / tileSize
  const rotation = textureConfig?.rotation ?? 0
  const normalScale = textureConfig?.normalScale ?? 0.6
  const roughnessIntensity = textureConfig?.roughnessIntensity ?? 1
  const aoIntensity = textureConfig?.aoIntensity ?? 1

  useEffect(() => {
    if (!textureSet) {
      return
    }

    const baseTextures = [textureSet.albedo, textureSet.normal, textureSet.roughness, textureSet.ao]
    baseTextures.forEach((texture) => {
      texture.anisotropy = maxAnisotropy
      setTextureRepeat(texture, baseUvRepeat, rotation)
    })

  }, [textureSet, baseUvRepeat, rotation, maxAnisotropy])

  const normalScaleVector = useMemo(
    () => new THREE.Vector2(normalScale, normalScale),
    [normalScale]
  )

  if (!textureSet || material.type !== 'texture') {
    return <meshStandardMaterial color={material.color ?? fallbackColor} />
  }

  const tintColor = material.color ?? fallbackColor

  return (
    <meshStandardMaterial
      color={tintColor}
      map={textureSet.albedo}
      normalMap={textureSet.normal}
      roughnessMap={textureSet.roughness}
      aoMap={textureSet.ao}
      roughness={roughnessIntensity}
      metalness={0}
      aoMapIntensity={aoIntensity}
      normalScale={normalScaleVector}
      envMapIntensity={0.7}
    />
  )
}
