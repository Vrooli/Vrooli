/**
 * ClothingAccessory - Visual clothing items for members.
 * Renders procedural geometry for tops, bottoms, and footwear.
 */

import { useMemo } from 'react'
import * as THREE from 'three'
import type { ClothingTopType, ClothingBottomType, FootwearType } from '@/types/accessory'
import { ACCESSORY_OFFSETS } from '@/types/accessory'

// Stable default colors
const DEFAULT_TOP_COLOR = '#4f46e5'
const DEFAULT_BOTTOM_COLOR = '#1e293b'
const DEFAULT_FOOTWEAR_COLOR = '#374151'

interface ClothingTopProps {
  type: ClothingTopType
  color?: string
  accentColor?: string
  castShadow?: boolean
}

interface ClothingBottomProps {
  type: ClothingBottomType
  color?: string
  castShadow?: boolean
}

interface FootwearProps {
  type: FootwearType
  color?: string
  castShadow?: boolean
}

/**
 * Clothing top component - wraps around the body capsule
 */
export function ClothingTop({
  type,
  color = DEFAULT_TOP_COLOR,
  accentColor,
  castShadow = true,
}: ClothingTopProps) {
  const offset = ACCESSORY_OFFSETS.torso

  const material = useMemo(
    () =>
      new THREE.MeshStandardMaterial({
        color,
        roughness: 0.8,
        metalness: 0.1,
      }),
    [color]
  )

  const accentMaterial = useMemo(
    () =>
      new THREE.MeshStandardMaterial({
        color: accentColor ?? color,
        roughness: 0.7,
        metalness: 0.1,
      }),
    [accentColor, color]
  )

  if (type === 'none') {
    return null
  }

  const pos = offset.position as [number, number, number]

  switch (type) {
    case 'tshirt':
      return (
        <group position={pos}>
          {/* Main body of t-shirt - slightly larger than body capsule */}
          <mesh position={[0, 0.05, 0]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.27, 0.45, 8, 16]} />
          </mesh>
          {/* Short sleeves */}
          <mesh position={[-0.32, 0.15, 0]} rotation={[0, 0, Math.PI / 4]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.07, 0.1, 4, 8]} />
          </mesh>
          <mesh position={[0.32, 0.15, 0]} rotation={[0, 0, -Math.PI / 4]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.07, 0.1, 4, 8]} />
          </mesh>
          {/* Collar */}
          <mesh position={[0, 0.35, 0.1]} castShadow={castShadow} material={accentMaterial}>
            <torusGeometry args={[0.08, 0.015, 8, 16, Math.PI]} />
          </mesh>
        </group>
      )

    case 'hoodie':
      return (
        <group position={pos}>
          {/* Main body */}
          <mesh position={[0, 0.05, 0]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.29, 0.5, 8, 16]} />
          </mesh>
          {/* Hood (back part) */}
          <mesh position={[0, 0.4, -0.1]} castShadow={castShadow} material={material}>
            <sphereGeometry args={[0.18, 16, 16, 0, Math.PI * 2, 0, Math.PI / 2]} />
          </mesh>
          {/* Long sleeves */}
          <mesh position={[-0.35, 0, 0]} rotation={[0, 0, Math.PI / 6]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.07, 0.25, 4, 8]} />
          </mesh>
          <mesh position={[0.35, 0, 0]} rotation={[0, 0, -Math.PI / 6]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.07, 0.25, 4, 8]} />
          </mesh>
          {/* Kangaroo pocket */}
          <mesh position={[0, -0.15, 0.25]} castShadow={castShadow} material={accentMaterial}>
            <boxGeometry args={[0.25, 0.12, 0.02]} />
          </mesh>
        </group>
      )

    case 'jacket':
      return (
        <group position={pos}>
          {/* Main body - open front style */}
          <mesh position={[0, 0.05, 0]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.28, 0.48, 8, 16]} />
          </mesh>
          {/* Lapels */}
          <mesh position={[-0.1, 0.25, 0.22]} rotation={[0.3, 0.2, 0]} castShadow={castShadow} material={accentMaterial}>
            <boxGeometry args={[0.08, 0.15, 0.02]} />
          </mesh>
          <mesh position={[0.1, 0.25, 0.22]} rotation={[0.3, -0.2, 0]} castShadow={castShadow} material={accentMaterial}>
            <boxGeometry args={[0.08, 0.15, 0.02]} />
          </mesh>
          {/* Long sleeves */}
          <mesh position={[-0.35, 0, 0]} rotation={[0, 0, Math.PI / 6]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.07, 0.28, 4, 8]} />
          </mesh>
          <mesh position={[0.35, 0, 0]} rotation={[0, 0, -Math.PI / 6]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.07, 0.28, 4, 8]} />
          </mesh>
        </group>
      )

    case 'vest':
      return (
        <group position={pos}>
          {/* Main body - sleeveless */}
          <mesh position={[0, 0.05, 0]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.27, 0.45, 8, 16]} />
          </mesh>
          {/* V-neck opening */}
          <mesh position={[0, 0.3, 0.2]} rotation={[Math.PI / 4, 0, 0]} castShadow={castShadow} material={accentMaterial}>
            <coneGeometry args={[0.06, 0.1, 3]} />
          </mesh>
        </group>
      )

    case 'dress':
      return (
        <group position={pos}>
          {/* Upper bodice */}
          <mesh position={[0, 0.1, 0]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.26, 0.3, 8, 16]} />
          </mesh>
          {/* Flared skirt part */}
          <mesh position={[0, -0.25, 0]} castShadow={castShadow} material={material}>
            <coneGeometry args={[0.35, 0.4, 16]} />
          </mesh>
          {/* Belt/waist accent */}
          <mesh position={[0, -0.05, 0]} castShadow={castShadow} material={accentMaterial}>
            <torusGeometry args={[0.26, 0.02, 8, 16]} />
          </mesh>
        </group>
      )

    default:
      return null
  }
}

/**
 * Clothing bottom component - for pants, shorts, skirts
 */
export function ClothingBottom({
  type,
  color = DEFAULT_BOTTOM_COLOR,
  castShadow = true,
}: ClothingBottomProps) {
  const offset = ACCESSORY_OFFSETS.legs

  const material = useMemo(
    () =>
      new THREE.MeshStandardMaterial({
        color,
        roughness: 0.75,
        metalness: 0.05,
      }),
    [color]
  )

  if (type === 'none') {
    return null
  }

  const pos = offset.position as [number, number, number]

  switch (type) {
    case 'pants':
      return (
        <group position={pos}>
          {/* Waist */}
          <mesh position={[0, 0.25, 0]} castShadow={castShadow} material={material}>
            <cylinderGeometry args={[0.22, 0.2, 0.1, 16]} />
          </mesh>
          {/* Left leg */}
          <mesh position={[-0.1, 0, 0]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.08, 0.35, 4, 8]} />
          </mesh>
          {/* Right leg */}
          <mesh position={[0.1, 0, 0]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.08, 0.35, 4, 8]} />
          </mesh>
        </group>
      )

    case 'shorts':
      return (
        <group position={pos}>
          {/* Waist */}
          <mesh position={[0, 0.25, 0]} castShadow={castShadow} material={material}>
            <cylinderGeometry args={[0.22, 0.2, 0.1, 16]} />
          </mesh>
          {/* Short left leg */}
          <mesh position={[-0.1, 0.12, 0]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.085, 0.12, 4, 8]} />
          </mesh>
          {/* Short right leg */}
          <mesh position={[0.1, 0.12, 0]} castShadow={castShadow} material={material}>
            <capsuleGeometry args={[0.085, 0.12, 4, 8]} />
          </mesh>
        </group>
      )

    case 'skirt':
      return (
        <group position={pos}>
          {/* Waist band */}
          <mesh position={[0, 0.25, 0]} castShadow={castShadow} material={material}>
            <cylinderGeometry args={[0.22, 0.22, 0.06, 16]} />
          </mesh>
          {/* Flared skirt */}
          <mesh position={[0, 0.05, 0]} castShadow={castShadow} material={material}>
            <coneGeometry args={[0.3, 0.35, 16]} />
          </mesh>
        </group>
      )

    default:
      return null
  }
}

/**
 * Footwear component - shoes, boots, etc.
 */
export function FootwearAccessory({
  type,
  color = DEFAULT_FOOTWEAR_COLOR,
  castShadow = true,
}: FootwearProps) {
  const offset = ACCESSORY_OFFSETS.feet

  const material = useMemo(
    () =>
      new THREE.MeshStandardMaterial({
        color,
        roughness: 0.6,
        metalness: 0.2,
      }),
    [color]
  )

  if (type === 'none') {
    return null
  }

  const pos = offset.position as [number, number, number]

  switch (type) {
    case 'shoes':
      return (
        <group position={pos}>
          {/* Left shoe */}
          <group position={[-0.1, 0, 0.03]}>
            <mesh castShadow={castShadow} material={material}>
              <boxGeometry args={[0.08, 0.04, 0.14]} />
            </mesh>
            {/* Toe */}
            <mesh position={[0, 0, 0.06]} castShadow={castShadow} material={material}>
              <sphereGeometry args={[0.04, 8, 8, 0, Math.PI * 2, 0, Math.PI / 2]} />
            </mesh>
          </group>
          {/* Right shoe */}
          <group position={[0.1, 0, 0.03]}>
            <mesh castShadow={castShadow} material={material}>
              <boxGeometry args={[0.08, 0.04, 0.14]} />
            </mesh>
            <mesh position={[0, 0, 0.06]} castShadow={castShadow} material={material}>
              <sphereGeometry args={[0.04, 8, 8, 0, Math.PI * 2, 0, Math.PI / 2]} />
            </mesh>
          </group>
        </group>
      )

    case 'boots':
      return (
        <group position={pos}>
          {/* Left boot */}
          <group position={[-0.1, 0.05, 0.02]}>
            {/* Shaft */}
            <mesh castShadow={castShadow} material={material}>
              <cylinderGeometry args={[0.045, 0.05, 0.15, 8]} />
            </mesh>
            {/* Foot part */}
            <mesh position={[0, -0.06, 0.02]} castShadow={castShadow} material={material}>
              <boxGeometry args={[0.09, 0.05, 0.14]} />
            </mesh>
          </group>
          {/* Right boot */}
          <group position={[0.1, 0.05, 0.02]}>
            <mesh castShadow={castShadow} material={material}>
              <cylinderGeometry args={[0.045, 0.05, 0.15, 8]} />
            </mesh>
            <mesh position={[0, -0.06, 0.02]} castShadow={castShadow} material={material}>
              <boxGeometry args={[0.09, 0.05, 0.14]} />
            </mesh>
          </group>
        </group>
      )

    case 'sneakers':
      return (
        <group position={pos}>
          {/* Left sneaker */}
          <group position={[-0.1, 0, 0.03]}>
            <mesh castShadow={castShadow} material={material}>
              <boxGeometry args={[0.09, 0.05, 0.15]} />
            </mesh>
            {/* Sole - slightly different color effect */}
            <mesh position={[0, -0.025, 0]} castShadow={castShadow}>
              <boxGeometry args={[0.1, 0.02, 0.16]} />
              <meshStandardMaterial color="#ffffff" roughness={0.9} />
            </mesh>
          </group>
          {/* Right sneaker */}
          <group position={[0.1, 0, 0.03]}>
            <mesh castShadow={castShadow} material={material}>
              <boxGeometry args={[0.09, 0.05, 0.15]} />
            </mesh>
            <mesh position={[0, -0.025, 0]} castShadow={castShadow}>
              <boxGeometry args={[0.1, 0.02, 0.16]} />
              <meshStandardMaterial color="#ffffff" roughness={0.9} />
            </mesh>
          </group>
        </group>
      )

    case 'sandals':
      return (
        <group position={pos}>
          {/* Left sandal */}
          <group position={[-0.1, -0.02, 0.03]}>
            {/* Sole */}
            <mesh castShadow={castShadow} material={material}>
              <boxGeometry args={[0.08, 0.015, 0.13]} />
            </mesh>
            {/* Straps */}
            <mesh position={[0, 0.015, 0.02]} rotation={[Math.PI / 2, 0, 0]} castShadow={castShadow} material={material}>
              <torusGeometry args={[0.035, 0.008, 4, 8, Math.PI]} />
            </mesh>
            <mesh position={[0, 0.015, -0.03]} rotation={[Math.PI / 2, 0, 0]} castShadow={castShadow} material={material}>
              <torusGeometry args={[0.035, 0.008, 4, 8, Math.PI]} />
            </mesh>
          </group>
          {/* Right sandal */}
          <group position={[0.1, -0.02, 0.03]}>
            <mesh castShadow={castShadow} material={material}>
              <boxGeometry args={[0.08, 0.015, 0.13]} />
            </mesh>
            <mesh position={[0, 0.015, 0.02]} rotation={[Math.PI / 2, 0, 0]} castShadow={castShadow} material={material}>
              <torusGeometry args={[0.035, 0.008, 4, 8, Math.PI]} />
            </mesh>
            <mesh position={[0, 0.015, -0.03]} rotation={[Math.PI / 2, 0, 0]} castShadow={castShadow} material={material}>
              <torusGeometry args={[0.035, 0.008, 4, 8, Math.PI]} />
            </mesh>
          </group>
        </group>
      )

    default:
      return null
  }
}
