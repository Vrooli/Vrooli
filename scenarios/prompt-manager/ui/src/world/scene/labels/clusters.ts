/**
 * Cluster collapse: past a camera distance, a room's member labels collapse
 * into one label carrying the count and the room name.
 */

import { MathUtils } from 'three'

export interface ClusterMember {
  id: string
  roomId?: string
  /** World-space anchor; the cluster label sits at the room centroid. */
  x: number
  z: number
}

export interface ClusterLabel {
  roomId: string
  count: number
  x: number
  z: number
}

export interface ClusterResult {
  /** Member ids that keep their own label. */
  individual: string[]
  clusters: ClusterLabel[]
}

export function clusterLabels(members: readonly ClusterMember[], cameraDistance: number, collapseDistance: number): ClusterResult {
  if (cameraDistance < collapseDistance) return { individual: members.map((m) => m.id), clusters: [] }
  const groups = new Map<string, ClusterMember[]>()
  const individual: string[] = []
  for (const member of members) {
    if (!member.roomId) {
      individual.push(member.id)
      continue
    }
    const list = groups.get(member.roomId) ?? []
    list.push(member)
    groups.set(member.roomId, list)
  }
  const clusters: ClusterLabel[] = []
  for (const [roomId, list] of groups) {
    if (list.length === 1 && list[0]) {
      individual.push(list[0].id)
      continue
    }
    const x = list.reduce((sum, m) => sum + m.x, 0) / list.length
    const z = list.reduce((sum, m) => sum + m.z, 0) / list.length
    clusters.push({ roomId, count: list.length, x, z })
  }
  clusters.sort((a, b) => a.roomId.localeCompare(b.roomId))
  return { individual, clusters }
}

/**
 * World-space label height that renders between min and max screen pixels
 * for a perspective camera at `distance` with vertical `fovDeg` on a viewport
 * `heightPx` tall.
 */
export function labelWorldSize(distance: number, fovDeg: number, heightPx: number, basePx: number, minPx: number, maxPx: number): number {
  const px = Math.min(maxPx, Math.max(minPx, basePx))
  const worldPerPx = (2 * distance * Math.tan(MathUtils.degToRad(fovDeg) / 2)) / Math.max(1, heightPx)
  return px * worldPerPx
}
