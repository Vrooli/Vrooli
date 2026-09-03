import { Text } from '@react-three/drei'
import { useFrame, useThree } from '@react-three/fiber'
import { useMemo, useRef } from 'react'
import { Vector3, type Mesh } from 'three'
import type { LabelsTuning, QualityProfile } from '../../config'
import { WORLD_ASSETS, worldAssetUrl } from '../../engine/assets'
import type { Actor, Place } from '../../sim'
import { useWorldStore } from '../WorldStoreContext'
import { bodyPose } from '../actors/pose'
import { clusterLabels, labelWorldSize } from './clusters'
import { resolveCollisions, type LabelRect } from './collision'

const LABEL_COLOR = '#ffffff'
// Stroke instead of outline: troika's outline path builds a second material
// through a prototype chain that three 0.185 materials no longer support.
const LABEL_STROKE = '#101423'
const STROKE_WIDTH = '6%'
const CHAR_WIDTH_FACTOR = 0.58
const REFRESH_EVERY_FRAMES = 3
const BASE_PX_PER_UNIT = 40
const PRIORITY: Record<Actor['state'], number> = { failed: 5, working: 4, walkingToDesk: 4, gathered: 3, walkingToTable: 3, socializing: 2, idle: 1 }
const PINNED_BONUS = 10

interface LabelsProps {
  labels: LabelsTuning
  profile: QualityProfile
  fovDeg: number
  focusedId: string | null
  hoveredId: string | null
}

/** troika's Text mesh surface the pool mutates each refresh. */
interface TextMesh extends Mesh {
  text: string
  fontSize: number
  sync: () => void
}

interface Candidate {
  text: string
  size: number
  x: number
  y: number
  z: number
}

/**
 * SDF name labels above actors with a screen-space collision pass and room
 * clustering. A fixed pool of Text meshes (the label budget) is assigned
 * imperatively every few frames; nothing here calls setState.
 */
export function Labels({ labels, profile, fovDeg, focusedId, hoveredId }: LabelsProps) {
  const store = useWorldStore()
  const camera = useThree((s) => s.camera)
  const size = useThree((s) => s.size)
  const pool = useRef<Array<TextMesh | null>>([])
  const frames = useRef(0)
  const projected = useMemo(() => new Vector3(), [])
  const anchor = useMemo(() => new Vector3(), [])
  const fontUrl = useMemo(() => worldAssetUrl(WORLD_ASSETS.labelFont), [])
  const budget = Math.max(1, Math.min(labels.budget, profile.labelBudget))
  const assigned = useRef<string[]>([])

  useFrame(() => {
    frames.current += 1
    // Billboard every frame; recompute visibility every few frames.
    for (const mesh of pool.current) if (mesh?.visible) mesh.quaternion.copy(camera.quaternion)
    if (frames.current % REFRESH_EVERY_FRAMES !== 0) return
    const state = store.getState()
    const t = store.tuning()
    anchor.set(state.bounds.center[0], 0, state.bounds.center[1])
    const cameraDistance = camera.position.distanceTo(anchor)
    const rooms = new Map<string, Place>()
    for (const id of state.placeOrder) {
      const place = state.places[id]
      if (place?.kind === 'room' && place.teamId) rooms.set(place.teamId, place)
    }
    const members = state.actorOrder.flatMap((id) => {
      const actor = state.actors[id]
      if (!actor) return []
      return [{ id, roomId: actor.teamId ? rooms.get(actor.teamId)?.id : undefined, x: actor.position[0], z: actor.position[1] }]
    })
    const pinned = new Set([focusedId, hoveredId].filter((v): v is string => v !== null))
    const clustered = clusterLabels(members, cameraDistance, labels.collapseDistance)
    const rects: LabelRect[] = []
    const candidates = new Map<string, Candidate>()
    const consider = (id: string, text: string, wx: number, wy: number, wz: number, priority: number) => {
      anchor.set(wx, wy, wz)
      projected.copy(anchor).project(camera)
      if (projected.z > 1) return
      const distance = camera.position.distanceTo(anchor)
      const worldSize = labelWorldSize(distance, fovDeg, size.height, labels.fontSize * BASE_PX_PER_UNIT, labels.minScreenPx, labels.maxScreenPx)
      const heightPx = (worldSize / (2 * distance * Math.tan((fovDeg * Math.PI) / 360))) * size.height
      rects.push({
        id,
        x: ((projected.x + 1) / 2) * size.width,
        y: ((1 - projected.y) / 2) * size.height,
        width: heightPx * CHAR_WIDTH_FACTOR * text.length,
        height: heightPx,
        priority,
        distance,
      })
      candidates.set(id, { text, size: worldSize, x: wx, y: wy, z: wz })
    }
    for (const id of clustered.individual) {
      const actor = state.actors[id]
      if (!actor) continue
      const pose = bodyPose(actor, t.actor)
      consider(id, actor.name, pose.x, pose.y + t.labels.offsetY, pose.z, PRIORITY[actor.state] + (pinned.has(id) ? PINNED_BONUS : 0))
    }
    for (const cluster of clustered.clusters) {
      const room = state.places[cluster.roomId]
      consider(`cluster:${cluster.roomId}`, `${room?.label ?? cluster.roomId} · ${cluster.count}`, cluster.x, t.labels.offsetY, cluster.z, PRIORITY.working)
    }
    const visible = [...resolveCollisions(rects, { paddingPx: labels.paddingPx, budget, pinned })]
    // Keep stable slots for ids already shown so text does not jump between meshes.
    const next: string[] = new Array<string>(budget).fill('')
    const remaining = new Set(visible)
    assigned.current.forEach((id, slot) => {
      if (slot < budget && remaining.has(id)) {
        next[slot] = id
        remaining.delete(id)
      }
    })
    for (const id of remaining) {
      const slot = next.indexOf('')
      if (slot === -1) break
      next[slot] = id
    }
    assigned.current = next
    next.forEach((id, slot) => {
      const mesh = pool.current[slot]
      if (!mesh) return
      const candidate = id ? candidates.get(id) : undefined
      if (!candidate) {
        mesh.visible = false
        return
      }
      mesh.visible = true
      mesh.position.set(candidate.x, candidate.y, candidate.z)
      if (mesh.text !== candidate.text || Math.abs(mesh.fontSize - candidate.size) > 1e-4) {
        mesh.text = candidate.text
        mesh.fontSize = candidate.size
        mesh.sync()
      }
    })
  })

  return (
    <group name="labels">
      {Array.from({ length: budget }, (_, slot) => (
        <Text
          key={slot}
          ref={(mesh: TextMesh | null) => {
            pool.current[slot] = mesh
          }}
          font={fontUrl}
          fontSize={labels.fontSize}
          color={LABEL_COLOR}
          strokeColor={LABEL_STROKE}
          strokeWidth={STROKE_WIDTH}
          anchorX="center"
          anchorY="bottom"
          renderOrder={10}
          visible={false}
        >
          {' '}
        </Text>
      ))}
    </group>
  )
}
