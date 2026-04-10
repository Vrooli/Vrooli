/**
 * BoundaryOutline - Renders a visual boundary for the world on the XZ plane.
 */

import { useMemo } from 'react'
import { Line } from '@react-three/drei'
import { useEnvironmentStore } from '@/stores/environmentStore'
import { getBoundaryLinePoints, resolveBoundary } from '@/lib/world'

interface BoundaryOutlineProps {
  /** Ground size used to resolve boundary defaults */
  groundSize?: number
}

export function BoundaryOutline({ groundSize }: BoundaryOutlineProps) {
  const boundaryConfig = useEnvironmentStore((state) => state.current.boundary)

  const resolvedBoundary = useMemo(
    () => resolveBoundary(boundaryConfig, groundSize),
    [boundaryConfig, groundSize]
  )

  const linePoints = useMemo(() => {
    if (!resolvedBoundary) return null
    const points = getBoundaryLinePoints(resolvedBoundary)
    if (points.length === 0) return null
    return points.map(([x, z]) => [x, 0, z] as [number, number, number])
  }, [resolvedBoundary])

  if (!resolvedBoundary || !linePoints) {
    return null
  }

  return (
    <Line
      points={linePoints}
      position={[0, resolvedBoundary.position, 0]}
      color={resolvedBoundary.color ?? '#94a3b8'}
      transparent
      opacity={resolvedBoundary.opacity ?? 0.4}
      lineWidth={1}
    />
  )
}
