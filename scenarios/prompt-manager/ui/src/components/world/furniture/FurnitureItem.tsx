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
        return <ParkBench color={finalColor} castShadow={castShadow} receiveShadow={receiveShadow} />

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
        return <PicnicTableDetailed color={finalColor} castShadow={castShadow} receiveShadow={receiveShadow} />

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

// ── Detailed furniture components ──

/** Park bench with slatted seat, angled backrest, and metal frame */
function ParkBench({
  color,
  castShadow,
  receiveShadow,
}: {
  color: string
  castShadow: boolean
  receiveShadow: boolean
}) {
  const wood = useMemo(
    () => new THREE.MeshStandardMaterial({ color, roughness: 0.78, metalness: 0.05 }),
    [color]
  )
  const metal = useMemo(
    () => new THREE.MeshStandardMaterial({ color: '#2a2a2a', roughness: 0.25, metalness: 0.85 }),
    []
  )

  // Dimensions
  const W = 1.2 // bench width (x)
  const seatY = 0.30 // seat surface height
  const slat = 0.022 // plank thickness (y)
  const slatD = 0.052 // plank depth (z)
  const gap = 0.007
  const nSeat = 5
  const seatZ = nSeat * slatD + (nSeat - 1) * gap // total seat depth ~0.288
  const backTilt = -0.18 // backrest lean (radians, ~10°)
  const fX = W / 2 + 0.02 // frame x offset
  // Backrest top world-y ≈ seatY + topSlat(0.20) ≈ 0.50
  const backLegH = 0.52 // back leg height (ground to just above backrest)
  const armrestY = 0.50 // armrest at top backrest level

  return (
    <group>
      {/* Seat slats */}
      {Array.from({ length: nSeat }, (_, i) => (
        <mesh
          key={`s${i}`}
          position={[0, seatY, (i - 2) * (slatD + gap)]}
          castShadow={castShadow}
          receiveShadow={receiveShadow}
          material={wood}
        >
          <boxGeometry args={[W, slat, slatD]} />
        </mesh>
      ))}

      {/* Backrest (3 slats, tilted back) */}
      <group
        position={[0, seatY, -seatZ / 2 + slatD / 2]}
        rotation={[backTilt, 0, 0]}
      >
        {[0.06, 0.13, 0.20].map((y, i) => (
          <mesh
            key={`b${i}`}
            position={[0, y, 0]}
            castShadow={castShadow}
            receiveShadow={receiveShadow}
            material={wood}
          >
            <boxGeometry args={[W, slatD, slat]} />
          </mesh>
        ))}
      </group>

      {/* Metal side frames (left & right) */}
      {[-fX, fX].map((x, fi) => (
        <group key={`f${fi}`} position={[x, 0, 0]}>
          {/* Front leg (ground to seat) */}
          <mesh
            position={[0, seatY / 2, seatZ / 2 - slatD / 2]}
            castShadow={castShadow}
            material={metal}
          >
            <boxGeometry args={[0.035, seatY, 0.035]} />
          </mesh>
          {/* Back leg (ground to backrest top) */}
          <mesh
            position={[0, backLegH / 2, -seatZ / 2 + slatD / 2]}
            rotation={[backTilt * 0.3, 0, 0]}
            castShadow={castShadow}
            material={metal}
          >
            <boxGeometry args={[0.035, backLegH, 0.035]} />
          </mesh>
          {/* Seat support rail (under slats) */}
          <mesh
            position={[0, seatY - slat - 0.012, 0]}
            castShadow={castShadow}
            material={metal}
          >
            <boxGeometry args={[0.028, 0.018, seatZ + 0.01]} />
          </mesh>
          {/* Armrest */}
          <mesh
            position={[0, armrestY, 0.02]}
            castShadow={castShadow}
            material={metal}
          >
            <boxGeometry args={[0.042, 0.022, 0.22]} />
          </mesh>
          {/* Armrest front support (bridges front leg top to armrest) */}
          <mesh
            position={[0, (seatY + armrestY) / 2, seatZ / 2 - slatD / 2]}
            castShadow={castShadow}
            material={metal}
          >
            <boxGeometry args={[0.024, armrestY - seatY, 0.024]} />
          </mesh>
        </group>
      ))}

      {/* Stretcher bars (cylindrical, front & back) */}
      {[seatZ / 2 - slatD / 2, -seatZ / 2 + slatD / 2].map((z, i) => (
        <mesh
          key={`st${i}`}
          position={[0, 0.06, z]}
          rotation={[0, 0, Math.PI / 2]}
          castShadow={castShadow}
          material={metal}
        >
          <cylinderGeometry args={[0.01, 0.01, W + 0.05, 8]} />
        </mesh>
      ))}
    </group>
  )
}

/** Picnic table with slatted top, bench seats, and A-frame legs */
function PicnicTableDetailed({
  color,
  castShadow,
  receiveShadow,
}: {
  color: string
  castShadow: boolean
  receiveShadow: boolean
}) {
  const wood = useMemo(
    () => new THREE.MeshStandardMaterial({ color, roughness: 0.78, metalness: 0.05 }),
    [color]
  )
  const brace = useMemo(
    () =>
      new THREE.MeshStandardMaterial({
        color: new THREE.Color(color).multiplyScalar(0.8),
        roughness: 0.85,
        metalness: 0.05,
      }),
    [color]
  )

  // Dimensions
  const tableW = 1.15 // tabletop width (x) — extends past legs
  const W = 0.9 // bench width (x)
  const tableY = 0.44 // tabletop height (2x bench height)
  const benchY = 0.22 // bench seat height
  const plank = 0.022 // plank thickness (y)
  const plankD = 0.072 // plank depth (z)
  const gap = 0.006
  const benchZ = 0.40 // bench center z offset

  // Table top: 10 planks (wide in z-direction, extending past benches)
  const nTable = 10
  const tableDepth = nTable * plankD + (nTable - 1) * gap

  // Criss-cross A-frame leg geometry
  // Each leg attaches near the OPPOSITE table edge and extends to ground past the far bench
  const fX = W / 2 - 0.07
  const legSec = 0.050
  const topZ = tableDepth / 2 - 0.03 // leg attaches near opposite table edge
  const botZ = benchZ + 0.04 // leg reaches ground past bench
  const totalZ = topZ + botZ
  const legAng = Math.atan2(totalZ / 2, tableY / 2)
  const legLen = Math.hypot(totalZ, tableY)
  const fwdCenterZ = (botZ - topZ) / 2 // z-center of forward leg

  // Cross beam directly under bench seats (top of beam touches bottom of seat planks)
  const crossBeamH = 0.035
  const crossY = benchY - plank / 2 - crossBeamH / 2

  return (
    <group>
      {/* Table top planks */}
      {Array.from({ length: nTable }, (_, i) => (
        <mesh
          key={`t${i}`}
          position={[0, tableY, (i - (nTable - 1) / 2) * (plankD + gap)]}
          castShadow={castShadow}
          receiveShadow={receiveShadow}
          material={wood}
        >
          <boxGeometry args={[tableW, plank, plankD]} />
        </mesh>
      ))}

      {/* Front bench planks */}
      {Array.from({ length: 2 }, (_, i) => (
        <mesh
          key={`bf${i}`}
          position={[0, benchY, benchZ + (i - 0.5) * (plankD + gap)]}
          castShadow={castShadow}
          receiveShadow={receiveShadow}
          material={wood}
        >
          <boxGeometry args={[W, plank, plankD]} />
        </mesh>
      ))}

      {/* Back bench planks */}
      {Array.from({ length: 2 }, (_, i) => (
        <mesh
          key={`bb${i}`}
          position={[0, benchY, -benchZ + (i - 0.5) * (plankD + gap)]}
          castShadow={castShadow}
          receiveShadow={receiveShadow}
          material={wood}
        >
          <boxGeometry args={[W, plank, plankD]} />
        </mesh>
      ))}

      {/* Criss-cross A-frame leg assemblies */}
      {[-fX, fX].map((x, fi) => (
        <group key={`a${fi}`} position={[x, 0, 0]}>
          {/* Forward leg: top near -z table edge, bottom past +z bench */}
          <mesh
            position={[0, tableY / 2, fwdCenterZ]}
            rotation={[-legAng, 0, 0]}
            castShadow={castShadow}
            material={brace}
          >
            <boxGeometry args={[legSec, legLen, legSec]} />
          </mesh>
          {/* Back leg: top near +z table edge, bottom past -z bench */}
          <mesh
            position={[0, tableY / 2, -fwdCenterZ]}
            rotation={[legAng, 0, 0]}
            castShadow={castShadow}
            material={brace}
          >
            <boxGeometry args={[legSec, legLen, legSec]} />
          </mesh>
          {/* Cross beam at the leg crossing point */}
          <mesh
            position={[0, crossY, 0]}
            castShadow={castShadow}
            material={brace}
          >
            <boxGeometry args={[legSec, crossBeamH, benchZ * 2 + 0.1]} />
          </mesh>
        </group>
      ))}

      {/* Table support rail (under tabletop, connects A-frames in x) */}
      <mesh
        position={[0, tableY - plank / 2 - 0.025, 0]}
        castShadow={castShadow}
        material={brace}
      >
        <boxGeometry args={[tableW - 0.06, 0.035, 0.05]} />
      </mesh>

      {/* Bench support rails */}
      {[benchZ, -benchZ].map((bz, i) => (
        <mesh
          key={`br${i}`}
          position={[0, benchY - plank / 2 - 0.018, bz]}
          castShadow={castShadow}
          material={brace}
        >
          <boxGeometry args={[W - 0.08, 0.025, 0.04]} />
        </mesh>
      ))}
    </group>
  )
}
