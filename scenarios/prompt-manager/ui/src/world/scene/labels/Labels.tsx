import { architecture } from '../../config/architecture'
import { insideSpace } from '../../sim/layout/spaces'
import { heightAt } from '../../sim/terrain'
import { Text } from '@react-three/drei'
import { useFrame, useThree } from '@react-three/fiber'
import { useMemo, useRef } from 'react'
import { MathUtils, Vector3, type Mesh } from 'three'
import type { LabelsTuning, QualityProfile } from '../../config'
import { WORLD_ASSETS, worldAssetUrl } from '../../engine/assets'
import type { Place, WorldState } from '../../sim'
import { useWorldStore } from '../WorldStoreContext'
import type { BodyPose } from '../actors/pose'
import { POSE, POSE_STRIDE, readPose, usePoseBuffer } from '../actors/PoseBuffer'
import { clusterLabels, labelWorldSize } from './clusters'
import { resolveCollisions, type LabelRect } from './collision'

interface LabelsProps {
  unreadIds?: ReadonlySet<string>
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
export function Labels({ labels, profile, fovDeg, focusedId, hoveredId, unreadIds }: LabelsProps) {
  const store = useWorldStore()
  const camera = useThree((s) => s.camera)
  const size = useThree((s) => s.size)
  const pool = useRef<Array<TextMesh | null>>([])
  const frames = useRef(0)
  const projected = useMemo(() => new Vector3(), [])
  const anchor = useMemo(() => new Vector3(), [])
  const fontUrl = useMemo(() => worldAssetUrl(WORLD_ASSETS.labelFont), [])
  const budget = Math.max(0, Math.min(labels.budget, profile.labelBudget))
  const assigned = useRef<string[]>([])
  const membership = useRef<{ actorOrder: WorldState['actorOrder'] | null; places: WorldState['places'] | null; placeOrder: WorldState['placeOrder'] | null; actorIndices: Map<string, number>; rooms: Map<string, Place> }>({ actorOrder: null, places: null, placeOrder: null, actorIndices: new Map(), rooms: new Map() })
  const poses = usePoseBuffer()
  const pose = useMemo<BodyPose>(() => ({ x: 0, y: 0, z: 0, facing: 0, scaleXZ: 0, scaleY: 0 }), [])

  useFrame(() => {
    if (budget === 0) {
      assigned.current.length = 0
      return
    }
    frames.current += 1
    // Billboard every frame; recompute visibility every few frames.
    for (const mesh of pool.current) if (mesh?.visible) mesh.quaternion.copy(camera.quaternion)
    if (frames.current % labels.refreshEveryFrames !== 0) return
    const state = store.getState()
    const t = store.tuning()
    const cached = membership.current
    if (cached.actorOrder !== state.actorOrder) {
      cached.actorIndices = new Map(state.actorOrder.map((id, index) => [id, index]))
      cached.actorOrder = state.actorOrder
    }
    const actorIndices = cached.actorIndices
    anchor.set(state.bounds.center[0], 0, state.bounds.center[1])
    const cameraDistance = camera.position.distanceTo(anchor)
    if (cached.places !== state.places || cached.placeOrder !== state.placeOrder) {
      cached.rooms.clear()
      for (const id of state.placeOrder) {
        const place = state.places[id]
        if (place?.kind === 'room' && place.teamId) cached.rooms.set(place.teamId, place)
      }
      cached.places = state.places
      cached.placeOrder = state.placeOrder
    }
    const rooms = cached.rooms
    const members = state.actorOrder.flatMap((id) => {
      const actor = state.actors[id]
      if (!actor) return []
      return [{ id, roomId: actor.teamId ? rooms.get(actor.teamId)?.id : undefined, x: actor.position[0], z: actor.position[1] }]
    })
    const pinned = new Set([focusedId, hoveredId, ...(unreadIds ?? [])].filter((v): v is string => v !== null))
    const clustered = clusterLabels(members, cameraDistance, labels.collapseDistance)
    const rects: LabelRect[] = []
    const candidates = new Map<string, Candidate>()
    const consider = (id: string, text: string, wx: number, wy: number, wz: number, priority: number) => {
      anchor.set(wx, wy, wz)
      projected.copy(anchor).project(camera)
      if (projected.z > 1) return
      const distance = camera.position.distanceTo(anchor)
      const worldSize = labelWorldSize(distance, fovDeg, size.height, labels.fontSize * labels.basePxPerUnit, labels.minScreenPx, labels.maxScreenPx)
      const heightPx = (worldSize / (2 * distance * Math.tan(MathUtils.degToRad(fovDeg) / 2))) * size.height
      rects.push({
        id,
        x: ((projected.x + 1) / 2) * size.width,
        y: ((1 - projected.y) / 2) * size.height,
        width: heightPx * labels.charWidthFactor * text.length,
        height: heightPx,
        priority,
        distance,
      })
      candidates.set(id, { text, size: worldSize, x: wx, y: wy, z: wz })
    }
    for (const id of new Set([...clustered.individual, ...(unreadIds ?? [])])) {
      const actor = state.actors[id]
      if (!actor) continue
      const poseIndex = actorIndices.get(id)
      if (poseIndex === undefined || (poses.data[poseIndex * POSE_STRIDE + POSE.visible] ?? 0) === 0) continue
      readPose(poses, poseIndex, pose)
      // An unread marker must clear an enclosing roof, not disappear inside it.
      const enclosure = unreadIds?.has(id) ? [...rooms.values()].find(room => room.space && insideSpace(room, actor.position)) : undefined
      const markerY = enclosure ? Math.max(pose.y + t.labels.offsetY, heightAt(state.terrain, ...enclosure.position) + architecture.wallHeight + architecture.cabinRoofHeight + architecture.enclosureMarkerClearance) : pose.y + t.labels.offsetY
      consider(id, unreadIds?.has(id) ? `New message · ${actor.name}` : actor.name, pose.x, markerY, pose.z, labels.priorities[actor.state] + (pinned.has(id) ? labels.pinnedBonus : 0))
    }
    for (const cluster of clustered.clusters) {
      const room = state.places[cluster.roomId]
      consider(`cluster:${cluster.roomId}`, `${room?.label ?? cluster.roomId} · ${cluster.count}`, cluster.x, t.labels.roomOffsetY, cluster.z, labels.priorities.working)
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
      const scale = candidate.size / labels.fontSize
      if (Math.abs(mesh.scale.x - scale) > labels.syncSizeEpsilon) mesh.scale.setScalar(scale)
      if (mesh.text !== candidate.text || mesh.fontSize !== labels.fontSize) {
        mesh.text = candidate.text
        mesh.fontSize = labels.fontSize
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
          color={labels.color}
          strokeColor={labels.strokeColor}
          strokeWidth={`${labels.strokePercent}%`}
          anchorX="center"
          anchorY="bottom"
          renderOrder={labels.renderOrder}
          visible={false}
        >
          {' '}
        </Text>
      ))}
    </group>
  )
}
