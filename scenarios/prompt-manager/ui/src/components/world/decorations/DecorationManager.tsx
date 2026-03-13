/**
 * DecorationManager - Renders all decorations in the scene.
 * Connects to the decoration store and renders DecorationItem components.
 */

import { memo, useCallback, useEffect, useMemo, useRef } from 'react'
import { useThree } from '@react-three/fiber'
import * as THREE from 'three'
import { DecorationItem } from './DecorationItem'
import { DraggableObject } from '../interaction'
import { useDecorationStore, useDecorationList } from '@/stores/decorationStore'
import { useWorldScaleStore } from '@/stores/worldScaleStore'
import { useEnvironmentStore } from '@/stores/environmentStore'
import { useGraphicsStore } from '@/stores/graphicsStore'
import { useLODStore } from '@/stores/lodStore'
import type { DecorationInstance } from '@/types/decoration'
import { DECORATION_CONFIGS } from '@/types/decoration'
import { applyPlacementConstraints } from '@/lib/world'

interface DecorationManagerProps {
  /** Called when decoration is clicked */
  onDecorationClick?: (decoration: DecorationInstance) => void
  /** Whether decorations are interactive (clickable) */
  interactive?: boolean
  /** Whether decorations are draggable */
  draggable?: boolean
}

const LOCAL_ORIGIN: [number, number, number] = [0, 0, 0]

interface DecorationNodeProps {
  decoration: DecorationInstance
  decorationScale: number
  draggable: boolean
  interactive: boolean
  hoverEnabled: boolean
  simplifiedMaterials: boolean
  castShadow: boolean
  receiveShadow: boolean
  isNightTime: boolean
  constrainPosition: (position: [number, number, number]) => [number, number, number]
  onDecorationClick?: (decoration: DecorationInstance) => void
  onPositionChange: (decorationId: string, newPosition: [number, number, number]) => void
}

const DecorationNode = memo(function DecorationNode({
  decoration,
  decorationScale,
  draggable,
  interactive,
  hoverEnabled,
  simplifiedMaterials,
  castShadow,
  receiveShadow,
  isNightTime,
  constrainPosition,
  onDecorationClick,
  onPositionChange,
}: DecorationNodeProps) {
  const handleClick = useCallback(() => {
    onDecorationClick?.(decoration)
  }, [onDecorationClick, decoration])

  // Resolve effective light state from mode + time of day
  const config = DECORATION_CONFIGS[decoration.type]
  let effectiveLightOn: boolean | undefined
  if (config.emitsLight) {
    const mode = decoration.lightMode ?? 'auto'
    effectiveLightOn = mode === 'on' ? true : mode === 'off' ? false : isNightTime
  }

  const item = (
    <DecorationItem
      id={decoration.id}
      type={decoration.type}
      position={draggable ? LOCAL_ORIGIN : decoration.position}
      rotation={decoration.rotation}
      scale={(decoration.scale ?? 1) * decorationScale}
      color={decoration.color}
      lightOn={effectiveLightOn}
      castShadow={castShadow}
      receiveShadow={receiveShadow}
      hoverEnabled={hoverEnabled}
      simplifiedMaterials={simplifiedMaterials}
      onClick={interactive ? handleClick : undefined}
    />
  )

  if (draggable) {
    return (
      <DraggableObject
        objectId={decoration.id}
        position={decoration.position}
        onPositionChange={(pos) => onPositionChange(decoration.id, pos)}
        constrainPosition={constrainPosition}
      >
        {item}
      </DraggableObject>
    )
  }

  return item
})

type TreeType = 'oak-tree' | 'pine-tree' | 'birch-tree'
const TREE_TYPES = new Set<TreeType>(['oak-tree', 'pine-tree', 'birch-tree'])

interface TreeInstance {
  position: [number, number, number]
  rotation: number
  scale: number
}

interface TreeCandidate {
  decoration: DecorationInstance
  type: TreeType
  distance: number
  forwardDot: number
  lodLevel: 'high' | 'medium' | 'low' | 'culled'
}

function TreeInstancedGroup({
  type,
  instances,
  castShadow,
  receiveShadow,
}: {
  type: TreeType
  instances: TreeInstance[]
  castShadow: boolean
  receiveShadow: boolean
}) {
  const trunkRef = useRef<THREE.InstancedMesh>(null)
  const canopyRef = useRef<THREE.InstancedMesh>(null)
  const dummy = useMemo(() => new THREE.Object3D(), [])

  const profile = useMemo(() => {
    if (type === 'pine-tree') {
      return {
        trunkRadius: 0.08,
        trunkHeight: 1.4,
        canopyRadius: 0.7,
        canopyHeight: 2.4,
        trunkColor: '#5c3a1e',
        canopyColor: '#1a4d2e',
        canopyY: 1.9,
      }
    }
    if (type === 'birch-tree') {
      return {
        trunkRadius: 0.07,
        trunkHeight: 1.2,
        canopyRadius: 0.6,
        canopyHeight: 1.8,
        trunkColor: '#c5baa6',
        canopyColor: '#5a8f3c',
        canopyY: 1.6,
      }
    }
    return {
      trunkRadius: 0.09,
      trunkHeight: 1.3,
      canopyRadius: 0.85,
      canopyHeight: 2.0,
      trunkColor: '#5c3a1e',
      canopyColor: '#2d5a1e',
      canopyY: 1.7,
    }
  }, [type])

  const trunkMaterial = useMemo(
    () => new THREE.MeshBasicMaterial({ color: profile.trunkColor }),
    [profile.trunkColor]
  )
  const canopyMaterial = useMemo(
    () => new THREE.MeshBasicMaterial({ color: profile.canopyColor }),
    [profile.canopyColor]
  )

  useEffect(() => {
    if (!trunkRef.current || !canopyRef.current) return

    for (let i = 0; i < instances.length; i++) {
      const inst = instances[i]
      if (!inst) continue

      dummy.position.set(inst.position[0], inst.position[1], inst.position[2])
      dummy.rotation.set(0, inst.rotation, 0)
      dummy.scale.set(inst.scale, inst.scale, inst.scale)
      dummy.updateMatrix()
      trunkRef.current.setMatrixAt(i, dummy.matrix)

      dummy.position.set(inst.position[0], inst.position[1] + profile.canopyY * inst.scale, inst.position[2])
      dummy.rotation.set(0, inst.rotation, 0)
      dummy.scale.set(inst.scale, inst.scale, inst.scale)
      dummy.updateMatrix()
      canopyRef.current.setMatrixAt(i, dummy.matrix)
    }

    trunkRef.current.instanceMatrix.needsUpdate = true
    canopyRef.current.instanceMatrix.needsUpdate = true
  }, [dummy, instances, profile.canopyY])

  return (
    <group>
      <instancedMesh
        ref={trunkRef}
        args={[undefined, undefined, instances.length]}
        castShadow={castShadow}
        receiveShadow={receiveShadow}
      >
        <cylinderGeometry args={[profile.trunkRadius, profile.trunkRadius * 1.4, profile.trunkHeight, 8]} />
        <primitive object={trunkMaterial} attach="material" />
      </instancedMesh>
      <instancedMesh
        ref={canopyRef}
        args={[undefined, undefined, instances.length]}
        castShadow={castShadow}
        receiveShadow={receiveShadow}
      >
        <coneGeometry args={[profile.canopyRadius, profile.canopyHeight, 8]} />
        <primitive object={canopyMaterial} attach="material" />
      </instancedMesh>
    </group>
  )
}

function TreeForestInstanced({
  instancesByType,
  castShadow,
  receiveShadow,
}: {
  instancesByType: Record<TreeType, TreeInstance[]>
  castShadow: boolean
  receiveShadow: boolean
}) {
  return (
    <group name="tree-forest-instanced">
      {(Object.keys(instancesByType) as TreeType[]).map((type) => {
        const instances = instancesByType[type]
        if (instances.length === 0) return null
        return (
          <TreeInstancedGroup
            key={type}
            type={type}
            instances={instances}
            castShadow={castShadow}
            receiveShadow={receiveShadow}
          />
        )
      })}
    </group>
  )
}

/**
 * Manages rendering of all decoration instances in the world.
 */
export function DecorationManager({
  onDecorationClick,
  interactive = true,
  draggable = false,
}: DecorationManagerProps) {
  const decorationList = useDecorationList()
  const decorationScale = useWorldScaleStore((state) => state.decoration)
  const moveDecoration = useDecorationStore((state) => state.moveDecoration)
  const tier = useGraphicsStore((state) => state.tier)
  const shadowsEnabled = useGraphicsStore((state) => state.config.shadows)
  // Reactive signal so LOD changes are reflected in render decisions.
  useLODStore((state) => state.levelCounts)
  const camera = useThree((state) => state.camera)
  const placementConfig = useEnvironmentStore((state) => state.current.placement)
  const boundaryConfig = useEnvironmentStore((state) => state.current.boundary)
  const groundSize = useEnvironmentStore((state) => state.current.ground.size)
  // Subscribe to derived day/night state so manager doesn't rerender on every time tick.
  const isNightTime = useEnvironmentStore((state) => state.timeValue < 6 || state.timeValue >= 18)

  const constrainPosition = useMemo(() => {
    return (position: [number, number, number]) =>
      applyPlacementConstraints(position, {
        placement: placementConfig,
        boundary: boundaryConfig,
        groundSize,
      })
  }, [placementConfig, boundaryConfig, groundSize])

  const handleClick = useCallback(
    (decoration: DecorationInstance) => {
      onDecorationClick?.(decoration)
    },
    [onDecorationClick]
  )

  const handlePositionChange = useCallback(
    (decorationId: string, newPosition: [number, number, number]) => {
      moveDecoration(decorationId, newPosition)
    },
    [moveDecoration]
  )

  const treeInteractiveDistance =
    tier === 'low' ? 10 : tier === 'medium' ? 16 : tier === 'high' ? 24 : 30
  const boostedTreeInteractiveDistance =
    tier === 'low' ? treeInteractiveDistance * 1.5 :
    tier === 'medium' ? treeInteractiveDistance * 1.35 :
    tier === 'high' ? treeInteractiveDistance * 1.15 :
    treeInteractiveDistance
  const maxInteractiveTrees =
    tier === 'low' ? 20 : tier === 'medium' ? 60 : 200
  const maxInstancedTrees =
    tier === 'low' ? 120 : tier === 'medium' ? 200 : 400
  const interactiveMinForwardDot =
    tier === 'low' ? 0 :
    tier === 'medium' ? -0.1 :
    tier === 'high' ? -0.25 :
    -0.35
  const instancedMinForwardDot =
    tier === 'low' ? 0 :
    tier === 'medium' ? -0.1 :
    tier === 'high' ? -0.4 :
    -0.6
  const enableTreeInstancing = !draggable
  const allowShadows = shadowsEnabled && tier !== 'low'

  const { individualDecorations, instancedTrees } = useMemo(() => {
    if (!enableTreeInstancing) {
      return {
        individualDecorations: decorationList,
        instancedTrees: {
          'oak-tree': [] as TreeInstance[],
          'pine-tree': [] as TreeInstance[],
          'birch-tree': [] as TreeInstance[],
        },
      }
    }

    const individual: DecorationInstance[] = []
    const instanced: Record<TreeType, TreeInstance[]> = {
      'oak-tree': [],
      'pine-tree': [],
      'birch-tree': [],
    }
    const treeCandidates: TreeCandidate[] = []
    const cameraForward = new THREE.Vector3()
    camera.getWorldDirection(cameraForward)
    const cameraForwardLength = Math.sqrt(
      cameraForward.x * cameraForward.x + cameraForward.z * cameraForward.z
    )
    const hasHorizontalForward = cameraForwardLength > 0.0001
    const forwardX = hasHorizontalForward ? cameraForward.x / cameraForwardLength : 0
    const forwardZ = hasHorizontalForward ? cameraForward.z / cameraForwardLength : 0

    for (const decoration of decorationList) {
      const lodLevel = useLODStore.getState().getObjectLOD(`decoration:${decoration.id}`)?.level ?? 'high'
      if (lodLevel === 'culled') continue

      const isTree = TREE_TYPES.has(decoration.type as TreeType)
      if (!isTree) {
        individual.push(decoration)
        continue
      }

      const dx = decoration.position[0] - camera.position.x
      const dz = decoration.position[2] - camera.position.z
      const distance = Math.sqrt(dx * dx + dz * dz)
      const dirLength = distance
      const forwardDot = !hasHorizontalForward || dirLength <= 0.0001
        ? 1
        : ((dx / dirLength) * forwardX + (dz / dirLength) * forwardZ)

      if (forwardDot < instancedMinForwardDot) {
        continue
      }

      treeCandidates.push({
        decoration,
        type: decoration.type as TreeType,
        distance,
        forwardDot,
        lodLevel,
      })
    }

    const interactiveCandidates = interactive
      ? treeCandidates
        .filter((candidate) =>
          candidate.lodLevel !== 'low' &&
          candidate.distance <= boostedTreeInteractiveDistance &&
          candidate.forwardDot >= interactiveMinForwardDot
        )
        .sort((a, b) => {
          const aScore = (a.forwardDot * 2) - (a.distance / boostedTreeInteractiveDistance)
          const bScore = (b.forwardDot * 2) - (b.distance / boostedTreeInteractiveDistance)
          if (aScore !== bScore) return bScore - aScore
          return a.decoration.id.localeCompare(b.decoration.id)
        })
      : []

    const interactiveTreeIds = new Set<string>()
    for (const candidate of interactiveCandidates) {
      if (interactiveTreeIds.size >= maxInteractiveTrees) break
      interactiveTreeIds.add(candidate.decoration.id)
      individual.push(candidate.decoration)
    }

    let instancedTreeCount = 0
    for (const candidate of treeCandidates) {
      if (interactiveTreeIds.has(candidate.decoration.id)) continue
      if (instancedTreeCount >= maxInstancedTrees) break
      instanced[candidate.type].push({
        position: candidate.decoration.position,
        rotation: candidate.decoration.rotation,
        scale: candidate.decoration.scale ?? 1,
      })
      instancedTreeCount++
    }

    return { individualDecorations: individual, instancedTrees: instanced }
  }, [
    boostedTreeInteractiveDistance,
    camera,
    decorationList,
    enableTreeInstancing,
    instancedMinForwardDot,
    interactive,
    interactiveMinForwardDot,
    maxInstancedTrees,
    maxInteractiveTrees,
  ])

  return (
    <group name="decoration-manager">
      {individualDecorations.map((decoration) => {
        const lodLevel = useLODStore.getState().getObjectLOD(`decoration:${decoration.id}`)?.level ?? 'high'
        const hoverEnabled = lodLevel === 'high' || lodLevel === 'medium'
        const simplifiedMaterials = tier === 'low' || lodLevel === 'low'
        const nodeShadows = allowShadows && lodLevel !== 'low'

        return (
        <DecorationNode
          key={decoration.id}
          decoration={decoration}
          decorationScale={decorationScale}
          draggable={draggable}
          interactive={interactive}
          hoverEnabled={hoverEnabled}
          simplifiedMaterials={simplifiedMaterials}
          castShadow={nodeShadows}
          receiveShadow={nodeShadows}
          isNightTime={isNightTime}
          constrainPosition={constrainPosition}
          onDecorationClick={handleClick}
          onPositionChange={handlePositionChange}
        />
        )
      })}

      <TreeForestInstanced
        instancesByType={instancedTrees}
        castShadow={allowShadows}
        receiveShadow={allowShadows}
      />
    </group>
  )
}

/**
 * Hook to add decoration at a position
 */
