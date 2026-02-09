/**
 * HeadAccessory - Visual head accessories for members.
 * Renders procedural geometry for hats, glasses, etc.
 */

import { useMemo } from 'react'
import type { HeadAccessoryProps } from './types'
import { useMaterial } from '../materials'
import type { AccessoryOffset } from '@/types/accessory'

/**
 * Type-specific offsets for head accessories.
 * Different accessories sit at different positions relative to the head.
 *
 * SlimeAgent body: sphere at origin, radius 0.4, Y-squash ~0.85
 * - Top of dome: Y ≈ 0.35
 * - Front of face: Z ≈ 0.3 (eyes at [±0.12, 0.1, 0.3])
 */
const HEAD_ACCESSORY_OFFSETS: Record<string, AccessoryOffset> = {
  // Hat sits on top of dome
  hat: { position: [0, 0.4, 0], rotation: [0, 0, 0], scale: 1 },
  // Crown sits on top of dome
  crown: { position: [0, 0.4, 0], rotation: [0, 0, 0], scale: 1 },
  // Glasses sit on the face (eyes at Y=0.1, Z=0.3)
  glasses: { position: [0, 0.1, 0.05], rotation: [0, 0, 0], scale: 1 },
  // Headphones wrap around at eye level
  headphones: { position: [0, 0.1, 0], rotation: [0, 0, 0], scale: 1 },
  // Halo floats above the dome
  halo: { position: [0, 0.55, 0], rotation: [0, 0, 0], scale: 1 },
}

const DEFAULT_HEAD_OFFSET: AccessoryOffset = { position: [0, 0.4, 0], rotation: [0, 0, 0], scale: 1 }

/**
 * Renders a head accessory based on type.
 * Uses procedural geometry until GLTF models are available.
 */
export function HeadAccessory({
  type,
  position,
  rotation,
  scale = 1,
  color = '#333333',
  castShadow = true,
}: HeadAccessoryProps) {
  const offset = HEAD_ACCESSORY_OFFSETS[type] ?? DEFAULT_HEAD_OFFSET
  const material = useMaterial('plastic', { color })

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
    case 'hat':
      return (
        <group position={pos} rotation={rot} scale={finalScale}>
          {/* Hat crown */}
          <mesh position={[0, 0.05, 0]} castShadow={castShadow} material={material}>
            <cylinderGeometry args={[0.12, 0.15, 0.1, 16]} />
          </mesh>
          {/* Hat brim */}
          <mesh position={[0, 0, 0]} castShadow={castShadow} material={material}>
            <cylinderGeometry args={[0.2, 0.2, 0.02, 16]} />
          </mesh>
        </group>
      )

    case 'glasses':
      return (
        <group position={pos} rotation={rot} scale={finalScale}>
          {/* Left lens */}
          <mesh position={[-0.08, 0, 0.25]} castShadow={castShadow}>
            <torusGeometry args={[0.04, 0.005, 8, 16]} />
            <meshStandardMaterial color={color} metalness={0.8} roughness={0.2} />
          </mesh>
          {/* Right lens */}
          <mesh position={[0.08, 0, 0.25]} castShadow={castShadow}>
            <torusGeometry args={[0.04, 0.005, 8, 16]} />
            <meshStandardMaterial color={color} metalness={0.8} roughness={0.2} />
          </mesh>
          {/* Bridge */}
          <mesh position={[0, 0, 0.25]} rotation={[0, 0, Math.PI / 2]}>
            <cylinderGeometry args={[0.004, 0.004, 0.04, 8]} />
            <meshStandardMaterial color={color} metalness={0.8} roughness={0.2} />
          </mesh>
          {/* Temples (side pieces) */}
          {[-0.12, 0.12].map((x, i) => (
            <mesh key={i} position={[x, 0, 0.15]} rotation={[Math.PI / 2, 0, 0]}>
              <cylinderGeometry args={[0.003, 0.003, 0.15, 8]} />
              <meshStandardMaterial color={color} metalness={0.8} roughness={0.2} />
            </mesh>
          ))}
        </group>
      )

    case 'crown':
      return (
        <group position={pos} rotation={rot} scale={finalScale}>
          {/* Crown base */}
          <mesh position={[0, 0.02, 0]} castShadow={castShadow}>
            <cylinderGeometry args={[0.15, 0.14, 0.04, 16]} />
            <meshStandardMaterial color="#ffd700" metalness={0.9} roughness={0.1} />
          </mesh>
          {/* Crown points */}
          {Array.from({ length: 5 }, (_, i) => {
            const angle = (i / 5) * Math.PI * 2
            const x = Math.cos(angle) * 0.12
            const z = Math.sin(angle) * 0.12
            return (
              <mesh key={i} position={[x, 0.08, z]}>
                <coneGeometry args={[0.03, 0.08, 4]} />
                <meshStandardMaterial color="#ffd700" metalness={0.9} roughness={0.1} />
              </mesh>
            )
          })}
          {/* Jewels */}
          {Array.from({ length: 5 }, (_, i) => {
            const angle = (i / 5) * Math.PI * 2 + Math.PI / 5
            const x = Math.cos(angle) * 0.13
            const z = Math.sin(angle) * 0.13
            return (
              <mesh key={i} position={[x, 0.03, z]}>
                <sphereGeometry args={[0.015, 8, 8]} />
                <meshStandardMaterial
                  color={['#ff0000', '#00ff00', '#0000ff', '#ff00ff', '#00ffff'][i]}
                  emissive={['#ff0000', '#00ff00', '#0000ff', '#ff00ff', '#00ffff'][i]}
                  emissiveIntensity={0.2}
                />
              </mesh>
            )
          })}
        </group>
      )

    case 'headphones':
      return (
        <group position={pos} rotation={rot} scale={finalScale}>
          {/* Headband */}
          <mesh position={[0, 0.1, 0]} rotation={[0, 0, Math.PI / 2]}>
            <torusGeometry args={[0.15, 0.015, 8, 16, Math.PI]} />
            <meshStandardMaterial color={color} />
          </mesh>
          {/* Ear cups */}
          {[-0.15, 0.15].map((x, i) => (
            <mesh key={i} position={[x, 0, 0]} rotation={[0, Math.PI / 2, 0]}>
              <cylinderGeometry args={[0.05, 0.05, 0.03, 16]} />
              <meshStandardMaterial color={color} />
            </mesh>
          ))}
          {/* Ear cushions */}
          {[-0.16, 0.16].map((x, i) => (
            <mesh key={i} position={[x, 0, 0]} rotation={[0, Math.PI / 2, 0]}>
              <torusGeometry args={[0.04, 0.015, 8, 16]} />
              <meshStandardMaterial color="#222222" />
            </mesh>
          ))}
        </group>
      )

    case 'halo':
      return (
        <group position={pos} rotation={rot} scale={finalScale}>
          {/* Glowing halo ring */}
          <mesh rotation={[Math.PI / 2, 0, 0]}>
            <torusGeometry args={[0.18, 0.015, 8, 32]} />
            <meshStandardMaterial
              color="#ffd700"
              emissive="#ffd700"
              emissiveIntensity={0.8}
              toneMapped={false}
            />
          </mesh>
        </group>
      )

    default:
      return null
  }
}
