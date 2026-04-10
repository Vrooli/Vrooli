/**
 * HeldItemAccessory - Visual items held by members.
 * Renders procedural geometry for books, tools, etc.
 */

import { useMemo } from 'react'
import type { HeldItemProps } from './types'
import { getDefaultOffset } from './types'
import { useMaterial } from '../materials'

/**
 * Renders a held item based on type.
 * Uses procedural geometry until GLTF models are available.
 */
export function HeldItemAccessory({
  type,
  hand = 'right',
  position,
  rotation,
  scale = 1,
  color = '#6b4423',
  castShadow = true,
}: HeldItemProps) {
  const offsetKey = hand === 'left' ? 'leftHand' : 'rightHand'
  const offset = getDefaultOffset(offsetKey)
  const material = useMaterial('matte', { color })

  const pos = useMemo<[number, number, number]>(() => {
    const [ox, oy, oz] = offset.position
    return position
      ? [position[0] + ox, position[1] + oy, position[2] + oz]
      : [ox, oy, oz]
  }, [position, offset.position])

  const rot = useMemo<[number, number, number]>(() => {
    const [rx, ry, rz] = offset.rotation
    return rotation
      ? [rotation[0] + rx, rotation[1] + ry, rotation[2] + rz]
      : [rx, ry, rz]
  }, [rotation, offset.rotation])

  const finalScale = scale * offset.scale

  if (type === 'none') {
    return null
  }

  switch (type) {
    case 'book':
      return (
        <group position={pos} rotation={rot} scale={finalScale}>
          {/* Book cover */}
          <mesh castShadow={castShadow} material={material}>
            <boxGeometry args={[0.08, 0.1, 0.02]} />
          </mesh>
          {/* Pages */}
          <mesh position={[0.005, 0, 0]}>
            <boxGeometry args={[0.06, 0.09, 0.015]} />
            <meshStandardMaterial color="#f5f5dc" />
          </mesh>
          {/* Spine accent */}
          <mesh position={[-0.042, 0, 0]}>
            <boxGeometry args={[0.005, 0.1, 0.02]} />
            <meshStandardMaterial color="#8b4513" />
          </mesh>
        </group>
      )

    case 'tool':
      return (
        <group position={pos} rotation={rot} scale={finalScale}>
          {/* Wrench/tool handle */}
          <mesh rotation={[0, 0, Math.PI / 4]} castShadow={castShadow}>
            <cylinderGeometry args={[0.012, 0.012, 0.15, 8]} />
            <meshStandardMaterial color="#444444" metalness={0.8} roughness={0.3} />
          </mesh>
          {/* Tool head */}
          <mesh position={[0, 0.08, 0]} rotation={[0, 0, Math.PI / 4]}>
            <boxGeometry args={[0.06, 0.02, 0.01]} />
            <meshStandardMaterial color="#888888" metalness={0.9} roughness={0.2} />
          </mesh>
          {/* Tool opening */}
          <mesh position={[0.03, 0.08, 0]} rotation={[0, 0, Math.PI / 4]}>
            <boxGeometry args={[0.015, 0.025, 0.012]} />
            <meshStandardMaterial color="#333333" />
          </mesh>
        </group>
      )

    case 'orb':
      return (
        <group position={pos} rotation={rot} scale={finalScale}>
          {/* Glowing orb */}
          <mesh castShadow={castShadow}>
            <sphereGeometry args={[0.04, 16, 16]} />
            <meshStandardMaterial
              color={color}
              emissive={color}
              emissiveIntensity={0.5}
              transparent
              opacity={0.8}
              toneMapped={false}
            />
          </mesh>
          {/* Inner glow */}
          <mesh>
            <sphereGeometry args={[0.025, 12, 12]} />
            <meshBasicMaterial
              color="#ffffff"
              transparent
              opacity={0.6}
            />
          </mesh>
        </group>
      )

    case 'wand':
      return (
        <group position={pos} rotation={rot} scale={finalScale}>
          {/* Wand shaft */}
          <mesh rotation={[Math.PI / 6, 0, 0]} castShadow={castShadow}>
            <cylinderGeometry args={[0.008, 0.012, 0.2, 8]} />
            <meshStandardMaterial color="#3d2314" />
          </mesh>
          {/* Wand tip glow */}
          <mesh position={[0, 0.1, 0.03]}>
            <sphereGeometry args={[0.015, 8, 8]} />
            <meshStandardMaterial
              color="#6366f1"
              emissive="#6366f1"
              emissiveIntensity={1.0}
              toneMapped={false}
            />
          </mesh>
          {/* Handle decoration */}
          <mesh position={[0, -0.08, -0.02]}>
            <sphereGeometry args={[0.015, 8, 8]} />
            <meshStandardMaterial color="#c0c0c0" metalness={0.8} roughness={0.2} />
          </mesh>
        </group>
      )

    default:
      return null
  }
}
