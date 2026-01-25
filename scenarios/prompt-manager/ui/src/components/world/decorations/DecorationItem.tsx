/**
 * DecorationItem - Renders individual decorative objects in the 3D world.
 * Procedural geometry for plants, lamps, and other decorations.
 */

import { useMemo, useRef, useCallback } from 'react'
import { useFrame } from '@react-three/fiber'
import type { Mesh } from 'three'
import * as THREE from 'three'
import type { DecorationType } from '@/types/decoration'
import { DEFAULT_DECORATION_COLORS } from '@/types/decoration'
import { useHoverHighlight } from '@/hooks/useHoverHighlight'

interface DecorationItemProps {
  id: string
  type: DecorationType
  position: [number, number, number]
  rotation?: number
  scale?: number
  color?: string
  lightOn?: boolean
  castShadow?: boolean
  receiveShadow?: boolean
  onClick?: () => void
}

/**
 * Renders a decoration item based on type.
 */
export function DecorationItem({
  id,
  type,
  position,
  rotation = 0,
  scale = 1,
  color,
  lightOn = true,
  castShadow = true,
  receiveShadow = true,
  onClick,
}: DecorationItemProps) {
  const finalColor = color ?? DEFAULT_DECORATION_COLORS[type] ?? '#888888'

  const { isHovered, hoverProps } = useHoverHighlight(id, {
    enabled: !!onClick,
    cursor: onClick ? 'pointer' : 'default',
  })

  const handleClick = useCallback(
    (e: { stopPropagation: () => void }) => {
      e.stopPropagation()
      onClick?.()
    },
    [onClick]
  )

  const renderDecoration = () => {
    switch (type) {
      case 'potted-plant':
        return <PottedPlant color={finalColor} castShadow={castShadow} />

      case 'tall-plant':
        return <TallPlant color={finalColor} castShadow={castShadow} />

      case 'cactus':
        return <Cactus color={finalColor} castShadow={castShadow} />

      case 'flowers':
        return <Flowers color={finalColor} castShadow={castShadow} />

      case 'floor-lamp':
        return <FloorLamp lightOn={lightOn} castShadow={castShadow} />

      case 'desk-lamp':
        return <DeskLamp lightOn={lightOn} castShadow={castShadow} />

      case 'hanging-lamp':
        return <HangingLamp lightOn={lightOn} />

      case 'bookshelf':
        return <Bookshelf castShadow={castShadow} receiveShadow={receiveShadow} />

      case 'rug':
        return <Rug color={finalColor} receiveShadow={receiveShadow} />

      case 'vase':
        return <Vase color={finalColor} castShadow={castShadow} />

      case 'globe':
        return <Globe castShadow={castShadow} />

      case 'trophy':
        return <Trophy castShadow={castShadow} />

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
      {renderDecoration()}
      {isHovered && (
        <mesh position={[0, 0.3, 0]}>
          <sphereGeometry args={[0.4, 8, 8]} />
          <meshBasicMaterial color="#ffffff" transparent opacity={0.1} wireframe />
        </mesh>
      )}
    </group>
  )
}

// Individual decoration components

function PottedPlant({ color, castShadow }: { color: string; castShadow: boolean }) {
  const leafMaterial = useMemo(() => new THREE.MeshStandardMaterial({ color, roughness: 0.8 }), [color])
  const potMaterial = useMemo(() => new THREE.MeshStandardMaterial({ color: '#8b4513', roughness: 0.6 }), [])

  return (
    <group>
      {/* Pot */}
      <mesh position={[0, 0.1, 0]} castShadow={castShadow} material={potMaterial}>
        <cylinderGeometry args={[0.12, 0.1, 0.2, 12]} />
      </mesh>
      {/* Dirt */}
      <mesh position={[0, 0.2, 0]}>
        <cylinderGeometry args={[0.11, 0.11, 0.02, 12]} />
        <meshStandardMaterial color="#3d2817" roughness={1} />
      </mesh>
      {/* Leaves */}
      {[0, 60, 120, 180, 240, 300].map((angle, i) => (
        <mesh
          key={i}
          position={[
            Math.cos((angle * Math.PI) / 180) * 0.05,
            0.35 + (i % 2) * 0.05,
            Math.sin((angle * Math.PI) / 180) * 0.05,
          ]}
          rotation={[0.3, (angle * Math.PI) / 180, 0.5]}
          scale={[1, 0.3, 1]}
          castShadow={castShadow}
          material={leafMaterial}
        >
          <sphereGeometry args={[0.08, 8, 8]} />
        </mesh>
      ))}
    </group>
  )
}

function TallPlant({ color, castShadow }: { color: string; castShadow: boolean }) {
  const leafMaterial = useMemo(() => new THREE.MeshStandardMaterial({ color, roughness: 0.8 }), [color])

  return (
    <group>
      {/* Pot */}
      <mesh position={[0, 0.15, 0]} castShadow={castShadow}>
        <cylinderGeometry args={[0.18, 0.14, 0.3, 12]} />
        <meshStandardMaterial color="#5c4033" roughness={0.6} />
      </mesh>
      {/* Trunk */}
      <mesh position={[0, 0.6, 0]} castShadow={castShadow}>
        <cylinderGeometry args={[0.03, 0.04, 0.8, 8]} />
        <meshStandardMaterial color="#654321" roughness={0.9} />
      </mesh>
      {/* Leaves cluster */}
      {[0, 72, 144, 216, 288].map((angle, i) => (
        <mesh
          key={i}
          position={[
            Math.cos((angle * Math.PI) / 180) * 0.1,
            0.9 + (i % 2) * 0.1,
            Math.sin((angle * Math.PI) / 180) * 0.1,
          ]}
          rotation={[0.4 + (i % 2) * 0.2, (angle * Math.PI) / 180, 0]}
          castShadow={castShadow}
          material={leafMaterial}
        >
          <coneGeometry args={[0.12, 0.3, 8]} />
        </mesh>
      ))}
    </group>
  )
}

function Cactus({ color, castShadow }: { color: string; castShadow: boolean }) {
  const cactusMaterial = useMemo(() => new THREE.MeshStandardMaterial({ color, roughness: 0.8 }), [color])

  return (
    <group>
      {/* Small pot */}
      <mesh position={[0, 0.06, 0]} castShadow={castShadow}>
        <cylinderGeometry args={[0.08, 0.06, 0.12, 12]} />
        <meshStandardMaterial color="#cd853f" roughness={0.7} />
      </mesh>
      {/* Main body */}
      <mesh position={[0, 0.25, 0]} castShadow={castShadow} material={cactusMaterial}>
        <capsuleGeometry args={[0.05, 0.2, 8, 12]} />
      </mesh>
      {/* Arms */}
      <mesh position={[0.06, 0.28, 0]} rotation={[0, 0, -0.5]} castShadow={castShadow} material={cactusMaterial}>
        <capsuleGeometry args={[0.03, 0.08, 4, 8]} />
      </mesh>
      <mesh position={[-0.05, 0.32, 0]} rotation={[0, 0, 0.6]} castShadow={castShadow} material={cactusMaterial}>
        <capsuleGeometry args={[0.025, 0.06, 4, 8]} />
      </mesh>
    </group>
  )
}

function Flowers({ color, castShadow }: { color: string; castShadow: boolean }) {
  const flowerMaterial = useMemo(() => new THREE.MeshStandardMaterial({ color, roughness: 0.6 }), [color])
  const stemMaterial = useMemo(() => new THREE.MeshStandardMaterial({ color: '#228b22', roughness: 0.8 }), [])

  return (
    <group>
      {/* Vase */}
      <mesh position={[0, 0.1, 0]} castShadow={castShadow}>
        <cylinderGeometry args={[0.06, 0.08, 0.2, 12]} />
        <meshStandardMaterial color="#4169e1" roughness={0.3} metalness={0.2} />
      </mesh>
      {/* Flowers */}
      {[-0.03, 0, 0.03].map((x, i) => (
        <group key={i} position={[x, 0.2, (i - 1) * 0.02]}>
          {/* Stem */}
          <mesh material={stemMaterial}>
            <cylinderGeometry args={[0.008, 0.008, 0.15, 4]} />
          </mesh>
          {/* Flower head */}
          <mesh position={[0, 0.1, 0]} castShadow={castShadow} material={flowerMaterial}>
            <sphereGeometry args={[0.03, 8, 8]} />
          </mesh>
          {/* Petals */}
          {[0, 72, 144, 216, 288].map((angle, j) => (
            <mesh
              key={j}
              position={[
                Math.cos((angle * Math.PI) / 180) * 0.03,
                0.1,
                Math.sin((angle * Math.PI) / 180) * 0.03,
              ]}
              material={flowerMaterial}
            >
              <sphereGeometry args={[0.015, 6, 6]} />
            </mesh>
          ))}
        </group>
      ))}
    </group>
  )
}

function FloorLamp({ lightOn, castShadow }: { lightOn: boolean; castShadow: boolean }) {
  const poleMaterial = useMemo(() => new THREE.MeshStandardMaterial({ color: '#2f2f2f', roughness: 0.3, metalness: 0.8 }), [])

  return (
    <group>
      {/* Base */}
      <mesh position={[0, 0.02, 0]} castShadow={castShadow} material={poleMaterial}>
        <cylinderGeometry args={[0.15, 0.15, 0.03, 16]} />
      </mesh>
      {/* Pole */}
      <mesh position={[0, 0.7, 0]} castShadow={castShadow} material={poleMaterial}>
        <cylinderGeometry args={[0.02, 0.02, 1.35, 8]} />
      </mesh>
      {/* Shade */}
      <mesh position={[0, 1.35, 0]} castShadow={castShadow}>
        <coneGeometry args={[0.15, 0.2, 16, 1, true]} />
        <meshStandardMaterial color="#f5f5dc" side={THREE.DoubleSide} roughness={0.8} />
      </mesh>
      {/* Light bulb */}
      <mesh position={[0, 1.3, 0]}>
        <sphereGeometry args={[0.04, 8, 8]} />
        <meshStandardMaterial
          color={lightOn ? '#fffacd' : '#333333'}
          emissive={lightOn ? '#fffacd' : '#000000'}
          emissiveIntensity={lightOn ? 1 : 0}
        />
      </mesh>
      {/* Point light */}
      {lightOn && <pointLight position={[0, 1.25, 0]} intensity={0.5} color="#fff5e6" distance={3} />}
    </group>
  )
}

function DeskLamp({ lightOn, castShadow }: { lightOn: boolean; castShadow: boolean }) {
  const metalMaterial = useMemo(() => new THREE.MeshStandardMaterial({ color: '#1a1a1a', roughness: 0.2, metalness: 0.9 }), [])

  return (
    <group>
      {/* Base */}
      <mesh position={[0, 0.02, 0]} castShadow={castShadow} material={metalMaterial}>
        <cylinderGeometry args={[0.08, 0.08, 0.02, 12]} />
      </mesh>
      {/* Arm */}
      <mesh position={[0, 0.15, 0]} rotation={[0.3, 0, 0]} castShadow={castShadow} material={metalMaterial}>
        <cylinderGeometry args={[0.01, 0.01, 0.25, 6]} />
      </mesh>
      {/* Head */}
      <group position={[0, 0.28, 0.05]} rotation={[-0.5, 0, 0]}>
        <mesh castShadow={castShadow}>
          <coneGeometry args={[0.06, 0.08, 12, 1, true]} />
          <meshStandardMaterial color="#2f4f4f" side={THREE.DoubleSide} />
        </mesh>
        <mesh position={[0, -0.02, 0]}>
          <sphereGeometry args={[0.02, 8, 8]} />
          <meshStandardMaterial
            color={lightOn ? '#fffacd' : '#333333'}
            emissive={lightOn ? '#fffacd' : '#000000'}
            emissiveIntensity={lightOn ? 0.8 : 0}
          />
        </mesh>
      </group>
      {lightOn && <spotLight position={[0, 0.25, 0.05]} angle={0.5} intensity={0.3} distance={1} target-position={[0, 0, 0.5]} />}
    </group>
  )
}

function HangingLamp({ lightOn }: { lightOn: boolean }) {
  return (
    <group>
      {/* Chain/cord */}
      <mesh position={[0, 0.3, 0]}>
        <cylinderGeometry args={[0.005, 0.005, 0.5, 4]} />
        <meshStandardMaterial color="#1a1a1a" />
      </mesh>
      {/* Shade */}
      <mesh position={[0, 0, 0]}>
        <coneGeometry args={[0.15, 0.12, 16, 1, true]} />
        <meshStandardMaterial color="#d4af37" roughness={0.3} metalness={0.7} side={THREE.DoubleSide} />
      </mesh>
      {/* Bulb */}
      <mesh position={[0, -0.02, 0]}>
        <sphereGeometry args={[0.05, 8, 8]} />
        <meshStandardMaterial
          color={lightOn ? '#fffacd' : '#333333'}
          emissive={lightOn ? '#fffacd' : '#000000'}
          emissiveIntensity={lightOn ? 1.2 : 0}
        />
      </mesh>
      {lightOn && <pointLight position={[0, -0.05, 0]} intensity={0.8} color="#fff5e6" distance={4} />}
    </group>
  )
}

function Bookshelf({ castShadow, receiveShadow }: { castShadow: boolean; receiveShadow: boolean }) {
  const woodMaterial = useMemo(() => new THREE.MeshStandardMaterial({ color: '#8b4513', roughness: 0.7 }), [])
  const bookColors = ['#8b0000', '#00008b', '#006400', '#4b0082', '#8b4513']

  return (
    <group>
      {/* Frame */}
      {/* Sides */}
      <mesh position={[-0.38, 0.9, 0]} castShadow={castShadow} material={woodMaterial}>
        <boxGeometry args={[0.04, 1.8, 0.3]} />
      </mesh>
      <mesh position={[0.38, 0.9, 0]} castShadow={castShadow} material={woodMaterial}>
        <boxGeometry args={[0.04, 1.8, 0.3]} />
      </mesh>
      {/* Shelves */}
      {[0.02, 0.45, 0.9, 1.35, 1.78].map((y, i) => (
        <mesh key={i} position={[0, y, 0]} castShadow={castShadow} receiveShadow={receiveShadow} material={woodMaterial}>
          <boxGeometry args={[0.8, 0.03, 0.3]} />
        </mesh>
      ))}
      {/* Books on each shelf */}
      {[0.25, 0.7, 1.15, 1.58].map((shelfY, si) => (
        <group key={si} position={[0, shelfY, 0]}>
          {Array.from({ length: 6 }, (_, bi) => (
            <mesh
              key={bi}
              position={[-0.28 + bi * 0.1, 0, 0]}
              castShadow={castShadow}
            >
              <boxGeometry args={[0.08, 0.18, 0.2]} />
              <meshStandardMaterial color={bookColors[(si + bi) % bookColors.length]} roughness={0.8} />
            </mesh>
          ))}
        </group>
      ))}
    </group>
  )
}

function Rug({ color, receiveShadow }: { color: string; receiveShadow: boolean }) {
  const rugMaterial = useMemo(
    () => new THREE.MeshStandardMaterial({ color, roughness: 1, metalness: 0 }),
    [color]
  )

  return (
    <mesh rotation={[-Math.PI / 2, 0, 0]} receiveShadow={receiveShadow} material={rugMaterial}>
      <planeGeometry args={[2, 1.5]} />
    </mesh>
  )
}

function Vase({ color, castShadow }: { color: string; castShadow: boolean }) {
  const vaseMaterial = useMemo(
    () => new THREE.MeshStandardMaterial({ color, roughness: 0.2, metalness: 0.3 }),
    [color]
  )

  return (
    <group>
      {/* Base */}
      <mesh position={[0, 0.05, 0]} castShadow={castShadow} material={vaseMaterial}>
        <cylinderGeometry args={[0.05, 0.06, 0.1, 12]} />
      </mesh>
      {/* Body */}
      <mesh position={[0, 0.18, 0]} castShadow={castShadow} material={vaseMaterial}>
        <sphereGeometry args={[0.07, 12, 12, 0, Math.PI * 2, 0, Math.PI * 0.7]} />
      </mesh>
      {/* Neck */}
      <mesh position={[0, 0.28, 0]} castShadow={castShadow} material={vaseMaterial}>
        <cylinderGeometry args={[0.03, 0.05, 0.08, 12]} />
      </mesh>
    </group>
  )
}

function Globe({ castShadow }: { castShadow: boolean }) {
  const globeRef = useRef<Mesh>(null)

  // Slow rotation animation
  useFrame((_, delta) => {
    if (globeRef.current) {
      globeRef.current.rotation.y += delta * 0.2
    }
  })

  return (
    <group>
      {/* Stand base */}
      <mesh position={[0, 0.02, 0]} castShadow={castShadow}>
        <cylinderGeometry args={[0.08, 0.08, 0.02, 12]} />
        <meshStandardMaterial color="#654321" roughness={0.6} />
      </mesh>
      {/* Stand pole */}
      <mesh position={[0, 0.1, 0]} castShadow={castShadow}>
        <cylinderGeometry args={[0.015, 0.015, 0.15, 6]} />
        <meshStandardMaterial color="#b8860b" roughness={0.3} metalness={0.7} />
      </mesh>
      {/* Globe */}
      <mesh ref={globeRef} position={[0, 0.25, 0]} castShadow={castShadow}>
        <sphereGeometry args={[0.1, 24, 24]} />
        <meshStandardMaterial color="#4682b4" roughness={0.5} />
      </mesh>
      {/* Axis ring */}
      <mesh position={[0, 0.25, 0]} rotation={[0.4, 0, 0]}>
        <torusGeometry args={[0.11, 0.005, 8, 24]} />
        <meshStandardMaterial color="#b8860b" roughness={0.3} metalness={0.7} />
      </mesh>
    </group>
  )
}

function Trophy({ castShadow }: { castShadow: boolean }) {
  const goldMaterial = useMemo(
    () => new THREE.MeshStandardMaterial({ color: '#ffd700', roughness: 0.2, metalness: 0.9 }),
    []
  )

  return (
    <group>
      {/* Base */}
      <mesh position={[0, 0.02, 0]} castShadow={castShadow}>
        <boxGeometry args={[0.08, 0.03, 0.08]} />
        <meshStandardMaterial color="#1a1a1a" roughness={0.3} />
      </mesh>
      {/* Stem */}
      <mesh position={[0, 0.08, 0]} castShadow={castShadow} material={goldMaterial}>
        <cylinderGeometry args={[0.015, 0.02, 0.1, 8]} />
      </mesh>
      {/* Cup */}
      <mesh position={[0, 0.18, 0]} castShadow={castShadow} material={goldMaterial}>
        <cylinderGeometry args={[0.04, 0.025, 0.1, 12]} />
      </mesh>
      {/* Handles */}
      {[-0.05, 0.05].map((x, i) => (
        <mesh key={i} position={[x, 0.18, 0]} rotation={[0, 0, x > 0 ? -0.3 : 0.3]} material={goldMaterial}>
          <torusGeometry args={[0.02, 0.005, 6, 12, Math.PI]} />
        </mesh>
      ))}
    </group>
  )
}
