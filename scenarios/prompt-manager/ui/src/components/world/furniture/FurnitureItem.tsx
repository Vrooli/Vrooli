/**
 * FurnitureItem - Renders individual furniture pieces in the 3D world.
 * Procedural geometry for various furniture types.
 */

import { useMemo, useCallback } from 'react'
import * as THREE from 'three'
import type { FurnitureType } from '@/types/furniture'
import { DEFAULT_FURNITURE_COLORS } from '@/types/furniture'
import { useHoverHighlight } from '@/hooks/useHoverHighlight'

interface FurnitureItemProps {
  id: string
  type: FurnitureType
  position: [number, number, number]
  rotation?: number
  scale?: number
  color?: string
  castShadow?: boolean
  receiveShadow?: boolean
  onClick?: () => void
}

/**
 * Renders a furniture item based on type.
 */
export function FurnitureItem({
  id,
  type,
  position,
  rotation = 0,
  scale = 1,
  color,
  castShadow = true,
  receiveShadow = true,
  onClick,
}: FurnitureItemProps) {
  const finalColor = color ?? DEFAULT_FURNITURE_COLORS[type]

  const { isHovered, hoverProps } = useHoverHighlight(id, {
    enabled: !!onClick,
    cursor: onClick ? 'pointer' : 'default',
  })

  const material = useMemo(
    () =>
      new THREE.MeshStandardMaterial({
        color: finalColor,
        roughness: 0.7,
        metalness: 0.1,
      }),
    [finalColor]
  )

  const metalMaterial = useMemo(
    () =>
      new THREE.MeshStandardMaterial({
        color: '#333333',
        roughness: 0.3,
        metalness: 0.8,
      }),
    []
  )

  const cushionMaterial = useMemo(
    () =>
      new THREE.MeshStandardMaterial({
        color: '#4a4a6a',
        roughness: 0.9,
        metalness: 0,
      }),
    []
  )

  const handleClick = useCallback(
    (e: { stopPropagation: () => void }) => {
      e.stopPropagation()
      onClick?.()
    },
    [onClick]
  )

  const renderFurniture = () => {
    switch (type) {
      case 'chair':
        return (
          <group>
            {/* Seat */}
            <mesh position={[0, 0.3, 0]} castShadow={castShadow} receiveShadow={receiveShadow} material={material}>
              <boxGeometry args={[0.4, 0.05, 0.4]} />
            </mesh>
            {/* Back */}
            <mesh position={[0, 0.55, -0.17]} castShadow={castShadow} receiveShadow={receiveShadow} material={material}>
              <boxGeometry args={[0.4, 0.45, 0.05]} />
            </mesh>
            {/* Legs */}
            {([[-0.15, -0.15], [0.15, -0.15], [-0.15, 0.15], [0.15, 0.15]] as [number, number][]).map(([x, z], i) => (
              <mesh key={i} position={[x, 0.14, z]} castShadow={castShadow} material={material}>
                <cylinderGeometry args={[0.02, 0.02, 0.28, 8]} />
              </mesh>
            ))}
          </group>
        )

      case 'bench':
        return (
          <group>
            {/* Seat plank */}
            <mesh position={[0, 0.3, 0]} castShadow={castShadow} receiveShadow={receiveShadow} material={material}>
              <boxGeometry args={[1.2, 0.06, 0.35]} />
            </mesh>
            {/* Support beams */}
            <mesh position={[-0.4, 0.15, 0]} castShadow={castShadow} material={material}>
              <boxGeometry args={[0.08, 0.3, 0.3]} />
            </mesh>
            <mesh position={[0.4, 0.15, 0]} castShadow={castShadow} material={material}>
              <boxGeometry args={[0.08, 0.3, 0.3]} />
            </mesh>
          </group>
        )

      case 'stool':
        return (
          <group>
            {/* Round seat */}
            <mesh position={[0, 0.4, 0]} castShadow={castShadow} receiveShadow={receiveShadow} material={material}>
              <cylinderGeometry args={[0.15, 0.15, 0.04, 16]} />
            </mesh>
            {/* Central pole */}
            <mesh position={[0, 0.2, 0]} castShadow={castShadow} material={metalMaterial}>
              <cylinderGeometry args={[0.03, 0.03, 0.35, 8]} />
            </mesh>
            {/* Base */}
            <mesh position={[0, 0.02, 0]} castShadow={castShadow} material={metalMaterial}>
              <cylinderGeometry args={[0.2, 0.2, 0.02, 16]} />
            </mesh>
          </group>
        )

      case 'armchair':
        return (
          <group>
            {/* Seat cushion */}
            <mesh position={[0, 0.25, 0.05]} castShadow={castShadow} receiveShadow={receiveShadow} material={cushionMaterial}>
              <boxGeometry args={[0.5, 0.12, 0.45]} />
            </mesh>
            {/* Back cushion */}
            <mesh position={[0, 0.5, -0.18]} castShadow={castShadow} receiveShadow={receiveShadow} material={cushionMaterial}>
              <boxGeometry args={[0.5, 0.4, 0.1]} />
            </mesh>
            {/* Armrests */}
            <mesh position={[-0.28, 0.35, 0]} castShadow={castShadow} material={material}>
              <boxGeometry args={[0.08, 0.1, 0.4]} />
            </mesh>
            <mesh position={[0.28, 0.35, 0]} castShadow={castShadow} material={material}>
              <boxGeometry args={[0.08, 0.1, 0.4]} />
            </mesh>
            {/* Frame */}
            <mesh position={[0, 0.1, 0]} castShadow={castShadow} material={material}>
              <boxGeometry args={[0.6, 0.2, 0.5]} />
            </mesh>
          </group>
        )

      case 'desk':
        return (
          <group>
            {/* Tabletop */}
            <mesh position={[0, 0.75, 0]} castShadow={castShadow} receiveShadow={receiveShadow} material={material}>
              <boxGeometry args={[1.2, 0.04, 0.6]} />
            </mesh>
            {/* Legs */}
            {([[-0.55, -0.25], [0.55, -0.25], [-0.55, 0.25], [0.55, 0.25]] as [number, number][]).map(([x, z], i) => (
              <mesh key={i} position={[x, 0.36, z]} castShadow={castShadow} material={material}>
                <boxGeometry args={[0.05, 0.73, 0.05]} />
              </mesh>
            ))}
            {/* Drawer front */}
            <mesh position={[0.3, 0.6, 0.28]} castShadow={castShadow} material={material}>
              <boxGeometry args={[0.4, 0.15, 0.02]} />
            </mesh>
          </group>
        )

      case 'table':
        return (
          <group>
            {/* Tabletop */}
            <mesh position={[0, 0.75, 0]} castShadow={castShadow} receiveShadow={receiveShadow} material={material}>
              <cylinderGeometry args={[0.5, 0.5, 0.04, 24]} />
            </mesh>
            {/* Central pole */}
            <mesh position={[0, 0.4, 0]} castShadow={castShadow} material={material}>
              <cylinderGeometry args={[0.06, 0.08, 0.7, 8]} />
            </mesh>
            {/* Base */}
            <mesh position={[0, 0.03, 0]} castShadow={castShadow} material={material}>
              <cylinderGeometry args={[0.3, 0.3, 0.04, 16]} />
            </mesh>
          </group>
        )

      case 'picnic-table':
        return (
          <group>
            {/* Tabletop */}
            <mesh position={[0, 0.7, 0]} castShadow={castShadow} receiveShadow={receiveShadow} material={material}>
              <boxGeometry args={[0.9, 0.05, 0.5]} />
            </mesh>
            {/* Benches */}
            <mesh position={[0, 0.35, 0.55]} castShadow={castShadow} receiveShadow={receiveShadow} material={material}>
              <boxGeometry args={[0.9, 0.04, 0.25]} />
            </mesh>
            <mesh position={[0, 0.35, -0.55]} castShadow={castShadow} receiveShadow={receiveShadow} material={material}>
              <boxGeometry args={[0.9, 0.04, 0.25]} />
            </mesh>
            {/* A-frame supports */}
            {[-0.35, 0.35].map((x, i) => (
              <group key={i} position={[x, 0, 0]}>
                {/* Table support */}
                <mesh position={[0, 0.35, 0.15]} rotation={[0.3, 0, 0]} castShadow={castShadow} material={material}>
                  <boxGeometry args={[0.06, 0.75, 0.06]} />
                </mesh>
                <mesh position={[0, 0.35, -0.15]} rotation={[-0.3, 0, 0]} castShadow={castShadow} material={material}>
                  <boxGeometry args={[0.06, 0.75, 0.06]} />
                </mesh>
              </group>
            ))}
          </group>
        )

      case 'coffee-table':
        return (
          <group>
            {/* Tabletop */}
            <mesh position={[0, 0.4, 0]} castShadow={castShadow} receiveShadow={receiveShadow} material={material}>
              <boxGeometry args={[0.8, 0.04, 0.5]} />
            </mesh>
            {/* Legs */}
            {([[-0.35, -0.2], [0.35, -0.2], [-0.35, 0.2], [0.35, 0.2]] as [number, number][]).map(([x, z], i) => (
              <mesh key={i} position={[x, 0.19, z]} castShadow={castShadow} material={material}>
                <boxGeometry args={[0.04, 0.38, 0.04]} />
              </mesh>
            ))}
            {/* Shelf */}
            <mesh position={[0, 0.12, 0]} castShadow={castShadow} material={material}>
              <boxGeometry args={[0.7, 0.02, 0.4]} />
            </mesh>
          </group>
        )

      default:
        return null
    }
  }

  return (
    <group
      position={position}
      rotation={[0, rotation, 0]}
      scale={scale}
      onClick={handleClick}
      {...hoverProps}
    >
      {renderFurniture()}
      {/* Hover highlight */}
      {isHovered && (
        <mesh position={[0, 0.4, 0]}>
          <boxGeometry args={[0.6, 0.8, 0.6]} />
          <meshBasicMaterial color="#ffffff" transparent opacity={0.1} wireframe />
        </mesh>
      )}
    </group>
  )
}
