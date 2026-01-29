/**
 * GroundMaterial - Applies textured ground materials with triplanar projection
 * and macro variation overlay to reduce visible tiling artifacts.
 */

import { useEffect, useMemo, useRef } from 'react'
import * as THREE from 'three'
import { useThree } from '@react-three/fiber'
import type { GroundMaterialConfig } from '@/types/environment'
import { getGroundTextureSet } from '@/lib/groundTextures'
import { bindGroundShader, syncGroundShader, type GroundShaderConfig } from '@/lib/shaders/groundShader'

interface GroundMaterialProps {
  material: GroundMaterialConfig
  groundSize: number
  fallbackColor: string
}

const DEFAULT_TILE_SIZE = 4
const DEFAULT_MACRO_SCALE = 20
const DEFAULT_MACRO_INTENSITY = 0.15
const DEFAULT_TRIPLANAR_SHARPNESS = 4

export function GroundMaterial({ material, groundSize, fallbackColor }: GroundMaterialProps) {
  const { gl } = useThree()
  const materialRef = useRef<THREE.MeshStandardMaterial | null>(null)
  const shaderBoundRef = useRef(false)

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
  const tintColor = material.color ?? fallbackColor

  const normalScaleVector = useMemo(
    () => new THREE.Vector2(normalScale, normalScale),
    [normalScale]
  )

  // Build shader config for triplanar projection and macro variation
  const shaderConfig = useMemo((): GroundShaderConfig | null => {
    if (!textureSet || !textureConfig) return null

    const macroConfig = textureConfig.macroVariation ?? {
      enabled: true,
      scale: DEFAULT_MACRO_SCALE,
      intensity: DEFAULT_MACRO_INTENSITY,
    }

    const macroScale = macroConfig.scale || DEFAULT_MACRO_SCALE

    return {
      projection: textureConfig.projection ?? 'uv',
      rotation,
      baseUvRepeat,
      baseWorldScale: 1 / tileSize,
      macroUvRepeat: groundSize / macroScale,
      macroWorldScale: 1 / macroScale,
      macroIntensity: macroConfig.enabled ? macroConfig.intensity : 0,
      macroMap: textureSet.macro,
      triplanarSharpness: DEFAULT_TRIPLANAR_SHARPNESS,
      stochasticEnabled: textureConfig.stochasticEnabled ?? true,
    }
  }, [textureSet, textureConfig, rotation, baseUvRepeat, tileSize, groundSize])

  // Create material imperatively to support onBeforeCompile shader injection
  const mat = useMemo(() => {
    if (!textureSet || material.type !== 'texture') {
      return new THREE.MeshStandardMaterial({ color: tintColor })
    }

    const newMat = new THREE.MeshStandardMaterial({
      color: tintColor,
      map: textureSet.albedo,
      normalMap: textureSet.normal,
      roughnessMap: textureSet.roughness,
      aoMap: textureSet.ao,
      roughness: roughnessIntensity,
      metalness: 0,
      aoMapIntensity: aoIntensity,
      normalScale: normalScaleVector,
      envMapIntensity: 0.7,
    })

    return newMat
  }, [textureSet, material.type, tintColor, roughnessIntensity, aoIntensity, normalScaleVector])

  // Configure texture repeat and anisotropy
  useEffect(() => {
    if (!textureSet) return

    const baseTextures = [textureSet.albedo, textureSet.normal, textureSet.roughness, textureSet.ao]
    baseTextures.forEach((texture) => {
      texture.wrapS = THREE.RepeatWrapping
      texture.wrapT = THREE.RepeatWrapping
      texture.anisotropy = maxAnisotropy
      texture.repeat.set(baseUvRepeat, baseUvRepeat)
      texture.rotation = rotation
      texture.center.set(0.5, 0.5)
      texture.needsUpdate = true
    })

    // Also configure macro texture
    textureSet.macro.wrapS = THREE.RepeatWrapping
    textureSet.macro.wrapT = THREE.RepeatWrapping
    textureSet.macro.needsUpdate = true
  }, [textureSet, baseUvRepeat, rotation, maxAnisotropy])

  // Bind shader on material creation (once)
  useEffect(() => {
    if (!shaderConfig || shaderBoundRef.current) return
    bindGroundShader(mat, shaderConfig)
    shaderBoundRef.current = true
    materialRef.current = mat
  }, [mat, shaderConfig])

  // Sync uniforms when config changes
  useEffect(() => {
    if (!materialRef.current || !shaderConfig) return
    syncGroundShader(materialRef.current, shaderConfig)
  }, [shaderConfig])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      mat.dispose()
      shaderBoundRef.current = false
    }
  }, [mat])

  return <primitive object={mat} attach="material" />
}
