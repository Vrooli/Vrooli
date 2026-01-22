/**
 * SkillTreeNodes - Renders skill tree nodes as 3D spheres.
 * Uses instanced meshes for performance with large trees.
 */

import { useRef, useCallback, useMemo } from 'react'
import { useFrame, ThreeEvent } from '@react-three/fiber'
import { Text, Billboard } from '@react-three/drei'
import type { InstancedMesh } from 'three'
import * as THREE from 'three'
import type { SkillTreeNode } from '@/types/skilltree'

interface SkillTreeNodesProps {
  nodes: SkillTreeNode[]
  hoveredNodeId: string | null
  onNodeClick: (nodeId: string, event: MouseEvent) => void
  onNodeHover: (nodeId: string | null) => void
}

// Temp objects for instanced mesh updates
const tempObject = new THREE.Object3D()
const tempColor = new THREE.Color()

export function SkillTreeNodes({
  nodes,
  hoveredNodeId,
  onNodeClick,
  onNodeHover,
}: SkillTreeNodesProps) {
  const meshRef = useRef<InstancedMesh>(null)
  const glowRef = useRef<InstancedMesh>(null)

  // Create color array for instances
  const colorArray = useMemo(() => {
    const colors = new Float32Array(nodes.length * 3)
    nodes.forEach((node, i) => {
      tempColor.set(node.color)
      colors[i * 3] = tempColor.r
      colors[i * 3 + 1] = tempColor.g
      colors[i * 3 + 2] = tempColor.b
    })
    return colors
  }, [nodes])

  // Update instance matrices
  useFrame(() => {
    const mesh = meshRef.current
    const glow = glowRef.current
    if (!mesh || !glow) return

    nodes.forEach((node, i) => {
      const [x, y, z] = node.position
      const scale = node.isSelected ? node.size * 1.3 : node.size
      const hoverScale = hoveredNodeId === node.id ? 1.15 : 1

      tempObject.position.set(x, y, z)
      tempObject.scale.setScalar(scale * hoverScale)
      tempObject.updateMatrix()

      mesh.setMatrixAt(i, tempObject.matrix)
      glow.setMatrixAt(i, tempObject.matrix)

      // Update colors for selection state
      if (node.isSelected) {
        tempColor.set('#fbbf24') // Amber for selected
      } else {
        tempColor.set(node.color)
      }
      mesh.setColorAt(i, tempColor)
    })

    mesh.instanceMatrix.needsUpdate = true
    if (mesh.instanceColor) {
      mesh.instanceColor.needsUpdate = true
    }
    glow.instanceMatrix.needsUpdate = true
  })

  // Handle click on instances
  const handleClick = useCallback(
    (event: ThreeEvent<MouseEvent>) => {
      event.stopPropagation()
      const instanceId = event.instanceId
      if (instanceId !== undefined && nodes[instanceId]) {
        onNodeClick(nodes[instanceId].id, event.nativeEvent)
      }
    },
    [nodes, onNodeClick]
  )

  // Handle hover
  const handlePointerOver = useCallback(
    (event: ThreeEvent<PointerEvent>) => {
      event.stopPropagation()
      const instanceId = event.instanceId
      if (instanceId !== undefined && nodes[instanceId]) {
        onNodeHover(nodes[instanceId].id)
        document.body.style.cursor = 'pointer'
      }
    },
    [nodes, onNodeHover]
  )

  const handlePointerOut = useCallback(() => {
    onNodeHover(null)
    document.body.style.cursor = 'default'
  }, [onNodeHover])

  if (nodes.length === 0) return null

  return (
    <group>
      {/* Main node spheres */}
      <instancedMesh
        ref={meshRef}
        args={[undefined, undefined, nodes.length]}
        onClick={handleClick}
        onPointerOver={handlePointerOver}
        onPointerOut={handlePointerOut}
      >
        <sphereGeometry args={[1, 32, 32]}>
          <instancedBufferAttribute
            attach="attributes-color"
            args={[colorArray, 3]}
          />
        </sphereGeometry>
        <meshStandardMaterial
          vertexColors
          metalness={0.3}
          roughness={0.6}
        />
      </instancedMesh>

      {/* Glow effect for selected/hovered nodes */}
      <instancedMesh
        ref={glowRef}
        args={[undefined, undefined, nodes.length]}
      >
        <sphereGeometry args={[1.2, 16, 16]} />
        <meshBasicMaterial
          color="#fbbf24"
          transparent
          opacity={0.15}
          depthWrite={false}
        />
      </instancedMesh>

      {/* Node labels */}
      {nodes.map((node) => (
        <NodeLabel
          key={node.id}
          node={node}
          isHovered={hoveredNodeId === node.id}
        />
      ))}
    </group>
  )
}

/**
 * Label for a skill tree node.
 */
function NodeLabel({
  node,
  isHovered,
}: {
  node: SkillTreeNode
  isHovered: boolean
}) {
  const opacity = node.isSelected || isHovered ? 1 : 0.7
  const yOffset = node.size + 0.3

  return (
    <Billboard
      follow
      position={[node.position[0], node.position[1] + yOffset, node.position[2]]}
    >
      <Text
        fontSize={isHovered ? 0.22 : 0.18}
        color={node.isSelected ? '#fbbf24' : '#e2e8f0'}
        anchorX="center"
        anchorY="bottom"
        outlineWidth={0.02}
        outlineColor="#0f172a"
        fillOpacity={opacity}
      >
        {node.name}
      </Text>
    </Billboard>
  )
}
