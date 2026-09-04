import { useFrame, useThree } from '@react-three/fiber'
import { createContext, useContext, useMemo, type ReactNode } from 'react'
import { PerspectiveCamera } from 'three'
import type { ActorTuning } from '../../config'
import { groundSampler, type GroundSampler, type WorldState } from '../../sim'
import { useWorldStore } from '../WorldStoreContext'
import { bodyPose, type BodyPose } from './pose'

export const POSE_STRIDE = 8
export const POSE = { x: 0, y: 1, z: 2, facing: 3, scaleXZ: 4, scaleY: 5, squash: 6, visible: 7 } as const

export interface PoseBuffer {
  readonly data: Float32Array
  readonly count: number
}

export function createPoseBuffer(count: number): PoseBuffer {
  return { data: new Float32Array(Math.max(1, count) * POSE_STRIDE), count }
}

export function writePoseBuffer(
  buffer: PoseBuffer,
  state: WorldState,
  tuning: ActorTuning,
  hasDetail: (pose: BodyPose) => boolean = () => true,
  computePose: (actor: WorldState['actors'][string], tuning: ActorTuning, ground: GroundSampler) => BodyPose = bodyPose,
): void {
  const ground = groundSampler(state.terrain)
  state.actorOrder.forEach((id, index) => {
    const actor = state.actors[id]
    if (!actor || index >= buffer.count) return
    const pose = computePose(actor, tuning, ground)
    const offset = index * POSE_STRIDE
    buffer.data[offset + POSE.x] = pose.x
    buffer.data[offset + POSE.y] = pose.y
    buffer.data[offset + POSE.z] = pose.z
    buffer.data[offset + POSE.facing] = pose.facing
    buffer.data[offset + POSE.scaleXZ] = pose.scaleXZ
    buffer.data[offset + POSE.scaleY] = pose.scaleY
    buffer.data[offset + POSE.squash] = actor.anim.squash
    buffer.data[offset + POSE.visible] = hasDetail(pose) ? 1 : 0
  })
}

export function readPose(buffer: PoseBuffer, index: number, target: BodyPose): BodyPose {
  const offset = index * POSE_STRIDE
  target.x = buffer.data[offset + POSE.x] ?? 0
  target.y = buffer.data[offset + POSE.y] ?? 0
  target.z = buffer.data[offset + POSE.z] ?? 0
  target.facing = buffer.data[offset + POSE.facing] ?? 0
  target.scaleXZ = buffer.data[offset + POSE.scaleXZ] ?? 0
  target.scaleY = buffer.data[offset + POSE.scaleY] ?? 0
  return target
}

const PoseBufferContext = createContext<PoseBuffer | null>(null)

export function ActorPoseProvider({ children }: { children: ReactNode }) {
  const store = useWorldStore()
  const count = store.getState().actorOrder.length
  const buffer = useMemo(() => createPoseBuffer(count), [count])
  return (
    <PoseBufferContext.Provider value={buffer}>
      <PoseWriter buffer={buffer} />
      {children}
    </PoseBufferContext.Provider>
  )
}

function PoseWriter({ buffer }: { buffer: PoseBuffer }) {
  const store = useWorldStore()
  const camera = useThree((state) => state.camera)
  const viewport = useThree((state) => state.size)

  useFrame(() => {
    const tuning = store.tuning().actor
    const halfFovTangent = camera instanceof PerspectiveCamera ? Math.tan((camera.fov * Math.PI) / 360) : 0
    writePoseBuffer(buffer, store.getState(), tuning, (pose) => {
      if (halfFovTangent === 0 || tuning.look.minDetailPx === 0) return true
      const distance = Math.hypot(camera.position.x - pose.x, camera.position.y - pose.y, camera.position.z - pose.z)
      const projectedHeight = (pose.scaleY / Math.max(distance * halfFovTangent, 0.001)) * viewport.height
      return projectedHeight >= tuning.look.minDetailPx
    })
  }, -1)

  return null
}

export function usePoseBuffer(): PoseBuffer {
  const buffer = useContext(PoseBufferContext)
  if (!buffer) throw new Error('actor pose consumers must be mounted inside ActorPoseProvider')
  return buffer
}
