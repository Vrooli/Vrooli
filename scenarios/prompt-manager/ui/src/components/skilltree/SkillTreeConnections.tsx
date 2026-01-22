/**
 * SkillTreeConnections - Renders connections between nodes.
 * Uses line segments with gradient coloring.
 */

import { useMemo } from 'react'
import { Line } from '@react-three/drei'
import type { SkillTreeConnection } from '@/types/skilltree'

interface SkillTreeConnectionsProps {
  connections: SkillTreeConnection[]
  selectedNodeIds: string[]
}

export function SkillTreeConnections({
  connections,
  selectedNodeIds,
}: SkillTreeConnectionsProps) {
  return (
    <group>
      {connections.map((connection) => (
        <Connection
          key={connection.id}
          connection={connection}
          isHighlighted={
            selectedNodeIds.includes(connection.sourceId.replace('node-', '')) ||
            selectedNodeIds.includes(connection.targetId.replace('node-', ''))
          }
        />
      ))}
    </group>
  )
}

/**
 * Individual connection line.
 */
function Connection({
  connection,
  isHighlighted,
}: {
  connection: SkillTreeConnection
  isHighlighted: boolean
}) {
  const points = useMemo(
    () => [connection.source, connection.target],
    [connection.source, connection.target]
  )

  return (
    <Line
      points={points}
      color={isHighlighted ? '#fbbf24' : '#475569'}
      lineWidth={isHighlighted ? 2 : 1}
      opacity={isHighlighted ? 0.8 : 0.4}
      transparent
      dashed={!isHighlighted}
      dashScale={2}
      dashSize={0.3}
      dashOffset={0}
    />
  )
}
