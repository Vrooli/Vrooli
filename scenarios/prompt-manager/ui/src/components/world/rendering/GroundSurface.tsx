/**
 * GroundSurface - Renders the ground plane or grid with textured materials.
 */

import { useEffect, useMemo } from 'react'
import * as THREE from 'three'
import type { GroundConfig, GroundMaterialConfig } from '@/types/environment'
import { GroundMaterial } from './GroundMaterial'

interface GroundSurfaceProps {
  ground: GroundConfig
  groundY: number
  isDarkMode: boolean
}

const DEFAULT_GROUND_SIZE = 100

export function GroundSurface({ ground, groundY, isDarkMode }: GroundSurfaceProps) {
  const isVisible = ground.visible && ground.type !== 'none'
  const size = ground.size ?? DEFAULT_GROUND_SIZE
  const groundColor = ground.color ?? (isDarkMode ? '#1e293b' : '#e2e8f0')
  const materialConfig: GroundMaterialConfig = ground.material ?? {
    type: 'solid',
    color: groundColor,
  }

  const planeRotation = useMemo(() => new THREE.Euler(-Math.PI / 2, 0, 0), [])
  const planePosition = useMemo(() => [0, groundY, 0] as const, [groundY])

  const gridArgs = useMemo(
    () => ([
      ground.size ?? 30,
      ground.divisions ?? 30,
      ground.color ?? (isDarkMode ? '#1e293b' : '#e2e8f0'),
      ground.color ?? (isDarkMode ? '#1e293b' : '#e2e8f0'),
    ] as [number, number, string, string]),
    [ground.size, ground.divisions, ground.color, isDarkMode]
  )

  const planeGeometry = useMemo(() => {
    if (ground.type !== 'plane') {
      return null
    }

    const geometry = new THREE.PlaneGeometry(size, size)
    const uvAttribute = geometry.getAttribute('uv') as THREE.BufferAttribute
    geometry.setAttribute('uv2', uvAttribute.clone())
    return geometry
  }, [ground.type, size])

  useEffect(() => {
    if (!planeGeometry) {
      return
    }
    return () => planeGeometry.dispose()
  }, [planeGeometry])

  if (!isVisible) {
    return null
  }

  if (ground.type === 'grid') {
    return (
      <gridHelper
        args={gridArgs}
        position={planePosition}
      />
    )
  }

  if (!planeGeometry) {
    return null
  }

  return (
    <mesh
      rotation={planeRotation}
      position={planePosition}
      receiveShadow
    >
      <primitive object={planeGeometry} attach="geometry" />
      <GroundMaterial
        material={materialConfig}
        groundSize={size}
        fallbackColor={groundColor}
      />
    </mesh>
  )
}
