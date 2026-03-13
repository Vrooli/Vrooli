/**
 * DecorationItem - Renders individual decorative objects in the 3D world.
 * Procedural geometry for plants, lamps, and other decorations.
 *
 * DOC: docs/guides/ASSET-GENERATION.md
 */

import { Suspense, useMemo, useRef, useCallback, useEffect } from 'react'
import { useFrame } from '@react-three/fiber'
import { useGLTF } from '@react-three/drei'
import type { Mesh } from 'three'
import * as THREE from 'three'
import type { DecorationType } from '@/types/decoration'
import { DEFAULT_DECORATION_COLORS, DECORATION_CONFIGS } from '@/types/decoration'
import { getAssetPath } from '@/config/assetManifest'
import { useHoverHighlight } from '@/hooks/useHoverHighlight'
import { usePerformanceStore } from '@/stores/performanceStore'

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
  hoverEnabled?: boolean
  simplifiedMaterials?: boolean
  onClick?: () => void
}

// Cached tree templates with shadow flags applied once per model path + shadow mode.
const TREE_TEMPLATE_CACHE = new Map<string, THREE.Object3D>()

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
  hoverEnabled = true,
  simplifiedMaterials = false,
  onClick,
}: DecorationItemProps) {
  const finalColor = color ?? DEFAULT_DECORATION_COLORS[type] ?? '#888888'
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

      case 'oak-tree':
      case 'pine-tree':
      case 'birch-tree': {
        const assetId = type === 'oak-tree' ? 'tree-oak' : type === 'pine-tree' ? 'tree-pine' : 'tree-birch'
        const modelPath = getAssetPath(assetId)
        if (!modelPath) return <TreeFallback color={finalColor} castShadow={castShadow} />
        return (
          <Suspense fallback={<TreeFallback color={finalColor} castShadow={castShadow} />}>
            <TreeModel modelPath={modelPath} castShadow={castShadow} />
          </Suspense>
        )
      }

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
      ref={groupRef}
      position={position}
      rotation={[0, rotation, 0]}
      scale={scale}
      onClick={handleClick}
      {...hoverProps}
    >
      {renderDecoration()}
      {isHovered && (() => {
        const size = DECORATION_CONFIGS[type].size
        const highlightY = size[1] / 2
        const highlightRadius = Math.max(size[0], size[2]) / 2
        return (
          <mesh position={[0, highlightY, 0]}>
            <sphereGeometry args={[highlightRadius, 8, 8]} />
            <meshBasicMaterial color="#ffffff" transparent opacity={0.1} wireframe />
          </mesh>
        )
      })()}
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
          emissiveIntensity={lightOn ? 2 : 0}
        />
      </mesh>
      {/* Glow halo (visible without bloom post-processing) */}
      {lightOn && (
        <mesh position={[0, 1.3, 0]}>
          <sphereGeometry args={[0.15, 12, 12]} />
          <meshBasicMaterial color="#fff5e6" transparent opacity={0.12} depthWrite={false} />
        </mesh>
      )}
      {/* Point light */}
      {lightOn && <pointLight position={[0, 1.25, 0]} intensity={1.5} color="#fff5e6" distance={5} decay={2} />}
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
            emissiveIntensity={lightOn ? 1.5 : 0}
          />
        </mesh>
        {/* Glow halo */}
        {lightOn && (
          <mesh position={[0, -0.02, 0]}>
            <sphereGeometry args={[0.08, 10, 10]} />
            <meshBasicMaterial color="#fff5e6" transparent opacity={0.1} depthWrite={false} />
          </mesh>
        )}
      </group>
      {lightOn && <spotLight position={[0, 0.25, 0.05]} angle={0.5} intensity={0.8} distance={2} target-position={[0, 0, 0.5]} />}
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
          emissiveIntensity={lightOn ? 2 : 0}
        />
      </mesh>
      {/* Glow halo */}
      {lightOn && (
        <mesh position={[0, -0.02, 0]}>
          <sphereGeometry args={[0.18, 12, 12]} />
          <meshBasicMaterial color="#fff5e6" transparent opacity={0.12} depthWrite={false} />
        </mesh>
      )}
      {lightOn && <pointLight position={[0, -0.05, 0]} intensity={2} color="#fff5e6" distance={6} decay={2} />}
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
  const perfWindowMsRef = useRef(0)
  const perfWindowCallbacksRef = useRef(0)

  // Slow rotation animation
  useFrame((_, delta) => {
    const t0 = performance.now()
    if (globeRef.current) {
      globeRef.current.rotation.y += delta * 0.2
    }
    perfWindowMsRef.current += performance.now() - t0
    perfWindowCallbacksRef.current += 1
    if (perfWindowCallbacksRef.current >= 120) {
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

/** Procedural fallback tree (shown while GLB loads or if model is missing) */
function TreeFallback({ color, castShadow }: { color: string; castShadow: boolean }) {
  return (
    <group>
      {/* Trunk */}
      <mesh position={[0, 0.6, 0]} castShadow={castShadow}>
        <cylinderGeometry args={[0.08, 0.12, 1.2, 8]} />
        <meshStandardMaterial color="#5c3a1e" roughness={0.9} />
      </mesh>
      {/* Canopy */}
      <mesh position={[0, 1.8, 0]} castShadow={castShadow}>
        <coneGeometry args={[0.8, 1.8, 8]} />
        <meshStandardMaterial color={color} roughness={0.85} />
      </mesh>
    </group>
  )
}

/** Renders an external GLB tree model */
function TreeModel({ modelPath, castShadow }: { modelPath: string; castShadow: boolean }) {
  const { scene } = useGLTF(modelPath)
  const cacheKey = `${modelPath}:${castShadow ? 'shadow-on' : 'shadow-off'}`
  const template = useMemo(() => {
    const cached = TREE_TEMPLATE_CACHE.get(cacheKey)
    if (cached) return cached

    const root = scene.clone(true)
    root.traverse((child) => {
      if (child instanceof THREE.Mesh) {
        child.castShadow = castShadow
        child.receiveShadow = true
      }
    })
    TREE_TEMPLATE_CACHE.set(cacheKey, root)
    return root
  }, [scene, castShadow, cacheKey])

  const instance = useMemo(() => template.clone(true), [template])
  return <primitive object={instance} />
}

// Preload tree models so they start loading immediately
const treeAssetIds = ['tree-oak', 'tree-pine', 'tree-birch'] as const
for (const id of treeAssetIds) {
  const path = getAssetPath(id)
  if (path) useGLTF.preload(path)
}
