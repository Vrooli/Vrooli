/**
 * FurnitureItem - Renders individual furniture pieces in the 3D world.
 * Procedural geometry for various furniture types.
 */

import { useMemo, useCallback, useRef, useEffect } from 'react'
import { useFrame } from '@react-three/fiber'
import * as THREE from 'three'
import type { FurnitureType } from '@/types/furniture'
import { DEFAULT_FURNITURE_COLORS } from '@/types/furniture'
import { useHoverHighlight } from '@/hooks/useHoverHighlight'
import { usePerformanceStore } from '@/stores/performanceStore'

interface FurnitureItemProps {
  id: string
  type: FurnitureType
  position: [number, number, number]
  rotation?: number
  scale?: number
  color?: string
  castShadow?: boolean
  receiveShadow?: boolean
  lightOn?: boolean
  hoverEnabled?: boolean
  simplifiedMaterials?: boolean
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
  lightOn,
  hoverEnabled = true,
  simplifiedMaterials = false,
  onClick,
}: FurnitureItemProps) {
  const finalColor = color ?? DEFAULT_FURNITURE_COLORS[type]
  const groupRef = useRef<THREE.Group>(null)

  const { isHovered, hoverProps } = useHoverHighlight(id, {
    enabled: hoverEnabled && !!onClick,
    cursor: onClick ? 'pointer' : 'default',
  })

  useEffect(() => {
    if (!simplifiedMaterials || !groupRef.current) return

    groupRef.current.traverse((child) => {
      if (!(child instanceof THREE.Mesh)) return
      if (Array.isArray(child.material)) return
      if ((child.userData as { lowTierSimplified?: boolean }).lowTierSimplified) return

      const source = child.material as THREE.Material
      if (
        !(source instanceof THREE.MeshStandardMaterial) &&
        !(source instanceof THREE.MeshPhysicalMaterial)
      ) {
        return
      }

      child.material = new THREE.MeshBasicMaterial({
        color: source.color,
        transparent: source.transparent,
        opacity: source.opacity,
        side: source.side,
        depthWrite: source.depthWrite,
      })
      ;(child.userData as { lowTierSimplified?: boolean }).lowTierSimplified = true
    })
  }, [simplifiedMaterials])

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

      case 'campfire':
        return <Campfire lightOn={lightOn ?? false} castShadow={castShadow} receiveShadow={receiveShadow} />

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
      ref={groupRef}
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

// ── Shared geometries (module scope, allocated once) ──
const stoneLargeGeo = new THREE.DodecahedronGeometry(0.038, 0)
const stoneSmallGeo = new THREE.DodecahedronGeometry(0.025, 0)
const stoneTinyGeo = new THREE.DodecahedronGeometry(0.016, 0)
const flameGeo = new THREE.ConeGeometry(1, 1, 6)

/** Campfire with stone ring, logs, embers, animated flames, and point light */
function Campfire({
  lightOn,
  castShadow,
  receiveShadow,
}: {
  lightOn: boolean
  castShadow: boolean
  receiveShadow: boolean
}) {
  // Refs for animation (5 flames + light)
  const flameRefs = [
    useRef<THREE.Mesh>(null),
    useRef<THREE.Mesh>(null),
    useRef<THREE.Mesh>(null),
    useRef<THREE.Mesh>(null),
    useRef<THREE.Mesh>(null),
  ]
  const lightRef = useRef<THREE.PointLight>(null)
  const timeRef = useRef(0)
  const perfWindowMsRef = useRef(0)
  const perfWindowCallbacksRef = useRef(0)

  // Materials
  const charcoal = useMemo(
    () => new THREE.MeshStandardMaterial({ color: '#1a1a1a', roughness: 0.95, metalness: 0.05 }),
    []
  )
  const stoneDark = useMemo(
    () => new THREE.MeshStandardMaterial({ color: '#4a4a4a', roughness: 0.92, metalness: 0.05 }),
    []
  )
  const stoneMid = useMemo(
    () => new THREE.MeshStandardMaterial({ color: '#6a6a6a', roughness: 0.88, metalness: 0.06 }),
    []
  )
  const stoneLight = useMemo(
    () => new THREE.MeshStandardMaterial({ color: '#808080', roughness: 0.85, metalness: 0.08 }),
    []
  )
  const stoneWarm = useMemo(
    () => new THREE.MeshStandardMaterial({ color: '#7a6a5a', roughness: 0.9, metalness: 0.04 }),
    []
  )
  const wood = useMemo(
    () => new THREE.MeshStandardMaterial({ color: '#4a2a10', roughness: 0.85, metalness: 0.02 }),
    []
  )
  const woodBark = useMemo(
    () => new THREE.MeshStandardMaterial({ color: '#3a1e08', roughness: 0.9, metalness: 0 }),
    []
  )
  const ember = useMemo(
    () =>
      new THREE.MeshStandardMaterial({
        color: '#3a0000',
        emissive: '#cc2200',
        emissiveIntensity: 1.8,
        roughness: 0.95,
        metalness: 0,
      }),
    []
  )
  const ashMat = useMemo(
    () => new THREE.MeshStandardMaterial({ color: '#333333', emissive: '#220000', emissiveIntensity: 0.3, roughness: 1, metalness: 0 }),
    []
  )
  // Flame materials: core (white-yellow), inner (orange), outer (red-orange), tip (yellow), wisp (pale orange)
  const flameMats = useMemo(
    () => [
      new THREE.MeshStandardMaterial({ color: '#ffcc44', emissive: '#ffaa00', emissiveIntensity: 4.0, transparent: true, opacity: 0.95, depthWrite: false, side: THREE.DoubleSide }),
      new THREE.MeshStandardMaterial({ color: '#ff6600', emissive: '#ff4400', emissiveIntensity: 3.0, transparent: true, opacity: 0.85, depthWrite: false, side: THREE.DoubleSide }),
      new THREE.MeshStandardMaterial({ color: '#ff4400', emissive: '#cc2200', emissiveIntensity: 2.5, transparent: true, opacity: 0.7, depthWrite: false, side: THREE.DoubleSide }),
      new THREE.MeshStandardMaterial({ color: '#ffdd66', emissive: '#ffcc44', emissiveIntensity: 3.5, transparent: true, opacity: 0.8, depthWrite: false, side: THREE.DoubleSide }),
      new THREE.MeshStandardMaterial({ color: '#ff8844', emissive: '#ff6622', emissiveIntensity: 2.0, transparent: true, opacity: 0.5, depthWrite: false, side: THREE.DoubleSide }),
    ],
    []
  )
  const glow = useMemo(
    () =>
      new THREE.MeshBasicMaterial({
        color: '#ff6622',
        transparent: true,
        opacity: 0.08,
        depthWrite: false,
        side: THREE.BackSide,
      }),
    []
  )

  // Stone ring: 20 primary stones + 8 small gap-fillers for a natural look
  const stoneMats = useMemo(() => [stoneDark, stoneMid, stoneLight, stoneWarm], [stoneDark, stoneMid, stoneLight, stoneWarm])
  const stones = useMemo(() => {
    const ringR = 0.30
    const primary = 20
    const result: Array<{
      position: [number, number, number]
      rotation: [number, number, number]
      scale: number
      geo: 'large' | 'small' | 'tiny'
      matIdx: number
    }> = []
    // Primary ring
    for (let i = 0; i < primary; i++) {
      const angle = (i / primary) * Math.PI * 2 + ((i * 13) % 7) * 0.03
      const r = ringR + ((i * 7) % 5) * 0.008 - 0.01
      const yOff = ((i * 11) % 5) * 0.004
      const scaleVar = 0.8 + ((i * 3) % 7) * 0.06
      result.push({
        position: [Math.cos(angle) * r, 0.02 + yOff, Math.sin(angle) * r],
        rotation: [i * 0.8 + 0.3, i * 1.5, i * 0.6 + 0.2],
        scale: scaleVar,
        geo: i % 4 === 0 ? 'small' : 'large',
        matIdx: (i * 3 + 1) % 4,
      })
    }
    // Small gap-fillers between primary stones (slightly inside/outside ring)
    for (let i = 0; i < 8; i++) {
      const baseAngle = ((i * 2.5 + 0.5) / primary) * Math.PI * 2
      const offset = i % 2 === 0 ? 0.035 : -0.03
      const r = ringR + offset
      result.push({
        position: [Math.cos(baseAngle) * r, 0.015 + ((i * 3) % 4) * 0.003, Math.sin(baseAngle) * r],
        rotation: [i * 1.2, i * 2.1, i * 0.9],
        scale: 0.7 + ((i * 5) % 3) * 0.15,
        geo: 'tiny',
        matIdx: (i + 2) % 4,
      })
    }
    return result
  }, [])

  // Ember positions (scattered densely in ash bed)
  const embers = useMemo(
    () =>
      [
        { pos: [0, 0.035, 0] as [number, number, number], r: 0.024 },
        { pos: [0.05, 0.03, 0.03] as [number, number, number], r: 0.018 },
        { pos: [-0.04, 0.03, -0.025] as [number, number, number], r: 0.02 },
        { pos: [0.02, 0.035, -0.04] as [number, number, number], r: 0.016 },
        { pos: [-0.03, 0.032, 0.05] as [number, number, number], r: 0.019 },
        { pos: [0.06, 0.028, -0.01] as [number, number, number], r: 0.015 },
        { pos: [-0.06, 0.03, 0.02] as [number, number, number], r: 0.017 },
        { pos: [0.03, 0.033, 0.06] as [number, number, number], r: 0.014 },
        { pos: [-0.01, 0.036, -0.06] as [number, number, number], r: 0.02 },
        { pos: [0.07, 0.028, 0.04] as [number, number, number], r: 0.013 },
        { pos: [-0.05, 0.031, -0.05] as [number, number, number], r: 0.016 },
        { pos: [0.01, 0.034, 0.03] as [number, number, number], r: 0.022 },
      ],
    []
  )

  // Flame configs: [xOff, baseY, zOff, xScale, yScale, zScale, phaseOffset]
  type FlameConfig = [number, number, number, number, number, number, number]
  const flameConfigs = useMemo<FlameConfig[]>(
    () => [
      [0, 0.18, 0, 0.07, 0.28, 0.07, 0],          // center core
      [0.03, 0.15, 0.02, 0.09, 0.22, 0.09, 1.2],   // front-right
      [-0.02, 0.16, -0.03, 0.08, 0.25, 0.08, 2.8],  // back-left
      [0.01, 0.20, -0.01, 0.05, 0.18, 0.05, 4.0],   // thin tip
      [-0.04, 0.14, 0.01, 0.06, 0.15, 0.06, 5.5],   // low wisp
    ],
    []
  )

  // Animate flames and light
  useFrame((_, delta) => {
    const t0 = performance.now()
    timeRef.current += delta
    const t = timeRef.current

    if (lightOn) {
      for (let i = 0; i < flameConfigs.length; i++) {
        const ref = flameRefs[i]
        const cfg = flameConfigs[i] as FlameConfig
        if (!ref?.current) continue
        const [fx, fy, fz, sx, sy, sz, phase] = cfg
        // Y scale oscillation with two frequencies for organic feel
        const yOsc = 1.0 + Math.sin(t * 4.5 + phase) * 0.2 + Math.sin(t * 11 + phase * 0.7) * 0.08
        // Slight X/Z wobble
        const xWob = Math.sin(t * 3.2 + phase) * 0.005
        const zWob = Math.cos(t * 2.8 + phase * 1.3) * 0.005
        ref.current.scale.set(sx, sy * yOsc, sz)
        ref.current.position.set(
          fx + xWob,
          fy + Math.sin(t * 5.5 + phase) * 0.015,
          fz + zWob,
        )
        ref.current.rotation.y = t * (0.3 + i * 0.15) * (i % 2 === 0 ? 1 : -1)
      }
      // Flicker light intensity
      if (lightRef.current) {
        lightRef.current.intensity = 2.8 + Math.sin(t * 7) * 0.4 + Math.sin(t * 17) * 0.15 + Math.sin(t * 31) * 0.05
      }
    }

    perfWindowMsRef.current += performance.now() - t0
    perfWindowCallbacksRef.current += 1
    if (perfWindowCallbacksRef.current >= 60) {
      usePerformanceStore.getState().recordFrameLoopAggregate(
        perfWindowMsRef.current,
        perfWindowCallbacksRef.current
      )
      perfWindowMsRef.current = 0
      perfWindowCallbacksRef.current = 0
    }
  })

  return (
    <group>
      {/* Fire pit base (slightly concave — two overlapping discs) */}
      <mesh position={[0, 0.005, 0]} receiveShadow={receiveShadow} material={charcoal}>
        <cylinderGeometry args={[0.35, 0.38, 0.01, 20]} />
      </mesh>
      {/* Ash bed */}
      <mesh position={[0, 0.015, 0]} receiveShadow={receiveShadow} material={ashMat}>
        <cylinderGeometry args={[0.22, 0.24, 0.015, 14]} />
      </mesh>

      {/* Stone ring */}
      {stones.map((s, i) => (
        <mesh
          key={`st${i}`}
          position={s.position}
          rotation={s.rotation}
          scale={s.scale}
          castShadow={castShadow}
          receiveShadow={receiveShadow}
          geometry={s.geo === 'large' ? stoneLargeGeo : s.geo === 'small' ? stoneSmallGeo : stoneTinyGeo}
          material={stoneMats[s.matIdx]}
        />
      ))}

      {/* Logs — flat cross-hatch with slight lean toward center */}
      {/* Log 1: lying mostly flat, angled ~15° */}
      <mesh position={[0, 0.05, 0]} rotation={[0.25, 0.3, Math.PI / 2 - 0.15]} castShadow={castShadow} material={wood}>
        <cylinderGeometry args={[0.028, 0.022, 0.42, 6]} />
      </mesh>
      {/* Log 2: crossing log 1 at ~60° */}
      <mesh position={[0, 0.06, 0]} rotation={[0.2, 1.3, Math.PI / 2 - 0.12]} castShadow={castShadow} material={woodBark}>
        <cylinderGeometry args={[0.03, 0.02, 0.40, 6]} />
      </mesh>
      {/* Log 3: third crossing, slight upward lean */}
      <mesh position={[0, 0.07, 0.01]} rotation={[-0.18, -0.5, Math.PI / 2 - 0.2]} castShadow={castShadow} material={wood}>
        <cylinderGeometry args={[0.025, 0.018, 0.36, 6]} />
      </mesh>
      {/* Kindling: small sticks scattered low */}
      <mesh position={[0.08, 0.025, 0.03]} rotation={[0.1, 1.8, Math.PI / 2]} castShadow={castShadow} material={woodBark}>
        <cylinderGeometry args={[0.01, 0.007, 0.16, 5]} />
      </mesh>
      <mesh position={[-0.06, 0.025, -0.05]} rotation={[0.2, 0.6, Math.PI / 2]} castShadow={castShadow} material={wood}>
        <cylinderGeometry args={[0.009, 0.006, 0.14, 5]} />
      </mesh>

      {/* Ember bed (always visible, low glow) */}
      {embers.map(({ pos, r }, i) => (
        <mesh key={`em${i}`} position={pos} material={ember}>
          <sphereGeometry args={[r, 6, 5]} />
        </mesh>
      ))}

      {/* Flames (when lit) */}
      {lightOn && (
        <group>
          {flameConfigs.map(([fx, fy, fz, sx, sy, sz], i) => (
            <mesh
              key={`fl${i}`}
              ref={flameRefs[i]}
              position={[fx, fy, fz]}
              scale={[sx, sy, sz]}
              material={flameMats[i]}
              geometry={flameGeo}
            />
          ))}
        </group>
      )}

      {/* Glow halo (when lit) */}
      {lightOn && (
        <mesh position={[0, 0.15, 0]} material={glow}>
          <sphereGeometry args={[0.45, 12, 10]} />
        </mesh>
      )}

      {/* Point light (when lit) */}
      {lightOn && (
        <pointLight
          ref={lightRef}
          position={[0, 0.35, 0]}
          color="#ff8844"
          intensity={2.8}
          distance={8}
          decay={2}
          castShadow={castShadow}
        />
      )}
    </group>
  )
}
