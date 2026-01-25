/**
 * BackpackAccessory - Visual indicator for skill-based back accessory.
 * Renders procedural geometry based on backpack type.
 */

import { useMemo } from 'react'
import type { BackAccessoryProps } from './types'
import { getDefaultOffset } from './types'
import { useMaterial } from '../materials'

/**
 * Renders a back accessory based on skill count.
 * Uses procedural geometry until GLTF models are available.
 */
export function BackpackAccessory({
  type,
  position,
  rotation,
  scale = 1,
  color = '#8b5a2b',
  castShadow = true,
  skillCount = 0,
}: BackAccessoryProps) {
  const offset = getDefaultOffset('back')
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

  // Render different geometries based on type
  switch (type) {
    case 'paper':
      return (
        <group position={pos} rotation={rot} scale={finalScale}>
          {/* Stack of paper sheets */}
          {[0, 0.01, 0.02].map((y, i) => (
            <mesh key={i} position={[0, y, 0]} castShadow={castShadow} material={material}>
              <boxGeometry args={[0.12, 0.005, 0.15]} />
            </mesh>
          ))}
        </group>
      )

    case 'folder':
      return (
        <group position={pos} rotation={rot} scale={finalScale}>
          {/* Folder shape */}
          <mesh castShadow={castShadow} material={material}>
            <boxGeometry args={[0.15, 0.02, 0.2]} />
          </mesh>
          {/* Tab */}
          <mesh position={[0, 0.015, 0.08]} material={material}>
            <boxGeometry args={[0.06, 0.01, 0.04]} />
          </mesh>
        </group>
      )

    case 'briefcase':
      return (
        <group position={pos} rotation={rot} scale={finalScale}>
          {/* Briefcase body */}
          <mesh castShadow={castShadow} material={material}>
            <boxGeometry args={[0.2, 0.12, 0.06]} />
          </mesh>
          {/* Handle */}
          <mesh position={[0, 0.08, 0]}>
            <torusGeometry args={[0.03, 0.008, 8, 16, Math.PI]} />
            <meshStandardMaterial color="#444444" metalness={0.8} roughness={0.2} />
          </mesh>
          {/* Clasps */}
          {[-0.06, 0.06].map((x, i) => (
            <mesh key={i} position={[x, 0, 0.035]}>
              <boxGeometry args={[0.02, 0.015, 0.008]} />
              <meshStandardMaterial color="#c0c0c0" metalness={0.9} roughness={0.1} />
            </mesh>
          ))}
        </group>
      )

    case 'backpack':
      return (
        <group position={pos} rotation={rot} scale={finalScale}>
          {/* Main body */}
          <mesh castShadow={castShadow} material={material}>
            <boxGeometry args={[0.18, 0.22, 0.1]} />
          </mesh>
          {/* Front pocket */}
          <mesh position={[0, -0.04, 0.055]} material={material}>
            <boxGeometry args={[0.14, 0.1, 0.02]} />
          </mesh>
          {/* Straps (simplified) */}
          {[-0.07, 0.07].map((x, i) => (
            <mesh key={i} position={[x, 0.02, 0.06]} rotation={[0.1, 0, 0]}>
              <boxGeometry args={[0.02, 0.18, 0.01]} />
              <meshStandardMaterial color="#333333" />
            </mesh>
          ))}
          {/* Skill count indicator (small orbs on backpack) */}
          {skillCount > 10 &&
            [...Array(Math.min(skillCount - 10, 5))].map((_, i) => (
              <mesh
                key={i}
                position={[
                  -0.06 + (i % 3) * 0.06,
                  0.08 - Math.floor(i / 3) * 0.06,
                  0.06,
                ]}
              >
                <sphereGeometry args={[0.015, 8, 8]} />
                <meshStandardMaterial
                  color="#6366f1"
                  emissive="#6366f1"
                  emissiveIntensity={0.3}
                />
              </mesh>
            ))}
        </group>
      )

    default:
      return null
  }
}
