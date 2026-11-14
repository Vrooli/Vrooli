import { useFrame, useThree } from '@react-three/fiber'
import { createContext, useContext, useMemo, type ReactNode } from 'react'
import { MathUtils, PerspectiveCamera } from 'three'
import type { ActorTuning } from '../../config'
import { groundSampler, type GroundSampler, type WorldState } from '../../sim'
import { useWorldStore } from '../WorldStoreContext'
import { bodyPose, type BodyPose } from './pose'
import { cameraFacing, facingWeight } from './faceCamera'

export const POSE_STRIDE = 8
export const POSE = { x: 0, y: 1, z: 2, facing: 3, scaleXZ: 4, scaleY: 5, squash: 6, visible: 7 } as const

export interface PoseBuffer {
  readonly data: Float32Array
  readonly facingWeights: Float32Array
  readonly count: number
}

export function createPoseBuffer(count: number): PoseBuffer {
  return { data: new Float32Array(Math.max(1, count) * POSE_STRIDE), facingWeights: new Float32Array(count), count }
}

export function writePoseBuffer(
  buffer: PoseBuffer,
  state: WorldState,
  tuning: ActorTuning,
  hasDetail: (pose: BodyPose) => boolean = () => true,
  computePose: (actor: WorldState['actors'][string], tuning: ActorTuning, ground: GroundSampler) => BodyPose = bodyPose,
  presentation?: { focusedId: string | null; cameraX: number; cameraZ: number; dt: number },
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
    let facing = pose.facing
    if (presentation) {
      const weight = facingWeight(buffer.facingWeights[index] ?? 0, id === presentation.focusedId, actor.speed, presentation.dt, tuning.facing)
      buffer.facingWeights[index] = weight
      // Bodies, faces and extras share this rendered yaw. Gameplay reads actor.facing,
      // which is never modified by camera focus or by any presentation blend.
      facing = cameraFacing(pose.facing, Math.atan2(presentation.cameraX - pose.x, presentation.cameraZ - pose.z), weight, MathUtils.degToRad(tuning.facing.maxYawDeg))
    }
    buffer.data[offset + POSE.facing] = facing
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

export function ActorPoseProvider({ children, focusedId = null }: { children: ReactNode; focusedId?: string | null }) {
  const store = useWorldStore()
  const count = store.getState().actorOrder.length
  const buffer = useMemo(() => createPoseBuffer(count), [count])
  return (
    <PoseBufferContext.Provider value={buffer}>
      <PoseWriter buffer={buffer} focusedId={focusedId} />
      {children}
    </PoseBufferContext.Provider>
  )
}

function PoseWriter({ buffer, focusedId }: { buffer: PoseBuffer; focusedId: string | null }) {
  const store = useWorldStore()
  const camera = useThree((state) => state.camera)
  const viewport = useThree((state) => state.size)

  useFrame((_, dt) => {
    const tuning = store.tuning().actor
    const halfFovTangent = camera instanceof PerspectiveCamera ? Math.tan(MathUtils.degToRad(camera.fov) / 2) : 0
    writePoseBuffer(buffer, store.getState(), tuning, (pose) => {
      if (halfFovTangent === 0 || tuning.look.minDetailPx === 0) return true
      const distance = Math.hypot(camera.position.x - pose.x, camera.position.y - pose.y, camera.position.z - pose.z)
      const projectedHeight = (pose.scaleY / Math.max(distance * halfFovTangent, tuning.look.minimumProjectionDepth)) * viewport.height
      return projectedHeight >= tuning.look.minDetailPx
    }, bodyPose, { focusedId, cameraX: camera.position.x, cameraZ: camera.position.z, dt })
  }, -1)

  return null
}

export function usePoseBuffer(): PoseBuffer {
  const buffer = useContext(PoseBufferContext)
  if (!buffer) throw new Error('actor pose consumers must be mounted inside ActorPoseProvider')
  return buffer
}
